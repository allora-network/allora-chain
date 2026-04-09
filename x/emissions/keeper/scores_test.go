package keeper_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestGetScoreEmas() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	worker := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	forecaster := "allo13kenskkx7e0v253m3kcgwfc67cmx00fgwpgj6h"
	reputer := "allo144nqxgt6jdrm4srzzgx4dvz04hd8q2e8cel9hu"

	// Test getting latest scores when none are set
	infererScore, err := k.GetInfererScoreEma(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching latest inferer score should not fail")
	s.Require().Equal(types.Score{
		TopicId:     topicId,
		BlockHeight: 0,
		Address:     worker,
		Score:       alloraMath.ZeroDec(),
	}, infererScore, "Inferer score should be zero if not set")

	forecasterScore, err := k.GetForecasterScoreEma(ctx, topicId, forecaster)
	s.Require().NoError(err, "Fetching latest forecaster score should not fail")
	s.Require().Equal(types.Score{
		TopicId:     topicId,
		BlockHeight: 0,
		Address:     forecaster,
		Score:       alloraMath.ZeroDec(),
	}, forecasterScore, "Forecaster score should be empty if not set")

	reputerScore, err := k.GetReputerScoreEma(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching latest reputer score should not fail")
	s.Require().Equal(types.Score{
		TopicId:     topicId,
		BlockHeight: 0,
		Address:     reputer,
		Score:       alloraMath.ZeroDec(),
	}, reputerScore, "Reputer score should be empty if not set")
}

func (s *KeeperTestSuite) TestSetScoreEmas() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	worker := "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5"
	forecaster := "allo13kenskkx7e0v253m3kcgwfc67cmx00fgwpgj6h"
	reputer := "allo144nqxgt6jdrm4srzzgx4dvz04hd8q2e8cel9hu"
	score := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker, Score: alloraMath.NewDecFromInt64(95)}

	// Set an initial score for inferer and attempt to update with an older score
	err := k.SetInfererScoreEma(ctx, topicId, worker, score)
	s.Require().NoError(err)
	infererScore, err := k.GetInfererScoreEma(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(score.Score, infererScore.Score, "Newer inferer score should be set")

	// Set a new score for forecaster
	err = k.SetForecasterScoreEma(ctx, topicId, forecaster, score)
	s.Require().NoError(err)
	forecasterScore, err := k.GetForecasterScoreEma(ctx, topicId, forecaster)
	s.Require().NoError(err)
	s.Require().Equal(score.Score, forecasterScore.Score, "Newer forecaster score should be set")

	// Set a new score for reputer
	err = k.SetReputerScoreEma(ctx, topicId, reputer, score)
	s.Require().NoError(err)
	reputerScore, err := k.GetReputerScoreEma(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().Equal(score.Score, reputerScore.Score, "Newer reputer score should be set")
}

func (s *KeeperTestSuite) TestInsertWorkerInferenceScore() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	blockHeight := int64(100)
	score := types.Score{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Address:     "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
		Score:       alloraMath.NewDecFromInt64(95),
	}

	// Set the maximum number of scores using system parameters
	maxNumScores := 5
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = uint64(maxNumScores)
	params.MaxTopInferersToReward = 1
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < maxNumScores+2; i++ {
		err := k.InsertWorkerInferenceScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting worker inference score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := k.GetWorkerInferenceScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching scores at block should not fail")
	s.Require().Len(scores.Scores, maxNumScores, "Scores should not exceed the maximum limit")
}

func (s *KeeperTestSuite) TestInsertWorkerInferenceScore2() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	blockHeight := int64(100)

	// Set the maximum number of scores using system parameters
	maxNumScores := 5
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = uint64(maxNumScores)
	params.MaxTopInferersToReward = 1
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < maxNumScores+2; i++ { // Inserting 7 scores where the limit is 5
		scoreValue := alloraMath.NewDecFromInt64(int64(90 + i)) // Increment score value to simulate variation
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
			Score:       scoreValue,
		}
		err := k.InsertWorkerInferenceScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting worker inference score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := k.GetWorkerInferenceScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching scores at block should not fail")
	s.Require().Len(scores.Scores, maxNumScores, "Scores should not exceed the maximum limit")

	// Check that the retained scores are the last five inserted
	for idx, score := range scores.Scores {
		expectedScoreValue := alloraMath.NewDecFromInt64(int64(92 + idx)) // Expecting the last 5 scores: 94, 95, 96, 97
		s.Require().Equal(expectedScoreValue, score.Score, "Score should match the expected last scores")
	}
}

