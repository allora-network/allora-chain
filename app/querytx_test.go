package app

import (
	"context"
	"errors"
	"sync"
	"testing"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/x/authz"
	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	emissionsv2 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v2"
	emissionsv3 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v3"
	emissionsv4 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v4"
	emissionsv5 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v5"
	emissionsv6 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v6"
	emissionsv7 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v7"
	emissionsv8 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v8"
	emissionsv9 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v9"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	mintv2 "github.com/allora-network/allora-chain/x/mint/api/mint/v2"
)

const (
	mintV2TypeURL = "/mint.v2.UpdateParamsRequest"
	v10WorkerURL  = "/emissions.v10.InsertWorkerPayloadRequest"
)

// One AlloraApp is enough for every read-only assertion here; build it once.
var (
	sharedAppOnce sync.Once
	sharedAppInst *AlloraApp
)

func sharedApp(t *testing.T) *AlloraApp {
	t.Helper()
	sharedAppOnce.Do(func() {
		a, err := NewAlloraApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.EmptyAppOptions{})
		require.NoError(t, err)
		sharedAppInst = a
	})
	return sharedAppInst
}

// marshalCodec only serializes tx envelopes; the Anys inside are pre-built, so it
// needs no registered types.
func marshalCodec() codec.Codec { //nolint:ireturn // returns the SDK codec interface by design
	return codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
}

// buildTxBytes wraps msgs into a minimal signed-shaped TxRaw and returns the bytes.
func buildTxBytes(t *testing.T, msgs ...sdk.Msg) []byte {
	t.Helper()
	cdc := marshalCodec()
	anys := make([]*codectypes.Any, 0, len(msgs))
	for _, m := range msgs {
		a, err := codectypes.NewAnyWithValue(m)
		require.NoError(t, err)
		anys = append(anys, a)
	}
	//nolint:exhaustruct // minimal tx envelope; only these fields are read on decode
	bodyBz, err := cdc.Marshal(&txtypes.TxBody{Messages: anys})
	require.NoError(t, err)
	//nolint:exhaustruct // minimal tx envelope
	aiBz, err := cdc.Marshal(&txtypes.AuthInfo{Fee: &txtypes.Fee{GasLimit: 200000}})
	require.NoError(t, err)
	//nolint:exhaustruct // minimal tx envelope
	rawBz, err := cdc.Marshal(&txtypes.TxRaw{BodyBytes: bodyBz, AuthInfoBytes: aiBz, Signatures: [][]byte{{0x01}}})
	require.NoError(t, err)
	return rawBz
}

// historicalV9Worker is a pre-v10 worker payload: the exact shape that stopped
// decoding after the v10 proto bump.
func historicalV9Worker() sdk.Msg {
	//nolint:exhaustruct // only the fields exercised by the nested walk are set
	return &emissionsv9.InsertWorkerPayloadRequest{
		Sender: "allo1sender",
		WorkerDataBundle: &emissionsv9.InputWorkerDataBundle{
			Worker:  "allo1worker",
			Nonce:   &emissionsv3.Nonce{BlockHeight: 1},
			TopicId: 1,
			InferenceForecastsBundle: &emissionsv9.InputInferenceForecastBundle{
				Inference: &emissionsv9.InputInference{
					TopicId: 1, BlockHeight: 1, Inferer: "allo1worker", Value: "1.0",
				},
			},
			InferencesForecastsBundleSignature: []byte{0x01},
			Pubkey:                             "pubkey",
		},
	}
}

type workerPayloadCase struct {
	name         string
	msg          sdk.Msg
	wantFallback bool // true only where the strict decoder is known to reject it
}

