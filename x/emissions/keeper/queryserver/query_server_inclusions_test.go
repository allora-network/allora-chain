package queryserver_test

import (
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestGetCountInfererInclusionsInTopic() {
	ctx := s.Ctx()
	queryserver := s.EmissionsQueryServer()
	topicId := uint64(1)
	inferer := s.AddrsStr()[0]

	results, err := queryserver.GetCountInfererInclusionsInTopic(ctx, &types.GetCountInfererInclusionsInTopicRequest{
		TopicId: topicId,
		Inferer: inferer,
	})
	s.Require().NoError(err)
	s.Equal(results.Count, uint64(0))

	err = s.EmissionsKeeper().IncrementCountInfererInclusionsInTopic(s.Ctx(), topicId, inferer)
	s.Require().NoError(err)
	results, err = queryserver.GetCountInfererInclusionsInTopic(ctx, &types.GetCountInfererInclusionsInTopicRequest{
		TopicId: topicId,
		Inferer: inferer,
	})
	s.Require().NoError(err)
	s.Equal(results.Count, uint64(1))
}

func (s *QueryServerTestSuite) TestGetCountForecasterInclusionsInTopic() {
	ctx := s.Ctx()
	queryserver := s.EmissionsQueryServer()
	topicId := uint64(1)
	forecaster := s.AddrsStr()[0]

	results, err := queryserver.GetCountForecasterInclusionsInTopic(ctx, &types.GetCountForecasterInclusionsInTopicRequest{
		TopicId:    topicId,
		Forecaster: forecaster,
	})
	s.Require().NoError(err)
	s.Equal(results.Count, uint64(0))

	err = s.EmissionsKeeper().IncrementCountForecasterInclusionsInTopic(s.Ctx(), topicId, forecaster)
	s.Require().NoError(err)
	results, err = queryserver.GetCountForecasterInclusionsInTopic(ctx, &types.GetCountForecasterInclusionsInTopicRequest{
		TopicId:    topicId,
		Forecaster: forecaster,
	})
	s.Require().NoError(err)
	s.Equal(results.Count, uint64(1))
}
