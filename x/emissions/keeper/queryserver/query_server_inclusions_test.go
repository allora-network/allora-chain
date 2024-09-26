package queryserver_test

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestGetCountInfererInclusionsInTopic() {
	s.CreateOneTopic()
	ctx := s.ctx
	queryserver := s.queryServer
	topicId := uint64(1)
	inferer := "allo10es2a97cr7u2m3aa08tcu7yd0d300thdct45ve"

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
	forecaster := "allo1snm6pxg7p9jetmkhz0jz9ku3vdzmszegy9q5lh"

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
