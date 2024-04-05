package module_test

import (
	"log"
	"math"

	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *ModuleTestSuite) TestGradient() {
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
			result, err := module.Gradient(tc.p, tc.x)

			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {
				s.Require().NoError(err)
				s.Require().InEpsilon(tc.expected, result, 1e-5, "result should match expected value within epsilon")
			}
		})
	}
}

func (s *ModuleTestSuite) TestCalcForcastImpliedInferences() {
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
			result, err := module.CalcForcastImpliedInferences(tc.inferenceByWorker, tc.forecasts, tc.networkCombinedLoss, tc.epsilon, tc.pInferenceSynthesis)

			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {

				for key, expectedValue := range tc.expected {
					actualValue, exists := result[key]
					s.Require().True(exists, "Expected key does not exist in result map")
					log.Printf("Checking if values are within tolerance for key: %s. Expected: %v, Actual: %v, Tolerance: %v", key, expectedValue.Value, actualValue.Value, 1e-5)
					s.Require().InEpsilon(expectedValue.Value, actualValue.Value, 1e-5, "Values do not match for key: %s", key)
				}
			}
		})
	}
}

func (s *ModuleTestSuite) TestFindMaxRegret() {
	tests := []struct {
		name              string
		regrets           module.RegretsByWorkerByType
		epsilon           float64
		expectedMaxRegret float64
	}{

		{
			name: "inference regret higher",
			regrets: module.RegretsByWorkerByType{
				InferenceRegrets: &map[string]*float64{"worker1": floatPtr(0.2), "worker2": floatPtr(0.5)},
				ForecastRegrets:  &map[string]*float64{"worker3": floatPtr(0.1), "worker4": floatPtr(0.4)},
			},
			epsilon:           0.05,
			expectedMaxRegret: 0.5,
		},
		{
			name: "forecast regret higher",
			regrets: module.RegretsByWorkerByType{
				InferenceRegrets: &map[string]*float64{"worker1": floatPtr(0.2), "worker2": floatPtr(0.3)},
				ForecastRegrets:  &map[string]*float64{"worker3": floatPtr(0.1), "worker4": floatPtr(0.4), "worker5": floatPtr(0.6)},
			},
			epsilon:           0.05,
			expectedMaxRegret: 0.6,
		},
		{
			name: "all below epsilon",
			regrets: module.RegretsByWorkerByType{
				InferenceRegrets: &map[string]*float64{"worker1": floatPtr(0.01)},
				ForecastRegrets:  &map[string]*float64{"worker2": floatPtr(0.02)},
			},
			epsilon:           0.05,
			expectedMaxRegret: 0.05,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			maxRegret := module.FindMaxRegret(&tc.regrets, tc.epsilon)
			s.Require().Equal(tc.expectedMaxRegret, maxRegret, "Expected and actual max regret do not match")
		})
	}
}

// Helper function to create pointers to float64 values
func floatPtr(f float64) *float64 {
	return &f
}

func (s *ModuleTestSuite) TestCalcWeightedInference() {
	tests := []struct {
		name                                  string
		inferenceByWorker                     map[string]*emissions.Inference
		forecastImpliedInferenceByWorker      map[string]*emissions.Inference
		regrets                               module.RegretsByWorkerByType
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
			regrets: module.RegretsByWorkerByType{
				InferenceRegrets: &map[string]*float64{"worker1": floatPtr(0.2), "worker2": floatPtr(0.5)},
				ForecastRegrets:  &map[string]*float64{"worker1": floatPtr(0.1), "worker2": floatPtr(0.4)},
			},
			epsilon:                               1e-4,
			pInferenceSynthesis:                   2.0,
			expectedNetworkCombinedInferenceValue: 1.9157039893342431,
			expectedErr:                           nil,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			networkCombinedInferenceValue, err := module.CalcWeightedInference(tc.inferenceByWorker, tc.forecastImpliedInferenceByWorker, &tc.regrets, tc.epsilon, tc.pInferenceSynthesis)

			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {
				s.Require().NoError(err)
				s.Require().InEpsilon(tc.expectedNetworkCombinedInferenceValue, networkCombinedInferenceValue, 1e-5, "Network combined inference value should match expected value within epsilon")
			}
		})
	}
}

func (s *ModuleTestSuite) TestCalcOneOutInferences() {
	test := struct {
		name                             string
		inferenceByWorker                map[string]*emissions.Inference
		forecastImpliedInferenceByWorker map[string]*emissions.Inference
		forecasts                        *emissions.Forecasts
		regrets                          module.RegretsByWorkerByType
		networkCombinedInference         float64
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
		regrets: module.RegretsByWorkerByType{
			InferenceRegrets: &map[string]*float64{"worker1": floatPtr(0.2), "worker2": floatPtr(0.5)},
			ForecastRegrets:  &map[string]*float64{"worker1": floatPtr(0.1), "worker2": floatPtr(0.4)},
		},
		networkCombinedInference: 10.0,
		epsilon:                  0.0001,
		expectedOneOutInferences: []*emissions.WithheldWorkerAttributedValue{
			{Worker: "worker1", Value: 9.8},
			{Worker: "worker2", Value: 9.5},
		},
		expectedOneOutImpliedInferences: []*emissions.WithheldWorkerAttributedValue{
			{Worker: "worker1", Value: 9.7},
			{Worker: "worker2", Value: 9.4},
		},
		pInferenceSynthesis: 2.0,
	}

	s.Run(test.name, func() {
		oneOutInferences, oneOutImpliedInferences, err := module.CalcOneOutInferences(
			test.inferenceByWorker,
			test.forecastImpliedInferenceByWorker,
			test.forecasts,
			&test.regrets,
			test.networkCombinedInference,
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

func (s *ModuleTestSuite) TestCalcOneInInferences() {
	tests := []struct {
		name                      string
		inferences                map[string]*emissions.Inference
		forecastImpliedInferences map[string]*emissions.Inference
		regrets                   module.RegretsByWorkerByType
		epsilon                   float64
		pInferenceSynthesis       float64
		expectedOneInInferences   []*emissions.WorkerAttributedValue
		expectedErr               error
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
			regrets: module.RegretsByWorkerByType{
				InferenceRegrets: &map[string]*float64{"worker1": floatPtr(0.2), "worker2": floatPtr(0.3)},
				ForecastRegrets:  &map[string]*float64{"worker1": floatPtr(0.1), "worker2": floatPtr(0.4)},
			},
			epsilon:             0.0001,
			pInferenceSynthesis: 2.0,
			expectedOneInInferences: []*emissions.WorkerAttributedValue{
				{Worker: "worker1", Value: 1.2},
				{Worker: "worker2", Value: 1.3},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			oneInInferences, err := module.CalcOneInInferences(
				tc.inferences,
				tc.forecastImpliedInferences,
				&tc.regrets,
				tc.epsilon,
				tc.pInferenceSynthesis,
			)

			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {
				s.Require().NoError(err)
				s.Require().Len(oneInInferences, len(tc.expectedOneInInferences), "Unexpected number of one-in inferences")

				for i, expected := range tc.expectedOneInInferences {
					actual := oneInInferences[i]
					s.Require().Equal(expected.Worker, actual.Worker, "Mismatch in worker for one-in inference")
					s.Require().InEpsilon(expected.Value, actual.Value, 1e-5, "Mismatch in value for one-in inference of worker %s", expected.Worker)
				}
			}
		})
	}
}
