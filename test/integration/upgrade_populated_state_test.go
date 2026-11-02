package integration_test

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

const (
	seedEpochLength       = int64(10)
	seedWorkerWindow      = int64(5)
	seedGroundTruthLag    = int64(10)
	blockerEpochLength    = int64(20)
	blockerWorkerWindow   = int64(19)
	driftTopicEpochLength = int64(5)
	// Stake sizes keep the weights ordered ghost < drift < blocker with a wide margin,
	// so the temporary minimum weight inactivates the drift topic alone.
	ghostTopicStake   = int64(10_000)
	driftTopicStake   = int64(1_000_000)
	blockerTopicStake = int64(100_000_000)
	activeTopicStake  = int64(5_000_000)
	// A delay long enough to keep seeded stake removals pending across the upgrade.
	pendingRemovalDelay   = int64(80)
	shortRemovalDelay     = int64(10)
	reputerRemovalAmount  = int64(40_000)
	delegateRemovalAmount = int64(50_000)
	inactiveRemovalAmount = int64(100_000)
	postUpgradeStake      = int64(70_000)
	// The ceiling differs from the module default, so a backfill from the default is caught.
	defaultSeedCeiling  = uint64(17)
	snapshotRetryWindow = 2 * time.Minute
	rewardWaitBlocks    = 6 * seedEpochLength
	rewardsEventType    = "emissions.v10.EventRewardsSettled"
	staticKeyPrefix     = "static/"
	runtimeTopicPrefix  = "runtime/topic/"
	runtimeGlobalPrefix = "runtime/global/"
)

// stateSnapshot maps a named on-chain value to its rendered form at one height.
// Static keys must survive the upgrade block unchanged; runtime keys may also move
// when a topic churns or closes a worker window at that block.
type stateSnapshot map[string]string

// populatedStateChecks seeds a baseline chain through real transactions and verifies
// what the emissions v16 migration does to that state.
type populatedStateChecks struct {
	seedCeiling uint64

	regressionTopic uint64
	multiLabelTopic uint64
	neverActive     uint64
	driftTopic      uint64
	ghostTopic      uint64
	blockerTopic    uint64
	topicIds        []uint64
	seededEpochs    map[uint64]int64

	teamAcc       cosmosaccount.Account
	teamAddr      string
	investorsAcc  cosmosaccount.Account
	investorsAddr string

	stakeRemovalBlock    int64
	delegateRemovalBlock int64

	baseline stateSnapshot
	lastOld  stateSnapshot
}

var _ upgradeStateChecks = (*populatedStateChecks)(nil)

// newPopulatedStateChecks reads the seeded ceiling, which selects the backfill
// branch that this run reaches.
func newPopulatedStateChecks() *populatedStateChecks {
	ceiling := defaultSeedCeiling
	if raw, found := os.LookupEnv("UPGRADE_SEED_MAX_TOP_INFERERS"); found {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("UPGRADE_SEED_MAX_TOP_INFERERS=%q: %v", raw, err))
		}
		ceiling = parsed
	}
	return &populatedStateChecks{seedCeiling: ceiling, seededEpochs: map[uint64]int64{}} //nolint:exhaustruct // the remaining fields are populated while seeding
}

// migratedCeilingAndFloor returns what the migration specifies for a seeded ceiling:
// a zero ceiling is repaired to the module default, and the backfilled floor is the
// default floor clamped to that ceiling.
func migratedCeilingAndFloor(seedCeiling uint64) (ceiling uint64, floor uint64) {
	ceiling = seedCeiling
	if ceiling == 0 {
		ceiling = emissionstypes.DefaultParams().MaxTopInferersToReward
	}
	return ceiling, min(emissionstypes.DefaultMinTopInferersToReward, ceiling)
}

// broadcastOK submits msgs in one tx, requires success, and returns the tx height.
func broadcastOK(m testCommon.TestConfig, acc cosmosaccount.Account, msgs ...sdktypes.Msg) int64 {
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, acc, msgs...)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
	require.Zerof(m.T, txResp.Code, "tx failed: %s", txResp.RawLog)
	return txResp.Height
}

func actorFromKeyring(m testCommon.TestConfig, name string) (cosmosaccount.Account, string) {
	acc, err := m.Client.Clients[0].AccountRegistry.GetByName(name)
	require.NoError(m.T, err)
	addr, err := acc.Address(params.HumanCoinUnit)
	require.NoError(m.T, err)
	return acc, addr
}

// createSeedTopic creates a topic on the baseline. MaxTopInferersToReward stays zero
// because v0.17.2 rejects the field as unknown, and proto3 omits a zero value.
func createSeedTopic(
	m testCommon.TestConfig,
	metadata string,
	epochLength, workerWindow int64,
	topicType emissionstypes.TopicType,
	arity emissionstypes.TopicOutputArity,
) uint64 {
	ctx := context.Background()
	request := &emissionstypes.CreateNewTopicRequest{ //nolint:exhaustruct // label fields are set per topic type below
		Creator:                  m.AliceAddr,
		Metadata:                 metadata,
		LossMethod:               "mse",
		EpochLength:              epochLength,
		GroundTruthLag:           max(seedGroundTruthLag, epochLength),
		WorkerSubmissionWindow:   workerWindow,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.05"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.05"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.05"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
		TopicType:                topicType,
		OutputArity:              arity,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   emissionstypes.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:           nil,
		LabelDefaultValue:        alloraMath.ZeroDec(),
		LabelCaseSensitive:       true,
	}
	if arity == emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI {
		request.RequireUnity = true
		request.UnityTolerance = alloraMath.MustNewDecFromString("0.01")
		request.MaxLabelsPerSubmission = 8
		request.LabelWhitelist = []string{"UP", "MID", "LOW"}
	}
	txResp, err := m.Client.BroadcastTx(ctx, m.AliceAcc, request)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
	response := &emissionstypes.CreateNewTopicResponse{} //nolint:exhaustruct // the fields are populated by decode
	require.NoError(m.T, txResp.Decode(response))
	return response.TopicId
}

func addStakeMsg(sender string, topicId uint64, amount int64) *emissionstypes.AddStakeRequest {
	return &emissionstypes.AddStakeRequest{Sender: sender, TopicId: topicId, Amount: cosmosMath.NewInt(amount)}
}

//nolint:exhaustruct // a params update names only the fields it changes
func updateEmissionsParams(m testCommon.TestConfig, update emissionstypes.OptionalParams) {
	broadcastOK(m, m.AliceAcc, &emissionstypes.UpdateParamsRequest{Sender: m.AliceAddr, Params: &update})
}

func waitForHeight(m testCommon.TestConfig, height int64) {
	require.NoError(m.T, m.Client.WaitForBlockHeight(context.Background(), height))
}

// atHeight pins the emissions queries made with the returned context to one height.
func atHeight(height int64) context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, strconv.FormatInt(height, 10))
}

