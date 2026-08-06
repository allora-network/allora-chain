package keeper_test

import (
	"fmt"
	"time"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
)

func (s *KeeperTestSuite) TestArbitrateStartNewEpochPostponesLowerWeightTopics() {
	ctx := s.Ctx()
	params, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	// DefaultParams uses MaxActiveTopicsPerBlock=1; widen so both topics can activate.
	params.MaxActiveTopicsPerBlock = 2
	s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

	heavyTopic := s.CreateTopic(testutil.WithEpochLength(60), testutil.WithGroundTruthLag(60), testutil.WithWorkerSubmissionWindow(10))
	lightTopic := s.CreateTopic(testutil.WithEpochLength(60), testutil.WithGroundTruthLag(60), testutil.WithWorkerSubmissionWindow(10))

	s.Require().NoError(s.StakingKeeper().SetTopicStake(ctx, heavyTopic, cosmosMath.NewInt(1_000_000)))
	s.Require().NoError(s.TopicKeeper().AddTopicFeeRevenue(ctx, heavyTopic, cosmosMath.NewInt(1_000_000)))
	s.Require().NoError(s.StakingKeeper().SetTopicStake(ctx, lightTopic, cosmosMath.NewInt(1)))
	s.Require().NoError(s.TopicKeeper().AddTopicFeeRevenue(ctx, lightTopic, cosmosMath.NewInt(1)))

	s.Require().NoError(s.TopicKeeper().ActivateTopic(ctx, heavyTopic))
	s.Require().NoError(s.TopicKeeper().ActivateTopic(ctx, lightTopic))
	activeHeavy, err := s.TopicKeeper().IsTopicActive(ctx, heavyTopic)
	s.Require().NoError(err)
	s.Require().True(activeHeavy)
	activeLight, err := s.TopicKeeper().IsTopicActive(ctx, lightTopic)
	s.Require().NoError(err)
	s.Require().True(activeLight)

	// Tighten the shared per-block topic budget used by arbitration.
	params, err = s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	params.MaxActiveTopicsPerBlock = 1
	s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

	heavyNonceBefore, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, heavyTopic)
	s.Require().NoError(err)
	s.Require().True(found)
	lightNonceBefore, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, lightTopic)
	s.Require().NoError(err)
	s.Require().True(found)

	// Both periodic StartNewEpoch tasks become due.
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(60*time.Second + time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	heavyNonceAfter, _, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, heavyTopic)
	s.Require().NoError(err)
	lightNonceAfter, _, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, lightTopic)
	s.Require().NoError(err)

	s.Require().Equal(heavyNonceBefore.NextNonce(), heavyNonceAfter, "higher-weight topic should execute")
	s.Require().Equal(lightNonceBefore, lightNonceAfter, "lower-weight topic should be postponed")

	// Postponed task must still exist (no silent drop) and retry next BeginBlock.
	lightPeriodicID := schedulertypes.TaskID(fmt.Sprintf("%s:%d", types.StartNewEpochTask, lightTopic))
	lightTask, err := s.SchedulerKeeper().GetTask(ctx, lightPeriodicID)
	s.Require().NoError(err)
	s.Require().NotNil(lightTask.ScheduledFor)
	s.Require().NotNil(lightTask.Interval, "periodic interval must be preserved across arbitration reschedule")

	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	lightNonceRetried, _, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, lightTopic)
	s.Require().NoError(err)
	s.Require().Equal(lightNonceBefore.NextNonce(), lightNonceRetried, "postponed topic should run on retry")
}

func (s *KeeperTestSuite) TestArbitrateCompleteEpochPostponesLowerWeightTopics() {
	ctx := s.Ctx().WithBlockHeight(2_000)
	params, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	params.MaxActiveTopicsPerBlock = 1
	s.Require().NoError(s.ParamsKeeper().SetParams(ctx, params))

	heavyTopic := s.CreateTopic(testutil.WithEpochLength(100), testutil.WithGroundTruthLag(100), testutil.WithWorkerSubmissionWindow(10))
	lightTopic := s.CreateTopic(testutil.WithEpochLength(100), testutil.WithGroundTruthLag(100), testutil.WithWorkerSubmissionWindow(10))

	s.Require().NoError(s.StakingKeeper().SetTopicStake(ctx, heavyTopic, cosmosMath.NewInt(1_000_000)))
	s.Require().NoError(s.TopicKeeper().AddTopicFeeRevenue(ctx, heavyTopic, cosmosMath.NewInt(1_000_000)))
	s.Require().NoError(s.StakingKeeper().SetTopicStake(ctx, lightTopic, cosmosMath.NewInt(1)))
	s.Require().NoError(s.TopicKeeper().AddTopicFeeRevenue(ctx, lightTopic, cosmosMath.NewInt(1)))

	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, heavyTopic))
	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, lightTopic))

	heavyNonce, _, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, heavyTopic)
	s.Require().NoError(err)
	lightNonce, _, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, lightTopic)
	s.Require().NoError(err)

	heavyEpoch, err := s.EmissionsKeeper().GetEpoch(ctx, heavyTopic, heavyNonce)
	s.Require().NoError(err)
	lightEpoch, err := s.EmissionsKeeper().GetEpoch(ctx, lightTopic, lightNonce)
	s.Require().NoError(err)

	// Drive both epochs to PENDING_COMPLETION via scheduled lifecycle tasks.
	ctx = ctx.WithBlockTime(heavyEpoch.WorkerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))
	heavyEpoch, err = s.EmissionsKeeper().GetEpoch(ctx, heavyTopic, heavyNonce)
	s.Require().NoError(err)
	lightEpoch, err = s.EmissionsKeeper().GetEpoch(ctx, lightTopic, lightNonce)
	s.Require().NoError(err)

	ctx = ctx.WithBlockTime(heavyEpoch.ReputerSubmissionWindow.OpenAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))
	heavyEpoch, err = s.EmissionsKeeper().GetEpoch(ctx, heavyTopic, heavyNonce)
	s.Require().NoError(err)
	lightEpoch, err = s.EmissionsKeeper().GetEpoch(ctx, lightTopic, lightNonce)
	s.Require().NoError(err)

	// Close reputer + complete are both due at CloseAt; arbitration should allow only one complete.
	ctx = ctx.WithBlockTime(heavyEpoch.ReputerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	_, err = s.EmissionsKeeper().GetEpoch(ctx, heavyTopic, heavyNonce)
	s.Require().Error(err, "higher-weight complete should remove the epoch")
	_, err = s.EmissionsKeeper().GetEpoch(ctx, lightTopic, lightNonce)
	s.Require().NoError(err, "lower-weight complete should be postponed; epoch still present")
	lightEpoch, err = s.EmissionsKeeper().GetEpoch(ctx, lightTopic, lightNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_PENDING_COMPLETION, lightEpoch.State)

	completeID := schedulertypes.TaskID(fmt.Sprintf("%s:%d-%d", types.CompleteEpochTask, lightTopic, lightNonce))
	_, err = s.SchedulerKeeper().GetTask(ctx, completeID)
	s.Require().NoError(err, "postponed complete task must not be dropped")

	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))
	_, err = s.EmissionsKeeper().GetEpoch(ctx, lightTopic, lightNonce)
	s.Require().Error(err, "postponed complete should succeed on retry")
}
