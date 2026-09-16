package keeper_test

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestAllocateNextEpochNonceMonotonicPerTopic() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicA := uint64(1)
	topicB := uint64(2)

	_, found, err := k.GetTopicLastEpochNonce(ctx, topicA)
	s.Require().NoError(err)
	s.Require().False(found)

	firstA, err := k.AllocateNextEpochNonce(ctx, topicA)
	s.Require().NoError(err)
	s.Require().Equal(types.ZeroNonce().NextNonce(), firstA)
	s.Require().Equal(types.NonceVersionV1, firstA.Version())
	s.Require().Equal(uint64(1), firstA.Payload())

	secondA, err := k.AllocateNextEpochNonce(ctx, topicA)
	s.Require().NoError(err)
	s.Require().Equal(firstA.NextNonce(), secondA)
	s.Require().Equal(uint64(2), secondA.Payload())

	thirdA, err := k.AllocateNextEpochNonce(ctx, topicA)
	s.Require().NoError(err)
	s.Require().Equal(secondA.NextNonce(), thirdA)

	firstB, err := k.AllocateNextEpochNonce(ctx, topicB)
	s.Require().NoError(err)
	s.Require().Equal(types.ZeroNonce().NextNonce(), firstB)
	s.Require().NotEqual(thirdA, firstB)

	lastA, found, err := k.GetTopicLastEpochNonce(ctx, topicA)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(thirdA, lastA)

	lastB, found, err := k.GetTopicLastEpochNonce(ctx, topicB)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(firstB, lastB)
}

func (s *KeeperTestSuite) TestTopicLastEpochNonceGenesisRoundTrip() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()
	topicA := uint64(7)
	topicB := uint64(9)

	_, err := k.AllocateNextEpochNonce(ctx, topicA)
	s.Require().NoError(err)
	_, err = k.AllocateNextEpochNonce(ctx, topicA)
	s.Require().NoError(err)
	nonceA, err := k.AllocateNextEpochNonce(ctx, topicA)
	s.Require().NoError(err)

	nonceB, err := k.AllocateNextEpochNonce(ctx, topicB)
	s.Require().NoError(err)

	exported, err := k.ExportGenesis(ctx)
	s.Require().NoError(err)
	s.Require().Len(exported.TopicLastEpochNonces, 2)

	byTopic := make(map[uint64]types.NonceV2, len(exported.TopicLastEpochNonces))
	for _, row := range exported.TopicLastEpochNonces {
		s.Require().NotNil(row)
		byTopic[row.TopicId] = row.Nonce
	}
	s.Require().Equal(nonceA, byTopic[topicA])
	s.Require().Equal(nonceB, byTopic[topicB])

	fresh := s.newFreshGenesisSuite()
	s.Require().NoError(fresh.EmissionsKeeper().InitGenesis(fresh.Ctx(), exported))

	gotA, found, err := fresh.EmissionsKeeper().GetTopicLastEpochNonce(fresh.Ctx(), topicA)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(nonceA, gotA)

	gotB, found, err := fresh.EmissionsKeeper().GetTopicLastEpochNonce(fresh.Ctx(), topicB)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(nonceB, gotB)

	nextA, err := fresh.EmissionsKeeper().AllocateNextEpochNonce(fresh.Ctx(), topicA)
	s.Require().NoError(err)
	s.Require().Equal(nonceA.NextNonce(), nextA)
}
