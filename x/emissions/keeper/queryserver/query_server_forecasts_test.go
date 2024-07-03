package queryserver_test

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestGetForecastsAtBlock() {
	s.CreateOneTopic()
	ctx := s.ctx
	keeper := s.emissionsKeeper
	queryserver := s.queryServer
	topicId := uint64(1)
	blockHeight := types.BlockHeight(100)
	expectedForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:    topicId,
				Forecaster: "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve",
			},
			{
				TopicId:    topicId,
				Forecaster: "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh",
			},
		},
	}

	// Assume InsertForecasts correctly sets up forecasts
	nonce := types.Nonce{BlockHeight: int64(blockHeight)}
	err := keeper.InsertForecasts(ctx, topicId, nonce, expectedForecasts)
	s.Require().NoError(err)

	results, err := queryserver.GetForecastsAtBlock(
		ctx,
		&types.QueryForecastsAtBlockRequest{
			TopicId:     topicId,
			BlockHeight: blockHeight,
		},
	)
	s.Require().NoError(err)
	s.Equal(results.Forecasts.Forecasts[0].Forecaster, expectedForecasts.Forecasts[0].Forecaster)
	s.Equal(results.Forecasts.Forecasts[1].Forecaster, expectedForecasts.Forecasts[1].Forecaster)
}
