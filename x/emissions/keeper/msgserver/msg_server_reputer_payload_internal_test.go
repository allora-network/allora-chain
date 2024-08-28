package msgserver

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *MsgServerInternalTestSuite) TestGetLowScoreFromAllLossBundles() {
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
	lowScore, lowScoreIndex, err := lowestReputerScoreEma(ctx, k, topicId, allReputerLosses)
	s.Require().NoError(err)
	s.Require().Equal(lowScore, score2)
	s.Require().Equal(lowScoreIndex, 1)
}
