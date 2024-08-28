package msgserver

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *MsgServerInternalTestSuite) TestGetLowScoreFromAllInferences() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := uint64(1)
	blockHeightInferences := int64(10)

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"

	score1 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker3, Score: alloraMath.NewDecFromInt64(99)}
	_ = k.SetInfererScoreEma(ctx, topicId, worker1, score1)
	_ = k.SetInfererScoreEma(ctx, topicId, worker2, score2)
	_ = k.SetInfererScoreEma(ctx, topicId, worker3, score3)

	allInferences := types.Inferences{
		Inferences: []*types.Inference{
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.52")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.71")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.71")},
		},
	}
	lowScore, lowScoreIndex, err := lowestInfererScoreEma(ctx, k, topicId, allInferences)
	s.Require().NoError(err)
	s.Require().Equal(lowScore, score2)
	s.Require().Equal(lowScoreIndex, 1)
}

func (s *MsgServerInternalTestSuite) TestGetLowScoreFromAllForecasts() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := uint64(1)
	blockHeightInferences := int64(10)

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"

	score1 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker3, Score: alloraMath.NewDecFromInt64(99)}
	_ = k.SetForecasterScoreEma(ctx, topicId, worker1, score1)
	_ = k.SetForecasterScoreEma(ctx, topicId, worker2, score2)
	_ = k.SetForecasterScoreEma(ctx, topicId, worker3, score3)

	allForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
				Forecaster:  worker1,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: worker1,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
					{
						Inferer: worker2,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
				Forecaster:  worker2,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: worker1,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
					{
						Inferer: worker2,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
				Forecaster:  worker3,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: worker1,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
					{
						Inferer: worker2,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
				},
			},
		},
	}
	lowScore, lowScoreIndex, err := lowestForecasterScoreEma(ctx, k, topicId, allForecasts)
	s.Require().NoError(err)
	s.Require().Equal(lowScore, score2)
	s.Require().Equal(lowScoreIndex, 1)
}
