package keeper_test

import (
	"fmt"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
)

// markTopicActiveWithoutScheduler marks a topic active without invoking lifecycle
// hooks (which would StartNewEpoch). Used to stage legacy pre-cutover state.
func (s *KeeperTestSuite) markTopicActiveWithoutScheduler(ctx sdk.Context, topicID uint64) {
	block := ctx.BlockHeight() + 10_000
	s.Require().NoError(s.TopicKeeper().SetTopicToNextPossibleChurningBlock(ctx, topicID, block))
	s.Require().NoError(s.TopicKeeper().SetActiveTopics(ctx, topicID))
	active, err := s.TopicKeeper().IsTopicActive(ctx, topicID)
	s.Require().NoError(err)
	s.Require().True(active)
}

func (s *KeeperTestSuite) TestMigrateToSchedulerEpochsReconstructsWorkerWindow() {
	const blockTimeSecs int64 = 6
	legacyHeight := int64(1_000)
	ctx := s.Ctx().WithBlockHeight(legacyHeight + 5).WithBlockTime(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))

	topicID := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(100),
		testutil.WithWorkerSubmissionWindow(10),
	)
	s.markTopicActiveWithoutScheduler(ctx, topicID)
	s.Require().NoError(s.NonceKeeper().AddWorkerNonce(ctx, topicID, &types.Nonce{BlockHeight: legacyHeight}))

	paramsBefore, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	minEpochBefore := paramsBefore.MinEpochLength

	s.Require().NoError(s.EmissionsKeeper().MigrateToSchedulerEpochsWithBlockTime(ctx, blockTimeSecs))

	topic, err := s.TopicKeeper().GetTopic(ctx, topicID)
	s.Require().NoError(err)
	s.Require().Equal(int64(100*blockTimeSecs), topic.EpochLength)
	s.Require().Equal(int64(100*blockTimeSecs), topic.GroundTruthLag)
	s.Require().Equal(int64(10*blockTimeSecs), topic.WorkerSubmissionWindow)

	paramsAfter, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	s.Require().Equal(minEpochBefore*blockTimeSecs, paramsAfter.MinEpochLength)

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicID)
	s.Require().NoError(err)
	s.Require().True(found)

	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicID, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WORKER_SUBMISSION, epoch.State)
	s.Require().Equal(legacyHeight, epoch.StartBlockHeight)
	s.Require().Equal(types.Nonce{BlockHeight: legacyHeight}, epoch.LegacyNonce())

	blocksElapsed := int64(5)
	startAt := ctx.BlockTime().Add(-time.Duration(blocksElapsed*blockTimeSecs) * time.Second)
	s.Require().Equal(startAt, epoch.WorkerSubmissionWindow.OpenAt)
	s.Require().Equal(startAt.Add(time.Duration(10*blockTimeSecs)*time.Second), epoch.WorkerSubmissionWindow.CloseAt)
	s.Require().Equal(startAt.Add(time.Duration(100*blockTimeSecs)*time.Second), epoch.ReputerSubmissionWindow.OpenAt)

	taskIDSuffix := fmt.Sprintf(":%d-%d", topicID, lastNonce)
	closeWorker, err := s.SchedulerKeeper().GetTask(ctx, schedulertypes.TaskID(types.CloseEpochWorkerWindowTask+taskIDSuffix))
	s.Require().NoError(err)
	s.Require().Equal(epoch.WorkerSubmissionWindow.CloseAt, *closeWorker.ScheduledFor)

	periodicID := schedulertypes.TaskID(fmt.Sprintf("%s:%d", types.StartNewEpochTask, topicID))
	periodic, err := s.SchedulerKeeper().GetTask(ctx, periodicID)
	s.Require().NoError(err)
	s.Require().NotNil(periodic.Interval)
	s.Require().Equal(time.Duration(topic.EpochLength)*time.Second, *periodic.Interval)

	// Wall-clock submission auth still works via reconstructed LegacyNonce.
	ok, err := s.EmissionsKeeper().CheckWorkerSubmissionWindow(ctx, topicID, epoch.LegacyNonce())
	s.Require().NoError(err)
	s.Require().True(ok)
}