func (s *KeeperTestSuite) TestGetInferenceScoresUntilBlock() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	workerAddress := s.Addrs(0)
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
		err := k.InsertWorkerInferenceScore(ctx, topicId, blockHeight, scoreForWorker)
		s.Require().NoError(err, "Inserting worker inference score should not fail")
	}

	// Get scores for the worker up to block 105
	scores, err := k.GetInferenceScoresUntilBlock(ctx, topicId, blockHeight)
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

func (s *KeeperTestSuite) TestInsertWorkerForecastScore() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	blockHeight := int64(100)

	// Set the maximum number of scores using system parameters
	maxNumScores := 5
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = uint64(maxNumScores)
	params.MaxTopForecastersToReward = 1
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < maxNumScores+2; i++ { // Inserting 7 scores where the limit is 5
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     "allo15lvs3m3urm4kts4tp2um5u3aeuz3whqrhz47r5",
			Score:       alloraMath.NewDecFromInt64(int64(90 + i)), // Increment score value to simulate variation
		}
		err := k.InsertWorkerForecastScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting worker forecast score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := k.GetWorkerForecastScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching forecast scores at block should not fail")
	s.Require().Len(scores.Scores, maxNumScores, "Scores should not exceed the maximum limit")
}

func (s *KeeperTestSuite) TestGetForecastScoresUntilBlock() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	blockHeight := int64(105)

	// Insert scores for the worker at various blocks
	for i := int64(100); i <= 110; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: i,
			Score:       alloraMath.NewDecFromInt64(i),
			Address:     s.AddrsStr(0),
		}
		err := k.InsertWorkerForecastScore(ctx, topicId, i, score)
		s.Require().NoError(err, "Inserting worker forecast score should not fail")
	}

	// Get forecast scores for the worker up to block 105
	scores, err := k.GetForecastScoresUntilBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching worker forecast scores until block should not fail")
	s.Require().Len(scores, 6, "Should retrieve correct number of scores up to block 105")
}

func (s *KeeperTestSuite) TestGetWorkerForecastScoresAtBlock() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	blockHeight := int64(100)

	// Insert scores at the block
	for i := 0; i < 5; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     s.AddrsStr(i),
			Score:       alloraMath.NewDecFromInt64(int64(100 + i)),
		}
		err := k.InsertWorkerForecastScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting worker forecast score should not fail")
	}

	// Fetch scores at the specific block
	scores, err := k.GetWorkerForecastScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching forecast scores at block should not fail")
	s.Require().Len(scores.Scores, 5, "Should retrieve all scores at the block")
}

func (s *KeeperTestSuite) TestInsertReputerScore() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	blockHeight := int64(100)

	// Set the maximum number of scores using system parameters
	maxNumScores := 5
	params := types.DefaultParams()
	params.MaxSamplesToScaleScores = uint64(maxNumScores)
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err, "Setting parameters should not fail")

	// Insert scores more than the max limit to test trimming
	for i := 0; i < maxNumScores+2; i++ { // Inserting 7 scores where the limit is 5
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     "allo144nqxgt6jdrm4srzzgx4dvz04hd8q2e8cel9hu",
			Score:       alloraMath.NewDecFromInt64(int64(90 + i)), // Increment score value to simulate variation
		}
		err := k.InsertReputerScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting reputer score should not fail")
	}

	// Fetch scores to check if trimming happened
	scores, err := k.GetReputersScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching reputer scores at block should not fail")
	s.Require().Len(scores.Scores, maxNumScores, "Scores should not exceed the maximum limit")
}