func topicNextChurn(m testCommon.TestConfig, ctx context.Context, topicId uint64) int64 {
	response, err := m.Client.QueryEmissions().GetNextChurningBlockByTopicId(ctx, &emissionstypes.GetNextChurningBlockByTopicIdRequest{TopicId: topicId})
	require.NoError(m.T, err)
	return response.BlockHeight
}

func topicPreviousWeight(m testCommon.TestConfig, ctx context.Context, topicId uint64) (alloraMath.Dec, bool) {
	response, err := m.Client.QueryEmissions().GetPreviousTopicWeight(ctx, &emissionstypes.GetPreviousTopicWeightRequest{TopicId: topicId})
	require.NoError(m.T, err)
	return response.Weight, response.NotFound
}

func totalSumWeights(m testCommon.TestConfig, ctx context.Context) alloraMath.Dec {
	response, err := m.Client.QueryEmissions().GetTotalSumPreviousTopicWeights(ctx, &emissionstypes.GetTotalSumPreviousTopicWeightsRequest{})
	require.NoError(m.T, err)
	return response.Weight
}

// sumOfScheduledWeights adds the stored weights of the topics with a next churning block.
func (c *populatedStateChecks) sumOfScheduledWeights(m testCommon.TestConfig, ctx context.Context) alloraMath.Dec {
	sum := alloraMath.ZeroDec()
	for _, topicId := range c.topicIds {
		if topicNextChurn(m, ctx, topicId) == 0 {
			continue
		}
		weight, notFound := topicPreviousWeight(m, ctx, topicId)
		if notFound {
			continue
		}
		var err error
		sum, err = sum.Add(weight)
		require.NoError(m.T, err)
	}
	return sum
}

// SeedBaseline drives the baseline chain into every kind of state the migration reads.
// Scenarios that need an exact activity state run first, while the per-block topic limit
// and the minimum weight can change without affecting the topics that come later.
func (c *populatedStateChecks) SeedBaseline(m testCommon.TestConfig) {
	m.T.Log("--- Seed: actors and whitelists ---")
	c.teamAcc, c.teamAddr = actorFromKeyring(m, "team")
	c.investorsAcc, c.investorsAddr = actorFromKeyring(m, "investors")
	addTopicCreator(m, m.AliceAddr)
	for _, addr := range []string{m.AliceAddr, m.BobAddr, c.teamAddr, c.investorsAddr} {
		addGlobalActor(m, addr)
	}
	updateEmissionsParams(m, emissionstypes.OptionalParams{ //nolint:exhaustruct // a params update names only the fields it changes
		MinEpochLength:          []int64{1},
		MaxActiveTopicsPerBlock: []uint64{1},
		MaxTopReputersToReward:  []uint64{24},
	})

	c.seedGhostTopic(m)
	updateEmissionsParams(m, emissionstypes.OptionalParams{MaxActiveTopicsPerBlock: []uint64{5}}) //nolint:exhaustruct // a params update names only the fields it changes
	c.seedDriftedInactiveTopic(m)
	c.seedActiveTopics(m)

	m.T.Log("--- Seed: topic that never activated ---")
	c.neverActive = createSeedTopic(m, "never active, no stake", seedEpochLength, seedWorkerWindow,
		emissionstypes.TopicType_TOPIC_TYPE_REGRESSION, emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE)
	c.topicIds = append(c.topicIds, c.neverActive)
	registerActor(m, c.investorsAcc, c.neverActive, false)

	c.seedPendingRemovals(m)

	c.baseline = c.snapshot(m, context.Background())
	c.requireSeedReachesMigration(m, c.baseline)
}

// seedGhostTopic produces a topic that v0.17.2 leaves in the active set with its weight in
// the total but without a schedule. Both topics activate in one tx, so the short-epoch topic
// asks for the block already holding the heavier topic, and the limit of one refuses it.
func (c *populatedStateChecks) seedGhostTopic(m testCommon.TestConfig) {
	m.T.Log("--- Seed: refused reactivation (ghost topic) ---")
	ctx := context.Background()
	regression, single := emissionstypes.TopicType_TOPIC_TYPE_REGRESSION, emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE
	c.ghostTopic = createSeedTopic(m, "refused reactivation", seedEpochLength, seedWorkerWindow, regression, single)
	c.blockerTopic = createSeedTopic(m, "blocker, stays active", blockerEpochLength, blockerWorkerWindow, regression, single)
	c.topicIds = append(c.topicIds, c.ghostTopic, c.blockerTopic)
	registerActor(m, m.AliceAcc, c.ghostTopic, true)
	registerActor(m, m.AliceAcc, c.blockerTopic, true)

	broadcastOK(m, m.AliceAcc,
		addStakeMsg(m.AliceAddr, c.ghostTopic, ghostTopicStake),
		addStakeMsg(m.AliceAddr, c.blockerTopic, blockerTopicStake),
	)
	ghostChurn := topicNextChurn(m, ctx, c.ghostTopic)
	blockerChurn := topicNextChurn(m, ctx, c.blockerTopic)
	require.Equal(m.T, ghostChurn+seedEpochLength, blockerChurn, "both topics must activate in the same block")

	waitForHeight(m, ghostChurn+1)
	require.Zero(m.T, topicNextChurn(m, ctx, c.ghostTopic), "the refused topic must lose its schedule")
	weight, notFound := topicPreviousWeight(m, ctx, c.ghostTopic)
	require.False(m.T, notFound)
	require.True(m.T, weight.IsPositive(), "the refused topic must keep its stored weight")
}

