package keeper_test

import (
	cosmosMath "cosmossdk.io/math"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// TestTopicInvariantTotalSumPreviousTopicWeightsEqualActiveTopicsSum walks a topic through
// activation, inactivation and a stake removal while inactive, checking the invariant holds at
// every step, and then checks that a corrupted total is detected.
func (s *KeeperTestSuite) TestTopicInvariantTotalSumPreviousTopicWeightsEqualActiveTopicsSum() {
	ctx := s.Ctx()
	k := s.TopicKeeper()
	invariant := keeper.TopicInvariantTotalSumPreviousTopicWeightsEqualActiveTopicsSum(*s.EmissionsKeeper())
	reputerAddr := s.AddrsStr(0)
	stakeAmount := cosmosMath.NewInt(1000)
	epochLength := int64(100)

	params := types.DefaultParams()
	params.TopicRewardAlpha = alloraMath.MustNewDecFromString("0.5")
	params.TopicRewardStakeImportance = alloraMath.OneDec()
	params.TopicRewardFeeRevenueImportance = alloraMath.OneDec()
	params.MaxActiveTopicsPerBlock = 2
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	assertHolds := func(step string) {
		msg, broken := invariant(ctx)
		s.Require().False(broken, "invariant broken after %s: %s", step, msg)
	}

	assertHolds("empty state")

	stayingTopicId := s.CreateTopic(testutil.WithEpochLength(epochLength), testutil.WithWorkerSubmissionWindow(epochLength))
	churnedTopicId := s.CreateTopic(testutil.WithEpochLength(epochLength), testutil.WithWorkerSubmissionWindow(epochLength))
	for _, topicId := range []uint64{stayingTopicId, churnedTopicId} {
		err = k.ActivateTopic(ctx, topicId)
		s.Require().NoError(err)
		err = s.StakingKeeper().AddReputerStake(ctx, topicId, reputerAddr, stakeAmount)
		s.Require().NoError(err)
		err = k.AddTopicFeeRevenue(ctx, topicId, cosmosMath.NewInt(100))
		s.Require().NoError(err)
		weight, _, _, err := k.GetCurrentTopicWeight(
			ctx, topicId, epochLength,
			params.TopicRewardAlpha, params.TopicRewardStakeImportance, params.TopicRewardFeeRevenueImportance, params.BlocksPerMonth,
		)
		s.Require().NoError(err)
		err = k.SetPreviousTopicWeight(ctx, topicId, weight)
		s.Require().NoError(err)
	}
	assertHolds("two active topics with weights")

	err = k.InactivateTopic(ctx, churnedTopicId)
	s.Require().NoError(err)
	assertHolds("inactivation")

	moduleParams, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	endBlock := ctx.BlockHeight() + moduleParams.RemoveStakeDelayWindow
	err = s.StakingKeeper().SetStakeRemoval(ctx, types.StakeRemovalInfo{
		TopicId:               churnedTopicId,
		Reputer:               reputerAddr,
		Amount:                stakeAmount,
		BlockRemovalStarted:   ctx.BlockHeight(),
		BlockRemovalCompleted: endBlock,
	})
	s.Require().NoError(err)
	err = s.StakingKeeper().RemoveReputerStake(ctx, endBlock, churnedTopicId, reputerAddr, stakeAmount)
	s.Require().NoError(err)
	assertHolds("stake removal from inactive topic")

	err = k.ActivateTopic(ctx, churnedTopicId)
	s.Require().NoError(err)
	assertHolds("reactivation")

	// A drifted total must be reported.
	totalSum, err := k.GetTotalSumPreviousTopicWeights(ctx)
	s.Require().NoError(err)
	drifted, err := totalSum.Add(alloraMath.OneDec())
	s.Require().NoError(err)
	err = k.SetTotalSumPreviousTopicWeights(ctx, drifted)
	s.Require().NoError(err)
	_, broken := invariant(ctx)
	s.Require().True(broken, "a drifted total sum must break the invariant")
}