func (s *KeeperTestSuite) TestGetReputersScoresAtBlock() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	blockHeight := int64(100)

	// Insert multiple scores at the block
	for i := 0; i < 5; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     s.AddrsStr(i),
			Score:       alloraMath.NewDecFromInt64(int64(100 + i)),
		}
		err := k.InsertReputerScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err, "Inserting reputer score should not fail")
	}

	// Fetch scores at the specific block
	scores, err := k.GetReputersScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err, "Fetching reputer scores at block should not fail")
	s.Require().Len(scores.Scores, 5, "Should retrieve all scores at the block")
}

func (s *KeeperTestSuite) TestSetListeningCoefficient() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	reputer := "allo17srupely9uux7axep5shdsezva4znz6g30ntdw"

	// Define a listening coefficient
	coefficient := types.ListeningCoefficient{
		Coefficient: alloraMath.NewDecFromInt64(10),
	}

	// Set the listening coefficient
	err := k.SetListeningCoefficient(ctx, topicId, reputer, coefficient)
	s.Require().NoError(err, "Setting listening coefficient should not fail")

	// Retrieve the set coefficient to verify it was set correctly
	retrievedCoef, err := k.GetListeningCoefficient(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching listening coefficient should not fail")
	s.Require().Equal(coefficient.Coefficient, retrievedCoef.Coefficient, "The retrieved coefficient should match the set value")
}

func (s *KeeperTestSuite) TestGetListeningCoefficient() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	reputer := "allo17srupely9uux7axep5shdsezva4znz6g30ntdw"

	// Attempt to fetch a coefficient before setting it
	defaultCoef, err := k.GetListeningCoefficient(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching coefficient should not fail when not set")
	s.Require().Equal(alloraMath.NewDecFromInt64(1), defaultCoef.Coefficient, "Should return the default coefficient when not set")

	// Now set a specific coefficient
	setCoef := types.ListeningCoefficient{
		Coefficient: alloraMath.NewDecFromInt64(5),
	}
	err = k.SetListeningCoefficient(ctx, topicId, reputer, setCoef)
	s.Require().NoError(err, "Setting listening coefficient should not fail")
	// Fetch and verify the coefficient after setting
	fetchedCoef, err := k.GetListeningCoefficient(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching coefficient should not fail after setting")
	s.Require().Equal(setCoef.Coefficient, fetchedCoef.Coefficient, "The fetched coefficient should match the set value")
}

// REWARD FRACTION

func (s *KeeperTestSuite) TestSetPreviousReputerRewardFraction() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	reputer := s.AddrsStr(2)

	// Define a reward fraction to set
	rewardFraction := alloraMath.NewDecFromInt64(75) // Assuming 0.75 as a fraction example

	// Set the reward fraction
	err := k.SetPreviousReputerRewardFraction(ctx, topicId, reputer, rewardFraction)
	s.Require().NoError(err, "Setting previous reputer reward fraction should not fail")

	// Verify by fetching the same
	fetchedReward, noPrior, err := k.GetPreviousReputerRewardFraction(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching the set reward fraction should not fail")
	s.Require().True(fetchedReward.Equal(rewardFraction), "The fetched reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value when set")
}

func (s *KeeperTestSuite) TestGetPreviousReputerRewardFraction() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	reputer := s.AddrsStr(2)

	// Attempt to fetch a reward fraction before setting it
	defaultReward, _, err := k.GetPreviousReputerRewardFraction(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching reward fraction should not fail when not set")
	s.Require().True(defaultReward.IsZero(), "Should return zero reward fraction when not set")

	// Now set a specific reward fraction
	setReward := alloraMath.NewDecFromInt64(50) // Assuming 0.50 as a fraction example
	err = k.SetPreviousReputerRewardFraction(ctx, topicId, reputer, setReward)
	s.Require().NoError(err, "Setting previous reputer reward fraction should not fail")

	// Fetch and verify the reward fraction after setting
	fetchedReward, noPrior, err := k.GetPreviousReputerRewardFraction(ctx, topicId, reputer)
	s.Require().NoError(err, "Fetching reward fraction should not fail after setting")
	s.Require().True(fetchedReward.Equal(setReward), "The fetched reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value after setting")
}

func (s *KeeperTestSuite) TestSetPreviousInferenceRewardFraction() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	worker := s.AddrsStr(1)

	// Define a reward fraction to set
	rewardFraction := alloraMath.NewDecFromInt64(25)

	// Set the reward fraction
	err := k.SetPreviousInferenceRewardFraction(ctx, topicId, worker, rewardFraction)
	s.Require().NoError(err, "Setting previous inference reward fraction should not fail")

	// Verify by fetching the same
	fetchedReward, noPrior, err := k.GetPreviousInferenceRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching the set reward fraction should not fail")
	s.Require().True(fetchedReward.Equal(rewardFraction), "The fetched reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value when set")
}