// seedDriftedInactiveTopic inactivates a topic that holds a stored weight, by raising the
// minimum weight above it for one churn, then completes a stake removal on it. v0.17.2
// moves the total by the weight change although the topic no longer counts in it.
func (c *populatedStateChecks) seedDriftedInactiveTopic(m testCommon.TestConfig) {
	m.T.Log("--- Seed: stake removal on an inactivated topic (weight sum drift) ---")
	ctx := context.Background()
	genesisParams := GetEmissionsParams(m)
	c.driftTopic = createSeedTopic(m, "inactive with stake and weight", driftTopicEpochLength, 2,
		emissionstypes.TopicType_TOPIC_TYPE_REGRESSION, emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE)
	c.topicIds = append(c.topicIds, c.driftTopic)
	registerActor(m, m.AliceAcc, c.driftTopic, true)
	addStake(m, m.AliceAcc, c.driftTopic, driftTopicStake)

	firstChurn := topicNextChurn(m, ctx, c.driftTopic)
	waitForHeight(m, firstChurn+1)
	driftWeight, notFound := topicPreviousWeight(m, ctx, c.driftTopic)
	require.False(m.T, notFound)
	blockerWeight, notFound := topicPreviousWeight(m, ctx, c.blockerTopic)
	require.False(m.T, notFound)
	require.True(m.T, driftWeight.Lt(blockerWeight), "drift weight %s must stay below blocker weight %s", driftWeight, blockerWeight)
	raised, err := driftWeight.Add(blockerWeight)
	require.NoError(m.T, err)
	raised, err = raised.Quo(alloraMath.NewDecFromInt64(2))
	require.NoError(m.T, err)

	updateEmissionsParams(m, emissionstypes.OptionalParams{MinTopicWeight: []alloraMath.Dec{raised}}) //nolint:exhaustruct // a params update names only the fields it changes
	churn := topicNextChurn(m, ctx, c.driftTopic)
	waitForHeight(m, churn+1)
	require.Zero(m.T, topicNextChurn(m, ctx, c.driftTopic), "the raised minimum must inactivate the topic")
	require.NotZero(m.T, topicNextChurn(m, ctx, c.blockerTopic), "the raised minimum must leave the blocker active")
	updateEmissionsParams(m, emissionstypes.OptionalParams{MinTopicWeight: []alloraMath.Dec{genesisParams.MinTopicWeight}}) //nolint:exhaustruct // a params update names only the fields it changes

	weightBefore, _ := topicPreviousWeight(m, ctx, c.driftTopic)
	sumBefore := totalSumWeights(m, ctx)
	broadcastOK(m, m.AliceAcc, &emissionstypes.RemoveStakeRequest{
		Sender:  m.AliceAddr,
		TopicId: c.driftTopic,
		Amount:  cosmosMath.NewInt(driftTopicStake / 2),
	})
	removal, err := m.Client.QueryEmissions().GetStakeRemovalInfo(ctx, &emissionstypes.GetStakeRemovalInfoRequest{TopicId: c.driftTopic, Reputer: m.AliceAddr})
	require.NoError(m.T, err)
	waitForHeight(m, removal.Removal.BlockRemovalCompleted+1)

	weightAfter, _ := topicPreviousWeight(m, ctx, c.driftTopic)
	require.False(m.T, weightAfter.Equal(weightBefore), "the completed removal must change the stored weight")
	require.Zero(m.T, topicNextChurn(m, ctx, c.driftTopic), "the topic must stay inactive")
	m.T.Logf("drift topic %d: weight %s -> %s, total %s -> %s", c.driftTopic, weightBefore, weightAfter, sumBefore, totalSumWeights(m, ctx))
}

// runEpoch submits a worker payload for the next nonce, then the matching reputer payload
// once ground truth is due, and returns the nonce.
func runEpoch(m testCommon.TestConfig, topicId uint64) int64 {
	ctx := context.Background()
	nonce := submitWorkerPayload(m, topicId)
	response, err := m.Client.QueryEmissions().GetTopic(ctx, &emissionstypes.GetTopicRequest{TopicId: topicId})
	require.NoError(m.T, err)
	waitForHeight(m, nonce+response.Topic.GroundTruthLag)
	require.NoError(m.T, InsertReputerBundle(m, response.Topic, nonce))
	waitForHeight(m, nonce+response.Topic.GroundTruthLag+response.Topic.EpochLength)
	ValidateGetNetworkLossBundle(m, topicId, nonce)
	return nonce
}

// submitWorkerPayload waits for the topic's next churn and submits for the nonce it opens.
func submitWorkerPayload(m testCommon.TestConfig, topicId uint64) int64 {
	ctx := context.Background()
	topic, err := waitForNextChurningBlock(m, topicId)
	require.NoError(m.T, err)
	nonce, err := RunWithRetry(m, 3, 2*time.Second, func() (int64, error) {
		return InsertWorkerBundle(m, topic)
	})
	require.NoError(m.T, err)
	fresh, err := m.Client.QueryEmissions().GetTopic(ctx, &emissionstypes.GetTopicRequest{TopicId: topicId})
	require.NoError(m.T, err)
	require.GreaterOrEqual(m.T, fresh.Topic.EpochLastEnded, nonce)
	return nonce
}

func withLabels(m testCommon.TestConfig, values ...*emissionstypes.InputLabeledValue) testCommon.TestConfig {
	m.InfererValues = values
	return m
}

func regressionValues() []*emissionstypes.InputLabeledValue {
	return []*emissionstypes.InputLabeledValue{{Label: "y", Value: alloraMath.MustNewBoundedExp40DecFromString("100")}}
}

func multiLabelValues() []*emissionstypes.InputLabeledValue {
	return []*emissionstypes.InputLabeledValue{
		{Label: "UP", Value: alloraMath.MustNewBoundedExp40DecFromString("0.3")},
		{Label: "MID", Value: alloraMath.MustNewBoundedExp40DecFromString("0.2")},
		{Label: "LOW", Value: alloraMath.MustNewBoundedExp40DecFromString("0.5")},
	}
}

// seedActiveTopics creates the regression and classification topics with registered actors,
// stake and delegated stake, and completes one epoch with payloads on each.
func (c *populatedStateChecks) seedActiveTopics(m testCommon.TestConfig) {
	m.T.Log("--- Seed: active topics with registrations, stake, delegation and a completed epoch ---")
	c.regressionTopic = createSeedTopic(m, "active regression", seedEpochLength, seedWorkerWindow,
		emissionstypes.TopicType_TOPIC_TYPE_REGRESSION, emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE)
	c.multiLabelTopic = createSeedTopic(m, "active classification", seedEpochLength, seedWorkerWindow,
		emissionstypes.TopicType_TOPIC_TYPE_CLASSIFICATION, emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI)
	c.topicIds = append(c.topicIds, c.regressionTopic, c.multiLabelTopic)
	for _, topicId := range []uint64{c.regressionTopic, c.multiLabelTopic} {
		registerActor(m, m.AliceAcc, topicId, true)
		registerActor(m, m.BobAcc, topicId, false)
		addStake(m, m.AliceAcc, topicId, activeTopicStake)
		addDelegateStake(m, m.BobAcc, m.AliceAddr, topicId, 200_000)
	}
	registerActor(m, c.teamAcc, c.regressionTopic, true)
	addStake(m, c.teamAcc, c.regressionTopic, 300_000)
	addDelegateStake(m, c.teamAcc, m.AliceAddr, c.regressionTopic, 300_000)
	registerActor(m, c.investorsAcc, c.regressionTopic, false)

	c.seededEpochs[c.regressionTopic] = runEpoch(withLabels(m, regressionValues()...), c.regressionTopic)
	c.seededEpochs[c.multiLabelTopic] = runEpoch(withLabels(m, multiLabelValues()...), c.multiLabelTopic)
}

