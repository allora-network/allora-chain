package rewards_test

import (
	cosmosMath "cosmossdk.io/math"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Forces dripping of two topics, which initially has two active topics, but next round
// leads to one of them not reaching the MinTopicWeight thus being inactivated.
func (s *RewardsTestSuite) TestGetAndUpdateActiveTopicWeights() {
	maxActiveTopicsNum := uint64(3)
	params := types.DefaultParams()
	params.BlocksPerMonth = 864000
	params.MaxActiveTopicsPerBlock = maxActiveTopicsNum
	params.MaxPageLimit = uint64(100)
	params.MinTopicWeight = alloraMath.MustNewDecFromString("100")
	params.TopicRewardAlpha = alloraMath.MustNewDecFromString("0.5")
	params.TopicRewardStakeImportance = alloraMath.OneDec()
	params.TopicRewardFeeRevenueImportance = alloraMath.OneDec()
	err := s.EmissionsKeeper().SetParams(s.Ctx(), params)
	s.Require().NoError(err, "Setting parameters should not fail")

	setTopicWeight := func(topicId uint64, revenue, stake int64) {
		err = s.EmissionsKeeper().AddTopicFeeRevenue(s.Ctx(), topicId, cosmosMath.NewInt(revenue))
		s.Require().NoError(err)
		err = s.EmissionsKeeper().SetTopicStake(s.Ctx(), topicId, cosmosMath.NewInt(stake))
		s.Require().NoError(err)
	}

	s.WithBlockHeight(1)
	// Assume topic initially active
	topic1 := s.MockTopic()
	topic1.Id = 2
	topic1.EpochLength = 15
	topic1.GroundTruthLag = topic1.EpochLength
	topic1.WorkerSubmissionWindow = topic1.EpochLength
	topic2 := s.MockTopic()
	topic2.Id = 3
	topic2.EpochLength = 15
	topic2.GroundTruthLag = topic2.EpochLength
	topic2.WorkerSubmissionWindow = topic2.EpochLength

	totalSumPreviousTopicWeights, err := s.EmissionsKeeper().GetTotalSumPreviousTopicWeights(s.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(totalSumPreviousTopicWeights, alloraMath.ZeroDec(), "Total sum of previous topic weights at start should be zero")
	setTopicWeight(topic1.Id, 10, 10)
	err = s.EmissionsKeeper().SetTopic(s.Ctx(), topic1.Id, topic1)
	s.Require().NoError(err)
	err = s.EmissionsKeeper().ActivateTopic(s.Ctx(), topic1.Id)
	s.Require().NoError(err, "Activating topic should not fail")

	totalSumPreviousTopicWeights, err = s.EmissionsKeeper().GetTotalSumPreviousTopicWeights(s.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(totalSumPreviousTopicWeights, alloraMath.ZeroDec(), "Total sum of previous topic weights should still be 0 bc previous topic weight is not set")

	setTopicWeight(topic2.Id, 30, 10)
	err = s.EmissionsKeeper().SetTopic(s.Ctx(), topic2.Id, topic2)
	s.Require().NoError(err)
	err = s.EmissionsKeeper().ActivateTopic(s.Ctx(), topic2.Id)
	s.Require().NoError(err, "Activating topic should not fail")

	block := 10
	s.WithBlockHeight(int64(block))
	_, _, _, err = rewards.GetAndUpdateActiveTopicWeights(s.Ctx(), *s.EmissionsKeeper(), int64(block))
	s.Require().NoError(err, "Activating topic should not fail")

	// Previous weights still not moved
	totalSumPreviousTopicWeights, err = s.EmissionsKeeper().GetTotalSumPreviousTopicWeights(s.Ctx())
	s.T().Logf("totalSumPreviousTopicWeights: %v", totalSumPreviousTopicWeights)
	s.Require().NoError(err)
	s.Require().Equal(totalSumPreviousTopicWeights, alloraMath.ZeroDec(), "Total sum of previous topic weights should not be 0 after settings topic weights")

	previousTopicWeights, _, err := s.EmissionsKeeper().GetPreviousTopicWeight(s.Ctx(), topic1.Id)
	s.T().Logf("topic1 previousTopicWeights: %v", previousTopicWeights)
	s.Require().NoError(err)
	s.Require().Equal(previousTopicWeights, alloraMath.ZeroDec(), "Previous topic weights should still be 0 after settings topic weights")

	block = 16
	s.WithBlockHeight(int64(block))
	_, _, _, err = rewards.GetAndUpdateActiveTopicWeights(s.Ctx(), *s.EmissionsKeeper(), int64(block))
	s.Require().NoError(err, "Activating topic should not fail")

	activeTopics, err := s.EmissionsKeeper().GetActiveTopicIdsAtBlock(s.Ctx(), 31)
	s.Require().NoError(err, "Fetching active topics should not produce an error")
	s.Require().Equal(2, len(activeTopics.TopicIds), "Should retrieve exactly two active topics")

	err = s.EmissionsKeeper().SetParams(s.Ctx(), params)
	s.Require().NoError(err)

	block = 31
	s.WithBlockHeight(int64(block))
	_, _, _, err = rewards.GetAndUpdateActiveTopicWeights(s.Ctx(), *s.EmissionsKeeper(), int64(block))
	s.Require().NoError(err, "Activating topic should not fail")

	activeTopics, err = s.EmissionsKeeper().GetActiveTopicIdsAtBlock(s.Ctx(), 46)
	s.Require().NoError(err, "Fetching active topics should not produce an error")
	s.Require().Equal(1, len(activeTopics.TopicIds), "Should retrieve exactly one active topics")
}

func (s *RewardsTestSuite) TestGetRewardAndRemovedRewardableTopics() {
	block := int64(1)
	s.WithBlockHeight(block)

	s.SetParamsForTest()

	reputerIndexes := testutil.ReturnIndexes(0, 3)
	workerIndexes := testutil.ReturnIndexes(3, 3)

	// setup topic
	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	epochLength := int64(100)
	topic := s.FullTopicSetup(workerIndexes, reputerIndexes, testutil.WithAlphaRegret(alphaRegret), testutil.WithEpochLength(epochLength))
	topicId := topic.Id

	// mint some rewards to give out
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(100000))

	// Move to end of this epoch block
	nextBlock, _, err := s.EmissionsKeeper().GetNextPossibleChurningBlockByTopicId(s.Ctx(), topicId)
	s.Require().NoError(err)
	block = nextBlock
	s.WithBlockHeight(block)
	err = s.EmissionsAppModule().EndBlock(s.Ctx())
	s.Require().NoError(err)

	topic, err = s.EmissionsKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Insert inference
	s.SetupInferences(topicId, topic.EpochLastEnded, workerIndexes)

	// Advance to close the window
	block = block + topic.WorkerSubmissionWindow
	s.WithBlockHeight(block)

	// EndBlock closes the  worker nonce
	err = s.EmissionsAppModule().EndBlock(s.Ctx())
	s.Require().NoError(err)

	// Advance to close the window
	block = block + topic.WorkerSubmissionWindow
	s.WithBlockHeight(block)

	// EndBlock closes the  worker nonce
	err = s.EmissionsAppModule().EndBlock(s.Ctx())
	s.Require().NoError(err)

	// Run endBlock at 201, epoch end block - important bc of the topic in/activation
	block = block + (topic.GroundTruthLag - topic.WorkerSubmissionWindow)
	s.WithBlockHeight(block)
	s.T().Logf("****  Moved to next block %d", block)
	err = s.EmissionsAppModule().EndBlock(s.Ctx())
	s.Require().NoError(err)

	// Insert reputation
	block = block + 1
	s.WithBlockHeight(block)

	err = s.InsertReputerLossBundle(topicId, topic.EpochLastEnded, reputerIndexes)
	s.Require().NoError(err)

	block = block + 1
	s.WithBlockHeight(block)
	err = s.EmissionsAppModule().EndBlock(s.Ctx())
	s.Require().NoError(err)

	// Move to next end of epoch
	nextBlock, _, err = s.EmissionsKeeper().GetNextPossibleChurningBlockByTopicId(s.Ctx(), topicId)
	s.Require().NoError(err)
	block = nextBlock
	s.WithBlockHeight(block)
	// EndBlock for closes the reputer nonce & add rewardable nonce
	err = s.EmissionsAppModule().EndBlock(s.Ctx())
	s.Require().NoError(err)

	totalSumPreviousTopicWeights, err := s.EmissionsKeeper().GetTotalSumPreviousTopicWeights(s.Ctx())
	s.T().Logf("totalSumPreviousTopicWeights: %v", totalSumPreviousTopicWeights)
	s.Require().NoError(err)
	s.Require().NotEqual(totalSumPreviousTopicWeights, alloraMath.MustNewDecFromString("0"), "Total sum of previous topic weights should not be 0 after endBlocker topic weights")
}

func (s *RewardsTestSuite) TestPreviousTopicWeightsAfterInactivation() {
	block := int64(1)
	s.WithBlockHeight(block)

	s.SetParamsForTest()

	reputerIndexes := testutil.ReturnIndexes(0, 3)
	workerIndexes := testutil.ReturnIndexes(3, 3)

	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	epochLength := int64(100)
	groundTruthLag := int64(100)
	topic := s.FullTopicSetup(
		workerIndexes,
		reputerIndexes,
		testutil.WithAlphaRegret(alphaRegret),
		testutil.WithEpochLength(epochLength),
		testutil.WithGroundTruthLag(groundTruthLag),
	)
	topicId := topic.Id

	reputerValues := s.GetReputerValuesFromIndexes(reputerIndexes, workerIndexes, "0.2")
	workerValues := testutil.GetWorkerValuesFromIndexes(workerIndexes, "0.2")

	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(100000))
	totalSumPreviousTopicWeights, err := s.EmissionsKeeper().GetTotalSumPreviousTopicWeights(s.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(totalSumPreviousTopicWeights.String(), alloraMath.ZeroDec().String(), "Total sum of previous topic weights should be zero on start")

	// Move to end of this epoch block
	nextBlock, _, err := s.EmissionsKeeper().GetNextPossibleChurningBlockByTopicId(s.Ctx(), topicId)
	s.Require().NoError(err)
	block = nextBlock
	s.WithBlockHeight(block)
	s.T().Logf("****  Moved to next block %d", block)
	err = s.EmissionsAppModule().EndBlock(s.Ctx())
	s.Require().NoError(err)

	topicPreviousWeight, noPrior, err := s.EmissionsKeeper().GetPreviousTopicWeight(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().False(topicPreviousWeight.Equal(alloraMath.ZeroDec()), "Previous topic weight should be zero after endBlocker")
	s.Require().Equal(noPrior, false, "A prior weight should have been set")

	totalSumPreviousTopicWeights, err = s.EmissionsKeeper().GetTotalSumPreviousTopicWeights(s.Ctx())
	s.T().Logf("totalSumPreviousTopicWeights: %v", totalSumPreviousTopicWeights)
	s.Require().NoError(err)
	s.Require().NotEqual(totalSumPreviousTopicWeights.String(), alloraMath.ZeroDec().String(), "Total sum of previous topic weights should be zero after endBlocker")
	// At this point, topic total weight should be equal to the topic's previous weight
	s.Require().True(topicPreviousWeight.Equal(totalSumPreviousTopicWeights), "Topic total weight should be equal to the topic's previous weight")

	topic, err = s.EmissionsKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Insert inference
	s.SetupInferences(topicId, topic.EpochLastEnded, workerIndexes, workerValues...)

	// Advance to close the worker window
	block = block + topic.WorkerSubmissionWindow
	s.WithBlockHeight(block)
	s.T().Logf("****  Moved to next block %d", block)
	err = s.EmissionsAppModule().EndBlock(s.Ctx())
	s.Require().NoError(err)

	// Run endBlock at 201, epoch end block - important bc of the topic in/activation
	block = block + (topic.GroundTruthLag - topic.WorkerSubmissionWindow)
	s.WithBlockHeight(block)
	s.T().Logf("****  Moved to next block %d", block)
	err = s.EmissionsAppModule().EndBlock(s.Ctx())
	s.Require().NoError(err)

	// Run block at 202, epoch end block
	block = block + 1
	s.WithBlockHeight(block)
	s.T().Logf("****  Moved to next block %d", block)

	// Generate and insert loss data
	err = s.InsertReputerLossBundle(topicId, topic.EpochLastEnded, reputerIndexes, testutil.WithReputerValues(reputerValues))
	s.Require().NoError(err)
	s.T().Logf("Inserted loss data for topic %d at block %d", topicId, block)

	// Move to next end of epoch
	nextBlock, _, err = s.EmissionsKeeper().GetNextPossibleChurningBlockByTopicId(s.Ctx(), topicId)
	s.Require().NoError(err)
	block = nextBlock
	s.WithBlockHeight(block)
	s.T().Logf("****  Moved to next block %d", block)
	err = s.EmissionsAppModule().EndBlock(s.Ctx())
	s.Require().NoError(err)

	// Check previousTopicWeights and totalPreviousTopicWeights
	previousTopicWeight, _, err := s.EmissionsKeeper().GetPreviousTopicWeight(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().False(alloraMath.ZeroDec().Equal(previousTopicWeight), "Previous topic weight should not be zero after endBlocker")

	totalSumPreviousTopicWeights, err = s.EmissionsKeeper().GetTotalSumPreviousTopicWeights(s.Ctx())
	s.Require().NoError(err)
	s.Require().True(previousTopicWeight.Equal(totalSumPreviousTopicWeights), "Total sum of previous topic weights should not be zero after endBlocker")

	// Inactivate the topic
	err = s.EmissionsKeeper().InactivateTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Check previousTopicWeights and totalPreviousTopicWeights after inactivation
	inactivePreviousTopicWeight, _, err := s.EmissionsKeeper().GetPreviousTopicWeight(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().Equal(previousTopicWeight, inactivePreviousTopicWeight, "Previous topic weight should remain unchanged after inactivation")
	s.Require().False(alloraMath.ZeroDec().Equal(inactivePreviousTopicWeight), "Previous topic weight should not be zero after inactivation")

	inactiveTotalSumPreviousTopicWeights, err := s.EmissionsKeeper().GetTotalSumPreviousTopicWeights(s.Ctx())
	s.Require().NoError(err)
	s.Require().True(alloraMath.ZeroDec().Equal(inactiveTotalSumPreviousTopicWeights), "Total sum of previous topic weights should be zero after inactivation")

	// Reactivate the topic
	err = s.EmissionsKeeper().ActivateTopic(s.Ctx(), topicId)
	s.Require().NoError(err)

	// Check previousTopicWeights and totalPreviousTopicWeights after reactivation
	reactivePreviousTopicWeight, _, err := s.EmissionsKeeper().GetPreviousTopicWeight(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().Equal(previousTopicWeight, reactivePreviousTopicWeight, "Previous topic weight should remain unchanged after reactivation")
	s.Require().False(alloraMath.ZeroDec().Equal(reactivePreviousTopicWeight), "Previous topic weight should not be zero after reactivation")

	reactiveTotalSumPreviousTopicWeights, err := s.EmissionsKeeper().GetTotalSumPreviousTopicWeights(s.Ctx())
	s.Require().NoError(err)
	s.Require().True(previousTopicWeight.Equal(reactiveTotalSumPreviousTopicWeights), "Total sum of previous topic weights should be equal to previous topic weight after reactivation")
}
