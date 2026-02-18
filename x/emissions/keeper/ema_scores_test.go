package keeper_test

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestCalcAndSaveScoreEmaForActiveSet() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topic := s.MockTopic()

	testCases := []struct {
		name              string
		actorType         string
		actorIndex        int
		initialScore      string
		updateBlockOffset types.BlockHeight
		calcAndSaveFunc   func(context.Context, types.Topic, string, types.Score) (types.Score, error)
		getFunc           func(context.Context, uint64, string) (types.Score, error)
	}{
		{
			name:              "inferer new update",
			actorType:         "inferer",
			actorIndex:        1,
			initialScore:      "0.2",
			updateBlockOffset: 5,
			calcAndSaveFunc:   k.CalcAndSaveInfererScoreEmaForActiveSet,
			getFunc:           k.GetInfererScoreEma,
		},
		{
			name:              "forecaster new update",
			actorType:         "forecaster",
			actorIndex:        1,
			initialScore:      "0.5",
			updateBlockOffset: 5,
			calcAndSaveFunc:   k.CalcAndSaveForecasterScoreEmaForActiveSet,
			getFunc:           k.GetForecasterScoreEma,
		},
		{
			name:              "reputer new update",
			actorType:         "reputer",
			actorIndex:        2,
			initialScore:      "0.5",
			updateBlockOffset: 10,
			calcAndSaveFunc:   k.CalcAndSaveReputerScoreEmaForActiveSet,
			getFunc:           k.GetReputerScoreEma,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			block := types.BlockHeight(100)
			actor := s.AddrsStr(tc.actorIndex)

			// Test case 1: New update
			newScore := types.Score{
				TopicId:     topic.Id,
				BlockHeight: block,
				Address:     actor,
				Score:       alloraMath.MustNewDecFromString(tc.initialScore),
			}

			emaScore, err := tc.calcAndSaveFunc(ctx, topic, actor, newScore)
			s.Require().NoError(err)
			s.Require().Equal(tc.initialScore, emaScore.Score.String())

			// Verify the EMA score was saved
			savedScore, err := tc.getFunc(ctx, topic.Id, actor)
			s.Require().NoError(err)
			s.Require().Equal(newScore.Score, savedScore.Score)

			// Test case 2: Don't update blockheight of score
			newScore.BlockHeight = block + tc.updateBlockOffset
			emaScore, err = tc.calcAndSaveFunc(ctx, topic, actor, newScore)
			s.Require().NoError(err)
			s.Require().Equal(tc.initialScore, emaScore.Score.String())

			// Verify the blockheight of the EMA score was not updated
			savedScoreAgain, err := tc.getFunc(ctx, topic.Id, actor)
			s.Require().NoError(err)
			s.Require().Equal(savedScore.BlockHeight, savedScoreAgain.BlockHeight)
		})
	}
}

func (s *KeeperTestSuite) TestCalcAndSaveScoreEmaWithLastSavedTopicQuantile() {
	ctx := s.Ctx()
	k := s.ScoresKeeper()
	topic := s.MockTopic()
	previousQuantileScore := alloraMath.MustNewDecFromString("0.8")

	testCases := []struct {
		name            string
		actorType       string
		actorIndex      int
		setupQuantile   func(context.Context, uint64, alloraMath.Dec) error
		calcAndSaveFunc func(sdk.Context, types.Topic, int64, types.Score) error
		getFunc         func(context.Context, uint64, string) (types.Score, error)
	}{
		{
			name:            "inferer with topic quantile",
			actorType:       "inferer",
			actorIndex:      1,
			setupQuantile:   k.SetPreviousTopicQuantileInfererScoreEma,
			calcAndSaveFunc: k.CalcAndSaveInfererScoreEmaWithLastSavedTopicQuantile,
			getFunc:         k.GetInfererScoreEma,
		},
		{
			name:            "forecaster with topic quantile",
			actorType:       "forecaster",
			actorIndex:      1,
			setupQuantile:   k.SetPreviousTopicQuantileForecasterScoreEma,
			calcAndSaveFunc: k.CalcAndSaveForecasterScoreEmaWithLastSavedTopicQuantile,
			getFunc:         k.GetForecasterScoreEma,
		},
		{
			name:            "reputer with topic quantile",
			actorType:       "reputer",
			actorIndex:      2,
			setupQuantile:   k.SetPreviousTopicQuantileReputerScoreEma,
			calcAndSaveFunc: k.CalcAndSaveReputerScoreEmaWithLastSavedTopicQuantile,
			getFunc:         k.GetReputerScoreEma,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			actor := s.AddrsStr(tc.actorIndex)
			block := types.BlockHeight(100)

			// Set up a previous topic quantile score
			err := tc.setupQuantile(ctx, topic.Id, previousQuantileScore)
			s.Require().NoError(err)

			score := types.Score{
				TopicId:     topic.Id,
				BlockHeight: block,
				Address:     actor,
				Score:       previousQuantileScore,
			}

			err = tc.calcAndSaveFunc(ctx, topic, block, score)
			s.Require().NoError(err)

			// Verify the EMA score was calculated and saved
			savedScore, err := tc.getFunc(ctx, topic.Id, actor)
			s.Require().NoError(err)
			s.Require().Equal(previousQuantileScore, savedScore.Score)
			s.Require().Equal(block, savedScore.BlockHeight)
		})
	}
}
