package keeper_test

import (
	"log"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestCalcAndSaveInfererScoreEmaIfNewUpdate() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	topic := types.Topic{
		Id:                     uint64(1),
		WorkerSubmissionWindow: 10,
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.2"),
	}
	worker := "worker1"
	block := types.BlockHeight(100)

	// Test case 1: New update
	newScore := types.Score{
		TopicId:     topic.Id,
		BlockHeight: block,
		Address:     worker,
		Score:       alloraMath.MustNewDecFromString("0.2"),
	}
	err := keeper.CalcAndSaveInfererScoreEmaIfNewUpdate(ctx, topic, block, worker, newScore)
	s.Require().NoError(err)

	// Verify the EMA score was saved
	savedScore, err := keeper.GetInfererScoreEma(ctx, topic.Id, worker)
	s.Require().NoError(err)
	s.Require().Equal(newScore.Score, savedScore.Score)

	// Test case 2: No update (within submission window)
	newScore.BlockHeight = block + 5
	err = keeper.CalcAndSaveInfererScoreEmaIfNewUpdate(ctx, topic, newScore.BlockHeight, worker, newScore)
	s.Require().NoError(err)

	// Verify the EMA score was not updated
	savedScore, err = keeper.GetInfererScoreEma(ctx, topic.Id, worker)
	s.Require().NoError(err)
	s.Require().Equal(block, savedScore.BlockHeight)

	log.Println("savedScore", savedScore)
}

func (s *KeeperTestSuite) TestCalcAndSaveForecasterScoreEmaIfNewUpdate() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	topic := types.Topic{
		Id:                     uint64(1),
		WorkerSubmissionWindow: 10,
		MeritSortitionAlpha:    alloraMath.MustNewDecFromString("0.2"),
	}
	worker := "worker1"
	block := types.BlockHeight(100)

	// Test case 1: New update
	newScore := types.Score{
		TopicId:     topic.Id,
		BlockHeight: block,
		Address:     worker,
		Score:       alloraMath.MustNewDecFromString("0.5"),
	}
	err := keeper.CalcAndSaveForecasterScoreEmaIfNewUpdate(ctx, topic, block, worker, newScore)
	s.Require().NoError(err)

	// Verify the EMA score was saved
	savedScore, err := keeper.GetForecasterScoreEma(ctx, topic.Id, worker)
	s.Require().NoError(err)
	s.Require().Equal(newScore.Score, savedScore.Score)

	// Test case 2: No update (within submission window)
	newScore.BlockHeight = block + 5
	err = keeper.CalcAndSaveForecasterScoreEmaIfNewUpdate(ctx, topic, newScore.BlockHeight, worker, newScore)
	s.Require().NoError(err)

	// Verify the EMA score was not updated
	savedScore, err = keeper.GetForecasterScoreEma(ctx, topic.Id, worker)
	s.Require().NoError(err)
	s.Require().Equal(block, savedScore.BlockHeight)
}

func (s *KeeperTestSuite) TestCalcAndSaveReputerScoreEmaIfNewUpdate() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	topic := types.Topic{
		Id:                  uint64(1),
		EpochLength:         20,
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.2"),
	}
	reputer := "reputer1"
	block := types.BlockHeight(100)

	// Test case 1: New update
	newScore := types.Score{
		TopicId:     topic.Id,
		BlockHeight: block,
		Address:     reputer,
		Score:       alloraMath.MustNewDecFromString("0.5"),
	}
	err := keeper.CalcAndSaveReputerScoreEmaIfNewUpdate(ctx, topic, block, reputer, newScore)
	s.Require().NoError(err)

	// Verify the EMA score was saved
	savedScore, err := keeper.GetReputerScoreEma(ctx, topic.Id, reputer)
	s.Require().NoError(err)
	s.Require().Equal(newScore.Score, savedScore.Score)

	// Test case 2: No update (within epoch length)
	newScore.BlockHeight = block + 10
	err = keeper.CalcAndSaveReputerScoreEmaIfNewUpdate(ctx, topic, newScore.BlockHeight, reputer, newScore)
	s.Require().NoError(err)

	// Verify the EMA score was not updated
	savedScore, err = keeper.GetReputerScoreEma(ctx, topic.Id, reputer)
	s.Require().NoError(err)
	s.Require().Equal(block, savedScore.BlockHeight)
}

func (s *KeeperTestSuite) TestCalcAndSaveInfererScoreEmaWithLastSavedTopicQuantile() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	topic := types.Topic{
		Id:                  uint64(1),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.2"),
	}
	worker := "worker1"
	block := types.BlockHeight(100)

	// Set up a previous topic quantile score
	previousQuantileScore := alloraMath.MustNewDecFromString("0.8")
	err := keeper.SetPreviousTopicQuantileInfererScoreEma(ctx, topic.Id, previousQuantileScore)
	s.Require().NoError(err)

	err = keeper.CalcAndSaveInfererScoreEmaWithLastSavedTopicQuantile(ctx, topic, block, worker)
	s.Require().NoError(err)

	// Verify the EMA score was calculated and saved
	savedScore, err := keeper.GetInfererScoreEma(ctx, topic.Id, worker)
	s.Require().NoError(err)
	s.Require().Equal(previousQuantileScore, savedScore.Score)
	s.Require().Equal(block, savedScore.BlockHeight)
}

func (s *KeeperTestSuite) TestCalcAndSaveForecasterScoreEmaWithLastSavedTopicQuantile() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	topic := types.Topic{
		Id:                  uint64(1),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.2"),
	}
	worker := "worker1"
	block := types.BlockHeight(100)

	// Set up a previous topic quantile score
	previousQuantileScore := alloraMath.MustNewDecFromString("0.8")
	err := keeper.SetPreviousTopicQuantileForecasterScoreEma(ctx, topic.Id, previousQuantileScore)
	s.Require().NoError(err)

	err = keeper.CalcAndSaveForecasterScoreEmaWithLastSavedTopicQuantile(ctx, topic, block, worker)
	s.Require().NoError(err)

	// Verify the EMA score was calculated and saved
	savedScore, err := keeper.GetForecasterScoreEma(ctx, topic.Id, worker)
	s.Require().NoError(err)
	s.Require().Equal(previousQuantileScore, savedScore.Score)
	s.Require().Equal(block, savedScore.BlockHeight)
}

func (s *KeeperTestSuite) TestCalcAndSaveReputerScoreEmaWithLastSavedTopicQuantile() {
	ctx := s.ctx
	keeper := s.emissionsKeeper

	topic := types.Topic{
		Id:                  uint64(1),
		MeritSortitionAlpha: alloraMath.MustNewDecFromString("0.2"),
	}
	reputer := "reputer1"
	block := types.BlockHeight(100)

	// Set up a previous topic quantile score
	previousQuantileScore := alloraMath.MustNewDecFromString("0.8")
	err := keeper.SetPreviousTopicQuantileReputerScoreEma(ctx, topic.Id, previousQuantileScore)
	s.Require().NoError(err)

	err = keeper.CalcAndSaveReputerScoreEmaWithLastSavedTopicQuantile(ctx, topic, block, reputer)
	s.Require().NoError(err)

	// Verify the EMA score was calculated and saved
	savedScore, err := keeper.GetReputerScoreEma(ctx, topic.Id, reputer)
	s.Require().NoError(err)
	s.Require().Equal(previousQuantileScore, savedScore.Score)
	s.Require().Equal(block, savedScore.BlockHeight)
}
