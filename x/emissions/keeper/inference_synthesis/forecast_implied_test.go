package inferencesynthesis_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// OLD TEST - COMMENTED OUT DUE TO ALGORITHM CHANGE
// This test was designed for the old bounds-based algorithm with complex threshold logic.
// The expected values are no longer valid since we switched to the Exp1DivExp1-based algorithm.
/*
func (s *InferenceSynthesisTestSuite) TestCalcWeightFromRegret_OLD_BOUNDS_ALGORITHM() {
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	// NOTE: These expected values were computed using the old algorithm:
	// - Complex bounds with upperBound=6.75, lowerBound=8.25, threshold=17.25
	// - Multiple conditional logic branches with capping and anchoring
	// - Final gradient calculation: φ'_p(normalized_regret)
	testCases := []struct {
		regretFrac     string
		maxRegret      string
		expectedWeight string // OLD ALGORITHM EXPECTED VALUES
	}{
		{"-24.5", "-24.5", "0.0007835709572871582"},
		{"-20.0", "-19.5", "0.00017487379698341595"},
		{"-15.0", "-14.0", "0.0000390213853994281"},
		{"-10.5", "-10.0", "0.00017487379698341595"},
		{"-5.75", "-5.0", "0.00008260707334375042"},
		{"-1.0", "-0.5", "0.015660377080675192"},
		{"-0.25", "0.0", "0.14227761953270035"},
		{"0.0", "0.0", "0.28604839469732846"},
		{"0.5", "0.5", "0.9624639024738211"},
		{"1.0", "1.0", "2.037536097526179"},
		{"-1.32345", "0.1238729", "0.00595380787049663"},
		{"-0.8712641", "-0.8712641", "0.022985964160663532"},
		{"0.01987392", "0.01987392", "0.30185357993405315"},
	}

	for _, tc := range testCases {
		regretFrac := alloraMath.MustNewDecFromString(tc.regretFrac)
		maxRegret := alloraMath.MustNewDecFromString(tc.maxRegret)

		weight, err := inferencesynthesis.CalcWeightFromNormalizedRegret(regretFrac, maxRegret, pNorm, cNorm)
		s.Require().NoError(err)

		testutil.InEpsilon5(s.T(), weight, tc.expectedWeight)
	}
}
*/

// NEW COMPREHENSIVE TESTS FOR EXP1DIVEXP1-BASED ALGORITHM
// These tests cover the new algorithm: weight = Exp1DivExp1(-p*(max_regret-c), -p*(regret-c))

