package keeper_test

import (
	"fmt"
	"time"

	"cosmossdk.io/collections"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
)

func (s *KeeperTestSuite) TestEpochFSMHappyPathWithWorkerAndReputerSubmissions() {
	s.WithBlockHeight(5_000)
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(100),
		testutil.WithWorkerSubmissionWindow(10),
	)
	s.SetupParticipants(topicId, []int{0, 1}, false)
	s.SetupParticipants(topicId, []int{2, 3}, true)

	// Reputer stake activates the topic (StartNewEpoch + periodic). A second
	// StartNewEpoch at the same height would share LegacyNonce and make the
	// later close-worker task fail as already fulfilled.
	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().True(found, "reputer stake should activate and start an epoch")
	periodicID := schedulertypes.TaskID(fmt.Sprintf("%s:%d", types.StartNewEpochTask, topicId))
	s.Require().NoError(s.SchedulerKeeper().CancelTask(s.Ctx(), periodicID))
	epoch, err := s.EmissionsKeeper().GetEpoch(s.Ctx(), topicId, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WORKER_SUBMISSION, epoch.State)
	legacy := epoch.LegacyNonce()

	s.SetupInferences(topicId, legacy.BlockHeight, []int{0, 1})

	s.WithBlockTime(epoch.WorkerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(s.Ctx()))

	epoch, err = s.EmissionsKeeper().GetEpoch(s.Ctx(), topicId, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WAITING_GROUND_TRUTH, epoch.State)

	networkInferences, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topicId, legacy.BlockHeight)
	s.Require().NoError(err)
	s.Require().NotEmpty(networkInferences.InfererValues, "close worker must produce network inferences")

	s.WithBlockTime(epoch.ReputerSubmissionWindow.OpenAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(s.Ctx()))

	epoch, err = s.EmissionsKeeper().GetEpoch(s.Ctx(), topicId, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_REPUTER_SUBMISSION, epoch.State)

	s.Require().NoError(s.InsertReputerLossBundle(topicId, legacy.BlockHeight, []int{2, 3}))

	s.WithBlockTime(epoch.ReputerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(s.Ctx()))

	_, err = s.EmissionsKeeper().GetEpoch(s.Ctx(), topicId, lastNonce)
	s.Require().ErrorIs(err, collections.ErrNotFound, "completed epoch is removed from store")

	lossBundle, err := s.ReputerLossKeeper().GetNetworkLossBundleAtBlock(s.Ctx(), topicId, legacy.BlockHeight)
	s.Require().NoError(err)
	s.Require().NotEmpty(lossBundle.InfererValues, "close reputer must produce network losses")

	rewardable, err := s.TopicKeeper().IsRewardableTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().True(rewardable, "complete must mark topic rewardable")
}

func (s *KeeperTestSuite) TestEpochFSMHappyPathInitToCompletedEmptySubmissions() {
	ctx := s.Ctx().WithBlockHeight(1_000)
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(100),
		testutil.WithWorkerSubmissionWindow(10),
	)

	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, topicId))

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)

	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WORKER_SUBMISSION, epoch.State)

	legacyNonce := epoch.LegacyNonce()
	workerUnfulfilled, err := s.NonceKeeper().IsWorkerNonceUnfulfilled(ctx, topicId, &legacyNonce)
	s.Require().NoError(err)
	s.Require().True(workerUnfulfilled, "open worker window must add unfulfilled worker nonce")

	// Close worker window (no inferers → soft-fail, still advances FSM).
	ctx = ctx.WithBlockTime(epoch.WorkerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	epoch, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WAITING_GROUND_TRUTH, epoch.State)

	workerUnfulfilled, err = s.NonceKeeper().IsWorkerNonceUnfulfilled(ctx, topicId, &legacyNonce)
	s.Require().NoError(err)
	s.Require().False(workerUnfulfilled, "close worker must fulfill worker nonce even with no inferers")

	// Open reputer window.
	ctx = ctx.WithBlockTime(epoch.ReputerSubmissionWindow.OpenAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	epoch, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_REPUTER_SUBMISSION, epoch.State)

	reputerUnfulfilled, err := s.NonceKeeper().IsReputerNonceUnfulfilled(ctx, topicId, &legacyNonce)
	s.Require().NoError(err)
	s.Require().True(reputerUnfulfilled, "open reputer window must ensure reputer nonce exists")

	// Close reputer + complete (both due at CloseAt; type deps run close before complete).
	ctx = ctx.WithBlockTime(epoch.ReputerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	_, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().ErrorIs(err, collections.ErrNotFound, "completed epoch is removed from store")

	rewardable, err := s.TopicKeeper().IsRewardableTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(rewardable, "complete must mark topic rewardable")

	taskIDSuffix := fmt.Sprintf(":%d-%d", topicId, lastNonce)
	for _, taskType := range []string{
		types.CloseEpochWorkerWindowTask,
		types.OpenEpochReputerWindowTask,
		types.CloseEpochReputerWindowTask,
		types.CompleteEpochTask,
	} {
		_, err = s.SchedulerKeeper().GetTask(ctx, schedulertypes.TaskID(taskType+taskIDSuffix))
		s.Require().ErrorIs(err, collections.ErrNotFound, "lifecycle task %s should be consumed", taskType)
	}
}

