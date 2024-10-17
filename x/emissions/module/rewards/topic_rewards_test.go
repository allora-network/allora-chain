package rewards_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func mockTopic(s *RewardsTestSuite) types.Topic {
	return types.Topic{
		Id:                       1,
		Creator:                  s.addrs[5].String(),
		Metadata:                 "test",
		LossMethod:               "mse",
		EpochLength:              10800,
		EpochLastEnded:           0,
		GroundTruthLag:           10800,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		InitialRegret:            alloraMath.ZeroDec(),
		AlphaRegret:              alloraMath.NewDecFromInt64(1),
		AllowNegative:            false,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		WorkerSubmissionWindow:   10,
	}
}

func (s *RewardsTestSuite) TestGetAndUpdateActiveTopicWeights() {
	ctx := s.ctx
	maxActiveTopicsNum := uint64(2)
	params := types.DefaultParams()
	params.BlocksPerMonth = 864000
	params.MaxActiveTopicsPerBlock = maxActiveTopicsNum
	params.MaxPageLimit = uint64(100)
	params.TopicRewardAlpha = alloraMath.MustNewDecFromString("0.5")
	params.TopicRewardStakeImportance = alloraMath.OneDec()
	params.TopicRewardFeeRevenueImportance = alloraMath.MustNewDecFromString("1")
	err := s.emissionsKeeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	setTopicWeight := func(topicId uint64, revenue, stake int64) {
		err = s.emissionsKeeper.AddTopicFeeRevenue(ctx, topicId, cosmosMath.NewInt(revenue))
		s.Require().NoError(err)
		err = s.emissionsKeeper.SetTopicStake(ctx, topicId, cosmosMath.NewInt(stake))
		s.Require().NoError(err)
	}

	ctx = s.ctx.WithBlockHeight(1)
	// Assume topic initially active
	topic1 := mockTopic(s)
	topic1.Id = 1
	topic1.EpochLength = 15
	topic1.GroundTruthLag = topic1.EpochLength
	topic1.WorkerSubmissionWindow = topic1.EpochLength
	topic2 := mockTopic(s)
	topic2.Id = 2
	topic2.EpochLength = 15
	topic2.GroundTruthLag = topic2.EpochLength
	topic2.WorkerSubmissionWindow = topic2.EpochLength

	totalSumPreviousTopicWeights, err := s.emissionsKeeper.GetTotalSumPreviousTopicWeights(ctx)
	s.Require().NoError(err)
	s.Require().Equal(totalSumPreviousTopicWeights, alloraMath.ZeroDec(), "Total sum of previous topic weights at start should be zero")
	setTopicWeight(topic1.Id, 150, 10)
	err = s.emissionsKeeper.SetTopic(ctx, topic1.Id, topic1)
	s.Require().NoError(err)
	err = s.emissionsKeeper.ActivateTopic(ctx, topic1.Id)
	s.Require().NoError(err, "Activating topic should not fail")

	totalSumPreviousTopicWeights, err = s.emissionsKeeper.GetTotalSumPreviousTopicWeights(ctx)
	s.Require().NoError(err)
	s.Require().Equal(totalSumPreviousTopicWeights, alloraMath.MustNewDecFromString("0"), "Total sum of previous topic weights should still be 0 bc previous topic weight is not set")

	setTopicWeight(topic2.Id, 300, 10)
	err = s.emissionsKeeper.SetTopic(ctx, topic2.Id, topic2)
	s.Require().NoError(err)
	err = s.emissionsKeeper.ActivateTopic(ctx, topic2.Id)
	s.Require().NoError(err, "Activating topic should not fail")

	block := 10
	ctx = s.ctx.WithBlockHeight(int64(block))
	_, _, _, err = rewards.GetAndUpdateActiveTopicWeights(ctx, s.emissionsKeeper, int64(block))
	s.Require().NoError(err, "Activating topic should not fail")

	// Previous weights still not moved
	totalSumPreviousTopicWeights, err = s.emissionsKeeper.GetTotalSumPreviousTopicWeights(ctx)
	s.T().Logf("totalSumPreviousTopicWeights: %v", totalSumPreviousTopicWeights)
	s.Require().NoError(err)
	s.Require().Equal(totalSumPreviousTopicWeights, alloraMath.MustNewDecFromString("0"), "Total sum of previous topic weights should not be 0 after settings topic weights")

	previousTopicWeights, _, err := s.emissionsKeeper.GetPreviousTopicWeight(ctx, topic1.Id)
	s.T().Logf("topic1 previousTopicWeights: %v", previousTopicWeights)
	s.Require().NoError(err)
	s.Require().Equal(previousTopicWeights, alloraMath.MustNewDecFromString("0"), "Previous topic weights should still be 0 after settings topic weights")

	block = 16
	ctx = s.ctx.WithBlockHeight(int64(block))
	_, _, _, err = rewards.GetAndUpdateActiveTopicWeights(ctx, s.emissionsKeeper, int64(block))
	s.Require().NoError(err, "Activating topic should not fail")

	activeTopics, err := s.emissionsKeeper.GetActiveTopicIdsAtBlock(ctx, 31)
	s.Require().NoError(err, "Fetching active topics should not produce an error")
	s.Require().Equal(2, len(activeTopics.TopicIds), "Should retrieve exactly two active topics")

	err = s.emissionsKeeper.SetParams(ctx, params)
	s.Require().NoError(err)

	block = 31
	ctx = s.ctx.WithBlockHeight(int64(block))
	_, _, _, err = rewards.GetAndUpdateActiveTopicWeights(ctx, s.emissionsKeeper, int64(block))
	s.Require().NoError(err, "Activating topic should not fail")

	activeTopics, err = s.emissionsKeeper.GetActiveTopicIdsAtBlock(ctx, 46)
	s.Require().NoError(err, "Fetching active topics should not produce an error")
	s.Require().Equal(1, len(activeTopics.TopicIds), "Should retrieve exactly one active topics")
}

