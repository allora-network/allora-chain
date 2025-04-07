package msgserver_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *MsgServerTestSuite) TestFundTopicSimple() {
	senderAddr := s.addrs[0]
	sender := s.addrsStr[0]
	topic := s.CreateOneTopic()
	// put some stake in the topic
	err := s.emissionsKeeper.AddReputerStake(s.ctx, topic.Id, s.addrsStr[1], cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.emissionsKeeper.InactivateTopic(s.ctx, topic.Id)
	s.Require().NoError(err)
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	err = s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.Require().NoError(err)
	err = s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	s.Require().NoError(err)
	r := types.FundTopicRequest{
		Sender:  sender,
		TopicId: topic.Id,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err, "GetParams should not return an error")
	topicWeightBefore, feeRevBefore, err := s.emissionsKeeper.GetCurrentTopicWeight(
		s.ctx,
		r.TopicId,
		10800,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)
	response, err := s.msgServer.FundTopic(s.ctx, &r)
	s.Require().NoError(err, "RequestInference should not return an error")
	s.Require().NotNil(response, "Response should not be nil")

	// Check if the topic is activated
	res, err := s.emissionsKeeper.IsTopicActive(s.ctx, r.TopicId)
	s.Require().NoError(err)
	s.Require().Equal(true, res, "TopicId is not activated")
	// check that the topic fee revenue has been updated
	topicWeightAfter, feeRevAfter, err := s.emissionsKeeper.GetCurrentTopicWeight(
		s.ctx,
		r.TopicId,
		10800,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)
	s.Require().True(feeRevAfter.GT(feeRevBefore), "Topic fee revenue should be greater after funding the topic")
	s.Require().True(topicWeightAfter.Gt(topicWeightBefore), "Topic weight should be greater after funding the topic")
}

func (s *MsgServerTestSuite) TestHighWeightForHighFundedTopic() {
	senderAddr := s.addrs[0]
	sender := s.addrsStr[0]
	reputer := s.addrsStr[1]
	topic1 := s.CreateOneTopic()
	topic2 := s.CreateCustomEpochTopic(10900)
	// put some stake in the topic
	err := s.emissionsKeeper.AddReputerStake(s.ctx, topic1.Id, reputer, cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.emissionsKeeper.InactivateTopic(s.ctx, topic1.Id)
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddReputerStake(s.ctx, topic2.Id, reputer, cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.emissionsKeeper.InactivateTopic(s.ctx, topic2.Id)
	s.Require().NoError(err)
	var initialStake int64 = 1000
	var initialStake2 int64 = 10000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake+initialStake2)))
	err = s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.Require().NoError(err)
	err = s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	s.Require().NoError(err)
	r := types.FundTopicRequest{
		Sender:  sender,
		TopicId: topic1.Id,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	r2 := types.FundTopicRequest{
		Sender:  sender,
		TopicId: topic2.Id,
		Amount:  cosmosMath.NewInt(initialStake2),
	}
	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err, "GetParams should not return an error")

	response, err := s.msgServer.FundTopic(s.ctx, &r)
	s.Require().NoError(err, "RequestInference should not return an error")
	s.Require().NotNil(response, "Response should not be nil")

	response2, err := s.msgServer.FundTopic(s.ctx, &r2)
	s.Require().NoError(err, "RequestInference should not return an error")
	s.Require().NotNil(response2, "Response should not be nil")

	// Check if the topic is activated
	res, err := s.emissionsKeeper.IsTopicActive(s.ctx, r.TopicId)
	s.Require().NoError(err)
	s.Require().Equal(true, res, "TopicId is not activated")
	// check that the topic fee revenue has been updated
	topicWeight, _, err := s.emissionsKeeper.GetCurrentTopicWeight(
		s.ctx,
		r.TopicId,
		10800,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)

	topic2Weight, _, err := s.emissionsKeeper.GetCurrentTopicWeight(
		s.ctx,
		r2.TopicId,
		10800,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)

	s.Require().Equal(topic2Weight.Gt(topicWeight), true, "Topic1 weight should be greater than Topic2 weight")
}

