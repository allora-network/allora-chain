package rewards_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func createNewTopic(s *RewardsTestSuite) uint64 {
	newTopicMsg := &types.MsgCreateNewTopic{
		Creator:         s.addrs[5].String(),
		Metadata:        "test",
		LossLogic:       "logic",
		LossMethod:      "method",
		EpochLength:     10800,
		GroundTruthLag:  10800,
		InferenceLogic:  "Ilogic",
		InferenceMethod: "Imethod",
		DefaultArg:      "ETH",
		AlphaRegret:     alloraMath.NewDecFromInt64(1),
		PNorm:           alloraMath.NewDecFromInt64(3),
		Epsilon:         alloraMath.MustNewDecFromString("0.01"),
	}
	res, err := s.msgServer.CreateNewTopic(s.ctx, newTopicMsg)
	s.Require().NoError(err)
	return res.TopicId
}

func (s *RewardsTestSuite) TestGetReputersRewardFractionsSimpleShouldOutputSameFractionsForEqualZeroScores() {
	topicId := createNewTopic(s)
	blockHeight := int64(1003)

	workerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
		s.addrs[3],
		s.addrs[4],
	}

	// Check with all scores being 0
	lastScores := make([]types.Score, 0)
	for _, workerAddr := range workerAddrs {
		for j := 0; j < 3; j++ {
			blockHeight := blockHeight - int64(j)
			scoreToAdd := types.Score{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Address:     workerAddr.String(),
				Score:       alloraMath.MustNewDecFromString("0"),
			}

			// Persist worker inference score
			err := s.emissionsKeeper.InsertWorkerInferenceScore(s.ctx, topicId, blockHeight, scoreToAdd)
			s.Require().NoError(err)

			// Persist worker forecast score
			err = s.emissionsKeeper.InsertWorkerForecastScore(s.ctx, topicId, blockHeight, scoreToAdd)
			s.Require().NoError(err)
		}

		lastScores = append(lastScores, types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     workerAddr.String(),
			Score:       alloraMath.MustNewDecFromString("0"),
		})
	}

	// Get worker rewards
	inferers, inferersRewardFractions, err := rewards.GetInferenceTaskRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		blockHeight,
		alloraMath.MustNewDecFromString("1.5"),
		alloraMath.MustNewDecFromString("0.75"),
		lastScores,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(inferersRewardFractions))

	forecasters, forecastersRewardFractions, err := rewards.GetForecastingTaskRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		blockHeight,
		alloraMath.MustNewDecFromString("1.5"),
		alloraMath.MustNewDecFromString("0.75"),
		lastScores,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(forecastersRewardFractions))

	// Check if fractions are equal for all inferers
	for i := 0; i < len(inferers); i++ {
		infererRewardFraction := inferersRewardFractions[i]
		for j := i + 1; j < len(inferers); j++ {
			if !infererRewardFraction.Equal(inferersRewardFractions[j]) {
				s.Require().Fail("Fractions are not equal for inferers")
			}
		}
	}

	// Check if fractions are equal for all forecasters
	for i := 0; i < len(forecasters); i++ {
		forecasterRewardFraction := forecastersRewardFractions[i]
		for j := i + 1; j < len(forecasters); j++ {
			if !forecasterRewardFraction.Equal(forecastersRewardFractions[j]) {
				s.Require().Fail("Fractions are not equal for forecasters")
			}
		}
	}
}

