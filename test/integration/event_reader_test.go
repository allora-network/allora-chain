package integration_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/grpc/metadata"

	testCommon "github.com/allora-network/allora-chain/test/common"
)

// eventPhase says which part of block execution produced an event.
type eventPhase int

const (
	phaseBeginBlock eventPhase = iota
	phaseTx
	phaseEndBlock
)

func (p eventPhase) String() string {
	switch p {
	case phaseBeginBlock:
		return "BeginBlock"
	case phaseTx:
		return "tx"
	case phaseEndBlock:
		return "EndBlock"
	default:
		return "unknown"
	}
}

// Attributes the SDK appends to an emitted event. They are not part of the
// typed payload, and the value of modeAttribute is not valid JSON.
const (
	modeAttribute     = "mode"
	msgIndexAttribute = "msg_index"
)

// chainEvent is one event as a consumer reads it from a node. typed is nil for
// events that are not emissions or mint typed events.
type chainEvent struct {
	phase   eventPhase
	txIndex int // position of the tx in its block, -1 outside phaseTx
	raw     abci.Event
	typed   proto.Message
}

// chainBlock holds a block's events in execution order: BeginBlock, the txs in
// block order, then EndBlock.
type chainBlock struct {
	height int64
	events []chainEvent
}

// isTypedEventType reports whether an event type belongs to a module whose
// events must decode into a registered proto message.
func isTypedEventType(eventType string) bool {
	return strings.HasPrefix(eventType, "emissions.") || strings.HasPrefix(eventType, "mint.")
}

// decodeTypedEvent parses an emissions or mint event into its proto type with
// the SDK's typed event parser.
func decodeTypedEvent(event abci.Event) (proto.Message, error) {
	payload := abci.Event{Type: event.Type, Attributes: make([]abci.EventAttribute, 0, len(event.Attributes))}
	for _, attribute := range event.Attributes {
		if attribute.Key != modeAttribute && attribute.Key != msgIndexAttribute {
			payload.Attributes = append(payload.Attributes, attribute)
		}
	}
	return sdk.ParseTypedEvent(payload)
}

// eventFingerprint renders events for comparison between nodes. The index flag
// is left out because it depends on node configuration, not on chain execution.
func eventFingerprint(events []abci.Event) []string {
	rendered := make([]string, len(events))
	for i, event := range events {
		var b strings.Builder
		b.WriteString(event.Type)
		for _, attribute := range event.Attributes {
			fmt.Fprintf(&b, "|%s=%s", attribute.Key, attribute.Value)
		}
		rendered[i] = b.String()
	}
	return rendered
}

// firstFingerprintDifference describes the first difference between two event
// lists, or returns "" when they are identical.
func firstFingerprintDifference(a, b []string) string {
	if len(a) != len(b) {
		return fmt.Sprintf("event counts differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			return fmt.Sprintf("event %d differs: %q vs %q", i, a[i], b[i])
		}
	}
	return ""
}

// atHeight makes a query read the state committed at height, which is the state
// after that block's EndBlock.
func atHeight(ctx context.Context, height int64) context.Context {
	return metadata.AppendToOutgoingContext(ctx, grpctypes.GRPCBlockHeightHeader, strconv.FormatInt(height, 10))
}

// splitBlockEvents separates the finalize block events into BeginBlock and
// EndBlock using the SDK's mode attribute.
func splitBlockEvents(events []abci.Event) (begin []abci.Event, end []abci.Event) {
	for _, event := range events {
		isEnd := false
		for _, attribute := range event.Attributes {
			if attribute.Key == modeAttribute && attribute.Value == "EndBlock" {
				isEnd = true
				break
			}
		}
		if isEnd {
			end = append(end, event)
		} else {
			begin = append(begin, event)
		}
	}
	return begin, end
}

// readChainBlock reads one block from every node through /block_results and
// /tx, the two endpoints the indexer consumes. It reports any disagreement
// between nodes or endpoints, and any event that does not decode.
func readChainBlock(ctx context.Context, m testCommon.TestConfig, height int64, fail func(string, ...any)) (chainBlock, error) {
	block := chainBlock{height: height, events: nil}
	reference, err := m.Client.Clients[0].RPC.BlockResults(ctx, &height)
	if err != nil {
		return block, fmt.Errorf("node 0 /block_results at %d: %w", height, err)
	}
	referenceFinalize := eventFingerprint(reference.FinalizeBlockEvents)
	referenceTxs := make([][]string, len(reference.TxsResults))
	for i, txResult := range reference.TxsResults {
		referenceTxs[i] = eventFingerprint(txResult.Events)
	}

	for node := 1; node < len(m.Client.Clients); node++ {
		results, err := m.Client.Clients[node].RPC.BlockResults(ctx, &height)
		if err != nil {
			return block, fmt.Errorf("node %d /block_results at %d: %w", node, height, err)
		}
		if diff := firstFingerprintDifference(referenceFinalize, eventFingerprint(results.FinalizeBlockEvents)); diff != "" {
			fail("node %d finalize block events disagree with node 0: %s", node, diff)
		}
		if len(results.TxsResults) != len(referenceTxs) {
			fail("node %d reports %d tx results, node 0 reports %d", node, len(results.TxsResults), len(referenceTxs))
			continue
		}
		for i, txResult := range results.TxsResults {
			if diff := firstFingerprintDifference(referenceTxs[i], eventFingerprint(txResult.Events)); diff != "" {
				fail("node %d tx %d events disagree with node 0: %s", node, i, diff)
			}
		}
	}

	if len(reference.TxsResults) > 0 {
		committed, err := m.Client.Clients[0].RPC.Block(ctx, &height)
		if err != nil {
			return block, fmt.Errorf("node 0 /block at %d: %w", height, err)
		}
		for i, rawTx := range committed.Block.Txs {
			hash := rawTx.Hash()
			for node := range m.Client.Clients {
				indexed, err := m.Client.Clients[node].RPC.Tx(ctx, hash, false)
				if err != nil {
					return block, fmt.Errorf("node %d /tx %X at %d: %w", node, hash, height, err)
				}
				if diff := firstFingerprintDifference(referenceTxs[i], eventFingerprint(indexed.TxResult.Events)); diff != "" {
					fail("node %d /tx disagrees with /block_results for tx %d: %s", node, i, diff)
				}
			}
		}
	}

	begin, end := splitBlockEvents(reference.FinalizeBlockEvents)
	appendEvents := func(phase eventPhase, txIndex int, events []abci.Event) {
		for _, event := range events {
			if !isTypedEventType(event.Type) {
				block.events = append(block.events, chainEvent{phase: phase, txIndex: txIndex, raw: event, typed: nil})
				continue
			}
			typed, err := decodeTypedEvent(event)
			if err != nil {
				fail("undecodable %s event %s: %v", phase, event.Type, err)
				continue
			}
			block.events = append(block.events, chainEvent{phase: phase, txIndex: txIndex, raw: event, typed: typed})
		}
	}
	appendEvents(phaseBeginBlock, -1, begin)
	for i, txResult := range reference.TxsResults {
		appendEvents(phaseTx, i, txResult.Events)
	}
	appendEvents(phaseEndBlock, -1, end)
	return block, nil
}
