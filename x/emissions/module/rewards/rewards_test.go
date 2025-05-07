package rewards_test

import (
	"encoding/json"
	"fmt"
	"slices"

	cosmosMath "cosmossdk.io/math"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *RewardsTestSuite) TestStandardRewardEmission() {
	block := int64(600)
	s.ctx = s.ctx.WithBlockHeight(block)

	workerIndexes := returnIndexes(5, 5)
	reputerIndexes := returnIndexes(0, 5)
	s.FullTopicPass(block, workerIndexes, reputerIndexes)
	block = 600 + 10800
	s.ctx = s.ctx.WithBlockHeight(block)

	// Trigger end block - rewards distribution
	s.EndBlock()
}

func (s *RewardsTestSuite) TestStandardRewardEmissionShouldRewardTopicsWithFulfilledNonces() {
	s.SetParamsForTest()
	block := int64(600)
	s.ctx = s.ctx.WithBlockHeight(block)

	workerIndexes := returnIndexes(5, 5)
	reputerIndexes := returnIndexes(0, 5)

	// TOPIC 1 - PRE PASS
	topicId, newBlockheight := s.FullTopicPass(block, workerIndexes, reputerIndexes)

	// TOPIC 1 - FIRST PASS
	_, newBlockheight = s.FullTopicPass(newBlockheight, workerIndexes, reputerIndexes, WithTopicID(topicId))

	workerIndexes2 := returnIndexes(15, 5)
	reputerIndexes2 := returnIndexes(10, 5)

	// TOPIC 2 - PRE PASS
	_, newBlockheight = s.FullTopicPass(block, workerIndexes2, reputerIndexes2)

	// TOPIC 2 - FIRST PASS
	topic2, _ := s.FullTopicSetup(newBlockheight, workerIndexes2, reputerIndexes2, WithTopicID(topicId))

	// Do not send bundles for topic 2 yet

	beforeRewardsTopic1FeeRevenue, err := s.emissionsKeeper.GetTopicFeeRevenue(s.ctx, topicId)
	s.Require().NoError(err)
	beforeRewardsTopic2FeeRevenue, err := s.emissionsKeeper.GetTopicFeeRevenue(s.ctx, topic2.GetId())
	s.Require().NoError(err)

	// mint some rewards to give out
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	newBlockheight += topic2.GroundTruthLag
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)

	// Trigger end block - rewards distribution
	s.EndBlock()

	afterRewardsTopic1FeeRevenue, err := s.emissionsKeeper.GetTopicFeeRevenue(s.ctx, topicId)
	s.Require().NoError(err)
	afterRewardsTopic2FeeRevenue, err := s.emissionsKeeper.GetTopicFeeRevenue(s.ctx, topic2.GetId())
	s.Require().NoError(err)

	// Topic 1 should have less revenue after rewards distribution -> rewards distributed
	s.Require().True(
		beforeRewardsTopic1FeeRevenue.GT(afterRewardsTopic1FeeRevenue),
		"Topic 1 should lose influence of their fee revenue: %s > %s",
		beforeRewardsTopic1FeeRevenue.String(),
		afterRewardsTopic1FeeRevenue.String(),
	)
	// Topic 2 should also have less revenue after rewards distribution as topic rewards
	// are shared among all topics whose epoch lengths modulo the current block height are 0
	s.Require().True(
		beforeRewardsTopic2FeeRevenue.GT(afterRewardsTopic2FeeRevenue),
		"Topic 2 should lose influence of their fee revenue: %s > %s",
		beforeRewardsTopic2FeeRevenue.String(),
		afterRewardsTopic2FeeRevenue.String(),
	)
}

// Moves to end of this epoch
// Sets current block emission
// Ends block
// -------
// Moves +1 block,
// Inserts worker bundles from workers
// Moves to end of epoch
// Closes reputer nonce
// Sets rewards distribution
// Ends Block
// -------
// Insert reputer bundles from reputers
// Moves to end of ground truth lag
// Closes reputer nonce
// Returns rewards distribution (GenerateRewardsDistributionByTopicParticipant)
func (s *RewardsTestSuite) getRewardsDistribution(
	topicId uint64,
	workerValues []TestWorkerValue,
	reputerValues []TestWorkerValue,
	workerZeroAddress sdk.AccAddress,
	workerZeroOneOutInfererValue string,
	workerZeroInfererValue string,
) []types.TaskReward {
	require := s.Require()

	params, err := s.emissionsKeeper.GetParams(s.ctx)
	require.NoError(err)

	// Move to end of this epoch block
	nextBlock, _, err := s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, topicId)
	s.T().Logf("Moving nonce for TopicId: %d, Next block: %v", topicId, nextBlock)
	s.Require().NoError(err)
	blockHeight := nextBlock
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(blockHeight)
	err = s.emissionsKeeper.SetRewardCurrentBlockEmission(s.ctx, cosmosMath.NewInt(100))
	s.Require().NoError(err)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	require.NoError(err)

	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)

	// Advance one to send the worker data
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(blockHeight + 1)

	workerIndexes := getIndexesFromValues(workerValues)

	inferenceBundles := generateSimpleWorkerDataBundles(s, topicId, topic.EpochLastEnded, blockHeight, workerValues, workerIndexes)
	for _, payload := range inferenceBundles {
		s.RegisterAllWorkersOfPayload(topicId, payload)
		_, err = s.msgServer.InsertWorkerPayload(s.ctx, &types.InsertWorkerPayloadRequest{
			Sender:           payload.Worker,
			WorkerDataBundle: payload,
		})
		require.NoError(err)
	}

	fmt.Println("Inference bundles:")
	printJSON(inferenceBundles)

	// Advance to close the window
	newBlock := blockHeight + topic.WorkerSubmissionWindow
	s.T().Logf("SubmissionWindow Next block: %v", nextBlock)
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlock)
	// EndBlock closes the nonce
	err = s.emissionsKeeper.SetRewardCurrentBlockEmission(s.ctx, cosmosMath.NewInt(100))
	s.Require().NoError(err)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)

	// Insert loss bundle from reputers
	lossBundles := generateSimpleLossBundles(
		s,
		topicId,
		topic.EpochLastEnded,
		workerValues,
		reputerValues,
		workerZeroAddress,
		workerZeroOneOutInfererValue,
		workerZeroInfererValue,
	)

	fmt.Println("Loss bundles:")
	printJSON(lossBundles.ReputerValueBundles[0].ValueBundle)

	newBlockheight := blockHeight + topic.GroundTruthLag
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(newBlockheight)
	for _, payload := range lossBundles.ReputerValueBundles {
		s.RegisterAllReputersOfPayload(topicId, payload)
		_, err = s.msgServer.InsertReputerPayload(s.ctx, &types.InsertReputerPayloadRequest{
			Sender:             payload.ValueBundle.Reputer,
			ReputerValueBundle: payload,
		})
		require.NoError(err)
	}
	err = actorutils.CloseReputerNonce(
		&s.emissionsKeeper, s.ctx, topic,
		*lossBundles.ReputerValueBundles[0].ValueBundle.ReputerRequestNonce.ReputerNonce,
	)
	s.Require().NoError(err)

	topicTotalRewards := alloraMath.NewDecFromInt64(1000000)

	rewardsDistributionByTopicParticipant, _, err := rewards.GenerateRewardsDistributionByTopicParticipant(
		rewards.GenerateRewardsDistributionByTopicParticipantArgs{
			Ctx:          s.ctx,
			K:            s.emissionsKeeper,
			TopicId:      topicId,
			TopicReward:  &topicTotalRewards,
			BlockHeight:  blockHeight,
			ModuleParams: params,
		},
	)
	require.NoError(err)

	return rewardsDistributionByTopicParticipant
}

func printJSON(i any) {
	jsn, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(jsn))
}

// We have 2 trials with 2 epochs each, and the first worker does better in the 2nd epoch in both trials.
// We show that keeping the TaskRewardAlpha the same means that the worker is rewarded the same amount
// in both cases.
// This is a sanity test to ensure that we are isolating the effect of TaskRewardAlpha in subsequent tests.
func (s *RewardsTestSuite) TestFixingTaskRewardAlphaDoesNotChangePerformanceImportanceOfPastVsPresent() {
	require := s.Require()

	currentParams, err := s.emissionsKeeper.GetParams(s.ctx)
	require.NoError(err)
	currentParams.TaskRewardAlpha = alloraMath.MustNewDecFromString("0.1")
	err = s.emissionsKeeper.SetParams(s.ctx, currentParams)
	require.NoError(err)

	blockHeight := int64(100)
	blockHeightDelta := int64(1)
	s.ctx = s.ctx.WithBlockHeight(blockHeight)

	workerIndexes := returnIndexes(0, 3)
	reputerIndexes := returnIndexes(3, 3)
	values := []string{"0.1", "0.2", "0.3"}
	workerValues := getWorkerValuesFromIndexes(workerIndexes, values...)
	reputerValues := s.getReputerValuesFromIndexes(reputerIndexes, workerIndexes, values...)
	stake := cosmosMath.NewInt(1000000000000000000).Mul(inferencesynthesis.CosmosIntOneE18())
	alphaRegret := alloraMath.MustNewDecFromString("0.1")

	topicPassOpts := []option{
		WithReputerStake(&stake),
		WithReputerValues(reputerValues),
		WithWorkerValues(workerValues),
		WithAlphaRegret(alphaRegret),
	}

	var (
		numTopics  = 2
		allRewards = make([][][]types.TaskReward, numTopics)
		topicIds   = make([]uint64, numTopics)
	)

	for topicNum := 0; topicNum < numTopics; topicNum++ {
		allRewards[topicNum] = make([][]types.TaskReward, 2)

		currentBlockHeight := blockHeight + int64(topicNum)*(blockHeightDelta*2)
		s.ctx = s.ctx.WithBlockHeight(currentBlockHeight)

		// pre-emptive run for topic setup
		topicId, newBlockHeight := s.FullTopicPass(currentBlockHeight, workerIndexes, reputerIndexes, topicPassOpts...)
		topicIds[topicNum] = topicId
		currentBlockHeight = newBlockHeight

		// for both parts of the test (A and B)
		for part := 0; part < 2; part++ {
			s.ctx = s.ctx.WithBlockHeight(currentBlockHeight)
			// run a full topic pass
			opts := append(topicPassOpts, WithTopicID(topicId))
			s.FullTopicPass(currentBlockHeight, workerIndexes, reputerIndexes, opts...)

			// generate rewards
			allRewards[topicNum][part], _ = s.GenerateRewards(topicId, newBlockHeight)
			currentBlockHeight = newBlockHeight + blockHeightDelta
		}
	}

	require.True(areTaskRewardsEqualIgnoringTopicId(s, allRewards[0][0], allRewards[1][0]),
		"First part rewards should be equal between topics")
	require.True(areTaskRewardsEqualIgnoringTopicId(s, allRewards[0][1], allRewards[1][1]),
		"Second part rewards should be equal between topics")
}

