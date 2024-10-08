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

	worker1 := s.addrsStr[0]
	worker2 := s.addrsStr[1]
	worker3 := s.addrsStr[2]
	workerAddresses := []string{worker1, worker2, worker3}

	score1 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker3, Score: alloraMath.NewDecFromInt64(99)}
	_ = k.SetInfererScoreEma(ctx, topicId, worker1, score1)
	_ = k.SetInfererScoreEma(ctx, topicId, worker2, score2)
	_ = k.SetInfererScoreEma(ctx, topicId, worker3, score3)

	lowScore, err := keeper.GetLowestScoreFromAllInferers(ctx, &k, topicId, workerAddresses)
	s.Require().NoError(err)
	s.Require().Equal(lowScore, score2)
}

func (s *KeeperTestSuite) TestGetLowScoreFromAllForecasts() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := uint64(1)

	worker1 := s.addrsStr[0]
	worker2 := s.addrsStr[1]
	worker3 := s.addrsStr[2]
	forecasterAddresses := []string{worker1, worker2, worker3}

	score1 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker1, Score: alloraMath.NewDecFromInt64(95)}
	score2 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker2, Score: alloraMath.NewDecFromInt64(90)}
	score3 := types.Score{TopicId: topicId, BlockHeight: 2, Address: worker3, Score: alloraMath.NewDecFromInt64(99)}
	_ = k.SetForecasterScoreEma(ctx, topicId, worker1, score1)
	_ = k.SetForecasterScoreEma(ctx, topicId, worker2, score2)
	_ = k.SetForecasterScoreEma(ctx, topicId, worker3, score3)

	lowScore, err := keeper.GetLowestScoreFromAllForecasters(ctx, &k, topicId, forecasterAddresses)
	s.Require().NoError(err)
	s.Require().Equal(lowScore, score2)
}

func (s *KeeperTestSuite) TestGetLowScoreFromAllLossBundles() {
	ctx := s.ctx
	k := s.emissionsKeeper
	topicId := uint64(1)

	reputer1 := s.addrsStr[0]
	reputer2 := s.addrsStr[1]
	reputer3 := s.addrsStr[2]
	reputerAddresses := []string{reputer1, reputer2, reputer3}

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
					Reputer:                       reputer1,
					CombinedValue:                 alloraMath.MustNewDecFromString(".0000117005278862668"),
					ReputerRequestNonce:           reputerRequestNonce,
					TopicId:                       topicId,
					ExtraData:                     nil,
					InfererValues:                 nil,
					ForecasterValues:              nil,
					NaiveValue:                    alloraMath.MustNewDecFromString("0.0"),
					OneOutInfererValues:           nil,
					OneOutForecasterValues:        nil,
					OneInForecasterValues:         nil,
					OneOutInfererForecasterValues: nil,
				},
			},
			{
				ValueBundle: &types.ValueBundle{
					Reputer:                       reputer2,
					CombinedValue:                 alloraMath.MustNewDecFromString(".00000962701954026944"),
					ReputerRequestNonce:           reputerRequestNonce,
					TopicId:                       topicId,
					ExtraData:                     nil,
					InfererValues:                 nil,
					ForecasterValues:              nil,
					NaiveValue:                    alloraMath.MustNewDecFromString("0.0"),
					OneOutInfererValues:           nil,
					OneOutForecasterValues:        nil,
					OneInForecasterValues:         nil,
					OneOutInfererForecasterValues: nil,
				},
			},
			{
				ValueBundle: &types.ValueBundle{
					Reputer:                       reputer3,
					CombinedValue:                 alloraMath.MustNewDecFromString(".0000256948644008351"),
					ReputerRequestNonce:           reputerRequestNonce,
					TopicId:                       topicId,
					ExtraData:                     nil,
					InfererValues:                 nil,
					ForecasterValues:              nil,
					NaiveValue:                    alloraMath.MustNewDecFromString("0.0"),
					OneOutInfererValues:           nil,
					OneOutForecasterValues:        nil,
					OneInForecasterValues:         nil,
					OneOutInfererForecasterValues: nil,
				},
			},
		},
	}
	lowScore, lowScoreIndex, err := keeper.GetLowScoreFromAllLossBundles(ctx, &k, topicId, allReputerLosses)
	s.Require().NoError(err)
	s.Require().Equal(lowScore, score2)
}