func (s *RewardsTestSuite) TestGetRewardAndRemovedRewardableTopics() {
	block := int64(1)
	s.ctx = s.ctx.WithBlockHeight(block)

	s.SetParamsForTest()

	reputerIndexes := s.returnIndexes(0, 3)
	workerIndexes := s.returnIndexes(3, 3)
	// setup topics
	stake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())

	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	epochLength := int64(100)
	topicId0 := s.setUpTopicWithEpochLength(block, workerIndexes, reputerIndexes, stake, alphaRegret, epochLength)
	//topicId1 := s.setUpTopicWithEpochLength(block, workerAddrs, reputerAddrs, stake, alphaRegret, epochLength)
	//
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

	// mint some rewards to give out
	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(100000))

	// Move to end of this epoch block
	nextBlock, _, err := s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, topicId0)
	s.Require().NoError(err)
	block = nextBlock
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(block)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)

	topic0, err := s.emissionsKeeper.GetTopic(s.ctx, topicId0)
	s.Require().NoError(err)

	// Insert inference
	inferenceBundles := generateSimpleWorkerDataBundles(s, topicId0, topic0.EpochLastEnded, block, workerValues, workerIndexes)
	for _, payload := range inferenceBundles {
		s.RegisterAllWorkersOfPayload(topicId0, payload)
		_, err = s.msgServer.InsertWorkerPayload(s.ctx, &types.InsertWorkerPayloadRequest{
			Sender:           payload.Worker,
			WorkerDataBundle: payload,
		})
		s.Require().NoError(err)
	}

	// Advance to close the window
	block = block + topic0.WorkerSubmissionWindow
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(block)

	// EndBlock closes the  worker nonce
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)

	// Generate loss data
	lossBundles := generateSimpleLossBundles(
		s,
		topicId0,
		topic0.EpochLastEnded,
		workerValues,
		reputerValues,
		s.addrs[workerIndexes[0]],
		"0.1",
		"0.1",
	)

	// Run endBlock at 201, epoch end block - important bc of the topic in/activation
	block = block + (topic0.GroundTruthLag - topic0.WorkerSubmissionWindow)
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(block)
	s.T().Logf("****  Moved to next block %d", block)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)

	// Insert reputation
	block = block + 1
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(block)
	for _, payload := range lossBundles.ReputerValueBundles {
		s.RegisterAllReputersOfPayload(topicId0, payload)
		_, err = s.msgServer.InsertReputerPayload(s.ctx, &types.InsertReputerPayloadRequest{
			Sender:             payload.ValueBundle.Reputer,
			ReputerValueBundle: payload,
		})
		s.Require().NoError(err)
	}

	block = block + 1
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(block)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)

	// Move to next end of epoch
	nextBlock, _, err = s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, topicId0)
	s.Require().NoError(err)
	block = nextBlock
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(block)
	// EndBlock for closes the reputer nonce & add rewardable nonce
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)

	totalSumPreviousTopicWeights, err := s.emissionsKeeper.GetTotalSumPreviousTopicWeights(s.ctx)
	s.T().Logf("totalSumPreviousTopicWeights: %v", totalSumPreviousTopicWeights)
	s.Require().NoError(err)
	s.Require().NotEqual(totalSumPreviousTopicWeights, alloraMath.MustNewDecFromString("0"), "Total sum of previous topic weights should not be 0 after endBlocker topic weights")
}