func (s *InferenceSynthesisTestSuite) TestCalcWeightFromNormalizedRegret_Exp1DivExp1_Basic() {
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	testCases := []struct {
		name           string
		regretFrac     string
		maxRegret      string
		expectedWeight string
		tolerance      string
	}{
		// Core test cases - covering various regret scenarios
		{
			name:           "Equal very negative regrets",
			regretFrac:     "-24.5",
			maxRegret:      "-24.5",
			expectedWeight: "1", // When regret == maxRegret, result is always 1
			tolerance:      "1e-15",
		},
		{
			name:           "Different negative regrets - regret worse than max",
			regretFrac:     "-20.0",
			maxRegret:      "-19.5",
			expectedWeight: "0.2231301601484298",
			tolerance:      "1e-10",
		},
		{
			name:           "Moderate negative regrets with gap",
			regretFrac:     "-15.0",
			maxRegret:      "-14.0",
			expectedWeight: "0.04978706836786394",
			tolerance:      "1e-10",
		},
		{
			name:           "Large negative regret, slightly better max",
			regretFrac:     "-10.5",
			maxRegret:      "-10.0",
			expectedWeight: "0.2231301601484315",
			tolerance:      "1e-10",
		},
		{
			name:           "Moderate negative regret, better max",
			regretFrac:     "-5.75",
			maxRegret:      "-5.0",
			expectedWeight: "0.1053992276019573",
			tolerance:      "1e-10",
		},
		{
			name:           "Small negative regret, better max",
			regretFrac:     "-1.0",
			maxRegret:      "-0.5",
			expectedWeight: "0.2271855183599896",
			tolerance:      "1e-10",
		},
		{
			name:           "Near-zero regret, zero max",
			regretFrac:     "-0.25",
			maxRegret:      "0.0",
			expectedWeight: "0.4973900296949617",
			tolerance:      "1e-10",
		},
		{
			name:           "Zero regret and max",
			regretFrac:     "0.0",
			maxRegret:      "0.0",
			expectedWeight: "1", // Identity case
			tolerance:      "1e-15",
		},
		{
			name:           "Small positive regret and max",
			regretFrac:     "0.5",
			maxRegret:      "0.5",
			expectedWeight: "1", // Identity case
			tolerance:      "1e-15",
		},
		{
			name:           "Positive regret and max",
			regretFrac:     "1.0",
			maxRegret:      "1.0",
			expectedWeight: "1", // Identity case
			tolerance:      "1e-15",
		},
		{
			name:           "Complex decimal regret, better decimal max",
			regretFrac:     "-1.32345",
			maxRegret:      "0.1238729",
			expectedWeight: "0.0149696696173212",
			tolerance:      "1e-10",
		},
		{
			name:           "Equal negative decimal regrets",
			regretFrac:     "-0.8712641",
			maxRegret:      "-0.8712641",
			expectedWeight: "1", // Identity case
			tolerance:      "1e-15",
		},
		{
			name:           "Small positive decimal regret and max",
			regretFrac:     "0.01987392",
			maxRegret:      "0.01987392",
			expectedWeight: "1", // Identity case
			tolerance:      "1e-15",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			regretFrac := alloraMath.MustNewDecFromString(tc.regretFrac)
			maxRegret := alloraMath.MustNewDecFromString(tc.maxRegret)

			weight, err := inferencesynthesis.CalcWeightFromNormalizedRegret(regretFrac, maxRegret, pNorm, cNorm)
			s.Require().NoError(err, "Failed to calculate weight for case: %s", tc.name)

			expected := alloraMath.MustNewDecFromString(tc.expectedWeight)
			tolerance := alloraMath.MustNewDecFromString(tc.tolerance)

			withinTolerance, err := alloraMath.InDelta(expected, weight, tolerance)
			s.Require().NoError(err, "Error in tolerance check for case: %s", tc.name)
			s.Require().True(withinTolerance,
				"Case %s: Expected %s, got %s, tolerance %s",
				tc.name, expected.String(), weight.String(), tolerance.String())

			// Additional sanity checks
			s.Require().True(weight.IsFinite(), "Weight should be finite for case: %s", tc.name)
			s.Require().False(weight.IsNaN(), "Weight should not be NaN for case: %s", tc.name)
			s.Require().True(weight.IsPositive(), "Weight should be positive for case: %s", tc.name)
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcWeightFromNormalizedRegret_Exp1DivExp1_EdgeCases() {
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	testCases := []struct {
		name           string
		regretFrac     string
		maxRegret      string
		expectedWeight string
		tolerance      string
		description    string
	}{
		{
			name:           "Very negative both - extreme case",
			regretFrac:     "-50",
			maxRegret:      "-50",
			expectedWeight: "1",
			tolerance:      "1e-15",
			description:    "Extreme negative values should still follow identity rule",
		},
		{
			name:           "Large positive both - extreme case",
			regretFrac:     "10",
			maxRegret:      "10",
			expectedWeight: "1",
			tolerance:      "1e-15",
			description:    "Extreme positive values should still follow identity rule",
		},
		{
			name:           "Zero regret, positive max",
			regretFrac:     "0",
			maxRegret:      "1",
			expectedWeight: "0.1403893629392022",
			tolerance:      "1e-10",
			description:    "Zero regret with positive max should give reasonable weight",
		},
		{
			name:           "Negative regret, zero max",
			regretFrac:     "-1",
			maxRegret:      "0",
			expectedWeight: "0.05474729930662831",
			tolerance:      "1e-10",
			description:    "Negative regret with zero max",
		},
		{
			name:           "Small positive regret, much larger positive max",
			regretFrac:     "0.1",
			maxRegret:      "2",
			expectedWeight: "0.1274825724107806",
			tolerance:      "1e-10",
			description:    "Small regret with much larger max",
		},
		{
			name:           "Large negative regret, small negative max",
			regretFrac:     "-10",
			maxRegret:      "-0.1",
			expectedWeight: "1.361775598712234e-13",
			tolerance:      "1e-20",
			description:    "Very small weight when regret much worse than max",
		},
		{
			name:           "Equal regrets near zero - negative",
			regretFrac:     "-0.001",
			maxRegret:      "-0.001",
			expectedWeight: "1",
			tolerance:      "1e-15",
			description:    "Very small negative equal regrets",
		},
		{
			name:           "Equal regrets near zero - positive",
			regretFrac:     "0.001",
			maxRegret:      "0.001",
			expectedWeight: "1",
			tolerance:      "1e-15",
			description:    "Very small positive equal regrets",
		},
		{
			name:           "Regret much smaller than max - cross-zero",
			regretFrac:     "-5",
			maxRegret:      "2",
			expectedWeight: "3.300012235137296e-08",
			tolerance:      "1e-15",
			description:    "Very small weight when regret much worse and across zero",
		},
		{
			name:           "Regret much larger than max - extreme weight",
			regretFrac:     "2",
			maxRegret:      "-5",
			expectedWeight: "30302917.95140558",
			tolerance:      "1e-2",
			description:    "Very large weight when regret much better than max",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			regretFrac := alloraMath.MustNewDecFromString(tc.regretFrac)
			maxRegret := alloraMath.MustNewDecFromString(tc.maxRegret)

			weight, err := inferencesynthesis.CalcWeightFromNormalizedRegret(regretFrac, maxRegret, pNorm, cNorm)
			s.Require().NoError(err, "Failed to calculate weight for case: %s", tc.name)

			expected := alloraMath.MustNewDecFromString(tc.expectedWeight)
			tolerance := alloraMath.MustNewDecFromString(tc.tolerance)

			withinTolerance, err := alloraMath.InDelta(expected, weight, tolerance)
			s.Require().NoError(err, "Error in tolerance check for case: %s", tc.name)
			s.Require().True(withinTolerance,
				"Case %s: %s\nExpected %s, got %s, tolerance %s",
				tc.name, tc.description, expected.String(), weight.String(), tolerance.String())

			// Additional sanity checks
			s.Require().True(weight.IsFinite(), "Weight should be finite for case: %s", tc.name)
			s.Require().False(weight.IsNaN(), "Weight should not be NaN for case: %s", tc.name)
			s.Require().True(weight.IsPositive(), "Weight should be positive for case: %s", tc.name)
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcWeightFromNormalizedRegret_Exp1DivExp1_DifferentParameters() {
	testCases := []struct {
		name        string
		pNorm       string
		cNorm       string
		regretFrac  string
		maxRegret   string
		description string
	}{
		{
			name:        "Low p, low c parameters",
			pNorm:       "1.0",
			cNorm:       "0.1",
			regretFrac:  "-1.0",
			maxRegret:   "-1.0",
			description: "With lower parameters, identity should still hold",
		},
		{
			name:        "High p, high c parameters",
			pNorm:       "5.0",
			cNorm:       "1.5",
			regretFrac:  "0.0",
			maxRegret:   "0.0",
			description: "With higher parameters, identity should still hold",
		},
		{
			name:        "Low p, high c parameters",
			pNorm:       "1.5",
			cNorm:       "2.0",
			regretFrac:  "1.0",
			maxRegret:   "1.0",
			description: "With mixed parameters, identity should still hold",
		},
		{
			name:        "High p, low c parameters",
			pNorm:       "4.0",
			cNorm:       "0.25",
			regretFrac:  "-0.5",
			maxRegret:   "-0.5",
			description: "With opposite mixed parameters, identity should still hold",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			pNorm := alloraMath.MustNewDecFromString(tc.pNorm)
			cNorm := alloraMath.MustNewDecFromString(tc.cNorm)
			regretFrac := alloraMath.MustNewDecFromString(tc.regretFrac)
			maxRegret := alloraMath.MustNewDecFromString(tc.maxRegret)

			weight, err := inferencesynthesis.CalcWeightFromNormalizedRegret(regretFrac, maxRegret, pNorm, cNorm)
			s.Require().NoError(err, "Failed to calculate weight for case: %s", tc.name)

			// For identity cases (regret == maxRegret), result should always be 1
			expectedOne := alloraMath.OneDec()
			tolerance := alloraMath.MustNewDecFromString("1e-15")

			withinTolerance, err := alloraMath.InDelta(expectedOne, weight, tolerance)
			s.Require().NoError(err, "Error in tolerance check for case: %s", tc.name)
			s.Require().True(withinTolerance,
				"Case %s: %s\nExpected 1, got %s (p=%s, c=%s)",
				tc.name, tc.description, weight.String(), tc.pNorm, tc.cNorm)
		})
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcWeightFromNormalizedRegret_Exp1DivExp1_MathematicalProperties() {
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	s.Run("Identity Property: regret == maxRegret => weight == 1", func() {
		testValues := []string{"-10", "-1", "-0.5", "0", "0.5", "1", "5"}
		for _, val := range testValues {
			regret := alloraMath.MustNewDecFromString(val)
			weight, err := inferencesynthesis.CalcWeightFromNormalizedRegret(regret, regret, pNorm, cNorm)
			s.Require().NoError(err)

			one := alloraMath.OneDec()
			tolerance := alloraMath.MustNewDecFromString("1e-15")
			withinTolerance, err := alloraMath.InDelta(one, weight, tolerance)
			s.Require().NoError(err)
			s.Require().True(withinTolerance, "For regret=%s, expected weight=1, got %s", val, weight.String())
		}
	})

	s.Run("Monotonicity: better regret (closer to max) => higher weight", func() {
		maxRegret := alloraMath.MustNewDecFromString("1.0")

		// Test points from worst to best regret
		regrets := []string{"-5", "-2", "-1", "0", "0.5", "1.0"}
		weights := make([]alloraMath.Dec, len(regrets))

		for i, regretStr := range regrets {
			regret := alloraMath.MustNewDecFromString(regretStr)
			weight, err := inferencesynthesis.CalcWeightFromNormalizedRegret(regret, maxRegret, pNorm, cNorm)
			s.Require().NoError(err)
			weights[i] = weight
		}

		// Check that weights are monotonically increasing (better regret => higher weight)
		for i := 1; i < len(weights); i++ {
			s.Require().True(weights[i].Gte(weights[i-1]),
				"Weight should increase with better regret: regret[%d]=%s gave weight=%s, regret[%d]=%s gave weight=%s",
				i-1, regrets[i-1], weights[i-1].String(), i, regrets[i], weights[i].String())
		}
	})

	s.Run("Positivity: all weights should be positive", func() {
		testCases := []struct{ regret, maxRegret string }{
			{"-10", "-5"},
			{"-1", "1"},
			{"0", "2"},
			{"1", "1"},
			{"5", "-2"}, // Even when regret > max
		}

		for _, tc := range testCases {
			regret := alloraMath.MustNewDecFromString(tc.regret)
			maxRegret := alloraMath.MustNewDecFromString(tc.maxRegret)
			weight, err := inferencesynthesis.CalcWeightFromNormalizedRegret(regret, maxRegret, pNorm, cNorm)
			s.Require().NoError(err)
			s.Require().True(weight.IsPositive(),
				"Weight should be positive for regret=%s, maxRegret=%s, got %s",
				tc.regret, tc.maxRegret, weight.String())
		}
	})
}

func (s *InferenceSynthesisTestSuite) TestCalcWeightFromNormalizedRegret_Exp1DivExp1_ErrorCases() {
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	s.Run("NaN inputs should return error", func() {
		nan := alloraMath.NewNaN()
		regret := alloraMath.MustNewDecFromString("1.0")
		maxRegret := alloraMath.MustNewDecFromString("1.0")

		// Test NaN regret
		_, err := inferencesynthesis.CalcWeightFromNormalizedRegret(nan, maxRegret, pNorm, cNorm)
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "NaN")

		// Test NaN maxRegret
		_, err = inferencesynthesis.CalcWeightFromNormalizedRegret(regret, nan, pNorm, cNorm)
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "NaN")

		// Test NaN pNorm
		_, err = inferencesynthesis.CalcWeightFromNormalizedRegret(regret, maxRegret, nan, cNorm)
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "NaN")

		// Test NaN cNorm
		_, err = inferencesynthesis.CalcWeightFromNormalizedRegret(regret, maxRegret, pNorm, nan)
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "NaN")
	})
}

func (s *InferenceSynthesisTestSuite) TestIncreasingPNormIncreasesRegretSpread() {
	cNorm := alloraMath.MustNewDecFromString("0.75")

	testCases := []struct {
		regretFrac string
		maxRegret  string
	}{
		{"-24.5", "-24.5"},
		{"-20.0", "-19.5"},
		{"-15.0", "-14.0"},
		{"-10.5", "-10.0"},
		{"-5.75", "-5.0"},
		{"-1.0", "-0.5"},
		{"-0.25", "0.0"},
		{"0.0", "0.0"},
		{"0.5", "0.5"},
		{"1.0", "1.0"},
		{"-1.32345", "0.1238729"},
		{"-0.8712641", "-0.8712641"},
		{"0.01987392", "0.01987392"},
	}

	weightWithPNorm2_point_5 := make([]alloraMath.Dec, len(testCases))
	weightWithPNorm4_point_5 := make([]alloraMath.Dec, len(testCases))

	for i, tc := range testCases {
		regretFrac := alloraMath.MustNewDecFromString(tc.regretFrac)
		maxRegret := alloraMath.MustNewDecFromString(tc.maxRegret)
		pNorm2_point_5 := alloraMath.MustNewDecFromString("2.5")
		pNorm4_point_5 := alloraMath.MustNewDecFromString("4.5")

		weght2_point_5, err := inferencesynthesis.CalcWeightFromNormalizedRegret(regretFrac, maxRegret, pNorm2_point_5, cNorm)
		s.Require().NoError(err)
		weightWithPNorm2_point_5[i] = weght2_point_5

		weght4_point_5, err := inferencesynthesis.CalcWeightFromNormalizedRegret(regretFrac, maxRegret, pNorm4_point_5, cNorm)
		s.Require().NoError(err)
		weightWithPNorm4_point_5[i] = weght4_point_5
	}

	stdDev2_point_5, err := alloraMath.StdDev(weightWithPNorm2_point_5)
	s.Require().NoError(err)

	stdDev4_point_5, err := alloraMath.StdDev(weightWithPNorm4_point_5)
	s.Require().NoError(err)

	s.Require().True(stdDev2_point_5.Lt(stdDev4_point_5))
}

func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesTwoWorkersOneForecaster() {
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.5")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	topicId := uint64(1)

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("3")},
					{Inferer: s.AddrsStr(1), Value: alloraMath.MustNewDecFromString("4")},
				},
			},
		},
	}

	expected := map[string]*emissionstypes.Inference{
		"forecaster0": {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("1.063129668414686004388530315977163")}},
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0":     {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("1")}},
		s.AddrsStr(1): {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("2")}},
	}

	allInferersAreNew := false
	inferers := []string{"worker0", s.AddrsStr(1)}
	forecasters := []string{"forecaster0"}
	forecastByWorker := map[string]*emissionstypes.Forecast{"forecaster0": forecasts.Forecasts[0]}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		"worker0":     &zero,
		s.AddrsStr(1): &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{

		"forecaster0": &zero,
	}

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	_, err = s.TopicKeeper().RegisterEpochLabel(s.Ctx(), topic.Id, 1, "y")
	s.Require().NoError(err)

	registry, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, 1)
	s.Require().NoError(err)

	result, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:                 s.Ctx().Logger(),
			TopicId:                topicId,
			TopicArity:             emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			AllInferersAreNew:      allInferersAreNew,
			Inferers:               inferers,
			InfererToInference:     inferenceByWorker,
			InfererToRegret:        infererRegrets,
			Forecasters:            forecasters,
			ForecasterToForecast:   forecastByWorker,
			ForecasterToRegret:     forecasterRegrets,
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           epsilon,
			PNorm:                  pNorm,
			CNorm:                  cNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry:          &registry,
			NumLabels:              len(registry.GetLabels()),
		},
	)
	s.Require().NoError(err)

	for key, expectedValue := range expected {
		actualValue, exists := result[key]
		s.Require().True(exists, "Expected key does not exist in result map")
		inDelta, err := alloraMath.InDelta(
			expectedValue.Values[0],
			actualValue.Values[0],
			alloraMath.MustNewDecFromString("0.0001"),
		)
		s.Require().NoError(err)
		s.Require().True(
			inDelta, "Values do not match for key: %s %s %s",
			key,
			expectedValue.Values[0].String(),
			actualValue.Values[0].String(),
		)
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesTwoWorkersTwoForecastersWithoutSelfReport() {
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.5")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "worker0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: s.AddrsStr(1), Value: alloraMath.MustNewDecFromString("4")},
				},
			},
		},
	}

	expected := map[string]*emissionstypes.Inference{
		"worker0": {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("2")}},
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0":     {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("1")}},
		s.AddrsStr(1): {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("2")}},
	}

	topicId := uint64(1)
	allInferersAreNew := false
	inferers := []string{"worker0", s.AddrsStr(1)}
	forecasters := []string{"worker0"}
	forecastByWorker := map[string]*emissionstypes.Forecast{"worker0": forecasts.Forecasts[0]}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		"worker0":     &zero,
		s.AddrsStr(1): &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{
		"worker0": &zero,
	}

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	_, err = s.TopicKeeper().RegisterEpochLabel(s.Ctx(), topic.Id, 1, "y")
	s.Require().NoError(err)

	registry, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, 1)
	s.Require().NoError(err)

	result, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:                 s.Ctx().Logger(),
			TopicId:                topicId,
			TopicArity:             emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			AllInferersAreNew:      allInferersAreNew,
			Inferers:               inferers,
			InfererToInference:     inferenceByWorker,
			InfererToRegret:        infererRegrets,
			Forecasters:            forecasters,
			ForecasterToForecast:   forecastByWorker,
			ForecasterToRegret:     forecasterRegrets,
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           epsilon,
			PNorm:                  pNorm,
			CNorm:                  cNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry:          &registry,
			NumLabels:              len(registry.GetLabels()),
		},
	)
	s.Require().NoError(err)

	for key, expectedValue := range expected {
		actualValue, exists := result[key]
		s.Require().True(exists, "Expected key does not exist in result map")
		inDelta, err := alloraMath.InDelta(
			expectedValue.Values[0],
			actualValue.Values[0],
			alloraMath.MustNewDecFromString("0.00001"),
		)
		s.Require().NoError(err)
		s.Require().True(
			inDelta, "Values do not match for key: %s %s %s",
			key,
			expectedValue.Values[0].String(),
			actualValue.Values[0].String(),
		)
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesThreeWorkersThreeForecastersWithoutSelfReport() {
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.5")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "worker0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("1")},
					{Inferer: s.AddrsStr(1), Value: alloraMath.MustNewDecFromString("2")},
					{Inferer: s.AddrsStr(2), Value: alloraMath.MustNewDecFromString("3")},
				},
			},
			{
				Forecaster: s.AddrsStr(1),
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("4")},
					{Inferer: s.AddrsStr(1), Value: alloraMath.MustNewDecFromString("5")},
					{Inferer: s.AddrsStr(2), Value: alloraMath.MustNewDecFromString("6")},
				},
			},
		},
	}

	expected := map[string]*emissionstypes.Inference{
		"worker0":     {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("1.317025398914864082848526530373150")}},
		s.AddrsStr(1): {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("1.2974778440227044")}},
		s.AddrsStr(2): nil,
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0":     {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("1")}},
		s.AddrsStr(1): {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("2")}},
		s.AddrsStr(2): {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("3")}},
	}

	topicId := uint64(1)
	allInferersAreNew := false
	inferers := []string{"worker0", s.AddrsStr(1), s.AddrsStr(2)}
	forecasters := []string{"worker0", s.AddrsStr(1)}
	forecastByWorker := map[string]*emissionstypes.Forecast{
		"worker0":     forecasts.Forecasts[0],
		s.AddrsStr(1): forecasts.Forecasts[1],
	}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		"worker0":     &zero,
		s.AddrsStr(1): &zero,
		s.AddrsStr(2): &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{
		"worker0":     &zero,
		s.AddrsStr(1): &zero,
	}

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	_, err = s.TopicKeeper().RegisterEpochLabel(s.Ctx(), topic.Id, 1, "y")
	s.Require().NoError(err)

	registry, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, 1)
	s.Require().NoError(err)

	result, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:                 s.Ctx().Logger(),
			TopicId:                topicId,
			TopicArity:             emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			AllInferersAreNew:      allInferersAreNew,
			Inferers:               inferers,
			InfererToInference:     inferenceByWorker,
			InfererToRegret:        infererRegrets,
			Forecasters:            forecasters,
			ForecasterToForecast:   forecastByWorker,
			ForecasterToRegret:     forecasterRegrets,
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           epsilon,
			PNorm:                  pNorm,
			CNorm:                  cNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry:          &registry,
			NumLabels:              len(registry.GetLabels()),
		},
	)
	s.Require().NoError(err)

	for key, expectedValue := range expected {
		actualValue, exists := result[key]

		if expectedValue == nil {
			s.Require().False(exists, "Expected key %v exist unexpectedly in result map", key)
			s.Require().Nil(actualValue, "Expected key %v to be nil", key)
		} else {
			s.Require().True(exists, "Expected key %v does not exist in result map", key)
			inDelta, err := alloraMath.InDelta(
				expectedValue.Values[0],
				actualValue.Values[0],
				alloraMath.MustNewDecFromString("0.0001"),
			)
			s.Require().NoError(err)
			s.Require().True(
				inDelta, "Values do not match for key: %s %s %s",
				key,
				expectedValue.Values[0].String(),
				actualValue.Values[0].String(),
			)
		}
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcForcastImpliedInferencesEpoch2() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch2Get := epochGet[302]

	worker0 := s.AddrsStr(0)
	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)
	worker3 := s.AddrsStr(3)
	worker4 := s.AddrsStr(4)
	forecaster0 := s.AddrsStr(5)

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: forecaster0,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker0, Value: epoch2Get("forecasted_loss_0_for_0")},
					{Inferer: worker1, Value: epoch2Get("forecasted_loss_0_for_1")},
					{Inferer: worker2, Value: epoch2Get("forecasted_loss_0_for_2")},
					{Inferer: worker3, Value: epoch2Get("forecasted_loss_0_for_3")},
					{Inferer: worker4, Value: epoch2Get("forecasted_loss_0_for_4")},
				},
			},
		},
	}
	networkCombinedLoss := epoch2Get("network_loss")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expected := map[string]*emissionstypes.Inference{
		forecaster0: {Values: []alloraMath.Dec{epoch2Get("forecast_implied_inference_0")}},
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		worker0: {Values: []alloraMath.Dec{epoch2Get("inference_0")}},
		worker1: {Values: []alloraMath.Dec{epoch2Get("inference_1")}},
		worker2: {Values: []alloraMath.Dec{epoch2Get("inference_2")}},
		worker3: {Values: []alloraMath.Dec{epoch2Get("inference_3")}},
		worker4: {Values: []alloraMath.Dec{epoch2Get("inference_4")}},
	}

	topicId := uint64(1)
	allInferersAreNew := false
	inferers := []string{worker0, worker1, worker2, worker3, worker4}
	forecasters := []string{forecaster0}
	forecastByWorker := map[string]*emissionstypes.Forecast{forecaster0: forecasts.Forecasts[0]}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		worker0: &zero,
		worker1: &zero,
		worker2: &zero,
		worker3: &zero,
		worker4: &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{
		forecaster0: &zero,
	}

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	_, err = s.TopicKeeper().RegisterEpochLabel(s.Ctx(), topic.Id, 1, "y")
	s.Require().NoError(err)

	registry, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, 1)
	s.Require().NoError(err)

	result, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:                 s.Ctx().Logger(),
			TopicId:                topicId,
			TopicArity:             emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			AllInferersAreNew:      allInferersAreNew,
			Inferers:               inferers,
			InfererToInference:     inferenceByWorker,
			InfererToRegret:        infererRegrets,
			Forecasters:            forecasters,
			ForecasterToForecast:   forecastByWorker,
			ForecasterToRegret:     forecasterRegrets,
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           epsilon,
			PNorm:                  pNorm,
			CNorm:                  cNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry:          &registry,
			NumLabels:              len(registry.GetLabels()),
		})
	s.Require().NoError(err)
	for key, expectedValue := range expected {
		actualValue, exists := result[key]
		s.Require().True(exists, "Expected key does not exist in result map")
		inDelta, err := alloraMath.InDelta(
			expectedValue.Values[0],
			actualValue.Values[0],
			alloraMath.MustNewDecFromString("0.001"),
		)
		s.Require().NoError(err)
		s.Require().True(
			inDelta, "Values do not match for key: %s %s %s",
			key,
			expectedValue.Values[0].String(),
			actualValue.Values[0].String(),
		)
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcForcastImpliedInferencesEpoch3() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch3Get := epochGet[303]

	worker0 := s.AddrsStr(0)
	worker1 := s.AddrsStr(1)
	worker2 := s.AddrsStr(2)
	worker3 := s.AddrsStr(3)
	worker4 := s.AddrsStr(4)
	forecaster0 := s.AddrsStr(5)

	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: forecaster0,
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: worker0, Value: epoch3Get("forecasted_loss_0_for_0")},
					{Inferer: worker1, Value: epoch3Get("forecasted_loss_0_for_1")},
					{Inferer: worker2, Value: epoch3Get("forecasted_loss_0_for_2")},
					{Inferer: worker3, Value: epoch3Get("forecasted_loss_0_for_3")},
					{Inferer: worker4, Value: epoch3Get("forecasted_loss_0_for_4")},
				},
			},
		},
	}

	networkCombinedLoss := epoch3Get("network_loss")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("3.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	expected := map[string]*emissionstypes.Inference{
		forecaster0: {Values: []alloraMath.Dec{epoch3Get("forecast_implied_inference_0")}},
	}
	inferenceByWorker := map[string]*emissionstypes.Inference{
		worker0: {Values: []alloraMath.Dec{epoch3Get("inference_0")}},
		worker1: {Values: []alloraMath.Dec{epoch3Get("inference_1")}},
		worker2: {Values: []alloraMath.Dec{epoch3Get("inference_2")}},
		worker3: {Values: []alloraMath.Dec{epoch3Get("inference_3")}},
		worker4: {Values: []alloraMath.Dec{epoch3Get("inference_4")}},
	}
	topicId := uint64(1)
	allInferersAreNew := false
	inferers := []string{worker0, worker1, worker2, worker3, worker4}
	forecasters := []string{forecaster0}
	forecastByWorker := map[string]*emissionstypes.Forecast{forecaster0: forecasts.Forecasts[0]}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		worker0: &zero,
		worker1: &zero,
		worker2: &zero,
		worker3: &zero,
		worker4: &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{
		forecaster0: &zero,
	}

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	_, err = s.TopicKeeper().RegisterEpochLabel(s.Ctx(), topic.Id, 1, "y")
	s.Require().NoError(err)

	registry, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, 1)
	s.Require().NoError(err)

	result, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:                 s.Ctx().Logger(),
			TopicId:                topicId,
			TopicArity:             emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			AllInferersAreNew:      allInferersAreNew,
			Inferers:               inferers,
			InfererToInference:     inferenceByWorker,
			InfererToRegret:        infererRegrets,
			Forecasters:            forecasters,
			ForecasterToForecast:   forecastByWorker,
			ForecasterToRegret:     forecasterRegrets,
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           epsilon,
			PNorm:                  pNorm,
			CNorm:                  cNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry:          &registry,
			NumLabels:              len(registry.GetLabels()),
		})
	s.Require().NoError(err)
	for key, expectedValue := range expected {
		actualValue, exists := result[key]
		s.Require().True(exists, "Expected key does not exist in result map")
		inDelta, err := alloraMath.InDelta(
			expectedValue.Values[0],
			actualValue.Values[0],
			alloraMath.MustNewDecFromString("0.01"),
		)
		s.Require().NoError(err)
		s.Require().True(
			inDelta, "Values do not match for key: %s %s %s",
			key,
			expectedValue.Values[0].String(),
			actualValue.Values[0].String(),
		)
	}
}

