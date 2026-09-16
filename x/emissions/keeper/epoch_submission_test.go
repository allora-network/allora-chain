package keeper_test

import (
	"time"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestTimeWithinWindowInclusive() {
	open := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	closeAt := open.Add(10 * time.Second)
	window := &types.Window{OpenAt: open, CloseAt: closeAt}

	s.Require().True(keeper.TimeWithinWindow(open, window))
	s.Require().True(keeper.TimeWithinWindow(closeAt, window))
	s.Require().True(keeper.TimeWithinWindow(open.Add(5*time.Second), window))
	s.Require().False(keeper.TimeWithinWindow(open.Add(-time.Nanosecond), window))
	s.Require().False(keeper.TimeWithinWindow(closeAt.Add(time.Nanosecond), window))
	s.Require().False(keeper.TimeWithinWindow(open, nil))
}

func (s *KeeperTestSuite) TestGetEpochByLegacyNonce() {
	ctx := s.Ctx().WithBlockHeight(500)
	topicId := s.CreateTopic(testutil.WithEpochLength(60), testutil.WithGroundTruthLag(60), testutil.WithWorkerSubmissionWindow(10))
	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, topicId))

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)

	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	legacy := epoch.LegacyNonce()

	got, found, err := s.EmissionsKeeper().GetEpochByLegacyNonce(ctx, topicId, legacy)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(epoch.Nonce, got.Nonce)

	_, found, err = s.EmissionsKeeper().GetEpochByLegacyNonce(ctx, topicId, types.Nonce{BlockHeight: legacy.BlockHeight + 1})
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestCheckWorkerSubmissionWindowUsesWallClock() {
	ctx := s.Ctx().WithBlockHeight(600)
	start := ctx.BlockTime()
	topicId := s.CreateTopic(testutil.WithEpochLength(60), testutil.WithGroundTruthLag(60), testutil.WithWorkerSubmissionWindow(10))
	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, topicId))

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	legacy := epoch.LegacyNonce()

	found, err = s.EmissionsKeeper().CheckWorkerSubmissionWindow(ctx, topicId, legacy)
	s.Require().NoError(err)
	s.Require().True(found)

	// After CloseAt, wall-clock check fails even if block height would still be valid.
	ctx = ctx.WithBlockTime(start.Add(10*time.Second + time.Nanosecond))
	found, err = s.EmissionsKeeper().CheckWorkerSubmissionWindow(ctx, topicId, legacy)
	s.Require().ErrorIs(err, types.ErrWorkerNonceWindowNotAvailable)
	s.Require().True(found)
}

func (s *KeeperTestSuite) TestCheckWorkerSubmissionWindowRejectsWrongState() {
	ctx := s.Ctx().WithBlockHeight(700)
	topicId := s.CreateTopic(testutil.WithEpochLength(100), testutil.WithGroundTruthLag(100), testutil.WithWorkerSubmissionWindow(10))
	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, topicId))

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	legacy := epoch.LegacyNonce()

	// Advance FSM past worker submission.
	ctx = ctx.WithBlockTime(epoch.WorkerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	found, err = s.EmissionsKeeper().CheckWorkerSubmissionWindow(ctx, topicId, legacy)
	s.Require().ErrorIs(err, types.ErrWorkerNonceWindowNotAvailable)
	s.Require().True(found)
}

func (s *KeeperTestSuite) TestCheckReputerSubmissionWindowUsesWallClock() {
	ctx := s.Ctx().WithBlockHeight(800)
	topicId := s.CreateTopic(testutil.WithEpochLength(100), testutil.WithGroundTruthLag(100), testutil.WithWorkerSubmissionWindow(10))
	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, topicId))

	lastNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)
	legacy := epoch.LegacyNonce()

	ctx = ctx.WithBlockTime(epoch.WorkerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))
	epoch, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, lastNonce)
	s.Require().NoError(err)

	ctx = ctx.WithBlockTime(epoch.ReputerSubmissionWindow.OpenAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	found, err = s.EmissionsKeeper().CheckReputerSubmissionWindow(ctx, topicId, legacy)
	s.Require().NoError(err)
	s.Require().True(found)

	ctx = ctx.WithBlockTime(epoch.ReputerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	found, err = s.EmissionsKeeper().CheckReputerSubmissionWindow(ctx, topicId, legacy)
	s.Require().ErrorIs(err, types.ErrReputerNonceWindowNotAvailable)
	s.Require().True(found)
}

func (s *KeeperTestSuite) TestCheckWorkerSubmissionWindowFallsThroughWithoutEpoch() {
	ctx := s.Ctx()
	topicId := s.CreateTopic(testutil.WithEpochLength(60), testutil.WithGroundTruthLag(60), testutil.WithWorkerSubmissionWindow(10))

	found, err := s.EmissionsKeeper().CheckWorkerSubmissionWindow(ctx, topicId, types.Nonce{BlockHeight: 1})
	s.Require().NoError(err)
	s.Require().False(found, "no live epoch → caller uses height-based window check")
}