// historicalWorkerPayloads builds a worker-insert message for every pre-v10
// emissions version. v2 and v3 carry their own WorkerDataBundle; v4-v8 reuse v3's;
// v9 uses InputWorkerDataBundle. This is the full historical tx surface the query
// path must keep decoding.
func historicalWorkerPayloads() []workerPayloadCase {
	//nolint:exhaustruct // only the fields exercised by the nested walk are set
	v2Bundle := &emissionsv2.WorkerDataBundle{
		Worker:  "allo1worker",
		Nonce:   &emissionsv2.Nonce{BlockHeight: 1},
		TopicId: 1,
		InferenceForecastsBundle: &emissionsv2.InferenceForecastBundle{
			Inference: &emissionsv2.Inference{TopicId: 1, BlockHeight: 1, Inferer: "allo1worker", Value: "1.0"},
		},
		InferencesForecastsBundleSignature: []byte{0x01},
		Pubkey:                             "pubkey",
	}
	//nolint:exhaustruct // v4-v8 nest this exact v3 bundle type
	v3Bundle := &emissionsv3.WorkerDataBundle{
		Worker:  "allo1worker",
		Nonce:   &emissionsv3.Nonce{BlockHeight: 1},
		TopicId: 1,
		InferenceForecastsBundle: &emissionsv3.InferenceForecastBundle{
			Inference: &emissionsv3.Inference{TopicId: 1, BlockHeight: 1, Inferer: "allo1worker", Value: "1.0"},
		},
		InferencesForecastsBundleSignature: []byte{0x01},
		Pubkey:                             "pubkey",
	}
	//nolint:exhaustruct // only Sender and the bundle matter for the decode walk
	return []workerPayloadCase{
		{"v2", &emissionsv2.MsgInsertWorkerPayload{Sender: "allo1sender", WorkerDataBundle: v2Bundle}, false},
		{"v3", &emissionsv3.MsgInsertWorkerPayload{Sender: "allo1sender", WorkerDataBundle: v3Bundle}, false},
		{"v4", &emissionsv4.InsertWorkerPayloadRequest{Sender: "allo1sender", WorkerDataBundle: v3Bundle}, false},
		{"v5", &emissionsv5.InsertWorkerPayloadRequest{Sender: "allo1sender", WorkerDataBundle: v3Bundle}, false},
		{"v6", &emissionsv6.InsertWorkerPayloadRequest{Sender: "allo1sender", WorkerDataBundle: v3Bundle}, false},
		{"v7", &emissionsv7.InsertWorkerPayloadRequest{Sender: "allo1sender", WorkerDataBundle: v3Bundle}, false},
		{"v8", &emissionsv8.InsertWorkerPayloadRequest{Sender: "allo1sender", WorkerDataBundle: v3Bundle}, false},
		{"v9", historicalV9Worker(), true},
	}
}

// Scenario 1: every historical emissions worker payload (v2-v9) decodes on the
// query path, bare and wrapped in a routable authz.MsgExec, and yields the correct
// type URL. v9 additionally must take the tolerant fallback (its nested names were
// reclaimed by v10); the rest need only decode.
func TestQueryDecoderDecodesHistoricalWorkerPayloads(t *testing.T) {
	a := sharedApp(t)
	dec := a.queryTxConfig.TxDecoder()
	for _, tc := range historicalWorkerPayloads() {
		t.Run(tc.name+"_bare", func(t *testing.T) {
			tx, err := dec(buildTxBytes(t, tc.msg))
			require.NoError(t, err)
			require.Equal(t, sdk.MsgTypeURL(tc.msg), sdk.MsgTypeURL(tx.GetMsgs()[0]))
			if tc.wantFallback {
				_, ok := tx.(historicalTx)
				require.True(t, ok, "expected tolerant fallback for %s", tc.name)
			}
		})
		t.Run(tc.name+"_wrapped", func(t *testing.T) {
			inner, err := codectypes.NewAnyWithValue(tc.msg)
			require.NoError(t, err)
			//nolint:exhaustruct // only Grantee and Msgs matter
			exec := &authz.MsgExec{Grantee: "allo1grantee", Msgs: []*codectypes.Any{inner}}
			tx, err := dec(buildTxBytes(t, exec))
			require.NoError(t, err)
			require.Equal(t, "/cosmos.authz.v1beta1.MsgExec", sdk.MsgTypeURL(tx.GetMsgs()[0]))
		})
	}
}

// Scenario 2: the SAME bare v9 bytes are rejected by the consensus decoder. This
// is the guard that the fix does not widen what consensus accepts.
func TestConsensusDecoderRejectsBareHistoricalV9(t *testing.T) {
	a := sharedApp(t)
	_, err := a.txConfig.TxDecoder()(buildTxBytes(t, historicalV9Worker()))
	require.Error(t, err, "consensus decoder must still reject historical payloads")
}

// Scenario 4: a v9 payload wrapped in a routable authz.MsgExec is rejected by the
// consensus decoder. This
// is the exact vector that made the global-registry approach consensus-breaking.
func TestConsensusDecoderRejectsWrappedHistoricalV9(t *testing.T) {
	a := sharedApp(t)
	inner, err := codectypes.NewAnyWithValue(historicalV9Worker())
	require.NoError(t, err)
	//nolint:exhaustruct // only Grantee and Msgs matter
	exec := &authz.MsgExec{Grantee: "allo1grantee", Msgs: []*codectypes.Any{inner}}
	_, err = a.txConfig.TxDecoder()(buildTxBytes(t, exec))
	require.Error(t, err, "consensus decoder must reject a wrapped historical payload")
}