func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesForecasterWithNoMatchingInferences() {
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.5")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	topicId := uint64(1)

	// Forecaster provides forecasts for inferers that don't exist in InfererToInference
	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "nonexistent_worker1", Value: alloraMath.MustNewDecFromString("3")},
					{Inferer: "nonexistent_worker2", Value: alloraMath.MustNewDecFromString("4")},
				},
			},
		},
	}

	// Only one valid inferer exists
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0": {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("1")}},
	}

	allInferersAreNew := false
	inferers := []string{"worker0"}
	forecasters := []string{"forecaster0"}
	forecastByWorker := map[string]*emissionstypes.Forecast{"forecaster0": forecasts.Forecasts[0]}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		"worker0": &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{
		"forecaster0": &zero,
	}

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	_, err = s.TopicKeeper().RegisterEpochLabel(s.Ctx(), topic.Id, 1, "y")
	s.Require().NoError(err)

	registry, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, 1)
	s.Require().NoError(err)

	result, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:                 s.Ctx().Logger(),
			TopicId:                topicId,
			TopicArity:             emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			AllInferersAreNew:      allInferersAreNew,
			Inferers:               inferers,
			InfererToInference:     inferenceByWorker,
			InfererToRegret:        infererRegrets,
			Forecasters:            forecasters,
			ForecasterToForecast:   forecastByWorker,
			ForecasterToRegret:     forecasterRegrets,
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           epsilon,
			PNorm:                  pNorm,
			CNorm:                  cNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry:          &registry,
			NumLabels:              len(registry.GetLabels()),
		},
	)
	s.Require().NoError(err)

	// Should return empty result since no forecasts match existing inferences
	s.Require().Empty(result, "Expected empty result when no forecasts match existing inferences")
}

