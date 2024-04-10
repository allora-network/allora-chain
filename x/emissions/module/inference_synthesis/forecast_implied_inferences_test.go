package inference_synthesis_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	inference_synthesis "github.com/allora-network/allora-chain/x/emissions/module/inference_synthesis"
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
			name:        "normal operation",
			p:           alloraMath.MustNewDecFromString("2"),
			x:           alloraMath.MustNewDecFromString("1"),
			expected:    alloraMath.MustNewDecFromString("1.92014"),
			expectedErr: nil,
		},
		{
			name:        "normal operation",
			p:           alloraMath.MustNewDecFromString("10"),
			x:           alloraMath.MustNewDecFromString("3"),
			expected:    alloraMath.MustNewDecFromString("216664"),
			expectedErr: nil,
		},
		{
			name:        "normal operation",
			p:           alloraMath.MustNewDecFromString("9.2"),
			x:           alloraMath.MustNewDecFromString("3.4"),
			expected:    alloraMath.MustNewDecFromString("219724"),
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
				s.Require().True(alloraMath.InDelta(
					tc.expected,
					result,
					alloraMath.MustNewDecFromString("0.00001")), "result should match expected value within epsilon")
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
		{ // ROW 1
			name: "basic functionality, two workers, one forecaster",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("-0.08069823482294600")},
				"worker1": {Value: alloraMath.MustNewDecFromString("0.07295620166600340")},
			},
			forecasts: &emissions.Forecasts{
				Forecasts: []*emissions.Forecast{
					{
						Forecaster: "forecaster0",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.0050411396925730900")},
							{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0000331783420309334")},
						},
					},
				},
			},
			networkCombinedLoss: alloraMath.MustNewDecFromString("0.018593036157667700"),
			epsilon:             alloraMath.MustNewDecFromString("0.0001"),
			pInferenceSynthesis: alloraMath.MustNewDecFromString("2.0"),
			expected: map[string]*emissions.Inference{
				"forecaster0": {Value: alloraMath.MustNewDecFromString("0.04164414697995540")},
			},
			expectedErr: nil,
		},
		{ // ROW 2
			name: "basic functionality 2, two workers, two forecasters",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("-0.2797477698393250")},
				"worker1": {Value: alloraMath.MustNewDecFromString("0.26856211587161100")},
			},
			forecasts: &emissions.Forecasts{
				Forecasts: []*emissions.Forecast{
					{
						Forecaster: "forecaster0",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.02089366880023640")},
							{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.3342267861383700")},
						},
					},
					{
						Forecaster: "forecaster1",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.042947662325033700")},
							{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.2816951641179930")},
						},
					},
				},
			},
			networkCombinedLoss: alloraMath.MustNewDecFromString("0.01569376583279220"),
			epsilon:             alloraMath.MustNewDecFromString("0.0001"),
			pInferenceSynthesis: alloraMath.MustNewDecFromString("2.0"),
			expected: map[string]*emissions.Inference{
				"forecaster0": {Value: alloraMath.MustNewDecFromString("-0.036824032402771200")},
				"forecaster1": {Value: alloraMath.MustNewDecFromString("-0.025344200689386500")},
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