func (s *KeeperTestSuite) TestMigrateToSchedulerEpochsLegacyAlignedReputerOpenWithExtraLag() {
	const blockTimeSecs int64 = 6
	legacyHeight := int64(1_000)
	// Past GTL (1150) but before NewEpoch's GTL+ExtraLag open (1200).
	currentHeight := int64(1_160)
	ctx := s.Ctx().WithBlockHeight(currentHeight).WithBlockTime(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))

	topicID := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(150),
		testutil.WithWorkerSubmissionWindow(10),
	)
	s.markTopicActiveWithoutScheduler(ctx, topicID)
	s.Require().NoError(s.NonceKeeper().AddReputerNonce(ctx, topicID, &types.Nonce{BlockHeight: legacyHeight}))

	s.Require().Equal(int64(50), types.TopicExtraLag(types.Topic{EpochLength: 100, GroundTruthLag: 150}))

	s.Require().NoError(s.EmissionsKeeper().MigrateToSchedulerEpochsWithBlockTime(ctx, blockTimeSecs))

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicID)
	s.Require().NoError(err)
	s.Require().True(found)
	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicID, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_REPUTER_SUBMISSION, epoch.State)

	blocksElapsed := currentHeight - legacyHeight
	startAt := ctx.BlockTime().Add(-time.Duration(blocksElapsed*blockTimeSecs) * time.Second)
	// Legacy-aligned: open at GTL, not GTL+ExtraLag.
	legacyOpen := startAt.Add(time.Duration(150*blockTimeSecs) * time.Second)
	newEpochOpen := startAt.Add(time.Duration((150+50)*blockTimeSecs) * time.Second)
	s.Require().Equal(legacyOpen, epoch.ReputerSubmissionWindow.OpenAt)
	s.Require().NotEqual(newEpochOpen, epoch.ReputerSubmissionWindow.OpenAt)
	// Close spans ExtraLag + EpochLength after GTL open (legacy formula).
	s.Require().Equal(
		legacyOpen.Add(time.Duration((50+100)*blockTimeSecs)*time.Second),
		epoch.ReputerSubmissionWindow.CloseAt,
	)

	ok, err := s.EmissionsKeeper().CheckReputerSubmissionWindow(ctx, topicID, epoch.LegacyNonce())
	s.Require().NoError(err)
	s.Require().True(ok)
}

func (s *KeeperTestSuite) TestMigrateToSchedulerEpochsSkipsAlreadyManagedTopics() {
	const blockTimeSecs int64 = 6
	ctx := s.Ctx()
	topicID := s.CreateTopic(
		testutil.WithEpochLength(60),
		testutil.WithGroundTruthLag(60),
		testutil.WithWorkerSubmissionWindow(10),
	)
	s.Require().NoError(s.TopicKeeper().ActivateTopic(ctx, topicID))

	topicBefore, err := s.TopicKeeper().GetTopic(ctx, topicID)
	s.Require().NoError(err)
	lastBefore, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicID)
	s.Require().NoError(err)
	s.Require().True(found)

	s.Require().NoError(s.EmissionsKeeper().MigrateToSchedulerEpochsWithBlockTime(ctx, blockTimeSecs))

	topicAfter, err := s.TopicKeeper().GetTopic(ctx, topicID)
	s.Require().NoError(err)
	s.Require().Equal(topicBefore.EpochLength, topicAfter.EpochLength)
	s.Require().Equal(topicBefore.GroundTruthLag, topicAfter.GroundTruthLag)
	s.Require().Equal(topicBefore.WorkerSubmissionWindow, topicAfter.WorkerSubmissionWindow)

	lastAfter, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicID)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lastBefore, lastAfter)
}

func (s *KeeperTestSuite) TestMigrateToSchedulerEpochsEnrollsActiveTopicWithoutNonces() {
	const blockTimeSecs int64 = 6
	ctx := s.Ctx()
	topicID := s.CreateTopic(
		testutil.WithEpochLength(50),
		testutil.WithGroundTruthLag(50),
		testutil.WithWorkerSubmissionWindow(10),
	)
	s.markTopicActiveWithoutScheduler(ctx, topicID)

	s.Require().NoError(s.EmissionsKeeper().MigrateToSchedulerEpochsWithBlockTime(ctx, blockTimeSecs))

	topic, err := s.TopicKeeper().GetTopic(ctx, topicID)
	s.Require().NoError(err)
	s.Require().Equal(int64(50*blockTimeSecs), topic.EpochLength)

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicID)
	s.Require().NoError(err)
	s.Require().True(found)
	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicID, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WORKER_SUBMISSION, epoch.State)

	periodicID := schedulertypes.TaskID(fmt.Sprintf("%s:%d", types.StartNewEpochTask, topicID))
	_, err = s.SchedulerKeeper().GetTask(ctx, periodicID)
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) TestMigrateToSchedulerEpochsInactiveTopicConvertsButDoesNotEnroll() {
	const blockTimeSecs int64 = 6
	ctx := s.Ctx().WithBlockHeight(2_000)

	topicID := s.CreateTopic(
		testutil.WithEpochLength(40),
		testutil.WithGroundTruthLag(40),
		testutil.WithWorkerSubmissionWindow(10),
	)
	s.Require().NoError(s.NonceKeeper().AddWorkerNonce(ctx, topicID, &types.Nonce{BlockHeight: 1_990}))

	active, err := s.TopicKeeper().IsTopicActive(ctx, topicID)
	s.Require().NoError(err)
	s.Require().False(active)

	s.Require().NoError(s.EmissionsKeeper().MigrateToSchedulerEpochsWithBlockTime(ctx, blockTimeSecs))

	topic, err := s.TopicKeeper().GetTopic(ctx, topicID)
	s.Require().NoError(err)
	s.Require().Equal(int64(40*blockTimeSecs), topic.EpochLength)

	_, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicID)
	s.Require().NoError(err)
	s.Require().False(found)

	periodicID := schedulertypes.TaskID(fmt.Sprintf("%s:%d", types.StartNewEpochTask, topicID))
	_, err = s.SchedulerKeeper().GetTask(ctx, periodicID)
	s.Require().ErrorIs(err, collections.ErrNotFound)
}