// We have 2 trials with 2 epochs each, and the first worker does better in the 2nd epoch in both trials,
// due to a worse one out inferer value, indicating that the network is better off with the worker.
// We increase TaskRewardAlpha between the trials to show that weighting current performance more heavily
// means that the worker is rewarded more for their better performance in the 2nd epoch of the 2nd trial.
func (s *RewardsTestSuite) TestIncreasingTaskRewardAlphaIncreasesImportanceOfPresentPerformance() {
	require := s.Require()
	k := s.emissionsKeeper

	blockHeight := int64(100)
	blockHeightDelta := int64(1)
	s.ctx = s.ctx.WithBlockHeight(blockHeight)

	workerIndexes := returnIndexes(0, 3)
	reputerIndexes := returnIndexes(3, 3)
	stake := cosmosMath.NewInt(1000000000000000000).Mul(inferencesynthesis.CosmosIntOneE18())
	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	workerValues := getWorkerValuesFromIndexes(workerIndexes, "0.1")
	// define the different reputer performance values for each test phase
	normalPerformance := s.getReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.1")
	improvedPerformance := make([]map[string]string, len(normalPerformance))
	copy(improvedPerformance, normalPerformance)
	improvedPerformance[0][s.addrsStr[0]] = "0.2" // first reputer performs better
	alphaValues := []string{"0.1", "0.2"}

	var (
		rewardsDistributions [2][2][]types.TaskReward // [alphaIndex][testPhase]
		topicIds             [2]uint64
		newBlockHeights      [2][2]int64 // [alphaIndex][phase]
	)

	// run tests for each alpha value
	for alphaIndex, alpha := range alphaValues {
		// set the task reward alpha for this test series
		currentParams, err := k.GetParams(s.ctx)
		require.NoError(err)
		currentParams.TaskRewardAlpha = alloraMath.MustNewDecFromString(alpha)
		err = k.SetParams(s.ctx, currentParams)
		require.NoError(err)

		currentBlockHeight := blockHeight + int64(alphaIndex)*(blockHeightDelta*10)
		s.ctx = s.ctx.WithBlockHeight(currentBlockHeight)

		baseOpts := []option{
			WithReputerStake(&stake),
			WithAlphaRegret(alphaRegret),
			WithWorkerValues(workerValues),
		}

		// pre-emptive run to create the topic
		topicId, newBlockHeight := s.FullTopicPass(
			currentBlockHeight,
			workerIndexes,
			reputerIndexes,
			append(baseOpts, WithReputerValues(normalPerformance))...,
		)
		topicIds[alphaIndex] = topicId
		currentBlockHeight = newBlockHeight + blockHeightDelta*5

		// for each test phase (normal performance and improved performance)
		for phase := 0; phase < 2; phase++ {
			// set the reputer values based on the phase
			reputerPerformance := normalPerformance
			if phase == 1 {
				reputerPerformance = improvedPerformance
			}

			// run a full topic pass
			s.ctx = s.ctx.WithBlockHeight(currentBlockHeight)
			opts := append(baseOpts,
				WithTopicID(topicId),
				WithReputerValues(reputerPerformance),
			)

			_, newBlockHeight := s.FullTopicPass(
				currentBlockHeight,
				workerIndexes,
				reputerIndexes,
				opts...,
			)

			newBlockHeights[alphaIndex][phase] = newBlockHeight
			// generate and store rewards for comparison
			rewardsDistributions[alphaIndex][phase], _ = s.GenerateRewards(topicId, currentBlockHeight)
			currentBlockHeight = newBlockHeight + blockHeightDelta*5
		}
	}

	// ASSERTIONS

	// extract worker[0] rewards for comparing phase-to-phase improvements
	var worker0RewardsPhase0, worker0RewardsPhase1 [2]alloraMath.Dec

	for alphaIndex := 0; alphaIndex < 2; alphaIndex++ {
		// get phase 0 reward
		for _, reward := range rewardsDistributions[alphaIndex][0] {
			if reward.Address == s.addrsStr[workerIndexes[0]] &&
				reward.Type == types.WorkerInferenceRewardType {
				worker0RewardsPhase0[alphaIndex] = reward.Reward
			}
		}

		// get phase 1 reward
		for _, reward := range rewardsDistributions[alphaIndex][1] {
			if reward.Address == s.addrsStr[workerIndexes[0]] &&
				reward.Type == types.WorkerInferenceRewardType {
				worker0RewardsPhase1[alphaIndex] = reward.Reward
			}
		}
	}

	require.True(worker0RewardsPhase0[0].Equal(worker0RewardsPhase0[1]),
		"With the current implementation, reputer reward in the first pass stays the same: %s == %s",
		alphaValues[0], worker0RewardsPhase0[0].String(), worker0RewardsPhase0[1].String())

	require.True(worker0RewardsPhase1[0].Lt(worker0RewardsPhase1[1]),
		"With the current implementation, reputer reward with lower alpha (%s) should be lower: %s > %s",
		alphaValues[0], worker0RewardsPhase1[0].String(), worker0RewardsPhase1[1].String())
}

// We have 2 trials with 2 epochs each, and the first worker does worse in 2nd epoch in both trials,
// enacted by their increasing loss between epochs.
// We increase alpha between the trials to prove that their worsening performance decreases regret.
// This is somewhat counterintuitive, but can be explained by the following passage from the litepaper:
// "A positive regret implies that the inference of worker j is expected by worker k to outperform
// the network's previously reported accuracy, whereas a negative regret indicates that the network
// is expected to be more accurate."
func (s *RewardsTestSuite) TestIncreasingAlphaRegretIncreasesPresentEffectOnRegret0() {
	// SETUP
	require := s.Require()
	k := s.emissionsKeeper

	currentParams, err := k.GetParams(s.ctx)
	require.NoError(err)

	blockHeight0 := int64(100)
	blockHeightDelta := int64(1)
	s.ctx = s.ctx.WithBlockHeight(blockHeight0)

	workerIndexes := returnIndexes(0, 3)
	reputerIndexes := returnIndexes(3, 3)

	stake := cosmosMath.NewInt(1000000000000000000).Mul(inferencesynthesis.CosmosIntOneE18())
	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	topicId0 := s.setUpTopic(blockHeight0, workerIndexes, reputerIndexes, stake, alphaRegret)

	workerValues := []TestWorkerValue{
		{Index: workerIndexes[0], Value: "0.1"},
		{Index: workerIndexes[1], Value: "0.2"},
		{Index: workerIndexes[2], Value: "0.3"},
	}

	reputerValues := []TestWorkerValue{
		{Index: reputerIndexes[0], Value: "0.1"},
		{Index: reputerIndexes[1], Value: "0.2"},
		{Index: reputerIndexes[2], Value: "0.3"},
	}

	topic, err := k.GetTopic(s.ctx, topicId0)
	s.Require().NoError(err)
	topic.AlphaRegret = alloraMath.MustNewDecFromString("0.1")
	err = k.SetTopic(s.ctx, topicId0, topic)
	require.NoError(err)

	// TEST 0 PART A

	s.getRewardsDistribution(
		topicId0,
		workerValues,
		reputerValues,
		s.addrs[workerIndexes[0]],
		"0.1",
		"0.1",
	)

	worker0_0, _, err := k.GetInfererNetworkRegret(s.ctx, topicId0, s.addrsStr[workerIndexes[0]])
	require.NoError(err)

	// TEST 0 PART B

	blockHeight1 := blockHeight0 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight1)

	s.getRewardsDistribution(
		topicId0,
		workerValues,
		reputerValues,
		s.addrs[workerIndexes[0]],
		"0.1",
		"0.2",
	)

	worker0_0FirstRegret := worker0_0.Value

	worker0_0, _, err = k.GetInfererNetworkRegret(s.ctx, topicId0, s.addrsStr[workerIndexes[0]])
	require.NoError(err)

	fmt.Println(worker0_0FirstRegret.String(), worker0_0.Value.String())

	worker1_0, _, err := k.GetInfererNetworkRegret(s.ctx, topicId0, s.addrsStr[workerIndexes[1]])
	require.NoError(err)

	worker2_0, _, err := k.GetInfererNetworkRegret(s.ctx, topicId0, s.addrsStr[workerIndexes[2]])
	require.NoError(err)

	worker0_0RegretDecrease, err := worker0_0FirstRegret.Sub(worker0_0.Value)
	require.NoError(err)
	worker0_0RegretDecreaseRate, err := worker0_0RegretDecrease.Quo(worker0_0FirstRegret)
	require.NoError(err)

	require.Truef(worker0_0RegretDecreaseRate.Equal(alphaRegret), "%s == %s", worker0_0RegretDecreaseRate, alphaRegret)

	// INCREASE ALPHA REGRET

	alphaRegret = alloraMath.MustNewDecFromString("0.2")
	currentParams.TaskRewardAlpha = alphaRegret
	err = k.SetParams(s.ctx, currentParams)
	require.NoError(err)

	// TEST 1 PART A

	blockHeight2 := blockHeight1 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight2)

	topicId1 := s.setUpTopic(blockHeight2, workerIndexes, reputerIndexes, stake, alphaRegret)

	s.getRewardsDistribution(
		topicId1,
		workerValues,
		reputerValues,
		s.addrs[workerIndexes[0]],
		"0.1",
		"0.1",
	)

	worker0_1, _, err := k.GetInfererNetworkRegret(s.ctx, topicId1, s.addrsStr[workerIndexes[0]])
	require.NoError(err)

	// TEST 1 PART B

	blockHeight3 := blockHeight2 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight3)

	s.getRewardsDistribution(
		topicId1,
		workerValues,
		reputerValues,
		s.addrs[workerIndexes[0]],
		"0.1",
		"0.2",
	)

	worker0_1FirstRegret := worker0_1.Value

	worker0_1, _, err = k.GetInfererNetworkRegret(s.ctx, topicId1, s.addrsStr[workerIndexes[0]])
	require.NoError(err)

	worker1_1, _, err := k.GetInfererNetworkRegret(s.ctx, topicId1, s.addrsStr[workerIndexes[1]])
	require.NoError(err)

	worker2_1, _, err := k.GetInfererNetworkRegret(s.ctx, topicId1, s.addrsStr[workerIndexes[2]])
	require.NoError(err)

	worker0_1RegretDecrease, err := worker0_1FirstRegret.Sub(worker0_1.Value)
	require.NoError(err)
	worker0_1RegretDecreaseRate, err := worker0_1RegretDecrease.Quo(worker0_1FirstRegret)
	require.NoError(err)
	require.True(worker0_1RegretDecreaseRate.Equal(alphaRegret))

	// Check alpha impact in regrets - Topic 0 (alpha 0.1) vs Topic 1 (alpha 0.2)
	require.True(worker0_1RegretDecreaseRate.Gt(worker0_0RegretDecreaseRate))

	require.True(alloraMath.InDelta(worker1_0.Value, worker1_1.Value, alloraMath.MustNewDecFromString("0.00001")))
	require.True(worker2_0.Value.Gt(worker2_1.Value))
}

