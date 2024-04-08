package inference_synthesis_test

import (
	"log"
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
		{ // ROW 1
			name: "basic functionality, two workers, one forecaster",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: -0.08069823482294600},
				"worker1": {Value: 0.07295620166600340},
			},
			forecasts: &emissions.Forecasts{
				Forecasts: []*emissions.Forecast{
					{
						Forecaster: "forecaster0",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker0", Value: 0.0050411396925730900},
							{Inferer: "worker1", Value: 3.31783420309334e-05},
						},
					},
				},
			},
			networkCombinedLoss: 0.018593036157667700,
			epsilon:             1e-4,
			pInferenceSynthesis: 2.0,
			expected: map[string]*emissions.Inference{
				"forecaster0": {Value: 0.04164414697995540},
			},
			expectedErr: nil,
		},
		/*
			{ // ROW 2
				name: "basic functionality, two workers, two forecasters",
				inferenceByWorker: map[string]*emissions.Inference{
					"worker0": {Value: -0.2797477698393250},
					"worker1": {Value: 0.26856211587161100},
				},
				forecasts: &emissions.Forecasts{
					Forecasts: []*emissions.Forecast{
						{
							Forecaster: "forecaster0",
							ForecastElements: []*emissions.ForecastElement{
								{Inferer: "worker0", Value: 0.02089366880023640},
								{Inferer: "worker1", Value: 0.3342267861383700},
							},
						},
						{
							Forecaster: "forecaster1",
							ForecastElements: []*emissions.ForecastElement{
								{Inferer: "worker0", Value: 0.042947662325033700},
								{Inferer: "worker1", Value: 0.2816951641179930},
							},
						},
					},
				},
				networkCombinedLoss: 0.01569376583279220,
				epsilon:             1e-4,
				pInferenceSynthesis: 2.0,
				expected: map[string]*emissions.Inference{
					"forecaster0": {Value: -0.036824032402771200},
					"forecaster1": {Value: -0.025344200689386500},
				},
				expectedErr: nil,
			},
		*/
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
					log.Printf("Expected value: %v, Actual value: %v, Epsilon: %v. Values should match within epsilon.", expectedValue.Value, actualValue.Value, 1e-5)
					s.Require().InEpsilon(expectedValue.Value, actualValue.Value, 1e-5, "Values do not match for key: %s", key)
				}
			}
		})
	}
}
