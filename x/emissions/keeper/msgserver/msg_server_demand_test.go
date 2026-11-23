package msgserver_test

import (
	cosmosMath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
)

func (s *MsgServerTestSuite) ecosystemBalance(ctx sdk.Context) cosmosMath.Int {
	addr := authtypes.NewModuleAddress(minttypes.EcosystemModuleName)
	return s.BankKeeper().GetBalance(ctx, addr, params.DefaultBondDenom).Amount
}

func fundTopicEvents(ctx sdk.Context) []*types.EventFundTopic {
	out := make([]*types.EventFundTopic, 0)
	for _, ev := range ctx.EventManager().Events() {
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		if err != nil {
			continue
		}
		if e, ok := msg.(*types.EventFundTopic); ok {
			out = append(out, e)
		}
	}
	return out
}

func topicStatusEvents(ctx sdk.Context) []*types.EventTopicStatusChanged {
	out := make([]*types.EventTopicStatusChanged, 0)
	for _, ev := range ctx.EventManager().Events() {
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		if err != nil {
			continue
		}
		if e, ok := msg.(*types.EventTopicStatusChanged); ok {
			out = append(out, e)
		}
	}
	return out
}

// firstFundAndStatus indexes the first fund event and the first status event
// in emission order. A zero index means that event was absent.
func firstFundAndStatus(ctx sdk.Context) (fund *types.EventFundTopic, fundAt int, status *types.EventTopicStatusChanged, statusAt int) {
	seq := 0
	for _, ev := range ctx.EventManager().Events() {
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		if err != nil {
			continue
		}
		seq++
		switch e := msg.(type) {
		case *types.EventFundTopic:
			if fund == nil {
				fund = e
				fundAt = seq
			}
		case *types.EventTopicStatusChanged:
			if status == nil {
				status = e
				statusAt = seq
			}
		}
	}
	return fund, fundAt, status, statusAt
}