func (s *RewardsTestSuite) TestIncreasingAlphaRegretIncreasesPresentEffectOnRegret1() {
	// SETUP
	require := s.Require()
	k := s.emissionsKeeper

	currentParams, err := k.GetParams(s.ctx)
	require.NoError(err)

	blockHeight0 := int64(100)
	blockHeightDelta := int64(1)
	s.ctx = s.ctx.WithBlockHeight(blockHeight0)

	workerIndexes := returnIndexes(0, 3)
	workerAddresses := make([]string, len(workerIndexes))
	for _, workerIndex := range workerIndexes {
		workerAddresses[workerIndex] = s.addrsStr[workerIndex]
	}
	slices.Sort(workerAddresses)

	workerIndexes = []int{
		slices.Index(s.addrsStr, workerAddresses[0]),
		slices.Index(s.addrsStr, workerAddresses[1]),
		slices.Index(s.addrsStr, workerAddresses[2]),
	}

	reputerIndexes := returnIndexes(3, 3)

	stake := cosmosMath.NewInt(1000000000000000000).Mul(inferencesynthesis.CosmosIntOneE18())
	alphaRegret := alloraMath.MustNewDecFromString("0.1")

	currentParams.TaskRewardAlpha = alphaRegret
	err = k.SetParams(s.ctx, currentParams)

	workerValues := []TestWorkerValue{
		{Index: workerIndexes[0], Value: "0.1"},
		{Index: workerIndexes[1], Value: "0.2"},
		{Index: workerIndexes[2], Value: "0.3"},
	}

	addrs := []string{
		s.addrsStr[0],
		s.addrsStr[1],
		s.addrsStr[2],
	}
	slices.Sort(addrs)

	reputerValues := []map[string]string{
		{
			addrs[0]: "0.1",
			addrs[1]: "0.2",
			addrs[2]: "0.3",
		}, {
			addrs[0]: "0.1",
			addrs[1]: "0.2",
			addrs[2]: "0.3",
		}, {
			addrs[0]: "0.1",
			addrs[1]: "0.2",
			addrs[2]: "0.3",
		},
	}

	// TEST 0 PART A

	// pre-emptive run
	topicId0, blockHeight0 := s.FullTopicPass(
		blockHeight0,
		workerIndexes,
		reputerIndexes,
		WithReputerStake(&stake),
		WithWorkerValues(workerValues),
		WithReputerValues(reputerValues),
		WithAlphaRegret(alphaRegret),
	)

	blockHeight0 = blockHeight0 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight0)

	s.FullTopicPass(
		blockHeight0,
		workerIndexes,
		reputerIndexes,
		WithReputerStake(&stake),
		WithWorkerValues(workerValues),
		WithReputerValues(reputerValues),
		WithAlphaRegret(alphaRegret),
		WithTopicID(topicId0),
	)

	worker0_0, _, err := k.GetInfererNetworkRegret(s.ctx, topicId0, s.addrsStr[workerIndexes[0]])
	require.NoError(err)

	// TEST 0 PART B

	blockHeight1 := blockHeight0 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight1)

	reputerValues[0][addrs[0]] = "0.2"

	s.FullTopicPass(
		blockHeight1,
		workerIndexes,
		reputerIndexes,
		WithReputerStake(&stake),
		WithWorkerValues(workerValues),
		WithReputerValues(reputerValues),
		WithAlphaRegret(alphaRegret),
		WithTopicID(topicId0),
	)

	worker0_0FirstRegret := worker0_0.Value

	worker0_0, _, err = k.GetInfererNetworkRegret(s.ctx, topicId0, s.addrsStr[workerIndexes[0]])
	require.NoError(err)

	fmt.Println(worker0_0FirstRegret.String(), worker0_0.Value.String())

	worker1_0, _, err := k.GetInfererNetworkRegret(s.ctx, topicId0, s.addrsStr[workerIndexes[1]])
	require.NoError(err)

	worker2_0, _, err := k.GetInfererNetworkRegret(s.ctx, topicId0, s.addrsStr[workerIndexes[2]])
	require.NoError(err)

	worker0_0RegretDecrease, err := worker0_0FirstRegret.Sub(worker0_0.Value)
	require.NoError(err)
	worker0_0RegretDecreaseRate, err := worker0_0RegretDecrease.Quo(worker0_0FirstRegret)
	require.NoError(err)

	require.Truef(worker0_0RegretDecreaseRate.Equal(alphaRegret), "%s == %s", worker0_0RegretDecreaseRate, alphaRegret)

	// INCREASE ALPHA REGRET

	alphaRegret = alloraMath.MustNewDecFromString("0.2")
	currentParams.TaskRewardAlpha = alphaRegret
	err = k.SetParams(s.ctx, currentParams)
	require.NoError(err)

	// TEST 1 PART A

	blockHeight2 := blockHeight1 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight2)

	topicId1 := s.setUpTopic(blockHeight2, workerIndexes, reputerIndexes, stake, alphaRegret)
	/*
		s.getRewardsDistribution(
			topicId1,
			workerValues,
			reputerValues,
			s.addrs[workerIndexes[0]],
			"0.1",
			"0.1",
		)*/

	worker0_1, _, err := k.GetInfererNetworkRegret(s.ctx, topicId1, s.addrsStr[workerIndexes[0]])
	require.NoError(err)

	// TEST 1 PART B

	blockHeight3 := blockHeight2 + blockHeightDelta
	s.ctx = s.ctx.WithBlockHeight(blockHeight3)

	/*s.getRewardsDistribution(
		topicId1,
		workerValues,
		reputerValues,
		s.addrs[workerIndexes[0]],
		"0.1",
		"0.2",
	)*/

	worker0_1FirstRegret := worker0_1.Value

	worker0_1, _, err = k.GetInfererNetworkRegret(s.ctx, topicId1, s.addrsStr[workerIndexes[0]])
	require.NoError(err)

	worker1_1, _, err := k.GetInfererNetworkRegret(s.ctx, topicId1, s.addrsStr[workerIndexes[1]])
	require.NoError(err)

	worker2_1, _, err := k.GetInfererNetworkRegret(s.ctx, topicId1, s.addrsStr[workerIndexes[2]])
	require.NoError(err)

	worker0_1RegretDecrease, err := worker0_1FirstRegret.Sub(worker0_1.Value)
	require.NoError(err)
	worker0_1RegretDecreaseRate, err := worker0_1RegretDecrease.Quo(worker0_1FirstRegret)
	require.NoError(err)
	require.True(worker0_1RegretDecreaseRate.Equal(alphaRegret))

	// Check alpha impact in regrets - Topic 0 (alpha 0.1) vs Topic 1 (alpha 0.2)
	require.True(worker0_1RegretDecreaseRate.Gt(worker0_0RegretDecreaseRate))

	require.True(alloraMath.InDelta(worker1_0.Value, worker1_1.Value, alloraMath.MustNewDecFromString("0.00001")))
	require.True(worker2_0.Value.Gt(worker2_1.Value))
}

// We have 2 trials with 2 epochs each, and the reputer does worse in 2nd epoch in both trials,
// enacted by their increasing loss between epochs.
// We increase alpha between the trials to prove that their worsening performance decreases regret.
// This is somewhat counterintuitive, but can be explained by the following passage from the litepaper:
// "A positive regret implies that the inference of worker j is expected by worker k to outperform
// the network's previously reported accuracy, whereas a negative regret indicates that the network
// is expected to be more accurate."
func (s *RewardsTestSuite) TestIncreasingAlphaRegretIncreasesPresentEffectOnRegret() {
	// SETUP
	require := s.Require()
	k := s.emissionsKeeper

	// Initialize indices and values
	workerIndexes := returnIndexes(0, 3)
	reputerIndexes := returnIndexes(3, 3)

	// Common test parameters
	stake := cosmosMath.NewInt(1000000000000000000).Mul(cosmosMath.NewInt(1000000000000000000))
	blockHeight := int64(100)
	blockHeightDelta := int64(21) // Enough blocks to pass between phases
	epochLength := int64(60)

	// Alpha values for the two trials
	alphaValues := []string{"0.1", "0.2"}

	// Create constant reputer values across all phases
	reputerValues := s.getReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.1")

	// Worker values for different phases
	workerValues := [][]TestWorkerValue{
		// Phase 1 - initial values
		getWorkerValuesFromIndexes(workerIndexes, "0.1"),
		// Phase 2 - worse values (increased loss)
		getWorkerValuesFromIndexes(workerIndexes, "0.2"),
	}

	// Common options for FullTopicPass
	baseOptions := func(alphaRegret alloraMath.Dec, topicId uint64) []option {
		return []option{
			WithEpochLength(epochLength),
			WithTopicID(topicId),
			WithReputerStake(&stake),
			WithAlphaRegret(alphaRegret),
			WithReputerValues(reputerValues),
		}
	}

	// Define a trial structure to store results
	type Trial struct {
		alphaRegret   alloraMath.Dec
		topicId       uint64
		blockHeight   int64
		firstRegret   alloraMath.Dec
		secondRegret  alloraMath.Dec
		decrease      alloraMath.Dec
		decreaseRate  alloraMath.Dec
		workerRegrets map[int]alloraMath.Dec
	}

	trials := make([]Trial, 2)

	// Run both trials
	for i := range trials {
		trial := &trials[i]
		trial.alphaRegret = alloraMath.MustNewDecFromString(alphaValues[i])
		trial.blockHeight = blockHeight
		trial.workerRegrets = make(map[int]alloraMath.Dec)

		// Set task reward alpha in params for second trial
		if i == 1 {
			currentParams, err := k.GetParams(s.ctx)
			require.NoError(err)
			currentParams.TaskRewardAlpha = trial.alphaRegret
			err = k.SetParams(s.ctx, currentParams)
			require.NoError(err)
		}

		// Prepare options for this trial
		options := baseOptions(trial.alphaRegret, trial.topicId)

		// PHASE 0: Pre-emptive run to set up the topic
		s.ctx = s.ctx.WithBlockHeight(trial.blockHeight)

		// Set up topic and do initial run with first phase worker values
		phaseOptions := append(options, WithWorkerValues(workerValues[0]))
		trial.topicId, _ = s.FullTopicPass(trial.blockHeight, workerIndexes, reputerIndexes, phaseOptions...)

		// PHASE 1: First real run with initial worker values
		trial.blockHeight += blockHeightDelta
		s.ctx = s.ctx.WithBlockHeight(trial.blockHeight)

		phaseOptions = append(baseOptions(trial.alphaRegret, trial.topicId), WithWorkerValues(workerValues[0]))
		s.FullTopicPass(trial.blockHeight, workerIndexes, reputerIndexes, phaseOptions...)

		// Get first epoch regret for worker 0
		worker0, _, err := k.GetInfererNetworkRegret(s.ctx, trial.topicId, s.addrsStr[workerIndexes[0]])
		require.NoError(err)
		trial.firstRegret = worker0.Value

		// PHASE 2: Second run with worse worker values
		trial.blockHeight += blockHeightDelta
		s.ctx = s.ctx.WithBlockHeight(trial.blockHeight)

		phaseOptions = append(baseOptions(trial.alphaRegret, trial.topicId), WithWorkerValues(workerValues[1]))
		s.FullTopicPass(trial.blockHeight, workerIndexes, reputerIndexes, phaseOptions...)

		// Collect regrets for all workers
		for j, idx := range workerIndexes {
			worker, _, err := k.GetInfererNetworkRegret(s.ctx, trial.topicId, s.addrsStr[idx])
			require.NoError(err)
			if j == 0 {
				trial.secondRegret = worker.Value
			} else {
				trial.workerRegrets[idx] = worker.Value
			}
		}

		// Calculate regret decrease
		trial.decrease, err = trial.firstRegret.Sub(trial.secondRegret)
		require.NoError(err)
		trial.decreaseRate, err = trial.decrease.Quo(trial.firstRegret)
		require.NoError(err)

		// Verify the decrease rate equals the alpha
		require.True(trial.decreaseRate.Equal(trial.alphaRegret),
			"Regret decrease rate should equal alpha: got %s, expected %s",
			trial.decreaseRate.String(), trial.alphaRegret.String())

		// Set up for next trial
		blockHeight = trial.blockHeight + blockHeightDelta
	}

	// VERIFY CROSS-TRIAL EFFECTS

	// Check that higher alpha resulted in greater regret decrease rate
	require.True(trials[1].decreaseRate.Gt(trials[0].decreaseRate),
		"Higher alpha should cause greater regret decrease rate")

	// Verify other workers' regrets behave as expected across trials
	worker1Idx := workerIndexes[1]
	worker2Idx := workerIndexes[2]

	delta, err := alloraMath.InDelta(
		trials[0].workerRegrets[worker1Idx],
		trials[1].workerRegrets[worker1Idx],
		alloraMath.MustNewDecFromString("0.00001"))
	require.NoError(err)
	require.True(delta, "Worker 1 regrets should be nearly identical across trials")

	require.True(trials[0].workerRegrets[worker2Idx].Gt(trials[1].workerRegrets[worker2Idx]),
		"Worker 2 regret in trial 0 should be greater than in trial 1")
}

