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

// Mirrors the integration worker→reputer path (EpochLength=5, GroundTruthLag=10,
// WorkerSubmissionWindow=4) while the scheduler creates overlapping wall-clock epochs.
// EndBlocker still owns the nonce data plane on this PR; both must coexist.
func (s *KeeperTestSuite) TestWorkerThenReputerNonceWithSchedulerEpochs() {
	s.SetParamsForTest()
	params, err := s.ParamsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)
	params.MaxActiveTopicsPerBlock = 5
	s.Require().NoError(s.ParamsKeeper().SetParams(s.Ctx(), params))

	workerIndexes := testutil.ReturnIndexes(0, 1)
	reputerIndexes := testutil.ReturnIndexes(1, 1)
	topic := s.FullTopicSetup(
		workerIndexes,
		reputerIndexes,
		testutil.WithEpochLength(5),
		testutil.WithGroundTruthLag(10),
		testutil.WithWorkerSubmissionWindow(4),
	)
	// Extra active topics, as in TopicWeightDistributionChecks, so another
	// topic can churn on the same block as this topic's worker-window close.
	_ = s.FullTopicSetup(
		workerIndexes,
		reputerIndexes,
		testutil.WithEpochLength(10),
		testutil.WithGroundTruthLag(10),
		testutil.WithWorkerSubmissionWindow(4),
	)
	_ = s.FullTopicSetup(
		workerIndexes,
		reputerIndexes,
		testutil.WithEpochLength(20),
		testutil.WithGroundTruthLag(20),
		testutil.WithWorkerSubmissionWindow(4),
	)
	workerValues := testutil.GetWorkerValuesFromIndexes(workerIndexes, "100")

	// Localnet timeout_commit is often ~5s; EpochLength is still stored in blocks
	// but the periodic StartNewEpoch task treats it as seconds, so a new FSM epoch
	// is created every block.
	blockDuration := 5 * time.Second
	var insertedNonce int64
	skippedEpochs := 0
	for i := 0; i < 80; i++ {
		s.WithBlockHeight(s.Ctx().BlockHeight() + 1)
		s.WithBlockTime(s.Ctx().BlockTime().Add(blockDuration))
		s.Require().NoError(s.SchedulerKeeper().BeginBlock(s.Ctx()), "scheduler begin block at height %d", s.Ctx().BlockHeight())
		s.EndBlock()

		fresh, err := s.TopicKeeper().GetTopic(s.Ctx(), topic.Id)
		s.Require().NoError(err)

		if insertedNonce == 0 && fresh.EpochLastEnded > 0 && s.Ctx().BlockHeight() == fresh.EpochLastEnded {
			// Skip a few empty epochs first, matching the integration suite
			// (distribution / registration work happens before the first insert).
			if skippedEpochs < 3 {
				skippedEpochs++
				continue
			}
			s.SetupInferences(fresh.Id, fresh.EpochLastEnded, workerIndexes, workerValues...)
			insertedNonce = fresh.EpochLastEnded
			continue
		}
		if insertedNonce == 0 {
			continue
		}

		if s.Ctx().BlockHeight() == insertedNonce+fresh.GroundTruthLag {
			workerUnfulfilled, err := s.NonceKeeper().IsWorkerNonceUnfulfilled(s.Ctx(), fresh.Id, &types.Nonce{BlockHeight: insertedNonce})
			s.Require().NoError(err)
			reputerUnfulfilled, err := s.NonceKeeper().IsReputerNonceUnfulfilled(s.Ctx(), fresh.Id, &types.Nonce{BlockHeight: insertedNonce})
			s.Require().NoError(err)
			s.Require().False(workerUnfulfilled, "worker nonce %d should be fulfilled before reputer insert", insertedNonce)
			s.Require().True(reputerUnfulfilled, "reputer nonce %d should be unfulfilled before reputer insert", insertedNonce)

			err = s.InsertReputerLossBundle(fresh.Id, insertedNonce, reputerIndexes, testutil.WithWorkerValues(workerValues))
			s.Require().NoError(err)
			return
		}
	}
	s.Fail("did not complete worker/reputer cycle")
}
