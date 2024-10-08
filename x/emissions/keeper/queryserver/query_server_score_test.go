package queryserver_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *QueryServerTestSuite) TestGetInfererScoreEma() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := s.addrsStr[0]
	newScore := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker, Score: alloraMath.NewDecFromInt64(95)}

	// Set an initial score for inferer and attempt to update with an older score
	err := keeper.SetInfererScoreEma(ctx, topicId, worker, newScore)
	s.Require().NoError(err, "Setting an inferer score should not fail")

	req := &types.GetInfererScoreEmaRequest{
		TopicId: topicId,
		Inferer: worker,
	}
	response, err := s.queryServer.GetInfererScoreEma(ctx, req)
	s.Require().NoError(err)

	s.Require().True(
		newScore.Score.Equal(response.Score.Score),
		"Score should be set %s | %s",
		newScore.Score.String(),
		response.Score.Score.String())
}

func (s *QueryServerTestSuite) TestGetForecasterScoreEma() {
	ctx := s.ctx
	keeper := s.emissionsKeeper
	topicId := uint64(1)
	worker := s.addrsStr[0]
	forecaster := s.addrsStr[2]
	newScore := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker, Score: alloraMath.NewDecFromInt64(95)}

	// Set a new score for forecaster
	_ = keeper.SetForecasterScoreEma(ctx, topicId, forecaster, newScore)

	req := &types.GetForecasterScoreEmaRequest{
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
	worker := s.addrsStr[0]
	reputer := s.addrsStr[2]
	newScore := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker, Score: alloraMath.NewDecFromInt64(95)}

	// Set a new score for reputer
	_ = keeper.SetReputerScoreEma(ctx, topicId, reputer, newScore)

	req := &types.GetReputerScoreEmaRequest{
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
	workerAddress := s.addrs[0]
	blockHeight := int64(105)

	// Insert scores for different workers and blocks
	for blockHeight := int64(100); blockHeight <= 110; blockHeight++ {
		// Scores for the targeted worker
		scoreForWorker := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     s.addrsStr[0],
			Score:       alloraMath.NewDecFromInt64(blockHeight),
		}
		_ = keeper.InsertWorkerInferenceScore(ctx, topicId, blockHeight, scoreForWorker)
	}

	// Get scores for the worker up to block 105
	req := &types.GetInferenceScoresUntilBlockRequest{
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
		Address:     s.addrsStr[0],
		Score:       alloraMath.NewDecFromInt64(95),
	}

	// Set the maximum number of scores using system parameters
	maxNumScores := 5
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = uint64(maxNumScores)
	err := keeper.SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < maxNumScores+2; i++ {
		err := keeper.InsertWorkerInferenceScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting worker inference score should not fail")
	}

	// Fetch scores to check if trimming happened
	req := &types.GetWorkerInferenceScoresAtBlockRequest{
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
			Address:     s.addrsStr[i-100],
			Score:       alloraMath.NewDecFromInt64(i),
		}
		err := keeper.InsertWorkerForecastScore(ctx, topicId, i, score)
		s.Require().NoError(err, "Inserting worker forecast score should not fail")
	}

	req := &types.GetForecastScoresUntilBlockRequest{
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
			Address:     s.addrsStr[i],
			Score:       alloraMath.NewDecFromInt64(int64(100 + i)),
		}
		_ = keeper.InsertWorkerForecastScore(ctx, topicId, blockHeight, score)
	}

	// Fetch scores at the specific block
	req := &types.GetWorkerForecastScoresAtBlockRequest{
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
			Address:     s.addrsStr[i],
			Score:       alloraMath.NewDecFromInt64(int64(100 + i)),
		}
		_ = keeper.InsertReputerScore(ctx, topicId, blockHeight, score)
	}

	// Fetch scores at the specific block
	req := &types.GetReputersScoresAtBlockRequest{
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
	reputer := s.addrsStr[1]

	// Attempt to fetch a coefficient before setting it
	req := &types.GetListeningCoefficientRequest{
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

func (s *QueryServerTestSuite) TestGetCurrentLowestInfererScore() {
	ctx, keeper, require := s.ctx, s.emissionsKeeper, s.Require()

	topicId := uint64(1)
	blockHeight := int64(100)

	// set cuurent inferer lowest score
	score := types.Score{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Address:     s.addrsStr[0],
		Score:       alloraMath.NewDecFromInt64(95),
	}
	err := keeper.SetLowestInfererScoreEma(ctx, topicId, score)
	require.NoError(err, "Setting inferer score ema should not fail")

	req := &types.GetCurrentLowestInfererScoreRequest{TopicId: topicId}
	response, err := s.queryServer.GetCurrentLowestInfererScore(ctx, req)
	require.NoError(err, "Fetching current lowest inferer score should not fail")
	require.True(
		response.Score.Score.Equal(alloraMath.NewDecFromInt64(95)),
		"The lowest score should be 95",
		response.Score,
	)
}

func (s *QueryServerTestSuite) TestGetCurrentLowestForecasterScore() {
	ctx, keeper, require := s.ctx, s.emissionsKeeper, s.Require()

	topicId := uint64(1)
	blockHeight := int64(100)

	// set cuurent forecaster lowest score
	score := types.Score{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Address:     s.addrsStr[0],
		Score:       alloraMath.NewDecFromInt64(95),
	}
	err := keeper.SetLowestForecasterScoreEma(ctx, topicId, score)
	require.NoError(err, "Setting forecaster score ema should not fail")

	req := &types.GetCurrentLowestForecasterScoreRequest{TopicId: topicId}
	response, err := s.queryServer.GetCurrentLowestForecasterScore(ctx, req)
	require.NoError(err, "Fetching current lowest forecaster score should not fail")
	require.True(
		response.Score.Score.Equal(score.Score),
		"The lowest score should be 95",
		response.Score,
	)
}

func (s *QueryServerTestSuite) TestGetCurrentLowestReputerScore() {
	ctx, keeper, require := s.ctx, s.emissionsKeeper, s.Require()

	topicId := uint64(1)
	blockHeight := int64(100)

	// set cuurent forecaster lowest score
	score := types.Score{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Address:     s.addrsStr[0],
		Score:       alloraMath.NewDecFromInt64(95),
	}
	err := keeper.SetLowestReputerScoreEma(ctx, topicId, score)
	require.NoError(err, "Setting reputer score ema should not fail")

	req := &types.GetCurrentLowestReputerScoreRequest{TopicId: topicId}
	response, err := s.queryServer.GetCurrentLowestReputerScore(ctx, req)
	require.NoError(err, "Fetching current lowest reputer score should not fail")
	require.True(
		response.Score.Score.Equal(alloraMath.NewDecFromInt64(95)),
		"The lowest score should be 95",
		response.Score,
	)
}
