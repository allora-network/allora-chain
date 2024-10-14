package queryserver_test

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestGetCountInfererInclusionsInTopic() {
	s.CreateOneTopic()
	ctx := s.ctx
	queryserver := s.queryServer
	topicId := uint64(1)
	inferer := s.addrsStr[0]

	results, err := queryserver.GetCountInfererInclusionsInTopic(ctx, &types.GetCountInfererInclusionsInTopicRequest{
		TopicId: topicId,
		Inferer: inferer,
	})
	s.Require().NoError(err)
	s.Equal(results.Count, uint64(0))

	err = s.emissionsKeeper.IncrementCountInfererInclusionsInTopic(s.ctx, topicId, inferer)
	s.Require().NoError(err)
	results, err = queryserver.GetCountInfererInclusionsInTopic(ctx, &types.GetCountInfererInclusionsInTopicRequest{
		TopicId: topicId,
		Inferer: inferer,
	})
	s.Require().NoError(err)
	s.Equal(results.Count, uint64(1))
}

func (s *QueryServerTestSuite) TestGetCountForecasterInclusionsInTopic() {
	s.CreateOneTopic()
	ctx := s.ctx
	queryserver := s.queryServer
	topicId := uint64(1)
	forecaster := s.addrsStr[0]

	results, err := queryserver.GetCountForecasterInclusionsInTopic(ctx, &types.GetCountForecasterInclusionsInTopicRequest{
		TopicId:    topicId,
		Forecaster: forecaster,
	})
	s.Require().NoError(err)
	s.Equal(results.Count, uint64(0))

	err = s.emissionsKeeper.IncrementCountForecasterInclusionsInTopic(s.ctx, topicId, forecaster)
	s.Require().NoError(err)
	results, err = queryserver.GetCountForecasterInclusionsInTopic(ctx, &types.GetCountForecasterInclusionsInTopicRequest{
		TopicId:    topicId,
		Forecaster: forecaster,
	})
	s.Require().NoError(err)
	s.Equal(results.Count, uint64(1))
}
