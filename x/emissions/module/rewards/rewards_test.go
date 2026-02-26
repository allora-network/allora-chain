package rewards_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	cosmosMath "cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type RewardsTestSuite struct {
	testutil.TestSuite
}

func TestRewardsTestSuite(t *testing.T) {
	suite.Run(t, &RewardsTestSuite{
		testutil.NewTestSuite("rewards"),
	})
}

func (s *RewardsTestSuite) GenerateRewards(topicId uint64, block int64) ([]types.TaskReward, alloraMath.Dec) {
	topicTotalRewards := alloraMath.NewDecFromInt64(1000000)
	p, err := s.ParamsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)

	rewardsDistribution, totalReputerReward, err := rewards.GenerateRewardsDistributionByTopicParticipant(
		rewards.GenerateRewardsDistributionByTopicParticipantArgs{
			Ctx:          s.Ctx(),
			K:            *s.EmissionsKeeper(),
			TopicId:      topicId,
			TopicReward:  &topicTotalRewards,
			BlockHeight:  block,
			ModuleParams: p,
		})
	s.Require().NoError(err)
	return rewardsDistribution, totalReputerReward
}

func (s *RewardsTestSuite) TestStandardRewardEmissionShouldRewardTopicsWithFulfilledNonces() {
	s.SetParamsForTest()

	reputerIndexes := testutil.ReturnIndexes(0, 5)
	workerIndexes := testutil.ReturnIndexes(5, 5)
	reputerIndexes2 := testutil.ReturnIndexes(10, 5)
	workerIndexes2 := testutil.ReturnIndexes(15, 5)

	// TOPIC 1 - PRE PASS
	topicId, newBlockheight := s.FullTopicPass(workerIndexes, reputerIndexes)

	// TOPIC 2 - SETUP
	// Do not send bundles for topic 2 yet
	topic2 := s.FullTopicSetup(workerIndexes2, reputerIndexes2)

	beforeRewardsTopic1FeeRevenue, err := s.TopicKeeper().GetTopicFeeRevenue(s.Ctx(), topicId)
	s.Require().NoError(err)

	beforeRewardsTopic2FeeRevenue, err := s.TopicKeeper().GetTopicFeeRevenue(s.Ctx(), topic2.GetId())
	s.Require().NoError(err)

	// TOPIC 1 - FIRST PASS
	s.FullTopicPass(workerIndexes, reputerIndexes, testutil.WithTopicID(topicId))

	// mint some rewards to give out
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

	newBlockheight += topic2.EpochLength + topic2.GroundTruthLag
	s.WithBlockHeight(newBlockheight)

	// Trigger end block - rewards distribution
	s.EndBlock()

	afterRewardsTopic1FeeRevenue, err := s.TopicKeeper().GetTopicFeeRevenue(s.Ctx(), topicId)
	s.Require().NoError(err)
	afterRewardsTopic2FeeRevenue, err := s.TopicKeeper().GetTopicFeeRevenue(s.Ctx(), topic2.GetId())
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

// We have 2 trials with 2 epochs each, and the first worker does better in the 2nd epoch in both trials.
// We show that keeping the TaskRewardAlpha the same means that the worker is rewarded the same amount
// in both cases.
// This is a sanity test to ensure that we are isolating the effect of TaskRewardAlpha in subsequent tests.
func (s *RewardsTestSuite) TestFixingTaskRewardAlphaDoesNotChangePerformanceImportanceOfPastVsPresent() {
	require := s.Require()

	currentParams, err := s.ParamsKeeper().GetParams(s.Ctx())
	require.NoError(err)
	currentParams.TaskRewardAlpha = alloraMath.MustNewDecFromString("0.1")
	err = s.ParamsKeeper().SetParams(s.Ctx(), currentParams)
	require.NoError(err)

	workerIndexes := testutil.ReturnIndexes(0, 3)
	reputerIndexes := testutil.ReturnIndexes(3, 3)
	values := []string{"0.1", "0.2", "0.3"}
	workerValues := testutil.GetWorkerValuesFromIndexes(workerIndexes, values...)
	reputerValues := s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, values...)
	stake := cosmosMath.NewInt(1000000000000000000).Mul(inferencesynthesis.CosmosIntOneE18())
	alphaRegret := alloraMath.MustNewDecFromString("0.1")

	topicPassOpts := []testutil.Option{
		testutil.WithReputerStake(&stake),
		testutil.WithReputerValues(reputerValues),
		testutil.WithWorkerValues(workerValues),
		testutil.WithAlphaRegret(alphaRegret),
	}

	var (
		numTopics  = 2
		allRewards = make([][][]types.TaskReward, numTopics)
	)

	for topicNum := 0; topicNum < numTopics; topicNum++ {
		allRewards[topicNum] = make([][]types.TaskReward, 2)

		// pre-emptive run for topic setup
		topicId, _ := s.FullTopicPass(workerIndexes, reputerIndexes, topicPassOpts...)

		// for both parts of the test (A and B)
		for part := 0; part < 2; part++ {
			// run a full topic pass
			opts := append(topicPassOpts, testutil.WithTopicID(topicId))
			_, blockHeight := s.FullTopicPass(workerIndexes, reputerIndexes, opts...)

			// generate rewards
			allRewards[topicNum][part], _ = s.GenerateRewards(topicId, blockHeight)
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
// Note this test contains noise in the second phase, as in the second run there is a competing topic already created.
// For absolutely equal conditions, it should be two different clean chains with one topic only and different alpha.
// However this test is enough to prove the point
func (s *RewardsTestSuite) TestIncreasingTaskRewardAlphaIncreasesImportanceOfPresentPerformance() {
	require := s.Require()
	alphaRegret := alloraMath.MustNewDecFromString("0.1")

	workerIndexes := testutil.ReturnIndexes(0, 3)
	reputerIndexes := testutil.ReturnIndexes(3, 3)
	stake := cosmosMath.NewInt(1000000000000000000).Mul(inferencesynthesis.CosmosIntOneE18())
	workerValues := testutil.GetWorkerValuesFromIndexes(workerIndexes, "0.1", "0.2", "0.3")
	// define the different reputer performance values for each test phase
	normalPerformance := s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.1", "0.2", "0.3")
	improvedPerformance := s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.1", "0.01", "0.01") // first worker performs better
	alphaValues := []string{"0.1", "0.2"}

	var rewardsDistributions [2][2][]types.TaskReward // [alphaIndex][testPhase]

	// run tests for each alpha value
	for alphaIndex, alpha := range alphaValues {
		// set the task reward alpha for this test series
		currentParams, err := s.ParamsKeeper().GetParams(s.Ctx())
		require.NoError(err)
		currentParams.TaskRewardAlpha = alloraMath.MustNewDecFromString(alpha)
		err = s.ParamsKeeper().SetParams(s.Ctx(), currentParams)
		require.NoError(err)

		baseOpts := []testutil.Option{
			testutil.WithReputerStake(&stake),
			testutil.WithAlphaRegret(alphaRegret),
			testutil.WithWorkerValues(workerValues),
		}

		// pre-emptive run to create the topic
		topicId, _ := s.FullTopicPass(workerIndexes, reputerIndexes, baseOpts...)

		// for each test phase (normal performance and improved performance)
		for phase := 0; phase < 2; phase++ {
			// set the reputer values based on the phase
			reputerPerformance := normalPerformance
			if phase == 1 {
				reputerPerformance = improvedPerformance
			}

			// run a full topic pass
			opts := append(baseOpts,
				testutil.WithTopicID(topicId),
				testutil.WithReputerValues(reputerPerformance),
			)

			_, blockHeight := s.FullTopicPass(workerIndexes, reputerIndexes, opts...)

			// generate and store rewards for comparison
			rewardsDistributions[alphaIndex][phase], _ = s.GenerateRewards(topicId, blockHeight)
		}
	}

	// ASSERTIONS

	// Check that phase 0 rewards are equal between alpha values
	// TODO: is this necessary?
	// require.True(areTaskRewardsEqualIgnoringTopicId(s, rewardsDistributions[0][0], rewardsDistributions[1][0]),
	//	"Rewards in the first phase should be equal between alphas")

	// Check that phase 1 rewards are NOT equal between alpha values
	require.False(areTaskRewardsEqualIgnoringTopicId(s, rewardsDistributions[0][1], rewardsDistributions[1][1]),
		"Rewards in the second phase should differ between alphas")

	// Print the rewards distribution table
	s.printRewardsDistributionTable(rewardsDistributions)
	// Extract worker[0] rewards for comparing phase 1 improvements
	var workerReward_0_1_Reward, workerReward_1_1_Reward alloraMath.Dec

	// Get worker[0] reward for alpha=0.1, phase 1
	found := false
	for _, reward := range rewardsDistributions[0][1] {
		if reward.Address == s.AddrsStr(workerIndexes[0]) &&
			reward.Type == types.WorkerInferenceRewardType {
			found = true
			workerReward_0_1_Reward = reward.Reward
			break
		}
	}
	require.True(found, "Worker[0] reward not found for alpha=0.1 in phase 1")

	// Get worker[0] reward for alpha=0.2, phase 1
	found = false
	for _, reward := range rewardsDistributions[1][1] {
		if reward.Address == s.AddrsStr(workerIndexes[0]) &&
			reward.Type == types.WorkerInferenceRewardType {
			found = true
			workerReward_1_1_Reward = reward.Reward
			break
		}
	}
	require.True(found, "Worker[1] reward not found for alpha=0.2 in phase 1")

	// Check that worker[0] gets higher reward with higher alpha
	require.True(workerReward_0_1_Reward.Lt(workerReward_1_1_Reward),
		"Worker[1] reward should be higher with alpha=0.2 (%s) than with alpha=0.1 (%s)",
		workerReward_1_1_Reward.String(), workerReward_0_1_Reward.String())
}

// We have 2 trials with 2 epochs each, and the reputer does worse in 2nd epoch in both trials,
// enacted by their increasing loss between epochs.
// We increase alpha between the trials to prove that their worsening performance decreases regret.
// This is somewhat counterintuitive, but can be explained by the following passage from the whitepaper:
// "A positive regret implies that the inference of worker j is expected by worker k to outperform
// the network's previously reported accuracy, whereas a negative regret indicates that the network
// is expected to be more accurate."
func (s *RewardsTestSuite) TestIncreasingAlphaRegretIncreasesPresentEffectOnRegret() {
	// SETUP
	require := s.Require()

	// Initialize indices and values
	workerIndexes := testutil.ReturnIndexes(0, 3)
	reputerIndexes := testutil.ReturnIndexes(3, 3)

	// Common test parameters
	stake := cosmosMath.NewInt(1000000000000000000).Mul(cosmosMath.NewInt(1000000000000000000))
	epochLength := int64(60)
	groundTruthLag := int64(60)
	deltaRegret := alloraMath.MustNewDecFromString("0.000000000001")

	// Alpha values for the two trials
	alphaValues := []string{"0.1", "0.2"}

	addrs := []string{
		s.AddrsStr(0),
		s.AddrsStr(1),
		s.AddrsStr(2),
	}
	slices.Sort(addrs)

	// Create constant reputer values across all phases
	phase1ReputerValues := s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.1", "0.2", "0.3")
	phase2ReputerValues := s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.2", "0.2", "0.2")
	phase2ReputerValues[0].CombinedValue = "0.1"
	phase2ReputerValues[1].CombinedValue = "0.2"
	phase2ReputerValues[2].CombinedValue = "0.3"

	// Worker values
	workerValues := testutil.GetWorkerValuesFromIndexes(workerIndexes, "0.1", "0.2", "0.3")

	// Common options for FullTopicPass
	baseOptions := func(alphaRegret alloraMath.Dec, topicId uint64) []testutil.Option {
		return []testutil.Option{
			testutil.WithEpochLength(epochLength),
			testutil.WithGroundTruthLag(groundTruthLag),
			testutil.WithTopicID(topicId),
			testutil.WithReputerStake(&stake),
			testutil.WithAlphaRegret(alphaRegret),
			testutil.WithWorkerValues(workerValues),
		}
	}

	// Define a trial structure to store results
	type Trial struct {
		alphaRegret   alloraMath.Dec
		topicId       uint64
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
		trial.workerRegrets = make(map[int]alloraMath.Dec)

		// Set task reward alpha in params for second trial
		if i == 1 {
			currentParams, err := s.ParamsKeeper().GetParams(s.Ctx())
			require.NoError(err)
			currentParams.TaskRewardAlpha = trial.alphaRegret
			err = s.ParamsKeeper().SetParams(s.Ctx(), currentParams)
			require.NoError(err)
		}

		// Prepare options for this trial
		options := baseOptions(trial.alphaRegret, trial.topicId)

		// PHASE 0: Pre-emptive run to set up the topic
		// Set up topic and do initial run with first phase worker values
		phaseOptions := append(options, testutil.WithReputerValues(phase1ReputerValues))
		trial.topicId, _ = s.FullTopicPass(workerIndexes, reputerIndexes, phaseOptions...)

		// PHASE 1: First real run with initial worker values
		phaseOptions = append(baseOptions(trial.alphaRegret, trial.topicId), testutil.WithReputerValues(phase1ReputerValues))
		s.FullTopicPass(workerIndexes, reputerIndexes, phaseOptions...)

		// Get first epoch regret for worker 0
		worker0, _, err := s.RegretsKeeper().GetInfererNetworkRegret(s.Ctx(), trial.topicId, addrs[0])
		require.NoError(err)
		trial.firstRegret = worker0.Value

		// PHASE 2: Second run with worse worker values
		phaseOptions = append(baseOptions(trial.alphaRegret, trial.topicId), testutil.WithReputerValues(phase2ReputerValues))
		s.FullTopicPass(workerIndexes, reputerIndexes, phaseOptions...)

		// Collect regrets for all workers
		for j, idx := range workerIndexes {
			worker, _, err := s.RegretsKeeper().GetInfererNetworkRegret(s.Ctx(), trial.topicId, addrs[idx])
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

		delta, err := alloraMath.InDelta(
			trial.decreaseRate,
			trial.alphaRegret,
			deltaRegret)
		require.NoError(err)
		require.True(delta, "Regret decrease rate should equal alpha (within a delta): got %s, expected %s",
			trial.decreaseRate.String(), trial.alphaRegret.String())
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
	// TOPIC 1 - PRE PASS
	workerIndexes := testutil.ReturnIndexes(5, 5)
	reputerIndexes := testutil.ReturnIndexes(0, 3)
	topicId, _ := s.FullTopicPass(workerIndexes, reputerIndexes)

	// TOPIC 1 - FIRST PASS
	_, block := s.FullTopicPass(workerIndexes, reputerIndexes, testutil.WithTopicID(topicId))

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

	// TOPIC 2 - PRE PASS
	workerIndexes2 := testutil.ReturnIndexes(5, 5)
	reputerIndexes2 := testutil.ReturnIndexes(0, 5)
	topicId2, _ := s.FullTopicPass(workerIndexes2, reputerIndexes2)

	// TOPIC 2 - FIRST PASS
	_, block = s.FullTopicPass(workerIndexes2, reputerIndexes2, testutil.WithTopicID(topicId2))

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
	s.WithBlockHeight(block)

	s.SetParamsForTest()

	// Create topic with shorter epoch length for testing multiple epochs
	epochLength := int64(5)
	groundTruthLag := int64(5)
	workerIndexes := testutil.ReturnIndexes(0, 3)
	reputerIndexes := testutil.ReturnIndexes(3, 3)
	topicId, _ := s.FullTopicPass(
		workerIndexes,
		reputerIndexes,
		testutil.WithBlock(block),
		testutil.WithAlphaRegret(alloraMath.MustNewDecFromString("0.1")),
		testutil.WithEpochLength(epochLength),
		testutil.WithGroundTruthLag(groundTruthLag),
	)

	// Track weights and regret scale over epochs
	var (
		workerWeights = make(map[string][]alloraMath.Dec)
		regretScales  []alloraMath.Dec
	)

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	const numEpochs = 10
	// Run multiple epochs
	for range numEpochs {
		baseOptions := []testutil.Option{
			testutil.WithAlphaRegret(alloraMath.MustNewDecFromString("0.1")),
			testutil.WithBlock(block + epochLength + topic.GroundTruthLag),
			testutil.WithTopicID(topicId),
		}

		topicId, block = s.FullTopicPass(workerIndexes, reputerIndexes, baseOptions...)

		// Get current weight and regret scale before processing
		regretScale, err := s.WeightstsKeeper().GetLatestRegretScale(s.Ctx(), topicId)
		require.NoError(err)
		regretScales = append(regretScales, regretScale)

		// Mint rewards
		s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

		for _, index := range workerIndexes {
			weight, err := s.WeightstsKeeper().GetLatestInfererWeight(s.Ctx(), topicId, s.AddrsStr(index))
			s.Require().NoError(err)
			workerWeights[s.AddrsStr(index)] = append(workerWeights[s.AddrsStr(index)], weight)
		}
	}

	// Verify weight evolution
	require.Len(workerWeights, 3) // one entry per worker
	for i := 0; i < len(workerWeights); i++ {
		// Weight for the same worker should change between epochs
		currentWorker := s.AddrsStr(i)
		currentWorkerWeights := workerWeights[currentWorker]
		// check weights are different from diff actors
		for j := 1; j < len(currentWorkerWeights); j++ {
			require.NotEqual(
				currentWorkerWeights[j].String(),
				currentWorkerWeights[j-1].String(),
				"Weight should change between epochs %d and %d", j-1, j,
			)
		}
		s.T().Logf("Worker %d , %s Weight: %v", i, s.AddrsStr(i), currentWorkerWeights)
	}

	// Verify regret scale evolution
	require.Len(regretScales, numEpochs)
	for i := 1; i < len(regretScales); i++ {
		// Regret scale should adapt based on predictions
		require.NotEqual(
			regretScales[i].String(),
			regretScales[i-1].String(),
			"Regret scale should change between epochs %d and %d", i-1, i,
		)
		s.T().Logf("Regret scale: %v", regretScales[i].String())
	}
}

func (s *RewardsTestSuite) TestRewardsIncreasesBalance() {
	s.MintTokensToModule(types.AlloraStakingAccountName, cosmosMath.NewInt(10000000000))

	workerIndexes := testutil.ReturnIndexes(5, 5)
	reputerIndexes := testutil.ReturnIndexes(0, 5)
	// TOPIC 1 - PRE PASS
	topicId, _ := s.FullTopicPass(workerIndexes, reputerIndexes)

	var err error
	reputerStake := make([]cosmosMath.Int, 5)
	for _, index := range reputerIndexes {
		reputerStake[index], err = s.StakingKeeper().GetStakeReputerAuthority(s.Ctx(), topicId, s.AddrsStr(index))
		s.Require().NoError(err)
	}

	workerBalances := make(map[int]sdk.Coin)
	for _, index := range workerIndexes {
		workerBalances[index] = s.BankKeeper().GetBalance(s.Ctx(), s.Addrs(index), params.DefaultBondDenom)
	}

	// TOPIC 1 - FIRST PASS
	s.FullTopicPass(workerIndexes, reputerIndexes, testutil.WithTopicID(topicId))

	for i, index := range reputerIndexes {
		reputerStakeCurrent, err := s.StakingKeeper().GetStakeReputerAuthority(s.Ctx(), topicId, s.AddrsStr(index))
		s.Require().NoError(err)
		s.Require().True(
			reputerStakeCurrent.GT(reputerStake[i]),
			"Reputer %s stake did not increase: %s | %s",
			s.AddrsStr(index),
			reputerStakeCurrent.String(),
			reputerStake[i].String(),
		)
	}

	for _, index := range workerIndexes {
		s.Require().True(s.BankKeeper().GetBalance(s.Ctx(), s.Addrs(index), params.DefaultBondDenom).Amount.GT(workerBalances[index].Amount))
	}
}

func (s *RewardsTestSuite) TestStandardRewardEmissionWithOneInfererAndOneReputer() {
	epochLength := int64(10800)

	_, blockHeight := s.FullTopicPass([]int{0}, []int{5}, testutil.WithEpochLength(epochLength))

	blockHeight += epochLength * 3
	s.WithBlockHeight(blockHeight)

	// mint some rewards to give out
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(10000000000))

	// Trigger end block - rewards distribution
	s.EndBlock()
}

func (s *RewardsTestSuite) TestOnlyFewTopActorsGetReward() {
	s.SetParamsForTest()

	// Reputer Addresses
	reputerIndexes := testutil.ReturnIndexes(0, 25)
	workerIndexes := testutil.ReturnIndexes(25, 25)

	topicId, _ := s.FullTopicPass(workerIndexes, reputerIndexes)
	_, block := s.FullTopicPass(workerIndexes, reputerIndexes, testutil.WithTopicID(topicId))

	networkLossBundles, err := s.ReputerLossKeeper().GetNetworkLossBundleAtBlock(s.Ctx(), topicId, block)
	s.Require().NoError(err)
	s.Require().NotNil(networkLossBundles)

	infererScores, err := rewards.GenerateInferenceScores(
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		block,
		*networkLossBundles)
	s.Require().NoError(err)

	forecasterScores, err := rewards.GenerateForecastScores(
		s.Ctx(),
		*s.EmissionsKeeper(),
		topicId,
		block,
		*networkLossBundles)
	s.Require().NoError(err)

	p, err := s.ParamsKeeper().GetParams(s.Ctx())
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
	require := s.Require()

	const (
		epochLength    = int64(100)
		groundTruthLag = int64(100)
		numTopics      = 2
		numReputers    = 3
		numEpochs      = 2
		fundAmount     = 1000
	)

	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	stake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())

	s.WithBlockHeight(1)
	s.SetParamsForTest()

	reputerIndexes := testutil.ReturnIndexes(0, numReputers)
	workerIndexes := testutil.ReturnIndexes(3, numReputers)

	opts := []testutil.Option{
		testutil.WithEpochLength(epochLength),
		testutil.WithGroundTruthLag(groundTruthLag),
		testutil.WithAlphaRegret(alphaRegret),
		testutil.WithWorkerValues(testutil.GetWorkerValuesFromIndexes(workerIndexes, "0.2")),
		testutil.WithReputerStake(&stake),
		testutil.WithReputerValues(s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.2")),
	}

	// pre-emptive passes for topics
	var topicIDs [numTopics]uint64
	topicIDs[0], _ = s.FullTopicPass(workerIndexes, reputerIndexes, opts...)
	topicIDs[1], _ = s.FullTopicPass(workerIndexes, reputerIndexes, opts...)

	// validate topic weights consistency
	s.validateTopicWeightsConsistency(topicIDs)

	// first phase: equal rewards
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(fundAmount))

	recordStakes := func() [numTopics][numReputers]cosmosMath.Int {
		var stakes [numTopics][numReputers]cosmosMath.Int
		for i, topicID := range topicIDs {
			for j, reputerIdx := range reputerIndexes {
				var err error
				stakes[i][j], err = s.StakingKeeper().GetStakeReputerAuthority(s.Ctx(), topicID, s.AddrsStr(reputerIdx))
				require.NoError(err)
			}
		}
		return stakes
	}

	// record stakes before pass
	stakes0 := recordStakes()

	s.FullTopicPass(workerIndexes, reputerIndexes, append(opts, testutil.WithTopicID(topicIDs[0]))...)
	s.FullTopicPass(workerIndexes, reputerIndexes, append(opts, testutil.WithTopicID(topicIDs[1]))...)

	s.validateWorkerInclusions(topicIDs[0], workerIndexes)

	s.FundTopic(topicIDs[0], s.Addrs(0), cosmosMath.NewInt(fundAmount))
	s.FundTopic(topicIDs[1], s.Addrs(3), cosmosMath.NewInt(fundAmount))

	stakes1 := recordStakes()

	var topicRewardsTotal [numEpochs][numTopics]alloraMath.Dec

	rewardsFromStakes := func(phase int, stakesCurrent, stakesPrevious [2][3]cosmosMath.Int) {
		for i := range stakesCurrent {
			for j := range stakesCurrent[i] {
				currentRewards, err := alloraMath.NewDecFromSdkInt(stakesCurrent[i][j].Sub(stakesPrevious[i][j]))
				require.NoError(err)
				topicRewardsTotal[phase][i], err = topicRewardsTotal[phase][i].Add(currentRewards)
				require.NoError(err)
			}
		}
	}

	// calculate rewards for first epoch
	rewardsFromStakes(0, stakes1, stakes0)
	require.Equal(topicRewardsTotal[0][0], topicRewardsTotal[0][1])

	// second phase: increase stake for first reputer in topic1
	stakeIncrease := cosmosMath.NewInt(1000).Mul(stake)
	s.MintTokensToAddress(s.Addrs(reputerIndexes[0]), stakeIncrease)
	_, err := s.EmissionsMsgServer().AddStake(s.Ctx(), &types.AddStakeRequest{
		Sender:  s.AddrsStr(reputerIndexes[0]),
		Amount:  stakeIncrease,
		TopicId: topicIDs[1],
	})
	require.NoError(err)

	// update stakes and run second epoch
	stakes1[1][0], err = s.StakingKeeper().GetStakeReputerAuthority(s.Ctx(), topicIDs[1], s.AddrsStr(reputerIndexes[0]))
	require.NoError(err)

	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(fundAmount))

	s.FullTopicPass(workerIndexes, reputerIndexes, append(opts, testutil.WithTopicID(topicIDs[0]))...)
	_, block := s.FullTopicPass(workerIndexes, reputerIndexes, append(opts, testutil.WithTopicID(topicIDs[1]))...)

	s.validateStakeIncreaseRewards(topicIDs[1], block, reputerIndexes)

	stakes2 := recordStakes()

	// calculate rewards for second epoch
	rewardsFromStakes(1, stakes2, stakes1)

	// normalize rewards
	var totalRewardsEpoch [numEpochs]alloraMath.Dec
	for i, topicRewards := range topicRewardsTotal {
		totalRewardsEpoch[i], err = topicRewards[0].Add(topicRewards[1])
		require.NoError(err)
	}

	var topicRewardTotalNormalized [numTopics][numEpochs]alloraMath.Dec // [topic][epoch]
	for i, phaseTopicRewards := range topicRewardsTotal {
		for j, topicReward := range phaseTopicRewards {
			topicRewardTotalNormalized[j][i], err = topicReward.Quo(totalRewardsEpoch[i])
			require.NoError(err)
		}
	}

	// in the first round, the rewards should be equal for each topic
	require.True(topicRewardTotalNormalized[0][0].Equal(topicRewardTotalNormalized[1][0]), "%s != %s", topicRewardTotalNormalized[0][0], topicRewardTotalNormalized[1][0])
	// for topic 0, the rewards should be less in the second round
	require.True(topicRewardTotalNormalized[0][0].Gte(topicRewardTotalNormalized[0][1]), "%s <= %s", topicRewardTotalNormalized[0][0], topicRewardTotalNormalized[0][1])
	// in the second round, the rewards should be greater for topic 1
	require.True(topicRewardTotalNormalized[1][0].Lte(topicRewardTotalNormalized[1][1]), "%s >= %s", topicRewardTotalNormalized[1][0], topicRewardTotalNormalized[1][1])
	// the rewards for topic 1 should be greater in the second round
	require.True(topicRewardTotalNormalized[1][0].Lt(topicRewardTotalNormalized[1][1]), "%s >= %s", topicRewardTotalNormalized[1][0], topicRewardTotalNormalized[1][1])
}

// Helper methods (same as before)
func (s *RewardsTestSuite) validateTopicWeightsConsistency(topicIDs [2]uint64) {
	var topicWeights [2]alloraMath.Dec
	var err error
	for i, topicID := range topicIDs {
		topicWeights[i], _, err = s.TopicKeeper().GetPreviousTopicWeight(s.Ctx(), topicID)
		s.Require().NoError(err)
	}

	sumWeights, err := topicWeights[0].Add(topicWeights[1])
	s.Require().NoError(err)

	totalSum, err := s.TopicKeeper().GetTotalSumPreviousTopicWeights(s.Ctx())
	s.Require().NoError(err)

	inDelta, err := alloraMath.InDelta(totalSum, sumWeights, alloraMath.MustNewDecFromString("0.0001"))
	s.Require().NoError(err)
	s.Require().True(inDelta, "Topic weights sum mismatch: %s vs %s", totalSum, sumWeights)
}

func (s *RewardsTestSuite) validateWorkerInclusions(topicID uint64, workerIndexes []int) {
	for _, workerIndex := range workerIndexes {
		infererCount, _ := s.EmissionsKeeper().GetCountInfererInclusionsInTopic(s.Ctx(), topicID, s.AddrsStr(workerIndex))
		s.Require().GreaterOrEqual(infererCount, uint64(2))

		forecasterCount, _ := s.EmissionsKeeper().GetCountForecasterInclusionsInTopic(s.Ctx(), topicID, s.AddrsStr(workerIndex))
		s.Require().GreaterOrEqual(forecasterCount, uint64(1))
	}
}

func (s *RewardsTestSuite) validateStakeIncreaseRewards(topicID uint64, block int64, reputerIndexes []int) {
	taskRewards, _ := s.GenerateRewards(topicID, block)
	var reputerRewards [3]types.TaskReward
	for i, reputerIdx := range reputerIndexes {
		for _, reward := range taskRewards {
			if reward.Type == types.ReputerAndDelegatorRewardType && reward.Address == s.AddrsStr(reputerIdx) {
				reputerRewards[i] = reward
				break
			}
		}
	}
	s.Require().True(reputerRewards[0].Reward.Gt(reputerRewards[1].Reward))
	s.Require().True(reputerRewards[0].Reward.Gt(reputerRewards[2].Reward))
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
			epochLength := int64(5)
			groundTruthLag := int64(5)
			alphaRegret := alloraMath.MustNewDecFromString("0.1")

			s.SetParamsForTest()

			reputerIndexes := testutil.ReturnIndexes(0, 6)
			workerIndexes := testutil.ReturnIndexes(6, 3)
			stake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())

			// PRE RUN
			topicId, _ := s.FullTopicPass(
				workerIndexes,
				reputerIndexes,
				testutil.WithEpochLength(epochLength),
				testutil.WithGroundTruthLag(groundTruthLag),
				testutil.WithAlphaRegret(alphaRegret),
				testutil.WithReputerStake(&stake),
			)

			var err error
			reputerStakes0 := make([]cosmosMath.Int, len(reputerIndexes))
			for i, index := range reputerIndexes {
				reputerStakes0[i], err = s.StakingKeeper().GetStakeReputerAuthority(s.Ctx(), topicId, s.AddrsStr(index))
				require.NoError(err)
			}

			err = s.TopicKeeper().SetPreviousTopicWeight(s.Ctx(), topicId, alloraMath.MustNewDecFromString("100"))
			require.NoError(err)

			reputerValues := s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, tc.baseValue)
			exceptionReputerIdx := 5
			reputerValues[exceptionReputerIdx].WorkerValues[s.AddrsStr(6)] = tc.deltaValue // used as delta
			workerValues := testutil.GetWorkerValuesFromIndexes(workerIndexes, tc.baseValue)

			s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

			s.FullTopicPass(
				workerIndexes,
				reputerIndexes,
				testutil.WithWorkerValues(workerValues),
				testutil.WithReputerValues(reputerValues),
				testutil.WithReputerStake(&stake),
				testutil.WithTopicID(topicId),
			)

			reputerStakes1 := make([]cosmosMath.Int, len(reputerIndexes))
			for i, index := range reputerIndexes {
				reputerStakes1[i], err = s.StakingKeeper().GetStakeReputerAuthority(s.Ctx(), topicId, s.AddrsStr(index))
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
	require := s.Require()

	s.WithBlockHeight(1)
	s.SetParamsForTest()

	// Constants
	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	epochLength := int64(30)
	groundTruthLag := int64(30)
	stake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())

	// Participant configs
	reputerIndexes := testutil.ReturnIndexes(0, 3)
	workerIndexes := testutil.ReturnIndexes(3, 3)
	workerValues := testutil.GetWorkerValuesFromIndexes(workerIndexes, "0.2")

	// Test scenarios
	scenarios := []struct {
		name          string
		reputerIdxs   []int
		reputerValues []testutil.TestReputerValue
	}{
		{
			name:          "setup",
			reputerIdxs:   reputerIndexes,
			reputerValues: s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.2"),
		},
		{
			name:          "all_reputers",
			reputerIdxs:   reputerIndexes,
			reputerValues: s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.2"),
		},
		{
			name:          "reduced_reputers",
			reputerIdxs:   reputerIndexes[:2],
			reputerValues: s.GetReputerValuesFromIndexes(reputerIndexes[:2], workerIndexes, "0.2"),
		},
	}

	var topicId uint64
	reputerStakeSnapshots := make([][]cosmosMath.Int, len(scenarios))

	// Combined setup and execution function
	runScenario := func(scenarioIdx int) {
		// Prepare rewards for scenario
		s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(1000))

		opts := []testutil.Option{
			testutil.WithTopicID(topicId),
			testutil.WithEpochLength(epochLength),
			testutil.WithGroundTruthLag(groundTruthLag),
			testutil.WithAlphaRegret(alphaRegret),
			testutil.WithWorkerValues(workerValues),
			testutil.WithReputerStake(&stake),
			testutil.WithReputerValues(scenarios[scenarioIdx].reputerValues),
		}

		// Execute the pass
		topicId, _ = s.FullTopicPass(workerIndexes, scenarios[scenarioIdx].reputerIdxs, opts...)

		// Record stakes
		reputerStakeSnapshots[scenarioIdx] = make([]cosmosMath.Int, len(reputerIndexes))
		for i, idx := range reputerIndexes {
			var err error
			reputerStakeSnapshots[scenarioIdx][i], err = s.StakingKeeper().GetStakeReputerAuthority(s.Ctx(), topicId, s.AddrsStr(idx))
			require.NoError(err)
		}

		// Fund topic for next scenario if not the last one
		topicFundAmount := cosmosMath.NewInt(1000)
		s.FundTopic(topicId, s.Addrs(0), topicFundAmount)
	}

	// Execute scenarios
	for i := range scenarios {
		runScenario(i)
	}

	// Calculate rewards for each scenario
	calculateRewards := func(beforeIdx, afterIdx int) []cosmosMath.Int {
		reputerRewards := make([]cosmosMath.Int, len(reputerIndexes))
		for i := range reputerRewards {
			reputerRewards[i] = reputerStakeSnapshots[afterIdx][i].Sub(reputerStakeSnapshots[beforeIdx][i])
		}
		return reputerRewards
	}

	reputerRewards0 := calculateRewards(0, 1) // Initial to after first scenario
	reputerRewards1 := calculateRewards(1, 2) // After first to after second scenario

	// Assertions: verify behavior when participant drops out
	require.True(reputerRewards1[2].IsZero(), "Dropped reputer should get zero rewards")
	require.True(reputerRewards0[2].IsPositive(), "Dropped reputer should have gotten rewards in first round")
	require.True(reputerRewards1[0].GT(reputerRewards0[0]), "Remaining reputer 0 rewards should increase")
	require.True(reputerRewards1[1].GT(reputerRewards0[1]), "Remaining reputer 1 rewards should increase")
}

