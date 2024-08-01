package inference_synthesis_test

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

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("3")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("4")},
				},
			},
		},
	}

	expected := map[string]*emissionstypes.Inference{
		"forecaster0": {Value: alloraMath.MustNewDecFromString("1.055841253742177320400327600231111")},
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("1")},
		"worker1": {Value: alloraMath.MustNewDecFromString("2")},
	}
	palette := inferencesynthesis.SynthPalette{
		Logger:              inferencesynthesis.Logger(s.ctx),
		InferenceByWorker:   inferenceByWorker,
		ForecastByWorker:    map[string]*emissionstypes.Forecast{"forecaster0": forecasts.Forecasts[0]},
		Forecasters:         []string{"forecaster0"},
		Inferers:            []string{"worker0", "worker1"},
		NetworkCombinedLoss: networkCombinedLoss,
		Epsilon:             epsilon,
		PNorm:               pNorm,
		CNorm:               cNorm,
	}
	result, err := palette.CalcForecastImpliedInferences()
	s.Require().NoError(err)

	for key, expectedValue := range expected {
		actualValue, exists := result[key]
		s.Require().True(exists, "Expected key does not exist in result map")
		s.Require().True(
			alloraMath.InDelta(
				expectedValue.Value,
				actualValue.Value,
				alloraMath.MustNewDecFromString("0.0001"),
			), "Values do not match for key: %s %s %s",
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
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("4")},
				},
			},
		},
	}

	expected := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("2")},
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("1")},
		"worker1": {Value: alloraMath.MustNewDecFromString("2")},
	}
	palette := inferencesynthesis.SynthPalette{
		Logger:              inferencesynthesis.Logger(s.ctx),
		InferenceByWorker:   inferenceByWorker,
		ForecastByWorker:    map[string]*emissionstypes.Forecast{"worker0": forecasts.Forecasts[0]},
		Forecasters:         []string{"worker0"},
		Inferers:            []string{"worker0", "worker1"},
		NetworkCombinedLoss: networkCombinedLoss,
		Epsilon:             epsilon,
		PNorm:               pNorm,
		CNorm:               cNorm,
	}
	result, err := palette.CalcForecastImpliedInferences()
	s.Require().NoError(err)

	for key, expectedValue := range expected {
		actualValue, exists := result[key]
		s.Require().True(exists, "Expected key does not exist in result map")
		s.Require().True(
			alloraMath.InDelta(
				expectedValue.Value,
				actualValue.Value,
				alloraMath.MustNewDecFromString("0.00001"),
			), "Values do not match for key: %s %s %s",
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
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("2")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("3")},
				},
			},
			{
				Forecaster: "worker1",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("4")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("5")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("6")},
				},
			},
		},
	}

	expected := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("1.158380376510523897775902553985830")},
		"worker1": {Value: alloraMath.MustNewDecFromString("1.149124717287201046499545990921485")},
		"worker2": nil,
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("1")},
		"worker1": {Value: alloraMath.MustNewDecFromString("2")},
		"worker2": {Value: alloraMath.MustNewDecFromString("3")},
	}
	palette := inferencesynthesis.SynthPalette{
		Logger:            inferencesynthesis.Logger(s.ctx),
		InferenceByWorker: inferenceByWorker,
		ForecastByWorker: map[string]*emissionstypes.Forecast{
			"worker0": forecasts.Forecasts[0],
			"worker1": forecasts.Forecasts[1],
			// "worker2": forecasts.Forecasts[2],
		},
		Forecasters:         []string{"worker0", "worker1", "worker2"},
		Inferers:            []string{"worker0", "worker1", "worker2"},
		NetworkCombinedLoss: networkCombinedLoss,
		Epsilon:             epsilon,
		PNorm:               pNorm,
		CNorm:               cNorm,
	}
	result, err := palette.CalcForecastImpliedInferences()
	s.Require().NoError(err)

	for key, expectedValue := range expected {
		actualValue, exists := result[key]

		if expectedValue == nil {
			s.Require().False(exists, "Expected key %v exist unexpectedly in result map", key)
			s.Require().Nil(actualValue, "Expected key %v to be nil", key)
		} else {
			s.Require().True(exists, "Expected key %v does not exist in result map", key)
			s.Require().True(
				alloraMath.InDelta(
					expectedValue.Value,
					actualValue.Value,
					alloraMath.MustNewDecFromString("0.0001"),
				), "Values do not match for key: %s %s %s",
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

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: epoch2Get("forecasted_loss_0_for_0")},
					{Inferer: "worker1", Value: epoch2Get("forecasted_loss_0_for_1")},
					{Inferer: "worker2", Value: epoch2Get("forecasted_loss_0_for_2")},
					{Inferer: "worker3", Value: epoch2Get("forecasted_loss_0_for_3")},
					{Inferer: "worker4", Value: epoch2Get("forecasted_loss_0_for_4")},
				},
			},
		},
	}
	networkCombinedLoss := epoch2Get("network_loss")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expected := map[string]*emissionstypes.Inference{
		"forecaster0": {Value: epoch2Get("forecast_implied_inference_0")},
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: epoch2Get("inference_0")},
		"worker1": {Value: epoch2Get("inference_1")},
		"worker2": {Value: epoch2Get("inference_2")},
		"worker3": {Value: epoch2Get("inference_3")},
		"worker4": {Value: epoch2Get("inference_4")},
	}
	palette := inferencesynthesis.SynthPalette{
		InferenceByWorker:   inferenceByWorker,
		ForecastByWorker:    map[string]*emissionstypes.Forecast{"forecaster0": forecasts.Forecasts[0]},
		Forecasters:         []string{"forecaster0"},
		Inferers:            []string{"worker0", "worker1", "worker2", "worker3", "worker4"},
		NetworkCombinedLoss: networkCombinedLoss,
		Epsilon:             epsilon,
		PNorm:               pNorm,
		CNorm:               cNorm,
	}
	result, err := palette.CalcForecastImpliedInferences()
	s.Require().NoError(err)
	for key, expectedValue := range expected {
		actualValue, exists := result[key]
		s.Require().True(exists, "Expected key does not exist in result map")
		s.Require().True(
			alloraMath.InDelta(
				expectedValue.Value,
				actualValue.Value,
				alloraMath.MustNewDecFromString("0.001"),
			), "Values do not match for key: %s %s %s",
			key,
			expectedValue.Value.String(),
			actualValue.Value.String(),
		)
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcForcastImpliedInferencesEpoch3() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch3Get := epochGet[303]

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: epoch3Get("forecasted_loss_0_for_0")},
					{Inferer: "worker1", Value: epoch3Get("forecasted_loss_0_for_1")},
					{Inferer: "worker2", Value: epoch3Get("forecasted_loss_0_for_2")},
					{Inferer: "worker3", Value: epoch3Get("forecasted_loss_0_for_3")},
					{Inferer: "worker4", Value: epoch3Get("forecasted_loss_0_for_4")},
				},
			},
		},
	}

	networkCombinedLoss := epoch3Get("network_loss")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expected := map[string]*emissionstypes.Inference{
		"forecaster0": {Value: epoch3Get("forecast_implied_inference_0")},
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: epoch3Get("inference_0")},
		"worker1": {Value: epoch3Get("inference_1")},
		"worker2": {Value: epoch3Get("inference_2")},
		"worker3": {Value: epoch3Get("inference_3")},
		"worker4": {Value: epoch3Get("inference_4")},
	}
	palette := inferencesynthesis.SynthPalette{
		Logger:              inferencesynthesis.Logger(s.ctx),
		InferenceByWorker:   inferenceByWorker,
		ForecastByWorker:    map[string]*emissionstypes.Forecast{"forecaster0": forecasts.Forecasts[0]},
		Forecasters:         []string{"forecaster0"},
		Inferers:            []string{"worker0", "worker1", "worker2", "worker3", "worker4"},
		NetworkCombinedLoss: networkCombinedLoss,
		Epsilon:             epsilon,
		PNorm:               pNorm,
		CNorm:               cNorm,
	}
	result, err := palette.CalcForecastImpliedInferences()
	s.Require().NoError(err)
	for key, expectedValue := range expected {
		actualValue, exists := result[key]
		s.Require().True(exists, "Expected key does not exist in result map")
		s.Require().True(
			alloraMath.InDelta(
				expectedValue.Value,
				actualValue.Value,
				alloraMath.MustNewDecFromString("0.01"),
			), "Values do not match for key: %s %s %s",
			key,
			expectedValue.Value.String(),
			actualValue.Value.String(),
		)
	}
}
