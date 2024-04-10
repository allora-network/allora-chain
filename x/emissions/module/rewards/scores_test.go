package rewards_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *RewardsTestSuite) TestGetReputersScores() {
	topidId := uint64(1)
	block := int64(1003)

	// Generate reputers data for tests
	reportedLosses, err := mockReputersScoresTestData(s, topidId, block)
	s.Require().NoError(err)

	// Generate new reputer scores
	scores, err := rewards.GenerateReputerScores(
		s.ctx,
		s.emissionsKeeper,
		topidId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)

	expectedScores := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("17.98648"),
		alloraMath.MustNewDecFromString("20.32339"),
		alloraMath.MustNewDecFromString("26.44637"),
		alloraMath.MustNewDecFromString("11.17804"),
		alloraMath.MustNewDecFromString("14.93222"),
	}
	for i, reputerScore := range scores {
		scoreDelta, err := reputerScore.Score.Sub(expectedScores[i])
		s.Require().NoError(err)
		deltaTightness := scoreDelta.Abs().Cmp(alloraMath.MustNewDecFromString("0.001"))
		if !(deltaTightness == alloraMath.LessThan || deltaTightness == alloraMath.EqualTo) {
			s.Fail("Expected reward is not equal to the actual reward")
		}
	}
}

func (s *RewardsTestSuite) TestGetInferenceScores() {
	topidId := uint64(1)
	block := int64(1003)

	// Generate workers data for tests
	reportedLosses, err := mockNetworkLosses(s, topidId, block)
	s.Require().NoError(err)

	// Get inference scores
	scores, err := rewards.GenerateInferenceScores(
		s.ctx,
		s.emissionsKeeper,
		topidId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)

	expectedScores := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("-0.006859433456235635"),
		alloraMath.MustNewDecFromString("-0.015119372088498012"),
		alloraMath.MustNewDecFromString("0.0038085520495462163"),
		alloraMath.MustNewDecFromString("0.043747287132323336"),
		alloraMath.MustNewDecFromString("0.09712721396805202"),
	}
	for i, reputerScore := range scores {
		scoreDelta, err := reputerScore.Score.Sub(expectedScores[i])
		s.Require().NoError(err)
		deltaTightness := scoreDelta.Abs().
			Cmp(alloraMath.MustNewDecFromString("0.00001"))
		if !(deltaTightness == alloraMath.LessThan || deltaTightness == alloraMath.EqualTo) {
			s.Fail("Expected reward is not equal to the actual reward")
		}
	}
}

func (s *RewardsTestSuite) TestGetForecastScores() {
	topidId := uint64(1)
	block := int64(1003)

	// Generate workers data for tests
	reportedLosses, err := mockNetworkLosses(s, topidId, block)
	s.Require().NoError(err)

	// Get inference scores
	scores, err := rewards.GenerateForecastScores(
		s.ctx,
		s.emissionsKeeper,
		topidId,
		block,
		reportedLosses,
	)
	s.Require().NoError(err)

	expectedScores := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.012463129004928653"),
		alloraMath.MustNewDecFromString("-0.0053656225135989164"),
		alloraMath.MustNewDecFromString("0.07992212136127204"),
		alloraMath.MustNewDecFromString("-0.035977785673031996"),
		alloraMath.MustNewDecFromString("-0.0031785253425987165"),
	}
	for i, reputerScore := range scores {
		delta, err := reputerScore.Score.Sub(expectedScores[i])
		s.Require().NoError(err)
		deltaTightness := delta.Abs().Cmp(alloraMath.MustNewDecFromString("0.00001"))
		if !(deltaTightness == alloraMath.LessThan || deltaTightness == alloraMath.EqualTo) {
			s.Fail("Expected reward is not equal to the actual reward")
		}
	}
}