func (s *KeeperTestSuite) TestGetPreviousInferenceRewardFraction() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	worker := s.AddrsStr(1)

	// Attempt to fetch a reward fraction before setting it
	defaultReward, noPrior, err := k.GetPreviousInferenceRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching reward fraction should not fail when not set")
	s.Require().True(defaultReward.IsZero(), "Should return zero reward fraction when not set")
	s.Require().True(noPrior, "Should return no prior value when not set")

	// Now set a specific reward fraction
	setReward := alloraMath.NewDecFromInt64(75)
	err = k.SetPreviousInferenceRewardFraction(ctx, topicId, worker, setReward)
	s.Require().NoError(err, "Setting previous inference reward fraction should not fail")
	// Fetch and verify the reward fraction after setting
	fetchedReward, noPrior, err := k.GetPreviousInferenceRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching reward fraction should not fail after setting")
	s.Require().True(fetchedReward.Equal(setReward), "The fetched reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value after setting")
}

func (s *KeeperTestSuite) TestSetPreviousForecastRewardFraction() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	worker := s.AddrsStr(3)

	// Define a reward fraction to set
	rewardFraction := alloraMath.NewDecFromInt64(50) // Assume setting the fraction to 0.50

	// Set the forecast reward fraction
	err := k.SetPreviousForecastRewardFraction(ctx, topicId, worker, rewardFraction)
	s.Require().NoError(err, "Setting previous forecast reward fraction should not fail")

	// Verify by fetching the set value
	fetchedReward, noPrior, err := k.GetPreviousForecastRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching the set forecast reward fraction should not fail")
	s.Require().True(fetchedReward.Equal(rewardFraction), "The fetched forecast reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value when set")
}

func (s *KeeperTestSuite) TestGetPreviousForecastRewardFraction() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	worker := s.AddrsStr(3)

	// Attempt to fetch the reward fraction before setting it, expecting default value
	defaultReward, noPrior, err := k.GetPreviousForecastRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching forecast reward fraction should not fail when not set")
	s.Require().True(defaultReward.IsZero(), "Should return zero forecast reward fraction when not set")
	s.Require().True(noPrior, "Should return no prior value when not set")

	// Now set a specific reward fraction
	setReward := alloraMath.NewDecFromInt64(75) // Assume setting it to 0.75
	err = k.SetPreviousForecastRewardFraction(ctx, topicId, worker, setReward)
	s.Require().NoError(err, "Setting previous forecast reward fraction should not fail")

	// Fetch and verify the reward fraction after setting
	fetchedReward, noPrior, err := k.GetPreviousForecastRewardFraction(ctx, topicId, worker)
	s.Require().NoError(err, "Fetching forecast reward fraction should not fail after setting")
	s.Require().True(fetchedReward.Equal(setReward), "The fetched forecast reward fraction should match the set value")
	s.Require().False(noPrior, "Should not return no prior value after setting")
}

