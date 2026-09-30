package keeper_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestIsTopicSchedulerManagedAfterStartNewEpoch() {
	ctx := s.Ctx()
	topicId := s.CreateTopic(testutil.WithEpochLength(60), testutil.WithGroundTruthLag(60), testutil.WithWorkerSubmissionWindow(10))

	managed, err := s.EmissionsKeeper().IsTopicSchedulerManaged(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(managed, "create-only topic is not scheduler-managed")

	s.Require().NoError(s.EmissionsKeeper().StartNewEpoch(ctx, topicId))

	managed, err = s.EmissionsKeeper().IsTopicSchedulerManaged(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(managed)
}

func (s *KeeperTestSuite) TestUpdateNoncesSkipsSchedulerManagedTopic() {
	ctx := s.Ctx().WithBlockHeight(100)
	s.WithBlockHeight(100)

	topicId := s.CreateTopic(testutil.WithEpochLength(60), testutil.WithGroundTruthLag(60), testutil.WithWorkerSubmissionWindow(10))
	s.Require().NoError(s.TopicKeeper().ActivateTopic(ctx, topicId))

	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, mustLastEpochNonce(s, topicId))
	s.Require().NoError(err)
	legacy := epoch.LegacyNonce()

	noncesBefore, err := s.NonceKeeper().GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Len(noncesBefore.Nonces, 1)
	s.Require().Equal(legacy.BlockHeight, noncesBefore.Nonces[0].BlockHeight)

	// EndBlocker open at a different height must not create a second worker nonce.
	laterHeight := int64(250)
	ctx = ctx.WithBlockHeight(laterHeight)
	s.WithBlockHeight(laterHeight)
	weight := alloraMath.OneDec()
	err = rewards.UpdateNoncesOfActiveTopics(ctx, *s.EmissionsKeeper(), laterHeight, map[uint64]*alloraMath.Dec{
		topicId: &weight,
	})
	s.Require().NoError(err)

	noncesAfter, err := s.NonceKeeper().GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Len(noncesAfter.Nonces, 1, "scheduler-managed topic must not get an EndBlocker worker nonce")
	s.Require().Equal(legacy.BlockHeight, noncesAfter.Nonces[0].BlockHeight)

	// EpochLastEnded still advances for weight/activation cadence.
	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Equal(laterHeight, topic.EpochLastEnded)
}

func (s *KeeperTestSuite) TestUpdateNoncesStillOpensLegacyTopic() {
	ctx := s.Ctx().WithBlockHeight(300)
	s.WithBlockHeight(300)

	// Create without activation / StartNewEpoch → not scheduler-managed.
	topicId := s.CreateTopic(testutil.WithEpochLength(60), testutil.WithGroundTruthLag(60), testutil.WithWorkerSubmissionWindow(10))
	managed, err := s.EmissionsKeeper().IsTopicSchedulerManaged(ctx, topicId)
	s.Require().NoError(err)
	s.Require().False(managed)

	weight := alloraMath.OneDec()
	err = rewards.UpdateNoncesOfActiveTopics(ctx, *s.EmissionsKeeper(), 300, map[uint64]*alloraMath.Dec{
		topicId: &weight,
	})
	s.Require().NoError(err)

	nonces, err := s.NonceKeeper().GetUnfulfilledWorkerNonces(ctx, topicId)
	s.Require().NoError(err)
	s.Require().Len(nonces.Nonces, 1)
	s.Require().Equal(int64(300), nonces.Nonces[0].BlockHeight)
}

func (s *KeeperTestSuite) TestEndBlockerWorkerCloseSkipsSchedulerManagedTopic() {
	ctx := s.Ctx().WithBlockHeight(400)
	s.WithBlockHeight(400)

	topicId := s.CreateTopic(testutil.WithEpochLength(60), testutil.WithGroundTruthLag(60), testutil.WithWorkerSubmissionWindow(10))
	s.Require().NoError(s.TopicKeeper().ActivateTopic(ctx, topicId))

	epoch, err := s.EmissionsKeeper().GetEpoch(ctx, topicId, mustLastEpochNonce(s, topicId))
	s.Require().NoError(err)
	legacy := epoch.LegacyNonce()

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)
	closeHeight := legacy.BlockHeight + topic.WorkerSubmissionWindow

	// Simulate a stale EndBlocker close-index entry pointing at this topic.
	s.Require().NoError(s.NonceKeeper().AddWorkerWindowTopicId(ctx, closeHeight, topicId))

	ctx = ctx.WithBlockHeight(closeHeight)
	s.WithBlockHeight(closeHeight)
	s.Require().NoError(s.EmissionsAppModule().EndBlock(ctx))

	unfulfilled, err := s.NonceKeeper().IsWorkerNonceUnfulfilled(ctx, topicId, &legacy)
	s.Require().NoError(err)
	s.Require().True(unfulfilled, "EndBlocker must not close FSM-owned worker nonce")

	// Height index is still cleared so it does not linger.
	windowTopics := s.NonceKeeper().GetWorkerWindowTopicIds(ctx, closeHeight)
	s.Require().Empty(windowTopics.TopicIds)
}

func mustLastEpochNonce(s *KeeperTestSuite, topicId uint64) types.NonceV2 {
	nonce, found, err := s.EmissionsKeeper().GetTopicLastEpochNonce(s.Ctx(), topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	return nonce
}
