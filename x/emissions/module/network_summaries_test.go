package module_test

import (
	"math"

	"github.com/allora-network/allora-chain/x/emissions/module"
	"github.com/allora-network/allora-chain/x/emissions/types"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

func (s *ModuleTestSuite) TestGradient() {
	// Define test cases
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
			// Call the function under test
			result, err := module.Gradient(tc.p, tc.x)

			// Validate the results
			if tc.expectedErr != nil {
				s.Require().ErrorIs(err, tc.expectedErr)
			} else {
				s.Require().NoError(err)
				s.Require().InEpsilon(tc.expected, result, 1e-5, "result should match expected value within epsilon")
			}
		})
	}
}

func (s *ModuleTestSuite) TestCalcForecastImpliedInferencesAtTimeWithDummyData() {
	test := struct {
		name                string
		inferences          *emissions.Inferences
		forecasts           *emissions.Forecasts
		networkValueBundle  *emissions.ValueBundle
		epsilon             float64
		pInferenceSynthesis float64
		expected            []*emissions.Inference // Adjusted to slice of pointers
		expectedErr         error
	}{
		name: "basic operation with valid inputs",
		inferences: &emissions.Inferences{
			Inferences: []*emissions.Inference{ // Adjusted to slice of pointers
				{Worker: "worker1", Value: 1.0},
				{Worker: "worker2", Value: 1.5},
			},
		},
		forecasts: &emissions.Forecasts{
			Forecasts: []*emissions.Forecast{ // Adjusted to slice of pointers
				{
					Forecaster: "forecaster1",
					ForecastElements: []*emissions.ForecastElement{ // Adjusted to slice of pointers
						{Inferer: "worker1", Value: 2.0},
						{Inferer: "worker2", Value: 3.0},
					},
				},
				{
					Forecaster: "forecaster2",
					ForecastElements: []*emissions.ForecastElement{ // Adjusted to slice of pointers
						{Inferer: "worker1", Value: 2.5},
						{Inferer: "worker2", Value: 3.5},
					},
				},
			},
		},
		networkValueBundle: &emissions.ValueBundle{
			CombinedValue: 10.0,
		},
		epsilon:             1e-5,
		pInferenceSynthesis: 2,
		expected:            []*emissions.Inference{ // Adjusted to slice of pointers
			// Placeholder for actual expected values, to be filled based on your function's logic
		},
		expectedErr: nil,
	}

	// Running the test
	s.Run(test.name, func() {
		actual, err := module.CalcForcastImpliedInferencesAtTime(
			s.ctx,
			s.emissionsKeeper,
			1, // Mock topic ID
			test.inferences,
			test.forecasts,
			test.networkValueBundle,
			test.epsilon,
			test.pInferenceSynthesis,
		)

		// Validate the results
		if test.expectedErr != nil {
			s.Require().ErrorIs(err, test.expectedErr)
		} else {
			s.Require().NoError(err)
			// Now let's use 'actual' to compare against 'expected'
			s.Require().Equal(len(test.expected), len(actual), "Expected and actual slices should have the same length")

			for i, expInference := range test.expected {
				s.Require().InDelta(expInference.Value, actual[i].Value, 0.01, "Expected and actual values should be within delta for each inference")
				s.Require().Equal(expInference.Worker, actual[i].Worker, "Expected and actual worker IDs should match")
			}
		}
	})
}

func (s *ModuleTestSuite) TestCalcNetworkCombinedInference() {
	ctx := s.ctx // Assuming you have a mock sdk.Context already set up

	// Mock inferences
	inferences := &types.Inferences{
		Inferences: []*types.Inference{
			{Worker: "worker1", Value: 1.0},
			{Worker: "worker2", Value: 2.0},
		},
	}

	// Mock forecast-implied inferences
	forecastImpliedInferences := []types.Inference{
		{Worker: "worker1", Value: 1.5},
		{Worker: "worker2", Value: 2.5},
	}

	// Mock regrets
	regrets := &types.WorkerRegrets{
		WorkerRegrets: []*types.WorkerRegret{
			{Worker: "worker1", InferenceRegret: 0.1, ForecastRegret: 0.2},
			{Worker: "worker2", InferenceRegret: 0.2, ForecastRegret: 0.1},
		},
	}

	epsilon := 1e-5
	pInferenceSynthesis := 2.0

	// Expected output needs to be calculated based on the input and the function's logic.
	// This is a placeholder for illustration.
	expectedOutput := 1.75

	// Call the function under test
	actualOutput, err := module.CalcNetworkCombinedInference(ctx, s.emissionsKeeper, 1, inferences, forecastImpliedInferences, regrets, epsilon, pInferenceSynthesis)

	// Validate no error returned
	require.NoError(s.T(), err)

	// Validate the output
	require.InDelta(s.T(), expectedOutput, actualOutput, 0.01, "The actual output should closely match the expected output.")
}
