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
	s.Require().Equal(ctx.BlockHeight(), epoch.StartBlockHeight)
	s.Require().Equal(types.Nonce{BlockHeight: ctx.BlockHeight()}, epoch.LegacyNonce())
	s.Require().Equal(start, epoch.WorkerSubmissionWindow.OpenAt)
	s.Require().Equal(start.Add(10*time.Second), epoch.WorkerSubmissionWindow.CloseAt)

	// ExtraLag for GroundTruthLag=150, EpochLength=100 is 50 → reputer opens at start+200s.
	s.Require().Equal(int64(50), types.TopicExtraLag(types.Topic{
		EpochLength:    100,
		GroundTruthLag: 150,
	}))
	s.Require().Equal(start.Add(200*time.Second), epoch.ReputerSubmissionWindow.OpenAt)
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

func (s *KeeperTestSuite) TestStartNewEpochRecordsStartBlockHeightBridge() {
	ctx := s.Ctx().WithBlockHeight(1_234)
	topicId := s.CreateTopic(testutil.WithEpochLength(60), testutil.WithGroundTruthLag(60), testutil.WithWorkerSubmissionWindow(10))

	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, topicId))

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)

	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(int64(1_234), epoch.StartBlockHeight)
	s.Require().Equal(types.Nonce{BlockHeight: 1_234}, epoch.LegacyNonce())
	// Canonical id remains NonceV2 — not the start height payload.
	s.Require().NotEqual(uint64(1_234), epoch.Nonce.Payload())
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

func (s *KeeperTestSuite) TestInactivateTopicCancelsInFlightEpochs() {
	ctx := s.Ctx().WithBlockHeight(900)
	topicId := s.CreateTopic(testutil.WithEpochLength(60), testutil.WithGroundTruthLag(60), testutil.WithWorkerSubmissionWindow(10))

	s.Require().NoError(s.TopicKeeper().ActivateTopic(ctx, topicId))

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)

	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WORKER_SUBMISSION, epoch.State)
	legacy := epoch.LegacyNonce()

	workerUnfulfilled, err := s.NonceKeeper().IsWorkerNonceUnfulfilled(ctx, topicId, &legacy)
	s.Require().NoError(err)
	s.Require().True(workerUnfulfilled)

	taskIDSuffix := fmt.Sprintf(":%d-%d", topicId, lastNonce)
	_, err = s.SchedulerKeeper().GetTask(ctx, schedulertypes.TaskID(types.CloseEpochWorkerWindowTask+taskIDSuffix))
	s.Require().NoError(err, "lifecycle close-worker task should be scheduled")

	s.Require().NoError(s.TopicKeeper().InactivateTopic(ctx, topicId))

	_, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().ErrorIs(err, collections.ErrNotFound, "in-flight epoch must be cancelled and removed")

	workerUnfulfilled, err = s.NonceKeeper().IsWorkerNonceUnfulfilled(ctx, topicId, &legacy)
	s.Require().NoError(err)
	s.Require().False(workerUnfulfilled, "cancel must fulfill leftover worker nonce")

	for _, taskType := range []string{
		types.CloseEpochWorkerWindowTask,
		types.OpenEpochReputerWindowTask,
		types.CloseEpochReputerWindowTask,
		types.CompleteEpochTask,
	} {
		_, err = s.SchedulerKeeper().GetTask(ctx, schedulertypes.TaskID(taskType+taskIDSuffix))
		s.Require().ErrorIs(err, collections.ErrNotFound, "lifecycle task %s must be unscheduled", taskType)
	}

	periodicID := schedulertypes.TaskID(fmt.Sprintf("%s:%d", types.StartNewEpochTask, topicId))
	_, err = s.SchedulerKeeper().GetTask(ctx, periodicID)
	s.Require().ErrorIs(err, collections.ErrNotFound)
}

func (s *KeeperTestSuite) TestInactivateTopicCancelsMultipleInFlightEpochs() {
	ctx := s.Ctx().WithBlockHeight(1_000)
	topicId := s.CreateTopic(testutil.WithEpochLength(60), testutil.WithGroundTruthLag(60), testutil.WithWorkerSubmissionWindow(10))

	s.Require().NoError(s.TopicKeeper().ActivateTopic(ctx, topicId))
	firstNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)

	// A second StartNewEpoch while the first is still open (overlapping windows).
	ctx = ctx.WithBlockHeight(1_010)
	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, topicId))
	secondNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().NotEqual(firstNonce, secondNonce)

	_, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, firstNonce)
	s.Require().NoError(err)
	_, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, secondNonce)
	s.Require().NoError(err)

	s.Require().NoError(s.TopicKeeper().InactivateTopic(ctx, topicId))

	_, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, firstNonce)
	s.Require().ErrorIs(err, collections.ErrNotFound)
	_, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, secondNonce)
	s.Require().ErrorIs(err, collections.ErrNotFound)
}