// seedPendingRemovals queues reputer and delegate stake removals that complete after the
// upgrade. The reputer removal is on the blocker topic, which earns no rewards, so the
// stake change at completion is exact.
func (c *populatedStateChecks) seedPendingRemovals(m testCommon.TestConfig) {
	m.T.Log("--- Seed: stake removals pending across the upgrade ---")
	ctx := context.Background()
	updateEmissionsParams(m, emissionstypes.OptionalParams{RemoveStakeDelayWindow: []int64{pendingRemovalDelay}}) //nolint:exhaustruct // a params update names only the fields it changes
	broadcastOK(m, m.AliceAcc, &emissionstypes.RemoveStakeRequest{
		Sender:  m.AliceAddr,
		TopicId: c.blockerTopic,
		Amount:  cosmosMath.NewInt(reputerRemovalAmount),
	})
	broadcastOK(m, m.BobAcc, &emissionstypes.RemoveDelegateStakeRequest{
		Sender:  m.BobAddr,
		Reputer: m.AliceAddr,
		TopicId: c.regressionTopic,
		Amount:  cosmosMath.NewInt(delegateRemovalAmount),
	})
	stakeRemoval, err := m.Client.QueryEmissions().GetStakeRemovalInfo(ctx, &emissionstypes.GetStakeRemovalInfoRequest{TopicId: c.blockerTopic, Reputer: m.AliceAddr})
	require.NoError(m.T, err)
	c.stakeRemovalBlock = stakeRemoval.Removal.BlockRemovalCompleted
	delegateRemoval, err := m.Client.QueryEmissions().GetDelegateStakeRemovalInfo(ctx, &emissionstypes.GetDelegateStakeRemovalInfoRequest{TopicId: c.regressionTopic, Delegator: m.BobAddr, Reputer: m.AliceAddr})
	require.NoError(m.T, err)
	c.delegateRemovalBlock = delegateRemoval.Removal.BlockRemovalCompleted
}

// requireSeedReachesMigration fails unless the snapshot holds the pre-migration shape of
// every migration input that the seed is responsible for.
func (c *populatedStateChecks) requireSeedReachesMigration(m testCommon.TestConfig, snapshot stateSnapshot) {
	require.Equal(m.T, "0", snapshot[staticKeyPrefix+"params/min_top_inferers_to_reward"], "the floor must be unset on the baseline")
	for _, topicId := range c.topicIds {
		require.Equalf(m.T, "0", snapshot[fmt.Sprintf("%stopic/%d/max_top_inferers_to_reward", staticKeyPrefix, topicId)], "topic %d must decode with a zero cap", topicId)
	}
	require.Equal(m.T, "true", snapshot[fmt.Sprintf("%s%d/scheduled", runtimeTopicPrefix, c.regressionTopic)])
	require.Equal(m.T, "false", snapshot[fmt.Sprintf("%s%d/scheduled", runtimeTopicPrefix, c.ghostTopic)])
	require.Equal(m.T, "false", snapshot[fmt.Sprintf("%s%d/scheduled", runtimeTopicPrefix, c.driftTopic)])
	require.Equal(m.T, "false", snapshot[fmt.Sprintf("%s%d/scheduled", runtimeTopicPrefix, c.neverActive)])
	require.Equal(m.T, "true", snapshot[fmt.Sprintf("%s%d/previous_weight_not_found", runtimeTopicPrefix, c.neverActive)], "the never-active topic must hold no weight")
	require.Equal(m.T, "0", snapshot[fmt.Sprintf("%s%d/topic_stake", runtimeTopicPrefix, c.neverActive)], "the never-active topic must hold no stake")
	require.NotEqual(m.T, "0", snapshot[fmt.Sprintf("%s%d/topic_stake", runtimeTopicPrefix, c.driftTopic)], "the inactive drift topic must hold stake")

	total, err := alloraMath.NewDecFromString(snapshot[runtimeGlobalPrefix+"total_sum_previous_topic_weights"])
	require.NoError(m.T, err)
	scheduled, err := alloraMath.NewDecFromString(snapshot[runtimeGlobalPrefix+"sum_scheduled_weights"])
	require.NoError(m.T, err)
	require.Falsef(m.T, total.Equal(scheduled), "the seed must leave the total %s drifted from the scheduled sum %s", total, scheduled)
	m.T.Logf("Seeded drift: total %s, sum over scheduled topics %s", total, scheduled)
}

// snapshot renders every value the migration must preserve, or rewrite, at the context's height.
func (c *populatedStateChecks) snapshot(m testCommon.TestConfig, ctx context.Context) stateSnapshot {
	snapshot, err := c.tryChainSnapshot(m, ctx)
	require.NoError(m.T, err)
	return snapshot
}

// tryChainSnapshot returns the first query error, so that a caller racing the binary swap can retry.
func (c *populatedStateChecks) tryChainSnapshot(m testCommon.TestConfig, ctx context.Context) (stateSnapshot, error) {
	query := m.Client.QueryEmissions()
	out := stateSnapshot{}

	paramsResp, err := query.GetParams(ctx, &emissionstypes.GetParamsRequest{})
	if err != nil {
		return nil, err
	}
	migratedParams := paramsResp.Params
	out[staticKeyPrefix+"params/max_top_inferers_to_reward"] = strconv.FormatUint(migratedParams.MaxTopInferersToReward, 10)
	out[staticKeyPrefix+"params/min_top_inferers_to_reward"] = strconv.FormatUint(migratedParams.MinTopInferersToReward, 10)
	migratedParams.MaxTopInferersToReward, migratedParams.MinTopInferersToReward = 0, 0
	out[staticKeyPrefix+"params/rest"] = migratedParams.String()

	nextTopic, err := query.GetNextTopicId(ctx, &emissionstypes.GetNextTopicIdRequest{})
	if err != nil {
		return nil, err
	}
	out[staticKeyPrefix+"next_topic_id"] = strconv.FormatUint(nextTopic.NextTopicId, 10)

	totalStake, err := query.GetTotalStake(ctx, &emissionstypes.GetTotalStakeRequest{})
	if err != nil {
		return nil, err
	}
	out[runtimeGlobalPrefix+"total_stake"] = totalStake.Amount.String()
	stakingBalance, err := m.Client.QueryBank().Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: authtypes.NewModuleAddress(emissionstypes.AlloraStakingAccountName).String(),
		Denom:   params.DefaultBondDenom,
	})
	if err != nil {
		return nil, err
	}
	out[runtimeGlobalPrefix+"staking_bank_balance"] = stakingBalance.Balance.Amount.String()
	sumResp, err := query.GetTotalSumPreviousTopicWeights(ctx, &emissionstypes.GetTotalSumPreviousTopicWeightsRequest{})
	if err != nil {
		return nil, err
	}
	out[runtimeGlobalPrefix+"total_sum_previous_topic_weights"] = sumResp.Weight.String()

	scheduledSum := alloraMath.ZeroDec()
	for _, topicId := range c.topicIds {
		weight, scheduled, err := c.snapshotTopic(m, ctx, topicId, out)
		if err != nil {
			return nil, err
		}
		if scheduled {
			scheduledSum, err = scheduledSum.Add(weight)
			if err != nil {
				return nil, err
			}
		}
	}
	out[runtimeGlobalPrefix+"sum_scheduled_weights"] = scheduledSum.String()
	return out, nil
}

