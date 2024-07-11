package queryserver_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestGetLatestInfererScore() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := "worker1"
	oldScore := types.Score{TopicId: topicId, BlockHeight: 1, Address: worker, Score: alloraMath.NewDecFromInt64(90)}
	newScore := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker, Score: alloraMath.NewDecFromInt64(95)}

	// Set an initial score for inferer and attempt to update with an older score
	_ = keeper.SetLatestInfererScore(ctx, topicId, worker, newScore)
	err := keeper.SetLatestInfererScore(ctx, topicId, worker, oldScore)
	s.Require().NoError(err, "Setting an older inferer score should not fail but should not update")

	req := &types.QueryLatestInfererScoreRequest{
		TopicId: topicId,
		Worker:  worker,
	}
	response, err := s.queryServer.GetLatestInfererScore(ctx, req)
	s.Require().NoError(err)

	updatedScore := response.Score
	s.Require().NotEqual(oldScore.Score, updatedScore.Score, "Older score should not replace newer score")
}

func (s *KeeperTestSuite) TestGetLatestForecasterScore() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := "worker1"
	forecaster := "forecaster1"
	newScore := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker, Score: alloraMath.NewDecFromInt64(95)}

	// Set a new score for forecaster
	_ = keeper.SetLatestForecasterScore(ctx, topicId, forecaster, newScore)

	req := &types.QueryLatestForecasterScoreRequest{
		TopicId:    topicId,
		Forecaster: forecaster,
	}
	response, err := s.queryServer.GetLatestForecasterScore(ctx, req)
	s.Require().NoError(err)

	forecasterScore := response.Score
	s.Require().Equal(newScore.Score, forecasterScore.Score, "Newer forecaster score should be set")
}

func (s *KeeperTestSuite) TestGetLatestReputerScore() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := "worker1"
	reputer := "reputer1"
	newScore := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker, Score: alloraMath.NewDecFromInt64(95)}

	// Set a new score for reputer
	_ = keeper.SetLatestReputerScore(ctx, topicId, reputer, newScore)

	req := &types.QueryLatestReputerScoreRequest{
		TopicId: topicId,
		Reputer: reputer,
	}
	response, err := s.queryServer.GetLatestReputerScore(ctx, req)
	s.Require().NoError(err)

	reputerScore := response.Score
	s.Require().Equal(newScore.Score, reputerScore.Score, "Newer reputer score should be set")
}