func (s *RewardsTestSuite) TestGetWorkersRewardFractionsShouldOutputSameFractionsForEqualScores() {
	topicId := createNewTopic(s)
	blockHeight := int64(1003)

	workerAddrs := []sdk.AccAddress{
		s.addrs[0],
		s.addrs[1],
		s.addrs[2],
		s.addrs[3],
		s.addrs[4],
	}

	// Generate old scores - 3 equal past scores per worker
	lastScores := make([]types.Score, 0)
	for _, workerAddr := range workerAddrs {
		for j := 0; j < 3; j++ {
			blockHeight := blockHeight - int64(j)
			scoreToAdd := types.Score{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Address:     workerAddr.String(),
				Score:       alloraMath.MustNewDecFromString("0.5"),
			}

			// Persist worker inference score
			err := s.emissionsKeeper.InsertWorkerInferenceScore(s.ctx, topicId, blockHeight, scoreToAdd)
			s.Require().NoError(err)

			// Persist worker forecast score
			err = s.emissionsKeeper.InsertWorkerForecastScore(s.ctx, topicId, blockHeight, scoreToAdd)
			s.Require().NoError(err)
		}

		lastScores = append(lastScores, types.Score{
			TopicId:     topicId,
			BlockHeight: blockHeight,
			Address:     workerAddr.String(),
			Score:       alloraMath.MustNewDecFromString("0.5"),
		})
	}

	// Get worker rewards
	inferers, inferersRewardFractions, err := rewards.GetInferenceTaskRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		blockHeight,
		alloraMath.MustNewDecFromString("1.5"),
		alloraMath.MustNewDecFromString("0.75"),
		lastScores,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(inferersRewardFractions))
	forecasters, forecastersRewardFractions, err := rewards.GetForecastingTaskRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		blockHeight,
		alloraMath.MustNewDecFromString("1.5"),
		alloraMath.MustNewDecFromString("0.75"),
		lastScores,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(forecastersRewardFractions))

	// Check if fractions are equal for all inferers
	for i := 0; i < len(inferers); i++ {
		infererRewardFraction := inferersRewardFractions[i]
		for j := i + 1; j < len(inferers); j++ {
			if !infererRewardFraction.Equal(inferersRewardFractions[j]) {
				s.Require().Fail("Fractions are not equal for inferers")
			}
		}
	}

	// Check if fractions are equal for all forecasters
	for i := 0; i < len(forecasters); i++ {
		forecasterRewardFraction := forecastersRewardFractions[i]
		for j := i + 1; j < len(forecasters); j++ {
			if !forecasterRewardFraction.Equal(forecastersRewardFractions[j]) {
				s.Require().Fail("Fractions are not equal for forecasters")
			}
		}
	}
}

func (s *RewardsTestSuite) TestGetWorkersRewardFractionsFromCsv() {
	topicId := createNewTopic(s)
	blockHeight := int64(4)

	finalEpoch := 304
	initialEpoch := 301
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch4Get := epochGet[finalEpoch]

	inferer0 := "inferer0"
	inferer1 := "inferer1"
	inferer2 := "inferer2"
	inferer3 := "inferer3"
	inferer4 := "inferer4"
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	forecaster0 := "forecaster0"
	forecaster1 := "forecaster1"
	forecaster2 := "forecaster2"
	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

	// Add scores from previous epochs
	infererLastScores := make([]types.Score, 0)
	forecasterLastScores := make([]types.Score, 0)
	for j := 0; j < 4; j++ {
		epochGet := epochGet[initialEpoch+j]
		inferersScores := []alloraMath.Dec{
			epochGet("inferer_score_0"),
			epochGet("inferer_score_1"),
			epochGet("inferer_score_2"),
			epochGet("inferer_score_3"),
			epochGet("inferer_score_4"),
		}
		forecastersScores := []alloraMath.Dec{
			epochGet("forecaster_score_0"),
			epochGet("forecaster_score_1"),
			epochGet("forecaster_score_2"),
		}

		for i, infererAddr := range infererAddresses {
			blockHeight := int64(j)
			scoreToAdd := types.Score{
				TopicId:     topicId,
				BlockHeight: int64(initialEpoch + j),
				Address:     infererAddr,
				Score:       inferersScores[i],
			}

			// Persist worker inference score
			err := s.emissionsKeeper.InsertWorkerInferenceScore(s.ctx, topicId, blockHeight, scoreToAdd)
			s.Require().NoError(err)

			if j == 3 {
				infererLastScores = append(infererLastScores, scoreToAdd)
			}
		}
		for i, forecasterAddr := range forecasterAddresses {
			blockHeight := int64(j)
			scoreToAdd := types.Score{
				TopicId:     topicId,
				BlockHeight: int64(initialEpoch + j),
				Address:     forecasterAddr,
				Score:       forecastersScores[i],
			}

			// Persist worker forecast score
			err := s.emissionsKeeper.InsertWorkerForecastScore(s.ctx, topicId, blockHeight, scoreToAdd)
			s.Require().NoError(err)

			if j == 3 {
				forecasterLastScores = append(forecasterLastScores, scoreToAdd)
			}
		}
	}

	// Get worker rewards
	inferers, inferersRewardFractions, err := rewards.GetInferenceTaskRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		blockHeight,
		alloraMath.MustNewDecFromString("3"),
		alloraMath.MustNewDecFromString("0.75"),
		infererLastScores,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(inferersRewardFractions))
	expectedValues := map[string]alloraMath.Dec{
		inferer0: epoch4Get("inferer_reward_fraction_0"),
		inferer1: epoch4Get("inferer_reward_fraction_1"),
		inferer2: epoch4Get("inferer_reward_fraction_2"),
		inferer3: epoch4Get("inferer_reward_fraction_3"),
		inferer4: epoch4Get("inferer_reward_fraction_4"),
	}
	for i, inferer := range inferers {
		testutil.InEpsilon5(s.T(), inferersRewardFractions[i], expectedValues[inferer].String())
	}

	forecasters, forecastersRewardFractions, err := rewards.GetForecastingTaskRewardFractions(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		blockHeight,
		alloraMath.MustNewDecFromString("3"),
		alloraMath.MustNewDecFromString("0.75"),
		forecasterLastScores,
	)
	s.Require().NoError(err)
	s.Require().Equal(3, len(forecastersRewardFractions))
	expectedValues = map[string]alloraMath.Dec{
		forecaster0: epoch4Get("forecaster_reward_fraction_0"),
		forecaster1: epoch4Get("forecaster_reward_fraction_1"),
		forecaster2: epoch4Get("forecaster_reward_fraction_2"),
	}
	for i, forecaster := range forecasters {
		testutil.InEpsilon5(s.T(), forecastersRewardFractions[i], expectedValues[forecaster].String())
	}
}

