package rewards_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *RewardsTestSuite) TestGetAndUpdateActiveTopicWeights() {
	ctx := s.ctx
	maxActiveTopicsNum := uint64(2)
	params := types.Params{
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

	pagination := &types.SimpleCursorPaginationRequest{
		Key:   nil,
		Limit: 2,
	}
	activeTopics, _, err := s.emissionsKeeper.GetIdsActiveTopicAtBlock(ctx, 31, pagination)
	s.Require().NoError(err, "Fetching active topics should not produce an error")
	s.Require().Equal(2, len(activeTopics), "Should retrieve exactly one active topics")

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

	pagination = &types.SimpleCursorPaginationRequest{
		Key:   nil,
		Limit: 2,
	}
	activeTopics, _, err = s.emissionsKeeper.GetIdsActiveTopicAtBlock(ctx, 46, pagination)
	s.Require().NoError(err, "Fetching active topics should not produce an error")
	s.Require().Equal(1, len(activeTopics), "Should retrieve exactly one active topics")
}