// snapshotTopic records one topic and returns its stored weight and whether it is scheduled.
func (c *populatedStateChecks) snapshotTopic(m testCommon.TestConfig, ctx context.Context, topicId uint64, out stateSnapshot) (alloraMath.Dec, bool, error) {
	query := m.Client.QueryEmissions()
	static := fmt.Sprintf("%stopic/%d/", staticKeyPrefix, topicId)
	runtime := fmt.Sprintf("%s%d/", runtimeTopicPrefix, topicId)
	none := alloraMath.ZeroDec()

	topicResp, err := query.GetTopic(ctx, &emissionstypes.GetTopicRequest{TopicId: topicId})
	if err != nil {
		return none, false, err
	}
	topic := *topicResp.Topic
	out[static+"max_top_inferers_to_reward"] = strconv.FormatUint(topic.MaxTopInferersToReward, 10)
	out[runtime+"epoch_last_ended"] = strconv.FormatInt(topic.EpochLastEnded, 10)
	out[runtime+"effective_revenue"] = topicResp.EffectiveRevenue
	out[runtime+"topic_stake"] = topicResp.TopicStake
	topic.MaxTopInferersToReward, topic.EpochLastEnded = 0, 0
	out[static+"config"] = topic.String()

	churn, err := query.GetNextChurningBlockByTopicId(ctx, &emissionstypes.GetNextChurningBlockByTopicIdRequest{TopicId: topicId})
	if err != nil {
		return none, false, err
	}
	scheduled := churn.BlockHeight != 0
	out[runtime+"scheduled"] = strconv.FormatBool(scheduled)
	out[runtime+"next_churning_block"] = strconv.FormatInt(churn.BlockHeight, 10)

	weightResp, err := query.GetPreviousTopicWeight(ctx, &emissionstypes.GetPreviousTopicWeightRequest{TopicId: topicId})
	if err != nil {
		return none, false, err
	}
	out[runtime+"previous_weight"] = weightResp.Weight.String()
	out[runtime+"previous_weight_not_found"] = strconv.FormatBool(weightResp.NotFound)

	revenue, err := query.GetTopicFeeRevenue(ctx, &emissionstypes.GetTopicFeeRevenueRequest{TopicId: topicId})
	if err != nil {
		return none, false, err
	}
	out[runtime+"fee_revenue"] = revenue.FeeRevenue.String()
	rewardNonce, err := query.GetTopicRewardNonce(ctx, &emissionstypes.GetTopicRewardNonceRequest{TopicId: topicId})
	if err != nil {
		return none, false, err
	}
	out[runtime+"reward_nonce"] = strconv.FormatInt(rewardNonce.Nonce, 10)
	workerNonces, err := query.GetUnfulfilledWorkerNonces(ctx, &emissionstypes.GetUnfulfilledWorkerNoncesRequest{TopicId: topicId})
	if err != nil {
		return none, false, err
	}
	out[runtime+"unfulfilled_worker_nonces"] = workerNonces.Nonces.String()
	reputerNonces, err := query.GetUnfulfilledReputerNonces(ctx, &emissionstypes.GetUnfulfilledReputerNoncesRequest{TopicId: topicId})
	if err != nil {
		return none, false, err
	}
	out[runtime+"unfulfilled_reputer_nonces"] = reputerNonces.Nonces.String()

	workerGate, err := query.IsTopicWorkerWhitelistEnabled(ctx, &emissionstypes.IsTopicWorkerWhitelistEnabledRequest{TopicId: topicId})
	if err != nil {
		return none, false, err
	}
	out[static+"worker_whitelist_enabled"] = strconv.FormatBool(workerGate.IsTopicWorkerWhitelistEnabled)
	reputerGate, err := query.IsTopicReputerWhitelistEnabled(ctx, &emissionstypes.IsTopicReputerWhitelistEnabledRequest{TopicId: topicId})
	if err != nil {
		return none, false, err
	}
	out[static+"reputer_whitelist_enabled"] = strconv.FormatBool(reputerGate.IsTopicReputerWhitelistEnabled)

	for name, addr := range map[string]string{"alice": m.AliceAddr, "bob": m.BobAddr, "team": c.teamAddr, "investors": c.investorsAddr} {
		worker, err := query.IsWorkerRegisteredInTopicId(ctx, &emissionstypes.IsWorkerRegisteredInTopicIdRequest{TopicId: topicId, Address: addr})
		if err != nil {
			return none, false, err
		}
		out[static+"worker_registered/"+name] = strconv.FormatBool(worker.IsRegistered)
		reputer, err := query.IsReputerRegisteredInTopicId(ctx, &emissionstypes.IsReputerRegisteredInTopicIdRequest{TopicId: topicId, Address: addr})
		if err != nil {
			return none, false, err
		}
		out[static+"reputer_registered/"+name] = strconv.FormatBool(reputer.IsRegistered)
		stake, err := query.GetStakeFromReputerInTopicInSelf(ctx, &emissionstypes.GetStakeFromReputerInTopicInSelfRequest{TopicId: topicId, ReputerAddress: addr})
		if err != nil {
			return none, false, err
		}
		out[runtime+"self_stake/"+name] = stake.Amount.String()
		delegated, err := query.GetStakeFromDelegatorInTopicInReputer(ctx, &emissionstypes.GetStakeFromDelegatorInTopicInReputerRequest{TopicId: topicId, DelegatorAddress: addr, ReputerAddress: m.AliceAddr})
		if err != nil {
			return none, false, err
		}
		out[static+"delegated_on_alice/"+name] = delegated.Amount.String()
	}

	if err := recordOptional(out, static+"stake_removal/alice", func() (fmt.Stringer, error) {
		response, err := query.GetStakeRemovalInfo(ctx, &emissionstypes.GetStakeRemovalInfoRequest{TopicId: topicId, Reputer: m.AliceAddr})
		if err != nil {
			return nil, err
		}
		return response.Removal, nil
	}); err != nil {
		return none, false, err
	}
	if err := recordOptional(out, static+"delegate_removal/bob", func() (fmt.Stringer, error) {
		response, err := query.GetDelegateStakeRemovalInfo(ctx, &emissionstypes.GetDelegateStakeRemovalInfoRequest{TopicId: topicId, Delegator: m.BobAddr, Reputer: m.AliceAddr})
		if err != nil {
			return nil, err
		}
		return response.Removal, nil
	}); err != nil {
		return none, false, err
	}

	if nonce, seeded := c.seededEpochs[topicId]; seeded {
		inferences, err := query.GetInferencesAtBlock(ctx, &emissionstypes.GetInferencesAtBlockRequest{TopicId: topicId, BlockHeight: nonce})
		if err != nil {
			return none, false, err
		}
		out[static+"seeded_epoch/inferences"] = inferences.Inferences.String()
		lossBundle, err := query.GetNetworkLossBundleAtBlock(ctx, &emissionstypes.GetNetworkLossBundleAtBlockRequest{TopicId: topicId, BlockHeight: nonce})
		if err != nil {
			return none, false, err
		}
		out[static+"seeded_epoch/network_loss"] = lossBundle.LossBundle.String()
	}
	return weightResp.Weight, scheduled && !weightResp.NotFound, nil
}

// recordOptional stores a value that may legitimately be absent; only a not-found answer
// is recorded as absent, and any other error is returned.
func recordOptional(out stateSnapshot, key string, read func() (fmt.Stringer, error)) error {
	value, err := read()
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			out[key] = "absent"
			return nil
		}
		return err
	}
	out[key] = value.String()
	return nil
}