func (s *RewardsTestSuite) TestGenerateTasksRewardsShouldIncreaseRewardShareIfMoreParticipants() {
	block := int64(100)
	s.ctx = s.ctx.WithBlockHeight(block)

	// TOPIC 1 - PRE PASS
	workerIndexes := returnIndexes(5, 5)
	reputerIndexes := returnIndexes(0, 3)
	topicId, block := s.FullTopicPass(block, workerIndexes, reputerIndexes)

	// TOPIC 1 - FIRST PASS
	s.FullTopicPass(block, workerIndexes, reputerIndexes, WithTopicID(topicId))

	firstRewardsDistribution, firstTotalReputerReward := s.GenerateRewards(topicId, block)

	var (
		calcFirstTotalReputerReward = alloraMath.ZeroDec()
		err                         error
	)
	for _, reward := range firstRewardsDistribution {
		if reward.Type == types.ReputerAndDelegatorRewardType {
			calcFirstTotalReputerReward, err = calcFirstTotalReputerReward.Add(reward.Reward)
			s.Require().NoError(err)
		}
	}
	inDelta, err := alloraMath.InDelta(
		firstTotalReputerReward,
		calcFirstTotalReputerReward,
		alloraMath.MustNewDecFromString("0.0001"),
	)
	s.Require().NoError(err)
	s.Require().True(
		inDelta,
		"expected: %s, got: %s",
		firstTotalReputerReward.String(),
		calcFirstTotalReputerReward.String(),
	)

	block += 1
	s.ctx = s.ctx.WithBlockHeight(block)

	// TOPIC 2 - PRE PASS
	workerIndexes2 := returnIndexes(5, 5)
	reputerIndexes2 := returnIndexes(0, 5)
	topicId2, block := s.FullTopicPass(block, workerIndexes2, reputerIndexes2)

	// TOPIC 2 - FIRST PASS
	s.FullTopicPass(block, workerIndexes2, reputerIndexes2, WithTopicID(topicId2))

	secondRewardsDistribution, secondTotalReputerReward := s.GenerateRewards(topicId2, block)

	calcSecondTotalReputerReward := alloraMath.ZeroDec()
	for _, reward := range secondRewardsDistribution {
		if reward.Type == types.ReputerAndDelegatorRewardType {
			calcSecondTotalReputerReward, err = calcSecondTotalReputerReward.Add(reward.Reward)
			s.Require().NoError(err)
		}
	}
	inDelta, err = alloraMath.InDelta(
		secondTotalReputerReward,
		calcSecondTotalReputerReward,
		alloraMath.MustNewDecFromString("0.0001"),
	)
	s.Require().NoError(err)
	s.Require().True(
		inDelta,
		"expected: %s, got: %s",
		secondTotalReputerReward.String(),
		calcSecondTotalReputerReward.String(),
	)

	// Check if the reward share increased
	s.Require().True(secondTotalReputerReward.Gt(firstTotalReputerReward))
}

func (s *RewardsTestSuite) TestMultipleEpochsWeightAndStdNormEvolution() {
	require := s.Require()

	// Initial setup
	block := int64(1)
	s.ctx = s.ctx.WithBlockHeight(block)

	s.SetParamsForTest()

	// Create topic with shorter epoch length for testing multiple epochs
	epochLength := int64(5)
	workerIndexes := returnIndexes(0, 3)
	reputerIndexes := returnIndexes(3, 3)
	topicId, block := s.FullTopicPass(
		block,
		workerIndexes,
		reputerIndexes,
		WithAlphaRegret(alloraMath.MustNewDecFromString("0.1")),
		WithEpochLength(epochLength),
	)

	// Track weights and stdnorm over epochs
	var (
		workerWeights = make(map[string][]alloraMath.Dec)
		stdNorms      []alloraMath.Dec
	)

	const numEpochs = 5
	// Run multiple epochs
	for epoch := 0; epoch < numEpochs; epoch++ {
		// Get current weight and stdnorm before processing
		stdNorm, err := s.emissionsKeeper.GetLatestRegretStdNorm(s.ctx, topicId)
		require.NoError(err)
		stdNorms = append(stdNorms, stdNorm)

		_, newBlock := s.FullTopicPass(
			block,
			workerIndexes,
			reputerIndexes,
			WithTopicID(topicId),
			WithAlphaRegret(alloraMath.MustNewDecFromString("0.1")),
			WithEpochLength(epochLength),
		)

		s.GenerateRewards(topicId, block)

		for _, index := range workerIndexes {
			weight, err := s.emissionsKeeper.GetLatestInfererWeight(s.ctx, topicId, s.addrsStr[index])
			s.Require().NoError(err)
			workerWeights[s.addrsStr[index]] = append(workerWeights[s.addrsStr[index]], weight)
		}

		// Mint rewards
		s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

		// Move to next epoch
		block = newBlock + epochLength
		s.ctx = s.ctx.WithBlockHeight(block)

		// Distribute rewards
		err = s.emissionsKeeper.SetRewardCurrentBlockEmission(s.ctx, cosmosMath.NewInt(100))
		require.NoError(err)
		err = s.emissionsAppModule.EndBlock(s.ctx)
		require.NoError(err)
	}

	// Verify weight evolution
	require.Len(workerWeights, 3) // one entry per worker
	for i := 1; i < len(workerWeights); i++ {
		// Weight for the same worker should change between epochs
		currentWorker := s.addrsStr[i]
		currentWorkerWeights := workerWeights[currentWorker]
		// check weights are different from diff actors
		for j := 1; j < len(currentWorkerWeights); j++ {
			require.NotEqual(
				currentWorkerWeights[j].String(),
				currentWorkerWeights[j-1].String(),
				"Weight should change between epochs %d and %d", j-1, j,
			)
		}
		s.T().Logf("Worker %d , %s Weight: %v", i, s.addrsStr[i], currentWorkerWeights)
	}

	// Verify stdnorm evolution
	require.Len(stdNorms, numEpochs)
	for i := 1; i < len(stdNorms); i++ {
		// StdNorm should adapt based on predictions
		require.NotEqual(
			stdNorms[i].String(),
			stdNorms[i-1].String(),
			"StdNorm should change between epochs %d and %d", i-1, i,
		)
		s.T().Logf("StdNorm: %v", stdNorms[i].String())
	}
}

func (s *RewardsTestSuite) TestRewardsIncreasesBalance() {
	block := int64(600)
	s.ctx = s.ctx.WithBlockHeight(block)
	epochLength := int64(10800)
	s.MintTokensToModule(types.AlloraStakingAccountName, cosmosMath.NewInt(10000000000))

	workerIndexes := returnIndexes(5, 5)
	reputerIndexes := returnIndexes(0, 5)
	// TOPIC 1 - PRE PASS
	topicId, block := s.FullTopicPass(block, workerIndexes, reputerIndexes)

	// TOPIC 1 - FIRST PASS
	_, newBlockheight := s.FullTopicPass(block, workerIndexes, reputerIndexes, WithTopicID(topicId))

	var err error
	reputerBalances := make([]sdk.Coin, 5)
	reputerStake := make([]cosmosMath.Int, 5)
	for _, index := range reputerIndexes {
		reputerBalances[index] = s.bankKeeper.GetBalance(s.ctx, s.addrs[index], params.DefaultBondDenom)
		reputerStake[index], err = s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId, s.addrsStr[index])
		s.Require().NoError(err)
	}

	workerBalances := make(map[int]sdk.Coin)
	for _, index := range workerIndexes {
		workerBalances[index] = s.bankKeeper.GetBalance(s.ctx, s.addrs[index], params.DefaultBondDenom)
	}

	// mint some rewards to give out
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	// Trigger end block - rewards distribution
	newBlockheight += epochLength
	s.ctx = s.ctx.WithBlockHeight(newBlockheight)
	err = s.emissionsKeeper.SetRewardCurrentBlockEmission(s.ctx, cosmosMath.NewInt(100))
	s.Require().NoError(err)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)

	for i, index := range reputerIndexes {
		reputerStakeCurrent, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId, s.addrsStr[index])
		s.Require().NoError(err)
		s.Require().True(
			reputerStakeCurrent.GT(reputerStake[i]),
			"Reputer %s stake did not increase: %s | %s",
			s.addrsStr[index],
			reputerStakeCurrent.String(),
			reputerStake[i].String(),
		)
	}

	for _, index := range workerIndexes {
		s.Require().True(s.bankKeeper.GetBalance(s.ctx, s.addrs[index], params.DefaultBondDenom).Amount.GT(workerBalances[index].Amount))
	}
}

func (s *RewardsTestSuite) TestStandardRewardEmissionWithOneInfererAndOneReputer() {
	blockHeight := int64(600)
	s.ctx = s.ctx.WithBlockHeight(blockHeight)
	epochLength := int64(10800)

	s.FullTopicPass(blockHeight, []int{0}, []int{5}, WithEpochLength(epochLength))

	blockHeight += epochLength * 3
	s.ctx = s.ctx.WithBlockHeight(blockHeight)

	// mint some rewards to give out
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(10000000000))

	// Trigger end block - rewards distribution
	s.EndBlock()
}

func (s *RewardsTestSuite) TestOnlyFewTopActorsGetReward() {
	block := int64(600)
	s.ctx = s.ctx.WithBlockHeight(block)
	epochLength := int64(10800)
	s.SetParamsForTest()

	// Reputer Addresses
	reputerIndexes := returnIndexes(0, 25)
	workerIndexes := returnIndexes(25, 25)

	topicId, block := s.FullTopicPass(block, workerIndexes, reputerIndexes, WithEpochLength(epochLength))
	s.FullTopicPass(block, workerIndexes, reputerIndexes, WithEpochLength(epochLength), WithTopicID(topicId))

	networkLossBundles, err := s.emissionsKeeper.GetNetworkLossBundleAtBlock(s.ctx, topicId, block)
	s.Require().NoError(err)
	s.Require().NotNil(networkLossBundles)

	infererScores, err := rewards.GenerateInferenceScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block,
		*networkLossBundles)
	s.Require().NoError(err)

	forecasterScores, err := rewards.GenerateForecastScores(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		block,
		*networkLossBundles)
	s.Require().NoError(err)

	p, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)

	s.Require().Equal(p.GetMaxTopInferersToReward(), uint64(len(infererScores)), "Only few Top inferers can get reward")
	s.Require().Equal(p.GetMaxTopForecastersToReward(), uint64(len(forecasterScores)), "Only few Top forecasters can get reward")
}

