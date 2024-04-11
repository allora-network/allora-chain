package inference_synthesis_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	inference_synthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *InferenceSynthesisTestSuite) TestGradient() {
	tests := []struct {
		name        string
		p           alloraMath.Dec
		x           alloraMath.Dec
		expected    alloraMath.Dec
		expectedErr error
	}{
		{
			name:        "normal operation 1",
			p:           alloraMath.MustNewDecFromString("2"),
			x:           alloraMath.MustNewDecFromString("1"),
			expected:    alloraMath.MustNewDecFromString("1.92014"),
			expectedErr: nil,
		},
		{
			name:        "normal operation 2",
			p:           alloraMath.MustNewDecFromString("10"),
			x:           alloraMath.MustNewDecFromString("3"),
			expected:    alloraMath.MustNewDecFromString("216663.907950817"),
			expectedErr: nil,
		},
		{
			name:        "normal operation 3",
			p:           alloraMath.MustNewDecFromString("9.2"),
			x:           alloraMath.MustNewDecFromString("3.4"),
			expected:    alloraMath.MustNewDecFromString("219724.179615500"),
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			result, err := inference_synthesis.Gradient(tc.p, tc.x)

			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {
				s.Require().NoError(err)
				s.Require().True(
					alloraMath.InDelta(
						tc.expected,
						result,
						alloraMath.MustNewDecFromString("0.00001")),
					"result should match expected value within epsilon",
					tc.expected.String(),
					result.String(),
				)
			}
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcForcastImpliedInferences() {
	tests := []struct {
		name                string
		inferenceByWorker   map[string]*emissions.Inference
		forecasts           *emissions.Forecasts
		networkCombinedLoss alloraMath.Dec
		epsilon             alloraMath.Dec
		pInferenceSynthesis alloraMath.Dec
		expected            map[string]*emissions.Inference
		expectedErr         error
	}{

		{ // Dummy Data
			name: "simple example, two workers, one forecaster",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("1")},
				"worker1": {Value: alloraMath.MustNewDecFromString("2")},
			},
			forecasts: &emissions.Forecasts{
				Forecasts: []*emissions.Forecast{
					{
						Forecaster: "forecaster0",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("3")},
							{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("4")},
						},
					},
				},
			},
			networkCombinedLoss: alloraMath.MustNewDecFromString("0.5"),
			epsilon:             alloraMath.MustNewDecFromString("1e-4"),
			pInferenceSynthesis: alloraMath.MustNewDecFromString("2.0"),
			expected: map[string]*emissions.Inference{
				"forecaster0": {Value: alloraMath.MustNewDecFromString("1.4355951")},
			},
			expectedErr: nil,
		},
		{ // ROW 2
			name: "basic functionality 2, two workers, one forecaster",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("-0.2797477698393250")},
				"worker1": {Value: alloraMath.MustNewDecFromString("0.26856211587161100")},
				"worker2": {Value: alloraMath.MustNewDecFromString("0.003934174100448460")},
			},
			forecasts: &emissions.Forecasts{
				Forecasts: []*emissions.Forecast{
					{
						Forecaster: "forecaster0",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.02089366880023640")},
							{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.3342267861383700")},
							{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.0002604615062174660")},
						},
					},
				},
			},
			networkCombinedLoss: alloraMath.MustNewDecFromString("0.018593036157667700"), // <- from Row 1
			epsilon:             alloraMath.MustNewDecFromString("1e-4"),
			pInferenceSynthesis: alloraMath.MustNewDecFromString("2.0"),
			expected: map[string]*emissions.Inference{
				"forecaster0": {Value: alloraMath.MustNewDecFromString("-0.036824032402771200")},
			},
			expectedErr: nil,
		},
		{ // ROW 3
			name: "basic functionality 3, two workers, one forecaster",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("-0.05142348924899710")},
				"worker1": {Value: alloraMath.MustNewDecFromString("-0.031653221198924200")},
				"worker2": {Value: alloraMath.MustNewDecFromString("-0.1018014248041400")},
			},
			forecasts: &emissions.Forecasts{
				Forecasts: []*emissions.Forecast{
					{
						Forecaster: "forecaster0",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.00011708024633613200")},
							{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.013382222402411400")},
							{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("3.82471429104471e-05")},
						},
					},
				},
			},
			networkCombinedLoss: alloraMath.MustNewDecFromString("0.01569376583279220"), // <- from Row 2
			epsilon:             alloraMath.MustNewDecFromString("1e-4"),
			pInferenceSynthesis: alloraMath.MustNewDecFromString("2.0"),
			expected: map[string]*emissions.Inference{
				"forecaster0": {Value: alloraMath.MustNewDecFromString("-0.07075177115182300")},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			result, err := inference_synthesis.CalcForcastImpliedInferences(tc.inferenceByWorker, tc.forecasts, tc.networkCombinedLoss, tc.epsilon, tc.pInferenceSynthesis)

			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {

				for key, expectedValue := range tc.expected {
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
		})
	}
}
