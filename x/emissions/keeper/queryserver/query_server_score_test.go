package queryserver_test

import (
	"strconv"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *QueryServerTestSuite) TestGetInfererScoreEma() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := "worker1"
	expected := types.Score{TopicId: topicId, BlockHeight: 1, Address: worker, Score: alloraMath.NewDecFromInt64(90)}

	// Set an initial score for inferer and attempt to update with an older score
	err := keeper.SetInfererScoreEma(ctx, topicId, worker, expected)
	s.Require().NoError(err, "Setting an older inferer score should not fail but should not update")

	req := &types.QueryInfererScoreEmaRequest{
		TopicId: topicId,
		Inferer: worker,
	}
	response, err := s.queryServer.GetInfererScoreEma(ctx, req)
	s.Require().NoError(err)

	result := response.Score
	s.Require().Equal(expected.Score, result.Score, "Retrieved data should match inserted data")
}

func (s *QueryServerTestSuite) TestGetForecasterScoreEma() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := "worker1"
	forecaster := "forecaster1"
	newScore := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker, Score: alloraMath.NewDecFromInt64(95)}

	// Set a new score for forecaster
	_ = keeper.SetForecasterScoreEma(ctx, topicId, forecaster, newScore)

	req := &types.QueryForecasterScoreEmaRequest{
		TopicId:    topicId,
		Forecaster: forecaster,
	}
	response, err := s.queryServer.GetForecasterScoreEma(ctx, req)
	s.Require().NoError(err)

	forecasterScore := response.Score
	s.Require().Equal(newScore.Score, forecasterScore.Score, "Newer forecaster score should be set")
}

func (s *QueryServerTestSuite) TestGetReputerScoreEma() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := "worker1"
	reputer := "reputer1"
	newScore := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker, Score: alloraMath.NewDecFromInt64(95)}

	// Set a new score for reputer
	_ = keeper.SetReputerScoreEma(ctx, topicId, reputer, newScore)

	req := &types.QueryReputerScoreEmaRequest{
		TopicId: topicId,
		Reputer: reputer,
	}
	response, err := s.queryServer.GetReputerScoreEma(ctx, req)
	s.Require().NoError(err)

	reputerScore := response.Score
	s.Require().Equal(newScore.Score, reputerScore.Score, "Newer reputer score should be set")
}

func (s *QueryServerTestSuite) TestGetInferenceScoresUntilBlock() {
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

func (s *QueryServerTestSuite) TestGetWorkerInferenceScoresAtBlock() {
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
	maxNumScores := 5
	params := types.Params{MaxSamplesToScaleScores: uint64(maxNumScores)}
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < maxNumScores+2; i++ {
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
	s.Require().Len(scores.Scores, maxNumScores, "Scores should not exceed the maximum limit")
}

func (s *QueryServerTestSuite) TestGetForecastScoresUntilBlock() {
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

func (s *QueryServerTestSuite) TestGetWorkerForecastScoresAtBlock() {
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

func (s *QueryServerTestSuite) TestGetReputersScoresAtBlock() {
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

func (s *QueryServerTestSuite) TestGetListeningCoefficient() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	reputer := "sampleReputerAddress"

	// Attempt to fetch a coefficient before setting it
	req := &types.QueryListeningCoefficientRequest{
		TopicId: topicId,
		Reputer: reputer,
	}
	response, err := s.queryServer.GetListeningCoefficient(ctx, req)
	s.Require().NoError(err)
	defaultCoef := response.ListeningCoefficient

	s.Require().NoError(err, "Fetching coefficient should not fail when not set")
	s.Require().Equal(alloraMath.NewDecFromInt64(1), defaultCoef.Coefficient, "Should return the default coefficient when not set")

	// Now set a specific coefficient
	setCoef := types.ListeningCoefficient{
		Coefficient: alloraMath.NewDecFromInt64(5),
	}
	_ = keeper.SetListeningCoefficient(ctx, topicId, reputer, setCoef)

	// Fetch and verify the coefficient after setting
	response, err = s.queryServer.GetListeningCoefficient(ctx, req)
	s.Require().NoError(err)
	fetchedCoef := response.ListeningCoefficient

	s.Require().NoError(err, "Fetching coefficient should not fail after setting")
	s.Require().Equal(setCoef.Coefficient, fetchedCoef.Coefficient, "The fetched coefficient should match the set value")
}