// ChoosePlanHeight aligns the plan with a churn of the regression topic, so the boundary
// between a schedule at the upgrade block and one in the past is exercised.
func (c *populatedStateChecks) ChoosePlanHeight(m testCommon.TestConfig, earliest int64) int64 {
	ctx := context.Background()
	churn := topicNextChurn(m, ctx, c.regressionTopic)
	require.NotZero(m.T, churn)
	plan := churn
	if earliest > churn {
		plan = churn + (earliest-churn+seedEpochLength-1)/seedEpochLength*seedEpochLength
	}
	// A blocker churn at the plan height would close its worker window one block earlier,
	// which removes the unfulfilled worker nonce the last baseline height must hold.
	if (plan-topicNextChurn(m, ctx, c.blockerTopic))%blockerEpochLength == 0 {
		plan += seedEpochLength
	}
	return plan
}

// BeforeUpgradeBlock leaves an unfulfilled reputer nonce open across the upgrade, then
// snapshots the last height the baseline binary commits. A query may reach the upgraded
// binary if the baseline has already halted; both read the same pre-migration version.
func (c *populatedStateChecks) BeforeUpgradeBlock(m testCommon.TestConfig, planHeight int64) {
	m.T.Logf("--- Worker payload for nonce %d, reputer nonce stays open across the upgrade ---", planHeight-seedEpochLength)
	waitForHeight(m, planHeight-seedEpochLength)
	regression := withLabels(m, regressionValues()...)
	response, err := m.Client.QueryEmissions().GetTopic(context.Background(), &emissionstypes.GetTopicRequest{TopicId: c.regressionTopic})
	require.NoError(m.T, err)
	require.Equal(m.T, planHeight-seedEpochLength, response.Topic.EpochLastEnded)
	require.NoError(m.T, InsertSingleWorkerPayload(regression, response.Topic, planHeight-seedEpochLength))

	// The ceiling changes only after this worker window closes: a zero ceiling admits no
	// inferer on the baseline, so the payload above would open no reputer nonce.
	waitForHeight(m, planHeight-seedEpochLength+seedWorkerWindow+1)
	m.T.Logf("--- Seed: global max_top_inferers_to_reward = %d ---", c.seedCeiling)
	updateEmissionsParams(m, emissionstypes.OptionalParams{MaxTopInferersToReward: []uint64{c.seedCeiling}}) //nolint:exhaustruct // a params update names only the fields it changes

	waitForHeight(m, planHeight-1)
	deadline := time.Now().Add(snapshotRetryWindow)
	for {
		snapshot, err := c.tryChainSnapshot(m, atHeight(planHeight-1))
		if err == nil {
			c.lastOld = snapshot
			break
		}
		require.Truef(m.T, time.Now().Before(deadline), "snapshot at height %d: %v", planHeight-1, err)
		time.Sleep(upgradePollInterval)
	}
	c.requireSeedReachesMigration(m, c.lastOld)
	require.Equal(m.T, strconv.FormatUint(c.seedCeiling, 10), c.lastOld[staticKeyPrefix+"params/max_top_inferers_to_reward"])
	reputerNoncesKey := fmt.Sprintf("%s%d/unfulfilled_reputer_nonces", runtimeTopicPrefix, c.regressionTopic)
	require.Containsf(m.T, c.lastOld[reputerNoncesKey], fmt.Sprintf("block_height:%d", planHeight-seedEpochLength), "the regression topic must hold an unfulfilled reputer nonce")
	workerNoncesKey := fmt.Sprintf("%s%d/unfulfilled_worker_nonces", runtimeTopicPrefix, c.blockerTopic)
	require.Contains(m.T, c.lastOld[workerNoncesKey], "block_height:", "the blocker topic must hold an unfulfilled worker nonce")
	m.T.Logf("Pre-upgrade snapshot holds %d values at height %d", len(c.lastOld), planHeight-1)
}

// topicsMovingAt returns the topics whose runtime values the upgrade block's end blocker
// legitimately changes: those churning at the block or closing a worker window at it.
func (c *populatedStateChecks) topicsMovingAt(snapshot stateSnapshot, height int64) []uint64 {
	moving := make([]uint64, 0)
	for _, topicId := range c.topicIds {
		runtime := fmt.Sprintf("%s%d/", runtimeTopicPrefix, topicId)
		lastEnded, _ := strconv.ParseInt(snapshot[runtime+"epoch_last_ended"], 10, 64)
		if snapshot[runtime+"next_churning_block"] == strconv.FormatInt(height, 10) || (lastEnded != 0 && lastEnded+c.workerWindowOf(topicId) == height) {
			moving = append(moving, topicId)
		}
	}
	return moving
}

func (c *populatedStateChecks) workerWindowOf(topicId uint64) int64 {
	switch topicId {
	case c.blockerTopic:
		return blockerWorkerWindow
	case c.driftTopic:
		return 2
	default:
		return seedWorkerWindow
	}
}