func (s *KeeperTestSuite) TestSetGetPreviousPercentageRewardToStakedReputers() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	previousPercentageReward := alloraMath.NewDecFromInt64(50)

	// Set the previous percentage reward to staked reputers
	err := k.SetPreviousPercentageRewardToStakedReputers(ctx, previousPercentageReward)
	s.Require().NoError(err, "Setting previous percentage reward to staked reputers should not fail")

	// Get the previous percentage reward to staked reputers
	fetchedPercentageReward, err := k.GetPreviousPercentageRewardToStakedReputers(ctx)
	s.Require().NoError(err, "Fetching previous percentage reward to staked reputers should not fail")
	s.Require().Equal(previousPercentageReward, fetchedPercentageReward, "The fetched percentage reward should match the set value")
}

func (s *KeeperTestSuite) TestLowestScoreEmaFunctions() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	address := s.AddrsStr(2)

	lowestInfererScore := types.Score{
		TopicId:     topicId,
		BlockHeight: 100,
		Address:     address,
		Score:       alloraMath.NewDecFromInt64(50),
	}
	err := k.SetLowestInfererScoreEma(ctx, topicId, lowestInfererScore)
	s.Require().NoError(err)

	retrievedScore, found, err := k.GetLowestInfererScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestInfererScore, retrievedScore)

	lowestForecasterScore := types.Score{
		TopicId:     topicId,
		BlockHeight: 200,
		Address:     address,
		Score:       alloraMath.NewDecFromInt64(75),
	}
	err = k.SetLowestForecasterScoreEma(ctx, topicId, lowestForecasterScore)
	s.Require().NoError(err)

	retrievedScore, found, err = k.GetLowestForecasterScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestForecasterScore, retrievedScore)
}

func (s *KeeperTestSuite) TestLowestReputerScoreEmaFunctions() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := uint64(1)
	address := s.AddrsStr(4)

	lowestReputerScore := types.Score{
		TopicId:     topicId,
		BlockHeight: 300,
		Address:     address,
		Score:       alloraMath.NewDecFromInt64(60),
	}
	err := k.SetLowestReputerScoreEma(ctx, topicId, lowestReputerScore)
	s.Require().NoError(err)

	retrievedScore, found, err := k.GetLowestReputerScoreEma(ctx, topicId)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(lowestReputerScore, retrievedScore)
}

func (s *KeeperTestSuite) TestScoreLimiting() {
	k := s.ScoresKeeper()
	ctx := s.Ctx()
	topicId := s.CreateTopic()
	blockHeight := int64(10)

	params := types.DefaultParams()
	params.MaxTopInferersToReward = 2
	params.MaxSamplesToScaleScores = 3
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	for i := 0; i < 8; i++ {
		score := types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     s.AddrsStr(i),
			Score:       alloraMath.NewDecFromInt64(int64(90 + i)),
		}
		err := k.InsertWorkerInferenceScore(ctx, topicId, blockHeight, score)
		s.Require().NoError(err)
	}

	scores, err := k.GetWorkerInferenceScoresAtBlock(ctx, topicId, blockHeight)
	s.Require().NoError(err)
	s.Require().Len(scores.Scores, 6, "Should keep MaxSamplesToScaleScores * MaxTopInferersToReward scores")

	for i, score := range scores.Scores {
		expectedWorker := s.AddrsStr(i + 2)
		s.Require().Equal(expectedWorker, score.Address)
	}
}

func (s *KeeperTestSuite) TestInitialEmaScoreSettingInAppendInference() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := s.CreateTopic()
	worker := s.AddrsStr(0)
	blockHeight := int64(10)

	// Set initial EMA score for the topic
	initialScore := alloraMath.MustNewDecFromString("95.5")
	err := k.SetTopicInitialInfererEmaScore(ctx, topicId, initialScore)
	s.Require().NoError(err)

	// Create and append a new inference
	//nolint:exhaustruct
	inference := &types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.52")},
		Inferer:     worker,
	}

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Append the inference
	err = s.WorkerKeeper().AppendInference(ctx, topic, blockHeight, inference, 4)
	s.Require().NoError(err)

	// Verify the worker received the initial EMA score
	score, err := k.GetInfererScoreEma(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(initialScore, score.Score)
	s.Require().Equal(blockHeight, score.BlockHeight)
	s.Require().Equal(worker, score.Address)
	s.Require().Equal(topicId, score.TopicId)
}