func (s *RewardsTestSuite) TestRewardIncreaseContinuouslyAfterTopicReactivated() {
	require := s.Require()

	s.WithBlockHeight(1)
	s.SetParamsForTest()

	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	cosmosOneE18 := inferencesynthesis.CosmosIntOneE18()
	initialStake := cosmosMath.NewInt(1000)
	baseStake := cosmosMath.NewInt(1000).Mul(cosmosOneE18)

	// Predefined stakes
	stakes := []cosmosMath.Int{
		cosmosMath.NewInt(1176644).Mul(cosmosOneE18),
		cosmosMath.NewInt(384623).Mul(cosmosOneE18),
		cosmosMath.NewInt(394676).Mul(cosmosOneE18),
		cosmosMath.NewInt(207999).Mul(cosmosOneE18),
		cosmosMath.NewInt(368582).Mul(cosmosOneE18),
	}

	// Topic configs
	topicConfigs := []struct {
		rIdx, wIdx       []int
		epoch, gtl, mult int64
		values           []string
	}{
		{testutil.ReturnIndexes(0, 3), testutil.ReturnIndexes(3, 3), 30, 30, 1, []string{"0.1", "0.2", "0.3"}},
		{testutil.ReturnIndexes(6, 3), testutil.ReturnIndexes(9, 3), 60, 60, 2, []string{"0.4", "0.5", "0.6"}},
	}

	var topicIDs []uint64

	// Combined setup and pass function
	runPassWithSetup := func(topicIdx int, isSetup bool) []types.TaskReward {
		cfg := topicConfigs[topicIdx]
		workerValues := testutil.GetWorkerValuesFromIndexes(cfg.wIdx, cfg.values...)
		reputerValues := s.GetReputerValuesFromIndexes(cfg.rIdx, cfg.wIdx, cfg.values...)

		addrs := make([]string, len(cfg.wIdx))
		for i, idx := range cfg.wIdx {
			addrs[i] = s.AddrsStr(idx)
		}
		slices.Sort(addrs)
		reputerValues[0].OneOutInfValues = map[string]string{addrs[0]: "0.1"}
		// Apply special values based on topic index
		if topicIdx != 0 {
			reputerValues[0].WorkerValues[addrs[0]] = "0.2"
			reputerValues[0].OneOutInfValues = map[string]string{addrs[0]: "0.2"}
		}

		// Build common options
		opts := []testutil.Option{
			testutil.WithAlphaRegret(alphaRegret),
			testutil.WithReputerStake(&baseStake),
			testutil.WithEpochLength(cfg.epoch),
			testutil.WithGroundTruthLag(cfg.gtl),
			testutil.WithWorkerValues(workerValues),
			testutil.WithReputerValues(reputerValues),
		}

		if !isSetup {
			// Add topic ID for existing topic
			opts = append(opts, testutil.WithTopicID(topicIDs[topicIdx]))
		}

		topicID, block := s.FullTopicPass(cfg.wIdx, cfg.rIdx, opts...)

		if isSetup {
			topicIDs = append(topicIDs, topicID)
			// Add stakes for reputers
			for j, rIdx := range cfg.rIdx {
				stakeAmount := stakes[j].MulRaw(cfg.mult)
				s.MintTokensToAddress(s.Addrs(rIdx), stakeAmount)
				_, err := s.EmissionsMsgServer().AddStake(s.Ctx(), &types.AddStakeRequest{
					Sender: s.AddrsStr(rIdx), Amount: stakeAmount, TopicId: topicID})
				require.NoError(err)
			}
			// Fund topic
			fundAmount := initialStake.MulRaw(cfg.mult)
			s.FundTopic(topicID, s.Addrs(cfg.rIdx[0]), fundAmount)
			return nil // No rewards on setup pass
		}
		rews, _ := s.GenerateRewards(topicIDs[topicIdx], block)
		return rews
	}

	// Setup both topics
	runPassWithSetup(0, true)
	runPassWithSetup(1, true)

	// Execute actual test passes
	rewards0_0 := runPassWithSetup(0, false)
	runPassWithSetup(1, false)

	// Check topic 0 is inactive, reactivate, and verify rewards
	isActive, err := s.TopicKeeper().IsTopicActive(s.Ctx(), topicIDs[0])
	require.NoError(err)
	require.False(isActive)

	s.FundTopic(topicIDs[0], s.Addrs(topicConfigs[0].rIdx[0]), initialStake.MulRaw(3))

	isActive, err = s.TopicKeeper().IsTopicActive(s.Ctx(), topicIDs[0])
	require.NoError(err)
	require.True(isActive)

	rewards0_1 := runPassWithSetup(0, false)
	require.Equal(len(rewards0_1), len(rewards0_0))
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
					1: testutil.DecPtr("0.5"),
					2: testutil.DecPtr("0.3"),
					3: testutil.DecPtr("0.2"),
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
					1: testutil.DecPtr("500.0"),
					2: testutil.DecPtr("300.0"),
					3: testutil.DecPtr("200.0"),
				}
				return s.compareRewards(rewards, expected)
			},
			expectedError: nil,
		},
		{
			name: "Single topic",
			setupFunc: func() (map[uint64]*alloraMath.Dec, []uint64, alloraMath.Dec, alloraMath.Dec, map[uint64]int64, alloraMath.Dec) {
				weights := map[uint64]*alloraMath.Dec{
					1: testutil.DecPtr("1.0"),
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
					1: testutil.DecPtr("1000.0"),
				}
				return s.compareRewards(rewards, expected)
			},
			expectedError: nil,
		},
		{
			name: "Zero total reward",
			setupFunc: func() (map[uint64]*alloraMath.Dec, []uint64, alloraMath.Dec, alloraMath.Dec, map[uint64]int64, alloraMath.Dec) {
				weights := map[uint64]*alloraMath.Dec{
					1: testutil.DecPtr("0.5"),
					2: testutil.DecPtr("0.5"),
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
					1: testutil.DecPtr("0"),
					2: testutil.DecPtr("0"),
				}
				return s.compareRewards(rewards, expected)
			},
			expectedError: nil,
		},
		{
			name: "Different epoch lengths",
			setupFunc: func() (map[uint64]*alloraMath.Dec, []uint64, alloraMath.Dec, alloraMath.Dec, map[uint64]int64, alloraMath.Dec) {
				weights := map[uint64]*alloraMath.Dec{
					1: testutil.DecPtr("0.75"),
					2: testutil.DecPtr("0.25"),
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
					1: testutil.DecPtr("75"),
					2: testutil.DecPtr("50"),
				}
				return s.compareRewards(rewards, expected)
			},
			expectedError: nil,
		},
		{
			name: "Very small weights",
			setupFunc: func() (map[uint64]*alloraMath.Dec, []uint64, alloraMath.Dec, alloraMath.Dec, map[uint64]int64, alloraMath.Dec) {
				weights := map[uint64]*alloraMath.Dec{
					1: testutil.DecPtr("0.000000000000000001"),
					2: testutil.DecPtr("0.000000000000000002"),
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
					1: testutil.DecPtr("333.333333333333333333"),
					2: testutil.DecPtr("666.666666666666666667"),
				}
				return s.compareRewards(rewards, expected)
			},
			expectedError: nil,
		},
		{
			name: "Mismatched weights and sorted topics",
			setupFunc: func() (map[uint64]*alloraMath.Dec, []uint64, alloraMath.Dec, alloraMath.Dec, map[uint64]int64, alloraMath.Dec) {
				weights := map[uint64]*alloraMath.Dec{
					1: testutil.DecPtr("0.5"),
					2: testutil.DecPtr("0.5"),
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
					1: testutil.DecPtr("0.5"),
					2: testutil.DecPtr("0.3"),
					3: testutil.DecPtr("0.2"),
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
					1: testutil.DecPtr("25.0"),
					2: testutil.DecPtr("15.0"),
					3: testutil.DecPtr("10.0"),
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
				Ctx:                             s.Ctx(),
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

// Helper function to compare rewards
func (s *RewardsTestSuite) compareRewards(actual, expected map[uint64]*alloraMath.Dec) bool {
	if len(actual) != len(expected) {
		return false
	}
	for topicID, expectedReward := range expected {
		actualReward, exists := actual[topicID]
		if !exists {
			return false
		}
		inDelta, err := alloraMath.InDelta(*actualReward, *expectedReward, alloraMath.MustNewDecFromString("0.0001"))
		s.Require().NoError(err)
		s.Require().True(inDelta)
		if !inDelta {
			s.T().Logf("Actual   %v not found", actualReward)
			s.T().Logf("Expected %v not found", expectedReward)
			return false
		}
	}
	return true
}

func (s *RewardsTestSuite) TestNoActiveParticipantsNoRewardsForTopic() {
	require := s.Require()

	block := int64(1)
	s.WithBlockHeight(block)

	s.SetParamsForTest()

	creatorIndex := 40
	reputerIndex := 41

	topicId := s.SetupTopic(s.Addrs(creatorIndex))
	s.SetupParticipants(topicId, []int{reputerIndex}, true)

	// Record initial balance of the rewards module account
	rewardsModuleAddr := s.AccountKeeper().GetModuleAddress(types.AlloraRewardsAccountName)
	initialRewardsBalance := s.BankKeeper().GetBalance(s.Ctx(), rewardsModuleAddr, params.DefaultBondDenom)

	// Record initial balance of the ecosystem account
	ecosystemAddr := s.AccountKeeper().GetModuleAddress(minttypes.EcosystemModuleName)
	initialEcosystemBalance := s.BankKeeper().GetBalance(s.Ctx(), ecosystemAddr, params.DefaultBondDenom)

	// Simulate moving to the end of the first epoch
	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	require.NoError(err)
	nextBlock := block + topic.EpochLength
	s.WithBlockHeight(nextBlock)

	// Trigger EndBlocker to process potential rewards
	s.EndBlock()

	// Check topic weight after funding and epoch end
	topicWeight, _, err := s.TopicKeeper().GetPreviousTopicWeight(s.Ctx(), topicId)
	require.NoError(err)
	require.True(topicWeight.Gt(alloraMath.ZeroDec()), "Topic weight should be greater than zero after funding")

	// Verify no rewards were distributed for this topic (moved to ecosystem account)
	finalRewardsBalance := s.BankKeeper().GetBalance(s.Ctx(), rewardsModuleAddr, params.DefaultBondDenom)
	require.True(initialRewardsBalance.Amount.GT(finalRewardsBalance.Amount),
		"Rewards module - topic %d: Initial: %s, Final: %s",
		topicId, initialRewardsBalance.String(), finalRewardsBalance.String())

	// Verify if rewards were moved to the ecosystem account
	ecosystemBalance := s.BankKeeper().GetBalance(s.Ctx(), ecosystemAddr, params.DefaultBondDenom)
	require.True(ecosystemBalance.Amount.GT(initialEcosystemBalance.Amount),
		"Ecosystem account - topic %d: Initial: %s, Final: %s",
		topicId, initialEcosystemBalance.String(), ecosystemBalance.String())
}

func areTaskRewardsEqualIgnoringTopicId(s *RewardsTestSuite, A []types.TaskReward, B []types.TaskReward) bool {
	if len(A) != len(B) {
		s.Fail("Lengths are different")
	}

	for _, taskRewardA := range A {
		found := false
		for _, taskRewardB := range B {
			if taskRewardA.Address == taskRewardB.Address && taskRewardA.Type == taskRewardB.Type {
				if found {
					s.Fail("Worker %v found twice", taskRewardA.Address)
				}
				found = true
				inDelta, err := alloraMath.InDelta(
					taskRewardA.Reward,
					taskRewardB.Reward,
					alloraMath.MustNewDecFromString("0.00001"),
				)
				if err != nil {
					s.Fail("Error finding out if taskRewardA in delta with taskRewardB",
						taskRewardA.Reward.String(),
						taskRewardB.Reward.String(),
					)
				}
				if !inDelta {
					return false
				}
			}
		}
		if !found {
			s.T().Logf("Worker %v not found", taskRewardA.Address)
			return false
		}
	}

	return true
}

// printRewardsDistributionTable formats and prints the rewards distribution in a readable table format
func (s *RewardsTestSuite) printRewardsDistributionTable(rewardsDistributions [2][2][]types.TaskReward) {
	var sb strings.Builder
	sb.WriteString("\n\nREWARDS DISTRIBUTIONS TABLE\n")
	sb.WriteString("================================================================================\n")

	alphaValues := []string{"0.1", "0.2"}

	for alphaIndex := 0; alphaIndex < 2; alphaIndex++ {
		sb.WriteString(fmt.Sprintf("\n[ALPHA INDEX %d] TaskRewardAlpha = %s\n", alphaIndex, alphaValues[alphaIndex]))
		sb.WriteString("--------------------------------------------------------------------------------\n")

		for phase := 0; phase < 2; phase++ {
			phaseName := "Normal Performance"
			if phase == 1 {
				phaseName = "Improved Performance"
			}
			sb.WriteString(fmt.Sprintf("\n  [PHASE %d] %s - Total Rewards: %d\n", phase, phaseName, len(rewardsDistributions[alphaIndex][phase])))
			sb.WriteString("  ----------------------------------------------------------------------------\n")
			sb.WriteString(fmt.Sprintf("  %-4s | %-42s | %-20s | %-8s | %-20s\n", "Idx", "Address", "Type", "TopicId", "Reward"))
			sb.WriteString("  ----------------------------------------------------------------------------\n")

			for i, reward := range rewardsDistributions[alphaIndex][phase] {
				sb.WriteString(fmt.Sprintf("  %-4d | %-42s | %-20s | %-8d | %s\n",
					i,
					reward.Address,
					reward.Type,
					reward.TopicId,
					reward.Reward.String()))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("================================================================================\n")

	// Log the entire table in one statement to preserve formatting
	s.Ctx().Logger().Info(sb.String())
}