func (s *MsgServerTestSuite) TestFundTopicSimple() {
	senderAddr := s.Addrs(0)
	topic := uint64(1)
	// put some stake in the topic
	err := s.StakingKeeper().AddReputerStake(s.Ctx(), topic, s.AddrsStr(1), cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.TopicKeeper().InactivateTopic(s.Ctx(), topic, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().NoError(err)
	var initialStake int64 = 1000
	initialStakeCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, cosmosMath.NewInt(initialStake)))
	err = s.BankKeeper().MintCoins(s.Ctx(), types.AlloraStakingAccountName, initialStakeCoins)
	s.Require().NoError(err)
	err = s.BankKeeper().SendCoinsFromModuleToAccount(s.Ctx(), types.AlloraStakingAccountName, senderAddr, initialStakeCoins)
	s.Require().NoError(err)

	params, err := s.ParamsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err, "GetParams should not return an error")
	topicWeightBefore, feeRevBefore, _, err := s.TopicKeeper().GetCurrentTopicWeight(
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
	res, err := s.TopicKeeper().IsTopicActive(s.Ctx(), topic)
	s.Require().NoError(err)
	s.Require().Equal(true, res, "TopicId is not activated")
	// check that the topic fee revenue has been updated
	topicWeightAfter, feeRevAfter, _, err := s.TopicKeeper().GetCurrentTopicWeight(
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
	err := s.StakingKeeper().AddReputerStake(s.Ctx(), topic1, reputer, cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.TopicKeeper().InactivateTopic(s.Ctx(), topic1, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().NoError(err)
	err = s.StakingKeeper().AddReputerStake(s.Ctx(), topic2, reputer, cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.TopicKeeper().InactivateTopic(s.Ctx(), topic2, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec(), alloraMath.ZeroDec())
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
	res, err := s.TopicKeeper().IsTopicActive(s.Ctx(), topic1)
	s.Require().NoError(err)
	s.Require().Equal(true, res, "TopicId is not activated")

	params, err := s.ParamsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err, "GetParams should not return an error")

	// check that the topic fee revenue has been updated
	topicWeight, _, _, err := s.TopicKeeper().GetCurrentTopicWeight(
		s.Ctx(),
		topic1,
		10800,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)

	topic2Weight, _, _, err := s.TopicKeeper().GetCurrentTopicWeight(
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
	err := s.StakingKeeper().AddReputerStake(s.Ctx(), topic1, reputer, cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.TopicKeeper().InactivateTopic(s.Ctx(), topic1, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec(), alloraMath.ZeroDec())
	s.Require().NoError(err)

	err = s.StakingKeeper().AddReputerStake(s.Ctx(), topic2, reputer, cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	err = s.TopicKeeper().InactivateTopic(s.Ctx(), topic2, types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_UNSPECIFIED, alloraMath.ZeroDec(), alloraMath.ZeroDec(), alloraMath.ZeroDec())
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
	res, err := s.TopicKeeper().IsTopicActive(s.Ctx(), topic1)
	s.Require().NoError(err)
	s.Require().Equal(true, res, "Topic1 is not activated")

	res2, err := s.TopicKeeper().IsTopicActive(s.Ctx(), topic2)
	s.Require().NoError(err)
	s.Require().Equal(true, res2, "Topic2 is not activated")

	// Get params for weight calculation
	params, err := s.ParamsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err, "GetParams should not return an error")

	// Get weights for both topics
	topicWeight1, _, _, err := s.TopicKeeper().GetCurrentTopicWeight(
		s.Ctx(),
		topic1,
		epochLength1,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)

	topicWeight2, _, _, err := s.TopicKeeper().GetCurrentTopicWeight(
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
	err = s.TopicKeeper().SetPreviousTopicWeight(s.Ctx(), topic1, alloraMath.MustNewDecFromString("100"))
	s.Require().NoError(err)
	err = s.TopicKeeper().SetPreviousTopicWeight(s.Ctx(), topic2, alloraMath.MustNewDecFromString("100"))
	s.Require().NoError(err)

	// Recalculate having set previous topic weights
	topicWeight1, _, _, err = s.TopicKeeper().GetCurrentTopicWeight(
		s.Ctx(),
		topic1,
		epochLength1,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	s.Require().NoError(err)

	topicWeight2, _, _, err = s.TopicKeeper().GetCurrentTopicWeight(
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

// TestFundTopicZeroAmountStillActivatesWeightSufficientTopic pins that a
// zero-amount funding skips the bank transfer and adds zero to the current
// fee revenue. The fund event is still emitted for that unchanged total, and
// activation still runs because it depends on the topic's weight.
func (s *MsgServerTestSuite) TestFundTopicZeroAmountStillActivatesWeightSufficientTopic() {
	ctx := s.Ctx()
	topicId := uint64(1)

	// Give the topic enough weight to clear the activation threshold, directly
	// at the keeper so no activation trigger runs in setup. A topic with zero
	// fee revenue has zero weight, so both stake and revenue are needed.
	err := s.StakingKeeper().AddReputerStake(ctx, topicId, s.AddrsStr(1), cosmosMath.NewInt(500000))
	s.Require().NoError(err)
	_, err = s.TopicKeeper().AddTopicFeeRevenue(ctx, topicId, cosmosMath.NewInt(500000))
	s.Require().NoError(err)

	active, err := s.TopicKeeper().IsTopicActive(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(active, "topic must start inactive")

	feeRevenueBefore, err := s.TopicKeeper().GetTopicFeeRevenue(ctx, topicId)
	s.Require().NoError(err)
	sender := s.Addrs(0)
	balanceBefore := s.BankKeeper().GetBalance(ctx, sender, params.DefaultBondDenom)

	em := sdk.NewEventManager()
	callCtx := ctx.WithEventManager(em)
	_, err = s.EmissionsMsgServer().FundTopic(callCtx, &types.FundTopicRequest{
		Sender:  s.AddrsStr(0),
		TopicId: topicId,
		Amount:  cosmosMath.ZeroInt(),
	})
	s.Require().NoError(err)

	fund, fundAt, status, statusAt := firstFundAndStatus(callCtx)
	s.Require().Len(fundTopicEvents(callCtx), 1)
	s.Require().NotNil(fund)
	s.Require().Equal(topicId, fund.TopicId)
	s.Require().Equal(s.AddrsStr(0), fund.Funder)
	s.Require().True(fund.Amount.IsZero())
	s.Require().True(feeRevenueBefore.Equal(fund.NewRevenue))
	s.Require().Equal(types.TopicFundSource_TOPIC_FUND_SOURCE_TOPIC_FUNDING, fund.Source)
	s.Require().Len(topicStatusEvents(callCtx), 1)
	s.Require().NotNil(status)
	s.Require().Less(fundAt, statusAt, "the credit event precedes activation")

	feeRevenueAfter, err := s.TopicKeeper().GetTopicFeeRevenue(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(feeRevenueBefore, feeRevenueAfter, "zero-amount funding must not change fee revenue")
	balanceAfter := s.BankKeeper().GetBalance(ctx, sender, params.DefaultBondDenom)
	s.Require().True(balanceBefore.Equal(balanceAfter), "zero-amount funding must not move coins")

	active, err = s.TopicKeeper().IsTopicActive(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(active, "zero-amount funding must still attempt activation")
}

// A funding that activates a topic into a free slot reports ACTIVATED_BY_FUNDING
// with the topic's weight measured against the minimum.
func (s *MsgServerTestSuite) TestFundTopicActivationEmitsActivatedByFunding() {
	topicId := uint64(1)
	funder := s.Addrs(0)
	amount := cosmosMath.NewInt(1000)
	s.Require().NoError(s.StakingKeeper().AddReputerStake(s.Ctx(), topicId, s.AddrsStr(1), cosmosMath.NewInt(500000)))
	s.MintTokensToAddress(funder, amount)

	revenueBefore, err := s.TopicKeeper().GetTopicFeeRevenue(s.Ctx(), topicId)
	s.Require().NoError(err)
	ecosystemBefore := s.ecosystemBalance(s.Ctx())

	em := sdk.NewEventManager()
	callCtx := s.Ctx().WithEventManager(em)
	_, err = s.EmissionsMsgServer().FundTopic(callCtx, &types.FundTopicRequest{
		Sender:  funder.String(),
		TopicId: topicId,
		Amount:  amount,
	})
	s.Require().NoError(err)

	fund, fundAt, event, statusAt := firstFundAndStatus(callCtx)
	s.Require().Len(fundTopicEvents(callCtx), 1)
	s.Require().NotNil(fund)
	s.Require().Equal(topicId, fund.TopicId)
	s.Require().Equal(funder.String(), fund.Funder)
	s.Require().True(amount.Equal(fund.Amount))
	s.Require().True(revenueBefore.Add(amount).Equal(fund.NewRevenue))
	s.Require().Equal(types.TopicFundSource_TOPIC_FUND_SOURCE_TOPIC_FUNDING, fund.Source)
	s.Require().Len(topicStatusEvents(callCtx), 1)
	s.Require().NotNil(event)
	s.Require().Less(fundAt, statusAt, "the credit event precedes activation")
	s.Require().True(s.ecosystemBalance(s.Ctx()).Equal(ecosystemBefore.Add(amount)))

	moduleParams, err := s.ParamsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	weight, err := s.TopicKeeper().GetTopicWeightFromTopicId(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().False(weight.Equal(moduleParams.MinTopicWeight), "the weights must differ for the assertions to tell them apart")

	s.Require().Equal(topicId, event.TopicId)
	s.Require().True(event.IsActive)
	s.Require().Equal(types.TopicStatusChangeReason_TOPIC_STATUS_CHANGE_REASON_ACTIVATED_BY_FUNDING, event.Reason)
	s.Require().True(weight.Equal(event.Weight), "expected weight %s, got %s", weight.String(), event.Weight.String())
	s.Require().True(moduleParams.MinTopicWeight.Equal(event.MinWeight), "expected min %s, got %s", moduleParams.MinTopicWeight.String(), event.MinWeight.String())
	s.Require().True(moduleParams.MinTopicWeight.Equal(event.ThresholdWeight), "expected threshold %s, got %s", moduleParams.MinTopicWeight.String(), event.ThresholdWeight.String())
}