func (s *KeeperTestSuite) TestInitialEmaScoreSettingInAppendForecast() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := s.CreateTopic()
	worker := s.AddrsStr(0)
	blockHeight := int64(10)

	// Set initial EMA score for the topic
	initialScore := alloraMath.MustNewDecFromString("92.5")
	err := k.SetTopicInitialForecasterEmaScore(ctx, topicId, initialScore)
	s.Require().NoError(err)

	// Create and append a new forecast
	forecast := &types.Forecast{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Forecaster:  worker,
		ForecastElements: []*types.ForecastElement{
			{
				Inferer: s.AddrsStr(1),
				Value:   alloraMath.MustNewDecFromString("0.52"),
			},
		},
		ExtraData: nil,
	}

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Append the forecast
	err = s.WorkerKeeper().AppendForecast(ctx, topic, blockHeight, forecast, 4)
	s.Require().NoError(err)

	// Verify the worker received the initial EMA score
	score, err := k.GetForecasterScoreEma(ctx, topicId, worker)
	s.Require().NoError(err)
	s.Require().Equal(initialScore, score.Score)
	s.Require().Equal(blockHeight, score.BlockHeight)
	s.Require().Equal(worker, score.Address)
	s.Require().Equal(topicId, score.TopicId)
}

func (s *KeeperTestSuite) TestInitialEmaScoreSettingInAppendReputer() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := s.CreateTopic()
	reputer := s.AddrsStr(0)
	blockHeight := int64(10)

	// Set initial EMA score for the topic
	initialScore := alloraMath.MustNewDecFromString("97.5")
	err := k.SetTopicInitialReputerEmaScore(ctx, topicId, initialScore)
	s.Require().NoError(err)

	// Create and append a new reputer value bundle
	//nolint:exhaustruct
	valueBundle := &types.ValueBundle{
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
		},
		Reputer:       reputer,
		CombinedValue: alloraMath.MustNewDecFromString("0.52"),
		InfererValues: s.createDefaultInfererValues(),
		NaiveValue:    alloraMath.MustNewDecFromString("0.52"),
	}
	signature := s.SignValueBundle(valueBundle, s.PrivKeys(0))
	reputerValueBundle := &types.ReputerValueBundle{
		ValueBundle: valueBundle,
		Signature:   signature,
		Pubkey:      s.PubKeyHexStr(0),
	}

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	params := types.DefaultParams()
	// Append the reputer value bundle
	err = s.ReputerLossKeeper().AppendReputerLoss(ctx, topic, params, blockHeight, reputerValueBundle)
	s.Require().NoError(err)

	// Verify the reputer received the initial EMA score
	score, err := k.GetReputerScoreEma(ctx, topicId, reputer)
	s.Require().NoError(err)
	s.Require().Equal(initialScore, score.Score)
	s.Require().Equal(blockHeight, score.BlockHeight)
	s.Require().Equal(reputer, score.Address)
	s.Require().Equal(topicId, score.TopicId)
}