// mockReputersData generates reputer stakes and losses
func mockReputersScoresTestData(s *RewardsTestSuite, topicId uint64, block int64) (types.ReputerValueBundles, error) {
	reputers := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
		s.addrs[3],
		s.addrs[4],
	}

	workers := []sdk.AccAddress{
		s.addrs[5],
		s.addrs[6],
		s.addrs[7],
		s.addrs[8],
		s.addrs[9],
	}

	var stakes = []cosmosMath.Uint{
		cosmosMath.NewUint(1176644),
		cosmosMath.NewUint(384623),
		cosmosMath.NewUint(394676),
		cosmosMath.NewUint(207999),
		cosmosMath.NewUint(368582),
	}
	reputersLosses := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.01127"),
		alloraMath.MustNewDecFromString("0.01791"),
		alloraMath.MustNewDecFromString("0.01404"),
		alloraMath.MustNewDecFromString("0.02318"),
		alloraMath.MustNewDecFromString("0.01251"),
	}
	reputersInfererLosses := [][]alloraMath.Dec{
		{
			alloraMath.MustNewDecFromString("0.0112"),
			alloraMath.MustNewDecFromString("0.00231"),
			alloraMath.MustNewDecFromString("0.02274"),
			alloraMath.MustNewDecFromString("0.01299"),
			alloraMath.MustNewDecFromString("0.02515"),
		},
		{
			alloraMath.MustNewDecFromString("0.01635"),
			alloraMath.MustNewDecFromString("0.00179"),
			alloraMath.MustNewDecFromString("0.03396"),
			alloraMath.MustNewDecFromString("0.0153"),
			alloraMath.MustNewDecFromString("0.01988"),
		},
		{
			alloraMath.MustNewDecFromString("0.01345"),
			alloraMath.MustNewDecFromString("0.00209"),
			alloraMath.MustNewDecFromString("0.03249"),
			alloraMath.MustNewDecFromString("0.01688"),
			alloraMath.MustNewDecFromString("0.02126"),
		},
		{
			alloraMath.MustNewDecFromString("0.01675"),
			alloraMath.MustNewDecFromString("0.00318"),
			alloraMath.MustNewDecFromString("0.02623"),
			alloraMath.MustNewDecFromString("0.02734"),
			alloraMath.MustNewDecFromString("0.03526"),
		},
		{
			alloraMath.MustNewDecFromString("0.02093"),
			alloraMath.MustNewDecFromString("0.00213"),
			alloraMath.MustNewDecFromString("0.02462"),
			alloraMath.MustNewDecFromString("0.0203"),
			alloraMath.MustNewDecFromString("0.03115"),
		},
	}
	reputersForecasterLosses := [][]alloraMath.Dec{
		{
			alloraMath.MustNewDecFromString("0.0185"),
			alloraMath.MustNewDecFromString("0.01018"),
			alloraMath.MustNewDecFromString("0.02105"),
			alloraMath.MustNewDecFromString("0.01041"),
			alloraMath.MustNewDecFromString("0.0183"),
		},
		{
			alloraMath.MustNewDecFromString("0.00962"),
			alloraMath.MustNewDecFromString("0.01191"),
			alloraMath.MustNewDecFromString("0.01616"),
			alloraMath.MustNewDecFromString("0.01417"),
			alloraMath.MustNewDecFromString("0.01216"),
		},
		{
			alloraMath.MustNewDecFromString("0.01338"),
			alloraMath.MustNewDecFromString("0.0116"),
			alloraMath.MustNewDecFromString("0.01605"),
			alloraMath.MustNewDecFromString("0.0133"),
			alloraMath.MustNewDecFromString("0.01407"),
		},
		{
			alloraMath.MustNewDecFromString("0.02733"),
			alloraMath.MustNewDecFromString("0.01697"),
			alloraMath.MustNewDecFromString("0.01619"),
			alloraMath.MustNewDecFromString("0.01925"),
			alloraMath.MustNewDecFromString("0.02018"),
		},
		{
			alloraMath.MustNewDecFromString("0.01"),
			alloraMath.MustNewDecFromString("0.01545"),
			alloraMath.MustNewDecFromString("0.01785"),
			alloraMath.MustNewDecFromString("0.01662"),
			alloraMath.MustNewDecFromString("0.01156"),
		},
	}
	reputersNaiveLosses := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.0116"),
		alloraMath.MustNewDecFromString("0.01428"),
		alloraMath.MustNewDecFromString("0.01441"),
		alloraMath.MustNewDecFromString("0.01594"),
		alloraMath.MustNewDecFromString("0.01705"),
	}
	reputersInfererOneOutLosses := [][]alloraMath.Dec{
		{
			alloraMath.MustNewDecFromString("0.0148"),
			alloraMath.MustNewDecFromString("0.01046"),
			alloraMath.MustNewDecFromString("0.01192"),
			alloraMath.MustNewDecFromString("0.01381"),
			alloraMath.MustNewDecFromString("0.01687"),
		},
		{
			alloraMath.MustNewDecFromString("0.01043"),
			alloraMath.MustNewDecFromString("0.01308"),
			alloraMath.MustNewDecFromString("0.01455"),
			alloraMath.MustNewDecFromString("0.01607"),
			alloraMath.MustNewDecFromString("0.01205"),
		},
		{
			alloraMath.MustNewDecFromString("0.01339"),
			alloraMath.MustNewDecFromString("0.01053"),
			alloraMath.MustNewDecFromString("0.01424"),
			alloraMath.MustNewDecFromString("0.01428"),
			alloraMath.MustNewDecFromString("0.01446"),
		},
		{
			alloraMath.MustNewDecFromString("0.01674"),
			alloraMath.MustNewDecFromString("0.02944"),
			alloraMath.MustNewDecFromString("0.01796"),
			alloraMath.MustNewDecFromString("0.02187"),
			alloraMath.MustNewDecFromString("0.01895"),
		},
		{
			alloraMath.MustNewDecFromString("0.01049"),
			alloraMath.MustNewDecFromString("0.02068"),
			alloraMath.MustNewDecFromString("0.01573"),
			alloraMath.MustNewDecFromString("0.01487"),
			alloraMath.MustNewDecFromString("0.02639"),
		},
	}
	reputersForecasterOneOutLosses := [][]alloraMath.Dec{
		{
			alloraMath.MustNewDecFromString("0.01136"),
			alloraMath.MustNewDecFromString("0.01185"),
			alloraMath.MustNewDecFromString("0.01568"),
			alloraMath.MustNewDecFromString("0.00949"),
			alloraMath.MustNewDecFromString("0.01339"),
		},
		{
			alloraMath.MustNewDecFromString("0.01357"),
			alloraMath.MustNewDecFromString("0.01108"),
			alloraMath.MustNewDecFromString("0.01633"),
			alloraMath.MustNewDecFromString("0.01208"),
			alloraMath.MustNewDecFromString("0.01278"),
		},
		{
			alloraMath.MustNewDecFromString("0.01805"),
			alloraMath.MustNewDecFromString("0.01229"),
			alloraMath.MustNewDecFromString("0.01586"),
			alloraMath.MustNewDecFromString("0.01234"),
			alloraMath.MustNewDecFromString("0.01513"),
		},
		{
			alloraMath.MustNewDecFromString("0.01637"),
			alloraMath.MustNewDecFromString("0.01594"),
			alloraMath.MustNewDecFromString("0.01608"),
			alloraMath.MustNewDecFromString("0.02203"),
			alloraMath.MustNewDecFromString("0.01486"),
		},
		{
			alloraMath.MustNewDecFromString("0.01981"),
			alloraMath.MustNewDecFromString("0.02123"),
			alloraMath.MustNewDecFromString("0.02134"),
			alloraMath.MustNewDecFromString("0.0217"),
			alloraMath.MustNewDecFromString("0.01177"),
		},
	}
	reputersOneInNaiveLosses := [][]alloraMath.Dec{
		{
			alloraMath.MustNewDecFromString("0.01588"),
			alloraMath.MustNewDecFromString("0.01012"),
			alloraMath.MustNewDecFromString("0.01467"),
			alloraMath.MustNewDecFromString("0.0128"),
			alloraMath.MustNewDecFromString("0.01234"),
		},
		{
			alloraMath.MustNewDecFromString("0.01239"),
			alloraMath.MustNewDecFromString("0.01023"),
			alloraMath.MustNewDecFromString("0.01712"),
			alloraMath.MustNewDecFromString("0.0116"),
			alloraMath.MustNewDecFromString("0.01639"),
		},
		{
			alloraMath.MustNewDecFromString("0.01419"),
			alloraMath.MustNewDecFromString("0.01497"),
			alloraMath.MustNewDecFromString("0.01629"),
			alloraMath.MustNewDecFromString("0.01514"),
			alloraMath.MustNewDecFromString("0.01133"),
		},
		{
			alloraMath.MustNewDecFromString("0.01936"),
			alloraMath.MustNewDecFromString("0.01518"),
			alloraMath.MustNewDecFromString("0.018"),
			alloraMath.MustNewDecFromString("0.02212"),
			alloraMath.MustNewDecFromString("0.02259"),
		},
		{
			alloraMath.MustNewDecFromString("0.01602"),
			alloraMath.MustNewDecFromString("0.01194"),
			alloraMath.MustNewDecFromString("0.0153"),
			alloraMath.MustNewDecFromString("0.0199"),
			alloraMath.MustNewDecFromString("0.01673"),
		},
	}

	var reputerValueBundles types.ReputerValueBundles
	for i, reputer := range reputers {

		err := s.emissionsKeeper.AddStake(s.ctx, topicId, reputer, stakes[i])
		if err != nil {
			return types.ReputerValueBundles{}, err
		}

		bundle := &types.ReputerValueBundle{
			Reputer: reputer.String(),
			ValueBundle: &types.ValueBundle{
				CombinedValue:          reputersLosses[i],
				NaiveValue:             reputersNaiveLosses[i],
				InfererValues:          make([]*types.WorkerAttributedValue, len(workers)),
				ForecasterValues:       make([]*types.WorkerAttributedValue, len(workers)),
				OneOutInfererValues:    make([]*types.WithheldWorkerAttributedValue, len(workers)),
				OneOutForecasterValues: make([]*types.WithheldWorkerAttributedValue, len(workers)),
				OneInForecasterValues:  make([]*types.WorkerAttributedValue, len(workers)),
			},
		}

		for j, worker := range workers {
			bundle.ValueBundle.InfererValues[j] = &types.WorkerAttributedValue{Worker: worker.String(), Value: reputersInfererLosses[i][j]}
			bundle.ValueBundle.ForecasterValues[j] = &types.WorkerAttributedValue{Worker: worker.String(), Value: reputersForecasterLosses[i][j]}
			bundle.ValueBundle.OneOutInfererValues[j] = &types.WithheldWorkerAttributedValue{Worker: worker.String(), Value: reputersInfererOneOutLosses[i][j]}
			bundle.ValueBundle.OneOutForecasterValues[j] = &types.WithheldWorkerAttributedValue{Worker: worker.String(), Value: reputersForecasterOneOutLosses[i][j]}
			bundle.ValueBundle.OneInForecasterValues[j] = &types.WorkerAttributedValue{Worker: worker.String(), Value: reputersOneInNaiveLosses[i][j]}
		}
		reputerValueBundles.ReputerValueBundles = append(reputerValueBundles.ReputerValueBundles, bundle)
	}

	err := s.emissionsKeeper.InsertReputerLossBundlesAtBlock(s.ctx, topicId, block, reputerValueBundles)
	if err != nil {
		return types.ReputerValueBundles{}, err
	}

	return reputerValueBundles, nil
}