func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesForecasterWithZeroValidForecasts() {
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.5")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	topicId := uint64(1)

	// Forecaster has empty forecast elements
	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster:       "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{},
			},
		},
	}

	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0":     {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("1")}},
		s.AddrsStr(1): {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("2")}},
	}

	allInferersAreNew := false
	inferers := []string{"worker0", s.AddrsStr(1)}
	forecasters := []string{"forecaster0"}
	forecastByWorker := map[string]*emissionstypes.Forecast{"forecaster0": forecasts.Forecasts[0]}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		"worker0":     &zero,
		s.AddrsStr(1): &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{
		"forecaster0": &zero,
	}

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	_, err = s.TopicKeeper().RegisterEpochLabel(s.Ctx(), topic.Id, 1, "y")
	s.Require().NoError(err)

	registry, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, 1)
	s.Require().NoError(err)

	result, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:                 s.Ctx().Logger(),
			TopicId:                topicId,
			TopicArity:             emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			AllInferersAreNew:      allInferersAreNew,
			Inferers:               inferers,
			InfererToInference:     inferenceByWorker,
			InfererToRegret:        infererRegrets,
			Forecasters:            forecasters,
			ForecasterToForecast:   forecastByWorker,
			ForecasterToRegret:     forecasterRegrets,
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           epsilon,
			PNorm:                  pNorm,
			CNorm:                  cNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry:          &registry,
			NumLabels:              len(registry.GetLabels()),
		},
	)
	s.Require().NoError(err)

	// Should return empty result since forecaster has no valid forecasts
	s.Require().Empty(result, "Expected empty result when forecaster has zero valid forecasts")
}