// TestRewardForTopicGoesUpWhenRelativeStakeGoesUp tests that the reward for a topic increases
// when its relative stake compared to other topics increases.
//
// Setup:
// - Create two topics (topicId0 and topicId1) with identical initial stakes, workers, and reputers
// - Set up identical worker and reputer values for both topics
// - Record initial stakes for reputers on both topics
//
// Expected outcomes:
// 1. Initially, rewards for both topics should be similar due to identical setups
// 2. After increasing stake on one topic:
//   - The reward for the topic with increased stake should be higher
//   - The reward for the topic with unchanged stake should be lower
//
// 3. The total rewards across both topics should remain constant
//
// This test demonstrates that the reward distribution mechanism correctly
// adjusts rewards based on the relative stakes of topics in the network.
func (s *RewardsTestSuite) TestRewardForTopicGoesUpWhenRelativeStakeGoesUp() {
	// setup
	require := s.Require()

	alphaRegret := alloraMath.MustNewDecFromString("0.1")

	block := int64(1)
	s.ctx = s.ctx.WithBlockHeight(block)

	s.SetParamsForTest()
	reputerIndexes := returnIndexes(0, 3)
	workerIndexes := returnIndexes(3, 3)

	// setup topics
	stake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())

	epochLength := int64(100)
	topicId0 := s.setUpTopicWithEpochLength(block, workerIndexes, reputerIndexes, stake, alphaRegret, epochLength)
	topicId1 := s.setUpTopicWithEpochLength(block, workerIndexes, reputerIndexes, stake, alphaRegret, epochLength)
	// setup values to be identical for both topics
	reputerValues := []TestWorkerValue{
		{Index: reputerIndexes[0], Value: "0.2"},
		{Index: reputerIndexes[1], Value: "0.2"},
		{Index: reputerIndexes[2], Value: "0.2"},
	}

	workerValues := []TestWorkerValue{
		{Index: workerIndexes[0], Value: "0.2"},
		{Index: workerIndexes[1], Value: "0.2"},
		{Index: workerIndexes[2], Value: "0.2"},
	}

	// record the stakes on each topic so we can see the reward differences
	reputer0_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrsStr[reputerIndexes[0]])
	require.NoError(err)
	reputer1_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrsStr[reputerIndexes[1]])
	require.NoError(err)
	reputer2_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrsStr[reputerIndexes[2]])
	require.NoError(err)

	reputer3_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, s.addrsStr[reputerIndexes[0]])
	require.NoError(err)
	reputer4_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, s.addrsStr[reputerIndexes[1]])
	require.NoError(err)
	reputer5_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, s.addrsStr[reputerIndexes[2]])
	require.NoError(err)

	// do work on the topics to earn rewards. Beware: there are moving of blocks, so it's different epochs where
	// topicId0 and topicId1 get their action done.
	s.getRewardsDistribution(
		topicId0,
		workerValues,
		reputerValues,
		s.addrs[workerIndexes[0]],
		"0.1",
		"0.1",
	)

	s.getRewardsDistribution(
		topicId1,
		workerValues,
		reputerValues,
		s.addrs[workerIndexes[0]],
		"0.1",
		"0.1",
	)

	// force rewards to be distributed
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	nextBlock, _, err := s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, topicId0)

	s.T().Logf("Next block: %v", nextBlock)
	s.Require().NoError(err)
	s.ctx = s.ctx.WithBlockHeight(nextBlock)

	// Check that the total sum of previous topic weights is equal to the sum of the weights of the two topics
	topic1Weight0, _, err := s.emissionsKeeper.GetPreviousTopicWeight(s.ctx, topicId0)
	s.Require().NoError(err)
	topic0Weight1, _, err := s.emissionsKeeper.GetPreviousTopicWeight(s.ctx, topicId1)
	s.Require().NoError(err)
	totalSumPreviousTopicWeights, err := s.emissionsKeeper.GetTotalSumPreviousTopicWeights(s.ctx)
	s.Require().NoError(err)
	sumWeights, err := topic0Weight1.Add(topic1Weight0)
	s.Require().NoError(err)
	inDelta, err := alloraMath.InDelta(totalSumPreviousTopicWeights, sumWeights, alloraMath.MustNewDecFromString("0.0001"))
	s.Require().NoError(err)
	s.Require().True(inDelta, "Total sum of previous topic weights %s + %s = %s is not equal to the sum of the weights of the two topics %s", topic0Weight1, topic1Weight0, totalSumPreviousTopicWeights, sumWeights)

	err = s.emissionsKeeper.SetRewardCurrentBlockEmission(s.ctx, cosmosMath.NewInt(100))
	s.Require().NoError(err)
	err = s.emissionsAppModule.EndBlock(s.ctx)

	require.NoError(err)

	worker1InclusionNum, err := s.emissionsKeeper.GetCountInfererInclusionsInTopic(s.ctx, topicId0, s.addrsStr[workerIndexes[0]])
	require.NoError(err)
	require.Equal(uint64(1), worker1InclusionNum)
	worker2InclusionNum, err := s.emissionsKeeper.GetCountInfererInclusionsInTopic(s.ctx, topicId0, s.addrsStr[workerIndexes[1]])
	require.NoError(err)
	require.Equal(uint64(1), worker2InclusionNum)
	worker3InclusionNum, err := s.emissionsKeeper.GetCountInfererInclusionsInTopic(s.ctx, topicId0, s.addrsStr[workerIndexes[2]])
	require.Equal(uint64(1), worker3InclusionNum)
	require.NoError(err)

	worker1InclusionNum, err = s.emissionsKeeper.GetCountForecasterInclusionsInTopic(s.ctx, topicId0, s.addrsStr[workerIndexes[0]])
	require.NoError(err)
	require.Equal(uint64(1), worker1InclusionNum)
	worker2InclusionNum, err = s.emissionsKeeper.GetCountForecasterInclusionsInTopic(s.ctx, topicId0, s.addrsStr[workerIndexes[1]])
	require.NoError(err)
	require.Equal(uint64(1), worker2InclusionNum)
	worker3InclusionNum, err = s.emissionsKeeper.GetCountForecasterInclusionsInTopic(s.ctx, topicId0, s.addrsStr[workerIndexes[2]])
	require.Equal(uint64(1), worker3InclusionNum)
	require.NoError(err)
	const topicFundAmount int64 = 1000

	fundTopic := func(topicId uint64, funderAddr sdk.AccAddress, amount int64) {
		s.MintTokensToAddress(funderAddr, cosmosMath.NewInt(amount))
		fundTopicMessage := types.FundTopicRequest{
			Sender:  funderAddr.String(),
			TopicId: topicId,
			Amount:  cosmosMath.NewInt(amount),
		}
		_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
		require.NoError(err)
	}

	fundTopic(topicId0, s.addrs[0], topicFundAmount)
	fundTopic(topicId1, s.addrs[3], topicFundAmount)

	reputer0_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrsStr[reputerIndexes[0]])
	require.NoError(err)
	reputer1_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrsStr[reputerIndexes[1]])
	require.NoError(err)
	reputer2_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrsStr[reputerIndexes[2]])
	require.NoError(err)

	reputer3_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, s.addrsStr[reputerIndexes[0]])
	require.NoError(err)
	reputer4_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, s.addrsStr[reputerIndexes[1]])
	require.NoError(err)
	reputer5_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, s.addrsStr[reputerIndexes[2]])
	require.NoError(err)

	reputer0_Reward0 := reputer0_Stake1.Sub(reputer0_Stake0)
	reputer1_Reward0 := reputer1_Stake1.Sub(reputer1_Stake0)
	reputer2_Reward0 := reputer2_Stake1.Sub(reputer2_Stake0)
	reputer3_Reward0 := reputer3_Stake1.Sub(reputer3_Stake0)
	reputer4_Reward0 := reputer4_Stake1.Sub(reputer4_Stake0)
	reputer5_Reward0 := reputer5_Stake1.Sub(reputer5_Stake0)

	topic0RewardTotal0 := reputer0_Reward0.Add(reputer1_Reward0).Add(reputer2_Reward0)
	topic1RewardTotal0 := reputer3_Reward0.Add(reputer4_Reward0).Add(reputer5_Reward0)

	require.Equal(topic0RewardTotal0, topic1RewardTotal0)

	// Now, in second trial, significantly increase stake for first reputer in topic1
	stakeIncrease := cosmosMath.NewInt(1000).Mul(stake)
	s.MintTokensToAddress(s.addrs[reputerIndexes[0]], stakeIncrease)
	_, err = s.msgServer.AddStake(s.ctx, &types.AddStakeRequest{
		Sender:  s.addrsStr[reputerIndexes[0]],
		Amount:  stakeIncrease,
		TopicId: topicId1,
	})
	require.NoError(err)

	// record the updated stakes
	reputer3_Stake1, err = s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, s.addrsStr[reputerIndexes[0]])
	require.NoError(err)

	// force rewards to be distributed
	block = s.ctx.BlockHeight()
	block++
	s.ctx = s.ctx.WithBlockHeight(block)

	// do work on the topics to earn rewards. Beware: there are moving of blocks, so it's different epochs where
	// topicId0 and topicId1 get their action done.
	s.getRewardsDistribution(
		topicId0,
		workerValues,
		reputerValues,
		s.addrs[workerIndexes[0]],
		"0.1",
		"0.1",
	)
	taskRewards1 := s.getRewardsDistribution(
		topicId1,
		workerValues,
		reputerValues,
		s.addrs[workerIndexes[0]],
		"0.1",
		"0.1",
	)

	// Get and check the rewards for the reputer after the stake increase
	var taskRewardsReputer3AfterStakeIncrease,
		taskRewardsReputer4AfterStakeIncrease,
		taskRewardsReputer5AfterStakeIncrease types.TaskReward

	for _, reward := range taskRewards1 {
		if reward.Type == types.ReputerAndDelegatorRewardType {
			if reward.Address == s.addrsStr[reputerIndexes[0]] {
				taskRewardsReputer3AfterStakeIncrease = reward
			}
			if reward.Address == s.addrsStr[reputerIndexes[1]] {
				taskRewardsReputer4AfterStakeIncrease = reward
			}
			if reward.Address == s.addrsStr[reputerIndexes[2]] {
				taskRewardsReputer5AfterStakeIncrease = reward
			}
		}
	}
	require.True(taskRewardsReputer3AfterStakeIncrease.Reward.Gt(taskRewardsReputer4AfterStakeIncrease.Reward))
	require.True(taskRewardsReputer3AfterStakeIncrease.Reward.Gt(taskRewardsReputer5AfterStakeIncrease.Reward))

	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	err = s.emissionsKeeper.SetRewardCurrentBlockEmission(s.ctx, cosmosMath.NewInt(100))
	s.Require().NoError(err)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	require.NoError(err)

	// record the stakes after
	reputer0_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrsStr[reputerIndexes[0]])
	require.NoError(err)
	reputer1_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrsStr[reputerIndexes[1]])
	require.NoError(err)
	reputer2_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrsStr[reputerIndexes[2]])
	require.NoError(err)

	reputer3_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, s.addrsStr[reputerIndexes[0]])
	require.NoError(err)
	reputer4_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, s.addrsStr[reputerIndexes[1]])
	require.NoError(err)
	reputer5_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId1, s.addrsStr[reputerIndexes[2]])
	require.NoError(err)

	// calculate rewards (topic 0)
	reputer0_Reward1 := reputer0_Stake2.Sub(reputer0_Stake1)
	reputer1_Reward1 := reputer1_Stake2.Sub(reputer1_Stake1)
	reputer2_Reward1 := reputer2_Stake2.Sub(reputer2_Stake1)
	// calculate rewards (topic 1)
	reputer3_Reward1 := reputer3_Stake2.Sub(reputer3_Stake1)
	reputer4_Reward1 := reputer4_Stake2.Sub(reputer4_Stake1)
	reputer5_Reward1 := reputer5_Stake2.Sub(reputer5_Stake1)

	// calculate total rewards for each topic
	topic0RewardTotal1 := reputer0_Reward1.Add(reputer1_Reward1).Add(reputer2_Reward1)
	topic1RewardTotal1 := reputer3_Reward1.Add(reputer4_Reward1).Add(reputer5_Reward1)

	topic0Total0Dec, err := alloraMath.NewDecFromSdkInt(topic0RewardTotal0)
	require.NoError(err)
	topic1Total0Dec, err := alloraMath.NewDecFromSdkInt(topic1RewardTotal0)
	require.NoError(err)
	topic0Total1Dec, err := alloraMath.NewDecFromSdkInt(topic0RewardTotal1)
	require.NoError(err)
	topic1Total1Dec, err := alloraMath.NewDecFromSdkInt(topic1RewardTotal1)
	require.NoError(err)

	// Normalize to the amount of rewards given in each cycle and check the ratios of the whole rewards,
	// instead of absolute values (since rewards amount is different in each cycle).
	totalRewardsEpoch0, err := topic0Total0Dec.Add(topic1Total0Dec)
	require.NoError(err)
	totalRewardsEpoch1, err := topic0Total1Dec.Add(topic1Total1Dec)
	require.NoError(err)

	topic0RewardTotal0Normalized, err := topic0Total0Dec.Quo(totalRewardsEpoch0)
	require.NoError(err)
	topic1RewardTotal0Normalized, err := topic1Total0Dec.Quo(totalRewardsEpoch0)
	require.NoError(err)
	topic0RewardTotal1Normalized, err := topic0Total1Dec.Quo(totalRewardsEpoch1)
	require.NoError(err)
	topic1RewardTotal1Normalized, err := topic1Total1Dec.Quo(totalRewardsEpoch1)

	// in the first round, the rewards should be equal for each topic
	require.True(topic0RewardTotal0Normalized.Equal(topic1RewardTotal0Normalized), "%s != %s", topic0RewardTotal0Normalized, topic1RewardTotal0Normalized)
	// for topic 0, the rewards should be less in the second round
	require.True(topic0RewardTotal0Normalized.Gte(topic1RewardTotal0Normalized), "%s <= %s", topic0RewardTotal0Normalized, topic1RewardTotal0Normalized)
	// in the second round, the rewards should be greater for topic 1
	require.True(topic0RewardTotal1Normalized.Lte(topic1RewardTotal1Normalized), "%s >= %s", topic0RewardTotal1Normalized, topic1RewardTotal1Normalized)
	// the rewards for topic 1 should be greater in the second round
	require.True(topic1RewardTotal0Normalized.Lt(topic1RewardTotal1Normalized), "%s >= %s", topic1RewardTotal0Normalized, topic1RewardTotal1Normalized)
}

