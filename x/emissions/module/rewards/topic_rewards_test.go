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
		GroundTruthLag:           10800,
		PNorm:                    alloraMath.NewDecFromInt64(3),
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

	setTopicWeight(topic1.Id, 150, 10)
	err = s.emissionsKeeper.SetTopic(ctx, topic1.Id, topic1)
	s.Require().NoError(err)
	err = s.emissionsKeeper.ActivateTopic(ctx, topic1.Id)
	s.Require().NoError(err, "Activating topic should not fail")

	setTopicWeight(topic2.Id, 300, 10)
	err = s.emissionsKeeper.SetTopic(ctx, topic2.Id, topic2)
	s.Require().NoError(err)
	err = s.emissionsKeeper.ActivateTopic(ctx, topic2.Id)
	s.Require().NoError(err, "Activating topic should not fail")

	block := 10
	ctx = s.ctx.WithBlockHeight(int64(block))
	_, _, _, err = rewards.GetAndUpdateActiveTopicWeights(ctx, s.emissionsKeeper, int64(block))
	s.Require().NoError(err, "Activating topic should not fail")
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

	// Insert reputation
	block = block + topic0.GroundTruthLag
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
}
