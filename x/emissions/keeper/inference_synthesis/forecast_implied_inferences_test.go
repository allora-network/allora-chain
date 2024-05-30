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

func (s *InferenceSynthesisTestSuite) TestCalcForcastImpliedInferencesRow2() {
	forecasts := &emissions.Forecasts{
		Forecasts: []*emissions.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissions.ForecastElement{
					{
						Inferer: "worker0",
						Value:   alloraMath.MustNewDecFromString("0.02089366880023640"),
					},
					{
						Inferer: "worker1",
						Value:   alloraMath.MustNewDecFromString("0.3342267861383700"),
					},
					{
						Inferer: "worker2",
						Value:   alloraMath.MustNewDecFromString("0.0002604615062174660"),
					},
				},
			},
		},
	}
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.018593036157667700") // <- from Row 1
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expected := map[string]*emissions.Inference{
		"forecaster0": {Value: alloraMath.MustNewDecFromString("-0.05053616")},
	}
	inferenceByWorker := map[string]*emissions.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.2797477698393250")},
		"worker1": {Value: alloraMath.MustNewDecFromString("0.26856211587161100")},
		"worker2": {Value: alloraMath.MustNewDecFromString("0.003934174100448460")},
	}
	result, err := inference_synthesis.CalcForecastImpliedInferences(
		inferenceByWorker,
		alloraMath.GetSortedKeys(inferenceByWorker),
		forecasts,
		networkCombinedLoss,
		false,
		epsilon,
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
					{
						Inferer: "worker0",
						Value:   alloraMath.MustNewDecFromString("0.00011708024633613200"),
					},
					{
						Inferer: "worker1",
						Value:   alloraMath.MustNewDecFromString("0.013382222402411400"),
					},
					{
						Inferer: "worker2",
						Value:   alloraMath.MustNewDecFromString("3.82471429104471e-05"),
					},
				},
			},
		},
	}
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.01569376583279220")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expected := map[string]*emissions.Inference{
		"forecaster0": {Value: alloraMath.MustNewDecFromString("-0.06961422309")},
	}
	inferenceByWorker := map[string]*emissions.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.05142348924899710")},
		"worker1": {Value: alloraMath.MustNewDecFromString("-0.031653221198924200")},
		"worker2": {Value: alloraMath.MustNewDecFromString("-0.1018014248041400")},
	}
	result, err := inference_synthesis.CalcForecastImpliedInferences(
		inferenceByWorker,
		alloraMath.GetSortedKeys(inferenceByWorker),
		forecasts,
		networkCombinedLoss,
		false,
		epsilon,
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
