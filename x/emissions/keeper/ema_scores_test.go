package keeper_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestCalcAndSaveInfererScoreEmaIfNewUpdate() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	topic := types.Topic{
		Id:                       uint64(1),
		WorkerSubmissionWindow:   10,
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.2"),
		Creator:                  s.AddrsStr()[0],
		Metadata:                 "",
		LossMethod:               "",
		EpochLastEnded:           0,
		EpochLength:              10,
		GroundTruthLag:           10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.ZeroDec(),
		AllowNegative:            false,
		Epsilon:                  alloraMath.ZeroDec(),
		InitialRegret:            alloraMath.ZeroDec(),
		ActiveInfererQuantile:    alloraMath.ZeroDec(),
		ActiveForecasterQuantile: alloraMath.ZeroDec(),
		ActiveReputerQuantile:    alloraMath.ZeroDec(),
	}
	worker := s.AddrsStr()[1]
	block := types.BlockHeight(100)

	// Test case 1: New update
	newScore := types.Score{
		TopicId:     topic.Id,
		BlockHeight: block,
		Address:     worker,
		Score:       alloraMath.MustNewDecFromString("0.2"),
	}
	emaScore, err := k.CalcAndSaveInfererScoreEmaForActiveSet(ctx, topic, worker, newScore)
	s.Require().NoError(err)
	s.Require().Equal("0.2", emaScore.Score.String())

	// Verify the EMA score was saved
	savedScore, err := k.GetInfererScoreEma(ctx, topic.Id, worker)
	s.Require().NoError(err)
	s.Require().Equal(newScore.Score, savedScore.Score)

	// Test case 2: Don't update blockheight of score
	newScore.BlockHeight = block + 5
	emaScore, err = k.CalcAndSaveInfererScoreEmaForActiveSet(ctx, topic, worker, newScore)
	s.Require().NoError(err)
	s.Require().Equal("0.2", emaScore.Score.String())

	// Verify the blockheight of the EMA score was not updated
	savedScoreAgain, err := k.GetInfererScoreEma(ctx, topic.Id, worker)
	s.Require().NoError(err)
	s.Require().Equal(savedScore.BlockHeight, savedScoreAgain.BlockHeight)
}

func (s *KeeperTestSuite) TestCalcAndSaveForecasterScoreEmaIfNewUpdate() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	topic := types.Topic{
		Id:                       uint64(1),
		WorkerSubmissionWindow:   10,
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.2"),
		Creator:                  s.AddrsStr()[0],
		Metadata:                 "",
		LossMethod:               "",
		EpochLastEnded:           0,
		EpochLength:              10,
		GroundTruthLag:           10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.ZeroDec(),
		AllowNegative:            false,
		Epsilon:                  alloraMath.ZeroDec(),
		InitialRegret:            alloraMath.ZeroDec(),
		ActiveInfererQuantile:    alloraMath.ZeroDec(),
		ActiveForecasterQuantile: alloraMath.ZeroDec(),
		ActiveReputerQuantile:    alloraMath.ZeroDec(),
	}
	worker := s.AddrsStr()[1]
	block := types.BlockHeight(100)

	// Test case 1: New update
	newScore := types.Score{
		TopicId:     topic.Id,
		BlockHeight: block,
		Address:     worker,
		Score:       alloraMath.MustNewDecFromString("0.5"),
	}
	emaScore, err := k.CalcAndSaveForecasterScoreEmaForActiveSet(ctx, topic, worker, newScore)
	s.Require().NoError(err)
	s.Require().Equal("0.5", emaScore.Score.String())

	// Verify the EMA score was saved
	savedScore, err := k.GetForecasterScoreEma(ctx, topic.Id, worker)
	s.Require().NoError(err)
	s.Require().Equal(newScore.Score, savedScore.Score)

	// Test case 2: Not update blockheight of score
	newScore.BlockHeight = block + 5
	emaScore, err = k.CalcAndSaveForecasterScoreEmaForActiveSet(ctx, topic, worker, newScore)
	s.Require().NoError(err)
	s.Require().Equal("0.5", emaScore.Score.String())

	// Verify the blockheight of the EMA score was not updated
	savedScoreAgain, err := k.GetForecasterScoreEma(ctx, topic.Id, worker)
	s.Require().NoError(err)
	s.Require().Equal(savedScore.BlockHeight, savedScoreAgain.BlockHeight)
}

