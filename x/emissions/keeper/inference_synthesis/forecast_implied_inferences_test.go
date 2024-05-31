package inference_synthesis_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	inference_synthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesTwoWorkersOneForecaster() {
	forecasts := &emissions.Forecasts{
		Forecasts: []*emissions.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("3")},
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
		"forecaster0": {Value: alloraMath.MustNewDecFromString("1.019425413164753500920112202707832")},
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

func (s *InferenceSynthesisTestSuite) TestCalcForcastImpliedInferencesRow3() {
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
	networkCombinedLoss := alloraMath.MustNewDecFromString("-4.9196819027651495")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expected := map[string]*emissions.Inference{
		"forecaster0": {Value: alloraMath.MustNewDecFromString("0.05403102080389692")},
	}
	inferenceByWorker := map[string]*emissions.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.230622933739544")},
		"worker1": {Value: alloraMath.MustNewDecFromString("-0.19693894066605602")},
		"worker2": {Value: alloraMath.MustNewDecFromString("0.048704500498029504")},
		"worker3": {Value: alloraMath.MustNewDecFromString("0.054145121711977245")},
		"worker4": {Value: alloraMath.MustNewDecFromString("0.22919548623217473")},
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

func (s *InferenceSynthesisTestSuite) TestCalcForcastImpliedInferencesRow4() {
	forecasts := &emissions.Forecasts{
		Forecasts: []*emissions.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("-2.480767250656477")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("-3.5546685650440417")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("-4.6188184193555735")},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("-3.084052840898731")},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("-4.73003856038905")},
				},
			},
		},
	}
	networkCombinedLoss := alloraMath.MustNewDecFromString("-4.893498750410228") // <- from Row 1
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expected := map[string]*emissions.Inference{
		"forecaster0": {Value: alloraMath.MustNewDecFromString("-0.1025675327315208")},
	}
	inferenceByWorker := map[string]*emissions.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.035995138925040554")},
		"worker1": {Value: alloraMath.MustNewDecFromString("-0.07333303938740415")},
		"worker2": {Value: alloraMath.MustNewDecFromString("-0.1495482917094787")},
		"worker3": {Value: alloraMath.MustNewDecFromString("-0.12952123274063815")},
		"worker4": {Value: alloraMath.MustNewDecFromString("-0.0703055329498285")},
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
