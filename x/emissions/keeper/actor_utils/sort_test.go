package actorutils_test

import (
	"fmt"
	"strconv"

	alloraMath "github.com/allora-network/allora-chain/math"
	alloratestutil "github.com/allora-network/allora-chain/test/testutil"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *WorkerTestSuite) TestFindTopNByScoreDesc() {
	require := s.Require()
	topicId := uint64(1)

	reputerScoreEmas := []emissionstypes.Score{
		{TopicId: topicId, BlockHeight: 1, Address: s.AddrsStr(0), Score: alloraMath.NewDecFromInt64(90)},
		{TopicId: topicId, BlockHeight: 1, Address: s.AddrsStr(1), Score: alloraMath.NewDecFromInt64(40)},
		{TopicId: topicId, BlockHeight: 1, Address: s.AddrsStr(2), Score: alloraMath.NewDecFromInt64(80)},
		{TopicId: topicId, BlockHeight: 1, Address: s.AddrsStr(3), Score: alloraMath.NewDecFromInt64(20)},
		{TopicId: topicId, BlockHeight: 1, Address: s.AddrsStr(4), Score: alloraMath.NewDecFromInt64(100)},
	}

	topActors, _, topActorsBool := actorutils.FindTopNByScoreDesc(s.Ctx(), 3, reputerScoreEmas, 1)
	require.Equal(s.AddrsStr(4), topActors[0].Address)
	require.Equal(s.AddrsStr(0), topActors[1].Address)
	require.Equal(s.AddrsStr(2), topActors[2].Address)

	_, isTop := topActorsBool[s.AddrsStr(0)]
	require.True(isTop)
	_, isTop = topActorsBool[s.AddrsStr(1)]
	require.False(isTop)
	_, isTop = topActorsBool[s.AddrsStr(2)]
	require.True(isTop)
	_, isTop = topActorsBool[s.AddrsStr(3)]
	require.False(isTop)
	_, isTop = topActorsBool[s.AddrsStr(4)]
	require.True(isTop)
}

func (s *WorkerTestSuite) TestFindTopNByScoreDescCsv() {
	require := s.Require()

	for epoch := 301; epoch < 400; epoch++ {
		epochGet := alloratestutil.GetSortitionSimulatorValuesGetterForEpochs()[epoch]
		topicId := uint64(1)

		nParticipants, err := epochGet("n_participants").UInt64()
		require.NoError(err)
		reputerScoreEmas := make([]emissionstypes.Score, 0)
		for i := 0; uint64(i) < nParticipants; i++ {
			participantName := strconv.Itoa(i)
			reputerScoreEmas = append(reputerScoreEmas, emissionstypes.Score{
				TopicId:     topicId,
				BlockHeight: int64(epoch),
				Address:     participantName,
				Score:       epochGet(fmt.Sprintf("%s_prev_quality_ema", participantName)),
			})
		}

		nParticipantsDrawn, err := epochGet("n_participants_drawn").UInt64()
		require.NoError(err)

		_, _, topActorsBool := actorutils.FindTopNByScoreDesc(
			s.Ctx(),
			nParticipantsDrawn,
			reputerScoreEmas,
			1,
		)

		for i := 0; uint64(i) < nParticipants; i++ {
			participantName := strconv.Itoa(i)
			expectedTop := epochGet(fmt.Sprintf("%s_active", participantName))
			_, isTop := topActorsBool[participantName]
			require.True(
				expectedTop.Equal(alloraMath.OneDec()) || expectedTop.Equal(alloraMath.ZeroDec()),
				"expectedTop must be 0 or 1, got %s", expectedTop.String(),
			)
			expectedTopBool := expectedTop.Equal(alloraMath.OneDec())
			require.Equal(expectedTopBool, isTop)
		}
	}
}

// Not convinced we should be not throwing errors in FindTopNbyScoreDesc
// but for now instead of throwing errors, we find top N including empty scores
// and just file empty scores at the end of the list
func (s *WorkerTestSuite) TestFindTopNByScoreDescWithNils() {
	topicId := uint64(1)
	require := s.Require()

	reputerScoreEmas := []emissionstypes.Score{
		{TopicId: topicId, BlockHeight: 1, Address: s.AddrsStr(0), Score: alloraMath.NewDecFromInt64(80)},
		{TopicId: topicId, BlockHeight: 1, Address: s.AddrsStr(1), Score: alloraMath.NewDecFromInt64(100)},
		{}, //nolint:exhaustruct
	}

	// Actors with nil scores sent to the end
	topActors, allActorsSorted, actorIsTop := actorutils.FindTopNByScoreDesc(s.Ctx(), 3, reputerScoreEmas, 1)
	require.Len(topActors, 3)
	require.Equal(s.AddrsStr(1), topActors[0].Address)
	require.Equal(struct{}{}, actorIsTop[s.AddrsStr(1)])
	require.Equal(s.AddrsStr(0), topActors[1].Address)
	require.Equal(struct{}{}, actorIsTop[s.AddrsStr(0)])
	require.Equal("", topActors[2].Address)
	require.Len(allActorsSorted, 3)
}