func (s *RewardsTestSuite) TestReputerDeviatingFromConsensusGetsLessRewards() {
	testCases := []struct {
		name        string
		baseValue   string
		deltaValue  string // Value for a reputer that deviates from consensus
		description string
	}{
		{
			name:        "Reputer above consensus gets less rewards",
			baseValue:   "0.1",
			deltaValue:  "-0.8",
			description: "Higher than consensus value should result in lower rewards",
		},
		{
			name:        "Reputer below consensus gets less rewards",
			baseValue:   "0.9",
			deltaValue:  "0.8",
			description: "Lower than consensus value should result in lower rewards",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			require := s.Require()
			block := int64(1)
			epochLength := int64(5)
			alphaRegret := alloraMath.MustNewDecFromString("0.1")
			s.ctx = s.ctx.WithBlockHeight(block)

			s.SetParamsForTest()

			reputerIndexes := returnIndexes(0, 6)
			workerIndexes := returnIndexes(6, 3)
			reputerValues := s.getReputerValuesFromIndexes(reputerIndexes, workerIndexes, tc.baseValue)
			exceptionReputerIdx := 5
			reputerValues[exceptionReputerIdx][s.addrsStr[0]] = tc.deltaValue // used as delta
			workerValues := getWorkerValuesFromIndexes(workerIndexes, tc.baseValue)
			stake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())

			// PRE RUN
			topicId, block := s.FullTopicPass(
				block,
				workerIndexes,
				reputerIndexes,
				WithEpochLength(epochLength),
				WithAlphaRegret(alphaRegret),
				WithWorkerValues(workerValues),
				WithReputerValues(reputerValues),
				WithReputerStake(&stake),
			)

			var err error
			reputerStakes0 := make([]cosmosMath.Int, len(reputerIndexes))
			for i, index := range reputerIndexes {
				reputerStakes0[i], err = s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId, s.addrs[index].String())
				require.NoError(err)
			}

			err = s.emissionsKeeper.SetPreviousTopicWeight(s.ctx, topicId, alloraMath.MustNewDecFromString("100"))
			require.NoError(err)

			s.FullTopicPass(
				block,
				workerIndexes,
				reputerIndexes,
				WithEpochLength(epochLength),
				WithAlphaRegret(alphaRegret),
				WithWorkerValues(workerValues),
				WithReputerValues(reputerValues),
				WithReputerStake(&stake),
				WithTopicID(topicId),
			)

			s.GenerateRewards(topicId, block)
			s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))
			s.EndBlock()

			reputerStakes1 := make([]cosmosMath.Int, len(reputerIndexes))
			for i, index := range reputerIndexes {
				reputerStakes1[i], err = s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId, s.addrs[index].String())
				require.NoError(err)
			}

			reputerRewards := make([]cosmosMath.Int, len(reputerStakes0))
			for i := range reputerStakes1 {
				reputerRewards[i] = reputerStakes1[i].Sub(reputerStakes0[i])
			}

			for i := 0; i < len(reputerRewards)-2; i++ {
				require.False(reputerRewards[i].IsZero())
				require.True(reputerRewards[i].Equal(reputerRewards[i+1]))
			}

			require.Truef(reputerRewards[exceptionReputerIdx].LT(reputerRewards[0]), "%s < %s", reputerRewards[exceptionReputerIdx], reputerRewards[0])
		})
	}
}

func (s *RewardsTestSuite) TestRewardForRemainingParticipantsGoUpWhenParticipantDropsOut() {
	// SETUP
	require := s.Require()

	block := int64(1)
	s.ctx = s.ctx.WithBlockHeight(block)

	alphaRegret := alloraMath.MustNewDecFromString("0.1")

	s.SetParamsForTest()

	reputerIndexes := returnIndexes(0, 3)
	workerIndexes := returnIndexes(3, 3)

	stake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())

	topicId0 := s.setUpTopicWithEpochLength(block, workerIndexes, reputerIndexes, stake, alphaRegret, 30)

	// Define values to test
	reputer0Values := []TestWorkerValue{
		{Index: reputerIndexes[0], Value: "0.2"},
		{Index: reputerIndexes[1], Value: "0.2"},
		{Index: reputerIndexes[2], Value: "0.2"},
	}

	workerValues := []TestWorkerValue{
		{Index: workerIndexes[0], Value: "0.2"},
		{Index: workerIndexes[1], Value: "0.2"},
		{Index: workerIndexes[2], Value: "0.2"},
	}

	// Define second round values to test with one less reputer
	reputer1Values := []TestWorkerValue{
		{Index: reputerIndexes[0], Value: "0.2"},
		{Index: reputerIndexes[1], Value: "0.2"},
	}

	// record the stakes before rewards
	reputer0_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[0].String())
	require.NoError(err)
	reputer1_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[1].String())
	require.NoError(err)
	reputer2_Stake0, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[2].String())
	require.NoError(err)

	// do work on the current block
	s.getRewardsDistribution(
		topicId0,
		workerValues,
		reputer0Values,
		s.addrs[workerIndexes[0]],
		"0.1",
		"0.1",
	)

	// create tokens to reward with
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	nextBlock, _, err := s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, topicId0)
	s.Require().NoError(err)
	s.ctx = s.ctx.WithBlockHeight(nextBlock)
	// force rewards to be distributed
	err = s.emissionsKeeper.SetRewardCurrentBlockEmission(s.ctx, cosmosMath.NewInt(100))
	s.Require().NoError(err)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	require.NoError(err)

	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId0)
	s.Require().NoError(err)
	// record the updated stakes after rewards
	reputer0_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[0].String())
	require.NoError(err)
	reputer1_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[1].String())
	require.NoError(err)
	reputer2_Stake1, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[2].String())
	require.NoError(err)

	// calculate the rewards for each reputer
	reputer0_Reward0 := reputer0_Stake1.Sub(reputer0_Stake0)
	reputer1_Reward0 := reputer1_Stake1.Sub(reputer1_Stake0)
	reputer2_Reward0 := reputer2_Stake1.Sub(reputer2_Stake0)

	// fund the topic again for future rewards
	const topicFundAmount int64 = 1000

	fundTopic := func(topicId uint64, funderAddr sdk.AccAddress, amount int64) {
		s.MintTokensToAddress(funderAddr, cosmosMath.NewInt(amount))
		fundTopicMessage := types.FundTopicRequest{
			Sender:  funderAddr.String(),
			TopicId: topicId,
			Amount:  cosmosMath.NewInt(amount),
		}
		_, err = s.msgServer.FundTopic(s.ctx, &fundTopicMessage)
		require.NoError(err)
	}

	fundTopic(topicId0, s.addrs[0], topicFundAmount)

	// do work on the current block, but with one less reputer
	s.getRewardsDistribution(
		topic.Id,
		workerValues,
		reputer1Values,
		s.addrs[workerIndexes[0]],
		"0.1",
		"0.1",
	)

	// create tokens to reward with
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	nextBlock, _, err = s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, topicId0)
	s.Require().NoError(err)
	s.ctx = s.ctx.WithBlockHeight(nextBlock)
	// force rewards to be distributed
	err = s.emissionsKeeper.SetRewardCurrentBlockEmission(s.ctx, cosmosMath.NewInt(100))
	s.Require().NoError(err)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	require.NoError(err)

	// check the updated stakes after rewards
	reputer0_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[0].String())
	require.NoError(err)
	reputer1_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[1].String())
	require.NoError(err)
	reputer2_Stake2, err := s.emissionsKeeper.GetStakeReputerAuthority(s.ctx, topicId0, s.addrs[2].String())
	require.NoError(err)

	// calculate the rewards for each reputer
	reputer0_Reward1 := reputer0_Stake2.Sub(reputer0_Stake1)
	reputer1_Reward1 := reputer1_Stake2.Sub(reputer1_Stake1)
	reputer2_Reward1 := reputer2_Stake2.Sub(reputer2_Stake1)

	// sanity check that participating reputer rewards went up, but non participating reputer
	// rewards went to zero
	require.True(reputer0_Reward1.GT(reputer0_Reward0))
	require.True(reputer1_Reward1.GT(reputer1_Reward0))
	require.True(reputer2_Reward0.GT(cosmosMath.ZeroInt()))
	require.True(reputer2_Reward1.Equal(cosmosMath.ZeroInt()))
}

