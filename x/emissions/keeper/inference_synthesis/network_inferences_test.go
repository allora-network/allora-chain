package inference_synthesis_test

import (
	"reflect"
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/assert"

	inference_synthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
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
		inferences []*emissionstypes.Inference
		expected   map[string]*emissionstypes.Inference
	}{
		{
			name: "multiple workers",
			inferences: []*emissionstypes.Inference{
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
			expected: map[string]*emissionstypes.Inference{
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
			inferences: []*emissionstypes.Inference{},
			expected:   map[string]*emissionstypes.Inference{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := inference_synthesis.MakeMapFromWorkerToTheirWork(tc.inferences)
			assert.True(t, reflect.DeepEqual(result, tc.expected), "Expected and actual maps should be equal")
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcTheStdDevOfRegretsAmongWorkersWithLosses() {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"
	worker4 := "worker4"

	inferenceByWorker := map[string]*emissionstypes.Inference{
		worker1: {Value: alloraMath.MustNewDecFromString("0.5")},
		worker2: {Value: alloraMath.MustNewDecFromString("0.7")},
	}

	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		worker3: {Value: alloraMath.MustNewDecFromString("0.6")},
		worker4: {Value: alloraMath.MustNewDecFromString("0.8")},
	}

	epsilon := alloraMath.MustNewDecFromString("0.001")

	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	err = k.SetForecasterNetworkRegret(ctx, topicId, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker4, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker4, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	stdDevRegrets, err := inference_synthesis.CalcTheStdDevOfRegretsAmongWorkersWithLosses(
		ctx,
		k,
		topicId,
		inferenceByWorker,
		alloraMath.GetSortedKeys(inferenceByWorker),
		forecastImpliedInferenceByWorker,
		alloraMath.GetSortedKeys(forecastImpliedInferenceByWorker),
		epsilon,
	)
	s.Require().NoError(err)

	expectedStdDevInferenceRegret, err := alloraMath.MustNewDecFromString("0.050").Add(epsilon)
	s.Require().NoError(err)
	expectedStdDevForecastRegret, err := alloraMath.MustNewDecFromString("0.050").Add(epsilon)
	s.Require().NoError(err)
	expectedStdDevOneInForecastRegretWorker3, err := alloraMath.MustNewDecFromString("0.08164965809277260327324280249019638").Add(epsilon)
	s.Require().NoError(err)
	expectedStdDevOneInForecastRegretWorker4, err := alloraMath.MustNewDecFromString("0.08164965809277260327324280249019638").Add(epsilon)
	s.Require().NoError(err)

	s.Require().True(stdDevRegrets.StdDevInferenceRegret.Equal(expectedStdDevInferenceRegret), "StdDevInferenceRegret mismatch")
	s.Require().True(stdDevRegrets.StdDevForecastRegret.Equal(expectedStdDevForecastRegret), "StdDevForecastRegret mismatch")
	s.Require().True(stdDevRegrets.StdDevOneInForecastRegret[worker3].Equal(expectedStdDevOneInForecastRegretWorker3), "StdDevOneInForecastRegret[worker3] mismatch")
	s.Require().True(stdDevRegrets.StdDevOneInForecastRegret[worker4].Equal(expectedStdDevOneInForecastRegretWorker4), "StdDevOneInForecastRegret[worker4] mismatch")
}

/*
func (s *InferenceSynthesisTestSuite) TestCalcWeightedInferenceNormalOperation1() {
	topicId := inference_synthesis.TopicId(1)

	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.0514234892489971")},
		"worker1": {Value: alloraMath.MustNewDecFromString("-0.0316532211989242")},
		"worker2": {Value: alloraMath.MustNewDecFromString("-0.1018014248041400")},
	}
	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		"worker3": {Value: alloraMath.MustNewDecFromString("-0.0707517711518230")},
		"worker4": {Value: alloraMath.MustNewDecFromString("-0.0646463841210426")},
		"worker5": {Value: alloraMath.MustNewDecFromString("-0.0634099113416666")},
	}
	maxRegret := alloraMath.MustNewDecFromString("0.9871536722074480")
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	infererNetworkRegrets := map[string]inference_synthesis.Regret{
		"worker0": alloraMath.MustNewDecFromString("0.6975029322458370"),
		"worker1": alloraMath.MustNewDecFromString("0.910174442412618"),
		"worker2": alloraMath.MustNewDecFromString("0.9871536722074480"),
	}
	forecasterNetworkRegrets := map[string]inference_synthesis.Regret{
		"worker3": alloraMath.MustNewDecFromString("0.8308330665491310"),
		"worker4": alloraMath.MustNewDecFromString("0.8396961220162480"),
		"worker5": alloraMath.MustNewDecFromString("0.8017696138115460"),
	}
	expectedNetworkCombinedInferenceValue := alloraMath.MustNewDecFromString("-0.06470631905627390")

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	networkCombinedInferenceValue, err := inference_synthesis.CalcWeightedInference(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		inferenceByWorker,
		alloraMath.GetSortedKeys(inferenceByWorker),
		forecastImpliedInferenceByWorker,
		alloraMath.GetSortedKeys(forecastImpliedInferenceByWorker),
		NewWorkersAreNew(false),
		maxRegret,
		epsilon,
		pNorm,
		cNorm,
	)

	s.Require().NoError(err)

	s.Require().True(
		alloraMath.InDelta(
			expectedNetworkCombinedInferenceValue,
			networkCombinedInferenceValue,
			alloraMath.MustNewDecFromString("0.00001"),
		),
		"Network combined inference value should match expected value within epsilon",
		expectedNetworkCombinedInferenceValue.String(),
		networkCombinedInferenceValue.String(),
	)
}
*/

/*
func (s *InferenceSynthesisTestSuite) TestCalcWeightedInferenceNormalOperation2() {
	topicId := inference_synthesis.TopicId(1)

	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.14361768314408600")},
		"worker1": {Value: alloraMath.MustNewDecFromString("-0.23422685055675900")},
		"worker2": {Value: alloraMath.MustNewDecFromString("-0.18201270373970600")},
	}
	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		"worker3": {Value: alloraMath.MustNewDecFromString("-0.19840891048468800")},
		"worker4": {Value: alloraMath.MustNewDecFromString("-0.19696044261177800")},
		"worker5": {Value: alloraMath.MustNewDecFromString("-0.20289734770434400")},
	}
	maxRegret := alloraMath.MustNewDecFromString("0.9737035757621540")
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	pNorm := alloraMath.NewDecFromInt64(2)
	cNorm := alloraMath.MustNewDecFromString("0.75")
	infererNetworkRegrets := map[string]inference_synthesis.Regret{
		"worker0": alloraMath.MustNewDecFromString("0.5576393860961080"),
		"worker1": alloraMath.MustNewDecFromString("0.8588215562008240"),
		"worker2": alloraMath.MustNewDecFromString("0.9737035757621540"),
	}
	forecasterNetworkRegrets := map[string]inference_synthesis.Regret{
		"worker3": alloraMath.MustNewDecFromString("0.7535724745797420"),
		"worker4": alloraMath.MustNewDecFromString("0.7658774622830770"),
		"worker5": alloraMath.MustNewDecFromString("0.7185104293863190"),
	}
	expectedNetworkCombinedInferenceValue := alloraMath.MustNewDecFromString("-0.19486643996868")

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	networkCombinedInferenceValue, err := inference_synthesis.CalcWeightedInference(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		inferenceByWorker,
		alloraMath.GetSortedKeys(inferenceByWorker),
		forecastImpliedInferenceByWorker,
		alloraMath.GetSortedKeys(forecastImpliedInferenceByWorker),
		NewWorkersAreNew(false),
		maxRegret,
		epsilon,
		pNorm,
		cNorm,
	)

	s.Require().NoError(err)

	s.Require().True(
		alloraMath.InDelta(
			expectedNetworkCombinedInferenceValue,
			networkCombinedInferenceValue,
			alloraMath.MustNewDecFromString("0.00001"),
		),
		"Network combined inference value should match expected value within epsilon",
		expectedNetworkCombinedInferenceValue.String(),
		networkCombinedInferenceValue.String(),
	)
}
*/

/*
func (s *InferenceSynthesisTestSuite) TestCalcOneOutInferencesMultipleWorkers() {
	topicId := inference_synthesis.TopicId(1)
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.0514234892489971")},
		"worker1": {Value: alloraMath.MustNewDecFromString("-0.0316532211989242")},
		"worker2": {Value: alloraMath.MustNewDecFromString("-0.1018014248041400")},
	}
	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		"worker3": {Value: alloraMath.MustNewDecFromString("-0.0707517711518230")},
		"worker4": {Value: alloraMath.MustNewDecFromString("-0.0646463841210426")},
		"worker5": {Value: alloraMath.MustNewDecFromString("-0.0634099113416666")},
	}
	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "worker3",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.00011708024633613200")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.013382222402411400")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("3.82471429104471e-05")},
				},
			},
			{
				Forecaster: "worker4",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.00011486217283808300")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0060528036329761000")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.0005337255825785730")},
				},
			},
			{
				Forecaster: "worker5",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.001810780808278390")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0018544539679880700")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.001251454152216520")},
				},
			},
		},
	}
	infererNetworkRegrets := map[string]inference_synthesis.Regret{
		"worker0": alloraMath.MustNewDecFromString("0.6975029322458370"),
		"worker1": alloraMath.MustNewDecFromString("0.9101744424126180"),
		"worker2": alloraMath.MustNewDecFromString("0.9871536722074480"),
	}
	forecasterNetworkRegrets := map[string]inference_synthesis.Regret{
		"worker3": alloraMath.MustNewDecFromString("0.8308330665491310"),
		"worker4": alloraMath.MustNewDecFromString("0.8396961220162480"),
		"worker5": alloraMath.MustNewDecFromString("0.8017696138115460"),
	}
	maxRegret := alloraMath.MustNewDecFromString("0.987153672207448")
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.0156937658327922")
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expectedOneOutInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "worker0", Value: "-0.0711130346780"},
		{Worker: "worker1", Value: "-0.077954217717"},
		{Worker: "worker2", Value: "-0.0423024599518"},
	}
	expectedOneOutImpliedInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "worker3", Value: "-0.06351714496"},
		{Worker: "worker4", Value: "-0.06471822091"},
		{Worker: "worker5", Value: "-0.06495348528"},
	}

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	oneOutInfererValues, oneOutForecasterValues, err := inference_synthesis.CalcOneOutInferences(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		inferenceByWorker,
		alloraMath.GetSortedKeys(inferenceByWorker),
		forecastImpliedInferenceByWorker,
		alloraMath.GetSortedKeys(forecastImpliedInferenceByWorker),
		forecasts,
		NewWorkersAreNew(false),
		maxRegret,
		networkCombinedLoss,
		epsilon,
		fTolerance,
		pNorm,
		cNorm,
	)

	s.Require().NoError(err, "CalcOneOutInferences should not return an error")

	s.Require().Len(oneOutInfererValues, len(expectedOneOutInferences), "Unexpected number of one-out inferences")
	s.Require().Len(oneOutForecasterValues, len(expectedOneOutImpliedInferences), "Unexpected number of one-out implied inferences")

	for _, expected := range expectedOneOutInferences {
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

	for _, expected := range expectedOneOutImpliedInferences {
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
}
*/

/*
func (s *InferenceSynthesisTestSuite) TestCalcOneOutInferences5Workers3Forecasters() {
	topicId := inference_synthesis.TopicId(1)
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Value: alloraMath.MustNewDecFromString("-0.035995138925040600")},
		"worker1": {Value: alloraMath.MustNewDecFromString("-0.07333303938740420")},
		"worker2": {Value: alloraMath.MustNewDecFromString("-0.1495482917094790")},
		"worker3": {Value: alloraMath.MustNewDecFromString("-0.12952123274063800")},
		"worker4": {Value: alloraMath.MustNewDecFromString("-0.0703055329498285")},
	}
	forecastImpliedInferenceByWorker := map[string]*emissionstypes.Inference{
		"forecaster0": {Value: alloraMath.MustNewDecFromString("-0.08944493117005920")},
		"forecaster1": {Value: alloraMath.MustNewDecFromString("-0.07333218290300560")},
		"forecaster2": {Value: alloraMath.MustNewDecFromString("-0.07756206109376570")},
	}
	// epoch 3
	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.003305466418410120")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0002788248228566030")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString(".0000240536828602367")},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("0.0008240378476798250")},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("0.0000186192181193532")},
				},
			},
			{
				Forecaster: "forecaster1",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.002308441286328890")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0000214380788596749")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.012560171044167200")},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("0.017998563880697900")},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("0.00020024906252089700")},
				},
			},
			{
				Forecaster: "forecaster2",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("0.005369218152594270")},
					{Inferer: "worker1", Value: alloraMath.MustNewDecFromString("0.0002578158768320300")},
					{Inferer: "worker2", Value: alloraMath.MustNewDecFromString("0.0076008583603885900")},
					{Inferer: "worker3", Value: alloraMath.MustNewDecFromString("0.0076269073955871000")},
					{Inferer: "worker4", Value: alloraMath.MustNewDecFromString("0.00035670236460009500")},
				},
			},
		},
	}
	// epoch 2
	infererNetworkRegrets :=
		map[string]inference_synthesis.Regret{
			"worker0": alloraMath.MustNewDecFromString("0.29240710390153500"),
			"worker1": alloraMath.MustNewDecFromString("0.4182220944854450"),
			"worker2": alloraMath.MustNewDecFromString("0.17663501719135000"),
			"worker3": alloraMath.MustNewDecFromString("0.49617463489106400"),
			"worker4": alloraMath.MustNewDecFromString("0.27996060999688600"),
		}
	forecasterNetworkRegrets := map[string]inference_synthesis.Regret{
		"forecaster0": alloraMath.MustNewDecFromString("0.816066375505268"),
		"forecaster1": alloraMath.MustNewDecFromString("0.8234558901838660"),
		"forecaster2": alloraMath.MustNewDecFromString("0.8196673550408280"),
	}
	maxRegret := alloraMath.MustNewDecFromString("0.8234558901838660")
	// epoch 2
	networkCombinedLoss := alloraMath.MustNewDecFromString(".0000127791308799785")
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expectedOneOutInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "worker0", Value: "-0.0878179883784"},
		{Worker: "worker1", Value: "-0.0834415833800"},
		{Worker: "worker2", Value: "-0.0760530852479"},
		{Worker: "worker3", Value: "-0.0769113408092"},
		{Worker: "worker4", Value: "-0.0977096283034"},
	}
	expectedOneOutImpliedInferences := []struct {
		Worker string
		Value  string
	}{
		{Worker: "forecaster0", Value: "-0.0847805342051"},
		{Worker: "forecaster1", Value: "-0.0882088249132"},
		{Worker: "forecaster2", Value: "-0.0872998460256"},
	}

	for inferer, regret := range infererNetworkRegrets {
		s.emissionsKeeper.SetInfererNetworkRegret(
			s.ctx,
			topicId,
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	for forecaster, regret := range forecasterNetworkRegrets {
		s.emissionsKeeper.SetForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
		)
	}

	oneOutInfererValues, oneOutForecasterValues, err := inference_synthesis.CalcOneOutInferences(
		s.ctx,
		s.emissionsKeeper,
		topicId,
		inferenceByWorker,
		alloraMath.GetSortedKeys(inferenceByWorker),
		forecastImpliedInferenceByWorker,
		alloraMath.GetSortedKeys(forecastImpliedInferenceByWorker),
		forecasts,
		NewWorkersAreNew(false),
		maxRegret,
		networkCombinedLoss,
		epsilon,
		fTolerance,
		pNorm,
		cNorm,
	)

	s.Require().NoError(err, "CalcOneOutInferences should not return an error")

	s.Require().Len(oneOutInfererValues, len(expectedOneOutInferences), "Unexpected number of one-out inferences")
	s.Require().Len(oneOutForecasterValues, len(expectedOneOutImpliedInferences), "Unexpected number of one-out implied inferences")

	for _, expected := range expectedOneOutInferences {
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

	for _, expected := range expectedOneOutImpliedInferences {
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
}
*/

/*
func (s *InferenceSynthesisTestSuite) TestCalcOneInInferences() {
	topicId := inference_synthesis.TopicId(1)

	tests := []struct {
		name                        string
		inferenceByWorker           map[string]*emissionstypes.Inference
		forecastImpliedInferences   map[string]*emissionstypes.Inference
		maxRegretsByOneInForecaster map[string]inference_synthesis.Regret
		epsilon                     alloraMath.Dec
		pNorm                       alloraMath.Dec
		cNorm                       alloraMath.Dec
		infererNetworkRegrets       map[string]inference_synthesis.Regret
		forecasterNetworkRegrets    map[string]inference_synthesis.Regret
		expectedOneInInferences     []*emissionstypes.WorkerAttributedValue
		expectedErr                 error
	}{
		{ // EPOCH 3
			name: "basic functionality",
			inferenceByWorker: map[string]*emissionstypes.Inference{
				"worker0": {Value: alloraMath.MustNewDecFromString("-0.0514234892489971")},
				"worker1": {Value: alloraMath.MustNewDecFromString("-0.0316532211989242")},
				"worker2": {Value: alloraMath.MustNewDecFromString("-0.1018014248041400")},
			},
			forecastImpliedInferences: map[string]*emissionstypes.Inference{
				"worker3": {Value: alloraMath.MustNewDecFromString("-0.0707517711518230")},
				"worker4": {Value: alloraMath.MustNewDecFromString("-0.0646463841210426")},
				"worker5": {Value: alloraMath.MustNewDecFromString("-0.0634099113416666")},
			},
			maxRegretsByOneInForecaster: map[string]inference_synthesis.Regret{
				"worker3": alloraMath.MustNewDecFromString("0.9871536722074480"),
				"worker4": alloraMath.MustNewDecFromString("0.9871536722074480"),
				"worker5": alloraMath.MustNewDecFromString("0.9871536722074480"),
			},
			epsilon: alloraMath.MustNewDecFromString("0.0001"),
			pNorm:   alloraMath.MustNewDecFromString("2.0"),
			cNorm:   alloraMath.MustNewDecFromString("0.75"),
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
			expectedOneInInferences: []*emissionstypes.WorkerAttributedValue{
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
					inferer,
					emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
				)
			}

			for forecaster, regret := range tc.forecasterNetworkRegrets {
				s.emissionsKeeper.SetForecasterNetworkRegret(
					s.ctx,
					topicId,
					forecaster,
					emissionstypes.TimestampedValue{BlockHeight: 0, Value: regret},
				)
			}

			oneInInferences, err := inference_synthesis.CalcOneInInferences(
				s.ctx,
				s.emissionsKeeper,
				topicId,
				tc.inferenceByWorker,
				alloraMath.GetSortedKeys(tc.inferenceByWorker),
				tc.forecastImpliedInferences,
				alloraMath.GetSortedKeys(tc.forecastImpliedInferences),
				NewWorkersAreNew(false),
				tc.maxRegretsByOneInForecaster,
				tc.epsilon,
				tc.pNorm,
				tc.cNorm,
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
								"Mismatch in value for one-in inference of worker %s, expected %v, actual %v",
								expected.Worker,
								expected.Value,
								actual.Value,
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
*/

func (s *InferenceSynthesisTestSuite) TestCalcNetworkInferences() {
	k := s.emissionsKeeper
	ctx := s.ctx
	topicId := uint64(1)

	worker1 := "worker1"
	worker2 := "worker2"
	worker3 := "worker3"
	worker4 := "worker4"

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
			{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.7")},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: worker3,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.6")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
				},
			},
			{
				Forecaster: worker4,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("0.2")
	epsilon := alloraMath.MustNewDecFromString("0.001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker4, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)

	// Call the function
	valueBundle, err := inference_synthesis.CalcNetworkInferences(
		ctx,
		k,
		topicId,
		inferences,
		forecasts,
		networkCombinedLoss,
		epsilon,
		fTolerance,
		pNorm,
		cNorm,
	)
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

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.52")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.71")},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("1")
	epsilon := alloraMath.MustNewDecFromString("0.001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// Call the function
	valueBundle, err := inference_synthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, fTolerance, pNorm, cNorm)
	s.Require().NoError(err)
	s.Require().NotNil(valueBundle)
	s.Require().NotNil(valueBundle.CombinedValue)
	s.Require().NotNil(valueBundle.NaiveValue)
	s.Require().NotEmpty(valueBundle.OneOutInfererValues)
	s.Require().NotEmpty(valueBundle.OneOutForecasterValues)
	// s.Require().NotEmpty(valueBundle.OneInForecasterValues)

	// Set inferer network regrets
	err = k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker1, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker1, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker2, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker2, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)

	// Call the function
	valueBundle, err = inference_synthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, fTolerance, pNorm, cNorm)
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
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.52")},
			{TopicId: topicId, BlockHeight: blockHeightInferences, Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.71")},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
				},
			},
			{
				TopicId:     topicId,
				BlockHeight: blockHeightForecaster,
				Forecaster:  worker2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("1")
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// Call the function without setting regrets
	valueBundle, err := inference_synthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, fTolerance, pNorm, cNorm)
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

/*
func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesAtBlock() {
	require := s.Require()
	keeper := s.emissionsKeeper

	topicId := uint64(1)
	blockHeight := int64(3)
	require.True(blockHeight >= s.ctx.BlockHeight())
	s.ctx = s.ctx.WithBlockHeight(blockHeight)

	simpleNonce := emissionstypes.Nonce{BlockHeight: blockHeight}
	reputerRequestNonce := &emissionstypes.ReputerRequestNonce{
		ReputerNonce: &emissionstypes.Nonce{BlockHeight: blockHeight},
	}

	err := s.emissionsKeeper.SetTopic(s.ctx, topicId, emissionstypes.Topic{
		Id:              topicId,
		Creator:         "creator",
		Metadata:        "metadata",
		LossLogic:       "losslogic",
		LossMethod:      "lossmethod",
		InferenceLogic:  "inferencelogic",
		InferenceMethod: "inferencemethod",
		EpochLastEnded:  0,
		EpochLength:     100,
		GroundTruthLag:  10,
		DefaultArg:      "defaultarg",
		PNorm:           alloraMath.NewDecFromInt64(3),
		AlphaRegret:     alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:   false,
	})
	s.Require().NoError(err)

	reputer0 := "allo1m5v6rgjtxh4xszrrzqacwjh4ve6r0za2gxx9qr"
	reputer1 := "allo1e7cj9839ht2xm8urynqs5279hrvqd8neusvp2x"
	reputer2 := "allo1k9ss0xfer54nyack5678frl36e5g3rj2yzxtfj"
	reputer3 := "allo18ljxewge4vqrkk09tm5heldqg25yj8d9ekgkw5"
	reputer4 := "allo1k36ljvn8z0u49sagdg46p75psgreh23kdjn3l0"

	forecaster0 := "allo1pluvmvsmvecg2ccuqxa6ugzvc3a5udfyy0t76v"
	forecaster1 := "allo1e92saykj94jw3z55g4d3lfz098ppk0suwzc03a"
	forecaster2 := "allo1pk6mxny5p79t8zhkm23z7u3zmfuz2gn0snxkkt"

	// Set Loss bundles

	reputerLossBundles := emissionstypes.ReputerValueBundles{
		ReputerValueBundles: []*emissionstypes.ReputerValueBundle{
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer0,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000117005278862668"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer1,
					CombinedValue:       alloraMath.MustNewDecFromString(".00000962701954026944"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer2,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000256948644008351"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer3,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000123986052417188"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
			{
				ValueBundle: &emissionstypes.ValueBundle{
					Reputer:             reputer4,
					CombinedValue:       alloraMath.MustNewDecFromString(".0000115363240547692"),
					ReputerRequestNonce: reputerRequestNonce,
					TopicId:             topicId,
				},
			},
		},
	}

	err = keeper.InsertReputerLossBundlesAtBlock(s.ctx, topicId, blockHeight, reputerLossBundles)
	require.NoError(err)

	// Set Stake

	stake1, ok := cosmosMath.NewIntFromString("210535101370326000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer0, stake1)
	require.NoError(err)
	stake2, ok := cosmosMath.NewIntFromString("216697093951021000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer1, stake2)
	require.NoError(err)
	stake3, ok := cosmosMath.NewIntFromString("161740241803855000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer2, stake3)
	require.NoError(err)
	stake4, ok := cosmosMath.NewIntFromString("394848305052250000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer3, stake4)
	require.NoError(err)
	stake5, ok := cosmosMath.NewIntFromString("206169717590569000000000")
	s.Require().True(ok)
	err = keeper.AddStake(s.ctx, topicId, reputer4, stake5)
	require.NoError(err)

	// Set Inferences

	inferences := emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
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

	forecasts := emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: forecaster0,
				ForecastElements: []*emissionstypes.ForecastElement{
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
				ForecastElements: []*emissionstypes.ForecastElement{
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
				ForecastElements: []*emissionstypes.ForecastElement{
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
	s.Require().NoError(err)

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
			inferer,
			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
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
			forecaster,
			emissionstypes.TimestampedValue{BlockHeight: blockHeight, Value: regret},
		)
	}

	// Set one in forecaster network regrets

	setOneInForecasterNetworkRegret := func(forecaster string, inferer string, value string) {
		keeper.SetOneInForecasterNetworkRegret(
			s.ctx,
			topicId,
			forecaster,
			inferer,
			emissionstypes.TimestampedValue{
				BlockHeight: blockHeight,
				Value:       alloraMath.MustNewDecFromString(value),
			},
		)
	}

	/// Epoch 3 values

	setOneInForecasterNetworkRegret(forecaster0, reputer0, "-0.005488956369080480")
	setOneInForecasterNetworkRegret(forecaster0, reputer1, "0.17091263821766800")
	setOneInForecasterNetworkRegret(forecaster0, reputer2, "-0.15988639638192800")
	setOneInForecasterNetworkRegret(forecaster0, reputer3, "0.28690775330189800")
	setOneInForecasterNetworkRegret(forecaster0, reputer4, "-0.019476319822263300")

	setOneInForecasterNetworkRegret(forecaster0, forecaster0, "0.7370268872154170")

	setOneInForecasterNetworkRegret(forecaster1, reputer0, "-0.023601485104528100")
	setOneInForecasterNetworkRegret(forecaster1, reputer1, "0.1528001094822210")
	setOneInForecasterNetworkRegret(forecaster1, reputer2, "-0.1779989251173760")
	setOneInForecasterNetworkRegret(forecaster1, reputer3, "0.2687952245664510")
	setOneInForecasterNetworkRegret(forecaster1, reputer4, "-0.03758884855771100")

	setOneInForecasterNetworkRegret(forecaster1, forecaster1, "0.7307121775422120")

	setOneInForecasterNetworkRegret(forecaster2, reputer0, "-0.025084585281804600")
	setOneInForecasterNetworkRegret(forecaster2, reputer1, "0.15131700930494400")
	setOneInForecasterNetworkRegret(forecaster2, reputer2, "-0.17948202529465200")
	setOneInForecasterNetworkRegret(forecaster2, reputer3, "0.26731212438917400")
	setOneInForecasterNetworkRegret(forecaster2, reputer4, "-0.03907194873498750")

	setOneInForecasterNetworkRegret(forecaster2, forecaster2, "0.722844746771044")

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

	s.inEpsilon5(valueBundle.CombinedValue, "-0.08418238013037833391277949761424359")
	s.inEpsilon3(valueBundle.NaiveValue, "-0.09089296942031617121265381217201911")

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
			s.inEpsilon2(forecasterValue.Value, "-0.07360672083447152549990990449686835")
		case forecaster1:
			s.inEpsilon2(forecasterValue.Value, "-0.07263773178885971876894786458169429")
		case forecaster2:
			s.inEpsilon2(forecasterValue.Value, "-0.07333303938740419999999999999997501")
		default:
			require.Fail("Unexpected forecaster %v", forecasterValue.Worker)
		}
	}

	for _, oneOutInfererValue := range valueBundle.OneOutInfererValues {
		switch string(oneOutInfererValue.Worker) {
		case reputer0:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.09112256843970400263910853205529701")
		case reputer1:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.0849762526781571419651680849185323")
		case reputer2:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.07508631218815497306553765277306250")
		case reputer3:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.07762408532626815421861778958359602")
		case reputer4:
			s.inEpsilon2(oneOutInfererValue.Value, "-0.097732445271841")
		default:
			require.Fail("Unexpected worker %v", oneOutInfererValue.Worker)
		}
	}

	for _, oneInForecasterValue := range valueBundle.OneInForecasterValues {
		switch string(oneInForecasterValue.Worker) {
		case forecaster0:
			s.inEpsilon2(oneInForecasterValue.Value, "-0.08562185282145071310963674889631515")
		case forecaster1:
			s.inEpsilon2(oneInForecasterValue.Value, "-0.0857186447720307")
		case forecaster2:
			s.inEpsilon2(oneInForecasterValue.Value, "-0.0853937827718047")
		default:
			require.Fail("Unexpected worker %v", oneInForecasterValue.Worker)
		}
	}

	for _, oneOutForecasterValue := range valueBundle.OneOutForecasterValues {
		switch string(oneOutForecasterValue.Worker) {
		case forecaster0:
			s.inEpsilon2(oneOutForecasterValue.Value, "-0.08571218484894173358533220915281566")
		case forecaster1:
			s.inEpsilon2(oneOutForecasterValue.Value, "-0.08575177379258356927513673523336433")
		case forecaster2:
			s.inEpsilon2(oneOutForecasterValue.Value, "-0.08585235237690237323017634246068422")
		default:
			require.Fail("Unexpected worker %v", oneOutForecasterValue.Worker)
		}
	}
}
*/

func (s *InferenceSynthesisTestSuite) TestFilterNoncesWithinEpochLength() {
	tests := []struct {
		name          string
		nonces        emissionstypes.Nonces
		blockHeight   int64
		epochLength   int64
		expectedNonce emissionstypes.Nonces
	}{
		{
			name: "Nonces within epoch length",
			nonces: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 10},
					{BlockHeight: 15},
				},
			},
			blockHeight: 20,
			epochLength: 10,
			expectedNonce: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 10},
					{BlockHeight: 15},
				},
			},
		},
		{
			name: "Nonces outside epoch length",
			nonces: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 5},
					{BlockHeight: 15},
				},
			},
			blockHeight: 20,
			epochLength: 10,
			expectedNonce: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
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
		reputerRequestNonces     *emissionstypes.ReputerRequestNonces
		N                        int
		expectedTopNReputerNonce []*emissionstypes.ReputerRequestNonce
		currentBlockHeight       int64
		groundTruthLag           int64
	}{
		{
			name: "N greater than length of nonces, zero lag",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 1}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
				},
			},
			N: 5,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 1}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     0,
		},
		{
			name: "N less than length of nonces, zero lag",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 1}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 6}},
				},
			},
			N: 2,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 6}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     0,
		},
		{
			name: "Ground truth lag cutting selection midway",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 1}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			N: 3,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 1}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     6,
		},
		{
			name: "Big Ground truth lag, not selecting any nonces",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 1}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			N:                        3,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{},
			currentBlockHeight:       10,
			groundTruthLag:           10,
		},
		{
			name: "Small ground truth lag, selecting all nonces",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			N: 3,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
			},
			currentBlockHeight: 10,
			groundTruthLag:     2,
		},
		{
			name: "Mid ground truth lag, selecting some nonces",
			reputerRequestNonces: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 6}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			N: 3,
			expectedTopNReputerNonce: []*emissionstypes.ReputerRequestNonce{
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 4}},
				{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 4}, WorkerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
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
		workerNonces       emissionstypes.Nonces
		N                  int
		expectedTopNNonces []*emissionstypes.Nonce
	}{
		{
			name: "N greater than length of nonces",
			workerNonces: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 1},
					{BlockHeight: 2},
				},
			},
			N: 5,
			expectedTopNNonces: []*emissionstypes.Nonce{
				{BlockHeight: 1},
				{BlockHeight: 2},
			},
		},
		{
			name: "N less than length of nonces",
			workerNonces: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 1},
					{BlockHeight: 2},
					{BlockHeight: 3},
				},
			},
			N: 2,
			expectedTopNNonces: []*emissionstypes.Nonce{
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

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.5")},
			{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.7")},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: worker3,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.6")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
				},
			},
			{
				Forecaster: worker4,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("0.2")
	epsilon := alloraMath.MustNewDecFromString("0.0001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, worker4, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker3, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, worker4, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)

	// Call the function
	valueBundle, err := inference_synthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, fTolerance, pNorm, cNorm)
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

	forecaster1 := "forecaster1"
	forecaster2 := "forecaster2"
	forecaster3 := "forecaster3"

	// Set up input data
	inferences := &emissionstypes.Inferences{
		Inferences: []*emissionstypes.Inference{
			{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.1")},
			{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.2")},
			{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.3")},
		},
	}

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: forecaster1,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.4")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.5")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.6")},
				},
			},
			{
				Forecaster: forecaster2,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.7")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.8")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.9")},
				},
			},
			{
				Forecaster: forecaster3,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker1, Value: alloraMath.MustNewDecFromString("0.1")},
					{Inferer: worker2, Value: alloraMath.MustNewDecFromString("0.2")},
					{Inferer: worker3, Value: alloraMath.MustNewDecFromString("0.3")},
				},
			},
		},
	}

	networkCombinedLoss := alloraMath.MustNewDecFromString("0.3")
	epsilon := alloraMath.MustNewDecFromString("0.001")
	fTolerance := alloraMath.MustNewDecFromString("0.01")
	pNorm := alloraMath.MustNewDecFromString("2")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// Set inferer network regrets
	err := k.SetInfererNetworkRegret(ctx, topicId, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.1")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetInfererNetworkRegret(ctx, topicId, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	// Set forecaster network regrets
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)
	err = k.SetForecasterNetworkRegret(ctx, topicId, forecaster3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)

	// Set one-in forecaster network regrets
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.7")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.8")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster1, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.9")})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.1")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.2")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster2, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.3")})
	s.Require().NoError(err)

	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker1, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.4")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker2, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.5")})
	s.Require().NoError(err)
	err = k.SetOneInForecasterNetworkRegret(ctx, topicId, forecaster3, worker3, emissionstypes.TimestampedValue{Value: alloraMath.MustNewDecFromString("0.6")})
	s.Require().NoError(err)

	// Call the function
	valueBundle, err := inference_synthesis.CalcNetworkInferences(ctx, k, topicId, inferences, forecasts, networkCombinedLoss, epsilon, fTolerance, pNorm, cNorm)
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
		input  *emissionstypes.ReputerRequestNonces
		output *emissionstypes.ReputerRequestNonces
	}{
		{
			name: "Sorted in descending order",
			input: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 7}},
				},
			},
			output: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 7}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 5}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 2}},
				},
			},
		},
		{
			name: "Already sorted",
			input: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 10}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 9}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 7}},
				},
			},
			output: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 10}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 9}},
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 7}},
				},
			},
		},
		{
			name: "Empty input",
			input: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{},
			},
			output: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{},
			},
		},
		{
			name: "Single element",
			input: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
				},
			},
			output: &emissionstypes.ReputerRequestNonces{
				Nonces: []*emissionstypes.ReputerRequestNonce{
					{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 3}},
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
