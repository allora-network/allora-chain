package integration_test

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
)

// Magnitude window the chain applies to Dec values it emits in events. State
// values are clamped the same way before they are compared with an event.
var (
	eventClampMin = alloraMath.MustNewDecFromString("1e-40")
	eventClampMax = alloraMath.MustNewDecFromString("1e40")
)

// maxAuditViolationsReported bounds the failure message of one audit.
const maxAuditViolationsReported = 40

// requiredEventCoverage names what TestExternalTestSuite's flows produce on any
// chain. An event or reason missing here is one a live flow stopped reaching.
var requiredEventCoverage = []string{
	"emissions.v10.EventAddStake",
	"emissions.v10.EventRemoveStake",
	"emissions.v10.EventTopicWeightUpdated",
	"emissions.v10.EventTopicWeightUpdatedAfterStakeChange",
	"TOPIC_STATUS_CHANGE_REASON_ACTIVATED_BY_FUNDING",
	"TOPIC_STATUS_CHANGE_REASON_REACTIVATED",
	"emissions.v10.EventTopicRewardForfeitedNoRewardNonce",
	"EventNonceExpired/ACTOR_TYPE_INFERER_UNSPECIFIED/NONCE_EXPIRED_REASON_CLOSE_FAILED",
	"EventScoresSet/ACTOR_TYPE_INFERER_UNSPECIFIED",
	"EventScoresSet/ACTOR_TYPE_FORECASTER",
	"EventScoresSet/ACTOR_TYPE_REPUTER",
	"EventRewardsSettled/ACTOR_TYPE_INFERER_UNSPECIFIED",
	"EventRewardsSettled/ACTOR_TYPE_REPUTER",
	"EventRewardsSettled/ACTOR_TYPE_FORECASTER",
	"emissions.v10.EventNetworkInferenceInfererWeightsSet",
	"emissions.v10.EventNetworkInferenceForecasterWeightsSet",
	"emissions.v10.EventNetworkInferenceInfererRegretsUsedSet",
	"emissions.v10.EventNetworkInferenceForecasterRegretsUsedSet",
	"emissions.v10.EventDelegateRewardShareUpdated",
	"emissions.v10.EventDelegateRewardDistributed",
	"emissions.v10.EventEpochLabelRegistryFrozen",
	"emissions.v10.EventNetworkInferenceBundle",
	"emissions.v10.EventNetworkLossSet",
	"emissions.v10.EventWorkerLastCommitSet",
	"emissions.v10.EventReputerLastCommitSet",
	"emissions.v10.EventWorkerSubmissionWindowOpened",
	"emissions.v10.EventWorkerSubmissionWindowClosed",
	"emissions.v10.EventReputerSubmissionWindowOpened",
	"emissions.v10.EventReputerSubmissionWindowClosed",
	"emissions.v10.EventActiveInferersSet",
	"emissions.v10.EventActiveForecastersSet",
	"EventActiveForecastersSet/discarded",
	"emissions.v10.EventActiveReputersSet",
}

// requiredMultiLabelEventCoverage names what TestExternalMultiTestSuite's flows
// produce on any chain.
var requiredMultiLabelEventCoverage = []string{
	"emissions.v10.EventAddStake",
	"TOPIC_STATUS_CHANGE_REASON_ACTIVATED_BY_FUNDING",
	"emissions.v10.EventEpochLabelRegistryFrozen",
	"emissions.v10.EventNetworkInferenceBundle",
}

// eventAudit compares the events of a range of blocks with chain state queried
// at the heights the events were emitted. Every rule accounts for everything
// else that happened in the block, so it holds on a chain with active topics.
type eventAudit struct {
	ctx        context.Context
	m          testCommon.TestConfig
	violations []string
	coverage   map[string]int
	skipped    map[string]int

	ecosystemAddr string
	rewardsAddr   string
	pendingAddr   string

	frozenLabels map[string][]string
	frozenCount  map[string]int
	bundleLabels map[string][]string
	topicStatus  map[uint64]bool

	acceptedForecasters map[uint64][]string
	closedTopics        map[uint64]bool
}

// EventAuditStart returns the height a suite's event audit starts from.
func EventAuditStart(m testCommon.TestConfig) int64 {
	height, err := m.Client.BlockHeight(context.Background())
	require.NoError(m.T, err)
	return height
}

// EventLiveChecks audits every block since fromHeight on every node and fails on
// any event that disagrees with chain state, between nodes, or between endpoints.
// requiredCoverage names the events and reasons the suite's flows must produce.
func EventLiveChecks(m testCommon.TestConfig, fromHeight int64, requiredCoverage []string) {
	ctx := context.Background()
	require.NoError(m.T, m.Client.WaitForNextBlock(ctx))
	toHeight, err := m.Client.BlockHeight(ctx)
	require.NoError(m.T, err)
	// Nodes index txs asynchronously after commit. One more block on every node
	// makes the txs of toHeight readable through /tx.
	require.NoError(m.T, m.Client.WaitForBlockHeight(ctx, toHeight+1))

	audit := newEventAudit(ctx, m)
	audit.auditMonthlyResetBlock()
	audit.run(fromHeight, toHeight)
	m.T.Logf("event audit of blocks %d..%d: coverage %s; comparisons skipped %s",
		fromHeight, toHeight, renderCounts(audit.coverage), renderCounts(audit.skipped))

	// Violations are reported before coverage: a wrong payload can also hide an
	// event the coverage list requires, and the payload is the finding.
	reported := audit.violations
	if len(reported) > maxAuditViolationsReported {
		reported = reported[:maxAuditViolationsReported]
	}
	assert.Emptyf(m.T, audit.violations, "%d event violations in blocks %d..%d:\n%s",
		len(audit.violations), fromHeight, toHeight, strings.Join(reported, "\n"))
	missing := missingCoverage(audit.coverage, requiredCoverage)
	require.Empty(m.T, missing, "events the suite's flows must produce were not observed in blocks %d..%d", fromHeight, toHeight)
	require.Empty(m.T, audit.violations)
}

func newEventAudit(ctx context.Context, m testCommon.TestConfig) *eventAudit {
	moduleAddr := func(name string) string {
		addr, err := sdk.Bech32ifyAddressBytes(params.HumanCoinUnit, authtypes.NewModuleAddress(name))
		require.NoError(m.T, err)
		return addr
	}
	return &eventAudit{
		ctx:           ctx,
		m:             m,
		violations:    nil,
		coverage:      make(map[string]int),
		skipped:       make(map[string]int),
		ecosystemAddr: moduleAddr(minttypes.EcosystemModuleName),
		rewardsAddr:   moduleAddr(emissionstypes.AlloraRewardsAccountName),
		pendingAddr:   moduleAddr(emissionstypes.AlloraPendingRewardForDelegatorAccountName),
		frozenLabels:  make(map[string][]string),
		frozenCount:   make(map[string]int),
		bundleLabels:  make(map[string][]string),
		topicStatus:   make(map[uint64]bool),

		acceptedForecasters: make(map[uint64][]string),
		closedTopics:        make(map[uint64]bool),
	}
}

// missingCoverage lists the required coverage keys that were never observed.
func missingCoverage(coverage map[string]int, required []string) []string {
	missing := make([]string, 0)
	for _, key := range required {
		if coverage[key] == 0 {
			missing = append(missing, key)
		}
	}
	return missing
}

