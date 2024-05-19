package rewards_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *RewardsTestSuite) TestGetWorkersRewardsInferenceTask() {
	topicId := uint64(1)
	blockHeight := int64(1003)

	// Generate old scores
	lastScores, err := mockWorkerLastScores(s, topicId)
	s.Require().NoError(err)

	// Get worker rewards
	inferers, inferersRewardFractions, err := rewards.GetInferenceTaskRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		blockHeight,
		alloraMath.MustNewDecFromString("1.5"),
		lastScores,
	)
	s.Require().NoError(err)
	inferenceRewards, err := rewards.GetRewardPerWorker(
		topicId,
		rewards.WorkerInferenceRewardType,
		alloraMath.NewDecFromInt64(100),
		inferers,
		inferersRewardFractions,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(inferenceRewards))
}

func (s *RewardsTestSuite) TestGetWorkersRewardsForecastTask() {
	topicId := uint64(1)
	blockHeight := int64(1003)

	// Generate old scores
	lastScores, err := mockWorkerLastScores(s, topicId)
	s.Require().NoError(err)

	// Get worker rewards
	forecasters, forecastersRewardFractions, err := rewards.GetForecastingTaskRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		blockHeight,
		alloraMath.MustNewDecFromString("1.5"),
		lastScores,
	)
	s.Require().NoError(err)
	forecastRewards, err := rewards.GetRewardPerWorker(
		topicId,
		rewards.WorkerForecastRewardType,
		alloraMath.NewDecFromInt64(100),
		forecasters,
		forecastersRewardFractions,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(forecastRewards))
}

func mockNetworkLosses(s *RewardsTestSuite, topicId uint64, block int64) (types.ValueBundle, error) {
	oneOutInfererLosses := []*types.WithheldWorkerAttributedValue{
		{
			Worker: s.addrs[0].String(),
			Value:  alloraMath.MustNewDecFromString("0.01327"),
		},
		{
			Worker: s.addrs[1].String(),
			Value:  alloraMath.MustNewDecFromString("0.01302"),
		},
		{
			Worker: s.addrs[2].String(),
			Value:  alloraMath.MustNewDecFromString("0.0136"),
		},
		{
			Worker: s.addrs[3].String(),
			Value:  alloraMath.MustNewDecFromString("0.01491"),
		},
		{
			Worker: s.addrs[4].String(),
			Value:  alloraMath.MustNewDecFromString("0.01686"),
		},
	}

	oneOutForecasterLosses := []*types.WithheldWorkerAttributedValue{
		{
			Worker: s.addrs[0].String(),
			Value:  alloraMath.MustNewDecFromString("0.01402"),
		},
		{
			Worker: s.addrs[1].String(),
			Value:  alloraMath.MustNewDecFromString("0.01316"),
		},
		{
			Worker: s.addrs[2].String(),
			Value:  alloraMath.MustNewDecFromString("0.01657"),
		},
		{
			Worker: s.addrs[3].String(),
			Value:  alloraMath.MustNewDecFromString("0.0124"),
		},
		{
			Worker: s.addrs[4].String(),
			Value:  alloraMath.MustNewDecFromString("0.01341"),
		},
	}

	oneInNaiveLosses := []*types.WorkerAttributedValue{
		{
			Worker: s.addrs[0].String(),
			Value:  alloraMath.MustNewDecFromString("0.01529"),
		},
		{
			Worker: s.addrs[1].String(),
			Value:  alloraMath.MustNewDecFromString("0.01141"),
		},
		{
			Worker: s.addrs[2].String(),
			Value:  alloraMath.MustNewDecFromString("0.01562"),
		},
		{
			Worker: s.addrs[3].String(),
			Value:  alloraMath.MustNewDecFromString("0.01444"),
		},
		{
			Worker: s.addrs[4].String(),
			Value:  alloraMath.MustNewDecFromString("0.01396"),
		},
	}

	networkLosses := types.ValueBundle{
		TopicId:                topicId,
		CombinedValue:          alloraMath.MustNewDecFromString("0.013481256018186383"),
		NaiveValue:             alloraMath.MustNewDecFromString("0.01344474872292"),
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

func mockSimpleNetworkLosses(
	s *RewardsTestSuite,
	topicId uint64,
	block int64,
	worker0Value string,
) (types.ValueBundle, error) {
	genericLossesWithheld := []*types.WithheldWorkerAttributedValue{
		{
			Worker: s.addrs[0].String(),
			Value:  alloraMath.MustNewDecFromString(worker0Value),
		},
		{
			Worker: s.addrs[1].String(),
			Value:  alloraMath.MustNewDecFromString("0.3"),
		},
		{
			Worker: s.addrs[2].String(),
			Value:  alloraMath.MustNewDecFromString("0.4"),
		},
	}

	genericLosses := []*types.WorkerAttributedValue{
		{
			Worker: s.addrs[0].String(),
			Value:  alloraMath.MustNewDecFromString(worker0Value),
		},
		{
			Worker: s.addrs[1].String(),
			Value:  alloraMath.MustNewDecFromString("0.3"),
		},
		{
			Worker: s.addrs[2].String(),
			Value:  alloraMath.MustNewDecFromString("0.4"),
		},
	}

	networkLosses := types.ValueBundle{
		TopicId:                topicId,
		CombinedValue:          alloraMath.MustNewDecFromString("0.05"),
		NaiveValue:             alloraMath.MustNewDecFromString("0.05"),
		OneOutInfererValues:    genericLossesWithheld,
		OneOutForecasterValues: genericLossesWithheld,
		OneInForecasterValues:  genericLosses,
	}

	err := s.emissionsKeeper.InsertNetworkLossBundleAtBlock(s.ctx, topicId, block, networkLosses)
	if err != nil {
		return types.ValueBundle{}, err
	}

	return networkLosses, nil
}

func mockWorkerLastScores(s *RewardsTestSuite, topicId uint64) ([]types.Score, error) {
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
	var scores = [][]alloraMath.Dec{
		{alloraMath.MustNewDecFromString("-0.00675"), alloraMath.MustNewDecFromString("-0.00622"), alloraMath.MustNewDecFromString("-0.00388")},
		{alloraMath.MustNewDecFromString("-0.01502"), alloraMath.MustNewDecFromString("-0.01214"), alloraMath.MustNewDecFromString("-0.01554")},
		{alloraMath.MustNewDecFromString("0.00392"), alloraMath.MustNewDecFromString("0.00559"), alloraMath.MustNewDecFromString("0.00545")},
		{alloraMath.MustNewDecFromString("0.0438"), alloraMath.MustNewDecFromString("0.04304"), alloraMath.MustNewDecFromString("0.03906")},
		{alloraMath.MustNewDecFromString("0.09719"), alloraMath.MustNewDecFromString("0.09675"), alloraMath.MustNewDecFromString("0.09418")},
	}

	lastScores := make([]types.Score, 0)
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
				return nil, err
			}

			// Persist worker forecast score
			err = s.emissionsKeeper.InsertWorkerForecastScore(s.ctx, topicId, blocks[j], scoreToAdd)
			if err != nil {
				return nil, err
			}
		}
		lastScores = append(lastScores, types.Score{
			TopicId:     topicId,
			BlockNumber: blocks[len(blocks)-1],
			Address:     workerAddr.String(),
			Score:       scores[i][len(scores[i])-1],
		})
	}

	return lastScores, nil
}
