package inference_synthesis_test

import (
	"reflect"
	"testing"

	cosmosMath "cosmossdk.io/math"
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

	inferenceByWorker := map[string]*emissions.Inference{
		worker1: {Value: alloraMath.MustNewDecFromString("0.5")},
		worker2: {Value: alloraMath.MustNewDecFromString("0.7")},
	}

	forecastImpliedInferenceByWorker := map[string]*emissions.Inference{
		worker3: {Value: alloraMath.MustNewDecFromString("0.6")},
		worker4: {Value: alloraMath.MustNewDecFromString("0.8")},
	}

	epsilon := alloraMath.MustNewDecFromString("0.001")

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(s.ctx, topicId, worker1Address, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(s.ctx, topicId, worker2Address, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(s.ctx, topicId, worker3Address, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(s.ctx, topicId, worker4Address, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker3Address, worker1Address, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker3Address, worker2Address, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker3Address, worker3Address, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker4Address, worker1Address, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker4Address, worker2Address, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(s.ctx, topicId, worker4Address, worker4Address, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	maxRegrets, err := inference_synthesis.FindMaxRegretAmongWorkersWithLosses(
		s.ctx,
		k,
		topicId,
		inferenceByWorker,
		inference_synthesis.GetSortedStringKeys(inferenceByWorker),
		forecastImpliedInferenceByWorker,
		inference_synthesis.GetSortedStringKeys(forecastImpliedInferenceByWorker),
		epsilon,
	)
	s.Require().NoError(err)

	expectedMaxInfererRegret := alloraMath.MustNewDecFromString("0.3")
	expectedMaxForecasterRegret := alloraMath.MustNewDecFromString("0.5")

	s.Require().True(maxRegrets.MaxInferenceRegret.Equal(expectedMaxInfererRegret))
	s.Require().True(maxRegrets.MaxForecastRegret.Equal(expectedMaxForecasterRegret))

	s.Require().Equal(alloraMath.MustNewDecFromString("0.4"), maxRegrets.MaxOneInForecastRegret[worker3])
	s.Require().Equal(alloraMath.MustNewDecFromString("0.6"), maxRegrets.MaxOneInForecastRegret[worker4])
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
				inference_synthesis.GetSortedStringKeys(tc.inferenceByWorker),
				tc.forecastImpliedInferenceByWorker,
				inference_synthesis.GetSortedStringKeys(tc.forecastImpliedInferenceByWorker),
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

	tests := []struct {
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
		expectedOneOutInferences         []struct {
			Worker string
			Value  string
		}
		expectedOneOutImpliedInferences []struct {
			Worker string
			Value  string
		}
	}{
		{
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
			expectedOneOutInferences: []struct {
				Worker string
				Value  string
			}{
				{Worker: "worker0", Value: "-0.07291976702609980"},
				{Worker: "worker1", Value: "-0.07795430811050460"},
				{Worker: "worker2", Value: "-0.042814093565554400"},
			},
			expectedOneOutImpliedInferences: []struct {
				Worker string
				Value  string
			}{
				{Worker: "worker3", Value: "-0.0635171449618356"},
				{Worker: "worker4", Value: "-0.06471822091625930"},
				{Worker: "worker5", Value: "-0.0649534852873976"},
			},
		},
		{
			name: "basic functionality 2, 5 workers, 3 forecasters",
			inferenceByWorker: map[string]*emissions.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("-0.035995138925040600")},
				"worker1": {Value: alloraMath.MustNewDecFromString("-0.07333303938740420")},
				"worker2": {Value: alloraMath.MustNewDecFromString("-0.1495482917094790")},
				"worker3": {Value: alloraMath.MustNewDecFromString("-0.12952123274063800")},
				"worker4": {Value: alloraMath.MustNewDecFromString("-0.0703055329498285")},
			},
			forecastImpliedInferenceByWorker: map[string]*emissions.Inference{
				"forecaster0": {Value: alloraMath.MustNewDecFromString("-0.08944493117005920")},
				"forecaster1": {Value: alloraMath.MustNewDecFromString("-0.07333218290300560")},
				"forecaster2": {Value: alloraMath.MustNewDecFromString("-0.07756206109376570")},
			},
			// epoch 3
			forecasts: &emissions.Forecasts{
				Forecasts: []*emissions.Forecast{
					{
						Forecaster: "forecaster0",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.003305466418410120")},
							{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0002788248228566030")},
							{Inferer: "worker2", Value: alloraMath.MustNewDecFromString(".0000240536828602367")},
							{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("0.0008240378476798250")},
							{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("0.0000186192181193532")},
						},
					},
					{
						Forecaster: "forecaster1",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.002308441286328890")},
							{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0000214380788596749")},
							{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.012560171044167200")},
							{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("0.017998563880697900")},
							{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("0.00020024906252089700")},
						},
					},
					{
						Forecaster: "forecaster2",
						ForecastElements: []*emissions.ForecastElement{
							{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.005369218152594270")},
							{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0002578158768320300")},
							{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.0076008583603885900")},
							{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("0.0076269073955871000")},
							{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("0.00035670236460009500")},
						},
					},
				},
			},
			// epoch 2
			infererNetworkRegrets: map[string]inference_synthesis.Regret{
				"worker0": alloraMath.MustNewDecFromString("0.29240710390153500"),
				"worker1": alloraMath.MustNewDecFromString("0.4182220944854450"),
				"worker2": alloraMath.MustNewDecFromString("0.17663501719135000"),
				"worker3": alloraMath.MustNewDecFromString("0.49617463489106400"),
				"worker4": alloraMath.MustNewDecFromString("0.27996060999688600"),
			},
			forecasterNetworkRegrets: map[string]inference_synthesis.Regret{
				"forecaster0": alloraMath.MustNewDecFromString("0.816066375505268"),
				"forecaster1": alloraMath.MustNewDecFromString("0.8234558901838660"),
				"forecaster2": alloraMath.MustNewDecFromString("0.8196673550408280"),
			},
			maxRegret: alloraMath.MustNewDecFromString("0.8234558901838660"),
			// epoch 2
			networkCombinedLoss: alloraMath.MustNewDecFromString(".0000127791308799785"),
			epsilon:             alloraMath.MustNewDecFromString("0.0001"),
			pInferenceSynthesis: alloraMath.MustNewDecFromString("2.0"),
			expectedOneOutInferences: []struct {
				Worker string
				Value  string
			}{
				{Worker: "worker0", Value: "-0.09154683788664610"},
				{Worker: "worker1", Value: "-0.08794790996372430"},
				{Worker: "worker2", Value: "-0.07594021292207610"},
				{Worker: "worker3", Value: "-0.0792252490898395"},
				{Worker: "worker4", Value: "-0.0993013271015888"},
			},
			expectedOneOutImpliedInferences: []struct {
				Worker string
				Value  string
			}{
				{Worker: "forecaster0", Value: "-0.08503863595710710"},
				{Worker: "forecaster1", Value: "-0.08833982502870460"},
				{Worker: "forecaster2", Value: "-0.08746610645716590"},
			},
		},
	}

	for _, test := range tests {
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

			oneOutInfererValues, oneOutForecasterValues, err := inference_synthesis.CalcOneOutInferences(
				s.ctx,
				s.emissionsKeeper,
				topicId,
				test.inferenceByWorker,
				inference_synthesis.GetSortedStringKeys[*emissions.Inference](test.inferenceByWorker),
				test.forecastImpliedInferenceByWorker,
				inference_synthesis.GetSortedStringKeys[*emissions.Inference](test.forecastImpliedInferenceByWorker),
				test.forecasts,
				NewWorkersAreNew(false),
				test.maxRegret,
				test.networkCombinedLoss,
				test.epsilon,
				test.pInferenceSynthesis,
			)

			s.Require().NoError(err, "CalcOneOutInferences should not return an error")

			s.Require().Len(oneOutInfererValues, len(test.expectedOneOutInferences), "Unexpected number of one-out inferences")
			s.Require().Len(oneOutForecasterValues, len(test.expectedOneOutImpliedInferences), "Unexpected number of one-out implied inferences")

			for _, expected := range test.expectedOneOutInferences {
				found := false
				for _, oneOutInference := range oneOutInfererValues {
					if expected.Worker == oneOutInference.Worker {
						found = true
						s.inEpsilon2(oneOutInference.Value, expected.Value)
					}
				}
				if !found {
					s.FailNow("Matching worker not found", "Worker %s not found in returned inferences", expected.Worker)
				}
			}

			for _, expected := range test.expectedOneOutImpliedInferences {
				found := false
				for _, oneOutImpliedInference := range oneOutForecasterValues {
					if expected.Worker == oneOutImpliedInference.Worker {
						found = true
						s.inEpsilon3(oneOutImpliedInference.Value, expected.Value)
					}
				}
				if !found {
					s.FailNow("Matching worker not found", "Worker %s not found in returned inferences", expected.Worker)
				}
			}
		})
	}
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
				inference_synthesis.GetSortedStringKeys(tc.inferenceByWorker),
				tc.forecastImpliedInferences,
				inference_synthesis.GetSortedStringKeys(tc.forecastImpliedInferences),
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
			{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
			{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.7")},
		},
	}

	forecasts := &emissions.Forecasts{
		Forecasts: []*emissions.Forecast{
			{
				Forecaster: worker3,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.6")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
				},
			},
			{
				Forecaster: worker4,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("0.2")
	epsilon := alloraMath.MustNewDecFromString("0.001")
	pInferenceSynthesis := alloraMath.MustNewDecFromString("2")

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker3Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker4Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3Add, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3Add, worker2Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4Add, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4Add, worker2Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
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
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.52")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.71")},
		},
	}

	forecasts := &emissions.Forecasts{
		Forecasts: []*emissions.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker1,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker2,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("1")
	epsilon := alloraMath.MustNewDecFromString("0.001")
	pInferenceSynthesis := alloraMath.MustNewDecFromString("2")

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
	err = k.SetInfererNetworkRegret(ctx, topicId, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker1Add, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker1Add, worker2Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker2Add, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker2Add, worker2Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
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
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.52")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.71")},
		},
	}

	forecasts := &emissions.Forecasts{
		Forecasts: []*emissions.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker1,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker2,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("1")
	epsilon := alloraMath.MustNewDecFromString("0.001")
	pInferenceSynthesis := alloraMath.MustNewDecFromString("2")

	// Call the function without setting regrets
	valueBundle, err := inference_synthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, pInferenceSynthesis)
	s.Require().NoError(err)
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)
	s.Require().NotEmpty(valueBundle.OneOutInfererValues)
	s.Require().NotEmpty(valueBundle.OneOutForecasterValues)
	// OneInForecastValues come empty because regrets are epsilon
	s.Require().NotEmpty(valueBundle.OneInForecasterValues)
	s.Require().Len(valueBundle.OneInForecasterValues, 2)
}