// renderCounts prints a count map in sorted key order.
func renderCounts(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for key := range counts { //nolint:maprange // the keys are sorted below
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, key := range keys {
		parts[i] = fmt.Sprintf("%s=%d", key, counts[key])
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func (a *eventAudit) failf(height int64, format string, args ...any) {
	a.violations = append(a.violations, fmt.Sprintf("height %d: %s", height, fmt.Sprintf(format, args...)))
}

// auditMonthlyResetBlock audits EventPreviousPercentageRewardToStakedReputersSet in
// block 1, the one block of a test chain where the monthly reset runs. Block 1 has
// no predecessor, so only the rule for this event applies.
func (a *eventAudit) auditMonthlyResetBlock() {
	const monthlyResetHeight = 1
	block, err := readChainBlock(a.ctx, a.m, monthlyResetHeight, func(format string, args ...any) {
		a.failf(monthlyResetHeight, format, args...)
	})
	require.NoError(a.m.T, err)
	for _, event := range block.events {
		if _, ok := event.typed.(*emissionstypes.EventPreviousPercentageRewardToStakedReputersSet); ok {
			a.coverage[proto.MessageName(event.typed)]++
			a.auditEvent(monthlyResetHeight, newBlockContext(block), event)
		}
	}
}

func (a *eventAudit) run(fromHeight, toHeight int64) {
	for height := fromHeight; height <= toHeight; height++ {
		block, err := readChainBlock(a.ctx, a.m, height, func(format string, args ...any) {
			a.failf(height, format, args...)
		})
		require.NoError(a.m.T, err)
		a.auditBlock(block)
	}
	a.auditFinalTopicStatus(toHeight)
}

func (a *eventAudit) auditBlock(block chainBlock) {
	facts := newBlockContext(block)
	a.auditStakeLedger(block)
	a.auditModuleLedgers(block)
	a.auditRewardPayouts(block)
	a.auditTopicWeights(block, facts)
	a.auditCombinedValuesFromWeights(block.height, facts)
	for i, event := range block.events {
		if event.typed == nil {
			continue
		}
		a.coverage[proto.MessageName(event.typed)]++
		a.auditEpochNonceFields(block.height, event.typed)
		if status, ok := event.typed.(*emissionstypes.EventTopicStatusChanged); ok {
			a.auditTopicStatus(block.height, facts, event.phase, status, topicTouchedAfter(block, i, status.TopicId))
			a.topicStatus[status.TopicId] = status.IsActive
			continue
		}
		a.auditEvent(block.height, facts, event)
	}
}

// topicTouchedAfter reports whether anything after the event at position changed
// or may have changed topicId's weight inputs. The EventFundTopic of the same tx
// does not count: the funding msg credits the revenue before it activates the
// topic and emits this event last.
func topicTouchedAfter(block chainBlock, position int, topicId uint64) bool {
	origin := block.events[position]
	for _, event := range block.events[position+1:] {
		withTopic, ok := event.typed.(interface{ GetTopicId() uint64 })
		if !ok || withTopic.GetTopicId() != topicId {
			continue
		}
		_, isFunding := event.typed.(*emissionstypes.EventFundTopic)
		if isFunding && event.phase == origin.phase && event.txIndex == origin.txIndex {
			continue
		}
		return true
	}
	return false
}

// blockContext holds per-block facts several rules need.
type blockContext struct {
	lastReputerCommit map[uint64]int64
	droppedNonces     map[string]bool
	dripOldRevenue    map[uint64]cosmosMath.Int
	epochWeight       map[uint64]alloraMath.Dec
	networkBundles    map[string]*emissionstypes.EventNetworkInferenceBundle
	networkWeights    map[string]networkWeights
}

func newBlockContext(block chainBlock) blockContext {
	facts := blockContext{
		lastReputerCommit: make(map[uint64]int64),
		droppedNonces:     make(map[string]bool),
		dripOldRevenue:    make(map[uint64]cosmosMath.Int),
		epochWeight:       make(map[uint64]alloraMath.Dec),
		networkBundles:    make(map[string]*emissionstypes.EventNetworkInferenceBundle),
		networkWeights:    make(map[string]networkWeights),
	}
	for _, event := range block.events {
		switch typed := event.typed.(type) {
		case *emissionstypes.EventReputerLastCommitSet:
			facts.lastReputerCommit[typed.TopicId] = typed.Nonce.BlockHeight
		case *emissionstypes.EventNonceExpired:
			facts.droppedNonces[topicNonceKey(typed.TopicId, typed.Nonce)] = true
		case *emissionstypes.EventTopicFeeRevenueDripped:
			facts.dripOldRevenue[typed.TopicId] = typed.OldRevenue
		case *emissionstypes.EventTopicWeightUpdated:
			facts.epochWeight[typed.TopicId] = typed.NewWeight
		case *emissionstypes.EventNetworkInferenceBundle:
			if !typed.OutlierResistant {
				facts.networkBundles[topicNonceKey(typed.TopicId, typed.Nonce)] = typed
			}
		case *emissionstypes.EventNetworkInferenceInfererWeightsSet:
			key := topicNonceKey(typed.TopicId, typed.Nonce)
			weights := facts.networkWeights[key]
			weights.infererAddresses, weights.infererWeights = typed.Addresses, typed.Weights
			facts.networkWeights[key] = weights
		case *emissionstypes.EventNetworkInferenceForecasterWeightsSet:
			key := topicNonceKey(typed.TopicId, typed.Nonce)
			weights := facts.networkWeights[key]
			weights.forecasterAddresses, weights.forecasterWeights = typed.Addresses, typed.Weights
			facts.networkWeights[key] = weights
		}
	}
	return facts
}

func topicNonceKey(topicId uint64, nonce int64) string {
	return fmt.Sprintf("%d/%d", topicId, nonce)
}

// auditEpochNonceFields checks that the deprecated epoch field repeats nonce and
// that nonce names a block the chain has reached.
func (a *eventAudit) auditEpochNonceFields(height int64, typed proto.Message) {
	withNonce, ok := typed.(interface{ GetNonce() int64 })
	if !ok {
		return
	}
	name := proto.MessageName(typed)
	nonce := withNonce.GetNonce()
	if nonce <= 0 || nonce > height {
		a.failf(height, "%s nonce %d is outside 1..%d", name, nonce, height)
	}
	switch deprecated := typed.(type) {
	case interface{ GetBlockHeight() int64 }:
		if deprecated.GetBlockHeight() != nonce {
			a.failf(height, "%s block_height %d differs from nonce %d", name, deprecated.GetBlockHeight(), nonce)
		}
	case interface{ GetNonceBlockHeight() int64 }:
		if deprecated.GetNonceBlockHeight() != nonce {
			a.failf(height, "%s nonce_block_height %d differs from nonce %d", name, deprecated.GetNonceBlockHeight(), nonce)
		}
	}
}

// auditStakeLedger replays the block's stake events in execution order for each
// topic: each event's topic_stake must follow from the stake before it, starting
// at the queried stake of the previous block and ending at the queried stake of
// this block. Weight events report the stake at their point in that replay.
// A removal writes the stake, recomputes the weight and only then emits
// EventRemoveStake, so EventTopicWeightUpdatedAfterStakeChange must carry the
// topic_stake of the removal event that follows it.
func (a *eventAudit) auditStakeLedger(block chainBlock) {
	running := make(map[uint64]cosmosMath.Int)
	awaitingRemoval := make(map[uint64]cosmosMath.Int)
	order := make([]uint64, 0)
	current := func(topicId uint64) cosmosMath.Int {
		stake, seen := running[topicId]
		if !seen {
			stake = a.topicStake(block.height-1, topicId)
			running[topicId] = stake
			order = append(order, topicId)
		}
		return stake
	}
	apply := func(topicId uint64, name string, signedAmount, reported cosmosMath.Int) {
		if expected := current(topicId).Add(signedAmount); !expected.Equal(reported) {
			a.failf(block.height, "%s topic %d topic_stake %s, stake ledger expects %s", name, topicId, reported, expected)
		}
		running[topicId] = reported
	}
	for _, event := range block.events {
		switch typed := event.typed.(type) {
		case *emissionstypes.EventAddStake:
			apply(typed.TopicId, "EventAddStake", typed.Amount, typed.TopicStake)
		case *emissionstypes.EventRemoveStake:
			apply(typed.TopicId, "EventRemoveStake", typed.Amount.Neg(), typed.TopicStake)
			if weightStake, waiting := awaitingRemoval[typed.TopicId]; waiting {
				if !weightStake.Equal(typed.TopicStake) {
					a.failf(block.height, "EventTopicWeightUpdatedAfterStakeChange topic %d topic_stake %s, the removal it belongs to leaves %s", typed.TopicId, weightStake, typed.TopicStake)
				}
				delete(awaitingRemoval, typed.TopicId)
			}
		case *emissionstypes.EventTopicWeightUpdatedAfterStakeChange:
			if _, waiting := awaitingRemoval[typed.TopicId]; waiting {
				a.failf(block.height, "EventTopicWeightUpdatedAfterStakeChange topic %d has no EventRemoveStake before the next weight event", typed.TopicId)
			}
			awaitingRemoval[typed.TopicId] = typed.TopicStake
		case *emissionstypes.EventTopicWeightUpdated:
			if stake := current(typed.TopicId); !stake.Equal(typed.TopicStake) {
				a.failf(block.height, "EventTopicWeightUpdated topic %d topic_stake %s, stake ledger has %s at this point", typed.TopicId, typed.TopicStake, stake)
			}
		}
	}
	for topicId := range awaitingRemoval { //nolint:maprange // any entry is a violation and the order does not matter
		a.failf(block.height, "EventTopicWeightUpdatedAfterStakeChange topic %d has no EventRemoveStake after it", topicId)
	}
	for _, topicId := range order {
		if queried := a.topicStake(block.height, topicId); !queried.Equal(running[topicId]) {
			a.failf(block.height, "last stake event of topic %d reports topic_stake %s, GetTopicStake returns %s", topicId, running[topicId], queried)
		}
	}
}

func (a *eventAudit) topicStake(height int64, topicId uint64) cosmosMath.Int {
	response, err := a.m.Client.QueryEmissions().GetTopicStake(atHeight(a.ctx, height), &emissionstypes.GetTopicStakeRequest{TopicId: topicId})
	if err != nil {
		a.failf(height, "GetTopicStake topic %d: %v", topicId, err)
		return cosmosMath.ZeroInt()
	}
	return response.Amount
}

func (a *eventAudit) balance(height int64, address string) cosmosMath.Int {
	response, err := a.m.Client.QueryBank().Balance(atHeight(a.ctx, height), &banktypes.QueryBalanceRequest{Address: address, Denom: params.BaseCoinUnit})
	if err != nil {
		a.failf(height, "Balance of %s: %v", address, err)
		return cosmosMath.ZeroInt()
	}
	return response.Balance.Amount
}

// bankTransfer is one bank transfer event in the bond denom.
type bankTransfer struct {
	sender    string
	recipient string
	amount    cosmosMath.Int
}

// blockTransfers reads the block's bank transfers from their structured attributes.
func (a *eventAudit) blockTransfers(block chainBlock) []bankTransfer {
	transfers := make([]bankTransfer, 0)
	for _, event := range block.events {
		if event.raw.Type != banktypes.EventTypeTransfer {
			continue
		}
		transfer := bankTransfer{sender: "", recipient: "", amount: cosmosMath.ZeroInt()}
		for _, attribute := range event.raw.Attributes {
			switch attribute.Key {
			case banktypes.AttributeKeySender:
				transfer.sender = attribute.Value
			case banktypes.AttributeKeyRecipient:
				transfer.recipient = attribute.Value
			case sdk.AttributeKeyAmount:
				coins, err := sdk.ParseCoinsNormalized(attribute.Value)
				if err != nil {
					a.failf(block.height, "transfer amount %q: %v", attribute.Value, err)
					continue
				}
				transfer.amount = coins.AmountOf(params.BaseCoinUnit)
			}
		}
		transfers = append(transfers, transfer)
	}
	return transfers
}

// expectedLedgerEvents returns the sums the module ledgers compare with balances.
func expectedLedgerEvents(block chainBlock) (forfeited, delegateDistributed, minted, emitted cosmosMath.Int) {
	forfeited, delegateDistributed = cosmosMath.ZeroInt(), cosmosMath.ZeroInt()
	minted, emitted = cosmosMath.ZeroInt(), cosmosMath.ZeroInt()
	for _, event := range block.events {
		switch typed := event.typed.(type) {
		case *emissionstypes.EventTopicRewardForfeitedNoRewardNonce:
			forfeited = forfeited.Add(typed.Amount)
		case *emissionstypes.EventTopicRewardForfeitedDistributionFailed:
			forfeited = forfeited.Add(typed.Amount)
		case *emissionstypes.EventDelegateRewardDistributed:
			delegateDistributed = delegateDistributed.Add(typed.Amount)
		case *minttypes.EventEcosystemTokenMintSet:
			minted = minted.Add(typed.TokenAmount)
		case *minttypes.EventRewardCurrentBlockEmission:
			emitted = emitted.Add(typed.TokenAmount)
		}
	}
	return forfeited, delegateDistributed, minted, emitted
}

// auditModuleLedgers ties the amounts of forfeit, delegate reward and mint events
// to queried balances. Each account's balance change, minus every transfer the
// event does not describe, must equal what the events report. Blocks without such
// an event are checked too, so a movement with no event fails as well.
func (a *eventAudit) auditModuleLedgers(block chainBlock) {
	transfers := a.blockTransfers(block)
	forfeited, delegateDistributed, minted, emitted := expectedLedgerEvents(block)
	mintAddr, err := sdk.Bech32ifyAddressBytes(params.HumanCoinUnit, authtypes.NewModuleAddress(minttypes.ModuleName))
	require.NoError(a.m.T, err)

	ecosystemDelta := a.balance(block.height, a.ecosystemAddr).Sub(a.balance(block.height-1, a.ecosystemAddr))
	pendingDelta := a.balance(block.height, a.pendingAddr).Sub(a.balance(block.height-1, a.pendingAddr))
	ecosystemOtherFlows := cosmosMath.ZeroInt()
	pendingOtherFlows := cosmosMath.ZeroInt()
	mintedByBank := cosmosMath.ZeroInt()
	emittedByBank := cosmosMath.ZeroInt()
	for _, transfer := range transfers {
		if transfer.sender == a.ecosystemAddr {
			ecosystemOtherFlows = ecosystemOtherFlows.Sub(transfer.amount)
			if transfer.recipient == a.rewardsAddr {
				emittedByBank = emittedByBank.Add(transfer.amount)
			}
		}
		if transfer.recipient == a.ecosystemAddr && transfer.sender != a.rewardsAddr {
			ecosystemOtherFlows = ecosystemOtherFlows.Add(transfer.amount)
			if transfer.sender == mintAddr {
				mintedByBank = mintedByBank.Add(transfer.amount)
			}
		}
		if transfer.sender == a.pendingAddr {
			pendingOtherFlows = pendingOtherFlows.Sub(transfer.amount)
		}
		if transfer.recipient == a.pendingAddr && transfer.sender != a.rewardsAddr {
			pendingOtherFlows = pendingOtherFlows.Add(transfer.amount)
		}
	}
	if explained := ecosystemDelta.Sub(ecosystemOtherFlows); !explained.Equal(forfeited) {
		a.failf(block.height, "forfeit events report %s, ecosystem balance change %s less other flows %s is %s", forfeited, ecosystemDelta, ecosystemOtherFlows, explained)
	}
	if explained := pendingDelta.Sub(pendingOtherFlows); !explained.Equal(delegateDistributed) {
		a.failf(block.height, "EventDelegateRewardDistributed reports %s, pending account balance change %s less other flows %s is %s", delegateDistributed, pendingDelta, pendingOtherFlows, explained)
	}
	if !minted.Equal(mintedByBank) {
		a.failf(block.height, "EventEcosystemTokenMintSet reports %s, bank moved %s from mint to ecosystem", minted, mintedByBank)
	}
	if !emitted.Equal(emittedByBank) {
		a.failf(block.height, "EventRewardCurrentBlockEmission reports %s, bank moved %s from ecosystem to rewards", emitted, emittedByBank)
	}
}

// auditTopicWeights checks the block's last weight event of each topic against
// the stored weight, and that the two weight events come from the right path.
func (a *eventAudit) auditTopicWeights(block chainBlock, facts blockContext) {
	lastWeight := make(map[uint64]alloraMath.Dec)
	order := make([]uint64, 0)
	record := func(topicId uint64, weight alloraMath.Dec) {
		if _, seen := lastWeight[topicId]; !seen {
			order = append(order, topicId)
		}
		lastWeight[topicId] = weight
	}
	checkPayload := func(name string, phase eventPhase, topicId uint64, feeRevenue cosmosMath.Int) {
		if phase != phaseEndBlock {
			a.failf(block.height, "%s topic %d emitted during %s, every path that emits it runs in EndBlock", name, topicId, phase)
		}
		revenue, dripped := facts.dripOldRevenue[topicId]
		if !dripped {
			response, err := a.m.Client.QueryEmissions().GetTopicFeeRevenue(atHeight(a.ctx, block.height), &emissionstypes.GetTopicFeeRevenueRequest{TopicId: topicId})
			if err != nil {
				a.failf(block.height, "GetTopicFeeRevenue topic %d: %v", topicId, err)
				return
			}
			revenue = response.FeeRevenue
		}
		if !feeRevenue.Equal(revenue) {
			a.failf(block.height, "%s topic %d topic_fee_revenue %s, chain state has %s", name, topicId, feeRevenue, revenue)
		}
	}
	for _, event := range block.events {
		switch typed := event.typed.(type) {
		case *emissionstypes.EventTopicWeightUpdated:
			checkPayload("EventTopicWeightUpdated", event.phase, typed.TopicId, typed.TopicFeeRevenue)
			record(typed.TopicId, typed.NewWeight)
		case *emissionstypes.EventTopicWeightUpdatedAfterStakeChange:
			checkPayload("EventTopicWeightUpdatedAfterStakeChange", event.phase, typed.TopicId, typed.TopicFeeRevenue)
			record(typed.TopicId, typed.NewWeight)
		case *emissionstypes.EventTopicFeeRevenueDripped:
			response, err := a.m.Client.QueryEmissions().GetTopicFeeRevenue(atHeight(a.ctx, block.height), &emissionstypes.GetTopicFeeRevenueRequest{TopicId: typed.TopicId})
			if err != nil {
				a.failf(block.height, "GetTopicFeeRevenue topic %d: %v", typed.TopicId, err)
				continue
			}
			if !typed.NewRevenue.Equal(response.FeeRevenue) {
				a.failf(block.height, "EventTopicFeeRevenueDripped topic %d new_revenue %s, GetTopicFeeRevenue returns %s", typed.TopicId, typed.NewRevenue, response.FeeRevenue)
			}
		}
	}
	for _, topicId := range order {
		response, err := a.m.Client.QueryEmissions().GetPreviousTopicWeight(atHeight(a.ctx, block.height), &emissionstypes.GetPreviousTopicWeightRequest{TopicId: topicId})
		if err != nil {
			a.failf(block.height, "GetPreviousTopicWeight topic %d: %v", topicId, err)
			continue
		}
		a.requireDecEqual(block.height, fmt.Sprintf("last weight event of topic %d vs GetPreviousTopicWeight", topicId), lastWeight[topicId], response.Weight)
	}
}

// requireDecEqual compares an event Dec with its state value under the event clamp.
func (a *eventAudit) requireDecEqual(height int64, what string, event, state alloraMath.Dec) {
	clamped := alloraMath.ClampMagnitude(state, eventClampMin, eventClampMax)
	if event.IsNaN() != clamped.IsNaN() || !event.Equal(clamped) {
		a.failf(height, "%s: event has %s, chain state has %s", what, event, clamped)
	}
}

// expectedRewardNonce returns the nonce EmitRewards settles for topicId in this
// block: the one the same block's reputer close set, else the stored one.
func (a *eventAudit) expectedRewardNonce(height int64, facts blockContext, topicId uint64) int64 {
	if nonce, closed := facts.lastReputerCommit[topicId]; closed {
		return nonce
	}
	response, err := a.m.Client.QueryEmissions().GetTopicRewardNonce(atHeight(a.ctx, height-1), &emissionstypes.GetTopicRewardNonceRequest{TopicId: topicId})
	if err != nil {
		a.failf(height, "GetTopicRewardNonce topic %d: %v", topicId, err)
		return 0
	}
	return response.Nonce
}

func (a *eventAudit) requireRewardNonce(height int64, facts blockContext, name string, topicId uint64, nonce int64) {
	if expected := a.expectedRewardNonce(height, facts, topicId); nonce != expected {
		a.failf(height, "%s topic %d nonce %d, the topic's reward nonce is %d", name, topicId, nonce, expected)
	}
}

func (a *eventAudit) getTopic(height int64, topicId uint64) (*emissionstypes.GetTopicResponse, bool) {
	response, err := a.m.Client.QueryEmissions().GetTopic(atHeight(a.ctx, height), &emissionstypes.GetTopicRequest{TopicId: topicId})
	if err != nil {
		a.failf(height, "GetTopic %d: %v", topicId, err)
		return nil, false
	}
	return response, true
}

func (a *eventAudit) isTopicActive(height int64, topicId uint64) bool {
	response, err := a.m.Client.QueryEmissions().IsTopicActive(atHeight(a.ctx, height), &emissionstypes.IsTopicActiveRequest{TopicId: topicId})
	if err != nil {
		// A topic created during the audited range does not exist before its creation.
		return false
	}
	return response.IsActive
}

func (a *eventAudit) minTopicWeights(height int64) []alloraMath.Dec {
	weights := make([]alloraMath.Dec, 0, 2)
	for _, h := range []int64{height - 1, height} {
		response, err := a.m.Client.QueryEmissions().GetParams(atHeight(a.ctx, h), &emissionstypes.GetParamsRequest{})
		if err != nil {
			a.failf(height, "GetParams at %d: %v", h, err)
			continue
		}
		weights = append(weights, response.Params.MinTopicWeight)
	}
	return weights
}

// auditTopicStatus checks one EventTopicStatusChanged against params, the weight
// events of the block and the topic's queried status.
func (a *eventAudit) auditTopicStatus(height int64, facts blockContext, phase eventPhase, event *emissionstypes.EventTopicStatusChanged, touchedAfter bool) {
	a.coverage[event.Reason.String()]++

	inForce := false
	for _, minWeight := range a.minTopicWeights(height) {
		inForce = inForce || event.MinWeight.Equal(minWeight)
	}
	if !inForce {
		a.failf(height, "EventTopicStatusChanged topic %d min_weight %s is not the MinTopicWeight param in force", event.TopicId, event.MinWeight)
	}

	epochWeight, hasEpochWeight := facts.epochWeight[event.TopicId]
	requireEpochWeight := func() {
		if !hasEpochWeight {
			a.failf(height, "%s topic %d has no EventTopicWeightUpdated in its block", event.Reason, event.TopicId)
		} else if !event.Weight.Equal(epochWeight) {
			a.failf(height, "%s topic %d weight %s, EventTopicWeightUpdated has %s", event.Reason, event.TopicId, event.Weight, epochWeight)
		}
	}
	requireActive := func(expected bool) {
		if event.IsActive != expected {
			a.failf(height, "%s topic %d is_active %t", event.Reason, event.TopicId, event.IsActive)
		}
	}

	switch event.Reason {
	case emissionstypes.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_ACTIVATED_BY_FUNDING:
		requireActive(true)
		if phase != phaseTx {
			a.failf(height, "%s topic %d emitted during %s", event.Reason, event.TopicId, phase)
		}
		if a.isTopicActive(height-1, event.TopicId) {
			a.failf(height, "%s topic %d was already active at the previous height", event.Reason, event.TopicId)
		}
		if event.Weight.Lt(event.ThresholdWeight) {
			a.failf(height, "%s topic %d weight %s below threshold_weight %s", event.Reason, event.TopicId, event.Weight, event.ThresholdWeight)
		}
		a.requireFreeSlotThreshold(height, event)
		// GetTopic recomputes the weight from state, so the comparison is exact only
		// when nothing touched the topic after the activation.
		if touchedAfter {
			a.skipped["activation weight: topic touched later in block"]++
		} else if topic, ok := a.getTopic(height, event.TopicId); ok {
			queried, err := alloraMath.NewDecFromString(topic.Weight)
			if err != nil {
				a.failf(height, "GetTopic %d weight %q: %v", event.TopicId, topic.Weight, err)
			} else {
				a.requireDecEqual(height, fmt.Sprintf("%s topic %d weight vs GetTopic", event.Reason, event.TopicId), event.Weight, queried)
			}
		}
	case emissionstypes.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_REACTIVATED:
		requireActive(true)
		requireEpochWeight()
		a.requireFreeSlotThreshold(height, event)
		if !a.isTopicActive(height-1, event.TopicId) {
			a.failf(height, "%s topic %d was not active at the previous height", event.Reason, event.TopicId)
		}
		if topic, ok := a.getTopic(height, event.TopicId); ok {
			churn, err := a.m.Client.QueryEmissions().GetNextChurningBlockByTopicId(atHeight(a.ctx, height), &emissionstypes.GetNextChurningBlockByTopicIdRequest{TopicId: event.TopicId})
			if err != nil {
				a.failf(height, "GetNextChurningBlockByTopicId %d: %v", event.TopicId, err)
			} else if expected := height + topic.Topic.EpochLength; churn.BlockHeight != expected {
				a.failf(height, "%s topic %d next churning block %d, expected %d", event.Reason, event.TopicId, churn.BlockHeight, expected)
			}
		}
	case emissionstypes.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_WEIGHT_BELOW_MINIMUM:
		requireActive(false)
		requireEpochWeight()
		if !event.ThresholdWeight.Equal(event.MinWeight) {
			a.failf(height, "%s topic %d threshold_weight %s differs from min_weight %s", event.Reason, event.TopicId, event.ThresholdWeight, event.MinWeight)
		}
		if !event.Weight.Lt(event.MinWeight) {
			a.failf(height, "%s topic %d weight %s is not below min_weight %s", event.Reason, event.TopicId, event.Weight, event.MinWeight)
		}
	case emissionstypes.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_REACTIVATION_FAILED:
		requireActive(false)
		if !event.Weight.Lt(event.ThresholdWeight) {
			a.failf(height, "%s topic %d weight %s is not below threshold_weight %s", event.Reason, event.TopicId, event.Weight, event.ThresholdWeight)
		}
	case emissionstypes.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_EVICTED_BY_HIGHER_WEIGHT:
		requireActive(false)
		if event.ThresholdWeight.Lt(event.Weight) {
			a.failf(height, "%s topic %d challenger weight %s below the evicted weight %s", event.Reason, event.TopicId, event.ThresholdWeight, event.Weight)
		}
	case emissionstypes.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_REACTIVATION_ERROR:
		requireActive(false)
		if !event.ThresholdWeight.IsZero() {
			a.failf(height, "%s topic %d threshold_weight %s, no comparison happened", event.Reason, event.TopicId, event.ThresholdWeight)
		}
	default:
		a.failf(height, "EventTopicStatusChanged topic %d has reason %s", event.TopicId, event.Reason)
	}
}

// requireFreeSlotThreshold checks threshold_weight of an activation: MinTopicWeight
// for a free slot. An activation into a full block is paired with an eviction in
// the same block and carries the block's lowest weight, which no query returns.
func (a *eventAudit) requireFreeSlotThreshold(height int64, event *emissionstypes.EventTopicStatusChanged) {
	if event.ThresholdWeight.Equal(event.MinWeight) {
		return
	}
	a.skipped["activation threshold: full block slot"]++
	if event.Weight.Lt(event.ThresholdWeight) {
		a.failf(height, "%s topic %d took a full block's slot with weight %s below the lowest weight %s", event.Reason, event.TopicId, event.Weight, event.ThresholdWeight)
	}
}

// auditFinalTopicStatus replays the audited status events and compares the
// resulting status of each topic with the queried one, so a status change with
// no event fails.
func (a *eventAudit) auditFinalTopicStatus(toHeight int64) {
	for topicId, active := range a.topicStatus { //nolint:maprange // every entry is checked and the order does not matter
		if queried := a.isTopicActive(toHeight, topicId); queried != active {
			a.failf(toHeight, "status events end with topic %d active=%t, IsTopicActive returns %t", topicId, active, queried)
		}
	}
}

// auditEvent applies the rule for one typed event.
func (a *eventAudit) auditEvent(height int64, facts blockContext, event chainEvent) {
	switch typed := event.typed.(type) {
	case *emissionstypes.EventTopicRewardForfeitedNoRewardNonce:
		a.auditForfeit(height, "EventTopicRewardForfeitedNoRewardNonce", typed.TopicId, typed.Epoch)
		if nonce := a.expectedRewardNonce(height, facts, typed.TopicId); nonce != 0 {
			a.failf(height, "EventTopicRewardForfeitedNoRewardNonce topic %d has reward nonce %d", typed.TopicId, nonce)
		}
	case *emissionstypes.EventTopicRewardForfeitedDistributionFailed:
		a.auditForfeit(height, "EventTopicRewardForfeitedDistributionFailed", typed.TopicId, typed.Epoch)
		a.requireRewardNonce(height, facts, "EventTopicRewardForfeitedDistributionFailed", typed.TopicId, typed.RewardNonce)
	case *emissionstypes.EventRewardPayoutFailed:
		a.coverage[typed.Reason.String()]++
		a.requireRewardNonce(height, facts, "EventRewardPayoutFailed", typed.TopicId, typed.Nonce)
	case *emissionstypes.EventNonceExpired:
		a.auditNonceExpired(height, typed)
	case *emissionstypes.EventInsertForecasterPayload:
		a.recordForecasterPayload(typed)
	case *emissionstypes.EventScoresSet:
		a.auditScores(height, facts, typed)
	case *emissionstypes.EventRewardsSettled:
		a.coverage["EventRewardsSettled/"+typed.ActorType.String()]++
		a.requireRewardNonce(height, facts, "EventRewardsSettled", typed.TopicId, typed.Nonce)
		if typed.BlockHeightTx != height {
			a.failf(height, "EventRewardsSettled topic %d block_height_tx %d", typed.TopicId, typed.BlockHeightTx)
		}
	case *emissionstypes.EventForecastTaskScoreSet:
		a.requireRewardNonce(height, facts, "EventForecastTaskScoreSet", typed.TopicId, typed.Nonce)
	case *emissionstypes.EventDelegateRewardShareUpdated:
		a.requireRewardNonce(height, facts, "EventDelegateRewardShareUpdated", typed.TopicId, typed.Nonce)
		response, err := a.m.Client.QueryEmissions().GetDelegateRewardPerShare(atHeight(a.ctx, height), &emissionstypes.GetDelegateRewardPerShareRequest{TopicId: typed.TopicId, Reputer: typed.Reputer})
		if err != nil {
			a.failf(height, "GetDelegateRewardPerShare topic %d: %v", typed.TopicId, err)
		} else {
			a.requireDecEqual(height, "EventDelegateRewardShareUpdated reward_per_share", typed.RewardPerShare, response.RewardPerShare)
		}
	case *emissionstypes.EventDelegateRewardDistributed:
		a.requireRewardNonce(height, facts, "EventDelegateRewardDistributed", typed.TopicId, typed.Nonce)
	case *emissionstypes.EventEpochLabelRegistryFrozen:
		a.auditLabelRegistry(height, facts, typed)
	case *emissionstypes.EventNetworkInferenceBundle:
		a.auditNetworkInferenceBundle(height, facts, typed)
	case *emissionstypes.EventNetworkLossSet:
		a.auditNetworkLoss(height, typed)
	case *emissionstypes.EventWorkerLastCommitSet:
		response, err := a.m.Client.QueryEmissions().GetTopicLastWorkerCommitInfo(atHeight(a.ctx, height), &emissionstypes.GetTopicLastWorkerCommitInfoRequest{TopicId: typed.TopicId})
		if err != nil {
			a.failf(height, "GetTopicLastWorkerCommitInfo topic %d: %v", typed.TopicId, err)
		} else {
			a.requireLastCommit(height, "EventWorkerLastCommitSet", typed.TopicId, typed.BlockHeight, typed.Nonce.BlockHeight, response.LastCommit.BlockHeight, response.LastCommit.Nonce.BlockHeight)
		}
	case *emissionstypes.EventReputerLastCommitSet:
		response, err := a.m.Client.QueryEmissions().GetTopicLastReputerCommitInfo(atHeight(a.ctx, height), &emissionstypes.GetTopicLastReputerCommitInfoRequest{TopicId: typed.TopicId})
		if err != nil {
			a.failf(height, "GetTopicLastReputerCommitInfo topic %d: %v", typed.TopicId, err)
		} else {
			a.requireLastCommit(height, "EventReputerLastCommitSet", typed.TopicId, typed.BlockHeight, typed.Nonce.BlockHeight, response.LastCommit.BlockHeight, response.LastCommit.Nonce.BlockHeight)
		}
	case *emissionstypes.EventPreviousPercentageRewardToStakedReputersSet:
		if typed.BlockHeight != height {
			a.failf(height, "EventPreviousPercentageRewardToStakedReputersSet block_height %d", typed.BlockHeight)
		}
		response, err := a.m.Client.QueryEmissions().GetPreviousPercentageRewardToStakedReputers(atHeight(a.ctx, height), &emissionstypes.GetPreviousPercentageRewardToStakedReputersRequest{})
		if err != nil {
			a.failf(height, "GetPreviousPercentageRewardToStakedReputers: %v", err)
		} else {
			a.requireDecEqual(height, "EventPreviousPercentageRewardToStakedReputersSet percentage", typed.Percentage, response.PercentageReward)
		}
	case *minttypes.EventEcosystemTokenMintSet:
		if typed.BlockHeight != uint64(height) { //nolint:gosec // G115: heights of a running chain are positive
			a.failf(height, "EventEcosystemTokenMintSet block_height %d", typed.BlockHeight)
		}
	case *minttypes.EventRewardCurrentBlockEmission:
		if typed.BlockHeight != uint64(height) { //nolint:gosec // G115: heights of a running chain are positive
			a.failf(height, "EventRewardCurrentBlockEmission block_height %d", typed.BlockHeight)
		}
	case *emissionstypes.EventListeningCoefficientsSet:
		a.auditListeningCoefficients(height, facts, typed)
	case *emissionstypes.EventInfererNetworkRegretSet:
		a.auditRegrets(height, "EventInfererNetworkRegretSet", typed.TopicId, typed.Nonce, typed.Addresses, typed.Regrets, func(actor string) (*emissionstypes.TimestampedValue, error) {
			response, err := a.m.Client.QueryEmissions().GetInfererNetworkRegret(atHeight(a.ctx, height), &emissionstypes.GetInfererNetworkRegretRequest{TopicId: typed.TopicId, ActorId: actor})
			if err != nil {
				return nil, err
			}
			return response.Regret, nil
		})
	case *emissionstypes.EventForecasterNetworkRegretSet:
		a.auditRegrets(height, "EventForecasterNetworkRegretSet", typed.TopicId, typed.Nonce, typed.Addresses, typed.Regrets, func(actor string) (*emissionstypes.TimestampedValue, error) {
			response, err := a.m.Client.QueryEmissions().GetForecasterNetworkRegret(atHeight(a.ctx, height), &emissionstypes.GetForecasterNetworkRegretRequest{TopicId: typed.TopicId, Worker: actor})
			if err != nil {
				return nil, err
			}
			return response.Regret, nil
		})
	case *emissionstypes.EventNaiveInfererNetworkRegretSet:
		a.auditRegrets(height, "EventNaiveInfererNetworkRegretSet", typed.TopicId, typed.Nonce, typed.Addresses, typed.Regrets, func(actor string) (*emissionstypes.TimestampedValue, error) {
			response, err := a.m.Client.QueryEmissions().GetNaiveInfererNetworkRegret(atHeight(a.ctx, height), &emissionstypes.GetNaiveInfererNetworkRegretRequest{TopicId: typed.TopicId, Inferer: actor})
			if err != nil {
				return nil, err
			}
			return response.Regret, nil
		})
	case *emissionstypes.EventRegretStdNormSet:
		a.requireReputerNonceOpen(height, "EventRegretStdNormSet", typed.TopicId, typed.Nonce)
		response, err := a.m.Client.QueryEmissions().GetLatestRegretStdNorm(atHeight(a.ctx, height), &emissionstypes.GetLatestRegretStdNormRequest{TopicId: typed.TopicId})
		if err != nil {
			a.failf(height, "GetLatestRegretStdNorm topic %d: %v", typed.TopicId, err)
		} else {
			a.requireDecEqual(height, "EventRegretStdNormSet stdnorm", typed.Stdnorm, response.Value)
		}
	case *emissionstypes.EventInfererWeightsSet:
		a.requireReputerNonceOpen(height, "EventInfererWeightsSet", typed.TopicId, typed.Nonce)
		for i, actor := range typed.Addresses {
			response, err := a.m.Client.QueryEmissions().GetLatestInfererWeight(atHeight(a.ctx, height), &emissionstypes.GetLatestInfererWeightRequest{TopicId: typed.TopicId, ActorId: actor})
			if err != nil {
				a.failf(height, "GetLatestInfererWeight topic %d: %v", typed.TopicId, err)
				continue
			}
			a.requireDecEqual(height, "EventInfererWeightsSet weight of "+actor, typed.Weights[i], response.Weight)
		}
	case *emissionstypes.EventForecasterWeightsSet:
		a.requireReputerNonceOpen(height, "EventForecasterWeightsSet", typed.TopicId, typed.Nonce)
		for i, actor := range typed.Addresses {
			response, err := a.m.Client.QueryEmissions().GetLatestForecasterWeight(atHeight(a.ctx, height), &emissionstypes.GetLatestForecasterWeightRequest{TopicId: typed.TopicId, ActorId: actor})
			if err != nil {
				a.failf(height, "GetLatestForecasterWeight topic %d: %v", typed.TopicId, err)
				continue
			}
			a.requireDecEqual(height, "EventForecasterWeightsSet weight of "+actor, typed.Weights[i], response.Weight)
		}
	case *emissionstypes.EventTopicInitialRegretSet:
		a.requireReputerNonceOpen(height, "EventTopicInitialRegretSet", typed.TopicId, typed.Nonce)
	case *emissionstypes.EventTopicInitialEmaScoreSet:
		a.requireRewardNonce(height, facts, "EventTopicInitialEmaScoreSet", typed.TopicId, typed.Nonce)
		a.auditInitialEmaScore(height, typed)
	case *emissionstypes.EventActiveReputersSet:
		a.requireReputerNonceOpen(height, "EventActiveReputersSet", typed.TopicId, typed.Nonce)
		a.auditActiveReputers(height, facts, typed)
	case *emissionstypes.EventActiveInferersSet:
		a.requireWorkerNonceOpen(height, "EventActiveInferersSet", typed.TopicId, typed.Nonce)
		a.auditActiveInferers(height, typed)
	case *emissionstypes.EventActiveForecastersSet:
		a.requireWorkerNonceOpen(height, "EventActiveForecastersSet", typed.TopicId, typed.Nonce)
		a.auditActiveForecasters(height, typed)
	case *emissionstypes.EventNetworkInferenceInfererWeightsSet:
		a.requireWorkerNonceOpen(height, "EventNetworkInferenceInfererWeightsSet", typed.TopicId, typed.Nonce)
	case *emissionstypes.EventNetworkInferenceForecasterWeightsSet:
		a.requireWorkerNonceOpen(height, "EventNetworkInferenceForecasterWeightsSet", typed.TopicId, typed.Nonce)
	case *emissionstypes.EventNetworkInferenceInfererRegretsUsedSet:
		a.requireWorkerNonceOpen(height, "EventNetworkInferenceInfererRegretsUsedSet", typed.TopicId, typed.Nonce)
		a.auditRegretsUsed(height, "EventNetworkInferenceInfererRegretsUsedSet", typed.TopicId, typed.Addresses, typed.Regrets, func(actor string) (*emissionstypes.TimestampedValue, error) {
			response, err := a.m.Client.QueryEmissions().GetInfererNetworkRegret(atHeight(a.ctx, height), &emissionstypes.GetInfererNetworkRegretRequest{TopicId: typed.TopicId, ActorId: actor})
			if err != nil {
				return nil, err
			}
			return response.Regret, nil
		})
	case *emissionstypes.EventNetworkInferenceForecasterRegretsUsedSet:
		a.requireWorkerNonceOpen(height, "EventNetworkInferenceForecasterRegretsUsedSet", typed.TopicId, typed.Nonce)
		a.auditRegretsUsed(height, "EventNetworkInferenceForecasterRegretsUsedSet", typed.TopicId, typed.Addresses, typed.Regrets, func(actor string) (*emissionstypes.TimestampedValue, error) {
			response, err := a.m.Client.QueryEmissions().GetForecasterNetworkRegret(atHeight(a.ctx, height), &emissionstypes.GetForecasterNetworkRegretRequest{TopicId: typed.TopicId, Worker: actor})
			if err != nil {
				return nil, err
			}
			return response.Regret, nil
		})
	case *emissionstypes.EventWorkerSubmissionWindowOpened:
		a.auditWorkerWindowOpened(height, typed)
	case *emissionstypes.EventWorkerSubmissionWindowClosed:
		a.requireWorkerNonceOpen(height, "EventWorkerSubmissionWindowClosed", typed.TopicId, typed.Nonce)
		if a.isWorkerNonceUnfulfilled(height, typed.TopicId, typed.Nonce) {
			a.failf(height, "EventWorkerSubmissionWindowClosed topic %d nonce %d is still unfulfilled", typed.TopicId, typed.Nonce)
		}
		a.resetForecasterPayloads(typed.TopicId)
	case *emissionstypes.EventReputerSubmissionWindowOpened:
		a.auditReputerWindowOpened(height, typed)
	case *emissionstypes.EventReputerSubmissionWindowClosed:
		a.requireReputerNonceOpen(height, "EventReputerSubmissionWindowClosed", typed.TopicId, typed.Nonce)
		if a.isReputerNonceUnfulfilled(height, typed.TopicId, typed.Nonce) {
			a.failf(height, "EventReputerSubmissionWindowClosed topic %d nonce %d is still unfulfilled", typed.TopicId, typed.Nonce)
		}
	}
}

// auditForfeit checks a forfeit's epoch. The amount is checked by auditModuleLedgers.
func (a *eventAudit) auditForfeit(height int64, name string, topicId uint64, epoch int64) {
	if topic, ok := a.getTopic(height, topicId); ok && topic.Topic.EpochLastEnded != epoch {
		a.failf(height, "%s topic %d epoch %d, EpochLastEnded is %d", name, topicId, epoch, topic.Topic.EpochLastEnded)
	}
	if epoch != height {
		a.failf(height, "%s topic %d epoch %d is not the height of the block rewarding the topic", name, topicId, epoch)
	}
}

func (a *eventAudit) isWorkerNonceUnfulfilled(height int64, topicId uint64, nonce int64) bool {
	response, err := a.m.Client.QueryEmissions().IsWorkerNonceUnfulfilled(atHeight(a.ctx, height), &emissionstypes.IsWorkerNonceUnfulfilledRequest{TopicId: topicId, BlockHeight: nonce})
	if err != nil {
		a.failf(height, "IsWorkerNonceUnfulfilled topic %d nonce %d: %v", topicId, nonce, err)
		return false
	}
	return response.IsWorkerNonceUnfulfilled
}

func (a *eventAudit) isReputerNonceUnfulfilled(height int64, topicId uint64, nonce int64) bool {
	response, err := a.m.Client.QueryEmissions().IsReputerNonceUnfulfilled(atHeight(a.ctx, height), &emissionstypes.IsReputerNonceUnfulfilledRequest{TopicId: topicId, BlockHeight: nonce})
	if err != nil {
		a.failf(height, "IsReputerNonceUnfulfilled topic %d nonce %d: %v", topicId, nonce, err)
		return false
	}
	return response.IsReputerNonceUnfulfilled
}

// requireWorkerNonceOpen checks that nonce was an open worker nonce when the block began.
func (a *eventAudit) requireWorkerNonceOpen(height int64, name string, topicId uint64, nonce int64) {
	if !a.isWorkerNonceUnfulfilled(height-1, topicId, nonce) {
		a.failf(height, "%s topic %d nonce %d was not an open worker nonce at the previous height", name, topicId, nonce)
	}
}

// requireReputerNonceOpen checks that nonce was an open reputer nonce when the block began.
func (a *eventAudit) requireReputerNonceOpen(height int64, name string, topicId uint64, nonce int64) {
	if !a.isReputerNonceUnfulfilled(height-1, topicId, nonce) {
		a.failf(height, "%s topic %d nonce %d was not an open reputer nonce at the previous height", name, topicId, nonce)
	}
}

// auditNonceExpired checks that the nonce was open before the block and gone after it.
func (a *eventAudit) auditNonceExpired(height int64, event *emissionstypes.EventNonceExpired) {
	a.coverage[fmt.Sprintf("EventNonceExpired/%s/%s", event.ActorType, event.Reason)]++
	if event.Reason == emissionstypes.NonceExpiredReason_NONCE_EXPIRED_REASON_UNSPECIFIED {
		a.failf(height, "EventNonceExpired topic %d nonce %d has no reason", event.TopicId, event.Nonce)
	}
	switch event.ActorType {
	case emissionstypes.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED:
		a.requireWorkerNonceOpen(height, "EventNonceExpired", event.TopicId, event.Nonce)
		if a.isWorkerNonceUnfulfilled(height, event.TopicId, event.Nonce) {
			a.failf(height, "EventNonceExpired topic %d worker nonce %d is still unfulfilled", event.TopicId, event.Nonce)
		}
	case emissionstypes.ActorType_ACTOR_TYPE_REPUTER:
		a.requireReputerNonceOpen(height, "EventNonceExpired", event.TopicId, event.Nonce)
		if a.isReputerNonceUnfulfilled(height, event.TopicId, event.Nonce) {
			a.failf(height, "EventNonceExpired topic %d reputer nonce %d is still unfulfilled", event.TopicId, event.Nonce)
		}
	case emissionstypes.ActorType_ACTOR_TYPE_FORECASTER:
		a.failf(height, "EventNonceExpired topic %d nonce %d carries actor type %s", event.TopicId, event.Nonce, event.ActorType)
	}
}

// auditScores checks EventScoresSet against the scores stored under its nonce.
func (a *eventAudit) auditScores(height int64, facts blockContext, event *emissionstypes.EventScoresSet) {
	a.coverage["EventScoresSet/"+event.ActorType.String()]++
	a.requireRewardNonce(height, facts, "EventScoresSet", event.TopicId, event.Nonce)
	queryCtx := atHeight(a.ctx, height)
	var stored *emissionstypes.Scores
	var err error
	switch event.ActorType {
	case emissionstypes.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED:
		response, queryErr := a.m.Client.QueryEmissions().GetWorkerInferenceScoresAtBlock(queryCtx, &emissionstypes.GetWorkerInferenceScoresAtBlockRequest{TopicId: event.TopicId, BlockHeight: event.Nonce})
		err = queryErr
		if queryErr == nil {
			stored = response.Scores
		}
	case emissionstypes.ActorType_ACTOR_TYPE_FORECASTER:
		response, queryErr := a.m.Client.QueryEmissions().GetWorkerForecastScoresAtBlock(queryCtx, &emissionstypes.GetWorkerForecastScoresAtBlockRequest{TopicId: event.TopicId, BlockHeight: event.Nonce})
		err = queryErr
		if queryErr == nil {
			stored = response.Scores
		}
	case emissionstypes.ActorType_ACTOR_TYPE_REPUTER:
		response, queryErr := a.m.Client.QueryEmissions().GetReputersScoresAtBlock(queryCtx, &emissionstypes.GetReputersScoresAtBlockRequest{TopicId: event.TopicId, BlockHeight: event.Nonce})
		err = queryErr
		if queryErr == nil {
			stored = response.Scores
		}
	}
	if err != nil {
		a.failf(height, "stored %s scores topic %d nonce %d: %v", event.ActorType, event.TopicId, event.Nonce, err)
		return
	}
	if stored == nil {
		a.failf(height, "no stored %s scores for topic %d nonce %d", event.ActorType, event.TopicId, event.Nonce)
		return
	}
	for i, address := range event.Addresses {
		found := false
		for _, score := range stored.Scores {
			if score.Address == address {
				found = true
				a.requireDecEqual(height, fmt.Sprintf("EventScoresSet %s %s nonce %d", event.ActorType, address, event.Nonce), event.Scores[i], score.Score)
				if score.BlockHeight != event.Nonce {
					a.failf(height, "EventScoresSet nonce %d, stored score has block height %d", event.Nonce, score.BlockHeight)
				}
			}
		}
		if !found {
			a.failf(height, "EventScoresSet %s %s topic %d nonce %d has no stored score", event.ActorType, address, event.TopicId, event.Nonce)
		}
	}
}

// auditLabelRegistry checks a frozen registry against the topic's arity, the
// network inferences stored under its nonce and the suite's own submissions.
func (a *eventAudit) auditLabelRegistry(height int64, facts blockContext, event *emissionstypes.EventEpochLabelRegistryFrozen) {
	key := topicNonceKey(event.TopicId, event.Nonce)
	a.frozenCount[key]++
	if a.frozenCount[key] > 1 {
		a.failf(height, "EventEpochLabelRegistryFrozen emitted %d times for topic %d nonce %d", a.frozenCount[key], event.TopicId, event.Nonce)
	}
	a.frozenLabels[key] = event.Labels
	if event.RegistrySize != uint64(len(event.Labels)) {
		a.failf(height, "EventEpochLabelRegistryFrozen topic %d registry_size %d with %d labels", event.TopicId, event.RegistrySize, len(event.Labels))
	}
	a.requireWorkerNonceOpen(height, "EventEpochLabelRegistryFrozen", event.TopicId, event.Nonce)
	topic, ok := a.getTopic(height, event.TopicId)
	if !ok {
		return
	}
	if topic.Topic.OutputArity == emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE && event.RegistrySize != 1 {
		a.failf(height, "EventEpochLabelRegistryFrozen SINGLE topic %d registry_size %d", event.TopicId, event.RegistrySize)
	}
	if event.TopicId == a.m.TopicID {
		submitted := make([]string, len(a.m.InfererValues))
		for i, value := range a.m.InfererValues {
			submitted[i] = value.Label
		}
		sortedLabels := sortedStrings(event.Labels)
		if !slices.Equal(sortedLabels, sortedStrings(submitted)) {
			a.failf(height, "EventEpochLabelRegistryFrozen topic %d labels %v, the worker submitted %v", event.TopicId, event.Labels, submitted)
		}
	}
	if facts.droppedNonces[key] {
		a.skipped["label registry vs network inferences: close failed"]++
		return
	}
	response, err := a.m.Client.QueryEmissions().GetNetworkInferencesAtBlock(atHeight(a.ctx, height), &emissionstypes.GetNetworkInferencesAtBlockRequest{TopicId: event.TopicId, BlockHeightLastInference: event.Nonce})
	if err != nil {
		a.failf(height, "GetNetworkInferencesAtBlock topic %d nonce %d: %v", event.TopicId, event.Nonce, err)
		return
	}
	stored := make([]string, 0, len(response.NetworkInferences.CombinedValue))
	for _, value := range response.NetworkInferences.CombinedValue {
		stored = append(stored, value.LabelName)
	}
	if !slices.Equal(stored, event.Labels) {
		a.failf(height, "EventEpochLabelRegistryFrozen topic %d nonce %d labels %v, stored network inferences carry %v", event.TopicId, event.Nonce, event.Labels, stored)
	}
}

func (a *eventAudit) requireLastCommit(height int64, name string, topicId uint64, eventHeight, eventNonce, storedHeight, storedNonce int64) {
	if eventHeight != height || storedHeight != height {
		a.failf(height, "%s topic %d block_height %d, stored commit height %d", name, topicId, eventHeight, storedHeight)
	}
	if eventNonce != storedNonce {
		a.failf(height, "%s topic %d nonce %d, stored commit nonce %d", name, topicId, eventNonce, storedNonce)
	}
}

func (a *eventAudit) auditListeningCoefficients(height int64, facts blockContext, event *emissionstypes.EventListeningCoefficientsSet) {
	a.requireRewardNonce(height, facts, "EventListeningCoefficientsSet", event.TopicId, event.Nonce)
	for i, reputer := range event.Addresses {
		response, err := a.m.Client.QueryEmissions().GetListeningCoefficient(atHeight(a.ctx, height), &emissionstypes.GetListeningCoefficientRequest{TopicId: event.TopicId, Reputer: reputer})
		if err != nil {
			a.failf(height, "GetListeningCoefficient topic %d: %v", event.TopicId, err)
			continue
		}
		a.requireDecEqual(height, "EventListeningCoefficientsSet coefficient of "+reputer, event.Coefficients[i], response.ListeningCoefficient.Coefficient)
	}
}

// auditRegrets checks a regret event against the regret stored for each actor.
func (a *eventAudit) auditRegrets(height int64, name string, topicId uint64, nonce int64, addresses []string, regrets []alloraMath.Dec, query func(string) (*emissionstypes.TimestampedValue, error)) {
	a.requireReputerNonceOpen(height, name, topicId, nonce)
	for i, actor := range addresses {
		stored, err := query(actor)
		if err != nil {
			a.failf(height, "%s stored regret topic %d actor %s: %v", name, topicId, actor, err)
			continue
		}
		a.requireDecEqual(height, fmt.Sprintf("%s topic %d actor %s", name, topicId, actor), regrets[i], stored.Value)
		if stored.BlockHeight != nonce {
			a.failf(height, "%s topic %d actor %s nonce %d, stored regret has block height %d", name, topicId, actor, nonce, stored.BlockHeight)
		}
	}
}

func (a *eventAudit) auditInitialEmaScore(height int64, event *emissionstypes.EventTopicInitialEmaScoreSet) {
	queryCtx := atHeight(a.ctx, height)
	var stored alloraMath.Dec
	var err error
	switch event.ActorType {
	case emissionstypes.ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED:
		response, queryErr := a.m.Client.QueryEmissions().GetTopicInitialInfererEmaScore(queryCtx, &emissionstypes.GetTopicInitialInfererEmaScoreRequest{TopicId: event.TopicId})
		err = queryErr
		if queryErr == nil {
			stored = response.Score
		}
	case emissionstypes.ActorType_ACTOR_TYPE_FORECASTER:
		response, queryErr := a.m.Client.QueryEmissions().GetTopicInitialForecasterEmaScore(queryCtx, &emissionstypes.GetTopicInitialForecasterEmaScoreRequest{TopicId: event.TopicId})
		err = queryErr
		if queryErr == nil {
			stored = response.Score
		}
	case emissionstypes.ActorType_ACTOR_TYPE_REPUTER:
		response, queryErr := a.m.Client.QueryEmissions().GetTopicInitialReputerEmaScore(queryCtx, &emissionstypes.GetTopicInitialReputerEmaScoreRequest{TopicId: event.TopicId})
		err = queryErr
		if queryErr == nil {
			stored = response.Score
		}
	}
	if err != nil {
		a.failf(height, "stored initial %s EMA score topic %d: %v", event.ActorType, event.TopicId, err)
		return
	}
	a.requireDecEqual(height, fmt.Sprintf("EventTopicInitialEmaScoreSet %s topic %d", event.ActorType, event.TopicId), event.Score, stored)
}

// sortedStrings returns a sorted copy of values.
func sortedStrings(values []string) []string {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	return sorted
}

// requireSameAddresses compares an active-set event's addresses with stored actors.
func (a *eventAudit) requireSameAddresses(height int64, name string, topicId uint64, nonce int64, event, stored []string) {
	if !slices.Equal(sortedStrings(event), sortedStrings(stored)) {
		a.failf(height, "%s topic %d nonce %d addresses %v, stored actors %v", name, topicId, nonce, event, stored)
	}
}

func (a *eventAudit) auditActiveReputers(height int64, facts blockContext, event *emissionstypes.EventActiveReputersSet) {
	if facts.droppedNonces[topicNonceKey(event.TopicId, event.Nonce)] {
		a.skipped["active reputers vs stored bundles: close failed"]++
		return
	}
	response, err := a.m.Client.QueryEmissions().GetReputerLossBundlesAtBlock(atHeight(a.ctx, height), &emissionstypes.GetReputerLossBundlesAtBlockRequest{TopicId: event.TopicId, BlockHeight: event.Nonce})
	if err != nil {
		a.failf(height, "GetReputerLossBundlesAtBlock topic %d nonce %d: %v", event.TopicId, event.Nonce, err)
		return
	}
	stored := make([]string, 0, len(response.LossBundles))
	for _, bundle := range response.LossBundles {
		stored = append(stored, bundle.Reputer)
	}
	a.requireSameAddresses(height, "EventActiveReputersSet", event.TopicId, event.Nonce, event.Addresses, stored)
}

func (a *eventAudit) auditActiveInferers(height int64, event *emissionstypes.EventActiveInferersSet) {
	response, err := a.m.Client.QueryEmissions().GetInferencesAtBlock(atHeight(a.ctx, height), &emissionstypes.GetInferencesAtBlockRequest{TopicId: event.TopicId, BlockHeight: event.Nonce})
	if err != nil {
		a.failf(height, "GetInferencesAtBlock topic %d nonce %d: %v", event.TopicId, event.Nonce, err)
		return
	}
	stored := make([]string, 0)
	for _, inference := range response.GetInferences().GetInferences() {
		stored = append(stored, inference.Inferer)
	}
	a.requireSameAddresses(height, "EventActiveInferersSet", event.TopicId, event.Nonce, event.Addresses, stored)
}

func (a *eventAudit) auditActiveForecasters(height int64, event *emissionstypes.EventActiveForecastersSet) {
	response, err := a.m.Client.QueryEmissions().GetForecastsAtBlock(atHeight(a.ctx, height), &emissionstypes.GetForecastsAtBlockRequest{TopicId: event.TopicId, BlockHeight: event.Nonce})
	if err != nil {
		a.failf(height, "GetForecastsAtBlock topic %d nonce %d: %v", event.TopicId, event.Nonce, err)
		return
	}
	stored := make([]string, 0)
	for _, forecast := range response.GetForecasts().GetForecasts() {
		stored = append(stored, forecast.Forecaster)
	}
	a.requireSameAddresses(height, "EventActiveForecastersSet", event.TopicId, event.Nonce, event.Addresses, stored)
	for _, discarded := range event.DiscardedAddresses {
		if slices.Contains(stored, discarded) {
			a.failf(height, "EventActiveForecastersSet topic %d nonce %d discards %s, whose forecast is stored", event.TopicId, event.Nonce, discarded)
		}
	}
	if len(event.DiscardedAddresses) > 0 {
		a.coverage["EventActiveForecastersSet/discarded"]++
	}

	// The close reads the topic's active forecasters, which accepted payloads add and
	// every close resets, so both lists together are the forecasters whose payloads
	// the chain accepted since the previous close. Before the first audited close of a
	// topic the audit has not seen every payload.
	if !a.closedTopics[event.TopicId] {
		a.skipped["forecaster lists vs payloads: first audited close"]++
		return
	}
	accepted := a.acceptedForecasters[event.TopicId]
	reported := append(slices.Clone(event.Addresses), event.DiscardedAddresses...)
	if !slices.Equal(sortedStrings(reported), sortedStrings(accepted)) {
		a.failf(height, "EventActiveForecastersSet topic %d nonce %d addresses %v and discarded_addresses %v, accepted forecaster payloads since the previous close %v", event.TopicId, event.Nonce, event.Addresses, event.DiscardedAddresses, accepted)
	}
}

// recordForecasterPayload notes a forecaster whose payload the chain accepted; the
// topic's next close must report it in one of its two lists.
func (a *eventAudit) recordForecasterPayload(event *emissionstypes.EventInsertForecasterPayload) {
	if !slices.Contains(a.acceptedForecasters[event.TopicId], event.Forecaster) {
		a.acceptedForecasters[event.TopicId] = append(a.acceptedForecasters[event.TopicId], event.Forecaster)
	}
}

// resetForecasterPayloads follows the close's reset of the topic's active workers,
// which runs on every close, failed or not, and ends with the window-closed event.
func (a *eventAudit) resetForecasterPayloads(topicId uint64) {
	delete(a.acceptedForecasters, topicId)
	a.closedTopics[topicId] = true
}

// auditWorkerWindowOpened checks a window opened at the topic's churn for the
// nonce the same block created.
func (a *eventAudit) auditWorkerWindowOpened(height int64, event *emissionstypes.EventWorkerSubmissionWindowOpened) {
	if event.Nonce != height {
		a.failf(height, "EventWorkerSubmissionWindowOpened topic %d nonce %d, expected the opening block", event.TopicId, event.Nonce)
	}
	if !a.isWorkerNonceUnfulfilled(height, event.TopicId, event.Nonce) {
		a.failf(height, "EventWorkerSubmissionWindowOpened topic %d nonce %d is not unfulfilled", event.TopicId, event.Nonce)
	}
	if topic, ok := a.getTopic(height, event.TopicId); ok {
		if expected := event.Nonce + topic.Topic.WorkerSubmissionWindow; event.WindowEndBlock != expected {
			a.failf(height, "EventWorkerSubmissionWindowOpened topic %d window_end_block %d, expected %d", event.TopicId, event.WindowEndBlock, expected)
		}
	}
}

// auditReputerWindowOpened checks that the nonce awaits losses and the window has
// not already ended.
func (a *eventAudit) auditReputerWindowOpened(height int64, event *emissionstypes.EventReputerSubmissionWindowOpened) {
	if !a.isReputerNonceUnfulfilled(height, event.TopicId, event.Nonce) {
		a.failf(height, "EventReputerSubmissionWindowOpened topic %d nonce %d is not unfulfilled", event.TopicId, event.Nonce)
	}
	if event.WindowEndBlock < height {
		a.failf(height, "EventReputerSubmissionWindowOpened topic %d window_end_block %d is in the past", event.TopicId, event.WindowEndBlock)
	}
}
