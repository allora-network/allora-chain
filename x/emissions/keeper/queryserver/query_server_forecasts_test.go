package queryserver_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestGetForecastsAtBlock() {
	ctx := s.Ctx()
	queryserver := s.EmissionsQueryServer()
	topicId := uint64(1)
	blockHeight := types.BlockHeight(100)
	expectedForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:     topicId,
				Forecaster:  s.AddrsStr(6),
				BlockHeight: blockHeight,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: s.AddrsStr(4),
						Value:   alloraMath.MustNewDecFromString("0.5"),
					},
				},
				ExtraData: nil,
			},
			{
				TopicId:     topicId,
				Forecaster:  s.AddrsStr(7),
				BlockHeight: blockHeight,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: s.AddrsStr(4),
						Value:   alloraMath.MustNewDecFromString("0.5"),
					},
				},
				ExtraData: nil,
			},
		},
	}

	// Assume InsertActiveForecasts correctly sets up forecasts
	nonce := types.Nonce{BlockHeight: blockHeight}
	err := s.WorkerKeeper().InsertActiveForecasts(ctx, topicId, nonce.BlockHeight, expectedForecasts)
	s.Require().NoError(err)

	results, err := queryserver.GetForecastsAtBlock(
		ctx,
		&types.GetForecastsAtBlockRequest{
			TopicId:     topicId,
			BlockHeight: blockHeight,
		},
	)
	s.Require().NoError(err)
	s.Equal(results.Forecasts.Forecasts[0].Forecaster, expectedForecasts.Forecasts[0].Forecaster)
	s.Equal(results.Forecasts.Forecasts[1].Forecaster, expectedForecasts.Forecasts[1].Forecaster)
}