func (s *KeeperTestSuite) TestEpochFSMCancelFromWorkerSubmissionUnschedulesTasks() {
	ctx := s.Ctx().WithBlockHeight(2_000)
	topicId := s.CreateTopic(
		testutil.WithEpochLength(60),
		testutil.WithGroundTruthLag(60),
		testutil.WithWorkerSubmissionWindow(10),
	)

	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, topicId))

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)

	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WORKER_SUBMISSION, epoch.State)

	legacyNonce := epoch.LegacyNonce()
	s.Require().NoError(s.EmissionsKeeper().CancelEpoch(ctx, topicId, lastNonce))

	_, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().ErrorIs(err, collections.ErrNotFound, "cancelled epoch is removed from store")

	workerUnfulfilled, err := s.NonceKeeper().IsWorkerNonceUnfulfilled(ctx, topicId, &legacyNonce)
	s.Require().NoError(err)
	s.Require().False(workerUnfulfilled, "cancel must fulfill leftover worker nonce")

	taskIDSuffix := fmt.Sprintf(":%d-%d", topicId, lastNonce)
	for _, taskType := range []string{
		types.CloseEpochWorkerWindowTask,
		types.OpenEpochReputerWindowTask,
		types.CloseEpochReputerWindowTask,
		types.CompleteEpochTask,
	} {
		_, err = s.SchedulerKeeper().GetTask(ctx, schedulertypes.TaskID(taskType+taskIDSuffix))
		s.Require().ErrorIs(err, collections.ErrNotFound, "cancel must unschedule %s", taskType)
	}
}

func (s *KeeperTestSuite) TestEpochFSMCancelFromReputerSubmission() {
	ctx := s.Ctx().WithBlockHeight(3_000)
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(100),
		testutil.WithWorkerSubmissionWindow(10),
	)

	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, topicId))

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)

	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)

	// Advance through close worker + open reputer.
	ctx = ctx.WithBlockTime(epoch.WorkerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))
	epoch, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)

	ctx = ctx.WithBlockTime(epoch.ReputerSubmissionWindow.OpenAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	epoch, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_REPUTER_SUBMISSION, epoch.State)

	legacyNonce := epoch.LegacyNonce()
	reputerUnfulfilled, err := s.NonceKeeper().IsReputerNonceUnfulfilled(ctx, topicId, &legacyNonce)
	s.Require().NoError(err)
	s.Require().True(reputerUnfulfilled)

	s.Require().NoError(s.EmissionsKeeper().CancelEpoch(ctx, topicId, lastNonce))

	_, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().ErrorIs(err, collections.ErrNotFound)

	reputerUnfulfilled, err = s.NonceKeeper().IsReputerNonceUnfulfilled(ctx, topicId, &legacyNonce)
	s.Require().NoError(err)
	s.Require().False(reputerUnfulfilled, "cancel must fulfill leftover reputer nonce")
}

func (s *KeeperTestSuite) TestStartNewEpochOpensWorkerNonceViaSideEffect() {
	ctx := s.Ctx().WithBlockHeight(4_000)
	topicId := s.CreateTopic(
		testutil.WithEpochLength(60),
		testutil.WithGroundTruthLag(60),
		testutil.WithWorkerSubmissionWindow(10),
	)

	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, topicId))

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)

	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	legacy := epoch.LegacyNonce()

	unfulfilled, err := s.NonceKeeper().IsWorkerNonceUnfulfilled(ctx, topicId, &legacy)
	s.Require().NoError(err)
	s.Require().True(unfulfilled)
	s.Require().Equal(ctx.BlockHeight(), legacy.BlockHeight)
}