func (s *RewardsTestSuite) TestRewardIncreaseContiouslyAfterTopicReactivated() {
	// SETUP
	require := s.Require()

	block := int64(1)
	s.ctx = s.ctx.WithBlockHeight(block)

	alphaRegret := alloraMath.MustNewDecFromString("0.1")

	s.SetParamsForTest()

	reputer0Indexes := returnIndexes(0, 3)
	worker0Indexes := returnIndexes(3, 3)

	reputer1Indexes := returnIndexes(6, 3)
	worker1Indexes := returnIndexes(9, 3)

	stake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())

	topicId0 := s.setUpTopicWithEpochLength(block, worker0Indexes, reputer0Indexes, stake, alphaRegret, 30)
	topicId1 := s.setUpTopicWithEpochLength(block, worker1Indexes, reputer1Indexes, stake, alphaRegret, 60)
	cosmosOneE18 := inferencesynthesis.CosmosIntOneE18()
	// Add Stake for reputers
	var stakes = []cosmosMath.Int{
		cosmosMath.NewInt(1176644).Mul(cosmosOneE18),
		cosmosMath.NewInt(384623).Mul(cosmosOneE18),
		cosmosMath.NewInt(394676).Mul(cosmosOneE18),
		cosmosMath.NewInt(207999).Mul(cosmosOneE18),
		cosmosMath.NewInt(368582).Mul(cosmosOneE18),
	}
	for i, index := range reputer0Indexes {
		s.MintTokensToAddress(s.addrs[index], stakes[i])
		_, err := s.msgServer.AddStake(s.ctx, &types.AddStakeRequest{
			Sender:  s.addrs[index].String(),
			Amount:  stakes[i],
			TopicId: topicId0,
		})
		s.Require().NoError(err)
	}
	for i, index := range reputer1Indexes {
		s.MintTokensToAddress(s.addrs[index], stakes[i].MulRaw(2))
		_, err := s.msgServer.AddStake(s.ctx, &types.AddStakeRequest{
			Sender:  s.addrs[index].String(),
			Amount:  stakes[i].MulRaw(2),
			TopicId: topicId1,
		})
		s.Require().NoError(err)
	}

	initialStake := cosmosMath.NewInt(1000)
	s.MintTokensToAddress(s.addrs[reputer0Indexes[0]], initialStake)
	fundTopic0Message := types.FundTopicRequest{
		Sender:  s.addrs[reputer0Indexes[0]].String(),
		TopicId: topicId0,
		Amount:  initialStake,
	}
	_, err := s.msgServer.FundTopic(s.ctx, &fundTopic0Message)
	s.Require().NoError(err)

	s.MintTokensToAddress(s.addrs[reputer1Indexes[0]], initialStake.MulRaw(2))
	fundTopic1Message := types.FundTopicRequest{
		Sender:  s.addrs[reputer1Indexes[0]].String(),
		TopicId: topicId1,
		Amount:  initialStake.MulRaw(2),
	}
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopic1Message)
	s.Require().NoError(err)

	topic0, err := s.emissionsKeeper.GetTopic(s.ctx, topicId0)
	require.NoError(err)
	blocklHeight := topic0.EpochLength
	s.ctx = s.ctx.WithBlockHeight(blocklHeight)
	err = s.emissionsKeeper.SetRewardCurrentBlockEmission(s.ctx, cosmosMath.NewInt(100))
	s.Require().NoError(err)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)
	worker0Values := []TestWorkerValue{
		{Index: worker0Indexes[0], Value: "0.1"},
		{Index: worker0Indexes[1], Value: "0.2"},
		{Index: worker0Indexes[2], Value: "0.3"},
	}
	reputer0Values := []TestWorkerValue{
		{Index: reputer0Indexes[0], Value: "0.1"},
		{Index: reputer0Indexes[1], Value: "0.2"},
		{Index: reputer0Indexes[2], Value: "0.3"},
	}
	worker1Values := []TestWorkerValue{
		{Index: worker1Indexes[0], Value: "0.4"},
		{Index: worker1Indexes[1], Value: "0.5"},
		{Index: worker1Indexes[2], Value: "0.6"},
	}
	reputer1Values := []TestWorkerValue{
		{Index: reputer1Indexes[0], Value: "0.4"},
		{Index: reputer1Indexes[1], Value: "0.5"},
		{Index: reputer1Indexes[2], Value: "0.6"},
	}
	rewardsDistribution0_0 := s.getRewardsDistribution(
		topicId0,
		worker0Values,
		reputer0Values,
		s.addrs[worker0Indexes[0]],
		"0.1",
		"0.1",
	)

	_ = s.getRewardsDistribution(
		topicId1,
		worker1Values,
		reputer1Values,
		s.addrs[worker1Indexes[0]],
		"0.2",
		"0.2",
	)

	// Check if first topic is inactivated due to low weight
	isActivated, err := s.emissionsKeeper.IsTopicActive(s.ctx, topicId0)
	require.NoError(err)
	require.False(isActivated)

	// Activate first topic
	s.MintTokensToAddress(s.addrs[reputer0Indexes[0]], initialStake)
	fundTopic0Message = types.FundTopicRequest{
		Sender:  s.addrs[reputer0Indexes[0]].String(),
		TopicId: topicId0,
		Amount:  initialStake.MulRaw(3),
	}
	_, err = s.msgServer.FundTopic(s.ctx, &fundTopic0Message)
	s.Require().NoError(err)

	isActivated, err = s.emissionsKeeper.IsTopicActive(s.ctx, topicId0)
	require.NoError(err)
	require.True(isActivated)

	rewardsDistribution0_1 := s.getRewardsDistribution(
		topicId0,
		worker0Values,
		reputer0Values,
		s.addrs[worker0Indexes[0]],
		"0.1",
		"0.1",
	)
	require.Equal(len(rewardsDistribution0_1), len(rewardsDistribution0_0))
}

func (s *RewardsTestSuite) TestCalcTopicRewards() {
	testCases := []struct {
		name                string
		setupFunc           func() (map[uint64]*alloraMath.Dec, []uint64, alloraMath.Dec, alloraMath.Dec, map[uint64]int64, alloraMath.Dec)
		expectedRewardsFunc func(map[uint64]*alloraMath.Dec) bool
		expectedError       error
	}{
		{
			name: "Happy path - multiple topics",
			setupFunc: func() (map[uint64]*alloraMath.Dec,
				[]uint64,
				alloraMath.Dec,
				alloraMath.Dec,
				map[uint64]int64,
				alloraMath.Dec) {
				weights := map[uint64]*alloraMath.Dec{
					1: decPtr("0.5"),
					2: decPtr("0.3"),
					3: decPtr("0.2"),
				}
				sortedTopics := []uint64{1, 2, 3}
				sumWeight := alloraMath.MustNewDecFromString("1.0")
				totalReward := alloraMath.MustNewDecFromString("1000.0")
				epochLengths := map[uint64]int64{
					1: 100,
					2: 100,
					3: 100,
				}
				currentBlockEmission := alloraMath.MustNewDecFromString("10.0")
				return weights, sortedTopics, sumWeight, totalReward, epochLengths, currentBlockEmission
			},
			expectedRewardsFunc: func(rewards map[uint64]*alloraMath.Dec) bool {
				expected := map[uint64]*alloraMath.Dec{
					1: decPtr("500.0"),
					2: decPtr("300.0"),
					3: decPtr("200.0"),
				}
				return s.compareRewards(rewards, expected)
			},
			expectedError: nil,
		},
		{
			name: "Single topic",
			setupFunc: func() (map[uint64]*alloraMath.Dec, []uint64, alloraMath.Dec, alloraMath.Dec, map[uint64]int64, alloraMath.Dec) {
				weights := map[uint64]*alloraMath.Dec{
					1: decPtr("1.0"),
				}
				sortedTopics := []uint64{1}
				sumWeight := alloraMath.MustNewDecFromString("1.0")
				totalReward := alloraMath.MustNewDecFromString("1000.0")
				epochLengths := map[uint64]int64{
					1: 100,
				}
				currentBlockEmission := alloraMath.MustNewDecFromString("10.0")
				return weights, sortedTopics, sumWeight, totalReward, epochLengths, currentBlockEmission
			},
			expectedRewardsFunc: func(rewards map[uint64]*alloraMath.Dec) bool {
				expected := map[uint64]*alloraMath.Dec{
					1: decPtr("1000.0"),
				}
				return s.compareRewards(rewards, expected)
			},
			expectedError: nil,
		},
		{
			name: "Zero total reward",
			setupFunc: func() (map[uint64]*alloraMath.Dec, []uint64, alloraMath.Dec, alloraMath.Dec, map[uint64]int64, alloraMath.Dec) {
				weights := map[uint64]*alloraMath.Dec{
					1: decPtr("0.5"),
					2: decPtr("0.5"),
				}
				sortedTopics := []uint64{1, 2}
				sumWeight := alloraMath.MustNewDecFromString("1.0")
				totalReward := alloraMath.ZeroDec()
				epochLengths := map[uint64]int64{
					1: 100,
					2: 100,
				}
				currentBlockEmission := alloraMath.MustNewDecFromString("10.0")
				return weights, sortedTopics, sumWeight, totalReward, epochLengths, currentBlockEmission
			},
			expectedRewardsFunc: func(rewards map[uint64]*alloraMath.Dec) bool {
				expected := map[uint64]*alloraMath.Dec{
					1: decPtr("0"),
					2: decPtr("0"),
				}
				return s.compareRewards(rewards, expected)
			},
			expectedError: nil,
		},
		{
			name: "Different epoch lengths",
			setupFunc: func() (map[uint64]*alloraMath.Dec, []uint64, alloraMath.Dec, alloraMath.Dec, map[uint64]int64, alloraMath.Dec) {
				weights := map[uint64]*alloraMath.Dec{
					1: decPtr("0.75"),
					2: decPtr("0.25"),
				}
				sortedTopics := []uint64{1, 2}
				sumWeight := alloraMath.MustNewDecFromString("1")
				totalReward := alloraMath.MustNewDecFromString("1000.0")
				epochLengths := map[uint64]int64{
					1: 100,
					2: 200,
				}
				currentBlockEmission := alloraMath.MustNewDecFromString("1.0")
				return weights, sortedTopics, sumWeight, totalReward, epochLengths, currentBlockEmission
			},
			expectedRewardsFunc: func(rewards map[uint64]*alloraMath.Dec) bool {
				expected := map[uint64]*alloraMath.Dec{
					1: decPtr("75"),
					2: decPtr("50"),
				}
				return s.compareRewards(rewards, expected)
			},
			expectedError: nil,
		},
		{
			name: "Very small weights",
			setupFunc: func() (map[uint64]*alloraMath.Dec, []uint64, alloraMath.Dec, alloraMath.Dec, map[uint64]int64, alloraMath.Dec) {
				weights := map[uint64]*alloraMath.Dec{
					1: decPtr("0.000000000000000001"),
					2: decPtr("0.000000000000000002"),
				}
				sortedTopics := []uint64{1, 2}
				sumWeight := alloraMath.MustNewDecFromString("0.000000000000000003")
				totalReward := alloraMath.MustNewDecFromString("1000.0")
				epochLengths := map[uint64]int64{
					1: 100,
					2: 100,
				}
				currentBlockEmission := alloraMath.MustNewDecFromString("10.0")
				return weights, sortedTopics, sumWeight, totalReward, epochLengths, currentBlockEmission
			},
			expectedRewardsFunc: func(rewards map[uint64]*alloraMath.Dec) bool {
				expected := map[uint64]*alloraMath.Dec{
					1: decPtr("333.333333333333333333"),
					2: decPtr("666.666666666666666667"),
				}
				return s.compareRewards(rewards, expected)
			},
			expectedError: nil,
		},
		{
			name: "Mismatched weights and sorted topics",
			setupFunc: func() (map[uint64]*alloraMath.Dec, []uint64, alloraMath.Dec, alloraMath.Dec, map[uint64]int64, alloraMath.Dec) {
				weights := map[uint64]*alloraMath.Dec{
					1: decPtr("0.5"),
					2: decPtr("0.5"),
				}
				sortedTopics := []uint64{1, 2, 3}
				sumWeight := alloraMath.MustNewDecFromString("1.0")
				totalReward := alloraMath.MustNewDecFromString("1000.0")
				epochLengths := map[uint64]int64{
					1: 100,
					2: 100,
				}
				currentBlockEmission := alloraMath.MustNewDecFromString("10.0")
				return weights, sortedTopics, sumWeight, totalReward, epochLengths, currentBlockEmission
			},
			expectedRewardsFunc: func(rewards map[uint64]*alloraMath.Dec) bool {
				return len(rewards) == 2
			},
			expectedError: types.ErrInvalidValue,
		},
		{
			name: "Treasury lower than rewards",
			setupFunc: func() (map[uint64]*alloraMath.Dec, []uint64, alloraMath.Dec, alloraMath.Dec, map[uint64]int64, alloraMath.Dec) {
				weights := map[uint64]*alloraMath.Dec{
					1: decPtr("0.5"),
					2: decPtr("0.3"),
					3: decPtr("0.2"),
				}
				sortedTopics := []uint64{1, 2, 3}
				sumWeight := alloraMath.MustNewDecFromString("1.0")
				totalReward := alloraMath.MustNewDecFromString("50.0") // Lower than expected rewards
				epochLengths := map[uint64]int64{
					1: 100,
					2: 100,
					3: 100,
				}
				currentBlockEmission := alloraMath.MustNewDecFromString("1.0")
				return weights, sortedTopics, sumWeight, totalReward, epochLengths, currentBlockEmission
			},
			expectedRewardsFunc: func(rewards map[uint64]*alloraMath.Dec) bool {
				expected := map[uint64]*alloraMath.Dec{
					1: decPtr("25.0"),
					2: decPtr("15.0"),
					3: decPtr("10.0"),
				}
				return s.compareRewards(rewards, expected)
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			weights, sortedTopics, sumWeight, totalReward, epochLengths, currentBlockEmission := tc.setupFunc()
			args := rewards.CalcTopicRewardsArgs{
				Ctx:                             s.ctx,
				Weights:                         weights,
				SortedTopics:                    sortedTopics,
				SumTopicWeights:                 sumWeight,
				TotalAvailableInRewardsTreasury: totalReward,
				EpochLengths:                    epochLengths,
				CurrentRewardsEmissionPerBlock:  currentBlockEmission,
			}
			rewards, err := rewards.CalcTopicRewards(args)

			if tc.expectedError != nil {
				s.Require().ErrorIs(err, tc.expectedError)
			} else {
				s.Require().NoError(err)
				s.Require().True(tc.expectedRewardsFunc(rewards))
			}
		})
	}
}

