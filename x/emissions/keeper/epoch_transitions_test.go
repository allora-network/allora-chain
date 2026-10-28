package keeper_test

import (
	"fmt"
	"time"

	"cosmossdk.io/collections"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
)

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
	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	epochInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				TopicId:     topicId,
				BlockHeight: legacyNonce.BlockHeight,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(1)},
				Inferer:     s.AddrsStr(0),
			},
		},
	}
	s.Require().NoError(s.WorkerKeeper().InsertActiveInferences(ctx, topicId, legacyNonce.BlockHeight, epochInferences))

	otherHeight := legacyNonce.BlockHeight + 1
	otherInferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				TopicId:     topicId,
				BlockHeight: otherHeight,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(2)},
				Inferer:     s.AddrsStr(1),
			},
		},
	}
	s.Require().NoError(s.WorkerKeeper().InsertActiveInferences(ctx, topicId, otherHeight, otherInferences))

	s.Require().NoError(s.EmissionsKeeper().CancelEpoch(ctx, topicId, lastNonce))

	_, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().ErrorIs(err, collections.ErrNotFound, "cancelled epoch is removed from store")

	workerUnfulfilled, err := s.NonceKeeper().IsWorkerNonceUnfulfilled(ctx, topicId, &legacyNonce)
	s.Require().NoError(err)
	s.Require().False(workerUnfulfilled, "cancel must fulfill leftover worker nonce")

	pruned, err := s.WorkerKeeper().GetInferencesAtBlock(ctx, topic, legacyNonce.BlockHeight, false)
	s.Require().NoError(err)
	s.Require().Empty(pruned.Inferences, "cancel must prune this epoch's records")

	kept, err := s.WorkerKeeper().GetInferencesAtBlock(ctx, topic, otherHeight, false)
	s.Require().NoError(err)
	s.Require().Len(kept.Inferences, 1, "cancel must not prune other epochs' heights")

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

func (s *KeeperTestSuite) TestCancelOneLiveEpochLeavesTheOtherEpochRecords() {
	ctx := s.Ctx().WithBlockHeight(5_000)
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(100),
		testutil.WithWorkerSubmissionWindow(10),
	)

	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, topicId))
	firstNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	first, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, firstNonce)
	s.Require().NoError(err)
	s.Require().Equal(int64(5_000), first.StartBlockHeight)

	later := ctx.WithBlockHeight(5_100)
	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(later, topicId))
	secondNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(later, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().NotEqual(firstNonce, secondNonce)
	second, err := s.EmissionsKeeper().GetEpoch(later, topicId, secondNonce)
	s.Require().NoError(err)
	s.Require().Equal(int64(5_100), second.StartBlockHeight)
	s.Require().Equal(types.EpochState_WORKER_SUBMISSION, second.State)

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.insertHeightRecords(ctx, topicId, first.StartBlockHeight, 1)
	s.insertHeightRecords(ctx, topicId, second.StartBlockHeight, 2)

	s.Require().NoError(s.EmissionsKeeper().CancelEpoch(ctx, topicId, firstNonce))

	_, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, firstNonce)
	s.Require().ErrorIs(err, collections.ErrNotFound, "cancelled epoch is removed")
	keptEpoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, secondNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WORKER_SUBMISSION, keptEpoch.State)
	s.Require().Equal(second.StartBlockHeight, keptEpoch.StartBlockHeight)

	prunedInferences, err := s.WorkerKeeper().GetInferencesAtBlock(ctx, topic, first.StartBlockHeight, false)
	s.Require().NoError(err)
	s.Require().Empty(prunedInferences.Inferences)
	keptInferences, err := s.WorkerKeeper().GetInferencesAtBlock(ctx, topic, second.StartBlockHeight, false)
	s.Require().NoError(err)
	s.Require().Len(keptInferences.Inferences, 1)
	s.Require().True(keptInferences.Inferences[0].Values[0].Equal(alloraMath.NewDecFromInt64(2)))

	prunedLoss, err := s.ReputerLossKeeper().GetNetworkLossBundleAtBlock(ctx, topicId, first.StartBlockHeight)
	s.Require().NoError(err)
	s.Require().Empty(prunedLoss.InfererValues)
	keptLoss, err := s.ReputerLossKeeper().GetNetworkLossBundleAtBlock(ctx, topicId, second.StartBlockHeight)
	s.Require().NoError(err)
	s.Require().NotEmpty(keptLoss.InfererValues)
	s.Require().True(keptLoss.CombinedValue.Equal(alloraMath.NewDecFromInt64(2)))
}

