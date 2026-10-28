package v17_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/collections"
	"github.com/stretchr/testify/suite"

	v17 "github.com/allora-network/allora-chain/x/emissions/migrations/v17"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
)

type EmissionsV17MigrationTestSuite struct {
	testutil.TestSuite
}

func TestEmissionsV17MigrationTestSuite(t *testing.T) {
	suite.Run(t, &EmissionsV17MigrationTestSuite{
		testutil.NewTestSuite("emissions_V17Migrations"),
	})
}

func (s *EmissionsV17MigrationTestSuite) TestMigrateStoreConvertsTimingAndEnrolls() {
	ctx := s.Ctx()
	topicID := s.CreateTopic(
		testutil.WithEpochLength(30),
		testutil.WithGroundTruthLag(30),
		testutil.WithWorkerSubmissionWindow(10),
	)
	block := ctx.BlockHeight() + 10_000
	s.Require().NoError(s.TopicKeeper().SetTopicToNextPossibleChurningBlock(ctx, topicID, block))
	s.Require().NoError(s.TopicKeeper().SetActiveTopics(ctx, topicID))

	s.Require().NoError(v17.MigrateStore(ctx, *s.EmissionsKeeper()))

	topic, err := s.TopicKeeper().GetTopic(ctx, topicID)
	s.Require().NoError(err)
	s.Require().Equal(int64(30)*6, topic.EpochLength)

	_, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(ctx, topicID)
	s.Require().NoError(err)
	s.Require().True(found)

	params, err := s.ParamsKeeper().GetParams(ctx)
	s.Require().NoError(err)
	s.Require().Equal(types.DefaultParams().MinEpochLength*6, params.MinEpochLength)
}