func (s *KeeperTestSuite) TestFirstSubmissionDoesNotUpdateEMAUsingQuantile() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topicId := s.CreateTopic()

	params := types.DefaultParams()
	params.MaxTopInferersToReward = 4
	err := s.ParamsKeeper().SetParams(ctx, params)
	s.Require().NoError(err)

	// nolint: gosec
	for i := int64(0); i < int64(params.MaxTopInferersToReward); i++ {
		addr := s.AddrsStr(int(i))
		score := types.Score{
			TopicId:     topicId,
			Address:     addr,
			BlockHeight: 1,
			Score:       alloraMath.NewDecFromInt64(90 + i),
		}
		err := k.SetInfererScoreEma(ctx, topicId, addr, score)
		s.Require().NoError(err)

		err = s.WorkerKeeper().AddActiveInferer(ctx, topicId, addr)
		s.Require().NoError(err)

		if i == 0 {
			err = k.SetLowestInfererScoreEma(ctx, topicId, score)
			s.Require().NoError(err)
		}
	}

	// Set a low initial score for new actors
	initialScore := alloraMath.NewDecFromInt64(50)
	err = k.SetTopicInitialInfererEmaScore(ctx, topicId, initialScore)
	s.Require().NoError(err)

	// Create a new inference from a new actor
	inference := &types.Inference{
		TopicId:     topicId,
		BlockHeight: 2,
		Inferer:     s.AddrsStr(9), // Using a different address
		Values:      []alloraMath.Dec{alloraMath.NewDecFromInt64(100)},
		ExtraData:   nil,
		Proof:       "",
	}

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Submit inference - should not trigger EMA update using quantile since it's first submission
	// and score is lower than active set
	err = s.WorkerKeeper().AppendInference(ctx, topic, 2, inference, params.MaxTopInferersToReward)
	s.Require().NoError(err)

	// Verify score remains at initial value
	score, err := k.GetInfererScoreEma(ctx, topicId, s.AddrsStr(9))
	s.Require().NoError(err)
	s.Require().Equal(initialScore, score.Score)
}

func (s *KeeperTestSuite) TestLivenessPenaltyAppliedInAppendInference() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()

	epochLength := int64(1000)
	groundTruthLag := int64(1000)

	topicId := s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithGroundTruthLag(groundTruthLag),
		testutil.WithInitialRegret("0.5"),
		testutil.WithEpochLastEnded(10000),
	)
	worker := s.AddrsStr(0)
	blockHeight := int64(10000)

	// Set initial EMA score for the topic
	initialScore := alloraMath.MustNewDecFromString("50")
	s.Require().NoError(k.SetTopicInitialInfererEmaScore(ctx, topicId, initialScore))

	s.Require().NoError(k.SetInfererScoreEma(ctx, topicId, worker, types.Score{
		TopicId:     topicId,
		BlockHeight: 5000,
		Address:     worker,
		Score:       alloraMath.MustNewDecFromString("100"),
	}))

	// Create and append a new inference
	//nolint:exhaustruct
	inference := &types.Inference{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Values:      []alloraMath.Dec{alloraMath.MustNewDecFromString("0.52")},
		Inferer:     worker,
	}

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Append the inference
	err = s.WorkerKeeper().AppendInference(ctx, topic, blockHeight, inference, 4)
	s.Require().NoError(err)

	// Verify the worker's EMA score trended toward the topic initial score especially when there is a lapse in their
	// liveness
	score, err := k.GetInfererScoreEma(ctx, topicId, worker)
	s.Require().NoError(err)
	inDelta, err := alloraMath.InDelta(alloraMath.MustNewDecFromString("82.805"), score.Score, alloraMath.MustNewDecFromString("0.0001"))
	s.Require().NoError(err)
	s.Require().True(inDelta, "expected %s, got %s", "82.805", score.Score.String())
	s.Require().Equal(blockHeight, score.BlockHeight)
	s.Require().Equal(worker, score.Address)
	s.Require().Equal(topicId, score.TopicId)
}

