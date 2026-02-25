package inferencesynthesis_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	alloraMath "github.com/allora-network/allora-chain/math"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
)

type WeightsTestSuite struct {
	testutil.TestSuite
}

func TestWeightsTestSuite(t *testing.T) {
	suite.Run(t, &WeightsTestSuite{
		testutil.NewTestSuite("inference_synthesis_weights"),
	})
}

func (s *WeightsTestSuite) TestNormalizeWeights() {
	testCases := []struct {
		name        string
		weights     synth.RegretInformedWeights
		expectError bool
		expected    map[string]alloraMath.Dec // expected normalized weights for each worker
	}{
		{
			name: "simple case - three weights",
			weights: synth.RegretInformedWeights{
				Inferers: map[string]alloraMath.Dec{
					s.AddrsStr(0): alloraMath.MustNewDecFromString("2.0"),
					s.AddrsStr(1): alloraMath.MustNewDecFromString("3.0"),
				},
				Forecasters: map[string]alloraMath.Dec{
					s.AddrsStr(2): alloraMath.MustNewDecFromString("5.0"),
				},
			},
			expectError: false,
			expected: map[string]alloraMath.Dec{
				s.AddrsStr(0): alloraMath.MustNewDecFromString("0.2"), // 2/10
				s.AddrsStr(1): alloraMath.MustNewDecFromString("0.3"), // 3/10
				s.AddrsStr(2): alloraMath.MustNewDecFromString("0.5"), // 5/10
			},
		},
		{
			name: "equal weights",
			weights: synth.RegretInformedWeights{
				Inferers: map[string]alloraMath.Dec{
					s.AddrsStr(0): alloraMath.MustNewDecFromString("1.0"),
					s.AddrsStr(1): alloraMath.MustNewDecFromString("1.0"),
				},
				Forecasters: map[string]alloraMath.Dec{
					s.AddrsStr(2): alloraMath.MustNewDecFromString("1.0"),
				},
			},
			expectError: false,
			expected: map[string]alloraMath.Dec{
				s.AddrsStr(0): alloraMath.MustNewDecFromString("0.333333333333333333"),
				s.AddrsStr(1): alloraMath.MustNewDecFromString("0.333333333333333333"),
				s.AddrsStr(2): alloraMath.MustNewDecFromString("0.333333333333333333"),
			},
		},
		{
			name: "empty maps",
			weights: synth.RegretInformedWeights{
				Inferers:    map[string]alloraMath.Dec{},
				Forecasters: map[string]alloraMath.Dec{},
			},
			expectError: true,
			expected:    nil,
		},
		{
			name: "zero weights",
			weights: synth.RegretInformedWeights{
				Inferers: map[string]alloraMath.Dec{
					s.AddrsStr(0): alloraMath.ZeroDec(),
					s.AddrsStr(1): alloraMath.ZeroDec(),
				},
				Forecasters: map[string]alloraMath.Dec{
					s.AddrsStr(2): alloraMath.ZeroDec(),
				},
			},
			expectError: true,
			expected:    nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := tc.weights.NormalizeWeights()

			if tc.expectError {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)

			// Verify each weight matches expected
			for addr, expectedWeight := range tc.expected { //nolint:maprange // reason: order not relevant
				var actualWeight alloraMath.Dec
				if weight, ok := tc.weights.Inferers[addr]; ok {
					actualWeight = weight
				} else if weight, ok := tc.weights.Forecasters[addr]; ok {
					actualWeight = weight
				}
				ok, err := alloraMath.InDelta(expectedWeight, actualWeight, alloraMath.MustNewDecFromString("0.00000001"))
				s.Require().NoError(err)
				s.Require().True(ok,
					"Weight for %s: expected %s, got %s",
					addr, expectedWeight, actualWeight)
			}

			// Verify sum is 1.0
			sum := alloraMath.ZeroDec()
			for _, w := range tc.weights.Inferers { //nolint:maprange // reason: order not relevant
				sum, err = sum.Add(w)
				s.Require().NoError(err)
			}
			for _, w := range tc.weights.Forecasters { //nolint:maprange // reason: order not relevant
				sum, err = sum.Add(w)
				s.Require().NoError(err)
			}
			ok, err := alloraMath.InDelta(sum, alloraMath.OneDec(), alloraMath.MustNewDecFromString("0.00000001"))
			s.Require().NoError(err)
			s.Require().True(ok,
				"Sum of weights: expected %s, got %s",
				alloraMath.OneDec(), sum)
		})
	}
}