func (s *KeeperTestSuite) TestCalcAndSaveReputerScoreEmaIfNewUpdate() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	topic := types.Topic{
		Creator:                  s.AddrsStr()[0],
		Id:                       uint64(1),
		EpochLength:              20,
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.2"),
		Metadata:                 "",
		LossMethod:               "",
		EpochLastEnded:           0,
		GroundTruthLag:           20,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.ZeroDec(),
		AllowNegative:            false,
		Epsilon:                  alloraMath.ZeroDec(),
		InitialRegret:            alloraMath.ZeroDec(),
		WorkerSubmissionWindow:   20,
		ActiveInfererQuantile:    alloraMath.ZeroDec(),
		ActiveForecasterQuantile: alloraMath.ZeroDec(),
		ActiveReputerQuantile:    alloraMath.ZeroDec(),
	}
	reputer := s.AddrsStr()[2]
	block := types.BlockHeight(100)

	// Test case 1: New update
	newScore := types.Score{
		TopicId:     topic.Id,
		BlockHeight: block,
		Address:     reputer,
		Score:       alloraMath.MustNewDecFromString("0.5"),
	}
	emaScore, err := k.CalcAndSaveReputerScoreEmaForActiveSet(ctx, topic, reputer, newScore)
	s.Require().NoError(err)
	s.Require().Equal("0.5", emaScore.Score.String())

	// Verify the EMA score was saved
	savedScore, err := k.GetReputerScoreEma(ctx, topic.Id, reputer)
	s.Require().NoError(err)
	s.Require().Equal(newScore.Score, savedScore.Score)

	// Test case 2: Don't update blockheight of score
	newScore.BlockHeight = block + 10
	emaScore, err = k.CalcAndSaveReputerScoreEmaForActiveSet(ctx, topic, reputer, newScore)
	s.Require().NoError(err)
	s.Require().Equal("0.5", emaScore.Score.String())

	// Verify the blockheight of the EMA score was not updated
	savedScoreAgain, err := k.GetReputerScoreEma(ctx, topic.Id, reputer)
	s.Require().NoError(err)
	s.Require().Equal(savedScore.BlockHeight, savedScoreAgain.BlockHeight)
}

func (s *KeeperTestSuite) TestCalcAndSaveInfererScoreEmaWithLastSavedTopicQuantile() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	topic := types.Topic{
		Id:                       uint64(1),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.2"),
		Creator:                  s.AddrsStr()[0],
		Metadata:                 "",
		LossMethod:               "",
		EpochLastEnded:           0,
		EpochLength:              100,
		GroundTruthLag:           100,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.ZeroDec(),
		AllowNegative:            false,
		Epsilon:                  alloraMath.ZeroDec(),
		InitialRegret:            alloraMath.ZeroDec(),
		WorkerSubmissionWindow:   100,
		ActiveInfererQuantile:    alloraMath.ZeroDec(),
		ActiveForecasterQuantile: alloraMath.ZeroDec(),
		ActiveReputerQuantile:    alloraMath.ZeroDec(),
	}
	worker := s.AddrsStr()[1]
	block := types.BlockHeight(100)

	// Set up a previous topic quantile score
	previousQuantileScore := alloraMath.MustNewDecFromString("0.8")
	err := k.SetPreviousTopicQuantileInfererScoreEma(ctx, topic.Id, previousQuantileScore)
	s.Require().NoError(err)

	score := types.Score{
		TopicId:     topic.Id,
		BlockHeight: block,
		Address:     worker,
		Score:       previousQuantileScore,
	}
	err = k.CalcAndSaveInfererScoreEmaWithLastSavedTopicQuantile(ctx, topic, block, score)
	s.Require().NoError(err)

	// Verify the EMA score was calculated and saved
	savedScore, err := k.GetInfererScoreEma(ctx, topic.Id, worker)
	s.Require().NoError(err)
	s.Require().Equal(previousQuantileScore, savedScore.Score)
	s.Require().Equal(block, savedScore.BlockHeight)
}