func (s *KeeperTestSuite) TestLivenessPenaltyAppliedInAppendForecast() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()

	epochLength := int64(1000)
	groundTruthLag := int64(1000)

	topicId := s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithGroundTruthLag(groundTruthLag),
		testutil.WithEpochLastEnded(10000),
	)
	worker := s.AddrsStr(0)
	blockHeight := int64(10000)

	// Set initial EMA score for the topic
	initialScore := alloraMath.MustNewDecFromString("50")
	s.Require().NoError(k.SetTopicInitialForecasterEmaScore(ctx, topicId, initialScore))

	s.Require().NoError(k.SetForecasterScoreEma(ctx, topicId, worker, types.Score{
		TopicId:     topicId,
		BlockHeight: 5000,
		Address:     worker,
		Score:       alloraMath.MustNewDecFromString("100"),
	}))

	// Create and append a new forecast
	forecast := &types.Forecast{
		TopicId:     topicId,
		BlockHeight: 10000,
		Forecaster:  worker,
		ForecastElements: []*types.ForecastElement{
			{
				Inferer: worker,
				Value:   alloraMath.MustNewDecFromString("0.52"),
			},
		},
		ExtraData: nil,
	}

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Append the forecast
	err = s.WorkerKeeper().AppendForecast(ctx, topic, blockHeight, forecast, 4)
	s.Require().NoError(err)
	s.Require().NoError(err)

	// Verify the worker's EMA score trended toward the topic initial score especially when there is a lapse in their
	// liveness
	score, err := k.GetForecasterScoreEma(ctx, topicId, worker)
	s.Require().NoError(err)
	inDelta, err := alloraMath.InDelta(alloraMath.MustNewDecFromString("82.805"), score.Score, alloraMath.MustNewDecFromString("0.0001"))
	s.Require().NoError(err)
	s.Require().True(inDelta, "expected %s, got %s", "82.805", score.Score.String())
	s.Require().Equal(blockHeight, score.BlockHeight)
	s.Require().Equal(worker, score.Address)
	s.Require().Equal(topicId, score.TopicId)
}

func (s *KeeperTestSuite) TestLivenessPenaltyAppliedInAppendReputerLoss() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()

	epochLength := int64(1000)
	groundTruthLag := int64(1000)

	topicId := s.CreateTopic(
		testutil.WithEpochLength(epochLength),
		testutil.WithGroundTruthLag(groundTruthLag),
		testutil.WithEpochLastEnded(10000),
	)
	reputer := s.AddrsStr(0)
	blockHeight := int64(10000)
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
	}

	// Set initial EMA score for the topic
	initialScore := alloraMath.MustNewDecFromString("50")
	s.Require().NoError(k.SetTopicInitialReputerEmaScore(ctx, topicId, initialScore))

	s.Require().NoError(k.SetReputerScoreEma(ctx, topicId, reputer, types.Score{
		TopicId:     topicId,
		BlockHeight: 5000,
		Address:     reputer,
		Score:       alloraMath.MustNewDecFromString("100"),
	}))

	// Create and append a new reputer loss
	//nolint:exhaustruct
	valueBundleReputer := types.ValueBundle{
		Reputer:             reputer,
		CombinedValue:       alloraMath.MustNewDecFromString(".0000256948644008351"),
		ReputerRequestNonce: reputerRequestNonce,
		TopicId:             topicId,
		InfererValues:       s.createDefaultInfererValues(),
		NaiveValue:          alloraMath.MustNewDecFromString("0.0"),
	}
	signature := s.SignValueBundle(&valueBundleReputer, s.PrivKeys(0))
	reputerValueBundle := types.ReputerValueBundle{
		ValueBundle: &valueBundleReputer,
		Signature:   signature,
		Pubkey:      s.PubKeyHexStr(0),
	}

	topic, err := s.TopicKeeper().GetTopic(ctx, topicId)
	s.Require().NoError(err)

	// Append the reputer loss
	err = s.ReputerLossKeeper().AppendReputerLoss(ctx, topic, types.DefaultParams(), blockHeight, &reputerValueBundle)
	s.Require().NoError(err)

	// Verify the reputer's EMA score trended toward the topic initial score especially when there is a lapse in their
	// liveness
	score, err := k.GetReputerScoreEma(ctx, topicId, reputer)
	s.Require().NoError(err)
	inDelta, err := alloraMath.InDelta(alloraMath.MustNewDecFromString("86.450"), score.Score, alloraMath.MustNewDecFromString("0.0001"))
	s.Require().NoError(err)
	s.Require().True(inDelta, "expected %s, got %s", "86.450", score.Score.String())
	s.Require().Equal(blockHeight, score.BlockHeight)
	s.Require().Equal(reputer, score.Address)
	s.Require().Equal(topicId, score.TopicId)
}
