package msgserver_test

import (
	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *MsgServerTestSuite) TestFundTopicSimple() {
	senderAddr := s.Addrs(0)
	topic := uint64(1)
	// put some stake in the topic
	err := s.EmissionsKeeper().AddReputerStake(s.Ctx(), topic, s.AddrsStr(1), cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.EmissionsKeeper().InactivateTopic(s.Ctx(), topic)
	s.Require().NoError(err)
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	err = s.BankKeeper().MintCoins(s.Ctx(), types.AlloraStakingAccountName, initialStakeCoins)
	s.Require().NoError(err)
	err = s.BankKeeper().SendCoinsFromModuleToAccount(s.Ctx(), types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	s.Require().NoError(err)

	params, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err, "GetParams should not return an error")
	topicWeightBefore, feeRevBefore, _, err := s.EmissionsKeeper().GetCurrentTopicWeight(
		s.Ctx(),
		topic,
		10800,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)
	s.FundTopic(topic, senderAddr, cosmosMath.NewInt(initialStake))

	// Check if the topic is activated
	res, err := s.EmissionsKeeper().IsTopicActive(s.Ctx(), topic)
	s.Require().NoError(err)
	s.Require().Equal(true, res, "TopicId is not activated")
	// check that the topic fee revenue has been updated
	topicWeightAfter, feeRevAfter, _, err := s.EmissionsKeeper().GetCurrentTopicWeight(
		s.Ctx(),
		topic,
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
	senderAddr := s.Addrs(0)
	reputer := s.AddrsStr(1)
	topic1 := s.CreateTopic()
	topic2 := s.CreateTopic(testutil.WithEpochLength(10900), testutil.WithGroundTruthLag(10900))

	// put some stake in the topic
	err := s.EmissionsKeeper().AddReputerStake(s.Ctx(), topic1, reputer, cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.EmissionsKeeper().InactivateTopic(s.Ctx(), topic1)
	s.Require().NoError(err)
	err = s.EmissionsKeeper().AddReputerStake(s.Ctx(), topic2, reputer, cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.EmissionsKeeper().InactivateTopic(s.Ctx(), topic2)
	s.Require().NoError(err)
	var initialStake int64 = 1000
	var initialStake2 int64 = 10000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake+initialStake2)))
	err = s.BankKeeper().MintCoins(s.Ctx(), types.AlloraStakingAccountName, initialStakeCoins)
	s.Require().NoError(err)
	err = s.BankKeeper().SendCoinsFromModuleToAccount(s.Ctx(), types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	s.Require().NoError(err)

	s.FundTopic(topic1, senderAddr, cosmosMath.NewInt(initialStake))
	s.FundTopic(topic2, senderAddr, cosmosMath.NewInt(initialStake2))

	// Check if the topic is activated
	res, err := s.EmissionsKeeper().IsTopicActive(s.Ctx(), topic1)
	s.Require().NoError(err)
	s.Require().Equal(true, res, "TopicId is not activated")

	params, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err, "GetParams should not return an error")

	// check that the topic fee revenue has been updated
	topicWeight, _, _, err := s.EmissionsKeeper().GetCurrentTopicWeight(
		s.Ctx(),
		topic1,
		10800,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)

	topic2Weight, _, _, err := s.EmissionsKeeper().GetCurrentTopicWeight(
		s.Ctx(),
		topic2,
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
	senderAddr := s.Addrs(0)
	reputer := s.AddrsStr(1)

	// Create two topics with different epoch lengths
	epochLength1 := int64(125)
	epochLength2 := int64(50)
	topic1 := s.CreateTopic(testutil.WithEpochLength(epochLength1)) // Longer epoch
	topic2 := s.CreateTopic(testutil.WithEpochLength(epochLength2)) // Shorter epoch

	// Put same stake in both topics
	err := s.EmissionsKeeper().AddReputerStake(s.Ctx(), topic1, reputer, cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.EmissionsKeeper().InactivateTopic(s.Ctx(), topic1)
	s.Require().NoError(err)

	err = s.EmissionsKeeper().AddReputerStake(s.Ctx(), topic2, reputer, cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.EmissionsKeeper().InactivateTopic(s.Ctx(), topic2)
	s.Require().NoError(err)

	// Set up funding amounts
	var initialStake int64 = 10000
	var initialStake2 int64 = 10000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake+initialStake2)))
	err = s.BankKeeper().MintCoins(s.Ctx(), types.AlloraStakingAccountName, initialStakeCoins)
	s.Require().NoError(err)
	err = s.BankKeeper().SendCoinsFromModuleToAccount(s.Ctx(), types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	s.Require().NoError(err)

	// Fund both topics
	s.FundTopic(topic1, senderAddr, cosmosMath.NewInt(initialStake))
	s.FundTopic(topic2, senderAddr, cosmosMath.NewInt(initialStake2))

	// Check if both topics are activated
	res, err := s.EmissionsKeeper().IsTopicActive(s.Ctx(), topic1)
	s.Require().NoError(err)
	s.Require().Equal(true, res, "Topic1 is not activated")

	res2, err := s.EmissionsKeeper().IsTopicActive(s.Ctx(), topic2)
	s.Require().NoError(err)
	s.Require().Equal(true, res2, "Topic2 is not activated")

	// Get params for weight calculation
	params, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err, "GetParams should not return an error")

	// Get weights for both topics
	topicWeight1, _, _, err := s.EmissionsKeeper().GetCurrentTopicWeight(
		s.Ctx(),
		topic1,
		epochLength1,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)

	topicWeight2, _, _, err := s.EmissionsKeeper().GetCurrentTopicWeight(
		s.Ctx(),
		topic2,
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
	err = s.EmissionsKeeper().SetPreviousTopicWeight(s.Ctx(), topic1, alloraMath.MustNewDecFromString("100"))
	s.Require().NoError(err)
	err = s.EmissionsKeeper().SetPreviousTopicWeight(s.Ctx(), topic2, alloraMath.MustNewDecFromString("100"))
	s.Require().NoError(err)

	// Recalculate having set previous topic weights
	topicWeight1, _, _, err = s.EmissionsKeeper().GetCurrentTopicWeight(
		s.Ctx(),
		topic1,
		epochLength1,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)

	topicWeight2, _, _, err = s.EmissionsKeeper().GetCurrentTopicWeight(
		s.Ctx(),
		topic2,
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