func (s *KeeperTestSuite) TestCalcAndSaveForecasterScoreEmaWithLastSavedTopicQuantile() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	topic := types.Topic{
		Id:                       uint64(1),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.2"),
		Creator:                  s.AddrsStr()[0],
		Metadata:                 "",
		LossMethod:               "",
		EpochLastEnded:           0,
		EpochLength:              100,
		GroundTruthLag:           100,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.ZeroDec(),
		AllowNegative:            false,
		Epsilon:                  alloraMath.ZeroDec(),
		InitialRegret:            alloraMath.ZeroDec(),
		WorkerSubmissionWindow:   100,
		ActiveInfererQuantile:    alloraMath.ZeroDec(),
		ActiveForecasterQuantile: alloraMath.ZeroDec(),
		ActiveReputerQuantile:    alloraMath.ZeroDec(),
	}
	worker := s.AddrsStr()[1]
	block := types.BlockHeight(100)

	// Set up a previous topic quantile score
	previousQuantileScore := alloraMath.MustNewDecFromString("0.8")
	err := k.SetPreviousTopicQuantileForecasterScoreEma(ctx, topic.Id, previousQuantileScore)
	s.Require().NoError(err)

	score := types.Score{
		TopicId:     topic.Id,
		BlockHeight: block,
		Address:     worker,
		Score:       previousQuantileScore,
	}
	err = k.CalcAndSaveForecasterScoreEmaWithLastSavedTopicQuantile(ctx, topic, block, score)
	s.Require().NoError(err)

	// Verify the EMA score was calculated and saved
	savedScore, err := k.GetForecasterScoreEma(ctx, topic.Id, worker)
	s.Require().NoError(err)
	s.Require().Equal(previousQuantileScore, savedScore.Score)
	s.Require().Equal(block, savedScore.BlockHeight)
}

func (s *KeeperTestSuite) TestCalcAndSaveReputerScoreEmaWithLastSavedTopicQuantile() {
	ctx := s.Ctx()
	k := s.EmissionsKeeper()

	topic := types.Topic{
		Id:                       uint64(1),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.2"),
		Creator:                  s.AddrsStr()[0],
		Metadata:                 "",
		LossMethod:               "",
		EpochLastEnded:           0,
		EpochLength:              100,
		GroundTruthLag:           100,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.ZeroDec(),
		AllowNegative:            false,
		Epsilon:                  alloraMath.ZeroDec(),
		InitialRegret:            alloraMath.ZeroDec(),
		WorkerSubmissionWindow:   100,
		ActiveInfererQuantile:    alloraMath.ZeroDec(),
		ActiveForecasterQuantile: alloraMath.ZeroDec(),
		ActiveReputerQuantile:    alloraMath.ZeroDec(),
	}
	reputer := s.AddrsStr()[2]
	block := types.BlockHeight(100)

	// Set up a previous topic quantile score
	previousQuantileScore := alloraMath.MustNewDecFromString("0.8")
	err := k.SetPreviousTopicQuantileReputerScoreEma(ctx, topic.Id, previousQuantileScore)
	s.Require().NoError(err)

	score := types.Score{
		TopicId:     topic.Id,
		BlockHeight: block,
		Address:     reputer,
		Score:       previousQuantileScore,
	}
	err = k.CalcAndSaveReputerScoreEmaWithLastSavedTopicQuantile(ctx, topic, block, score)
	s.Require().NoError(err)

	// Verify the EMA score was calculated and saved
	savedScore, err := k.GetReputerScoreEma(ctx, topic.Id, reputer)
	s.Require().NoError(err)
	s.Require().Equal(previousQuantileScore, savedScore.Score)
	s.Require().Equal(block, savedScore.BlockHeight)
}
