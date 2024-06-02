package inference_synthesis_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	inference_synthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
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

		weight, err := inference_synthesis.CalcWeightFromRegret(regretFrac, maxRegret, pNorm, cNorm)
		s.Require().NoError(err)

		s.inEpsilon5(weight, tc.expectedWeight)
	}
}

/*
func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesTwoWorkersOneForecaster() {
	forecasts := &emissions.Forecasts{
		Forecasts: []*emissions.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("4")},
				},
			},
		},
	}
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.5")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expected := map[string]*emissions.Inference{
		"forecaster0": {Value: alloraMath.MustNewDecFromString("1.31351720")},
	}

	inferenceByWorker := map[string]*emissions.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("1")},
		"worker1": {Value: alloraMath.MustNewDecFromString("2")},
	}
	result, err := inference_synthesis.CalcForecastImpliedInferences(
		inferenceByWorker,
		alloraMath.GetSortedKeys(inferenceByWorker),
		forecasts,
		networkCombinedLoss,
		false,
		epsilon,
		fTolerance,
		pNorm,
		cNorm,
	)
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
*/

/*
func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesEpoch2() {
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// EPOCH 2 forecasted_loss_x_for_y
	forecasts := &emissions.Forecasts{
		Forecasts: []*emissions.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("-1.18172420646634")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.26621077264804827")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("-3.3897339254838474")},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("-2.571846047295651")},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("-2.0259184257783027")},
				},
			},
		},
	}

	// EPOCH 2 inference_x
	inferenceByWorker := map[string]*emissions.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.230622933739544")},
		"worker1": {Value: alloraMath.MustNewDecFromString("-0.19693894066605602")},
		"worker2": {Value: alloraMath.MustNewDecFromString("0.048704500498029504")},
		"worker3": {Value: alloraMath.MustNewDecFromString("0.054145121711977245")},
		"worker4": {Value: alloraMath.MustNewDecFromString("0.22919548623217473")},
	}

	// EPOCH 1 network_loss
	networkCombinedLoss := alloraMath.MustNewDecFromString("-4.9196819027651495")

	// EPOCH 2 forecast_implied_inference_0
	expected := map[string]*emissions.Inference{
		"forecaster0": {Value: alloraMath.MustNewDecFromString("0.05403102080389692")},
	}

	result, err := inference_synthesis.CalcForecastImpliedInferences(
		inferenceByWorker,
		alloraMath.GetSortedKeys(inferenceByWorker),
		forecasts,
		networkCombinedLoss,
		false,
		epsilon,
		fTolerance,
		pNorm,
		cNorm,
	)

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
*/