func (s *RewardsTestSuite) TestGetInferenceTaskEntropyFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch1Get := epochGet[301]
	epoch2Get := epochGet[302]
	topicId := uint64(1)

	taskRewardAlpha := alloraMath.MustNewDecFromString("0.1")
	betaEntropy := alloraMath.MustNewDecFromString("0.25")

	inferer0 := s.addrs[5].String()
	inferer1 := s.addrs[6].String()
	inferer2 := s.addrs[7].String()
	inferer3 := s.addrs[8].String()
	inferer4 := s.addrs[9].String()
	infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

	infererPreviousFractions := []alloraMath.Dec{
		epoch1Get("inferer_reward_fraction_smooth_0"),
		epoch1Get("inferer_reward_fraction_smooth_1"),
		epoch1Get("inferer_reward_fraction_smooth_2"),
		epoch1Get("inferer_reward_fraction_smooth_3"),
		epoch1Get("inferer_reward_fraction_smooth_4"),
	}

	// Add previous reward fractions
	for i, infererAddr := range infererAddresses {
		err := s.emissionsKeeper.SetPreviousInferenceRewardFraction(s.ctx, topicId, infererAddr, infererPreviousFractions[i])
		s.Require().NoError(err)
	}

	infererFractions := []alloraMath.Dec{
		epoch2Get("inferer_reward_fraction_0"),
		epoch2Get("inferer_reward_fraction_1"),
		epoch2Get("inferer_reward_fraction_2"),
		epoch2Get("inferer_reward_fraction_3"),
		epoch2Get("inferer_reward_fraction_4"),
	}

	inferenceEntropy, err := rewards.GetInferenceTaskEntropy(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		taskRewardAlpha,
		betaEntropy,
		infererAddresses,
		infererFractions,
	)
	s.Require().NoError(err)

	expectedEntropy := epoch2Get("inferers_entropy")
	testutil.InEpsilon5(s.T(), inferenceEntropy, expectedEntropy.String())
}

func (s *RewardsTestSuite) TestGetForecastTaskEntropyFromCsv() {
	topicId := createNewTopic(s)
	taskRewardAlpha := alloraMath.MustNewDecFromString("0.1")
	betaEntropy := alloraMath.MustNewDecFromString("0.25")

	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch1Get := epochGet[301]
	epoch2Get := epochGet[302]

	forecaster0 := s.addrs[10].String()
	forecaster1 := s.addrs[11].String()
	forecaster2 := s.addrs[12].String()
	forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

	forecasterPreviousFractions := []alloraMath.Dec{
		epoch1Get("forecaster_reward_fraction_smooth_0"),
		epoch1Get("forecaster_reward_fraction_smooth_1"),
		epoch1Get("forecaster_reward_fraction_smooth_2"),
	}

	// Add previous reward fractions
	for i, forecasterAddr := range forecasterAddresses {
		err := s.emissionsKeeper.SetPreviousForecastRewardFraction(s.ctx, topicId, forecasterAddr, forecasterPreviousFractions[i])
		s.Require().NoError(err)
	}

	forecasterFractions := []alloraMath.Dec{
		epoch2Get("forecaster_reward_fraction_0"),
		epoch2Get("forecaster_reward_fraction_1"),
		epoch2Get("forecaster_reward_fraction_2"),
	}

	forecastEntropy, err := rewards.GetForecastTaskEntropy(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		taskRewardAlpha,
		betaEntropy,
		forecasterAddresses,
		forecasterFractions,
	)
	s.Require().NoError(err)

	expectedEntropy := epoch2Get("forecasters_entropy")
	testutil.InEpsilon5(s.T(), forecastEntropy, expectedEntropy.String())
}