func (s *WeightsTestSuite) TestStoreLatestNormalizedWeights() {
	s.Run("store and retrieve normalized weights", func() {
		topicId := uint64(1)
		weights := synth.RegretInformedWeights{
			Inferers: map[string]alloraMath.Dec{
				s.AddrsStr(0): alloraMath.MustNewDecFromString("0.2"),
				s.AddrsStr(1): alloraMath.MustNewDecFromString("0.3"),
			},
			Forecasters: map[string]alloraMath.Dec{
				s.AddrsStr(2): alloraMath.MustNewDecFromString("0.5"),
			},
		}

		err := synth.StoreLatestNormalizedWeights(s.Ctx(), *s.EmissionsKeeper(), topicId, weights)
		s.Require().NoError(err)

		// Verify stored weights
		for worker, expectedWeight := range weights.Inferers { //nolint:maprange // reason: order not relevant
			storedWeight, err := s.EmissionsKeeper().GetLatestInfererWeight(s.Ctx(), topicId, worker)
			s.Require().NoError(err)
			s.Require().True(expectedWeight.Equal(storedWeight))
		}
	})
}

func (s *WeightsTestSuite) TestGatherWorkerRegrets() {
	s.Run("error on empty regrets", func() {
		_, _, _, err := synth.GatherWorkerRegrets(
			s.Ctx().Logger(),
			nil,
			nil,
			map[string]*alloraMath.Dec{},
			map[string]*alloraMath.Dec{},
		)
		s.Require().Error(err)
	})

	s.Run("gather regrets from workers", func() {
		inferers := []string{s.AddrsStr(0), s.AddrsStr(1)}
		forecasters := []string{s.AddrsStr(2)}

		dec1 := alloraMath.MustNewDecFromString("0.1")
		dec2 := alloraMath.MustNewDecFromString("0.2")
		dec3 := alloraMath.MustNewDecFromString("0.3")

		infererToRegret := map[string]*alloraMath.Dec{
			s.AddrsStr(0): &dec1,
			s.AddrsStr(1): &dec2,
		}
		forecasterToRegret := map[string]*alloraMath.Dec{
			s.AddrsStr(2): &dec3,
		}

		regrets, infererRegrets, forecasterRegrets, err := synth.GatherWorkerRegrets(
			s.Ctx().Logger(),
			inferers,
			forecasters,
			infererToRegret,
			forecasterToRegret,
		)
		s.Require().NoError(err)
		s.Require().Equal([]alloraMath.Dec{dec1, dec2, dec3}, regrets)
		s.Require().Equal([]alloraMath.Dec{dec1, dec2}, infererRegrets)
		s.Require().Equal([]alloraMath.Dec{dec3}, forecasterRegrets)
	})
}

func (s *WeightsTestSuite) TestCalcMadPlusEpsilon() {
	testCases := []struct {
		name     string
		regrets  []alloraMath.Dec
		epsilon  alloraMath.Dec
		expected alloraMath.Dec
	}{
		{
			name: "simple case - three values",
			regrets: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("0.1"),
				alloraMath.MustNewDecFromString("0.2"),
				alloraMath.MustNewDecFromString("0.3"),
			},
			epsilon:  alloraMath.MustNewDecFromString("0.01"),
			expected: alloraMath.MustNewDecFromString("0.15826"),
		},
		{
			name: "all same values",
			regrets: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("0.1"),
				alloraMath.MustNewDecFromString("0.1"),
				alloraMath.MustNewDecFromString("0.1"),
			},
			epsilon:  alloraMath.MustNewDecFromString("0.01"),
			expected: alloraMath.MustNewDecFromString("0.01"),
		},
		{
			name: "larger spread",
			regrets: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("0.0"),
				alloraMath.MustNewDecFromString("0.5"),
				alloraMath.MustNewDecFromString("1.0"),
			},
			epsilon:  alloraMath.MustNewDecFromString("0.1"),
			expected: alloraMath.MustNewDecFromString("0.8413"),
		},
		{
			name: "larger epsilon",
			regrets: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("0.1"),
				alloraMath.MustNewDecFromString("0.2"),
				alloraMath.MustNewDecFromString("0.3"),
			},
			epsilon:  alloraMath.MustNewDecFromString("0.5"),
			expected: alloraMath.MustNewDecFromString("0.64826"),
		},
		{
			name: "outlier dominated - MAD stays zero",
			regrets: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("0.0"),
				alloraMath.MustNewDecFromString("0.0"),
				alloraMath.MustNewDecFromString("0.0"),
				alloraMath.MustNewDecFromString("100.0"),
			},
			epsilon:  alloraMath.MustNewDecFromString("0.01"),
			expected: alloraMath.MustNewDecFromString("0.01"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result, err := synth.CalcRegretScalePlusEpsilon(tc.regrets, tc.epsilon)
			s.Require().NoError(err)
			s.Require().True(result.Gte(tc.epsilon), "result should be greater than epsilon")
			ok, err := alloraMath.InDelta(result, tc.expected, alloraMath.MustNewDecFromString("0.00000001"))
			s.Require().NoError(err)
			s.Require().True(ok,
				"expected %s but got %s", tc.expected.String(), result.String())
		})
	}
}

