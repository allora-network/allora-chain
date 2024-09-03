package keeper_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	keeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *KeeperTestSuite) TestGetLowScoreFromAllInferences() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := uint64(1)
	blockHeightInferences := int64(10)

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"

	score1 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker3, Score: alloraMath.NewDecFromInt64(99)}
	_ = k.SetInfererScoreEma(ctx, topicId, worker1, score1)
	_ = k.SetInfererScoreEma(ctx, topicId, worker2, score2)
	_ = k.SetInfererScoreEma(ctx, topicId, worker3, score3)

	allInferences := types.Inferences{
		Inferences: []*types.Inference{
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.52")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.71")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.71")},
		},
	}
	lowScore, lowScoreIndex, err := keeper.GetLowScoreFromAllInferences(ctx, &k, topicId, allInferences)
	s.Require().NoError(err)
	s.Require().Equal(lowScore, score2)
	s.Require().Equal(lowScoreIndex, 1)
}

func (s *KeeperTestSuite) TestGetLowScoreFromAllForecasts() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := uint64(1)
	blockHeightInferences := int64(10)

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"

	score1 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker3, Score: alloraMath.NewDecFromInt64(99)}
	_ = k.SetForecasterScoreEma(ctx, topicId, worker1, score1)
	_ = k.SetForecasterScoreEma(ctx, topicId, worker2, score2)
	_ = k.SetForecasterScoreEma(ctx, topicId, worker3, score3)

	allForecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
				Forecaster:  worker1,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: worker1,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
					{
						Inferer: worker2,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
				Forecaster:  worker2,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: worker1,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
					{
						Inferer: worker2,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightInferences,
				Forecaster:  worker3,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: worker1,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
					{
						Inferer: worker2,
						Value:   alloraMath.MustNewDecFromString("0.52"),
					},
				},
			},
		},
	}
	lowScore, lowScoreIndex, err := keeper.GetLowScoreFromAllForecasts(ctx, &k, topicId, allForecasts)
	s.Require().NoError(err)
	s.Require().Equal(lowScore, score2)
	s.Require().Equal(lowScoreIndex, 1)
}

func (s *KeeperTestSuite) TestGetLowScoreFromAllLossBundles() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := uint64(1)
	blockHeight := int64(10)
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
	}

	reputer1 := "reputer1"
	reputer2 := "reputer2"
	reputer3 := "reputer3"

	score1 := types.Score{TopicId: topicId, BlockHeight: 2, Address: reputer1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: 2, Address: reputer2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: 2, Address: reputer3, Score: alloraMath.NewDecFromInt64(99)}
	_ = k.SetReputerScoreEma(ctx, topicId, reputer1, score1)
	_ = k.SetReputerScoreEma(ctx, topicId, reputer2, score2)
	_ = k.SetReputerScoreEma(ctx, topicId, reputer3, score3)

	allReputerLosses := types.ReputerValueBundles{
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				ValueBundle: &types.ValueBundle{
					Reputer:             reputer1,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000117005278862668"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &types.ValueBundle{
					Reputer:             reputer2,
					CombinedValue:       alloraMath.MustNewDecFromString(".00000962701954026944"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &types.ValueBundle{
					Reputer:             reputer3,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000256948644008351"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
		},
	}
	lowScore, lowScoreIndex, err := keeper.GetLowScoreFromAllLossBundles(ctx, &k, topicId, allReputerLosses)
	s.Require().NoError(err)
	s.Require().Equal(lowScore, score2)
	s.Require().Equal(lowScoreIndex, 1)
}