func (s *RewardsTestSuite) TestGetWorkersRewardsInferenceTask() {
	topicId := createNewTopic(s)
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
		alloraMath.MustNewDecFromString("0.75"),
		lastScores,
	)
	s.Require().NoError(err)
	inferenceRewards, err := rewards.GetRewardPerWorker(
		topicId,
		types.WorkerInferenceRewardType,
		alloraMath.NewDecFromInt64(100),
		inferers,
		inferersRewardFractions,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(inferenceRewards))
}

func (s *RewardsTestSuite) TestGetWorkersRewardsForecastTask() {
	topicId := createNewTopic(s)
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
		alloraMath.MustNewDecFromString("0.75"),
		lastScores,
	)
	s.Require().NoError(err)
	forecastRewards, err := rewards.GetRewardPerWorker(
		topicId,
		types.WorkerForecastRewardType,
		alloraMath.NewDecFromInt64(100),
		forecasters,
		forecastersRewardFractions,
	)
	s.Require().NoError(err)
	s.Require().Equal(5, len(forecastRewards))
}

func (s *RewardsTestSuite) TestInferenceRewardsFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch3Get := epochGet[300]
	alpha := alloraMath.MustNewDecFromString("0.1")
	totalReward, err := testutil.GetTotalRewardForTopicInEpoch(epoch3Get)
	s.Require().NoError(err)
	infererScores := []types.Score{
		{Score: epoch3Get("inferer_score_0")},
		{Score: epoch3Get("inferer_score_1")},
		{Score: epoch3Get("inferer_score_2")},
		{Score: epoch3Get("inferer_score_3")},
		{Score: epoch3Get("inferer_score_4")},
	}
	chi, gamma, err := rewards.GetChiAndGamma(
		epoch3Get("network_naive_loss"),
		epoch3Get("network_loss"),
		epoch3Get("inferers_entropy"),
		epoch3Get("forecasters_entropy"),
		infererScores,
		epoch3Get("forecaster_score_ratio"),
		alpha,
	)
	s.Require().NoError(err)

	result, err := rewards.GetRewardForInferenceTaskInTopic(
		epoch3Get("inferers_entropy"),
		epoch3Get("forecasters_entropy"),
		epoch3Get("reputers_entropy"),
		&totalReward,
		chi,
		gamma,
	)
	s.Require().NoError(err)
	expectedTotalInfererReward, err := testutil.GetTotalInfererRewardForTopicInEpoch(epoch3Get)
	s.Require().NoError(err)
	testutil.InEpsilon5(s.T(), result, expectedTotalInfererReward.String())
}

func (s *RewardsTestSuite) TestForecastRewardsFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch3Get := epochGet[300]
	alpha := alloraMath.MustNewDecFromString("0.1")
	totalReward, err := testutil.GetTotalRewardForTopicInEpoch(epoch3Get)
	s.Require().NoError(err)
	infererScores := []types.Score{
		{Score: epoch3Get("inferer_score_0")},
		{Score: epoch3Get("inferer_score_1")},
		{Score: epoch3Get("inferer_score_2")},
		{Score: epoch3Get("inferer_score_3")},
		{Score: epoch3Get("inferer_score_4")},
	}
	chi, gamma, err := rewards.GetChiAndGamma(
		epoch3Get("network_naive_loss"),
		epoch3Get("network_loss"),
		epoch3Get("inferers_entropy"),
		epoch3Get("forecasters_entropy"),
		infererScores,
		epoch3Get("forecaster_score_ratio"),
		alpha,
	)
	s.Require().NoError(err)

	result, err := rewards.GetRewardForForecastingTaskInTopic(
		epoch3Get("inferers_entropy"),
		epoch3Get("forecasters_entropy"),
		epoch3Get("reputers_entropy"),
		&totalReward,
		chi,
		gamma,
	)
	s.Require().NoError(err)
	expectedTotalForecasterReward, err := testutil.GetTotalForecasterRewardForTopicInEpoch(epoch3Get)
	s.Require().NoError(err)
	testutil.InEpsilon5(s.T(), result, expectedTotalForecasterReward.String())
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
				BlockHeight: blocks[j],
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
			BlockHeight: blocks[len(blocks)-1],
			Address:     workerAddr.String(),
			Score:       scores[i][len(scores[i])-1],
		})
	}

	return lastScores, nil
}
