package inference_synthesis_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
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
		epsilon                               alloraMath.Dec
		pInferenceSynthesis                   alloraMath.Dec
		expectedNetworkCombinedInferenceValue alloraMath.Dec
		expectedErr                           error
	}{
		{ // ROW 3
			name: "normal operation",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("-0.05142348924899710")},
				"worker1": {Value: alloraMath.MustNewDecFromString("-0.03165322119892420")},
			},
			forecastImpliedInferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("-0.07075177115182300")},
				"worker1": {Value: alloraMath.MustNewDecFromString("-0.06464638412104260")},
			},
			maxRegret:                             alloraMath.MustNewDecFromString("0.5"),
			epsilon:                               alloraMath.MustNewDecFromString("0.0001"),
			pInferenceSynthesis:                   alloraMath.MustNewDecFromString("2"),
			expectedNetworkCombinedInferenceValue: alloraMath.MustNewDecFromString("-0.06470631905627390"),
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

				s.Require().True(
					alloraMath.InDelta(
						tc.expectedNetworkCombinedInferenceValue,
						networkCombinedInferenceValue,
						alloraMath.MustNewDecFromString("0.00001"),
					),
					"Network combined inference value should match expected value within epsilon",
				)
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
		epsilon                          alloraMath.Dec
		pInferenceSynthesis              alloraMath.Dec
		expectedOneOutInferences         []*emissions.WithheldWorkerAttributedValue
		expectedOneOutImpliedInferences  []*emissions.WithheldWorkerAttributedValue
	}{ // ROW 5
		name: "basic functionality, multiple workers",
		inferenceByWorker: map[string]*emissions.Inference{
			"worker0": {Value: alloraMath.MustNewDecFromString("0.09688553736890290")},
			"worker1": {Value: alloraMath.MustNewDecFromString("0.15603487178220000")},
		},
		forecastImpliedInferenceByWorker: map[string]*emissions.Inference{
			"worker0": {Value: alloraMath.MustNewDecFromString("0.09590746110637150")},
			"worker1": {Value: alloraMath.MustNewDecFromString("0.09199706634747750")},
		},
		forecasts: &emissions.Forecasts{
			Forecasts: []*emissions.Forecast{
				{
					Forecaster: "forecaster0",
					ForecastElements: []*emissions.ForecastElement{
						{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("9.65209481504552e-06")},
						{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0013204058258572500")},
					},
				},
				{
					Forecaster: "forecaster1",
					ForecastElements: []*emissions.ForecastElement{
						{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("1.57700563929882e-05")},
						{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.002446373314877150")},
					},
				},
			},
		},
		maxRegret:           alloraMath.MustNewDecFromString("0.5"),
		networkCombinedLoss: alloraMath.MustNewDecFromString("10.0"),
		epsilon:             alloraMath.MustNewDecFromString("0.0001"),
		expectedOneOutInferences: []*emissions.WithheldWorkerAttributedValue{
			{Worker: "worker0", Value: alloraMath.MustNewDecFromString("0.07868265511452390")},
			{Worker: "worker1", Value: alloraMath.MustNewDecFromString("0.05882929409106640")},
		},
		expectedOneOutImpliedInferences: []*emissions.WithheldWorkerAttributedValue{
			{Worker: "worker0", Value: alloraMath.MustNewDecFromString("2.166666666666666")},
			{Worker: "worker1", Value: alloraMath.MustNewDecFromString("1.833333333333333")},
		},
		pInferenceSynthesis: alloraMath.MustNewDecFromString("2.0"),
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
			s.Require().True(
				alloraMath.InDelta(
					expected.Value,
					oneOutInferences[i].Value,
					alloraMath.MustNewDecFromString("0.00001"),
				), "Mismatch in value for one-out inference of worker %s", expected.Worker)
		}

		for i, expected := range test.expectedOneOutImpliedInferences {
			s.Require().True(
				alloraMath.InDelta(
					expected.Value,
					oneOutImpliedInferences[i].Value,
					alloraMath.MustNewDecFromString("0.00001"),
				), "Mismatch in value for one-out implied inference of worker %s", expected.Worker)
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
		epsilon                     alloraMath.Dec
		pInferenceSynthesis         alloraMath.Dec
		expectedOneInInferences     []*emissions.WorkerAttributedValue
		expectedErr                 error
	}{
		{ // ROW 6
			name: "basic functionality, single worker",
			inferences: map[string]*emissions.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("0.10711562728325500")},
				"worker1": {Value: alloraMath.MustNewDecFromString("0.03008145586124120")},
			},
			forecastImpliedInferences: map[string]*emissions.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("0.08584946856167300")},
				"worker1": {Value: alloraMath.MustNewDecFromString("0.08215179314806270")},
			},
			maxRegretsByOneInForecaster: map[string]inference_synthesis.Regret{
				"worker0": alloraMath.MustNewDecFromString("0.1"),
				"worker1": alloraMath.MustNewDecFromString("0.2"),
			},
			epsilon:             alloraMath.MustNewDecFromString("0.0001"),
			pInferenceSynthesis: alloraMath.MustNewDecFromString("2.0"),
			expectedOneInInferences: []*emissions.WorkerAttributedValue{
				{Worker: "worker0", Value: alloraMath.MustNewDecFromString("0.0764686352947760")},
				{Worker: "worker1", Value: alloraMath.MustNewDecFromString("0.0755370605649977")},
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
							s.Require().True(
								alloraMath.InDelta(
									expected.Value,
									actual.Value,
									alloraMath.MustNewDecFromString("0.00001"),
								),
								"Mismatch in value for one-in inference of worker %s",
								expected.Worker,
							)
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
