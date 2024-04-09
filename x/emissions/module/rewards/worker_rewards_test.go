package rewards_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *RewardsTestSuite) TestGetWorkersRewardsInferenceTask() {
	// Generate old scores
	err := mockWorkerLastScores(s, 1)
	s.Require().NoError(err)

	// Generate last network loss
	_, err = mockNetworkLosses(s, 1, 1003)
	s.Require().NoError(err)

	// Get worker rewards
	workerRewards, err := rewards.GetWorkersRewardsInferenceTask(
		s.ctx,
		s.emissionsKeeper,
		1,
		1003,
		1.5,
		100.0,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(workerRewards))
}

func (s *RewardsTestSuite) TestGetWorkersRewardsForecastTask() {
	// Generate old scores
	err := mockWorkerLastScores(s, 1)
	s.Require().NoError(err)

	// Generate last network loss
	_, err = mockNetworkLosses(s, 1, 1003)
	s.Require().NoError(err)

	// Get worker rewards
	workerRewards, err := rewards.GetWorkersRewardsForecastTask(
		s.ctx,
		s.emissionsKeeper,
		1,
		1003,
		1.5,
		100.0,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(workerRewards))
}

func mockNetworkLosses(s *RewardsTestSuite, topicId uint64, block int64) (types.ValueBundle, error) {
	oneOutInfererLosses := []*types.WithheldWorkerAttributedValue{
		{
			Worker: s.addrs[0].String(),
			Value:  0.01327,
		},
		{
			Worker: s.addrs[1].String(),
			Value:  0.01302,
		},
		{
			Worker: s.addrs[2].String(),
			Value:  0.0136,
		},
		{
			Worker: s.addrs[3].String(),
			Value:  0.01491,
		},
		{
			Worker: s.addrs[4].String(),
			Value:  0.01686,
		},
	}

	oneOutForecasterLosses := []*types.WithheldWorkerAttributedValue{
		{
			Worker: s.addrs[0].String(),
			Value:  0.01402,
		},
		{
			Worker: s.addrs[1].String(),
			Value:  0.01316,
		},
		{
			Worker: s.addrs[2].String(),
			Value:  0.01657,
		},
		{
			Worker: s.addrs[3].String(),
			Value:  0.0124,
		},
		{
			Worker: s.addrs[4].String(),
			Value:  0.01341,
		},
	}

	oneInNaiveLosses := []*types.WorkerAttributedValue{
		{
			Worker: s.addrs[0].String(),
			Value:  0.01529,
		},
		{
			Worker: s.addrs[1].String(),
			Value:  0.01141,
		},
		{
			Worker: s.addrs[2].String(),
			Value:  0.01562,
		},
		{
			Worker: s.addrs[3].String(),
			Value:  0.01444,
		},
		{
			Worker: s.addrs[4].String(),
			Value:  0.01396,
		},
	}

	networkLosses := types.ValueBundle{
		TopicId:                topicId,
		CombinedValue:          0.013481256018186383,
		NaiveValue:             0.01344474872292,
		OneOutInfererValues:    oneOutInfererLosses,
		OneOutForecasterValues: oneOutForecasterLosses,
		OneInForecasterValues:  oneInNaiveLosses,
	}

	// Persist network losses
	err := s.emissionsKeeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, block, networkLosses)
	if err != nil {
		return types.ValueBundle{}, err
	}

	return networkLosses, nil
}

func mockWorkerLastScores(s *RewardsTestSuite, topicId uint64) error {
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

	for i, workerAddr := range workerAddrs {
		for j, workerNewScore := range scores[i] {
			scoreToAdd := types.Score{
				TopicId:     topicId,
				BlockNumber: blocks[j],
				Address:     workerAddr.String(),
				Score:       workerNewScore,
			}

			// Persist worker inference score
			err := s.emissionsKeeper.InsertWorkerInferenceScore(s.ctx, topicId, blocks[j], scoreToAdd)
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
