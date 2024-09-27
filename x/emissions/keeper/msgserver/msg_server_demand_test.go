package msgserver_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
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
	)
	s.Require().NoError(err)

	topic2Weight, _, err := s.emissionsKeeper.GetCurrentTopicWeight(
		s.ctx,
		r2.TopicId,
		10800,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
	)
	s.Require().NoError(err)

	s.Require().Equal(topic2Weight.Gt(topicWeight), true, "Topic1 weight should be greater than Topic2 weight")
}
