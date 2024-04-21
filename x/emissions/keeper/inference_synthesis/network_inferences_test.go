package inference_synthesis_test

import (
	"reflect"
	"testing"

	"github.com/allora-network/allora-chain/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"

	inference_synthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/types"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

// instantiate a AllWorkersAreNew struct
func NewWorkersAreNew(v bool) inference_synthesis.AllWorkersAreNew {
	return inference_synthesis.AllWorkersAreNew{
		AllInferersAreNew:    v,
		AllForecastersAreNew: v,
	}
}

// TestMakeMapFromWorkerToTheirWork tests the makeMapFromWorkerToTheirWork function for correctly mapping workers to their inferences.
func TestMakeMapFromWorkerToTheirWork(t *testing.T) {
	tests := []struct {
		name       string
		inferences []*emissions.Inference
		expected   map[string]*emissions.Inference
	}{
		{
			name: "multiple workers",
			inferences: []*emissions.Inference{
				{
					TopicId: 101,
					Inferer: "worker1",
					Value:   alloraMath.MustNewDecFromString("10"),
				},
				{
					TopicId: 102,
					Inferer: "worker2",
					Value:   alloraMath.MustNewDecFromString("20"),
				},
				{
					TopicId: 103,
					Inferer: "worker3",
					Value:   alloraMath.MustNewDecFromString("30"),
				},
			},
			expected: map[string]*emissions.Inference{
				"worker1": {
					TopicId: 101,
					Inferer: "worker1",
					Value:   alloraMath.MustNewDecFromString("10"),
				},
				"worker2": {
					TopicId: 102,
					Inferer: "worker2",
					Value:   alloraMath.MustNewDecFromString("20"),
				},
				"worker3": {
					TopicId: 103,
					Inferer: "worker3",
					Value:   alloraMath.MustNewDecFromString("30"),
				},
			},
		},
		{
			name:       "empty list",
			inferences: []*emissions.Inference{},
			expected:   map[string]*emissions.Inference{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := inference_synthesis.MakeMapFromWorkerToTheirWork(tc.inferences)
			assert.True(t, reflect.DeepEqual(result, tc.expected), "Expected and actual maps should be equal")
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestFindMaxRegretAmongWorkersWithLosses() {
	k := s.emissionsKeeper
	topicId := uint64(1)

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"
	worker4 := "worker4"
	worker1Address := sdk.AccAddress(worker1)
	worker2Address := sdk.AccAddress(worker2)
	worker3Address := sdk.AccAddress(worker3)
	worker4Address := sdk.AccAddress(worker4)

	inferenceByWorker := map[string]*types.Inference{
		worker1: {Value: math.MustNewDecFromString("0.5")},
		worker2: {Value: math.MustNewDecFromString("0.7")},
	}

	forecastImpliedInferenceByWorker := map[string]*types.Inference{
		worker3: {Value: math.MustNewDecFromString("0.6")},
		worker4: {Value: math.MustNewDecFromString("0.8")},
	}

	epsilon := math.MustNewDecFromString("0.001")

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(s.ctx, topicId, worker1Address, types.TimestampedValue{Value: math.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(s.ctx, topicId, worker2Address, types.TimestampedValue{Value: math.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(s.ctx, topicId, worker3Address, types.TimestampedValue{Value: math.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(s.ctx, topicId, worker4Address, types.TimestampedValue{Value: math.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker3Address, worker1Address, types.TimestampedValue{Value: math.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker3Address, worker2Address, types.TimestampedValue{Value: math.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker3Address, worker3Address, types.TimestampedValue{Value: math.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker4Address, worker1Address, types.TimestampedValue{Value: math.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker4Address, worker2Address, types.TimestampedValue{Value: math.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker4Address, worker4Address, types.TimestampedValue{Value: math.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	maxRegrets, err := inference_synthesis.FindMaxRegretAmongWorkersWithLosses(s.ctx, k, topicId, inferenceByWorker, forecastImpliedInferenceByWorker, epsilon)
	s.Require().NoError(err)

	expectedMaxInfererRegret := math.MustNewDecFromString("0.3")
	expectedMaxForecasterRegret := math.MustNewDecFromString("0.5")

	s.Require().True(maxRegrets.MaxInferenceRegret.Equal(expectedMaxInfererRegret))
	s.Require().True(maxRegrets.MaxForecastRegret.Equal(expectedMaxForecasterRegret))

	s.Require().Equal(math.MustNewDecFromString("0.4"), maxRegrets.MaxOneInForecastRegret[worker3])
	s.Require().Equal(math.MustNewDecFromString("0.6"), maxRegrets.MaxOneInForecastRegret[worker4])
}

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
		forecasterNetworkRegrets              map[string]inference_synthesis.Regret
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
			},
			forecasterNetworkRegrets: map[string]inference_synthesis.Regret{
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
			},
			forecasterNetworkRegrets: map[string]inference_synthesis.Regret{
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

			for forecaster, regret := range tc.forecasterNetworkRegrets {
				s.emissionsKeeper.SetForecasterNetworkRegret(
					s.ctx,
					topicId,
					[]byte(forecaster),
					emissions.TimestampedValue{BlockHeight: 0, Value: regret},
				)
			}

			networkCombinedInferenceValue, err := inference_synthesis.CalcWeightedInference(
				s.ctx,
				s.emissionsKeeper,
				topicId,
				tc.inferenceByWorker,
				tc.forecastImpliedInferenceByWorker,
				NewWorkersAreNew(false),
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
		name                             string
		inferenceByWorker                map[string]*emissions.Inference
		forecastImpliedInferenceByWorker map[string]*emissions.Inference
		forecasts                        *emissions.Forecasts
		maxRegret                        inference_synthesis.Regret
		networkCombinedLoss              inference_synthesis.Loss
		epsilon                          alloraMath.Dec
		pInferenceSynthesis              alloraMath.Dec
		infererNetworkRegrets            map[string]inference_synthesis.Regret
		forecasterNetworkRegrets         map[string]inference_synthesis.Regret
		expectedOneOutInferences         []*emissions.WithheldWorkerAttributedValue
		expectedOneOutImpliedInferences  []*emissions.WithheldWorkerAttributedValue
	}{ // EPOCH 3
		name: "basic functionality, multiple workers",
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
		},
		forecasterNetworkRegrets: map[string]inference_synthesis.Regret{
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

		for forecaster, regret := range test.forecasterNetworkRegrets {
			s.emissionsKeeper.SetForecasterNetworkRegret(
				s.ctx,
				topicId,
				[]byte(forecaster),
				emissions.TimestampedValue{BlockHeight: 0, Value: regret},
			)
		}

		oneOutInferences, oneOutImpliedInferences, err := inference_synthesis.CalcOneOutInferences(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			test.inferenceByWorker,
			test.forecastImpliedInferenceByWorker,
			test.forecasts,
			NewWorkersAreNew(false),
			test.maxRegret,
			test.networkCombinedLoss,
			test.epsilon,
			test.pInferenceSynthesis,
		)

		s.Require().NoError(err, "CalcOneOutInferences should not return an error")

		s.Require().Len(oneOutInferences, len(test.expectedOneOutInferences), "Unexpected number of one-out inferences")
		s.Require().Len(oneOutImpliedInferences, len(test.expectedOneOutImpliedInferences), "Unexpected number of one-out implied inferences")

		for _, expected := range test.expectedOneOutInferences {
			found := false
			for _, oneOutInference := range oneOutInferences {
				if expected.Worker == oneOutInference.Worker {
					found = true
					s.Require().True(
						alloraMath.InDelta(
							expected.Value,
							oneOutInference.Value,
							alloraMath.MustNewDecFromString("0.0001"),
						), "Mismatch in value for one-out inference of worker %s", expected.Worker)
				}
			}
			if !found {
				s.FailNow("Matching worker not found", "Worker %s not found in returned inferences", expected.Worker)
			}
		}

		for _, expected := range test.expectedOneOutImpliedInferences {
			found := false
			for _, oneOutImpliedInference := range oneOutImpliedInferences {
				if expected.Worker == oneOutImpliedInference.Worker {
					found = true
					s.Require().True(
						alloraMath.InDelta(
							expected.Value,
							oneOutImpliedInference.Value,
							alloraMath.MustNewDecFromString("0.01"),
						), "Mismatch in value for one-out implied inference of worker %s", expected.Worker)
				}
			}
			if !found {
				s.FailNow("Matching worker not found", "Worker %s not found in returned inferences", expected.Worker)
			}
		}
	})
}

func (s *InferenceSynthesisTestSuite) TestCalcOneInInferences() {
	topicId := inference_synthesis.TopicId(1)

	tests := []struct {
		name                        string
		inferenceByWorker           map[string]*emissions.Inference
		forecastImpliedInferences   map[string]*emissions.Inference
		maxRegretsByOneInForecaster map[string]inference_synthesis.Regret
		epsilon                     alloraMath.Dec
		pInferenceSynthesis         alloraMath.Dec
		infererNetworkRegrets       map[string]inference_synthesis.Regret
		forecasterNetworkRegrets    map[string]inference_synthesis.Regret
		expectedOneInInferences     []*emissions.WorkerAttributedValue
		expectedErr                 error
	}{
		{ // EPOCH 3
			name: "basic functionality",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("-0.0514234892489971")},
				"worker1": {Value: alloraMath.MustNewDecFromString("-0.0316532211989242")},
				"worker2": {Value: alloraMath.MustNewDecFromString("-0.1018014248041400")},
			},
			forecastImpliedInferences: map[string]*emissions.Inference{
				"worker3": {Value: alloraMath.MustNewDecFromString("-0.0707517711518230")},
				"worker4": {Value: alloraMath.MustNewDecFromString("-0.0646463841210426")},
				"worker5": {Value: alloraMath.MustNewDecFromString("-0.0634099113416666")},
			},
			maxRegretsByOneInForecaster: map[string]inference_synthesis.Regret{
				"worker3": alloraMath.MustNewDecFromString("0.9871536722074480"),
				"worker4": alloraMath.MustNewDecFromString("0.9871536722074480"),
				"worker5": alloraMath.MustNewDecFromString("0.9871536722074480"),
			},
			epsilon:             alloraMath.MustNewDecFromString("0.0001"),
			pInferenceSynthesis: alloraMath.MustNewDecFromString("2.0"),
			infererNetworkRegrets: map[string]inference_synthesis.Regret{
				"worker0": alloraMath.MustNewDecFromString("0.6975029322458370"),
				"worker1": alloraMath.MustNewDecFromString("0.9101744424126180"),
				"worker2": alloraMath.MustNewDecFromString("0.9871536722074480"),
			},
			forecasterNetworkRegrets: map[string]inference_synthesis.Regret{
				"worker3": alloraMath.MustNewDecFromString("0.8308330665491310"),
				"worker4": alloraMath.MustNewDecFromString("0.8396961220162480"),
				"worker5": alloraMath.MustNewDecFromString("0.8017696138115460"),
			},
			expectedOneInInferences: []*emissions.WorkerAttributedValue{
				{Worker: "worker3", Value: alloraMath.MustNewDecFromString("-0.06502630286365970")},
				{Worker: "worker4", Value: alloraMath.MustNewDecFromString("-0.06356081320547800")},
				{Worker: "worker5", Value: alloraMath.MustNewDecFromString("-0.06325114823960220")},
			},
			expectedErr: nil,
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

			for forecaster, regret := range tc.forecasterNetworkRegrets {
				s.emissionsKeeper.SetForecasterNetworkRegret(
					s.ctx,
					topicId,
					[]byte(forecaster),
					emissions.TimestampedValue{BlockHeight: 0, Value: regret},
				)
			}

			oneInInferences, err := inference_synthesis.CalcOneInInferences(
				s.ctx,
				s.emissionsKeeper,
				topicId,
				tc.inferenceByWorker,
				tc.forecastImpliedInferences,
				NewWorkersAreNew(false),
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
									alloraMath.MustNewDecFromString("0.0001"),
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

func (s *InferenceSynthesisTestSuite) TestCalcNetworkInferences() {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"
	worker4 := "worker4"
	worker1Add := sdk.AccAddress(worker1)
	worker2Add := sdk.AccAddress(worker2)
	worker3Add := sdk.AccAddress(worker3)
	worker4Add := sdk.AccAddress(worker4)

	// Set up input data
	inferences := &emissions.Inferences{
		Inferences: []*emissions.Inference{
			{Inferer: worker1, Value: math.MustNewDecFromString("0.5")},
			{Inferer: worker2, Value: math.MustNewDecFromString("0.7")},
		},
	}

	forecasts := &emissions.Forecasts{
		Forecasts: []*emissions.Forecast{
			{
				Forecaster: worker3,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: math.MustNewDecFromString("0.6")},
					{Inferer: worker2, Value: math.MustNewDecFromString("0.8")},
				},
			},
			{
				Forecaster: worker4,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: math.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: math.MustNewDecFromString("0.9")},
				},
			},
		},
	}

	networkCombinedLoss := math.MustNewDecFromString("0.2")
	epsilon := math.MustNewDecFromString("0.001")
	pInferenceSynthesis := math.MustNewDecFromString("2")

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker3Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker4Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3Add, worker1Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3Add, worker2Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4Add, worker1Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4Add, worker2Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.4")})
	s.Require().NoError(err)

	// Call the function
	valueBundle, err := inference_synthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, pInferenceSynthesis)
	s.Require().NoError(err)

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)
	s.Require().NotEmpty(valueBundle.OneOutInfererValues)
	s.Require().NotEmpty(valueBundle.OneOutForecasterValues)
	s.Require().NotEmpty(valueBundle.OneInForecasterValues)
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkInferencesSameInfererForecasters() {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)
	blockHeightInferences := int64(390)
	blockHeightForecaster := int64(380)

	worker1 := "worker1"
	worker2 := "worker2"
	worker1Add := sdk.AccAddress(worker1)
	worker2Add := sdk.AccAddress(worker2)

	// Set up input data
	inferences := &emissions.Inferences{
		Inferences: []*emissions.Inference{
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Value: math.MustNewDecFromString("0.52")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Value: math.MustNewDecFromString("0.71")},
		},
	}

	forecasts := &emissions.Forecasts{
		Forecasts: []*emissions.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker1,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: math.MustNewDecFromString("0.5")},
					{Inferer: worker2, Value: math.MustNewDecFromString("0.8")},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker2,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: math.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: math.MustNewDecFromString("0.9")},
				},
			},
		},
	}

	networkCombinedLoss := math.MustNewDecFromString("1")
	epsilon := math.MustNewDecFromString("0.001")
	pInferenceSynthesis := math.MustNewDecFromString("2")

	// Call the function
	valueBundle, err := inference_synthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, pInferenceSynthesis)
	s.Require().NoError(err)
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)
	s.Require().NotEmpty(valueBundle.OneOutInfererValues)
	s.Require().NotEmpty(valueBundle.OneOutForecasterValues)
	// s.Require().NotEmpty(valueBundle.OneInForecasterValues)

	// Set inferer network regrets
	err = k.SetInfererNetworkRegret(ctx, topicId, worker1Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker1Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker1Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker1Add, worker1Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker1Add, worker2Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker2Add, worker1Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker2Add, worker2Add, types.TimestampedValue{Value: math.MustNewDecFromString("0.4")})
	s.Require().NoError(err)

	// Call the function
	valueBundle, err = inference_synthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, pInferenceSynthesis)
	s.Require().NoError(err)

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)
	s.Require().NotEmpty(valueBundle.OneOutInfererValues)
	s.Require().NotEmpty(valueBundle.OneOutForecasterValues)
	s.Require().NotEmpty(valueBundle.OneInForecasterValues)
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkInferencesIncompleteData() {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)
	blockHeightInferences := int64(390)
	blockHeightForecaster := int64(380)

	worker1 := "worker1"
	worker2 := "worker2"

	// Set up input data
	inferences := &emissions.Inferences{
		Inferences: []*emissions.Inference{
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Value: math.MustNewDecFromString("0.52")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Value: math.MustNewDecFromString("0.71")},
		},
	}

	forecasts := &emissions.Forecasts{
		Forecasts: []*emissions.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker1,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: math.MustNewDecFromString("0.5")},
					{Inferer: worker2, Value: math.MustNewDecFromString("0.8")},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker2,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: math.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: math.MustNewDecFromString("0.9")},
				},
			},
		},
	}

	networkCombinedLoss := math.MustNewDecFromString("1")
	epsilon := math.MustNewDecFromString("0.001")
	pInferenceSynthesis := math.MustNewDecFromString("2")

	// Call the function without setting regrets
	valueBundle, err := inference_synthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, pInferenceSynthesis)
	s.Require().NoError(err)
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)
	s.Require().NotEmpty(valueBundle.OneOutInfererValues)
	s.Require().NotEmpty(valueBundle.OneOutForecasterValues)
	// OneInForecastValues come empty because regrets are epsilon
	s.Require().Empty(valueBundle.OneInForecasterValues)
}
