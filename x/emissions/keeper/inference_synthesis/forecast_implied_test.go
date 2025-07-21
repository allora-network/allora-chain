package inferencesynthesis_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *InferenceSynthesisTestSuite) TestCalcWeightFromRegret() {
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	testCases := []struct {
		regretFrac     string
		maxRegret      string
		expectedWeight string
	}{
		{"-24.5", "-24.5", "0.0007835709572871582"},
		{"-20.0", "-19.5", "0.00017487379698341595"},
		{"-15.0", "-14.0", "0.0000390213853994281"},
		{"-10.5", "-10.0", "0.00017487379698341595"},
		{"-5.75", "-5.0", "0.00008260707334375042"},
		{"-1.0", "-0.5", "0.015660377080675192"},
		{"-0.25", "0.0", "0.14227761953270035"},
		{"0.0", "0.0", "0.28604839469732846"},
		{"0.5", "0.5", "0.9624639024738211"},
		{"1.0", "1.0", "2.037536097526179"},
		{"-1.32345", "0.1238729", "0.00595380787049663"},
		{"-0.8712641", "-0.8712641", "0.022985964160663532"},
		{"0.01987392", "0.01987392", "0.30185357993405315"},
	}

	for _, tc := range testCases {
		regretFrac := alloraMath.MustNewDecFromString(tc.regretFrac)
		maxRegret := alloraMath.MustNewDecFromString(tc.maxRegret)

		weight, err := inferencesynthesis.CalcWeightFromNormalizedRegret(regretFrac, maxRegret, pNorm, cNorm)
		s.Require().NoError(err)

		testutil.InEpsilon5(s.T(), weight, tc.expectedWeight)
	}
}

func (s *InferenceSynthesisTestSuite) TestIncreasingPNormIncreasesRegretSpread() {
	cNorm := alloraMath.MustNewDecFromString("0.75")

	testCases := []struct {
		regretFrac string
		maxRegret  string
	}{
		{"-24.5", "-24.5"},
		{"-20.0", "-19.5"},
		{"-15.0", "-14.0"},
		{"-10.5", "-10.0"},
		{"-5.75", "-5.0"},
		{"-1.0", "-0.5"},
		{"-0.25", "0.0"},
		{"0.0", "0.0"},
		{"0.5", "0.5"},
		{"1.0", "1.0"},
		{"-1.32345", "0.1238729"},
		{"-0.8712641", "-0.8712641"},
		{"0.01987392", "0.01987392"},
	}

	weightWithPNorm2_point_5 := make([]alloraMath.Dec, len(testCases))
	weightWithPNorm4_point_5 := make([]alloraMath.Dec, len(testCases))

	for i, tc := range testCases {
		regretFrac := alloraMath.MustNewDecFromString(tc.regretFrac)
		maxRegret := alloraMath.MustNewDecFromString(tc.maxRegret)
		pNorm2_point_5 := alloraMath.MustNewDecFromString("2.5")
		pNorm4_point_5 := alloraMath.MustNewDecFromString("4.5")

		weght2_point_5, err := inferencesynthesis.CalcWeightFromNormalizedRegret(regretFrac, maxRegret, pNorm2_point_5, cNorm)
		s.Require().NoError(err)
		weightWithPNorm2_point_5[i] = weght2_point_5

		weght4_point_5, err := inferencesynthesis.CalcWeightFromNormalizedRegret(regretFrac, maxRegret, pNorm4_point_5, cNorm)
		s.Require().NoError(err)
		weightWithPNorm4_point_5[i] = weght4_point_5
	}

	stdDev2_point_5, err := alloraMath.StdDev(weightWithPNorm2_point_5)
	s.Require().NoError(err)

	stdDev4_point_5, err := alloraMath.StdDev(weightWithPNorm4_point_5)
	s.Require().NoError(err)

	s.Require().True(stdDev2_point_5.Lt(stdDev4_point_5))
}

