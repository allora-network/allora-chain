package module_test

import (
	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *ModuleTestSuite) TestGetWorkerScoreInferenceTask() {
	// Generate old scores
	err := mockWorkerLastScores(s, 1)
	s.Require().NoError(err)

	// Generate last network loss
	err = mockNetworkLosses(s, 1, 1003)
	s.Require().NoError(err)

	// Get worker rewards
	workerRewards, err := module.GetWorkersRewardsInferenceTask(
		s.ctx,
		s.appModule,
		1,
		1003,
		1.5,
		100.0,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(workerRewards))
}

func (s *ModuleTestSuite) TestGetWorkerScoreForecastTask() {
	// Generate old scores
	err := mockWorkerLastScores(s, 1)
	s.Require().NoError(err)

	// Generate last network loss
	err = mockNetworkLosses(s, 1, 1003)
	s.Require().NoError(err)

	// Get worker rewards
	workerRewards, err := module.GetWorkersRewardsForecastTask(
		s.ctx,
		s.appModule,
		1,
		1003,
		1.5,
		100.0,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(workerRewards))
}

func mockNetworkLosses(s *ModuleTestSuite, topicId uint64, block int64) error {
	// Generate network losses
	oneOutLosses := []*types.WorkerAttributedLoss{
		{
			Worker: s.addrs[0].String(),
			Value:  cosmosMath.NewUint(100),
		},
		{
			Worker: s.addrs[1].String(),
			Value:  cosmosMath.NewUint(200),
		},
		{
			Worker: s.addrs[2].String(),
			Value:  cosmosMath.NewUint(300),
		},
		{
			Worker: s.addrs[3].String(),
			Value:  cosmosMath.NewUint(400),
		},
		{
			Worker: s.addrs[4].String(),
			Value:  cosmosMath.NewUint(500),
		},
	}

	oneInNaiveLosses := []*types.WorkerAttributedLoss{
		{
			Worker: s.addrs[0].String(),
			Value:  cosmosMath.NewUint(500),
		},
		{
			Worker: s.addrs[1].String(),
			Value:  cosmosMath.NewUint(400),
		},
		{
			Worker: s.addrs[2].String(),
			Value:  cosmosMath.NewUint(300),
		},
		{
			Worker: s.addrs[3].String(),
			Value:  cosmosMath.NewUint(200),
		},
		{
			Worker: s.addrs[4].String(),
			Value:  cosmosMath.NewUint(100),
		},
	}

	networkLosses := types.LossBundle{
		TopicId:          topicId,
		OneOutLosses:     oneOutLosses,
		OneInNaiveLosses: oneInNaiveLosses,
		CombinedLoss:     cosmosMath.NewUint(1500),
		NaiveLoss:        cosmosMath.NewUint(1500),
	}

	// Persist network losses
	err := s.emissionsKeeper.InsertNetworkLossBundle(s.ctx, topicId, block, networkLosses)
	if err != nil {
		return err
	}

	return nil
}

func mockWorkerLastScores(s *ModuleTestSuite, topicId uint64) error {
	workerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
		s.addrs[3],
		s.addrs[4],
	}

	var blocks = []int64{
		1001,
		1002,
		1003,
	}

	var scores = [][]float64{
		{-0.00675, -0.00622, -0.00388},
		{-0.01502, -0.01214, -0.01554},
		{0.00392, 0.00559, 0.00545},
		{0.0438, 0.04304, 0.03906},
		{0.09719, 0.09675, 0.09418},
	}

	for i, wa := range workerAddrs {
		workerAddr, err := sdk.AccAddressFromBech32(wa.String())
		if err != nil {
			return err
		}
		workerAddrs = append(workerAddrs, workerAddr)

		for j, workerNewScore := range scores[i] {
			scoreToAdd := types.Score{
				TopicId:     topicId,
				BlockNumber: blocks[j],
				Address:     workerAddr.String(),
				Score:       workerNewScore,
			}

			// Persist worker inference score
			err = s.emissionsKeeper.InsertWorkerInferenceScore(s.ctx, topicId, blocks[j], scoreToAdd)
			if err != nil {
				return err
			}

			// Persist worker forecast score
			err = s.emissionsKeeper.InsertWorkerForecastScore(s.ctx, topicId, blocks[j], scoreToAdd)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
