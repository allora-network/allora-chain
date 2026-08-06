package keeper_test

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
)

func spikeUpgradeCtx(s *KeeperTestSuite, height int64) sdk.Context {
	// WithBlockHeight alone can leave BlockTime zero; reconstruction needs a real wall clock.
	return s.Ctx().WithBlockHeight(height).WithBlockTime(time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC))
}

// ENGN-9094 spike: reconstruct mid-flight worker window and keep wall-clock auth green.
func (s *KeeperTestSuite) TestSpikeReconstructWorkerWindowPreservesSubmissionAuth() {
	ctx := spikeUpgradeCtx(s, 1_000)
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(100),
		testutil.WithWorkerSubmissionWindow(20),
	)
	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	legacyHeight := int64(990) // 10 blocks into a 20-block WSW
	s.Require().NoError(s.EmissionsKeeper().GetNonceKeeper().AddWorkerNonce(
		ctx, topicId, &types.Nonce{BlockHeight: legacyHeight},
	))

	s.Require().NoError(s.EmissionsKeeper().ReconstructTopicInFlightEpochsSpike(
		ctx, topicId, keeper.SpikeWindowPolicyLegacyAligned,
	))

	managed, err := s.EmissionsKeeper().IsTopicSchedulerManaged(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(managed)

	legacy := types.Nonce{BlockHeight: legacyHeight}
	epoch, found, err := s.EmissionsKeeper().GetEpochByLegacyNonce(ctx, topicId, legacy)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(types.EpochState_WORKER_SUBMISSION, epoch.State)
	s.Require().Equal(legacyHeight, epoch.StartBlockHeight)

	found, err = s.EmissionsKeeper().CheckWorkerSubmissionWindow(ctx, topicId, legacy)
	s.Require().NoError(err)
	s.Require().True(found)

	closeID := schedulertypes.TaskID(fmt.Sprintf("%s:%d-%d", types.CloseEpochWorkerWindowTask, topicId, epoch.Nonce))
	task, err := s.SchedulerKeeper().GetTask(ctx, closeID)
	s.Require().NoError(err)
	s.Require().Equal(epoch.WorkerSubmissionWindow.CloseAt, *task.ScheduledFor)

	epochWorkerOK, _, heightWorkerOK, _, err := keeper.CompareReconstructedWindowsSpike(
		topic, legacyHeight, ctx.BlockHeight(), ctx.BlockTime(), epoch,
	)
	s.Require().NoError(err)
	s.Require().True(heightWorkerOK)
	s.Require().True(epochWorkerOK, "legacy-aligned 1s/block windows should accept mid-WSW now")
}

// ENGN-9094 spike: ExtraLag > 0 — NewEpoch math opens reputer later than legacy height math.
func (s *KeeperTestSuite) TestSpikeReconstructExtraLagNewEpochMathRejectsValidLegacyReputer() {
	// GTL=150, EpochLength=100 → ExtraLag=50. Legacy reputer opens at nonce+150;
	// NewEpoch opens at nonce+(150+50)=nonce+200.
	ctx := spikeUpgradeCtx(s, 1_160)
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(150),
		testutil.WithWorkerSubmissionWindow(10),
	)
	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(int64(50), types.TopicExtraLag(topic))

	legacyHeight := int64(1_000)
	// Mid legacy reputer window: open at 1150, NewEpoch open would be 1200.
	s.Require().NoError(s.EmissionsKeeper().GetNonceKeeper().AddReputerNonce(
		ctx, topicId, &types.Nonce{BlockHeight: legacyHeight},
	))

	// Reconstruct with NewEpoch math — should leave wall-clock closed while height math is open.
	s.Require().NoError(s.EmissionsKeeper().ReconstructTopicInFlightEpochsSpike(
		ctx, topicId, keeper.SpikeWindowPolicyNewEpochMath,
	))

	legacy := types.Nonce{BlockHeight: legacyHeight}
	epoch, found, err := s.EmissionsKeeper().GetEpochByLegacyNonce(ctx, topicId, legacy)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(types.EpochState_REPUTER_SUBMISSION, epoch.State)

	_, err = s.EmissionsKeeper().CheckReputerSubmissionWindow(ctx, topicId, legacy)
	s.Require().Error(err, "NewEpoch-math windows should reject during ExtraLag gap after GTL")

	_, epochReputerOK, _, heightReputerOK, err := keeper.CompareReconstructedWindowsSpike(
		topic, legacyHeight, ctx.BlockHeight(), ctx.BlockTime(), epoch,
	)
	s.Require().NoError(err)
	s.Require().True(heightReputerOK, "legacy height math still open")
	s.Require().False(epochReputerOK, "NewEpoch wall-clock not open yet")
}

// ENGN-9094 spike: same ExtraLag fixture with legacy-aligned windows preserves reputer auth.
func (s *KeeperTestSuite) TestSpikeReconstructExtraLagLegacyAlignedPreservesReputerAuth() {
	ctx := spikeUpgradeCtx(s, 1_160)
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(150),
		testutil.WithWorkerSubmissionWindow(10),
	)
	legacyHeight := int64(1_000)
	s.Require().NoError(s.EmissionsKeeper().GetNonceKeeper().AddReputerNonce(
		ctx, topicId, &types.Nonce{BlockHeight: legacyHeight},
	))

	s.Require().NoError(s.EmissionsKeeper().ReconstructTopicInFlightEpochsSpike(
		ctx, topicId, keeper.SpikeWindowPolicyLegacyAligned,
	))

	legacy := types.Nonce{BlockHeight: legacyHeight}
	found, err := s.EmissionsKeeper().CheckReputerSubmissionWindow(ctx, topicId, legacy)
	s.Require().NoError(err)
	s.Require().True(found)
}

// ENGN-9094 spike: remaining close-worker task eventually transitions the epoch.
func (s *KeeperTestSuite) TestSpikeReconstructSchedulesCloseWorkerTransition() {
	ctx := spikeUpgradeCtx(s, 1_000)
	topicId := s.CreateTopic(
		testutil.WithEpochLength(100),
		testutil.WithGroundTruthLag(100),
		testutil.WithWorkerSubmissionWindow(20),
	)
	legacyHeight := int64(990)
	s.Require().NoError(s.EmissionsKeeper().GetNonceKeeper().AddWorkerNonce(
		ctx, topicId, &types.Nonce{BlockHeight: legacyHeight},
	))

	s.Require().NoError(s.EmissionsKeeper().ReconstructTopicInFlightEpochsSpike(
		ctx, topicId, keeper.SpikeWindowPolicyLegacyAligned,
	))

	epoch, found, err := s.EmissionsKeeper().GetEpochByLegacyNonce(
		ctx, topicId, types.Nonce{BlockHeight: legacyHeight},
	)
	s.Require().NoError(err)
	s.Require().True(found)

	ctx = ctx.WithBlockTime(epoch.WorkerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(ctx))

	epoch, err = s.EmissionsKeeper().GetEpoch(ctx, topicId, epoch.Nonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WAITING_GROUND_TRUTH, epoch.State)
}
