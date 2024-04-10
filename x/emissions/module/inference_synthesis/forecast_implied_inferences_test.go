package inference_synthesis_test

import (
	"math"

	inference_synthesis "github.com/allora-network/allora-chain/x/emissions/module/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *InferenceSynthesisTestSuite) TestGradient() {
	tests := []struct {
		name        string
		p           float64
		x           float64
		expected    float64
		expectedErr error
	}{
		{
			name:        "normal operation",
			p:           2,
			x:           1,
			expected:    1.92014,
			expectedErr: nil,
		},
		{
			name:        "normal operation",
			p:           10,
			x:           3,
			expected:    216664,
			expectedErr: nil,
		},
		{
			name:        "normal operation",
			p:           9.2,
			x:           3.4,
			expected:    219724,
			expectedErr: nil,
		},
		{
			name:        "p is NaN",
			p:           math.NaN(),
			x:           1,
			expected:    0,
			expectedErr: types.ErrPhiInvalidInput,
		},
		{
			name:        "x is NaN",
			p:           2,
			x:           math.NaN(),
			expected:    0,
			expectedErr: types.ErrPhiInvalidInput,
		},
		{
			name:        "p is Inf",
			p:           math.Inf(1),
			x:           1,
			expected:    0,
			expectedErr: types.ErrPhiInvalidInput,
		},
		{
			name:        "x is Inf",
			p:           1,
			x:           math.Inf(1),
			expected:    0,
			expectedErr: types.ErrPhiInvalidInput,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			result, err := inference_synthesis.Gradient(tc.p, tc.x)

			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {
				s.Require().NoError(err)
				s.Require().InEpsilon(tc.expected, result, 1e-5, "result should match expected value within epsilon")
			}
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcForcastImpliedInferences() {
	tests := []struct {
		name                string
		inferenceByWorker   map[string]*emissions.Inference
		forecasts           *emissions.Forecasts
		networkCombinedLoss float64
		epsilon             float64
		pInferenceSynthesis float64
		expected            map[string]*emissions.Inference
		expectedErr         error
	}{

		{ // Dummy Data
			name: "simple example, two workers, one forecaster",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: 1},
				"worker1": {Value: 2},
			},
			forecasts: &emissions.Forecasts{
				Forecasts: []*emissions.Forecast{
					{
						Forecaster: "forecaster0",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker0", Value: 3},
							{Inferer: "worker1", Value: 4},
						},
					},
				},
			},
			networkCombinedLoss: 0.5,
			epsilon:             1e-4,
			pInferenceSynthesis: 2.0,
			expected: map[string]*emissions.Inference{
				"forecaster0": {Value: 1.4355951},
			},
			expectedErr: nil,
		},
		{ // ROW 2
			name: "basic functionality 2, two workers, one forecaster",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: -0.2797477698393250},
				"worker1": {Value: 0.26856211587161100},
				"worker2": {Value: 0.003934174100448460},
			},
			forecasts: &emissions.Forecasts{
				Forecasts: []*emissions.Forecast{
					{
						Forecaster: "forecaster0",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker0", Value: 0.02089366880023640},
							{Inferer: "worker1", Value: 0.3342267861383700},
							{Inferer: "worker2", Value: 0.0002604615062174660},
						},
					},
				},
			},
			networkCombinedLoss: 0.018593036157667700, // <- from Row 1
			epsilon:             1e-4,
			pInferenceSynthesis: 2.0,
			expected: map[string]*emissions.Inference{
				"forecaster0": {Value: -0.036824032402771200},
			},
			expectedErr: nil,
		},
		{ // ROW 3
			name: "basic functionality 3, two workers, one forecaster",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: -0.05142348924899710},
				"worker1": {Value: -0.031653221198924200},
				"worker2": {Value: -0.1018014248041400},
			},
			forecasts: &emissions.Forecasts{
				Forecasts: []*emissions.Forecast{
					{
						Forecaster: "forecaster0",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker0", Value: 0.00011708024633613200},
							{Inferer: "worker1", Value: 0.013382222402411400},
							{Inferer: "worker2", Value: 3.82471429104471e-05},
						},
					},
				},
			},
			networkCombinedLoss: 0.01569376583279220, // <- from Row 2
			epsilon:             1e-4,
			pInferenceSynthesis: 2.0,
			expected: map[string]*emissions.Inference{
				"forecaster0": {Value: -0.07075177115182300},
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
					s.Require().InEpsilon(expectedValue.Value, actualValue.Value, 1e-5, "Values do not match for key: %s", key)
				}
			}
		})
	}
}
