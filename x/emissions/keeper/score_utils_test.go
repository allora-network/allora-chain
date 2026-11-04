package keeper_test

import (
	"context"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestGetLowestScore() {
	k := s.ScoresKeeper()
	ctx := s.Ctx()
	topicId := uint64(1)

	testCases := []struct {
		name           string
		setScore       func(ctx context.Context, topicId uint64, addr string, score types.Score) error
		getLowestScore func(ctx context.Context, topicId uint64, addresses []string) (types.Score, error)
	}{
		{
			name:           "inferences",
			setScore:       k.SetInfererScoreEma,
			getLowestScore: k.GetLowestScoreFromAllInferers,
		}, {
			name:           "forecasts",
			setScore:       k.SetForecasterScoreEma,
			getLowestScore: k.GetLowestScoreFromAllForecasters,
		}, {
			name:           "loss bundles",
			setScore:       k.SetReputerScoreEma,
			getLowestScore: k.GetLowestScoreFromAllReputers,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			workers := []string{s.AddrsStr(0), s.AddrsStr(1), s.AddrsStr(2)}
			scores := []types.Score{
				{TopicId: topicId, BlockHeight: 2, Address: workers[0], Score: alloraMath.NewDecFromInt64(95)},
				{TopicId: topicId, BlockHeight: 2, Address: workers[1], Score: alloraMath.NewDecFromInt64(90)},
				{TopicId: topicId, BlockHeight: 2, Address: workers[2], Score: alloraMath.NewDecFromInt64(99)},
			}

			for i := range workers {
				_ = tc.setScore(ctx, topicId, workers[i], scores[i])
			}

			lowScore, err := tc.getLowestScore(ctx, topicId, workers)
			s.Require().NoError(err)
			s.Require().Equal(lowScore, scores[1])
		})
	}
}