func (s *RewardsTestSuite) TestPreviousTopicWeightsAfterInactivation() {
	block := int64(1)
	s.ctx = s.ctx.WithBlockHeight(block)

	s.SetParamsForTest()

	reputerIndexes := s.returnIndexes(0, 3)
	workerIndexes := s.returnIndexes(3, 3)
	stake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())

	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	epochLength := int64(100)
	topicId := s.setUpTopicWithEpochLength(block, workerIndexes, reputerIndexes, stake, alphaRegret, epochLength)

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

	s.MintTokensToModule(types.AlloraRewardsAccountName, cosmosMath.NewInt(100000))
	totalSumPreviousTopicWeights, err := s.emissionsKeeper.GetTotalSumPreviousTopicWeights(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(totalSumPreviousTopicWeights.String(), alloraMath.ZeroDec().String(), "Total sum of previous topic weights should be zero on start")

	// Move to end of this epoch block
	nextBlock, _, err := s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, topicId)
	s.Require().NoError(err)
	block = nextBlock
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(block)
	s.T().Logf("****  Moved to next block %d", block)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)

	topicPreviousWeight, noPrior, err := s.emissionsKeeper.GetPreviousTopicWeight(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(topicPreviousWeight.Equal(alloraMath.ZeroDec()), "Previous topic weight should be zero after endBlocker")
	s.Require().Equal(noPrior, false, "A prior weight should have been set")

	totalSumPreviousTopicWeights, err = s.emissionsKeeper.GetTotalSumPreviousTopicWeights(s.ctx)
	s.T().Logf("totalSumPreviousTopicWeights: %v", totalSumPreviousTopicWeights)
	s.Require().NoError(err)
	s.Require().NotEqual(totalSumPreviousTopicWeights.String(), alloraMath.ZeroDec().String(), "Total sum of previous topic weights should be zero after endBlocker")
	// At this point, topic total weight should be equal to the topic's previous weight
	s.Require().True(topicPreviousWeight.Equal(totalSumPreviousTopicWeights), "Topic total weight should be equal to the topic's previous weight")

	topic, err := s.emissionsKeeper.GetTopic(s.ctx, topicId)
	s.Require().NoError(err)

	// Insert inference
	inferenceBundles := generateSimpleWorkerDataBundles(s, topicId, topic.EpochLastEnded, block, workerValues, workerIndexes)
	for _, payload := range inferenceBundles {
		s.RegisterAllWorkersOfPayload(topicId, payload)
		_, err = s.msgServer.InsertWorkerPayload(s.ctx, &types.InsertWorkerPayloadRequest{
			Sender:           payload.Worker,
			WorkerDataBundle: payload,
		})
		s.Require().NoError(err)
	}

	// Advance to close the worker window
	block = block + topic.WorkerSubmissionWindow
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(block)
	s.T().Logf("****  Moved to next block %d", block)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)

	// Run endBlock at 201, epoch end block - important bc of the topic in/activation
	block = block + (topic.GroundTruthLag - topic.WorkerSubmissionWindow)
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(block)
	s.T().Logf("****  Moved to next block %d", block)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)

	// Run block at 202, epoch end block
	block = block + 1
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(block)
	s.T().Logf("****  Moved to next block %d", block)

	// Generate and insert loss data
	lossBundles := generateSimpleLossBundles(
		s,
		topicId,
		topic.EpochLastEnded,
		workerValues,
		reputerValues,
		s.addrs[workerIndexes[0]],
		"0.1",
		"0.1",
	)
	for _, payload := range lossBundles.ReputerValueBundles {
		s.RegisterAllReputersOfPayload(topicId, payload)
		_, err = s.msgServer.InsertReputerPayload(s.ctx, &types.InsertReputerPayloadRequest{
			Sender:             payload.ValueBundle.Reputer,
			ReputerValueBundle: payload,
		})
		s.Require().NoError(err)
	}
	s.T().Logf("Inserted loss data for topic %d at block %d", topicId, block)

	// Move to next end of epoch
	nextBlock, _, err = s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, topicId)
	s.Require().NoError(err)
	block = nextBlock
	s.ctx = sdk.UnwrapSDKContext(s.ctx).WithBlockHeight(block)
	s.T().Logf("****  Moved to next block %d", block)
	err = s.emissionsAppModule.EndBlock(s.ctx)
	s.Require().NoError(err)

	// Check previousTopicWeights and totalPreviousTopicWeights
	previousTopicWeight, _, err := s.emissionsKeeper.GetPreviousTopicWeight(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(alloraMath.ZeroDec().Equal(previousTopicWeight), "Previous topic weight should not be zero after endBlocker")

	totalSumPreviousTopicWeights, err = s.emissionsKeeper.GetTotalSumPreviousTopicWeights(s.ctx)
	s.Require().NoError(err)
	s.Require().True(previousTopicWeight.Equal(totalSumPreviousTopicWeights), "Total sum of previous topic weights should not be zero after endBlocker")

	// Inactivate the topic
	err = s.emissionsKeeper.InactivateTopic(s.ctx, topicId)
	s.Require().NoError(err)

	// Check previousTopicWeights and totalPreviousTopicWeights after inactivation
	inactivePreviousTopicWeight, _, err := s.emissionsKeeper.GetPreviousTopicWeight(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(previousTopicWeight, inactivePreviousTopicWeight, "Previous topic weight should remain unchanged after inactivation")
	s.Require().False(alloraMath.ZeroDec().Equal(inactivePreviousTopicWeight), "Previous topic weight should not be zero after inactivation")

	inactiveTotalSumPreviousTopicWeights, err := s.emissionsKeeper.GetTotalSumPreviousTopicWeights(s.ctx)
	s.Require().NoError(err)
	s.Require().True(alloraMath.ZeroDec().Equal(inactiveTotalSumPreviousTopicWeights), "Total sum of previous topic weights should be zero after inactivation")

	// Reactivate the topic
	err = s.emissionsKeeper.ActivateTopic(s.ctx, topicId)
	s.Require().NoError(err)

	// Check previousTopicWeights and totalPreviousTopicWeights after reactivation
	reactivePreviousTopicWeight, _, err := s.emissionsKeeper.GetPreviousTopicWeight(s.ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(previousTopicWeight, reactivePreviousTopicWeight, "Previous topic weight should remain unchanged after reactivation")
	s.Require().False(alloraMath.ZeroDec().Equal(reactivePreviousTopicWeight), "Previous topic weight should not be zero after reactivation")

	reactiveTotalSumPreviousTopicWeights, err := s.emissionsKeeper.GetTotalSumPreviousTopicWeights(s.ctx)
	s.Require().NoError(err)
	s.Require().True(previousTopicWeight.Equal(reactiveTotalSumPreviousTopicWeights), "Total sum of previous topic weights should be equal to previous topic weight after reactivation")
}