func (s *MsgServerTestSuite) TestTopicWeightDoesNotChangeWithDifferentEpochLengths() {
	senderAddr := s.addrs[0]
	sender := s.addrsStr[0]
	reputer := s.addrsStr[1]

	// Create two topics with different epoch lengths
	epochLength1 := int64(125)
	epochLength2 := int64(50)
	topic1 := s.CreateCustomEpochTopic(epochLength1) // Longer epoch
	topic2 := s.CreateCustomEpochTopic(epochLength2) // Shorter epoch

	// Put same stake in both topics
	err := s.emissionsKeeper.AddReputerStake(s.ctx, topic1.Id, reputer, cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.emissionsKeeper.InactivateTopic(s.ctx, topic1.Id)
	s.Require().NoError(err)

	err = s.emissionsKeeper.AddReputerStake(s.ctx, topic2.Id, reputer, cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.emissionsKeeper.InactivateTopic(s.ctx, topic2.Id)
	s.Require().NoError(err)

	// Set up funding amounts
	var initialStake int64 = 10000
	var initialStake2 int64 = 10000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake+initialStake2)))
	err = s.bankKeeper.MintCoins(s.ctx, types.AlloraStakingAccountName, initialStakeCoins)
	s.Require().NoError(err)
	err = s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	s.Require().NoError(err)

	// Create funding requests
	r := types.FundTopicRequest{
		Sender:  sender,
		TopicId: topic1.Id,
		Amount:  cosmosMath.NewInt(initialStake),
	}
	r2 := types.FundTopicRequest{
		Sender:  sender,
		TopicId: topic2.Id,
		Amount:  cosmosMath.NewInt(initialStake2),
	}

	// Get params for weight calculation
	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err, "GetParams should not return an error")

	// Fund both topics
	response, err := s.msgServer.FundTopic(s.ctx, &r)
	s.Require().NoError(err, "FundTopic should not return an error")
	s.Require().NotNil(response, "Response should not be nil")

	response2, err := s.msgServer.FundTopic(s.ctx, &r2)
	s.Require().NoError(err, "FundTopic should not return an error")
	s.Require().NotNil(response2, "Response should not be nil")

	// Check if both topics are activated
	res, err := s.emissionsKeeper.IsTopicActive(s.ctx, r.TopicId)
	s.Require().NoError(err)
	s.Require().Equal(true, res, "Topic1 is not activated")

	res2, err := s.emissionsKeeper.IsTopicActive(s.ctx, r2.TopicId)
	s.Require().NoError(err)
	s.Require().Equal(true, res2, "Topic2 is not activated")

	// Get weights for both topics
	topicWeight1, _, err := s.emissionsKeeper.GetCurrentTopicWeight(
		s.ctx,
		r.TopicId,
		epochLength1,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)

	topicWeight2, _, err := s.emissionsKeeper.GetCurrentTopicWeight(
		s.ctx,
		r2.TopicId,
		epochLength2,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)
	// Topics should have equal weights at this point because their previous topic weight is still zero
	s.Require().Equal(topicWeight2.Equal(topicWeight1), true, "Topic2 weight should be equal to Topic1 weight if no previous topic weight is set")

	// Setting previous topic weights
	err = s.emissionsKeeper.SetPreviousTopicWeight(s.ctx, r.TopicId, alloraMath.MustNewDecFromString("100"))
	s.Require().NoError(err)
	err = s.emissionsKeeper.SetPreviousTopicWeight(s.ctx, r2.TopicId, alloraMath.MustNewDecFromString("100"))
	s.Require().NoError(err)

	// Recalculate having set previous topic weights
	topicWeight1, _, err = s.emissionsKeeper.GetCurrentTopicWeight(
		s.ctx,
		r.TopicId,
		epochLength1,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)

	topicWeight2, _, err = s.emissionsKeeper.GetCurrentTopicWeight(
		s.ctx,
		r2.TopicId,
		epochLength2,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)
	// Topics should have equal weights because the target weight is not affected by the epoch length
	s.Require().Equal(topicWeight1.Gt(topicWeight2), true, "Topic1 weight should > Topic2 weight because prev topic weights are smaller than current ones and Topic1 has a longer epoch length")

}