// Helper function to create Dec pointer
func decPtr(s string) *alloraMath.Dec {
	dec := alloraMath.MustNewDecFromString(s)
	return &dec
}

func (s *WeightsTestSuite) TestCalcWeightsGivenWorkers() {
	testCases := []struct {
		name          string
		args          synth.CalcWeightsGivenWorkersArgs
		expectedError bool
		checkResult   func(result synth.RegretInformedWeights)
	}{
		{
			name: "basic calculation with single inferer and forecaster",
			args: synth.CalcWeightsGivenWorkersArgs{
				Logger:      s.Ctx().Logger(),
				Inferers:    []string{s.AddrsStr(0)},
				Forecasters: []string{s.AddrsStr(1)},
				InfererToRegret: map[string]*alloraMath.Dec{
					s.AddrsStr(0): decPtr("1.0"),
				},
				ForecasterToRegret: map[string]*alloraMath.Dec{
					s.AddrsStr(1): decPtr("2.0"),
				},
				EpsilonTopic:           alloraMath.MustNewDecFromString("0.01"),
				PNorm:                  alloraMath.MustNewDecFromString("3.0"),
				CNorm:                  alloraMath.MustNewDecFromString("0.75"),
				RegretScalePlusEpsilon: alloraMath.MustNewDecFromString("1.0"),
			},
			expectedError: false,
			checkResult: func(result synth.RegretInformedWeights) {
				s.Require().Len(result.Inferers, 1)
				s.Require().Len(result.Forecasters, 1)
				s.Require().True(result.Inferers[s.AddrsStr(0)].Lt(result.Forecasters[s.AddrsStr(1)]))
			},
		},
		{
			name: "basic calculation with negative inferer and positive forecaster",
			args: synth.CalcWeightsGivenWorkersArgs{
				Logger:      s.Ctx().Logger(),
				Inferers:    []string{s.AddrsStr(0)},
				Forecasters: []string{s.AddrsStr(1)},
				InfererToRegret: map[string]*alloraMath.Dec{
					s.AddrsStr(0): decPtr("-1.0"),
				},
				ForecasterToRegret: map[string]*alloraMath.Dec{
					s.AddrsStr(1): decPtr("2.0"),
				},
				EpsilonTopic:           alloraMath.MustNewDecFromString("0.01"),
				PNorm:                  alloraMath.MustNewDecFromString("3.0"),
				CNorm:                  alloraMath.MustNewDecFromString("0.75"),
				RegretScalePlusEpsilon: alloraMath.MustNewDecFromString("1.0"),
			},
			expectedError: false,
			checkResult: func(result synth.RegretInformedWeights) {
				s.T().Logf("Single worker test results:")
				s.Require().Len(result.Inferers, 1)
				s.Require().Len(result.Forecasters, 1)
				s.Require().True(result.Inferers[s.AddrsStr(0)].Lt(result.Forecasters[s.AddrsStr(1)]))
			},
		},
		{
			name: "basic calculation with positive inferer and negative forecaster",
			args: synth.CalcWeightsGivenWorkersArgs{
				Logger:      s.Ctx().Logger(),
				Inferers:    []string{s.AddrsStr(0)},
				Forecasters: []string{s.AddrsStr(1)},
				InfererToRegret: map[string]*alloraMath.Dec{
					s.AddrsStr(0): decPtr("1.0"),
				},
				ForecasterToRegret: map[string]*alloraMath.Dec{
					s.AddrsStr(1): decPtr("-2.0"),
				},
				EpsilonTopic:           alloraMath.MustNewDecFromString("0.01"),
				PNorm:                  alloraMath.MustNewDecFromString("3.0"),
				CNorm:                  alloraMath.MustNewDecFromString("0.75"),
				RegretScalePlusEpsilon: alloraMath.MustNewDecFromString("1.0"),
			},
			expectedError: false,
			checkResult: func(result synth.RegretInformedWeights) {
				s.Require().Len(result.Inferers, 1)
				s.Require().Len(result.Forecasters, 1)
				s.Require().True(result.Inferers[s.AddrsStr(0)].Gt(result.Forecasters[s.AddrsStr(1)]))
			},
		},
		{
			name: "calculation with multiple workers and mixed positive and negative regrets",
			args: synth.CalcWeightsGivenWorkersArgs{
				Logger:      s.Ctx().Logger(),
				Inferers:    []string{s.AddrsStr(0), s.AddrsStr(1)},
				Forecasters: []string{s.AddrsStr(2), s.AddrsStr(3)},
				InfererToRegret: map[string]*alloraMath.Dec{
					s.AddrsStr(0): decPtr("-1.0"),
					s.AddrsStr(1): decPtr("2.0"),
				},
				ForecasterToRegret: map[string]*alloraMath.Dec{
					s.AddrsStr(2): decPtr("1.5"),
					s.AddrsStr(3): decPtr("-0.5"),
				},
				EpsilonTopic:           alloraMath.MustNewDecFromString("0.01"),
				PNorm:                  alloraMath.MustNewDecFromString("3.0"),
				CNorm:                  alloraMath.MustNewDecFromString("0.75"),
				RegretScalePlusEpsilon: alloraMath.MustNewDecFromString("1.0"),
			},
			expectedError: false,
			checkResult: func(result synth.RegretInformedWeights) {
				s.Require().Len(result.Inferers, 2)
				s.Require().Len(result.Forecasters, 2)

				// Check that worker with higher regret has a higher weight
				s.Require().True(result.Inferers[s.AddrsStr(0)].Lt(result.Inferers[s.AddrsStr(1)]))
				s.Require().True(result.Forecasters[s.AddrsStr(2)].Gt(result.Forecasters[s.AddrsStr(3)]))
				// compare mixed
				s.Require().True(result.Forecasters[s.AddrsStr(3)].Gt(result.Inferers[s.AddrsStr(0)]))
				s.Require().True(result.Forecasters[s.AddrsStr(2)].Lt(result.Inferers[s.AddrsStr(1)]))

			},
		},
		{ //nolint:exhaustruct
			name: "empty workers should error",
			args: synth.CalcWeightsGivenWorkersArgs{
				Logger:                 s.Ctx().Logger(),
				Inferers:               []string{},
				Forecasters:            []string{},
				InfererToRegret:        map[string]*alloraMath.Dec{},
				ForecasterToRegret:     map[string]*alloraMath.Dec{},
				EpsilonTopic:           alloraMath.MustNewDecFromString("0.01"),
				PNorm:                  alloraMath.MustNewDecFromString("3.0"),
				CNorm:                  alloraMath.MustNewDecFromString("0.75"),
				RegretScalePlusEpsilon: alloraMath.MustNewDecFromString("1.0"),
			},
			expectedError: true,
		},
		{ //nolint:exhaustruct
			name: "missing regret values should error",
			args: synth.CalcWeightsGivenWorkersArgs{
				Logger:                 s.Ctx().Logger(),
				Inferers:               []string{s.AddrsStr(0)},
				Forecasters:            []string{s.AddrsStr(1)},
				InfererToRegret:        map[string]*alloraMath.Dec{},
				ForecasterToRegret:     map[string]*alloraMath.Dec{},
				EpsilonTopic:           alloraMath.MustNewDecFromString("0.01"),
				PNorm:                  alloraMath.MustNewDecFromString("3.0"),
				CNorm:                  alloraMath.MustNewDecFromString("0.75"),
				RegretScalePlusEpsilon: alloraMath.MustNewDecFromString("1.0"),
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result, err := synth.CalcWeightsGivenWorkers(tc.args)

			if tc.expectedError {
				s.Require().Error(err)
				return
			}

			s.Require().NoError(err)
			s.Require().NotNil(result)

			if tc.checkResult != nil {
				tc.checkResult(result)
			}
		})
	}
}