func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesTwoWorkersOneForecaster() {
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.5")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	topicId := uint64(1)

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("3")},
					{Inferer: s.AddrsStr(1), Value: alloraMath.MustNewDecFromString("4")},
				},
			},
		},
	}

	expected := map[string]*emissionstypes.Inference{
		"forecaster0": {Value: alloraMath.MustNewDecFromString("1.055841253742177320400327600231111")},
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0":     {Value: alloraMath.MustNewDecFromString("1")},
		s.AddrsStr(1): {Value: alloraMath.MustNewDecFromString("2")},
	}

	allInferersAreNew := false
	inferers := []string{"worker0", s.AddrsStr(1)}
	forecasters := []string{"forecaster0"}
	forecastByWorker := map[string]*emissionstypes.Forecast{"forecaster0": forecasts.Forecasts[0]}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		"worker0":     &zero,
		s.AddrsStr(1): &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{

		"forecaster0": &zero,
	}

	result, _, _, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:               s.Ctx().Logger(),
			TopicId:              topicId,
			AllInferersAreNew:    allInferersAreNew,
			Inferers:             inferers,
			InfererToInference:   inferenceByWorker,
			InfererToRegret:      infererRegrets,
			Forecasters:          forecasters,
			ForecasterToForecast: forecastByWorker,
			ForecasterToRegret:   forecasterRegrets,
			NetworkCombinedLoss:  networkCombinedLoss,
			EpsilonTopic:         epsilon,
			PNorm:                pNorm,
			CNorm:                cNorm,
			StdDevPlusEpsilon:    alloraMath.ZeroDec(),
		},
	)
	s.Require().NoError(err)

	for key, expectedValue := range expected {
		actualValue, exists := result[key]
		s.Require().True(exists, "Expected key does not exist in result map")
		inDelta, err := alloraMath.InDelta(
			expectedValue.Value,
			actualValue.Value,
			alloraMath.MustNewDecFromString("0.0001"),
		)
		s.Require().NoError(err)
		s.Require().True(
			inDelta, "Values do not match for key: %s %s %s",
			key,
			expectedValue.Value.String(),
			actualValue.Value.String(),
		)
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesTwoWorkersTwoForecastersWithoutSelfReport() {
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.5")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "worker0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: s.AddrsStr(1), Value: alloraMath.MustNewDecFromString("4")},
				},
			},
		},
	}

	expected := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("2")},
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0":     {Value: alloraMath.MustNewDecFromString("1")},
		s.AddrsStr(1): {Value: alloraMath.MustNewDecFromString("2")},
	}

	topicId := uint64(1)
	allInferersAreNew := false
	inferers := []string{"worker0", s.AddrsStr(1)}
	forecasters := []string{"worker0"}
	forecastByWorker := map[string]*emissionstypes.Forecast{"worker0": forecasts.Forecasts[0]}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		"worker0":     &zero,
		s.AddrsStr(1): &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{
		"worker0": &zero,
	}

	result, _, _, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:               s.Ctx().Logger(),
			TopicId:              topicId,
			AllInferersAreNew:    allInferersAreNew,
			Inferers:             inferers,
			InfererToInference:   inferenceByWorker,
			InfererToRegret:      infererRegrets,
			Forecasters:          forecasters,
			ForecasterToForecast: forecastByWorker,
			ForecasterToRegret:   forecasterRegrets,
			NetworkCombinedLoss:  networkCombinedLoss,
			EpsilonTopic:         epsilon,
			PNorm:                pNorm,
			CNorm:                cNorm,
			StdDevPlusEpsilon:    alloraMath.ZeroDec(),
		},
	)
	s.Require().NoError(err)

	for key, expectedValue := range expected {
		actualValue, exists := result[key]
		s.Require().True(exists, "Expected key does not exist in result map")
		inDelta, err := alloraMath.InDelta(
			expectedValue.Value,
			actualValue.Value,
			alloraMath.MustNewDecFromString("0.00001"),
		)
		s.Require().NoError(err)
		s.Require().True(
			inDelta, "Values do not match for key: %s %s %s",
			key,
			expectedValue.Value.String(),
			actualValue.Value.String(),
		)
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesThreeWorkersThreeForecastersWithoutSelfReport() {
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.5")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "worker0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("1")},
					{Inferer: s.AddrsStr(1), Value: alloraMath.MustNewDecFromString("2")},
					{Inferer: s.AddrsStr(2), Value: alloraMath.MustNewDecFromString("3")},
				},
			},
			{
				Forecaster: s.AddrsStr(1),
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("4")},
					{Inferer: s.AddrsStr(1), Value: alloraMath.MustNewDecFromString("5")},
					{Inferer: s.AddrsStr(2), Value: alloraMath.MustNewDecFromString("6")},
				},
			},
		},
	}

	expected := map[string]*emissionstypes.Inference{
		"worker0":     {Value: alloraMath.MustNewDecFromString("1.158380376510523897775902553985830")},
		s.AddrsStr(1): {Value: alloraMath.MustNewDecFromString("1.149124717287201046499545990921485")},
		s.AddrsStr(2): nil,
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0":     {Value: alloraMath.MustNewDecFromString("1")},
		s.AddrsStr(1): {Value: alloraMath.MustNewDecFromString("2")},
		s.AddrsStr(2): {Value: alloraMath.MustNewDecFromString("3")},
	}

	topicId := uint64(1)
	allInferersAreNew := false
	inferers := []string{"worker0", s.AddrsStr(1), s.AddrsStr(2)}
	forecasters := []string{"worker0", s.AddrsStr(1)}
	forecastByWorker := map[string]*emissionstypes.Forecast{
		"worker0":     forecasts.Forecasts[0],
		s.AddrsStr(1): forecasts.Forecasts[1],
	}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		"worker0":     &zero,
		s.AddrsStr(1): &zero,
		s.AddrsStr(2): &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{
		"worker0":     &zero,
		s.AddrsStr(1): &zero,
	}

	result, _, _, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:               s.Ctx().Logger(),
			TopicId:              topicId,
			AllInferersAreNew:    allInferersAreNew,
			Inferers:             inferers,
			InfererToInference:   inferenceByWorker,
			InfererToRegret:      infererRegrets,
			Forecasters:          forecasters,
			ForecasterToForecast: forecastByWorker,
			ForecasterToRegret:   forecasterRegrets,
			NetworkCombinedLoss:  networkCombinedLoss,
			EpsilonTopic:         epsilon,
			PNorm:                pNorm,
			CNorm:                cNorm,
			StdDevPlusEpsilon:    alloraMath.ZeroDec(),
		},
	)
	s.Require().NoError(err)

	for key, expectedValue := range expected {
		actualValue, exists := result[key]

		if expectedValue == nil {
			s.Require().False(exists, "Expected key %v exist unexpectedly in result map", key)
			s.Require().Nil(actualValue, "Expected key %v to be nil", key)
		} else {
			s.Require().True(exists, "Expected key %v does not exist in result map", key)
			inDelta, err := alloraMath.InDelta(
				expectedValue.Value,
				actualValue.Value,
				alloraMath.MustNewDecFromString("0.0001"),
			)
			s.Require().NoError(err)
			s.Require().True(
				inDelta, "Values do not match for key: %s %s %s",
				key,
				expectedValue.Value.String(),
				actualValue.Value.String(),
			)
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcForcastImpliedInferencesEpoch2() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch2Get := epochGet[302]

	worker0 := s.AddrsStr(0)
	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)
	worker3 := s.AddrsStr(3)
	worker4 := s.AddrsStr(4)
	forecaster0 := s.AddrsStr(5)

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: forecaster0,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker0, Value: epoch2Get("forecasted_loss_0_for_0")},
					{Inferer: worker1, Value: epoch2Get("forecasted_loss_0_for_1")},
					{Inferer: worker2, Value: epoch2Get("forecasted_loss_0_for_2")},
					{Inferer: worker3, Value: epoch2Get("forecasted_loss_0_for_3")},
					{Inferer: worker4, Value: epoch2Get("forecasted_loss_0_for_4")},
				},
			},
		},
	}
	networkCombinedLoss := epoch2Get("network_loss")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expected := map[string]*emissionstypes.Inference{
		forecaster0: {Value: epoch2Get("forecast_implied_inference_0")},
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		worker0: {Value: epoch2Get("inference_0")},
		worker1: {Value: epoch2Get("inference_1")},
		worker2: {Value: epoch2Get("inference_2")},
		worker3: {Value: epoch2Get("inference_3")},
		worker4: {Value: epoch2Get("inference_4")},
	}

	topicId := uint64(1)
	allInferersAreNew := false
	inferers := []string{worker0, worker1, worker2, worker3, worker4}
	forecasters := []string{forecaster0}
	forecastByWorker := map[string]*emissionstypes.Forecast{forecaster0: forecasts.Forecasts[0]}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		worker0: &zero,
		worker1: &zero,
		worker2: &zero,
		worker3: &zero,
		worker4: &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{
		forecaster0: &zero,
	}

	result, _, _, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:               s.Ctx().Logger(),
			TopicId:              topicId,
			AllInferersAreNew:    allInferersAreNew,
			Inferers:             inferers,
			InfererToInference:   inferenceByWorker,
			InfererToRegret:      infererRegrets,
			Forecasters:          forecasters,
			ForecasterToForecast: forecastByWorker,
			ForecasterToRegret:   forecasterRegrets,
			NetworkCombinedLoss:  networkCombinedLoss,
			EpsilonTopic:         epsilon,
			PNorm:                pNorm,
			CNorm:                cNorm,
			StdDevPlusEpsilon:    alloraMath.ZeroDec(),
		})
	s.Require().NoError(err)
	for key, expectedValue := range expected {
		actualValue, exists := result[key]
		s.Require().True(exists, "Expected key does not exist in result map")
		inDelta, err := alloraMath.InDelta(
			expectedValue.Value,
			actualValue.Value,
			alloraMath.MustNewDecFromString("0.001"),
		)
		s.Require().NoError(err)
		s.Require().True(
			inDelta, "Values do not match for key: %s %s %s",
			key,
			expectedValue.Value.String(),
			actualValue.Value.String(),
		)
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcForcastImpliedInferencesEpoch3() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch3Get := epochGet[303]

	worker0 := s.AddrsStr(0)
	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)
	worker3 := s.AddrsStr(3)
	worker4 := s.AddrsStr(4)
	forecaster0 := s.AddrsStr(5)

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: forecaster0,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker0, Value: epoch3Get("forecasted_loss_0_for_0")},
					{Inferer: worker1, Value: epoch3Get("forecasted_loss_0_for_1")},
					{Inferer: worker2, Value: epoch3Get("forecasted_loss_0_for_2")},
					{Inferer: worker3, Value: epoch3Get("forecasted_loss_0_for_3")},
					{Inferer: worker4, Value: epoch3Get("forecasted_loss_0_for_4")},
				},
			},
		},
	}

	networkCombinedLoss := epoch3Get("network_loss")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expected := map[string]*emissionstypes.Inference{
		forecaster0: {Value: epoch3Get("forecast_implied_inference_0")},
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		worker0: {Value: epoch3Get("inference_0")},
		worker1: {Value: epoch3Get("inference_1")},
		worker2: {Value: epoch3Get("inference_2")},
		worker3: {Value: epoch3Get("inference_3")},
		worker4: {Value: epoch3Get("inference_4")},
	}
	topicId := uint64(1)
	allInferersAreNew := false
	inferers := []string{worker0, worker1, worker2, worker3, worker4}
	forecasters := []string{forecaster0}
	forecastByWorker := map[string]*emissionstypes.Forecast{forecaster0: forecasts.Forecasts[0]}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		worker0: &zero,
		worker1: &zero,
		worker2: &zero,
		worker3: &zero,
		worker4: &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{
		forecaster0: &zero,
	}

	result, _, _, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:               s.Ctx().Logger(),
			TopicId:              topicId,
			AllInferersAreNew:    allInferersAreNew,
			Inferers:             inferers,
			InfererToInference:   inferenceByWorker,
			InfererToRegret:      infererRegrets,
			Forecasters:          forecasters,
			ForecasterToForecast: forecastByWorker,
			ForecasterToRegret:   forecasterRegrets,
			NetworkCombinedLoss:  networkCombinedLoss,
			EpsilonTopic:         epsilon,
			PNorm:                pNorm,
			CNorm:                cNorm,
			StdDevPlusEpsilon:    alloraMath.ZeroDec(),
		})
	s.Require().NoError(err)
	for key, expectedValue := range expected {
		actualValue, exists := result[key]
		s.Require().True(exists, "Expected key does not exist in result map")
		inDelta, err := alloraMath.InDelta(
			expectedValue.Value,
			actualValue.Value,
			alloraMath.MustNewDecFromString("0.01"),
		)
		s.Require().NoError(err)
		s.Require().True(
			inDelta, "Values do not match for key: %s %s %s",
			key,
			expectedValue.Value.String(),
			actualValue.Value.String(),
		)
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesForecasterWithNoMatchingInferences() {
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.5")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	topicId := uint64(1)

	// Forecaster provides forecasts for inferers that don't exist in InfererToInference
	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "nonexistent_worker1", Value: alloraMath.MustNewDecFromString("3")},
					{Inferer: "nonexistent_worker2", Value: alloraMath.MustNewDecFromString("4")},
				},
			},
		},
	}

	// Only one valid inferer exists
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("1")},
	}

	allInferersAreNew := false
	inferers := []string{"worker0"}
	forecasters := []string{"forecaster0"}
	forecastByWorker := map[string]*emissionstypes.Forecast{"forecaster0": forecasts.Forecasts[0]}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		"worker0": &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{
		"forecaster0": &zero,
	}

	result, _, _, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:               s.Ctx().Logger(),
			TopicId:              topicId,
			AllInferersAreNew:    allInferersAreNew,
			Inferers:             inferers,
			InfererToInference:   inferenceByWorker,
			InfererToRegret:      infererRegrets,
			Forecasters:          forecasters,
			ForecasterToForecast: forecastByWorker,
			ForecasterToRegret:   forecasterRegrets,
			NetworkCombinedLoss:  networkCombinedLoss,
			EpsilonTopic:         epsilon,
			PNorm:                pNorm,
			CNorm:                cNorm,
			StdDevPlusEpsilon:    alloraMath.ZeroDec(),
		},
	)
	s.Require().NoError(err)

	// Should return empty result since no forecasts match existing inferences
	s.Require().Empty(result, "Expected empty result when no forecasts match existing inferences")
}