func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesAtBlock() {
	require := s.Require()
	keeper := s.emissionsKeeper

	topicId := uint64(1)
	blockHeight := int64(3)
	require.True(blockHeight >= s.ctx.BlockHeight())
	s.ctx = s.ctx.WithBlockHeight(blockHeight)

	simpleNonce := types.Nonce{BlockHeight: blockHeight}
	reputerRequestNonce := &types.ReputerRequestNonce{
		ReputerNonce: &types.Nonce{BlockHeight: blockHeight},
	}

	reputer0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
	reputer1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
	reputer2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
	reputer3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
	reputer4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"

	reputer0Acc := sdk.AccAddress(reputer0)
	reputer1Acc := sdk.AccAddress(reputer1)
	reputer2Acc := sdk.AccAddress(reputer2)
	reputer3Acc := sdk.AccAddress(reputer3)
	reputer4Acc := sdk.AccAddress(reputer4)

	forecaster0 := "allo1pluvmvsmvecg2ccuqxa6ugzvc3a5udfyy0t76v"
	forecaster1 := "allo1e92saykj94jw3z55g4d3lfz098ppk0suwzc03a"
	forecaster2 := "allo1pk6mxny5p79t8zhkm23z7u3zmfuz2gn0snxkkt"

	forecaster0Acc := sdk.AccAddress(forecaster0)
	forecaster1Acc := sdk.AccAddress(forecaster1)
	forecaster2Acc := sdk.AccAddress(forecaster2)

	// Set Loss bundles

	reputerLossBundles := types.ReputerValueBundles{
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				ValueBundle: &types.ValueBundle{
					Reputer:             reputer0,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000117005278862668"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &types.ValueBundle{
					Reputer:             reputer1,
					CombinedValue:       alloraMath.MustNewDecFromString(".00000962701954026944"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &types.ValueBundle{
					Reputer:             reputer2,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000256948644008351"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &types.ValueBundle{
					Reputer:             reputer3,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000123986052417188"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &types.ValueBundle{
					Reputer:             reputer4,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000115363240547692"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
		},
	}

	err := keeper.InsertReputerLossBundlesAtBlock(s.ctx, topicId, blockHeight, reputerLossBundles)
	require.NoError(err)

	// Set Stake

	err = keeper.AddStake(s.ctx, topicId, reputer0Acc, cosmosMath.NewUintFromString("210535101370326000000000"))
	require.NoError(err)
	err = keeper.AddStake(s.ctx, topicId, reputer1Acc, cosmosMath.NewUintFromString("216697093951021000000000"))
	require.NoError(err)
	err = keeper.AddStake(s.ctx, topicId, reputer2Acc, cosmosMath.NewUintFromString("161740241803855000000000"))
	require.NoError(err)
	err = keeper.AddStake(s.ctx, topicId, reputer3Acc, cosmosMath.NewUintFromString("394848305052250000000000"))
	require.NoError(err)
	err = keeper.AddStake(s.ctx, topicId, reputer4Acc, cosmosMath.NewUintFromString("206169717590569000000000"))
	require.NoError(err)

	// Set Inferences

	inferences := types.Inferences{
		Inferences: []*types.Inference{
			{
				Inferer:     reputer0,
				Value:       alloraMath.MustNewDecFromString("-0.035995138925040600"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer1,
				Value:       alloraMath.MustNewDecFromString("-0.07333303938740420"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer2,
				Value:       alloraMath.MustNewDecFromString("-0.1495482917094790"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer3,
				Value:       alloraMath.MustNewDecFromString("-0.12952123274063800"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Inferer:     reputer4,
				Value:       alloraMath.MustNewDecFromString("-0.0703055329498285"),
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
		},
	}

	err = keeper.InsertInferences(s.ctx, topicId, simpleNonce, inferences)
	s.Require().NoError(err)

	// Set Forecasts

	forecasts := types.Forecasts{
		Forecasts: []*types.Forecast{
			{
				Forecaster: forecaster0,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: reputer0,
						Value:   alloraMath.MustNewDecFromString("0.003305466418410120"),
					},
					{
						Inferer: reputer1,
						Value:   alloraMath.MustNewDecFromString("0.0002788248228566030"),
					},
					{
						Inferer: reputer2,
						Value:   alloraMath.MustNewDecFromString(".0000240536828602367"),
					},
					{
						Inferer: reputer3,
						Value:   alloraMath.MustNewDecFromString("0.0008240378476798250"),
					},
					{
						Inferer: reputer4,
						Value:   alloraMath.MustNewDecFromString(".0000186192181193532"),
					},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Forecaster: forecaster1,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: reputer0,
						Value:   alloraMath.MustNewDecFromString("0.002308441286328890"),
					},
					{
						Inferer: reputer1,
						Value:   alloraMath.MustNewDecFromString(".0000214380788596749"),
					},
					{
						Inferer: reputer2,
						Value:   alloraMath.MustNewDecFromString("0.012560171044167200"),
					},
					{
						Inferer: reputer3,
						Value:   alloraMath.MustNewDecFromString("0.017998563880697900"),
					},
					{
						Inferer: reputer4,
						Value:   alloraMath.MustNewDecFromString("0.00020024906252089700"),
					},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
			{
				Forecaster: forecaster2,
				ForecastElements: []*types.ForecastElement{
					{
						Inferer: reputer0,
						Value:   alloraMath.MustNewDecFromString("0.005369218152594270"),
					},
					{
						Inferer: reputer1,
						Value:   alloraMath.MustNewDecFromString("0.0002578158768320300"),
					},
					{
						Inferer: reputer2,
						Value:   alloraMath.MustNewDecFromString("0.0076008583603885900"),
					},
					{
						Inferer: reputer3,
						Value:   alloraMath.MustNewDecFromString("0.0076269073955871000"),
					},
					{
						Inferer: reputer4,
						Value:   alloraMath.MustNewDecFromString("0.00035670236460009500"),
					},
				},
				TopicId:     topicId,
				BlockHeight: blockHeight,
			},
		},
	}

	err = keeper.InsertForecasts(s.ctx, topicId, simpleNonce, forecasts)

	// Set inferer network regrets

	infererNetworkRegrets := map[string]inference_synthesis.Regret{
		reputer0: alloraMath.MustNewDecFromString("0.29240710390153500"),
		reputer1: alloraMath.MustNewDecFromString("0.4182220944854450"),
		reputer2: alloraMath.MustNewDecFromString("0.17663501719135000"),
		reputer3: alloraMath.MustNewDecFromString("0.49617463489106400"),
		reputer4: alloraMath.MustNewDecFromString("0.27996060999688600"),
	}

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			[]byte(inferer),
			emissions.TimestampedValue{BlockHeight: blockHeight, Value: regret},
		)
	}

	// set forecaster network regrets

	forecasterNetworkRegrets := map[string]inference_synthesis.Regret{
		forecaster0: alloraMath.MustNewDecFromString("0.816066375505268"),
		forecaster1: alloraMath.MustNewDecFromString("0.8234558901838660"),
		forecaster2: alloraMath.MustNewDecFromString("0.8196673550408280"),
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			[]byte(forecaster),
			emissions.TimestampedValue{BlockHeight: blockHeight, Value: regret},
		)
	}

	// Set one in forecaster network regrets

	setOneInForecasterNetworkRegret := func(forecasterAcc sdk.AccAddress, infererAcc sdk.AccAddress, value string) {
		keeper.SetOneInForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecasterAcc,
			infererAcc,
			emissions.TimestampedValue{
				BlockHeight: blockHeight,
				Value:       alloraMath.MustNewDecFromString(value),
			},
		)
	}

	/// Epoch 3 values

	setOneInForecasterNetworkRegret(forecaster0Acc, reputer0Acc, "-0.005488956369080480")
	setOneInForecasterNetworkRegret(forecaster0Acc, reputer1Acc, "0.17091263821766800")
	setOneInForecasterNetworkRegret(forecaster0Acc, reputer2Acc, "-0.15988639638192800")
	setOneInForecasterNetworkRegret(forecaster0Acc, reputer3Acc, "0.28690775330189800")
	setOneInForecasterNetworkRegret(forecaster0Acc, reputer4Acc, "-0.019476319822263300")

	setOneInForecasterNetworkRegret(forecaster0Acc, forecaster0Acc, "0.7370268872154170")

	setOneInForecasterNetworkRegret(forecaster1Acc, reputer0Acc, "-0.023601485104528100")
	setOneInForecasterNetworkRegret(forecaster1Acc, reputer1Acc, "0.1528001094822210")
	setOneInForecasterNetworkRegret(forecaster1Acc, reputer2Acc, "-0.1779989251173760")
	setOneInForecasterNetworkRegret(forecaster1Acc, reputer3Acc, "0.2687952245664510")
	setOneInForecasterNetworkRegret(forecaster1Acc, reputer4Acc, "-0.03758884855771100")

	setOneInForecasterNetworkRegret(forecaster1Acc, forecaster1Acc, "0.7307121775422120")

	setOneInForecasterNetworkRegret(forecaster2Acc, reputer0Acc, "-0.025084585281804600")
	setOneInForecasterNetworkRegret(forecaster2Acc, reputer1Acc, "0.15131700930494400")
	setOneInForecasterNetworkRegret(forecaster2Acc, reputer2Acc, "-0.17948202529465200")
	setOneInForecasterNetworkRegret(forecaster2Acc, reputer3Acc, "0.26731212438917400")
	setOneInForecasterNetworkRegret(forecaster2Acc, reputer4Acc, "-0.03907194873498750")

	setOneInForecasterNetworkRegret(forecaster2Acc, forecaster2Acc, "0.722844746771044")

	// Calculate

	valueBundle, err :=
		inference_synthesis.GetNetworkInferencesAtBlock(
			s.ctx,
			s.emissionsKeeper,
			topicId,
			blockHeight,
			blockHeight,
		)
	require.NoError(err)

	s.inEpsilon5(valueBundle.CombinedValue, "-0.08578420625884590")
	s.inEpsilon3(valueBundle.NaiveValue, "-0.09179326141859620")

	for _, inference := range inferences.Inferences {
		found := false
		for _, infererValue := range valueBundle.InfererValues {
			if string(inference.Inferer) == infererValue.Worker {
				found = true
				require.Equal(inference.Value, infererValue.Value)
			}
		}
		require.True(found, "Inference not found")
	}

	for _, forecasterValue := range valueBundle.ForecasterValues {
		switch string(forecasterValue.Worker) {
		case forecaster0:
			s.inEpsilon2(forecasterValue.Value, "-0.08944493117005920")
		case forecaster1:
			s.inEpsilon2(forecasterValue.Value, "-0.07333218290300560")
		case forecaster2:
			s.inEpsilon2(forecasterValue.Value, "-0.07756206109376570")
		default:
			require.Fail("Unexpected forecaster %v", forecasterValue.Worker)
		}
	}

	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
		switch string(oneOutInfererValue.Worker) {
		case reputer0:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.09154683788664610")
		case reputer1:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.08794790996372430")
		case reputer2:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.07594021292207610")
		case reputer3:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.0792252490898395")
		case reputer4:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.0993013271015888")
		default:
			require.Fail("Unexpected worker %v", oneOutInfererValue.Worker)
		}
	}

	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
		switch string(oneInForecasterValue.Worker) {
		case forecaster0:
			s.inEpsilon2(oneInForecasterValue.Value, "-0.0911173989550551")
		case forecaster1:
			s.inEpsilon2(oneInForecasterValue.Value, "-0.08692482591184730")
		case forecaster2:
			s.inEpsilon2(oneInForecasterValue.Value, "-0.08802837922471950")
		default:
			require.Fail("Unexpected worker %v", oneInForecasterValue.Worker)
		}
	}

	for _, oneOutForecasterValue := range valueBundle.OneOutForecasterValues {
		switch string(oneOutForecasterValue.Worker) {
		case forecaster0:
			s.inEpsilon2(oneOutForecasterValue.Value, "-0.08503863595710710")
		case forecaster1:
			s.inEpsilon2(oneOutForecasterValue.Value, "-0.08833982502870460")
		case forecaster2:
			s.inEpsilon2(oneOutForecasterValue.Value, "-0.08746610645716590")
		default:
			require.Fail("Unexpected worker %v", oneOutForecasterValue.Worker)
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestFilterNoncesWithinEpochLength() {
	tests := []struct {
		name          string
		nonces        emissions.Nonces
		blockHeight   int64
		epochLength   int64
		expectedNonce emissions.Nonces
	}{
		{
			name: "Nonces within epoch length",
			nonces: emissions.Nonces{
				Nonces: []*emissions.Nonce{
					{BlockHeight: 10},
					{BlockHeight: 15},
				},
			},
			blockHeight: 20,
			epochLength: 10,
			expectedNonce: emissions.Nonces{
				Nonces: []*emissions.Nonce{
					{BlockHeight: 10},
					{BlockHeight: 15},
				},
			},
		},
		{
			name: "Nonces outside epoch length",
			nonces: emissions.Nonces{
				Nonces: []*emissions.Nonce{
					{BlockHeight: 5},
					{BlockHeight: 15},
				},
			},
			blockHeight: 20,
			epochLength: 10,
			expectedNonce: emissions.Nonces{
				Nonces: []*emissions.Nonce{
					{BlockHeight: 15},
				},
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			actual := inference_synthesis.FilterNoncesWithinEpochLength(tc.nonces, tc.blockHeight, tc.epochLength)
			s.Require().Equal(tc.expectedNonce, actual, "Filter nonces do not match")
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestSelectTopNReputerNonces() {
	// Define test cases
	tests := []struct {
		name                     string
		reputerRequestNonces     *emissions.ReputerRequestNonces
		N                        int
		expectedTopNReputerNonce []*emissions.ReputerRequestNonce
		currentBlockHeight       int64
		groundTruthLag           int64
	}{
		{
			name: "N greater than length of nonces, zero lag",
			reputerRequestNonces: &emissions.ReputerRequestNonces{
				Nonces: []*emissions.ReputerRequestNonce{
					{ReputerNonce: &emissions.Nonce{BlockHeight: 1}, WorkerNonce: &emissions.Nonce{BlockHeight: 2}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 3}, WorkerNonce: &emissions.Nonce{BlockHeight: 4}},
				},
			},
			N: 5,
			expectedTopNReputerNonce: []*emissions.ReputerRequestNonce{
				{ReputerNonce: &emissions.Nonce{BlockHeight: 3}, WorkerNonce: &emissions.Nonce{BlockHeight: 4}},
				{ReputerNonce: &emissions.Nonce{BlockHeight: 1}, WorkerNonce: &emissions.Nonce{BlockHeight: 2}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     0,
		},
		{
			name: "N less than length of nonces, zero lag",
			reputerRequestNonces: &emissions.ReputerRequestNonces{
				Nonces: []*emissions.ReputerRequestNonce{
					{ReputerNonce: &emissions.Nonce{BlockHeight: 1}, WorkerNonce: &emissions.Nonce{BlockHeight: 2}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 3}, WorkerNonce: &emissions.Nonce{BlockHeight: 4}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 5}, WorkerNonce: &emissions.Nonce{BlockHeight: 6}},
				},
			},
			N: 2,
			expectedTopNReputerNonce: []*emissions.ReputerRequestNonce{
				{ReputerNonce: &emissions.Nonce{BlockHeight: 5}, WorkerNonce: &emissions.Nonce{BlockHeight: 6}},
				{ReputerNonce: &emissions.Nonce{BlockHeight: 3}, WorkerNonce: &emissions.Nonce{BlockHeight: 4}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     0,
		},
		{
			name: "Ground truth lag cutting selection midway",
			reputerRequestNonces: &emissions.ReputerRequestNonces{
				Nonces: []*emissions.ReputerRequestNonce{
					{ReputerNonce: &emissions.Nonce{BlockHeight: 2}, WorkerNonce: &emissions.Nonce{BlockHeight: 1}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 6}, WorkerNonce: &emissions.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 4}, WorkerNonce: &emissions.Nonce{BlockHeight: 3}},
				},
			},
			N: 3,
			expectedTopNReputerNonce: []*emissions.ReputerRequestNonce{
				{ReputerNonce: &emissions.Nonce{BlockHeight: 4}, WorkerNonce: &emissions.Nonce{BlockHeight: 3}},
				{ReputerNonce: &emissions.Nonce{BlockHeight: 2}, WorkerNonce: &emissions.Nonce{BlockHeight: 1}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     6,
		},
		{
			name: "Big Ground truth lag, not selecting any nonces",
			reputerRequestNonces: &emissions.ReputerRequestNonces{
				Nonces: []*emissions.ReputerRequestNonce{
					{ReputerNonce: &emissions.Nonce{BlockHeight: 2}, WorkerNonce: &emissions.Nonce{BlockHeight: 1}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 6}, WorkerNonce: &emissions.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 4}, WorkerNonce: &emissions.Nonce{BlockHeight: 3}},
				},
			},
			N:                        3,
			expectedTopNReputerNonce: []*emissions.ReputerRequestNonce{},
			currentBlockHeight:       10,
			groundTruthLag:           10,
		},
		{
			name: "Small ground truth lag, selecting all nonces",
			reputerRequestNonces: &emissions.ReputerRequestNonces{
				Nonces: []*emissions.ReputerRequestNonce{
					{ReputerNonce: &emissions.Nonce{BlockHeight: 6}, WorkerNonce: &emissions.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 5}, WorkerNonce: &emissions.Nonce{BlockHeight: 4}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 4}, WorkerNonce: &emissions.Nonce{BlockHeight: 3}},
				},
			},
			N: 3,
			expectedTopNReputerNonce: []*emissions.ReputerRequestNonce{
				{ReputerNonce: &emissions.Nonce{BlockHeight: 6}, WorkerNonce: &emissions.Nonce{BlockHeight: 5}},
				{ReputerNonce: &emissions.Nonce{BlockHeight: 5}, WorkerNonce: &emissions.Nonce{BlockHeight: 4}},
				{ReputerNonce: &emissions.Nonce{BlockHeight: 4}, WorkerNonce: &emissions.Nonce{BlockHeight: 3}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     2,
		},
		{
			name: "Mid ground truth lag, selecting some nonces",
			reputerRequestNonces: &emissions.ReputerRequestNonces{
				Nonces: []*emissions.ReputerRequestNonce{
					{ReputerNonce: &emissions.Nonce{BlockHeight: 6}, WorkerNonce: &emissions.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 5}, WorkerNonce: &emissions.Nonce{BlockHeight: 4}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 4}, WorkerNonce: &emissions.Nonce{BlockHeight: 3}},
				},
			},
			N: 3,
			expectedTopNReputerNonce: []*emissions.ReputerRequestNonce{
				{ReputerNonce: &emissions.Nonce{BlockHeight: 5}, WorkerNonce: &emissions.Nonce{BlockHeight: 4}},
				{ReputerNonce: &emissions.Nonce{BlockHeight: 4}, WorkerNonce: &emissions.Nonce{BlockHeight: 3}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     5,
		},
	}

	// Run test cases
	for _, tc := range tests {
		s.Run(tc.name, func() {
			actual := inference_synthesis.SelectTopNReputerNonces(tc.reputerRequestNonces, tc.N, tc.currentBlockHeight, tc.groundTruthLag)
			s.Require().Equal(tc.expectedTopNReputerNonce, actual, "Reputer nonces do not match")
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestSelectTopNWorkerNonces() {
	// Define test cases
	tests := []struct {
		name               string
		workerNonces       emissions.Nonces
		N                  int
		expectedTopNNonces []*emissions.Nonce
	}{
		{
			name: "N greater than length of nonces",
			workerNonces: emissions.Nonces{
				Nonces: []*emissions.Nonce{
					{BlockHeight: 1},
					{BlockHeight: 2},
				},
			},
			N: 5,
			expectedTopNNonces: []*emissions.Nonce{
				{BlockHeight: 1},
				{BlockHeight: 2},
			},
		},
		{
			name: "N less than length of nonces",
			workerNonces: emissions.Nonces{
				Nonces: []*emissions.Nonce{
					{BlockHeight: 1},
					{BlockHeight: 2},
					{BlockHeight: 3},
				},
			},
			N: 2,
			expectedTopNNonces: []*emissions.Nonce{
				{BlockHeight: 1},
				{BlockHeight: 2},
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		s.Run(tc.name, func() {
			actual := inference_synthesis.SelectTopNWorkerNonces(tc.workerNonces, tc.N)
			s.Require().Equal(actual, tc.expectedTopNNonces, "Worker nonces to not match")
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkInferencesTwoWorkerTwoForecasters() {
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
			{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
			{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.7")},
		},
	}

	forecasts := &emissions.Forecasts{
		Forecasts: []*emissions.Forecast{
			{
				Forecaster: worker3,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.6")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
				},
			},
			{
				Forecaster: worker4,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("0.2")
	epsilon := alloraMath.MustNewDecFromString("0.001")
	pInferenceSynthesis := alloraMath.MustNewDecFromString("2")

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker3Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker4Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3Add, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3Add, worker2Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4Add, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4Add, worker2Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)

	// Call the function
	valueBundle, err := inference_synthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, pInferenceSynthesis)
	s.Require().NoError(err)

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)

	s.Require().Len(valueBundle.OneOutInfererValues, 2)
	s.Require().Len(valueBundle.OneOutForecasterValues, 2)
	s.Require().Len(valueBundle.OneInForecasterValues, 2)
}

func (s *InferenceSynthesisTestSuite) TestCalcNetworkInferencesThreeWorkerThreeForecasters() {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"
	worker1Add := sdk.AccAddress(worker1)
	worker2Add := sdk.AccAddress(worker2)
	worker3Add := sdk.AccAddress(worker3)

	forecaster1 := "forecaster1"
	forecaster2 := "forecaster2"
	forecaster3 := "forecaster3"
	forecaster1Add := sdk.AccAddress(forecaster1)
	forecaster2Add := sdk.AccAddress(forecaster2)
	forecaster3Add := sdk.AccAddress(forecaster3)

	// Set up input data
	inferences := &emissions.Inferences{
		Inferences: []*emissions.Inference{
			{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.1")},
			{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.2")},
			{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.3")},
		},
	}

	forecasts := &emissions.Forecasts{
		Forecasts: []*emissions.Forecast{
			{
				Forecaster: forecaster1,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.5")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.6")},
				},
			},
			{
				Forecaster: forecaster2,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.7")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
			{
				Forecaster: forecaster3,
				ForecastElements: []*emissions.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.1")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.2")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.3")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("0.3")
	epsilon := alloraMath.MustNewDecFromString("0.001")
	pInferenceSynthesis := alloraMath.MustNewDecFromString("2")

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.1")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker3Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster2Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster3Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1Add, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.7")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1Add, worker2Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.8")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1Add, worker3Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.9")})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2Add, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.1")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2Add, worker2Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2Add, worker3Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3Add, worker1Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3Add, worker2Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3Add, worker3Add, emissions.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)

	// Call the function
	valueBundle, err := inference_synthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, pInferenceSynthesis)
	s.Require().NoError(err)

	// Check the results
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)

	s.Require().Len(valueBundle.OneOutInfererValues, 3)
	s.Require().Len(valueBundle.OneOutForecasterValues, 3)
	s.Require().Len(valueBundle.OneInForecasterValues, 3)
}

func (s *InferenceSynthesisTestSuite) TestSortByBlockHeight() {
	// Create some test data
	tests := []struct {
		name   string
		input  *emissions.ReputerRequestNonces
		output *emissions.ReputerRequestNonces
	}{
		{
			name: "Sorted in descending order",
			input: &emissions.ReputerRequestNonces{
				Nonces: []*emissions.ReputerRequestNonce{
					{ReputerNonce: &emissions.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 2}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 7}},
				},
			},
			output: &emissions.ReputerRequestNonces{
				Nonces: []*emissions.ReputerRequestNonce{
					{ReputerNonce: &emissions.Nonce{BlockHeight: 7}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 2}},
				},
			},
		},
		{
			name: "Already sorted",
			input: &emissions.ReputerRequestNonces{
				Nonces: []*emissions.ReputerRequestNonce{
					{ReputerNonce: &emissions.Nonce{BlockHeight: 10}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 9}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 7}},
				},
			},
			output: &emissions.ReputerRequestNonces{
				Nonces: []*emissions.ReputerRequestNonce{
					{ReputerNonce: &emissions.Nonce{BlockHeight: 10}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 9}},
					{ReputerNonce: &emissions.Nonce{BlockHeight: 7}},
				},
			},
		},
		{
			name: "Empty input",
			input: &emissions.ReputerRequestNonces{
				Nonces: []*emissions.ReputerRequestNonce{},
			},
			output: &emissions.ReputerRequestNonces{
				Nonces: []*emissions.ReputerRequestNonce{},
			},
		},
		{
			name: "Single element",
			input: &emissions.ReputerRequestNonces{
				Nonces: []*emissions.ReputerRequestNonce{
					{ReputerNonce: &emissions.Nonce{BlockHeight: 3}},
				},
			},
			output: &emissions.ReputerRequestNonces{
				Nonces: []*emissions.ReputerRequestNonce{
					{ReputerNonce: &emissions.Nonce{BlockHeight: 3}},
				},
			},
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			// Call the sorting function
			inference_synthesis.SortByBlockHeight(test.input.Nonces)

			// Compare the sorted input with the expected output
			s.Require().Equal(test.input.Nonces, test.output.Nonces, "Sorting result mismatch.\nExpected: %v\nGot: %v")
		})
	}
}