func (s *InferenceSynthesisTestSuite) TestCalcForecastImpliedInferencesMultipleForecastersPartialCoverage() {
	networkCombinedLoss := alloraMath.MustNewDecFromString("0.5")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	pNorm := alloraMath.MustNewDecFromString("2.0")
	cNorm := alloraMath.MustNewDecFromString("0.75")
	topicId := uint64(1)

	// Multiple forecasters with different coverage patterns
	forecasts := &emissionstypes.Forecasts{
		Forecasts: []*emissionstypes.Forecast{
			{
				Forecaster: "forecaster0",
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("1.5")},
					{Inferer: s.AddrsStr(1), Value: alloraMath.MustNewDecFromString("2.5")},
					// Missing forecast for worker2
				},
			},
			{
				Forecaster: s.AddrsStr(1),
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "worker0", Value: alloraMath.MustNewDecFromString("1.8")},
					{Inferer: s.AddrsStr(2), Value: alloraMath.MustNewDecFromString("3.2")},
					// Missing forecast for worker1
				},
			},
			{
				Forecaster: s.AddrsStr(2),
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: s.AddrsStr(1), Value: alloraMath.MustNewDecFromString("2.1")},
					{Inferer: s.AddrsStr(2), Value: alloraMath.MustNewDecFromString("3.0")},
					// Missing forecast for worker0
				},
			},
			{
				Forecaster: s.AddrsStr(3),
				ForecastElements: []*emissionstypes.ForecastElement{
					{Inferer: "nonexistent_worker1", Value: alloraMath.MustNewDecFromString("4.0")},
					{Inferer: "nonexistent_worker2", Value: alloraMath.MustNewDecFromString("5.0")},
					// This forecaster doesn't hit any active inferers
				},
			},
		},
	}

	// All active inferers have inferences
	inferenceByWorker := map[string]*emissionstypes.Inference{
		"worker0":     {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("1")}},
		s.AddrsStr(1): {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("2")}},
		s.AddrsStr(2): {Values: []alloraMath.Dec{alloraMath.MustNewDecFromString("3")}},
	}

	allInferersAreNew := false
	inferers := []string{"worker0", s.AddrsStr(1), s.AddrsStr(2)}
	forecasters := []string{"forecaster0", s.AddrsStr(1), s.AddrsStr(2), s.AddrsStr(3)}
	forecastByWorker := map[string]*emissionstypes.Forecast{
		"forecaster0": forecasts.Forecasts[0],
		s.AddrsStr(1): forecasts.Forecasts[1],
		s.AddrsStr(2): forecasts.Forecasts[2],
		s.AddrsStr(3): forecasts.Forecasts[3],
	}
	zero := alloraMath.ZeroDec()
	infererRegrets := map[string]*alloraMath.Dec{
		"worker0":     &zero,
		s.AddrsStr(1): &zero,
		s.AddrsStr(2): &zero,
	}
	forecasterRegrets := map[string]*alloraMath.Dec{
		"forecaster0": &zero,
		s.AddrsStr(1): &zero,
		s.AddrsStr(2): &zero,
		s.AddrsStr(3): &zero,
	}

	topic, err := s.TopicKeeper().GetTopic(s.Ctx(), topicId)
	s.Require().NoError(err)
	_, err = s.TopicKeeper().RegisterEpochLabel(s.Ctx(), topic.Id, 1, "y")
	s.Require().NoError(err)

	registry, err := s.TopicKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, 1)
	s.Require().NoError(err)

	result, err := inferencesynthesis.CalcForecastImpliedInferences(
		inferencesynthesis.CalcForecastImpliedInferencesArgs{
			Logger:                 s.Ctx().Logger(),
			TopicId:                topicId,
			TopicArity:             emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			AllInferersAreNew:      allInferersAreNew,
			Inferers:               inferers,
			InfererToInference:     inferenceByWorker,
			InfererToRegret:        infererRegrets,
			Forecasters:            forecasters,
			ForecasterToForecast:   forecastByWorker,
			ForecasterToRegret:     forecasterRegrets,
			NetworkCombinedLoss:    &networkCombinedLoss,
			EpsilonTopic:           epsilon,
			PNorm:                  pNorm,
			CNorm:                  cNorm,
			RegretScalePlusEpsilon: alloraMath.ZeroDec(),
			LabelRegistry:          &registry,
			NumLabels:              len(registry.GetLabels()),
		},
	)
	s.Require().NoError(err)

	// Verify that only forecasters with valid forecasts have results
	s.Require().Len(result, 3, "Expected results for only 3 forecasters (excluding the one with no valid forecasts)")

	// Verify each forecaster with valid forecasts has a result
	s.Require().Contains(result, "forecaster0", "Expected forecaster0 to have a result")
	s.Require().Contains(result, s.AddrsStr(1), "Expected forecaster1 to have a result")
	s.Require().Contains(result, s.AddrsStr(2), "Expected forecaster2 to have a result")

	// Verify that forecaster3 (with no valid forecasts) is NOT in results
	s.Require().NotContains(result, s.AddrsStr(3), "Expected forecaster3 to NOT have a result since it has no valid forecasts")

	// Verify that each forecaster's result is a valid inference
	for forecaster, inference := range result {
		s.Require().NotNil(inference, "Expected inference to not be nil for forecaster: %s", forecaster)
		s.Require().Equal(topicId, inference.TopicId, "Expected correct topic ID for forecaster: %s", forecaster)
		s.Require().Equal(forecaster, inference.Inferer, "Expected correct inferer for forecaster: %s", forecaster)
		s.Require().True(inference.Values[0].IsPositive(), "Expected positive value for forecaster: %s", forecaster)
	}

	// Verify that the function handled missing forecasts gracefully
	// Each forecaster should have processed only the inferers they forecasted for
	// and ignored the ones they didn't forecast for
	s.Require().True(result["forecaster0"].Values[0].IsPositive(), "forecaster0 should have valid result despite missing worker2")
	s.Require().True(result[s.AddrsStr(1)].Values[0].IsPositive(), "forecaster1 should have valid result despite missing worker1")
	s.Require().True(result[s.AddrsStr(2)].Values[0].IsPositive(), "forecaster2 should have valid result despite missing worker0")
}
