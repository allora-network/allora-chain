package rewards_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *RewardsTestSuite) TestGetAndUpdateActiveTopicWeights() {
	ctx := s.ctx
	maxActiveTopicsNum := uint64(2)
	params := types.Params{
		BlocksPerMonth:                  864000,
		MaxActiveTopicsPerBlock:         maxActiveTopicsNum,
		MaxPageLimit:                    uint64(100),
		TopicRewardAlpha:                alloraMath.MustNewDecFromString("0.5"),
		TopicRewardStakeImportance:      alloraMath.MustNewDecFromString("1"),
		TopicRewardFeeRevenueImportance: alloraMath.MustNewDecFromString("3"),
	}
	err := s.emissionsKeeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	setTopicWeight := func(topicId uint64, revenue, stake int64) {
		_ = s.emissionsKeeper.AddTopicFeeRevenue(ctx, topicId, cosmosMath.NewInt(revenue))
		_ = s.emissionsKeeper.SetTopicStake(ctx, topicId, cosmosMath.NewInt(stake))
	}

	ctx = s.ctx.WithBlockHeight(1)
	// Assume topic initially active
	topic1 := types.Topic{Id: 1, EpochLength: 15}
	topic2 := types.Topic{Id: 2, EpochLength: 15}

	setTopicWeight(topic1.Id, 10, 10)
	_ = s.emissionsKeeper.SetTopic(ctx, topic1.Id, topic1)
	err = s.emissionsKeeper.ActivateTopic(ctx, topic1.Id)
	s.Require().NoError(err, "Activating topic should not fail")

	setTopicWeight(topic2.Id, 30, 10)
	_ = s.emissionsKeeper.SetTopic(ctx, topic2.Id, topic2)
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

	params = types.Params{
		MaxActiveTopicsPerBlock:         maxActiveTopicsNum,
		MaxPageLimit:                    uint64(100),
		TopicRewardAlpha:                alloraMath.MustNewDecFromString("0.5"),
		TopicRewardStakeImportance:      alloraMath.MustNewDecFromString("1"),
		TopicRewardFeeRevenueImportance: alloraMath.MustNewDecFromString("3"),
		MinTopicWeight:                  alloraMath.MustNewDecFromString("10"),
	}
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

	reputerAddrs := s.returnAddresses(0, 3)

	workerAddrs := s.returnAddresses(3, 3)
	// setup topics
	stake := cosmosMath.NewInt(1000).Mul(inferencesynthesis.CosmosIntOneE18())

	alphaRegret := alloraMath.MustNewDecFromString("0.1")
	epochLength := int64(100)
	topicId0 := s.setUpTopicWithEpochLength(block, workerAddrs, reputerAddrs, stake, alphaRegret, epochLength)
	//topicId1 := s.setUpTopicWithEpochLength(block, workerAddrs, reputerAddrs, stake, alphaRegret, epochLength)
	//
	// setup values to be identical for both topics
	reputerValues := []TestWorkerValue{
		{Address: reputerAddrs[0], Value: "0.2"},
		{Address: reputerAddrs[1], Value: "0.2"},
		{Address: reputerAddrs[2], Value: "0.2"},
	}

	workerValues := []TestWorkerValue{
		{Address: workerAddrs[0], Value: "0.2"},
		{Address: workerAddrs[1], Value: "0.2"},
		{Address: workerAddrs[2], Value: "0.2"},
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
	inferenceBundles := GenerateSimpleWorkerDataBundles(s, topicId0, topic0.EpochLastEnded, block, workerValues, workerAddrs)
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
	lossBundles := GenerateSimpleLossBundles(
		s,
		topicId0,
		topic0.EpochLastEnded,
		block,
		workerValues,
		reputerValues,
		workerAddrs[0],
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
