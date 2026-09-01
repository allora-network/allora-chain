package keeper_test

import (
	"fmt"
	"time"

	"cosmossdk.io/collections"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
)

func (s *KeeperTestSuite) TestStartNewEpochOpensWorkerWindowAndSchedulesLaterTransitions() {
	ctx := s.Ctx()
	// GroundTruthLag must be >= EpochLength; ExtraLag pads when GTL % EpochLength != 0.
	topicId := s.CreateTopic(testutil.WithEpochLength(100), testutil.WithGroundTruthLag(150), testutil.WithWorkerSubmissionWindow(10))
	start := ctx.BlockTime()

	err := s.EmissionsKeeper().StartNewEpoch(ctx, topicId)
	s.Require().NoError(err)

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)

	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WORKER_SUBMISSION, epoch.State)
	s.Require().Equal(start, epoch.WorkerSubmissionWindow.OpenAt)
	s.Require().Equal(start.Add(10*time.Second), epoch.WorkerSubmissionWindow.CloseAt)

	// ExtraLag for GroundTruthLag=150, EpochLength=100 is 50 → open at GTL (150s), close at GTL+ExtraLag+EpochLength (300s).
	s.Require().Equal(int64(50), types.TopicExtraLag(types.Topic{
		EpochLength:    100,
		GroundTruthLag: 150,
	}))
	s.Require().Equal(start.Add(150*time.Second), epoch.ReputerSubmissionWindow.OpenAt)
	s.Require().Equal(start.Add(300*time.Second), epoch.ReputerSubmissionWindow.CloseAt)

	taskIDSuffix := fmt.Sprintf(":%d-%d", topicId, lastNonce)
	_, err = s.SchedulerKeeper().GetTask(ctx, schedulertypes.TaskID(types.OpenEpochWorkerWindowTask+taskIDSuffix))
	s.Require().ErrorIs(err, collections.ErrNotFound, "open-worker is applied immediately, not scheduled")

	closeWorker, err := s.SchedulerKeeper().GetTask(ctx, schedulertypes.TaskID(types.CloseEpochWorkerWindowTask+taskIDSuffix))
	s.Require().NoError(err)
	s.Require().Equal(epoch.WorkerSubmissionWindow.CloseAt, *closeWorker.ScheduledFor)

	openReputer, err := s.SchedulerKeeper().GetTask(ctx, schedulertypes.TaskID(types.OpenEpochReputerWindowTask+taskIDSuffix))
	s.Require().NoError(err)
	s.Require().Equal(epoch.ReputerSubmissionWindow.OpenAt, *openReputer.ScheduledFor)
}

func (s *KeeperTestSuite) TestActivateTopicStartsEpochAndPeriodicTask() {
	ctx := s.Ctx()
	topicId := s.CreateTopic(testutil.WithEpochLength(60), testutil.WithGroundTruthLag(60), testutil.WithWorkerSubmissionWindow(10))

	err := s.TopicKeeper().ActivateTopic(ctx, topicId)
	s.Require().NoError(err)

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(types.ZeroNonce().NextNonce(), lastNonce)

	periodicID := schedulertypes.TaskID(fmt.Sprintf("%s:%d", types.StartNewEpochTask, topicId))
	periodic, err := s.SchedulerKeeper().GetTask(ctx, periodicID)
	s.Require().NoError(err)
	s.Require().NotNil(periodic.Interval)
	s.Require().Equal(60*time.Second, *periodic.Interval)
	s.Require().Equal(ctx.BlockTime().Add(60*time.Second), *periodic.ScheduledFor)

	// Advance past ScheduledFor (due check uses BlockTime.After, so equality is not enough).
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(60*time.Second + time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	nextNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lastNonce.NextNonce(), nextNonce)
}

func (s *KeeperTestSuite) TestInactivateTopicCancelsPeriodicNewEpochTask() {
	ctx := s.Ctx()
	topicId := s.CreateTopic(testutil.WithEpochLength(60), testutil.WithGroundTruthLag(60), testutil.WithWorkerSubmissionWindow(10))

	s.Require().NoError(s.TopicKeeper().ActivateTopic(ctx, topicId))
	periodicID := schedulertypes.TaskID(fmt.Sprintf("%s:%d", types.StartNewEpochTask, topicId))
	_, err := s.SchedulerKeeper().GetTask(ctx, periodicID)
	s.Require().NoError(err)

	firstNonce, _, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)

	s.Require().NoError(s.TopicKeeper().InactivateTopic(ctx, topicId))
	_, err = s.SchedulerKeeper().GetTask(ctx, periodicID)
	s.Require().ErrorIs(err, collections.ErrNotFound)

	// Periodic must not create another epoch after inactivation.
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(60*time.Second + time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(firstNonce, lastNonce)
}
