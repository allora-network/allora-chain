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
		infererNetworkRegrets                 map[string]inference_synthesis.Regret
		expectedErr                           error
	}{
		{ // EPOCH 3
			name: "normal operation 1",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("-0.0514234892489971")},
				"worker1": {Value: alloraMath.MustNewDecFromString("-0.0316532211989242")},
				"worker2": {Value: alloraMath.MustNewDecFromString("-0.1018014248041400")},
			},
			forecastImpliedInferenceByWorker: map[string]*emissions.Inference{
				"worker3": {Value: alloraMath.MustNewDecFromString("-0.0707517711518230")},
				"worker4": {Value: alloraMath.MustNewDecFromString("-0.0646463841210426")},
				"worker5": {Value: alloraMath.MustNewDecFromString("-0.0634099113416666")},
			},
			maxRegret:           alloraMath.MustNewDecFromString("0.9871536722074480"),
			epsilon:             alloraMath.MustNewDecFromString("0.0001"),
			pInferenceSynthesis: alloraMath.MustNewDecFromString("2"),
			infererNetworkRegrets: map[string]inference_synthesis.Regret{
				"worker0": alloraMath.MustNewDecFromString("0.6975029322458370"),
				"worker1": alloraMath.MustNewDecFromString("0.910174442412618"),
				"worker2": alloraMath.MustNewDecFromString("0.9871536722074480"),
				"worker3": alloraMath.MustNewDecFromString("0.8308330665491310"),
				"worker4": alloraMath.MustNewDecFromString("0.8396961220162480"),
				"worker5": alloraMath.MustNewDecFromString("0.8017696138115460"),
			},
			expectedNetworkCombinedInferenceValue: alloraMath.MustNewDecFromString("-0.06470631905627390"),
			expectedErr:                           nil,
		},
		{ // EPOCH 4
			name: "normal operation 2",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("-0.14361768314408600")},
				"worker1": {Value: alloraMath.MustNewDecFromString("-0.23422685055675900")},
				"worker2": {Value: alloraMath.MustNewDecFromString("-0.18201270373970600")},
			},
			forecastImpliedInferenceByWorker: map[string]*emissions.Inference{
				"worker3": {Value: alloraMath.MustNewDecFromString("-0.19840891048468800")},
				"worker4": {Value: alloraMath.MustNewDecFromString("-0.19696044261177800")},
				"worker5": {Value: alloraMath.MustNewDecFromString("-0.20289734770434400")},
			},
			maxRegret:           alloraMath.MustNewDecFromString("0.9737035757621540"),
			epsilon:             alloraMath.MustNewDecFromString("0.0001"),
			pInferenceSynthesis: alloraMath.NewDecFromInt64(2),
			infererNetworkRegrets: map[string]inference_synthesis.Regret{
				"worker0": alloraMath.MustNewDecFromString("0.5576393860961080"),
				"worker1": alloraMath.MustNewDecFromString("0.8588215562008240"),
				"worker2": alloraMath.MustNewDecFromString("0.9737035757621540"),
				"worker3": alloraMath.MustNewDecFromString("0.7535724745797420"),
				"worker4": alloraMath.MustNewDecFromString("0.7658774622830770"),
				"worker5": alloraMath.MustNewDecFromString("0.7185104293863190"),
			},
			expectedNetworkCombinedInferenceValue: alloraMath.MustNewDecFromString("-0.19466636004515200"),
			expectedErr:                           nil,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			for inferer, regret := range tc.infererNetworkRegrets {
				s.emissionsKeeper.SetInfererNetworkRegret(
					s.ctx,
					topicId,
					[]byte(inferer),
					emissions.TimestampedValue{BlockHeight: 0, Value: regret},
				)
			}

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
					tc.expectedNetworkCombinedInferenceValue.String(),
					networkCombinedInferenceValue.String(),
				)
			}
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcOneOutInferences() {
	topicId := inference_synthesis.TopicId(1)

	test := struct {
		name                            string
		inferenceByWorker               map[string]*emissions.Inference
		forecasts                       *emissions.Forecasts
		maxRegret                       inference_synthesis.Regret
		networkCombinedLoss             inference_synthesis.Loss
		epsilon                         alloraMath.Dec
		pInferenceSynthesis             alloraMath.Dec
		infererNetworkRegrets           map[string]inference_synthesis.Regret
		expectedOneOutInferences        []*emissions.WithheldWorkerAttributedValue
		expectedOneOutImpliedInferences []*emissions.WithheldWorkerAttributedValue
	}{ // EPOCH 3
		name: "basic functionality, multiple workers",
		inferenceByWorker: map[string]*emissions.Inference{
			"worker0": {Value: alloraMath.MustNewDecFromString("-0.0514234892489971")},
			"worker1": {Value: alloraMath.MustNewDecFromString("-0.0316532211989242")},
			"worker2": {Value: alloraMath.MustNewDecFromString("-0.1018014248041400")},
		},
		forecasts: &emissions.Forecasts{
			Forecasts: []*emissions.Forecast{
				{
					Forecaster: "worker3",
					ForecastElements: []*emissions.ForecastElement{
						{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.00011708024633613200")},
						{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.013382222402411400")},
						{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("3.82471429104471e-05")},
					},
				},
				{
					Forecaster: "worker4",
					ForecastElements: []*emissions.ForecastElement{
						{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.00011486217283808300")},
						{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0060528036329761000")},
						{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.0005337255825785730")},
					},
				},
				{
					Forecaster: "worker5",
					ForecastElements: []*emissions.ForecastElement{
						{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.001810780808278390")},
						{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0018544539679880700")},
						{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.001251454152216520")},
					},
				},
			},
		},
		infererNetworkRegrets: map[string]inference_synthesis.Regret{
			"worker0": alloraMath.MustNewDecFromString("0.6975029322458370"),
			"worker1": alloraMath.MustNewDecFromString("0.9101744424126180"),
			"worker2": alloraMath.MustNewDecFromString("0.9871536722074480"),
			"worker3": alloraMath.MustNewDecFromString("0.8308330665491310"),
			"worker4": alloraMath.MustNewDecFromString("0.8396961220162480"),
			"worker5": alloraMath.MustNewDecFromString("0.8017696138115460"),
		},
		maxRegret:           alloraMath.MustNewDecFromString("0.987153672207448"),
		networkCombinedLoss: alloraMath.MustNewDecFromString("0.0156937658327922"),
		epsilon:             alloraMath.MustNewDecFromString("0.0001"),
		pInferenceSynthesis: alloraMath.MustNewDecFromString("2.0"),
		expectedOneOutInferences: []*emissions.WithheldWorkerAttributedValue{
			{Worker: "worker0", Value: alloraMath.MustNewDecFromString("-0.07291976702609980")},
			{Worker: "worker1", Value: alloraMath.MustNewDecFromString("-0.07795430811050460")},
			{Worker: "worker2", Value: alloraMath.MustNewDecFromString("-0.042814093565554400")},
		},
		expectedOneOutImpliedInferences: []*emissions.WithheldWorkerAttributedValue{
			{Worker: "worker3", Value: alloraMath.MustNewDecFromString("-0.0635171449618356")},
			{Worker: "worker4", Value: alloraMath.MustNewDecFromString("-0.06471822091625930")},
			{Worker: "worker5", Value: alloraMath.MustNewDecFromString("-0.0649534852873976")},
		},
	}

	s.Run(test.name, func() {
		for inferer, regret := range test.infererNetworkRegrets {
			s.emissionsKeeper.SetInfererNetworkRegret(
				s.ctx,
				topicId,
				[]byte(inferer),
				emissions.TimestampedValue{BlockHeight: 0, Value: regret},
			)
		}

		oneOutInferences, oneOutImpliedInferences, err := inference_synthesis.CalcOneOutInferences(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			test.inferenceByWorker,
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
			s.Require().Equal(expected.Worker, oneOutInferences[i].Worker, "Mismatch in worker for one-out inference")
			s.Require().True(
				alloraMath.InDelta(
					expected.Value,
					oneOutInferences[i].Value,
					alloraMath.MustNewDecFromString("0.0001"),
				), "Mismatch in value for one-out inference of worker %s", expected.Worker)
		}

		for i, expected := range test.expectedOneOutImpliedInferences {
			s.Require().Equal(expected.Worker, oneOutImpliedInferences[i].Worker, "Mismatch in worker for one-out implied inference")
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
				"worker2": {Value: alloraMath.MustNewDecFromString("0.09269114998018040")},
			},
			forecastImpliedInferences: map[string]*emissions.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("0.08584946856167300")},
				"worker1": {Value: alloraMath.MustNewDecFromString("0.08215179314806270")},
				"worker2": {Value: alloraMath.MustNewDecFromString("0.0891905081396791")},
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
				{Worker: "worker2", Value: alloraMath.MustNewDecFromString("0.07705216278952520")},
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