func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesForecasterWithZeroValidForecasts() {
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.5")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	topicId := uint64(1)

	// Forecaster has empty forecast elements
	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster:       "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{},
			},
		},
	}

	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0":     {Value: alloraMath.MustNewDecFromString("1")},
		s.AddrsStr(1): {Value: alloraMath.MustNewDecFromString("2")},
	}

	allInferersAreNew := false
	inferers := []string{"worker0", s.AddrsStr(1)}
	forecasters := []string{"forecaster0"}
	forecastByWorker := map[string]*emissionstypes.Forecast{"forecaster0": forecasts.Forecasts[0]}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		"worker0":     &zero,
		s.AddrsStr(1): &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{
		"forecaster0": &zero,
	}

	result, _, _, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:               s.Ctx().Logger(),
			TopicId:              topicId,
			AllInferersAreNew:    allInferersAreNew,
			Inferers:             inferers,
			InfererToInference:   inferenceByWorker,
			InfererToRegret:      infererRegrets,
			Forecasters:          forecasters,
			ForecasterToForecast: forecastByWorker,
			ForecasterToRegret:   forecasterRegrets,
			NetworkCombinedLoss:  networkCombinedLoss,
			EpsilonTopic:         epsilon,
			PNorm:                pNorm,
			CNorm:                cNorm,
			StdDevPlusEpsilon:    alloraMath.ZeroDec(),
		},
	)
	s.Require().NoError(err)

	// Should return empty result since forecaster has no valid forecasts
	s.Require().Empty(result, "Expected empty result when forecaster has zero valid forecasts")
}