// VerifyMigration compares the committed state on both sides of the upgrade block.
// Every value is exact, apart from runtime values of the topics that block moves,
// which are checked against what the end blocker specifies.
func (c *populatedStateChecks) VerifyMigration(m testCommon.TestConfig, planHeight int64) {
	post, err := c.tryChainSnapshot(m, atHeight(planHeight))
	require.NoError(m.T, err)
	pre := c.lastOld
	ceiling, floor := migratedCeilingAndFloor(c.seedCeiling)

	require.Equal(m.T, strconv.FormatUint(ceiling, 10), post[staticKeyPrefix+"params/max_top_inferers_to_reward"], "ceiling after the migration")
	require.Equal(m.T, strconv.FormatUint(floor, 10), post[staticKeyPrefix+"params/min_top_inferers_to_reward"], "floor after the migration")
	require.Len(m.T, post, len(pre), "both snapshots must hold the same values")
	for _, topicId := range c.topicIds {
		require.Equalf(m.T, strconv.FormatUint(ceiling, 10), post[fmt.Sprintf("%stopic/%d/max_top_inferers_to_reward", staticKeyPrefix, topicId)], "topic %d backfilled cap", topicId)
	}

	moving := c.topicsMovingAt(pre, planHeight)
	require.Contains(m.T, moving, c.regressionTopic, "the plan height must be a churn of the regression topic")
	m.T.Logf("Topics moved by the end blocker at height %d: %v", planHeight, moving)
	migrated := map[string]bool{staticKeyPrefix + "params/max_top_inferers_to_reward": true, staticKeyPrefix + "params/min_top_inferers_to_reward": true}
	for _, topicId := range c.topicIds {
		migrated[fmt.Sprintf("%stopic/%d/max_top_inferers_to_reward", staticKeyPrefix, topicId)] = true
	}
	keys := make([]string, 0, len(pre))
	for key := range pre {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	compared := 0
	for _, key := range keys {
		if migrated[key] || c.movesAtUpgradeBlock(key, moving) {
			continue
		}
		require.Equalf(m.T, pre[key], post[key], "%s changed across the upgrade block", key)
		compared++
	}
	m.T.Logf("%d values unchanged across the upgrade block, %d migrated or moved by the end blocker", compared, len(keys)-compared)

	ctx := atHeight(planHeight)
	for _, topicId := range c.topicIds {
		scheduled := post[fmt.Sprintf("%s%d/scheduled", runtimeTopicPrefix, topicId)]
		require.Equalf(m.T, pre[fmt.Sprintf("%s%d/scheduled", runtimeTopicPrefix, topicId)], scheduled, "topic %d activity across the upgrade block", topicId)
	}
	require.Equal(m.T, post[runtimeGlobalPrefix+"sum_scheduled_weights"], post[runtimeGlobalPrefix+"total_sum_previous_topic_weights"], "total after the migration must equal the sum over scheduled topics")
	require.NotEqual(m.T, pre[runtimeGlobalPrefix+"total_sum_previous_topic_weights"], post[runtimeGlobalPrefix+"total_sum_previous_topic_weights"], "the migration must rewrite the drifted total")
	require.Equal(m.T, post[runtimeGlobalPrefix+"total_stake"], post[runtimeGlobalPrefix+"staking_bank_balance"], "total stake must equal the staking account balance")

	// A schedule at the upgrade block is valid: the topic must run that block's epoch.
	regressionRuntime := fmt.Sprintf("%s%d/", runtimeTopicPrefix, c.regressionTopic)
	require.Equal(m.T, strconv.FormatInt(planHeight, 10), post[regressionRuntime+"epoch_last_ended"], "the topic scheduled at the upgrade block must churn there")
	require.Equal(m.T, strconv.FormatInt(planHeight+seedEpochLength, 10), post[regressionRuntime+"next_churning_block"], "the topic scheduled at the upgrade block must be rescheduled")

	// Reconciled topics keep their stored weight, ready to count once on reactivation.
	for _, topicId := range []uint64{c.ghostTopic, c.driftTopic} {
		runtime := fmt.Sprintf("%s%d/", runtimeTopicPrefix, topicId)
		require.Equalf(m.T, pre[runtime+"previous_weight"], post[runtime+"previous_weight"], "topic %d stored weight", topicId)
		require.Equalf(m.T, "0", post[runtime+"next_churning_block"], "topic %d must stay inactive", topicId)
	}
	require.Equal(m.T, c.sumOfScheduledWeights(m, ctx).String(), post[runtimeGlobalPrefix+"total_sum_previous_topic_weights"])
	m.T.Logf("Reconciled total: %s -> %s, ceiling %d, floor %d", pre[runtimeGlobalPrefix+"total_sum_previous_topic_weights"], post[runtimeGlobalPrefix+"total_sum_previous_topic_weights"], ceiling, floor)
}

func (c *populatedStateChecks) movesAtUpgradeBlock(key string, moving []uint64) bool {
	if strings.HasPrefix(key, runtimeGlobalPrefix) {
		return true
	}
	for _, topicId := range moving {
		if strings.HasPrefix(key, fmt.Sprintf("%s%d/", runtimeTopicPrefix, topicId)) {
			return true
		}
	}
	return false
}

// DriveMigratedState runs real activity on the migrated state, then requires every
// validator to agree on the app hash of every block since the upgrade.
func (c *populatedStateChecks) DriveMigratedState(m testCommon.TestConfig, nodes []validatorNode, planHeight int64) {
	ctx := context.Background()
	regression := withLabels(m, regressionValues()...)

	m.T.Log("--- Post-upgrade: reputer payload for the nonce opened before the upgrade ---")
	response, err := m.Client.QueryEmissions().GetTopic(ctx, &emissionstypes.GetTopicRequest{TopicId: c.regressionTopic})
	require.NoError(m.T, err)
	require.NoError(m.T, InsertReputerBundle(regression, response.Topic, planHeight-seedEpochLength))
	waitForHeight(m, planHeight+seedEpochLength)
	ValidateGetNetworkLossBundle(m, c.regressionTopic, planHeight-seedEpochLength)

	m.T.Log("--- Post-upgrade: full epochs on pre-existing topics ---")
	stakeBefore := c.reputerStake(m, ctx, c.regressionTopic)
	nonce := runEpoch(regression, c.regressionTopic)
	m.T.Logf("Regression topic completed epoch %d", nonce)
	multiNonce := runEpoch(withLabels(m, multiLabelValues()...), c.multiLabelTopic)
	m.T.Logf("Classification topic completed epoch %d", multiNonce)

	m.T.Log("--- Post-upgrade: rewards payout ---")
	m.T.Log("Rewards for the epoch that crossed the upgrade")
	c.requireRewardsSettled(m, nodes, planHeight, c.regressionTopic)
	m.T.Log("Rewards for the epoch that ran entirely on the upgraded binary")
	c.requireRewardsSettled(m, nodes, nonce+1, c.regressionTopic)
	stakeAfter := c.reputerStake(m, ctx, c.regressionTopic)
	require.Truef(m.T, stakeAfter.GT(stakeBefore), "reputer rewards must compound into stake: %s -> %s", stakeBefore, stakeAfter)

	m.T.Log("--- Post-upgrade: stake changes ---")
	c.requireAddStakeExact(m)
	c.requirePendingRemovalsComplete(m)
	c.requireInactiveTopicStakeChangeKeepsTotal(m)
	c.requireGhostReactivationCountsWeightOnce(m)

	requireMatchingAppHashes(m, nodes, planHeight-1)
}

func (c *populatedStateChecks) reputerStake(m testCommon.TestConfig, ctx context.Context, topicId uint64) cosmosMath.Int {
	response, err := m.Client.QueryEmissions().GetStakeFromReputerInTopicInSelf(ctx, &emissionstypes.GetStakeFromReputerInTopicInSelfRequest{TopicId: topicId, ReputerAddress: m.AliceAddr})
	require.NoError(m.T, err)
	return response.Amount
}

// requireRewardsSettled scans finalized block events from the upgrade block on for a
// positive reward to both the worker and the reputer of the topic.
func (c *populatedStateChecks) requireRewardsSettled(m testCommon.TestConfig, nodes []validatorNode, fromHeight int64, topicId uint64) {
	ctx := context.Background()
	deadline, err := m.Client.BlockHeight(ctx)
	require.NoError(m.T, err)
	deadline += rewardWaitBlocks
	rewarded := map[string]bool{}
	for height := fromHeight; height <= deadline; height++ {
		waitForHeight(m, height)
		results, err := nodes[0].client.BlockResults(ctx, &height)
		require.NoError(m.T, err)
		for _, event := range results.FinalizeBlockEvents {
			if event.Type != rewardsEventType {
				continue
			}
			attributes := map[string]string{}
			for _, attribute := range event.Attributes {
				attributes[attribute.Key] = attribute.Value
			}
			if strings.Trim(attributes["topic_id"], `"`) != strconv.FormatUint(topicId, 10) {
				continue
			}
			m.T.Logf("Height %d %s: addresses=%s rewards=%s", height, attributes["actor_type"], attributes["addresses"], attributes["rewards"])
			rewarded[attributes["actor_type"]+"/"+rewardedAddresses(attributes)] = true
		}
		if rewardedActor(rewarded, m.BobAddr) && rewardedActor(rewarded, m.AliceAddr) {
			return
		}
	}
	require.Failf(m.T, "rewards not settled", "topic %d by height %d: %v", topicId, deadline, rewarded)
}

// rewardedAddresses keeps only the addresses of an event whose reward is positive.
func rewardedAddresses(attributes map[string]string) string {
	addresses := strings.Split(strings.Trim(attributes["addresses"], "[]"), ",")
	rewards := strings.Split(strings.Trim(attributes["rewards"], "[]"), ",")
	positive := make([]string, 0, len(addresses))
	for i, address := range addresses {
		if i >= len(rewards) {
			break
		}
		value, err := alloraMath.NewDecFromString(strings.Trim(rewards[i], `"`))
		if err == nil && value.IsPositive() {
			positive = append(positive, strings.Trim(address, `"`))
		}
	}
	return strings.Join(positive, ",")
}

func rewardedActor(rewarded map[string]bool, address string) bool {
	for key := range rewarded {
		if strings.Contains(key, address) {
			return true
		}
	}
	return false
}

// requireAddStakeExact adds stake on the topic with no reputer rewards, where the amount is exact.
func (c *populatedStateChecks) requireAddStakeExact(m testCommon.TestConfig) {
	height := broadcastOK(m, m.AliceAcc, addStakeMsg(m.AliceAddr, c.blockerTopic, postUpgradeStake))
	before := c.reputerStake(m, atHeight(height-1), c.blockerTopic)
	after := c.reputerStake(m, atHeight(height), c.blockerTopic)
	require.Equal(m.T, cosmosMath.NewInt(postUpgradeStake).String(), after.Sub(before).String())
}

// requirePendingRemovalsComplete checks that removals queued on the baseline pay out the
// exact amount at their recorded blocks under the upgraded binary.
func (c *populatedStateChecks) requirePendingRemovalsComplete(m testCommon.TestConfig) {
	query := m.Client.QueryEmissions()
	waitForHeight(m, c.stakeRemovalBlock)
	before := c.reputerStake(m, atHeight(c.stakeRemovalBlock-1), c.blockerTopic)
	after := c.reputerStake(m, atHeight(c.stakeRemovalBlock), c.blockerTopic)
	require.Equal(m.T, cosmosMath.NewInt(reputerRemovalAmount).String(), before.Sub(after).String(), "reputer removal amount")

	waitForHeight(m, c.delegateRemovalBlock)
	delegated := func(height int64) cosmosMath.Int {
		response, err := query.GetStakeFromDelegatorInTopicInReputer(atHeight(height), &emissionstypes.GetStakeFromDelegatorInTopicInReputerRequest{
			TopicId: c.regressionTopic, DelegatorAddress: m.BobAddr, ReputerAddress: m.AliceAddr,
		})
		require.NoError(m.T, err)
		return response.Amount
	}
	require.Equal(m.T, cosmosMath.NewInt(delegateRemovalAmount).String(),
		delegated(c.delegateRemovalBlock-1).Sub(delegated(c.delegateRemovalBlock)).String(), "delegate removal amount")
	requireScheduledSumAt(m, c, max(c.stakeRemovalBlock, c.delegateRemovalBlock))
}

// requireInactiveTopicStakeChangeKeepsTotal repeats on the migrated state the stake removal
// that drifted the total on the baseline: the total must stay the sum over scheduled topics.
func (c *populatedStateChecks) requireInactiveTopicStakeChangeKeepsTotal(m testCommon.TestConfig) {
	ctx := context.Background()
	updateEmissionsParams(m, emissionstypes.OptionalParams{RemoveStakeDelayWindow: []int64{shortRemovalDelay}}) //nolint:exhaustruct // a params update names only the fields it changes
	broadcastOK(m, m.AliceAcc, &emissionstypes.RemoveStakeRequest{
		Sender:  m.AliceAddr,
		TopicId: c.driftTopic,
		Amount:  cosmosMath.NewInt(inactiveRemovalAmount),
	})
	removal, err := m.Client.QueryEmissions().GetStakeRemovalInfo(ctx, &emissionstypes.GetStakeRemovalInfoRequest{TopicId: c.driftTopic, Reputer: m.AliceAddr})
	require.NoError(m.T, err)
	completed := removal.Removal.BlockRemovalCompleted
	waitForHeight(m, completed)
	before, _ := topicPreviousWeight(m, atHeight(completed-1), c.driftTopic)
	after, _ := topicPreviousWeight(m, atHeight(completed), c.driftTopic)
	require.False(m.T, before.Equal(after), "the completed removal must change the stored weight")
	require.Zero(m.T, topicNextChurn(m, atHeight(completed), c.driftTopic), "the topic must stay inactive")
	requireScheduledSumAt(m, c, completed)
}

// requireGhostReactivationCountsWeightOnce reactivates the reconciled topic and requires
// the total to include its stored weight exactly once.
func (c *populatedStateChecks) requireGhostReactivationCountsWeightOnce(m testCommon.TestConfig) {
	weight, _ := topicPreviousWeight(m, context.Background(), c.ghostTopic)
	height := broadcastOK(m, m.BobAcc, &emissionstypes.FundTopicRequest{Sender: m.BobAddr, TopicId: c.ghostTopic, Amount: cosmosMath.NewInt(10_000)})
	require.Equal(m.T, height+seedEpochLength, topicNextChurn(m, atHeight(height), c.ghostTopic), "the funded topic must be rescheduled")
	requireScheduledSumAt(m, c, height)
	m.T.Logf("Topic %d reactivated at %d with stored weight %s", c.ghostTopic, height, weight)
}

func requireScheduledSumAt(m testCommon.TestConfig, c *populatedStateChecks, height int64) {
	ctx := atHeight(height)
	require.Equalf(m.T, c.sumOfScheduledWeights(m, ctx).String(), totalSumWeights(m, ctx).String(), "total at height %d", height)
}

// requireMatchingAppHashes compares the app hash each validator reports for every
// block from fromHeight up to the latest block all of them have committed.
func requireMatchingAppHashes(m testCommon.TestConfig, nodes []validatorNode, fromHeight int64) {
	ctx := context.Background()
	lastCommon := int64(0)
	for i, node := range nodes {
		_, height, err := node.abciInfo()
		require.NoErrorf(m.T, err, "validator %s", node.url)
		if i == 0 || height < lastCommon {
			lastCommon = height
		}
	}
	require.Greater(m.T, lastCommon, fromHeight)
	for height := fromHeight; height <= lastCommon; height++ {
		hashes := make([]string, 0, len(nodes))
		for _, node := range nodes {
			block, err := node.client.Block(ctx, &height)
			require.NoErrorf(m.T, err, "validator %s block %d", node.url, height)
			hashes = append(hashes, block.Block.AppHash.String())
		}
		require.Lenf(m.T, slices.Compact(hashes), 1, "app hash mismatch at height %d: %v", height, hashes)
	}
	m.T.Logf("All %d validators agree on the app hash of heights %d..%d", len(nodes), fromHeight, lastCommon)
}