func (s *RewardsTestSuite) TestMonthlyPercentageRewardCalculation() {
	// 1. Setup Params
	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)
	blocksPerMonth := int64(10)
	params.BlocksPerMonth = uint64(blocksPerMonth)
	err = s.emissionsKeeper.SetParams(s.ctx, params)
	s.Require().NoError(err)

	// 2. Fund Rewards Module (required for EndBlocker checks)
	initialRewardAmount := cosmosMath.NewInt(1000000)
	s.MintTokensToModule(types.AlloraRewardsAccountName, initialRewardAmount)

	// 3. Define simulated reward increments per block
	reputerIncrement := cosmosMath.NewInt(10)
	topicIncrement := cosmosMath.NewInt(50)

	// 4. Simulate Block Progression and Reward Accumulation
	for i := int64(1); i < blocksPerMonth; i++ {
		loopCtx := s.ctx.WithBlockHeight(i)
		// Run EndBlocker (simulates standard block processing)
		err = module.EndBlocker(loopCtx, s.emissionsAppModule)
		s.Require().NoError(err, "EndBlocker failed during block %d", i)

		// Manually add rewards to simulate accumulation during the month
		err = s.emissionsKeeper.AddMonthlyRewards(loopCtx, reputerIncrement, topicIncrement)
		s.Require().NoError(err, "Failed to add rewards at block %d", i)
	}

	// 5. Calculate Expected Totals and Percentage *before* the reset block
	totalReputerRewards := reputerIncrement.MulRaw(blocksPerMonth - 1)
	totalTopicRewards := topicIncrement.MulRaw(blocksPerMonth - 1)

	expectedPercentage := alloraMath.ZeroDec()
	if !totalTopicRewards.IsZero() {
		reputersDec, err := alloraMath.NewDecFromSdkInt(totalReputerRewards)
		s.Require().NoError(err)
		topicDec, err := alloraMath.NewDecFromSdkInt(totalTopicRewards)
		s.Require().NoError(err)
		expectedPercentage, err = reputersDec.Quo(topicDec)
		s.Require().NoError(err)
	}

	// Sanity check the accumulated values before the final EndBlocker call
	reputerRewardsBeforeFinal, err := s.emissionsKeeper.GetMonthlyReputerRewards(s.ctx)
	s.Require().NoError(err)
	s.Require().True(totalReputerRewards.Equal(reputerRewardsBeforeFinal), "Mismatch in accumulated reputer rewards before reset")
	topicRewardsBeforeFinal, err := s.emissionsKeeper.GetMonthlyTopicRewards(s.ctx)
	s.Require().NoError(err)
	s.Require().True(totalTopicRewards.Equal(topicRewardsBeforeFinal), "Mismatch in accumulated topic rewards before reset")

	// 6. Trigger End Blocker execution at the end of the month
	endOfMonthCtx := s.ctx.WithBlockHeight(blocksPerMonth)
	err = module.EndBlocker(endOfMonthCtx, s.emissionsAppModule)
	s.Require().NoError(err, "EndBlocker execution failed at month boundary %d", blocksPerMonth)

	// 7. Verify State After End Blocker
	// Verify percentage calculated and stored by EndBlocker
	actualPercentageAfter, err := s.emissionsKeeper.GetPreviousPercentageRewardToStakedReputers(s.ctx)
	s.Require().NoError(err)
	s.T().Logf("Expected percentage %s, got %s", expectedPercentage.String(), actualPercentageAfter.String())
	s.Require().True(expectedPercentage.Equal(actualPercentageAfter), "Expected percentage %s, got %s", expectedPercentage.String(), actualPercentageAfter.String())

	// Verify counters reset by EndBlocker
	reputerRewardsAfter, err := s.emissionsKeeper.GetMonthlyReputerRewards(s.ctx)
	s.Require().NoError(err)
	s.Require().True(reputerRewardsAfter.IsZero(), "Monthly reputer rewards not reset by EndBlocker")

	topicRewardsAfter, err := s.emissionsKeeper.GetMonthlyTopicRewards(s.ctx)
	s.Require().NoError(err)
	s.Require().True(topicRewardsAfter.IsZero(), "Monthly topic rewards not reset by EndBlocker")
}

func (s *RewardsTestSuite) TestMonthlyPercentageRewardCalculation_ZeroTopicRewards() {
	// 1. Setup Params
	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)
	blocksPerMonth := int64(10)
	params.BlocksPerMonth = uint64(blocksPerMonth)
	err = s.emissionsKeeper.SetParams(s.ctx, params)
	s.Require().NoError(err)

	// 2. Fund Rewards Module (required for EndBlocker checks)
	initialRewardAmount := cosmosMath.NewInt(1000000)
	s.MintTokensToModule(types.AlloraRewardsAccountName, initialRewardAmount)

	// 3. Ensure No Topics or Reward Activity
	// No topics created, no workers/reputers registered, no data submitted.
	// This ensures EmitRewards calculates zero rewards throughout the month.

	// 4. Simulate Block Progression up to the Monthly Boundary
	for i := int64(1); i < blocksPerMonth; i++ {
		loopCtx := s.ctx.WithBlockHeight(i)
		err = module.EndBlocker(loopCtx, s.emissionsAppModule)
		s.Require().NoError(err, "EndBlocker failed during block %d", i)

		// Sanity check: Verify topic rewards remain zero during the month
		topicRewards, err := s.emissionsKeeper.GetMonthlyTopicRewards(loopCtx)
		s.Require().NoError(err)
		s.Require().True(topicRewards.IsZero(), "Topic rewards became non-zero before month end at block %d", i)
	}

	// 5. Trigger End Blocker execution at the end of the month
	endOfMonthCtx := s.ctx.WithBlockHeight(blocksPerMonth)
	err = module.EndBlocker(endOfMonthCtx, s.emissionsAppModule)
	s.Require().NoError(err, "EndBlocker execution failed at month boundary %d", blocksPerMonth)

	// 6. Verify State After End Blocker
	// Verify percentage is zero due to zero topic rewards
	expectedPercentage := alloraMath.ZeroDec()
	actualPercentageAfter, err := s.emissionsKeeper.GetPreviousPercentageRewardToStakedReputers(s.ctx)
	s.Require().NoError(err)
	s.Require().True(expectedPercentage.Equal(actualPercentageAfter), "Expected percentage %s, got %s", expectedPercentage.String(), actualPercentageAfter.String())

	// Verify counters reset by EndBlocker
	reputerRewardsAfter, err := s.emissionsKeeper.GetMonthlyReputerRewards(s.ctx)
	s.Require().NoError(err)
	s.Require().True(reputerRewardsAfter.IsZero(), "Monthly reputer rewards not reset by EndBlocker")

	topicRewardsAfter, err := s.emissionsKeeper.GetMonthlyTopicRewards(s.ctx)
	s.Require().NoError(err)
	s.Require().True(topicRewardsAfter.IsZero(), "Monthly topic rewards not reset by EndBlocker")
}

func (s *RewardsTestSuite) TestNoActiveParticipantsNoRewardsForTopic() {
	require := s.Require()

	block := int64(1)
	s.ctx = s.ctx.WithBlockHeight(block)

	s.SetParamsForTest()

	creatorIndex := 40
	reputerIndex := 41

	topicId := s.SetupTopic(s.addrs[creatorIndex])
	s.SetupParticipants(topicId, []int{reputerIndex}, true)

	// Record initial balance of the rewards module account
	rewardsModuleAddr := s.accountKeeper.GetModuleAddress(types.AlloraRewardsAccountName)
	initialRewardsBalance := s.bankKeeper.GetBalance(s.ctx, rewardsModuleAddr, params.DefaultBondDenom)

	// Record initial balance of the ecosystem account
	ecosystemAddr := s.accountKeeper.GetModuleAddress(minttypes.EcosystemModuleName)
	initialEcosystemBalance := s.bankKeeper.GetBalance(s.ctx, ecosystemAddr, params.DefaultBondDenom)

	// Simulate moving to the end of the first epoch
	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	require.NoError(err)
	nextBlock := block + topic.EpochLength
	s.ctx = s.ctx.WithBlockHeight(nextBlock)

	// Trigger EndBlocker to process potential rewards
	s.EndBlock()

	// Check topic weight after funding and epoch end
	topicWeight, _, err := s.emissionsKeeper.GetPreviousTopicWeight(s.ctx, topicId)
	require.NoError(err)
	require.True(topicWeight.Gt(alloraMath.ZeroDec()), "Topic weight should be greater than zero after funding")

	// Verify no rewards were distributed for this topic (moved to ecosystem account)
	finalRewardsBalance := s.bankKeeper.GetBalance(s.ctx, rewardsModuleAddr, params.DefaultBondDenom)
	require.True(initialRewardsBalance.Amount.GT(finalRewardsBalance.Amount),
		"Rewards module - topic %d: Initial: %s, Final: %s",
		topicId, initialRewardsBalance.String(), finalRewardsBalance.String())

	// Verify if rewards were moved to the ecosystem account
	ecosystemBalance := s.bankKeeper.GetBalance(s.ctx, ecosystemAddr, params.DefaultBondDenom)
	require.True(ecosystemBalance.Amount.GT(initialEcosystemBalance.Amount),
		"Ecosystem account - topic %d: Initial: %s, Final: %s",
		topicId, initialEcosystemBalance.String(), ecosystemBalance.String())
}
