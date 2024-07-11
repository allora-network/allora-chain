package queryserver_test

import (
	"strconv"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (s *KeeperTestSuite) TestGetInferenceScoresUntilBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	workerAddress := sdk.AccAddress("allo16jmt7f7r4e6j9k4ds7jgac2t4k4cz0wthv4u88")
	blockHeight := int64(105)

	// Insert scores for different workers and blocks
	for blockHeight := int64(100); blockHeight <= 110; blockHeight++ {
		// Scores for the targeted worker
		scoreForWorker := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     workerAddress.String(),
			Score:       alloraMath.NewDecFromInt64(blockHeight),
		}
		_ = keeper.InsertWorkerInferenceScore(ctx, topicId, blockHeight, scoreForWorker)
	}

	// Get scores for the worker up to block 105
	req := &types.QueryInferenceScoresUntilBlockRequest{
		TopicId:     topicId,
		BlockHeight: blockHeight,
	}
	response, err := s.queryServer.GetInferenceScoresUntilBlock(ctx, req)
	s.Require().NoError(err)
	scores := response.Scores

	s.Require().NoError(err, "Fetching worker inference scores until block should not fail")
	s.Require().Len(scores, 6, "Should retrieve correct number of scores up to block 105")

	// Verify that the scores are correct and ordered as expected (descending block number)
	expectedBlock := blockHeight
	for _, score := range scores {
		s.Require().Equal(workerAddress.String(), score.Address, "Only scores for the specified worker should be returned")
		s.Require().Equal(expectedBlock, score.BlockHeight, "Scores should be returned in descending order by block")
		s.Require().Equal(alloraMath.NewDecFromInt64(expectedBlock), score.Score, "Score value should match expected")
		expectedBlock--
	}
}

func (s *KeeperTestSuite) TestGetWorkerInferenceScoresAtBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockHeight := int64(100)
	score := types.Score{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Address:     "worker1",
		Score:       alloraMath.NewDecFromInt64(95),
	}

	// Set the maximum number of scores using system parameters
	maxNumScores := uint64(5)
	params := types.Params{MaxSamplesToScaleScores: maxNumScores}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < int(maxNumScores+2); i++ {
		err := keeper.InsertWorkerInferenceScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting worker inference score should not fail")
	}

	// Fetch scores to check if trimming happened
	req := &types.QueryWorkerInferenceScoresAtBlockRequest{
		TopicId:     topicId,
		BlockHeight: blockHeight,
	}
	response, err := s.queryServer.GetWorkerInferenceScoresAtBlock(ctx, req)
	scores := response.Scores

	s.Require().NoError(err, "Fetching scores at block should not fail")
	s.Require().Len(scores.Scores, int(maxNumScores), "Scores should not exceed the maximum limit")
}

func (s *KeeperTestSuite) TestGetForecastScoresUntilBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockHeight := int64(105)

	// Insert scores for the worker at various blocks
	for i := int64(100); i <= 110; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: i,
			Score:       alloraMath.NewDecFromInt64(i),
		}
		_ = keeper.InsertWorkerForecastScore(ctx, topicId, i, score)
	}

	req := &types.QueryForecastScoresUntilBlockRequest{
		TopicId:     topicId,
		BlockHeight: blockHeight,
	}
	response, err := s.queryServer.GetForecastScoresUntilBlock(ctx, req)
	scores := response.Scores
	s.Require().NoError(err, "Fetching worker forecast scores until block should not fail")
	s.Require().Len(scores, 6, "Should retrieve correct number of scores up to block 105")
}

func (s *KeeperTestSuite) TestGetWorkerForecastScoresAtBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockHeight := int64(100)

	// Insert scores at the block
	for i := 0; i < 5; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     "worker" + strconv.Itoa(i+1),
			Score:       alloraMath.NewDecFromInt64(int64(100 + i)),
		}
		_ = keeper.InsertWorkerForecastScore(ctx, topicId, blockHeight, score)
	}

	// Fetch scores at the specific block
	req := &types.QueryWorkerForecastScoresAtBlockRequest{
		TopicId:     topicId,
		BlockHeight: blockHeight,
	}
	response, err := s.queryServer.GetWorkerForecastScoresAtBlock(ctx, req)
	scores := response.Scores

	s.Require().NoError(err, "Fetching forecast scores at block should not fail")
	s.Require().Len(scores.Scores, 5, "Should retrieve all scores at the block")
}

func (s *KeeperTestSuite) TestGetReputersScoresAtBlock() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	blockHeight := int64(100)

	// Insert multiple scores at the block
	for i := 0; i < 5; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     "reputer" + strconv.Itoa(i+1),
			Score:       alloraMath.NewDecFromInt64(int64(100 + i)),
		}
		_ = keeper.InsertReputerScore(ctx, topicId, blockHeight, score)
	}

	// Fetch scores at the specific block
	req := &types.QueryReputersScoresAtBlockRequest{
		TopicId:     topicId,
		BlockHeight: blockHeight,
	}
	response, err := s.queryServer.GetReputersScoresAtBlock(ctx, req)
	s.Require().NoError(err)
	scores := response.Scores

	s.Require().NoError(err, "Fetching reputer scores at block should not fail")
	s.Require().Len(scores.Scores, 5, "Should retrieve all scores at the block")
}
