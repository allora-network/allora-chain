package inference_synthesis_test

import (
	inference_synthesis "github.com/allora-network/allora-chain/x/emissions/module/inference_synthesis"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *InferenceSynthesisTestSuite) TestCalcWeightedInference() {
	topicId := inference_synthesis.TopicId(1)

	tests := []struct {
		name                                  string
		inferenceByWorker                     map[string]*emissions.Inference
		forecastImpliedInferenceByWorker      map[string]*emissions.Inference
		maxRegret                             inference_synthesis.Regret
		epsilon                               float64
		pInferenceSynthesis                   float64
		expectedNetworkCombinedInferenceValue float64
		expectedErr                           error
	}{
		{
			name: "normal operation",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker1": {Value: 1.5},
				"worker2": {Value: 2.5},
			},
			forecastImpliedInferenceByWorker: map[string]*emissions.Inference{
				"worker1": {Value: 1.0},
				"worker2": {Value: 2.0},
			},
			maxRegret:                             0.5,
			epsilon:                               1e-4,
			pInferenceSynthesis:                   1,
			expectedNetworkCombinedInferenceValue: 2,
			expectedErr:                           nil,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			networkCombinedInferenceValue, err := inference_synthesis.CalcWeightedInference(
				s.ctx,
				s.emissionsKeeper,
				topicId,
				tc.inferenceByWorker,
				tc.forecastImpliedInferenceByWorker,
				tc.maxRegret,
				tc.epsilon,
				tc.pInferenceSynthesis,
			)

			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {
				s.Require().NoError(err)

				s.Require().InEpsilon(tc.expectedNetworkCombinedInferenceValue, networkCombinedInferenceValue, 1e-5, "Network combined inference value should match expected value within epsilon")
			}
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcOneOutInferences() {
	topicId := inference_synthesis.TopicId(1)

	test := struct {
		name                             string
		inferenceByWorker                map[string]*emissions.Inference
		forecastImpliedInferenceByWorker map[string]*emissions.Inference
		forecasts                        *emissions.Forecasts
		maxRegret                        inference_synthesis.Regret
		networkCombinedLoss              inference_synthesis.Loss
		epsilon                          float64
		pInferenceSynthesis              float64
		expectedOneOutInferences         []*emissions.WithheldWorkerAttributedValue
		expectedOneOutImpliedInferences  []*emissions.WithheldWorkerAttributedValue
	}{
		name: "example test case",
		inferenceByWorker: map[string]*emissions.Inference{
			"worker1": &emissions.Inference{Value: 1.5},
			"worker2": &emissions.Inference{Value: 2.5},
		},
		forecastImpliedInferenceByWorker: map[string]*emissions.Inference{
			"worker1": &emissions.Inference{Value: 1.0},
			"worker2": &emissions.Inference{Value: 2.0},
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
		maxRegret:           0.5,
		networkCombinedLoss: 10.0,
		epsilon:             1e-4,
		expectedOneOutInferences: []*emissions.WithheldWorkerAttributedValue{
			{Worker: "worker1", Value: 2},
			{Worker: "worker2", Value: 2},
		},
		expectedOneOutImpliedInferences: []*emissions.WithheldWorkerAttributedValue{
			{Worker: "worker1", Value: 2.166666666666666},
			{Worker: "worker2", Value: 1.833333333333333},
		},
		pInferenceSynthesis: 2.0,
	}

	s.Run(test.name, func() {
		oneOutInferences, oneOutImpliedInferences, err := inference_synthesis.CalcOneOutInferences(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			test.inferenceByWorker,
			test.forecastImpliedInferenceByWorker,
			test.forecasts,
			test.maxRegret,
			test.networkCombinedLoss,
			test.epsilon,
			test.pInferenceSynthesis,
		)

		s.Require().NoError(err, "CalcOneOutInferences should not return an error")

		s.Require().Len(oneOutInferences, len(test.expectedOneOutInferences), "Unexpected number of one-out inferences")
		s.Require().Len(oneOutImpliedInferences, len(test.expectedOneOutImpliedInferences), "Unexpected number of one-out implied inferences")

		for i, expected := range test.expectedOneOutInferences {
			s.Require().InEpsilon(expected.Value, oneOutInferences[i].Value, 1e-5, "Mismatch in value for one-out inference of worker %s", expected.Worker)
		}

		for i, expected := range test.expectedOneOutImpliedInferences {
			s.Require().InEpsilon(expected.Value, oneOutImpliedInferences[i].Value, 1e-5, "Mismatch in value for one-out implied inference of worker %s", expected.Worker)
		}
	})
}

func (s *InferenceSynthesisTestSuite) TestCalcOneInInferences() {
	topicId := inference_synthesis.TopicId(1)

	tests := []struct {
		name                        string
		inferences                  map[string]*emissions.Inference
		forecastImpliedInferences   map[string]*emissions.Inference
		maxRegretsByOneInForecaster map[string]inference_synthesis.Regret
		epsilon                     float64
		pInferenceSynthesis         float64
		expectedOneInInferences     []*emissions.WorkerAttributedValue
		expectedErr                 error
	}{
		{
			name: "basic functionality, single worker",
			inferences: map[string]*emissions.Inference{
				"worker1": {Value: 1.5},
				"worker2": {Value: 2.5},
			},
			forecastImpliedInferences: map[string]*emissions.Inference{
				"worker1": {Value: 1.0},
				"worker2": {Value: 2.0},
			},
			maxRegretsByOneInForecaster: map[string]inference_synthesis.Regret{
				"worker1": 0.1,
				"worker2": 0.2,
			},
			epsilon:             0.0001,
			pInferenceSynthesis: 2.0,
			expectedOneInInferences: []*emissions.WorkerAttributedValue{
				{Worker: "worker1", Value: 1.833333333333333},
				{Worker: "worker2", Value: 2.166666666666667},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			oneInInferences, err := inference_synthesis.CalcOneInInferences(
				s.ctx,
				s.emissionsKeeper,
				topicId,
				tc.inferences,
				tc.forecastImpliedInferences,
				tc.maxRegretsByOneInForecaster,
				tc.epsilon,
				tc.pInferenceSynthesis,
			)

			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {
				s.Require().NoError(err)
				s.Require().Len(oneInInferences, len(tc.expectedOneInInferences), "Unexpected number of one-in inferences")

				for _, expected := range tc.expectedOneInInferences {
					found := false
					for _, actual := range oneInInferences {
						if expected.Worker == actual.Worker {
							s.Require().InEpsilon(expected.Value, actual.Value, 1e-5, "Mismatch in value for one-in inference of worker %s", expected.Worker)
							found = true
							break
						}
					}
					if !found {
						s.FailNow("Matching worker not found", "Worker %s not found in actual inferences", expected.Worker)
					}
				}
			}
		})
	}
}
