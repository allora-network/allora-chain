package queryserver_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestGetInfererWeight() {
	ctx := s.Ctx()
	queryServer := s.EmissionsQueryServer()

	topicId := uint64(1)
	worker := s.AddrsStr(0)
	weight := alloraMath.NewDecFromInt64(100)

	// Set initial weight
	err := s.WeightstsKeeper().SetLatestInfererWeight(ctx, topicId, worker, weight)
	s.Require().NoError(err, "Setting inferer weight should not fail")

	req := &types.GetLatestInfererWeightRequest{
		TopicId: topicId,
		ActorId: worker,
	}
	response, err := queryServer.GetLatestInfererWeight(ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(weight, response.Weight, "Retrieved weight should match set weight")

	// Test non-existent worker
	nonExistentWorker := s.AddrsStr(1)
	req.ActorId = nonExistentWorker
	response, err = queryServer.GetLatestInfererWeight(ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(alloraMath.ZeroDec(), response.Weight, "Non-existent worker should have zero weight")
}

func (s *QueryServerTestSuite) TestGetForecasterWeight() {
	ctx := s.Ctx()
	queryServer := s.EmissionsQueryServer()

	topicId := uint64(1)
	forecaster := s.AddrsStr(0)
	weight := alloraMath.NewDecFromInt64(100)

	// Set initial weight
	err := s.WeightstsKeeper().SetLatestForecasterWeight(ctx, topicId, forecaster, weight)
	s.Require().NoError(err, "Setting forecaster weight should not fail")

	req := &types.GetLatestForecasterWeightRequest{
		TopicId: topicId,
		ActorId: forecaster,
	}
	response, err := queryServer.GetLatestForecasterWeight(ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(weight, response.Weight, "Retrieved weight should match set weight")

	// Test non-existent forecaster
	nonExistentForecaster := s.AddrsStr(1)
	req.ActorId = nonExistentForecaster
	response, err = queryServer.GetLatestForecasterWeight(ctx, req)
	s.Require().NoError(err)
	s.Require().Equal(alloraMath.ZeroDec(), response.Weight, "Non-existent forecaster should have zero weight")
}

func (s *QueryServerTestSuite) TestGetLatestStdnorm() {
	ctx := s.Ctx()
	queryServer := s.EmissionsQueryServer()

	topicId := uint64(1)
	stdnorm := alloraMath.NewDecFromInt64(100)

	// Set initial stdnorm
	err := s.WeightstsKeeper().SetLatestRegretStdNorm(ctx, topicId, stdnorm)
	s.Require().NoError(err, "Setting latest stdnorm should not fail")

	req := &types.GetLatestRegretStdNormRequest{
		TopicId: topicId,
	}
	response, err := queryServer.GetLatestRegretStdNorm(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(stdnorm, response.Value, "Retrieved stdnorm should match set stdnorm")

	// Test non-existent topic
	nonExistentTopicId := uint64(999)
	req = &types.GetLatestRegretStdNormRequest{
		TopicId: nonExistentTopicId,
	}
	response, err = queryServer.GetLatestRegretStdNorm(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(response, "The response should not be nil")
	s.Require().Equal(alloraMath.ZeroDec(), response.Value, "Non-existent topic should return zero stdnorm")
}
