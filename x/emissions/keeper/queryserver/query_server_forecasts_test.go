package queryserver_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestGetForecastsAtBlock() {
	s.CreateOneTopic()
	ctx := s.ctx
	keeper := s.emissionsKeeper
	queryserver := s.queryServer
	topicId := uint64(1)
	blockHeight := types.BlockHeight(100)
	expectedForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:     topicId,
				Forecaster:  s.addrsStr[6],
				BlockHeight: blockHeight,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: s.addrsStr[4],
						Value:   alloraMath.MustNewDecFromString("0.5"),
					},
				},
				ExtraData: nil,
			},
			{
				TopicId:     topicId,
				Forecaster:  s.addrsStr[7],
				BlockHeight: blockHeight,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: s.addrsStr[4],
						Value:   alloraMath.MustNewDecFromString("0.5"),
					},
				},
				ExtraData: nil,
			},
		},
	}

	// Assume InsertActiveForecasts correctly sets up forecasts
	nonce := types.Nonce{BlockHeight: blockHeight}
	err := keeper.InsertActiveForecasts(ctx, topicId, nonce.BlockHeight, expectedForecasts)
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

func (s *QueryServerTestSuite) TestGetActiveForecastersForTopic() {
	s.CreateOneTopic()
	ctx := s.ctx
	keeper := s.emissionsKeeper
	queryServer := s.queryServer
	topicId := uint64(1)

	// Add some active forecasters
	activeForecasters := []string{
		s.addrsStr[0],
		s.addrsStr[1],
		s.addrsStr[2],
	}

	for _, forecaster := range activeForecasters {
		err := keeper.AddActiveForecaster(ctx, topicId, forecaster)
		s.Require().NoError(err)
	}

	// Query for active forecasters
	response, err := queryServer.GetActiveForecastersForTopic(
		ctx,
		&types.GetActiveForecastersForTopicRequest{
			TopicId: topicId,
		},
	)

	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Equal(len(activeForecasters), len(response.Forecasters))

	// Check if all added forecasters are in the response
	for _, forecaster := range activeForecasters {
		s.Require().Contains(response.Forecasters, forecaster)
	}

	// Test with non-existent topic
	nonExistentTopicId := uint64(999)
	_, err = queryServer.GetActiveForecastersForTopic(
		ctx,
		&types.GetActiveForecastersForTopicRequest{
			TopicId: nonExistentTopicId,
		},
	)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "not found")

	// Test with no active forecasters
	emptyTopicId := s.CreateOneTopic()
	emptyResponse, err := queryServer.GetActiveForecastersForTopic(
		ctx,
		&types.GetActiveForecastersForTopicRequest{
			TopicId: emptyTopicId,
		},
	)
	s.Require().NoError(err)
	s.Require().NotNil(emptyResponse)
	s.Require().Empty(emptyResponse.Forecasters)
}