// Scenario 5: a current-format (v10) emissions tx decodes on the query path via the
// strict branch, so it yields the SDK's own tx type, not the tolerant fallback.
func TestQueryDecoderKeepsStrictPathForCurrentTx(t *testing.T) {
	a := sharedApp(t)
	//nolint:exhaustruct // a valid current (v10) msg is all that is needed
	msg := &emissionstypes.InsertWorkerPayloadRequest{Sender: "allo1sender"}
	tx, err := a.queryTxConfig.TxDecoder()(buildTxBytes(t, msg))
	require.NoError(t, err)
	_, isFallback := tx.(historicalTx)
	require.False(t, isFallback, "current txs must take the strict path")
	require.Equal(t, v10WorkerURL, sdk.MsgTypeURL(tx.GetMsgs()[0]))
}

// Scenario 6: the query registry resolves mint.v2 while the consensus registry and
// the process-global gogo registry do not. This proves the mint.v2 addition is
// isolated to the read path and cannot affect consensus.
func TestMintV2QueryRegistryDoesNotAffectConsensus(t *testing.T) {
	a := sharedApp(t)
	_, errQuery := a.queryInterfaceRegistry.Resolve(mintV2TypeURL)
	require.NoError(t, errQuery, "query registry should resolve mint.v2")
	_, errConsensus := a.interfaceRegistry.Resolve(mintV2TypeURL)
	require.Error(t, errConsensus, "consensus registry must not resolve mint.v2")
	require.Nil(t, gogoproto.MessageType("mint.v2.UpdateParamsRequest"),
		"mint.v2 must not leak into the gogo registry unknownproto reads")

	// And a mint.v2 tx decodes on the query path but not on the consensus path.
	//nolint:exhaustruct // only Sender is needed
	bz := buildTxBytes(t, &mintv2.UpdateParamsRequest{Sender: "allo1sender"})
	_, errQ := a.queryTxConfig.TxDecoder()(bz)
	require.NoError(t, errQ)
	_, errC := a.txConfig.TxDecoder()(bz)
	require.Error(t, errC)
}

// Scenario 7: the tolerant decoder is not "accept anything" - garbage and a
// truncated envelope still error.
func TestQueryDecoderRejectsGarbage(t *testing.T) {
	a := sharedApp(t)
	dec := a.queryTxConfig.TxDecoder()
	_, err := dec([]byte{0xff, 0xff, 0xff, 0xff})
	require.Error(t, err)
	good := buildTxBytes(t, historicalV9Worker())
	_, err = dec(good[:len(good)/2])
	require.Error(t, err)
}

// Scenario 8: the tolerant fallback type satisfies the three interfaces the tx
// service asserts on decoded txs: sdk.Tx, intoAny, and protoTxProvider.
func TestHistoricalTxImplementsReadInterfaces(t *testing.T) {
	//nolint:exhaustruct // empty envelope is enough for interface checks
	var h any = historicalTx{tx: &txtypes.Tx{Body: &txtypes.TxBody{}, AuthInfo: &txtypes.AuthInfo{}}}
	_, isTx := h.(sdk.Tx)
	require.True(t, isTx)
	_, isIntoAny := h.(interface{ AsAny() *codectypes.Any })
	require.True(t, isIntoAny)
	_, isProtoTx := h.(interface{ GetProtoTx() *txtypes.Tx })
	require.True(t, isProtoTx)
}

// Scenario 9: AsAny caches the concrete *txtypes.Tx that mkTxResult (GetTx /
// GetTxsEvent) then type-asserts; a plain pack would leave the cache empty.
func TestHistoricalTxAsAnyCachesConcreteTx(t *testing.T) {
	//nolint:exhaustruct // empty envelope is enough
	h := historicalTx{tx: &txtypes.Tx{Body: &txtypes.TxBody{}, AuthInfo: &txtypes.AuthInfo{}}}
	any := h.AsAny()
	cached, ok := any.GetCachedValue().(*txtypes.Tx)
	require.True(t, ok, "cached value must be *txtypes.Tx")
	require.Same(t, h.tx, cached)
}

// Scenario 10: GetMsgsV2 is unsupported on the read path and returns an explicit
// error rather than a nil slice that a caller might treat as "no messages".
func TestHistoricalTxGetMsgsV2Errors(t *testing.T) {
	//nolint:exhaustruct // empty envelope is enough
	h := historicalTx{tx: &txtypes.Tx{Body: &txtypes.TxBody{}}}
	msgs, err := h.GetMsgsV2()
	require.Error(t, err)
	require.Nil(t, msgs)
}