func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesMultipleForecastersPartialCoverage() {
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.5")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	topicId := uint64(1)

	// Multiple forecasters with different coverage patterns
	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("1.5")},
					{Inferer: s.AddrsStr(1), Value: alloraMath.MustNewDecFromString("2.5")},
					// Missing forecast for worker2
				},
			},
			{
				Forecaster: s.AddrsStr(1),
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("1.8")},
					{Inferer: s.AddrsStr(2), Value: alloraMath.MustNewDecFromString("3.2")},
					// Missing forecast for worker1
				},
			},
			{
				Forecaster: s.AddrsStr(2),
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: s.AddrsStr(1), Value: alloraMath.MustNewDecFromString("2.1")},
					{Inferer: s.AddrsStr(2), Value: alloraMath.MustNewDecFromString("3.0")},
					// Missing forecast for worker0
				},
			},
			{
				Forecaster: s.AddrsStr(3),
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "nonexistent_worker1", Value: alloraMath.MustNewDecFromString("4.0")},
					{Inferer: "nonexistent_worker2", Value: alloraMath.MustNewDecFromString("5.0")},
					// This forecaster doesn't hit any active inferers
				},
			},
		},
	}

	// All active inferers have inferences
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0":     {Value: alloraMath.MustNewDecFromString("1")},
		s.AddrsStr(1): {Value: alloraMath.MustNewDecFromString("2")},
		s.AddrsStr(2): {Value: alloraMath.MustNewDecFromString("3")},
	}

	allInferersAreNew := false
	inferers := []string{"worker0", s.AddrsStr(1), s.AddrsStr(2)}
	forecasters := []string{"forecaster0", s.AddrsStr(1), s.AddrsStr(2), s.AddrsStr(3)}
	forecastByWorker := map[string]*emissionstypes.Forecast{
		"forecaster0": forecasts.Forecasts[0],
		s.AddrsStr(1): forecasts.Forecasts[1],
		s.AddrsStr(2): forecasts.Forecasts[2],
		s.AddrsStr(3): forecasts.Forecasts[3],
	}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		"worker0":     &zero,
		s.AddrsStr(1): &zero,
		s.AddrsStr(2): &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{
		"forecaster0": &zero,
		s.AddrsStr(1): &zero,
		s.AddrsStr(2): &zero,
		s.AddrsStr(3): &zero,
	}

	result, _, _, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:               s.Ctx().Logger(),
			TopicId:              topicId,
			AllInferersAreNew:    allInferersAreNew,
			Inferers:             inferers,
			InfererToInference:   inferenceByWorker,
			InfererToRegret:      infererRegrets,
			Forecasters:          forecasters,
			ForecasterToForecast: forecastByWorker,
			ForecasterToRegret:   forecasterRegrets,
			NetworkCombinedLoss:  networkCombinedLoss,
			EpsilonTopic:         epsilon,
			PNorm:                pNorm,
			CNorm:                cNorm,
			StdDevPlusEpsilon:    alloraMath.ZeroDec(),
		},
	)
	s.Require().NoError(err)

	// Verify that only forecasters with valid forecasts have results
	s.Require().Len(result, 3, "Expected results for only 3 forecasters (excluding the one with no valid forecasts)")

	// Verify each forecaster with valid forecasts has a result
	s.Require().Contains(result, "forecaster0", "Expected forecaster0 to have a result")
	s.Require().Contains(result, s.AddrsStr(1), "Expected forecaster1 to have a result")
	s.Require().Contains(result, s.AddrsStr(2), "Expected forecaster2 to have a result")

	// Verify that forecaster3 (with no valid forecasts) is NOT in results
	s.Require().NotContains(result, s.AddrsStr(3), "Expected forecaster3 to NOT have a result since it has no valid forecasts")

	// Verify that each forecaster's result is a valid inference
	for forecaster, inference := range result {
		s.Require().NotNil(inference, "Expected inference to not be nil for forecaster: %s", forecaster)
		s.Require().Equal(topicId, inference.TopicId, "Expected correct topic ID for forecaster: %s", forecaster)
		s.Require().Equal(forecaster, inference.Inferer, "Expected correct inferer for forecaster: %s", forecaster)
		s.Require().True(inference.Value.IsPositive(), "Expected positive value for forecaster: %s", forecaster)
	}

	// Verify that the function handled missing forecasts gracefully
	// Each forecaster should have processed only the inferers they forecasted for
	// and ignored the ones they didn't forecast for
	s.Require().True(result["forecaster0"].Value.IsPositive(), "forecaster0 should have valid result despite missing worker2")
	s.Require().True(result[s.AddrsStr(1)].Value.IsPositive(), "forecaster1 should have valid result despite missing worker1")
	s.Require().True(result[s.AddrsStr(2)].Value.IsPositive(), "forecaster2 should have valid result despite missing worker0")
}
