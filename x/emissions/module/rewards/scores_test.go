package rewards_test

import (
	"math"

	cosmosMath "cosmossdk.io/math"
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

	expectedScores := []float64{17.98648, 20.32339, 26.44637, 11.17804, 14.93222}
	for i, reputerScore := range scores {
		if !(math.Abs(reputerScore.Score-expectedScores[i]) <= 1e-3) {
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

	expectedScores := []float64{-0.006859433456235635, -0.015119372088498012, 0.0038085520495462163, 0.043747287132323336, 0.09712721396805202}
	for i, reputerScore := range scores {
		if !(math.Abs(reputerScore.Score-expectedScores[i]) <= 1e-5) {
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

	expectedScores := []float64{0.012463129004928653, -0.0053656225135989164, 0.07992212136127204, -0.035977785673031996, -0.0031785253425987165}
	for i, reputerScore := range scores {
		if !(math.Abs(reputerScore.Score-expectedScores[i]) <= 1e-5) {
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

	reputersLosses := []float64{0.01127, 0.01791, 0.01404, 0.02318, 0.01251}
	reputersInfererLosses := [][]float64{
		{0.0112, 0.00231, 0.02274, 0.01299, 0.02515},
		{0.01635, 0.00179, 0.03396, 0.0153, 0.01988},
		{0.01345, 0.00209, 0.03249, 0.01688, 0.02126},
		{0.01675, 0.00318, 0.02623, 0.02734, 0.03526},
		{0.02093, 0.00213, 0.02462, 0.0203, 0.03115},
	}
	reputersForecasterLosses := [][]float64{
		{0.0185, 0.01018, 0.02105, 0.01041, 0.0183},
		{0.00962, 0.01191, 0.01616, 0.01417, 0.01216},
		{0.01338, 0.0116, 0.01605, 0.0133, 0.01407},
		{0.02733, 0.01697, 0.01619, 0.01925, 0.02018},
		{0.01, 0.01545, 0.01785, 0.01662, 0.01156},
	}
	reputersNaiveLosses := []float64{0.0116, 0.01428, 0.01441, 0.01594, 0.01705}
	reputersInfererOneOutLosses := [][]float64{
		{0.0148, 0.01046, 0.01192, 0.01381, 0.01687},
		{0.01043, 0.01308, 0.01455, 0.01607, 0.01205},
		{0.01339, 0.01053, 0.01424, 0.01428, 0.01446},
		{0.01674, 0.02944, 0.01796, 0.02187, 0.01895},
		{0.01049, 0.02068, 0.01573, 0.01487, 0.02639},
	}
	reputersForecasterOneOutLosses := [][]float64{
		{0.01136, 0.01185, 0.01568, 0.00949, 0.01339},
		{0.01357, 0.01108, 0.01633, 0.01208, 0.01278},
		{0.01805, 0.01229, 0.01586, 0.01234, 0.01513},
		{0.01637, 0.01594, 0.01608, 0.02203, 0.01486},
		{0.01981, 0.02123, 0.02134, 0.0217, 0.01177},
	}
	reputersOneInNaiveLosses := [][]float64{
		{0.01588, 0.01012, 0.01467, 0.0128, 0.01234},
		{0.01239, 0.01023, 0.01712, 0.0116, 0.01639},
		{0.01419, 0.01497, 0.01629, 0.01514, 0.01133},
		{0.01936, 0.01518, 0.018, 0.02212, 0.02259},
		{0.01602, 0.01194, 0.0153, 0.0199, 0.01673},
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

	err := s.emissionsKeeper.InsertValueBundles(s.ctx, topicId, block, reputerValueBundles)
	if err != nil {
		return types.ReputerValueBundles{}, err
	}

	return reputerValueBundles, nil
}