// Scenario 11: the composite routes only the three historical-read methods to the
// tolerant backend and every other method to the strict backend.
func TestCompositeTxServerRouting(t *testing.T) {
	var hits []string
	strict := &recordingTxServer{id: "strict", hits: &hits}
	tolerant := &recordingTxServer{id: "tolerant", hits: &hits}
	cts := compositeTxServer{ServiceServer: strict, tolerant: tolerant}
	ctx := context.Background()

	// exhaustruct is not meaningful for empty request probes
	//nolint:exhaustruct
	_, _ = cts.GetTx(ctx, &txtypes.GetTxRequest{})
	//nolint:exhaustruct
	_, _ = cts.GetTxsEvent(ctx, &txtypes.GetTxsEventRequest{})
	//nolint:exhaustruct
	_, _ = cts.GetBlockWithTxs(ctx, &txtypes.GetBlockWithTxsRequest{})
	//nolint:exhaustruct
	_, _ = cts.Simulate(ctx, &txtypes.SimulateRequest{})
	//nolint:exhaustruct
	_, _ = cts.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{})
	//nolint:exhaustruct
	_, _ = cts.TxDecode(ctx, &txtypes.TxDecodeRequest{})
	//nolint:exhaustruct
	_, _ = cts.TxEncode(ctx, &txtypes.TxEncodeRequest{})
	//nolint:exhaustruct
	_, _ = cts.TxEncodeAmino(ctx, &txtypes.TxEncodeAminoRequest{})
	//nolint:exhaustruct
	_, _ = cts.TxDecodeAmino(ctx, &txtypes.TxDecodeAminoRequest{})

	require.Equal(t, []string{
		"tolerant.GetTx",
		"tolerant.GetTxsEvent",
		"tolerant.GetBlockWithTxs",
		"strict.Simulate",
		"strict.BroadcastTx",
		"strict.TxDecode",
		"strict.TxEncode",
		"strict.TxEncodeAmino",
		"strict.TxDecodeAmino",
	}, hits)
}

// errStub is returned by every recordingTxServer method so the stubs never return
// (nil, nil); callers in the routing test ignore it.
var errStub = errors.New("recording stub")

// recordingTxServer records which backend and method were invoked.
type recordingTxServer struct {
	id   string
	hits *[]string
}

func (r *recordingTxServer) mark(method string) error {
	*r.hits = append(*r.hits, r.id+"."+method)
	return errStub
}

func (r *recordingTxServer) Simulate(context.Context, *txtypes.SimulateRequest) (*txtypes.SimulateResponse, error) {
	return nil, r.mark("Simulate")
}

func (r *recordingTxServer) GetTx(context.Context, *txtypes.GetTxRequest) (*txtypes.GetTxResponse, error) {
	return nil, r.mark("GetTx")
}

func (r *recordingTxServer) BroadcastTx(context.Context, *txtypes.BroadcastTxRequest) (*txtypes.BroadcastTxResponse, error) {
	return nil, r.mark("BroadcastTx")
}

func (r *recordingTxServer) GetTxsEvent(context.Context, *txtypes.GetTxsEventRequest) (*txtypes.GetTxsEventResponse, error) {
	return nil, r.mark("GetTxsEvent")
}

func (r *recordingTxServer) GetBlockWithTxs(context.Context, *txtypes.GetBlockWithTxsRequest) (*txtypes.GetBlockWithTxsResponse, error) {
	return nil, r.mark("GetBlockWithTxs")
}

func (r *recordingTxServer) TxDecode(context.Context, *txtypes.TxDecodeRequest) (*txtypes.TxDecodeResponse, error) {
	return nil, r.mark("TxDecode")
}

func (r *recordingTxServer) TxEncode(context.Context, *txtypes.TxEncodeRequest) (*txtypes.TxEncodeResponse, error) {
	return nil, r.mark("TxEncode")
}

func (r *recordingTxServer) TxEncodeAmino(context.Context, *txtypes.TxEncodeAminoRequest) (*txtypes.TxEncodeAminoResponse, error) {
	return nil, r.mark("TxEncodeAmino")
}

func (r *recordingTxServer) TxDecodeAmino(context.Context, *txtypes.TxDecodeAminoRequest) (*txtypes.TxDecodeAminoResponse, error) {
	return nil, r.mark("TxDecodeAmino")
}