func (s *KeeperTestSuite) insertHeightRecords(ctx sdk.Context, topicId uint64, height int64, value int64) {
	inferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				TopicId:     topicId,
				BlockHeight: height,
				Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(value)},
				Inferer:     s.AddrsStr(0),
			},
		},
	}
	s.Require().NoError(s.WorkerKeeper().InsertActiveInferences(ctx, topicId, height, inferences))

	//nolint:exhaustruct // only the fields the prune check reads back
	loss := types.ValueBundle{
		TopicId:       topicId,
		Reputer:       s.AddrsStr(4),
		CombinedValue: alloraMath.NewDecFromInt64(value),
		NaiveValue:    alloraMath.NewDecFromInt64(0),
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: height},
		},
		InfererValues: []*types.WorkerAttributedValue{
			{Worker: s.AddrsStr(0), Value: alloraMath.NewDecFromInt64(value)},
		},
	}
	s.Require().NoError(s.ReputerLossKeeper().InsertNetworkLossBundleAtBlock(ctx, topicId, height, loss))
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

func (s *KeeperTestSuite) TestEpochCloseUsesTimingCopiedAtCreation() {
	ctx := s.Ctx().WithBlockHeight(6_000)
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
	s.Require().Equal(int64(10), epoch.TopicWorkerSubmissionWindow)

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	topic.WorkerSubmissionWindow = 1
	s.Require().NoError(s.TopicKeeper().SetTopic(ctx, topicId, topic))

	ctx = ctx.WithBlockTime(epoch.WorkerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	epoch, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WAITING_GROUND_TRUTH, epoch.State)
}

func (s *KeeperTestSuite) TestCloseReputerWindowSoftFailsNoQualifiedInferers() {
	ctx := s.Ctx().WithBlockHeight(7_000)
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(100),
		testutil.WithWorkerSubmissionWindow(10),
	)

	k := s.EmissionsKeeper()
	k.SetEpochCloseHandlers(
		func(ctx sdk.Context, topic types.Topic, nonce types.Nonce) error {
			return actorutils.CloseWorkerNonce(k, ctx, topic, nonce)
		},
		func(sdk.Context, types.Topic, types.Nonce) error {
			return types.ErrNoQualifiedInferers
		},
	)
	s.T().Cleanup(func() {
		k.SetEpochCloseHandlers(
			func(ctx sdk.Context, topic types.Topic, nonce types.Nonce) error {
				return actorutils.CloseWorkerNonce(k, ctx, topic, nonce)
			},
			func(ctx sdk.Context, topic types.Topic, nonce types.Nonce) error {
				return actorutils.CloseReputerNonce(k, ctx, topic, nonce)
			},
		)
	})

	s.Require().NoError(k.StartNewEpoch(ctx, topicId))
	lastNonce, found, err := k.GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	epoch, err := k.GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)

	ctx = ctx.WithBlockTime(epoch.WorkerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))
	ctx = ctx.WithBlockTime(epoch.ReputerSubmissionWindow.OpenAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))
	ctx = ctx.WithBlockTime(epoch.ReputerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	_, err = k.GetEpoch(ctx, topicId, lastNonce)
	s.Require().ErrorIs(err, collections.ErrNotFound, "reputer close soft-fail must still let the epoch complete")
}

