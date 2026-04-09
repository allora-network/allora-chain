//nolint:exhaustruct
package inferencesynthesis_test

import (
	"testing"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/suite"

	alloraMath "github.com/allora-network/allora-chain/math"
	testutil2 "github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/utils/ptr"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
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
			storedWeight, err := s.WeightsKeeper().GetLatestInfererWeight(s.Ctx(), topicId, worker)
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

func (s *WeightsTestSuite) TestCalcWeightsPositive() {
	require := s.Require()

	inferers := []string{"A", "B", "C"}

	infererToRegret := map[string]*alloraMath.Dec{
		"A": ptr.To(alloraMath.MustNewDecFromString("0.1")),
		"B": ptr.To(alloraMath.MustNewDecFromString("0.2")),
		"C": ptr.To(alloraMath.MustNewDecFromString("0.3")),
	}

	args := synth.CalcWeightsGivenWorkersArgs{
		Logger:                 log.NewNopLogger(),
		Inferers:               inferers,
		InfererToRegret:        infererToRegret,
		EpsilonTopic:           alloraMath.MustNewDecFromString("0.0001"),
		PNorm:                  alloraMath.MustNewDecFromString("2"),
		CNorm:                  alloraMath.MustNewDecFromString("1"),
		RegretScalePlusEpsilon: alloraMath.MustNewDecFromString("1"),
	}

	weights, err := synth.CalcWeightsGivenWorkers(args)
	require.NoError(err)

	for _, w := range weights.Inferers {
		require.True(w.Gt(alloraMath.ZeroDec()))
	}
}

func (s *WeightsTestSuite) TestCalcWeightsEqualRegretsEqualWeights() {
	inferers := []string{"A", "B", "C"}

	infererToRegret := map[string]*alloraMath.Dec{
		"A": ptr.To(alloraMath.MustNewDecFromString("0.2")),
		"B": ptr.To(alloraMath.MustNewDecFromString("0.2")),
		"C": ptr.To(alloraMath.MustNewDecFromString("0.2")),
	}

	args := synth.CalcWeightsGivenWorkersArgs{
		Logger:                 log.NewNopLogger(),
		Inferers:               inferers,
		InfererToRegret:        infererToRegret,
		EpsilonTopic:           alloraMath.MustNewDecFromString("0.0001"),
		PNorm:                  alloraMath.MustNewDecFromString("2"),
		CNorm:                  alloraMath.MustNewDecFromString("1"),
		RegretScalePlusEpsilon: alloraMath.MustNewDecFromString("1"),
	}

	weights, err := synth.CalcWeightsGivenWorkers(args)
	s.Require().NoError(err)

	wA := weights.Inferers["A"]
	wB := weights.Inferers["B"]
	wC := weights.Inferers["C"]

	testutil2.InEpsilon5(s.T(), wA, wB.String())
	testutil2.InEpsilon5(s.T(), wB, wC.String())
}

func (s *WeightsTestSuite) TestGetCombinedInference() {
	require := s.Require()

	mustDec := func(v string) alloraMath.Dec {
		return alloraMath.MustNewDecFromString(v)
	}

	decPtr := func(v string) *alloraMath.Dec {
		d := mustDec(v)
		return &d
	}

	mkInf := func(worker string, vals ...string) *emissionstypes.Inference {
		out := make([]alloraMath.Dec, len(vals))
		for i, v := range vals {
			out[i] = mustDec(v)
		}
		return &emissionstypes.Inference{
			Inferer: worker,
			Values:  out,
		}
	}

	assertVecEqual := func(got emissionstypes.InferenceValues, want alloraMath.DecArray) {
		require.Len(got, len(want))
		for i := range want {
			testutil2.InEpsilon5(s.T(), got[i], want[i].String())
		}
	}

	type testCase struct {
		name          string
		args          synth.GetCombinedInferenceArgs
		wantCombined  alloraMath.DecArray
		assertWeights func(weights synth.RegretInformedWeights)
		assertResult  func(weights synth.RegretInformedWeights, combined emissionstypes.InferenceValues)
	}

	testCases := []testCase{
		{
			name: "all inferers new ignores forecasters and averages inferers only",
			args: synth.GetCombinedInferenceArgs{
				Logger:   log.NewNopLogger(),
				TopicId:  1,
				Inferers: []synth.Inferer{"i1", "i2"},
				InfererToInference: map[synth.Inferer]*emissionstypes.Inference{
					"i1": mkInf("i1", "1", "3"),
					"i2": mkInf("i2", "5", "7"),
				},
				InfererToRegret: map[synth.Inferer]*synth.Regret{
					"i1": decPtr("0"),
					"i2": decPtr("0"),
				},
				AllInferersAreNew: true,
				Forecasters:       []synth.Forecaster{"f1"},
				ForecasterToForecastImpliedInference: map[synth.Forecaster]*emissionstypes.Inference{
					"f1": mkInf("f1", "100", "100"),
				},
				ForecasterToRegret: map[synth.Forecaster]*synth.Regret{
					"f1": decPtr("0"),
				},
				EpsilonTopic:           mustDec("0.0001"),
				EpsilonSafeDiv:         mustDec("0.0000001"),
				PNorm:                  mustDec("2"),
				CNorm:                  mustDec("0.75"),
				RegretScalePlusEpsilon: mustDec("1"),
				NumLabels:              2,
			},
			wantCombined: alloraMath.DecArray{mustDec("3"), mustDec("5")},
		},
		{
			name: "equal regrets produce equal inferer weights and mean result",
			args: synth.GetCombinedInferenceArgs{
				Logger:   log.NewNopLogger(),
				TopicId:  1,
				Inferers: []synth.Inferer{"i1", "i2"},
				InfererToInference: map[synth.Inferer]*emissionstypes.Inference{
					"i1": mkInf("i1", "1"),
					"i2": mkInf("i2", "3"),
				},
				InfererToRegret: map[synth.Inferer]*synth.Regret{
					"i1": decPtr("0.2"),
					"i2": decPtr("0.2"),
				},
				AllInferersAreNew:                    false,
				Forecasters:                          nil,
				ForecasterToForecastImpliedInference: nil,
				ForecasterToRegret:                   nil,
				EpsilonTopic:                         mustDec("0.0001"),
				EpsilonSafeDiv:                       mustDec("0.0000001"),
				PNorm:                                mustDec("2"),
				CNorm:                                mustDec("0.75"),
				RegretScalePlusEpsilon:               mustDec("1"),
				NumLabels:                            1,
			},
			wantCombined: alloraMath.DecArray{mustDec("2")},
			assertWeights: func(weights synth.RegretInformedWeights) {
				require.Contains(weights.Inferers, "i1")
				require.Contains(weights.Inferers, "i2")
				require.True(weights.Inferers["i1"].Equal(weights.Inferers["i2"]))
			},
		},
		{
			name: "higher regret gets higher weight and pulls combined upward",
			args: synth.GetCombinedInferenceArgs{
				Logger:   log.NewNopLogger(),
				TopicId:  1,
				Inferers: []synth.Inferer{"i1", "i2", "i3"},
				InfererToInference: map[synth.Inferer]*emissionstypes.Inference{
					"i1": mkInf("i1", "1"),
					"i2": mkInf("i2", "2"),
					"i3": mkInf("i3", "10"),
				},
				InfererToRegret: map[synth.Inferer]*synth.Regret{
					"i1": decPtr("0.1"),
					"i2": decPtr("0.2"),
					"i3": decPtr("0.5"),
				},
				AllInferersAreNew:                    false,
				Forecasters:                          nil,
				ForecasterToForecastImpliedInference: nil,
				ForecasterToRegret:                   nil,
				EpsilonTopic:                         mustDec("0.0001"),
				EpsilonSafeDiv:                       mustDec("0.0000001"),
				PNorm:                                mustDec("2"),
				CNorm:                                mustDec("0.75"),
				RegretScalePlusEpsilon:               mustDec("1"),
				NumLabels:                            1,
			},
			assertWeights: func(weights synth.RegretInformedWeights) {
				require.True(weights.Inferers["i3"].Gt(weights.Inferers["i2"]))
				require.True(weights.Inferers["i2"].Gt(weights.Inferers["i1"]))
			},
			assertResult: func(_ synth.RegretInformedWeights, combined emissionstypes.InferenceValues) {
				require.Len(combined, 1)

				unweightedMean, err := mustDec("13").Quo(mustDec("3"))
				require.NoError(err)

				// The largest input (10) belongs to the highest-regret worker, so the
				// weighted result should move above the simple mean while remaining in-range.
				require.True(combined[0].Gt(unweightedMean))
				require.True(combined[0].Lt(mustDec("10")))
			},
		},
		{
			name: "forecaster implied inference contributes to combined inference",
			args: synth.GetCombinedInferenceArgs{
				Logger:   log.NewNopLogger(),
				TopicId:  1,
				Inferers: []synth.Inferer{"i1", "i2"},
				InfererToInference: map[synth.Inferer]*emissionstypes.Inference{
					"i1": mkInf("i1", "1", "0"),
					"i2": mkInf("i2", "0", "2"),
				},
				InfererToRegret: map[synth.Inferer]*synth.Regret{
					"i1": decPtr("0.2"),
					"i2": decPtr("0.2"),
				},
				AllInferersAreNew: false,
				Forecasters:       []synth.Forecaster{"f1"},
				ForecasterToForecastImpliedInference: map[synth.Forecaster]*emissionstypes.Inference{
					"f1": mkInf("f1", "5", "5"),
				},
				ForecasterToRegret: map[synth.Forecaster]*synth.Regret{
					"f1": decPtr("0.2"),
				},
				EpsilonTopic:           mustDec("0.0001"),
				EpsilonSafeDiv:         mustDec("0.0000001"),
				PNorm:                  mustDec("2"),
				CNorm:                  mustDec("0.75"),
				RegretScalePlusEpsilon: mustDec("1"),
				NumLabels:              2,
			},
			wantCombined: func() alloraMath.DecArray {
				sevenThirds, err := mustDec("7").Quo(mustDec("3"))
				require.NoError(err)
				return alloraMath.DecArray{mustDec("2"), sevenThirds}
			}(),
			assertWeights: func(weights synth.RegretInformedWeights) {
				require.Contains(weights.Inferers, "i1")
				require.Contains(weights.Inferers, "i2")
				require.Contains(weights.Forecasters, "f1")
				require.True(weights.Forecasters["f1"].Gt(alloraMath.ZeroDec()))
				require.True(weights.Inferers["i1"].Equal(weights.Inferers["i2"]))
				require.True(weights.Inferers["i2"].Equal(weights.Forecasters["f1"]))
			},
		},
		{
			name: "forecaster with distinct regret shifts combined inference toward implied values",
			args: synth.GetCombinedInferenceArgs{
				Logger:   log.NewNopLogger(),
				TopicId:  1,
				Inferers: []synth.Inferer{"i1", "i2"},
				InfererToInference: map[synth.Inferer]*emissionstypes.Inference{
					"i1": mkInf("i1", "1", "0"),
					"i2": mkInf("i2", "0", "2"),
				},
				InfererToRegret: map[synth.Inferer]*synth.Regret{
					"i1": decPtr("0.1"),
					"i2": decPtr("0.3"),
				},
				AllInferersAreNew: false,
				Forecasters:       []synth.Forecaster{"f1"},
				ForecasterToForecastImpliedInference: map[synth.Forecaster]*emissionstypes.Inference{
					"f1": mkInf("f1", "5", "5"),
				},
				ForecasterToRegret: map[synth.Forecaster]*synth.Regret{
					"f1": decPtr("0.2"),
				},
				EpsilonTopic:           mustDec("0.0001"),
				EpsilonSafeDiv:         mustDec("0.0000001"),
				PNorm:                  mustDec("2"),
				CNorm:                  mustDec("0.75"),
				RegretScalePlusEpsilon: mustDec("1"),
				NumLabels:              2,
			},
			assertWeights: func(weights synth.RegretInformedWeights) {
				require.Contains(weights.Inferers, "i1")
				require.Contains(weights.Inferers, "i2")
				require.Contains(weights.Forecasters, "f1")
				require.True(weights.Inferers["i2"].Gt(weights.Forecasters["f1"]))
				require.True(weights.Forecasters["f1"].Gt(weights.Inferers["i1"]))
			},
			assertResult: func(weights synth.RegretInformedWeights, combined emissionstypes.InferenceValues) {
				require.Len(combined, 2)

				w1 := weights.Inferers["i1"]
				w2 := weights.Inferers["i2"]
				wf := weights.Forecasters["f1"]

				sumWeights, err := w1.Add(w2)
				require.NoError(err)
				sumWeights, err = sumWeights.Add(wf)
				require.NoError(err)

				label0Numerator, err := wf.Mul(mustDec("5"))
				require.NoError(err)
				label0Numerator, err = label0Numerator.Add(w1)
				require.NoError(err)

				label1Numerator, err := w2.Mul(mustDec("2"))
				require.NoError(err)
				forecasterLabel1, err := wf.Mul(mustDec("5"))
				require.NoError(err)
				label1Numerator, err = label1Numerator.Add(forecasterLabel1)
				require.NoError(err)

				want0, err := label0Numerator.Quo(sumWeights)
				require.NoError(err)
				want1, err := label1Numerator.Quo(sumWeights)
				require.NoError(err)

				assertVecEqual(combined, alloraMath.DecArray{want0, want1})

				// Distinct regrets should pull the result toward the higher-regret inferer
				// while the forecaster still nudges both labels toward the implied [5, 5].
				require.True(combined[0].Gt(mustDec("1")))
				require.True(combined[0].Lt(mustDec("5")))
				require.True(combined[1].Gt(mustDec("2")))
				require.True(combined[1].Lt(mustDec("5")))
			},
		},
		{
			name: "missing inferer inference is ignored in aggregation",
			args: synth.GetCombinedInferenceArgs{
				Logger:   log.NewNopLogger(),
				TopicId:  1,
				Inferers: []synth.Inferer{"i1", "i2", "i3"},
				InfererToInference: map[synth.Inferer]*emissionstypes.Inference{
					"i1": mkInf("i1", "1"),
					"i2": mkInf("i2", "3"),
					// i3 intentionally missing
				},
				InfererToRegret: map[synth.Inferer]*synth.Regret{
					"i1": decPtr("0.2"),
					"i2": decPtr("0.2"),
					"i3": decPtr("0.2"),
				},
				AllInferersAreNew:                    false,
				Forecasters:                          nil,
				ForecasterToForecastImpliedInference: nil,
				ForecasterToRegret:                   nil,
				EpsilonTopic:                         mustDec("0.0001"),
				EpsilonSafeDiv:                       mustDec("0.0000001"),
				PNorm:                                mustDec("2"),
				CNorm:                                mustDec("0.75"),
				RegretScalePlusEpsilon:               mustDec("1"),
				NumLabels:                            1,
			},
			wantCombined: alloraMath.DecArray{mustDec("2")},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			weights, combined, err := synth.GetCombinedInference(tc.args)
			require.NoError(err)

			if tc.wantCombined != nil {
				assertVecEqual(combined, tc.wantCombined)
			}
			if tc.assertWeights != nil {
				tc.assertWeights(weights)
			}
			if tc.assertResult != nil {
				tc.assertResult(weights, combined)
			}
		})
	}
}