// An active topic with an open worker nonce and an open reputer nonce is what
// the v0.18 upgrade handler hands to this migration. The nonce comes back as a
// live epoch in seconds, and the next epoch accepts a submission at its start height.
func (s *EmissionsV17MigrationTestSuite) TestMigrateStoreReconstructsInFlightNonceThenSubmitsNextEpoch() {
	const (
		epochLengthBlocks int64 = 100
		gtlBlocks         int64 = 100
		wswBlocks         int64 = 10
		legacyHeight      int64 = 1_000
		blockTimeSecs     int64 = 6
	)
	migrateAt := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	s.WithBlockHeight(legacyHeight + 5)
	s.WithBlockTime(migrateAt)

	topicID := s.CreateTopic(
		testutil.WithEpochLength(epochLengthBlocks),
		testutil.WithGroundTruthLag(gtlBlocks),
		testutil.WithWorkerSubmissionWindow(wswBlocks),
	)
	s.Require().NoError(s.TopicKeeper().SetTopicToNextPossibleChurningBlock(s.Ctx(), topicID, s.Ctx().BlockHeight()+10_000))
	s.Require().NoError(s.TopicKeeper().SetActiveTopics(s.Ctx(), topicID))
	legacyNonce := &types.Nonce{BlockHeight: legacyHeight}
	s.Require().NoError(s.NonceKeeper().AddWorkerNonce(s.Ctx(), topicID, legacyNonce))
	s.Require().NoError(s.NonceKeeper().AddReputerNonce(s.Ctx(), topicID, legacyNonce))

	s.Require().NoError(v17.MigrateStore(s.Ctx(), *s.EmissionsKeeper()))

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicID)
	s.Require().NoError(err)
	s.Require().Equal(epochLengthBlocks*blockTimeSecs, topic.EpochLength)
	s.Require().Equal(gtlBlocks*blockTimeSecs, topic.GroundTruthLag)
	s.Require().Equal(wswBlocks*blockTimeSecs, topic.WorkerSubmissionWindow)

	migratedNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(s.Ctx(), topicID)
	s.Require().NoError(err)
	s.Require().True(found)
	migrated, err := s.EmissionsKeeper().GetEpoch(s.Ctx(), topicID, migratedNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WORKER_SUBMISSION, migrated.State)
	s.Require().Equal(legacyHeight, migrated.StartBlockHeight)
	s.Require().Equal(types.Nonce{BlockHeight: legacyHeight}, migrated.LegacyNonce())

	workerOpen, err := s.NonceKeeper().IsWorkerNonceUnfulfilled(s.Ctx(), topicID, legacyNonce)
	s.Require().NoError(err)
	s.Require().True(workerOpen)
	reputerOpen, err := s.NonceKeeper().IsReputerNonceUnfulfilled(s.Ctx(), topicID, legacyNonce)
	s.Require().NoError(err)
	s.Require().True(reputerOpen)

	periodicID := schedulertypes.TaskID(fmt.Sprintf("%s:%d", types.StartNewEpochTask, topicID))
	periodic, err := s.SchedulerKeeper().GetTask(s.Ctx(), periodicID)
	s.Require().NoError(err)
	s.Require().NotNil(periodic.Interval)
	s.Require().Equal(time.Duration(topic.EpochLength)*time.Second, *periodic.Interval)

	s.SetupParticipants(topicID, []int{0, 1}, false)
	s.SetupParticipants(topicID, []int{2, 3}, true)

	s.WithBlockHeight(s.Ctx().BlockHeight() + 1)
	s.WithBlockTime(migrateAt.Add(time.Duration(topic.EpochLength)*time.Second + time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(s.Ctx()))
	s.Require().NoError(s.SchedulerKeeper().CancelTask(s.Ctx(), periodicID))

	nextNonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(s.Ctx(), topicID)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Greater(nextNonce, migratedNonce)
	nextEpoch, err := s.EmissionsKeeper().GetEpoch(s.Ctx(), topicID, nextNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WORKER_SUBMISSION, nextEpoch.State)
	s.Require().Equal(s.Ctx().BlockHeight(), nextEpoch.StartBlockHeight)

	s.SetupInferences(topicID, nextEpoch.StartBlockHeight, []int{0, 1})
	stored, err := s.WorkerKeeper().GetWorkerLatestInferenceByTopicId(s.Ctx(), topicID, s.AddrsStr(0))
	s.Require().NoError(err)
	s.Require().Equal(nextEpoch.StartBlockHeight, stored.BlockHeight)

	s.WithBlockTime(nextEpoch.WorkerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(s.Ctx()))
	nextEpoch, err = s.EmissionsKeeper().GetEpoch(s.Ctx(), topicID, nextNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_WAITING_GROUND_TRUTH, nextEpoch.State)

	networkInferences, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topicID, nextEpoch.StartBlockHeight)
	s.Require().NoError(err)
	s.Require().NotEmpty(networkInferences.InfererValues)

	s.WithBlockTime(nextEpoch.ReputerSubmissionWindow.OpenAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(s.Ctx()))
	nextEpoch, err = s.EmissionsKeeper().GetEpoch(s.Ctx(), topicID, nextNonce)
	s.Require().NoError(err)
	s.Require().Equal(types.EpochState_REPUTER_SUBMISSION, nextEpoch.State)

	s.Require().NoError(s.InsertReputerLossBundle(topicID, nextEpoch.StartBlockHeight, []int{2, 3}))

	s.WithBlockTime(nextEpoch.ReputerSubmissionWindow.CloseAt.Add(time.Nanosecond))
	s.Require().NoError(s.SchedulerKeeper().BeginBlock(s.Ctx()))
	_, err = s.EmissionsKeeper().GetEpoch(s.Ctx(), topicID, nextNonce)
	s.Require().ErrorIs(err, collections.ErrNotFound)

	lossBundle, err := s.ReputerLossKeeper().GetNetworkLossBundleAtBlock(s.Ctx(), topicID, nextEpoch.StartBlockHeight)
	s.Require().NoError(err)
	s.Require().NotEmpty(lossBundle.InfererValues)
}