func (s *KeeperTestSuite) TestEpochFSMCancelFromWaitingGroundTruth() {
	ctx := s.Ctx().WithBlockHeight(8_000)
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

	ctx = ctx.WithBlockTime(epoch.WorkerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))
	epoch, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WAITING_GROUND_TRUTH, epoch.State)

	legacyNonce := epoch.LegacyNonce()
	s.Require().NoError(s.EmissionsKeeper().CancelEpoch(ctx, topicId, lastNonce))

	workerUnfulfilled, err := s.NonceKeeper().IsWorkerNonceUnfulfilled(ctx, topicId, &legacyNonce)
	s.Require().NoError(err)
	s.Require().False(workerUnfulfilled)
	reputerUnfulfilled, err := s.NonceKeeper().IsReputerNonceUnfulfilled(ctx, topicId, &legacyNonce)
	s.Require().NoError(err)
	s.Require().False(reputerUnfulfilled)
	s.Require().Greater(countTypedEvents(ctx, func(msg sdk.Msg) bool {
		_, ok := msg.(*types.EventWorkerSubmissionWindowClosed)
		return ok
	}), 0)

	taskIDSuffix := fmt.Sprintf(":%d-%d", topicId, lastNonce)
	for _, taskType := range []string{
		types.OpenEpochReputerWindowTask,
		types.CloseEpochReputerWindowTask,
		types.CompleteEpochTask,
	} {
		_, err = s.SchedulerKeeper().GetTask(ctx, schedulertypes.TaskID(taskType+taskIDSuffix))
		s.Require().ErrorIs(err, collections.ErrNotFound, "cancel must unschedule %s", taskType)
	}
}

func (s *KeeperTestSuite) TestEpochFSMCancelFromPendingCompletion() {
	ctx := s.Ctx().WithBlockHeight(9_000)
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

	ctx = ctx.WithBlockTime(epoch.WorkerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))
	ctx = ctx.WithBlockTime(epoch.ReputerSubmissionWindow.OpenAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	taskIDSuffix := fmt.Sprintf(":%d-%d", topicId, lastNonce)
	completeID := schedulertypes.TaskID(types.CompleteEpochTask + taskIDSuffix)
	s.Require().NoError(s.SchedulerKeeper().CancelTask(ctx, completeID))
	s.Require().NoError(s.SchedulerKeeper().ScheduleTask(
		ctx,
		types.CompleteEpochTask,
		completeID,
		&types.EpochTransitionTaskArgs{TopicId: topicId, Nonce: lastNonce},
		schedulertypes.ScheduleAt(epoch.ReputerSubmissionWindow.CloseAt.Add(time.Hour)),
	))

	ctx = ctx.WithBlockTime(epoch.ReputerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))
	epoch, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_PENDING_COMPLETION, epoch.State)

	legacyNonce := epoch.LegacyNonce()
	workerUnfulfilled, err := s.NonceKeeper().IsWorkerNonceUnfulfilled(ctx, topicId, &legacyNonce)
	s.Require().NoError(err)
	s.Require().False(workerUnfulfilled)
	reputerUnfulfilled, err := s.NonceKeeper().IsReputerNonceUnfulfilled(ctx, topicId, &legacyNonce)
	s.Require().NoError(err)
	s.Require().False(reputerUnfulfilled)

	_, err = s.SchedulerKeeper().GetTask(ctx, completeID)
	s.Require().NoError(err)

	s.Require().NoError(s.EmissionsKeeper().CancelEpoch(ctx, topicId, lastNonce))
	_, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().ErrorIs(err, collections.ErrNotFound)
	_, err = s.SchedulerKeeper().GetTask(ctx, completeID)
	s.Require().ErrorIs(err, collections.ErrNotFound)
}

func countTypedEvents(ctx sdk.Context, match func(sdk.Msg) bool) int {
	n := 0
	for _, ev := range ctx.EventManager().Events() {
		msg, err := sdk.ParseTypedEvent(abci.Event(ev))
		if err != nil {
			continue
		}
		if match(msg) {
			n++
		}
	}
	return n
}
