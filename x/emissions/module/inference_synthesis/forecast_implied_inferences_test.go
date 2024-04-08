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
		{
			name: "basic functionality, two workers, one forecaster",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker1": {Value: 1.5},
				"worker2": {Value: 2.5},
			},
			forecasts: &emissions.Forecasts{
				Forecasts: []*emissions.Forecast{
					{
						Forecaster: "forecaster1",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker1", Value: 1.0},
							{Inferer: "worker2", Value: 2.0},
						},
					},
				},
			},
			networkCombinedLoss: 10.0,
			epsilon:             1e-4,
			pInferenceSynthesis: 2.0,
			expected: map[string]*emissions.Inference{
				"forecaster1": {Value: 1.9340854555354376},
			},
			expectedErr: nil,
		},
		{
			name: "basic functionality, two workers, two forecasters",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker1": {Value: 1.5},
				"worker2": {Value: 2.5},
			},
			forecasts: &emissions.Forecasts{
				Forecasts: []*emissions.Forecast{
					{
						Forecaster: "forecaster1",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker1", Value: 1.0},
							{Inferer: "worker2", Value: 2.0},
						},
					},
					{
						Forecaster: "forecaster2",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker1", Value: 2.0},
							{Inferer: "worker2", Value: 3.0},
						},
					},
				},
			},
			networkCombinedLoss: 10.0,
			epsilon:             1e-4,
			pInferenceSynthesis: 2.0,
			expected: map[string]*emissions.Inference{
				"forecaster1": {Value: 1.9340854555354376},
				"forecaster2": {Value: 1.9453156124153894},
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
