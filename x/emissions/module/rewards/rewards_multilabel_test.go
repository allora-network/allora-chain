//nolint:gosec,exhaustruct
package rewards_test

import (
	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	testutil2 "github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (s *RewardsTestSuite) TestMultiLabelRewardsPipeline() {
	require := s.Require()

	s.MintTokensToModule(types.AlloraStakingAccountName, cosmosMath.NewInt(10000000000))

	workerIndexes := testutil.ReturnIndexes(5, 5)
	reputerIndexes := testutil.ReturnIndexes(0, 5)

	numLabels := 3
	workerValues := testutil.GetWorkerMultiValuesFromIndexes(
		workerIndexes,
		numLabels,
		"0.1", "0.2", "0.3",
	)

	// PRE PASS
	topicId, firstInferenceNonce := s.FullTopicPass(
		workerIndexes,
		reputerIndexes,
		testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI),
		testutil.WithWorkerValues(workerValues),
	)

	topic, err := s.EmissionsKeeper().GetTopic(s.Ctx(), topicId)
	require.NoError(err)
	require.Equal(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI, topic.OutputArity)

	firstRegistry, err := s.EmissionsKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, firstInferenceNonce)
	require.NoError(err)
	require.Len(firstRegistry.Labels, numLabels)

	expectedLabelNames := make([]string, numLabels)
	for i, lbl := range firstRegistry.Labels {
		require.NotNil(lbl)
		expectedLabelNames[i] = lbl.Name
		require.NotEmpty(lbl.Name)
		require.Equal(uint32(i+1), lbl.Id)
	}

	reputerStakeBefore := make(map[int]cosmosMath.Int)
	for _, idx := range reputerIndexes {
		stake, err := s.EmissionsKeeper().GetStakeReputerAuthority(s.Ctx(), topicId, s.AddrsStr(idx))
		require.NoError(err)
		reputerStakeBefore[idx] = stake
	}

	workerBalanceBefore := make(map[int]sdk.Coin)
	for _, idx := range workerIndexes {
		workerBalanceBefore[idx] = s.BankKeeper().GetBalance(s.Ctx(), s.Addrs(idx), params.DefaultBondDenom)
	}

	// SECOND PASS
	_, secondInferenceNonce := s.FullTopicPass(
		workerIndexes,
		reputerIndexes,
		testutil.WithTopicID(topicId),
		testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI),
		testutil.WithWorkerValues(workerValues),
	)

	// Rewards should still be distributed.
	for _, idx := range reputerIndexes {
		currentStake, err := s.EmissionsKeeper().GetStakeReputerAuthority(s.Ctx(), topicId, s.AddrsStr(idx))
		require.NoError(err)
		require.True(
			currentStake.GT(reputerStakeBefore[idx]),
			"reputer %s stake did not increase: before=%s after=%s",
			s.AddrsStr(idx),
			reputerStakeBefore[idx].String(),
			currentStake.String(),
		)
	}

	for _, idx := range workerIndexes {
		currentBalance := s.BankKeeper().GetBalance(s.Ctx(), s.Addrs(idx), params.DefaultBondDenom)
		require.True(
			currentBalance.Amount.GT(workerBalanceBefore[idx].Amount),
			"worker %s balance did not increase: before=%s after=%s",
			s.AddrsStr(idx),
			workerBalanceBefore[idx].Amount.String(),
			currentBalance.Amount.String(),
		)
	}

	// Network inference bundle should exist and be truly multi-label.
	bundle, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topicId, secondInferenceNonce)
	require.NoError(err)
	require.NotNil(bundle)

	require.Equal(topicId, bundle.TopicId)
	require.Equal(secondInferenceNonce, bundle.Nonce)

	assertLabeledValues(require, bundle.CombinedValue, numLabels, expectedLabelNames, "combined value")
	assertLabeledValues(require, bundle.NaiveValue, numLabels, expectedLabelNames, "naive value")

	assertWorkerInferences(
		require,
		bundle.InfererValues,
		len(workerIndexes),
		numLabels,
		expectedLabelNames,
		"inferer values",
	)

	assertWorkerInferences(
		require,
		bundle.ForecasterValues,
		len(workerIndexes),
		numLabels,
		expectedLabelNames,
		"forecaster values",
	)

	assertOneOutInfererValues(
		require,
		bundle.OneOutInfererValues,
		numLabels,
		expectedLabelNames,
		"one-out inferer values",
	)

	assertOneOutForecasterValues(
		require,
		bundle.OneOutForecasterValues,
		numLabels,
		expectedLabelNames,
		"one-out forecaster values",
	)

	assertOneInForecasterValues(
		require,
		bundle.OneInForecasterValues,
		numLabels,
		expectedLabelNames,
		"one-in forecaster values",
	)

	assertOneOutInfererForecasterValues(
		require,
		bundle.OneOutInfererForecasterValues,
		numLabels,
		expectedLabelNames,
		"one-out inferer forecaster values",
	)

	// Registry for the second inference nonce should also have the full label set.
	secondRegistry, err := s.EmissionsKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, secondInferenceNonce)
	require.NoError(err)
	require.Len(secondRegistry.Labels, numLabels)

	for i, lbl := range secondRegistry.Labels {
		require.NotNil(lbl)
		require.Equal(expectedLabelNames[i], lbl.Name)
		require.Equal(uint32(i+1), lbl.Id)
	}

	// Spot-check that inferer submissions were aligned to registry size.
	activeInferences, err := s.EmissionsKeeper().GetInferencesAtBlock(s.Ctx(), topicId, secondInferenceNonce, false)
	require.NoError(err)
	require.Len(activeInferences.Inferences, len(workerIndexes))

	for i, inf := range activeInferences.Inferences {
		require.NotNil(inf, "active inference %d is nil", i)
		require.Len(inf.Values, numLabels, "active inference %d not padded/aligned to registry size", i)
	}
}

func (s *RewardsTestSuite) TestMultiLabelLabelIndependence() {
	require := s.Require()

	workerIndexes := testutil.ReturnIndexes(5, 3)
	reputerIndexes := testutil.ReturnIndexes(0, 3)
	numLabels := 2
	workerValues := testutil.GetWorkerMultiValuesFromIndexes(
		workerIndexes,
		numLabels,
		"0.1", "0.9",
	)
	workerValues[2].Values[0].Value = "0.9"
	workerValues[2].Values[1].Value = "0.1"

	findLabelValue := func(vals []*types.LabeledValue, label string) alloraMath.Dec {
		for _, v := range vals {
			if v != nil && v.LabelName == label {
				return v.Value
			}
		}
		s.T().Fatalf("label %q not found", label)
		return alloraMath.Dec{}
	}

	topicId, inferenceNonce := s.FullTopicPass(
		workerIndexes,
		reputerIndexes,
		testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI),
		testutil.WithWorkerValues(workerValues),
	)

	topic, err := s.EmissionsKeeper().GetTopic(s.Ctx(), topicId)
	require.NoError(err)
	require.Equal(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI, topic.OutputArity)

	registry, err := s.EmissionsKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, inferenceNonce)
	require.NoError(err)
	require.Len(registry.Labels, 2)
	require.Equal("L0", registry.Labels[0].Name)
	require.Equal("L1", registry.Labels[1].Name)

	bundle, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topicId, inferenceNonce)
	require.NoError(err)
	require.NotNil(bundle)

	require.Len(bundle.CombinedValue, 2)
	require.Len(bundle.NaiveValue, 2)

	combinedL0 := findLabelValue(bundle.CombinedValue, "L0")
	combinedL1 := findLabelValue(bundle.CombinedValue, "L1")
	naiveL0 := findLabelValue(bundle.NaiveValue, "L0")
	naiveL1 := findLabelValue(bundle.NaiveValue, "L1")

	// Expected per-label averages:
	// L0 = (0.1 + 0.1 + 0.9) / 3 = 1.1 / 3
	// L1 = (0.9 + 0.9 + 0.1) / 3 = 1.9 / 3
	expectedL0, err := alloraMath.MustNewDecFromString("1.1").Quo(alloraMath.MustNewDecFromString("3"))
	require.NoError(err)
	expectedL1, err := alloraMath.MustNewDecFromString("1.9").Quo(alloraMath.MustNewDecFromString("3"))
	require.NoError(err)

	testutil2.InEpsilon5(s.T(), combinedL0, expectedL0.String())
	testutil2.InEpsilon5(s.T(), combinedL1, expectedL1.String())

	// On the first pass, naive and combined should match.
	testutil2.InEpsilon5(s.T(), naiveL0, expectedL0.String())
	testutil2.InEpsilon5(s.T(), naiveL1, expectedL1.String())

	// Core independence check: the two labels should not collapse to the same value.
	require.False(combinedL0.Equal(combinedL1))
	require.True(combinedL0.Lt(combinedL1), "L0 should stay below L1 if labels are aggregated independently")

	// Inferer values should remain aligned to the same 2-label space.
	require.Len(bundle.InfererValues, len(workerIndexes))
	for i, wi := range bundle.InfererValues {
		require.NotNil(wi, "inferer value %d is nil", i)
		require.Len(wi.Values, 2)
		require.Equal("L0", wi.Values[0].LabelName)
		require.Equal("L1", wi.Values[1].LabelName)
	}
}

func (s *RewardsTestSuite) TestMultiLabelPermutationInvariance() {
	require := s.Require()

	reputerIndexes := testutil.ReturnIndexes(0, 3)

	baseWorkerIndexes := []int{5, 6, 7}
	baseWorkerValues := []testutil.TestWorkerValue{
		{
			Index: baseWorkerIndexes[0],
			Values: []testutil.TestLabeledValue{
				{Label: "A", Value: "0.1"},
				{Label: "B", Value: "0.9"},
				{Label: "C", Value: "0.2"},
			},
		},
		{
			Index: baseWorkerIndexes[1],
			Values: []testutil.TestLabeledValue{
				{Label: "A", Value: "0.3"},
				{Label: "B", Value: "0.7"},
				{Label: "C", Value: "0.4"},
			},
		},
		{
			Index: baseWorkerIndexes[2],
			Values: []testutil.TestLabeledValue{
				{Label: "A", Value: "0.5"},
				{Label: "B", Value: "0.5"},
				{Label: "C", Value: "0.6"},
			},
		},
	}

	// - worker order is permuted
	// - label order inside each worker is permuted
	permutedWorkerIndexes := []int{7, 5, 6}
	permutedWorkerValues := []testutil.TestWorkerValue{
		{
			Index: permutedWorkerIndexes[0],
			Values: []testutil.TestLabeledValue{
				{Label: "C", Value: "0.6"},
				{Label: "A", Value: "0.5"},
				{Label: "B", Value: "0.5"},
			},
		},
		{
			Index: permutedWorkerIndexes[1],
			Values: []testutil.TestLabeledValue{
				{Label: "B", Value: "0.9"},
				{Label: "C", Value: "0.2"},
				{Label: "A", Value: "0.1"},
			},
		},
		{
			Index: permutedWorkerIndexes[2],
			Values: []testutil.TestLabeledValue{
				{Label: "C", Value: "0.4"},
				{Label: "B", Value: "0.7"},
				{Label: "A", Value: "0.3"},
			},
		},
	}

	// Run 1
	topicID1, nonce1 := s.FullTopicPass(
		baseWorkerIndexes,
		reputerIndexes,
		testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI),
		testutil.WithWorkerValues(baseWorkerValues),
	)

	bundle1, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topicID1, nonce1)
	require.NoError(err)
	require.NotNil(bundle1)

	combined1 := labeledValuesToMap(bundle1.CombinedValue)
	naive1 := labeledValuesToMap(bundle1.NaiveValue)

	s.SetupTest()

	// Run 2
	topicID2, nonce2 := s.FullTopicPass(
		permutedWorkerIndexes,
		reputerIndexes,
		testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI),
		testutil.WithWorkerValues(permutedWorkerValues),
	)

	bundle2, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topicID2, nonce2)
	require.NoError(err)
	require.NotNil(bundle2)

	combined2 := labeledValuesToMap(bundle2.CombinedValue)
	naive2 := labeledValuesToMap(bundle2.NaiveValue)

	labels := []string{"A", "B", "C"}

	for _, label := range labels {
		testutil2.InEpsilon5(s.T(), combined1[label], combined2[label].String())
		testutil2.InEpsilon5(s.T(), naive1[label], naive2[label].String())
	}
}

func (s *RewardsTestSuite) TestMultiLabelEpochLabelSpaceConstruction() {
	require := s.Require()

	testCases := []struct {
		name              string
		workerValues      []testutil.TestWorkerValue
		wantRegistryNames []string
		wantAlignedByAddr func(workerAddrs []string) map[string][]string
	}{
		{
			name: "disjoint_labels_build_union_space",
			workerValues: []testutil.TestWorkerValue{
				{Values: []testutil.TestLabeledValue{{Label: "A", Value: "0.1"}}},
				{Values: []testutil.TestLabeledValue{{Label: "B", Value: "0.2"}}},
				{Values: []testutil.TestLabeledValue{{Label: "C", Value: "0.3"}}},
			},
			wantRegistryNames: []string{"A", "B", "C"},
			wantAlignedByAddr: func(workerAddrs []string) map[string][]string {
				return map[string][]string{
					workerAddrs[0]: {"0.1", "0", "0"},
					workerAddrs[1]: {"0", "0.2", "0"},
					workerAddrs[2]: {"0", "0", "0.3"},
				}
			},
		},
		{
			name: "overlapping_labels_build_union_space",
			workerValues: []testutil.TestWorkerValue{
				{Values: []testutil.TestLabeledValue{{Label: "A", Value: "0.1"}, {Label: "B", Value: "0.2"}}},
				{Values: []testutil.TestLabeledValue{{Label: "B", Value: "0.3"}, {Label: "C", Value: "0.4"}}},
				{Values: []testutil.TestLabeledValue{{Label: "A", Value: "0.5"}, {Label: "C", Value: "0.6"}}},
			},
			wantRegistryNames: []string{"A", "B", "C"},
			wantAlignedByAddr: func(workerAddrs []string) map[string][]string {
				return map[string][]string{
					workerAddrs[0]: {"0.1", "0.2", "0"},
					workerAddrs[1]: {"0", "0.3", "0.4"},
					workerAddrs[2]: {"0.5", "0", "0.6"},
				}
			},
		},
		{
			name: "mixed_overlap_with_common_label",
			workerValues: []testutil.TestWorkerValue{
				{Values: []testutil.TestLabeledValue{{Label: "X", Value: "0.11"}, {Label: "Y", Value: "0.22"}}},
				{Values: []testutil.TestLabeledValue{{Label: "Y", Value: "0.33"}, {Label: "Z", Value: "0.44"}}},
				{Values: []testutil.TestLabeledValue{{Label: "Y", Value: "0.55"}}},
			},
			wantRegistryNames: []string{"X", "Y", "Z"},
			wantAlignedByAddr: func(workerAddrs []string) map[string][]string {
				return map[string][]string{
					workerAddrs[0]: {"0.11", "0.22", "0"},
					workerAddrs[1]: {"0", "0.33", "0.44"},
					workerAddrs[2]: {"0", "0.55", "0"},
				}
			},
		},
		{
			name: "full_plus_partial_workers",
			workerValues: []testutil.TestWorkerValue{
				{Values: []testutil.TestLabeledValue{{Label: "L0", Value: "0.1"}, {Label: "L1", Value: "0.2"}, {Label: "L2", Value: "0.3"}}},
				{Values: []testutil.TestLabeledValue{{Label: "L1", Value: "0.4"}}},
				{Values: []testutil.TestLabeledValue{{Label: "L2", Value: "0.5"}}},
			},
			wantRegistryNames: []string{"L0", "L1", "L2"},
			wantAlignedByAddr: func(workerAddrs []string) map[string][]string {
				return map[string][]string{
					workerAddrs[0]: {"0.1", "0.2", "0.3"},
					workerAddrs[1]: {"0", "0.4", "0"},
					workerAddrs[2]: {"0", "0", "0.5"},
				}
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			workerIndexes := testutil.ReturnIndexes(5, 3)
			reputerIndexes := testutil.ReturnIndexes(0, 3)

			workerAddrs := make([]string, len(workerIndexes))
			for i, idx := range workerIndexes {
				workerAddrs[i] = s.AddrsStr(idx)
				tc.workerValues[i].Index = idx
			}

			// pass 1
			topicId, firstNonce := s.FullTopicPass(
				workerIndexes,
				reputerIndexes,
				testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI),
				testutil.WithWorkerValues(tc.workerValues),
			)

			firstRegistry, err := s.EmissionsKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, firstNonce)
			require.NoError(err)
			assertRegistryLabelNames(require, &firstRegistry, tc.wantRegistryNames)

			firstBundle, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topicId, firstNonce)
			require.NoError(err)
			require.NotNil(firstBundle)

			require.Equal(topicId, firstBundle.TopicId)
			require.Equal(firstNonce, firstBundle.Nonce)

			assertLabeledValuesMatchNames(require, firstBundle.CombinedValue, tc.wantRegistryNames, "first pass combined value")
			assertLabeledValuesMatchNames(require, firstBundle.NaiveValue, tc.wantRegistryNames, "first pass naive value")

			expectedAligned := tc.wantAlignedByAddr(workerAddrs)

			require.Len(firstBundle.InfererValues, len(workerIndexes))
			assertWorkerInferenceBundleValuesMatchExpected(
				require,
				firstBundle.InfererValues,
				tc.wantRegistryNames,
				expectedAligned,
			)

			activeInferences, err := s.EmissionsKeeper().GetInferencesAtBlock(s.Ctx(), topicId, firstNonce, false)
			require.NoError(err)
			assertActiveInferencesMatchExpected(require, activeInferences, expectedAligned)

			// exact sparse-vector math assertion must be checked on first pass:
			// missing labels are treated as zero in the aligned vectors.
			expectedNaive := computeExpectedNaiveByLabel(require, tc.workerValues, tc.wantRegistryNames)
			gotFirstNaive := labeledValuesToMap(firstBundle.NaiveValue)
			gotFirstCombined := labeledValuesToMap(firstBundle.CombinedValue)

			for _, label := range tc.wantRegistryNames {
				want := expectedNaive[label]
				testutil2.InEpsilon5(s.T(), gotFirstNaive[label], want.String())
				testutil2.InEpsilon5(s.T(), gotFirstCombined[label], want.String())
			}

			// pass 2
			_, secondNonce := s.FullTopicPass(
				workerIndexes,
				reputerIndexes,
				testutil.WithTopicID(topicId),
				testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI),
				testutil.WithWorkerValues(tc.workerValues),
			)

			secondRegistry, err := s.EmissionsKeeper().GetEpochLabelRegistry(s.Ctx(), topicId, secondNonce)
			require.NoError(err)
			assertRegistryLabelNames(require, &secondRegistry, tc.wantRegistryNames)

			bundle, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topicId, secondNonce)
			require.NoError(err)
			require.NotNil(bundle)

			require.Equal(topicId, bundle.TopicId)
			require.Equal(secondNonce, bundle.Nonce)

			assertLabeledValuesMatchNames(require, bundle.CombinedValue, tc.wantRegistryNames, "second pass combined value")
			assertLabeledValuesMatchNames(require, bundle.NaiveValue, tc.wantRegistryNames, "second pass naive value")

			require.Len(bundle.InfererValues, len(workerIndexes))
			assertWorkerInferenceBundleValuesMatchExpected(
				require,
				bundle.InfererValues,
				tc.wantRegistryNames,
				expectedAligned,
			)

			activeInferencesSecond, err := s.EmissionsKeeper().GetInferencesAtBlock(s.Ctx(), topicId, secondNonce, false)
			require.NoError(err)
			assertActiveInferencesMatchExpected(require, activeInferencesSecond, expectedAligned)

			// second-pass sanity only: combined values should exist and be finite for every label
			gotCombined := labeledValuesToMap(bundle.CombinedValue)
			for _, label := range tc.wantRegistryNames {
				v := gotCombined[label]
				require.False(v.IsNaN(), "combined value for label %s is NaN", label)
				require.True(v.IsFinite(), "combined value for label %s is not finite", label)
			}
		})
	}
}

func (s *RewardsTestSuite) TestMultiLabelWorkerInfluence() {
	require := s.Require()

	workerIndexes := testutil.ReturnIndexes(5, 3)
	reputerIndexes := testutil.ReturnIndexes(0, 3)

	workerValues := []testutil.TestWorkerValue{
		{
			Index: workerIndexes[0],
			Values: []testutil.TestLabeledValue{
				{Label: "A", Value: "0.1"},
			},
		},
		{
			Index: workerIndexes[1],
			Values: []testutil.TestLabeledValue{
				{Label: "B", Value: "0.2"},
			},
		},
		{
			Index: workerIndexes[2],
			Values: []testutil.TestLabeledValue{
				{Label: "A", Value: "0.3"},
			},
		},
	}

	// -------- pre-run --------

	topicId, nonce := s.FullTopicPass(
		workerIndexes,
		reputerIndexes,
		testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI),
		testutil.WithWorkerValues(workerValues),
	)

	bundle, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topicId, nonce)
	require.NoError(err)

	baseline := labeledValuesToMap(bundle.CombinedValue)

	// -------- modify worker2 drastically --------

	workerValues[1].Values[0].Value = "100"

	s.SetupTest()

	for i := range workerValues {
		workerValues[i].Index = workerIndexes[i]
	}

	topicId2, nonce2 := s.FullTopicPass(
		workerIndexes,
		reputerIndexes,
		testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI),
		testutil.WithWorkerValues(workerValues),
	)

	bundle2, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topicId2, nonce2)
	require.NoError(err)

	modified := labeledValuesToMap(bundle2.CombinedValue)

	// Label A should not change (worker2 never contributed to A)
	testutil2.InEpsilon5(
		s.T(),
		modified["A"],
		baseline["A"].String(),
	)

	// Label B must change significantly
	require.NotEqual(
		baseline["B"].String(),
		modified["B"].String(),
	)
}

func (s *RewardsTestSuite) TestMultiLabelNaiveEqualsCombinedWhenEqualWeights() {
	require := s.Require()

	workerIndexes := testutil.ReturnIndexes(5, 4)
	reputerIndexes := testutil.ReturnIndexes(0, 4)

	workerValues := []testutil.TestWorkerValue{
		{
			Index: workerIndexes[0],
			Values: []testutil.TestLabeledValue{
				{Label: "A", Value: "0.1"},
				{Label: "B", Value: "0.2"},
			},
		},
		{
			Index: workerIndexes[1],
			Values: []testutil.TestLabeledValue{
				{Label: "A", Value: "0.3"},
				{Label: "B", Value: "0.4"},
			},
		},
		{
			Index: workerIndexes[2],
			Values: []testutil.TestLabeledValue{
				{Label: "A", Value: "0.5"},
				{Label: "B", Value: "0.6"},
			},
		},
		{
			Index: workerIndexes[3],
			Values: []testutil.TestLabeledValue{
				{Label: "A", Value: "0.7"},
				{Label: "B", Value: "0.8"},
			},
		},
	}

	topicId, firstNonce := s.FullTopicPass(
		workerIndexes,
		reputerIndexes,
		testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI),
		testutil.WithWorkerValues(workerValues),
	)

	bundle, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topicId, firstNonce)
	require.NoError(err)
	require.NotNil(bundle)

	require.Len(bundle.CombinedValue, len(bundle.NaiveValue))

	for i := range bundle.CombinedValue {
		combined := bundle.CombinedValue[i].Value
		naive := bundle.NaiveValue[i].Value

		testutil2.InEpsilon5(
			s.T(),
			combined,
			naive.String(),
		)
	}
}

func (s *RewardsTestSuite) TestMultiLabelMatchesSingleLabelPerLabel() {
	require := s.Require()

	workerIndexes := testutil.ReturnIndexes(5, 3)
	reputerIndexes := testutil.ReturnIndexes(0, 3)

	// --- MULTI LABEL RUN ---

	multiValues := []testutil.TestWorkerValue{
		{
			Index: workerIndexes[0],
			Values: []testutil.TestLabeledValue{
				{Label: "A", Value: "0.1"},
				{Label: "B", Value: "0.5"},
			},
		},
		{
			Index: workerIndexes[1],
			Values: []testutil.TestLabeledValue{
				{Label: "A", Value: "0.2"},
				{Label: "B", Value: "0.6"},
			},
		},
		{
			Index: workerIndexes[2],
			Values: []testutil.TestLabeledValue{
				{Label: "A", Value: "0.3"},
				{Label: "B", Value: "0.7"},
			},
		},
	}

	topicMulti, nonceMulti := s.FullTopicPass(
		workerIndexes,
		reputerIndexes,
		testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI),
		testutil.WithWorkerValues(multiValues),
	)

	bundleMulti, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topicMulti, nonceMulti)
	require.NoError(err)

	multiCombined := labeledValuesToMap(bundleMulti.CombinedValue)

	// --- SINGLE LABEL RUNS ---

	getSingleResult := func(label string) alloraMath.Dec {

		s.SetupTest()

		values := make([]testutil.TestWorkerValue, len(workerIndexes))

		for i := range workerIndexes {
			for _, lv := range multiValues[i].Values {
				if lv.Label == label {
					values[i] = testutil.TestWorkerValue{
						Index: workerIndexes[i],
						Values: []testutil.TestLabeledValue{
							{
								Label: "",
								Value: lv.Value,
							},
						},
					}
				}
			}
		}

		topic, nonce := s.FullTopicPass(
			workerIndexes,
			reputerIndexes,
			testutil.WithOutputArity(types.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE),
			testutil.WithWorkerValues(values),
		)

		bundle, err := s.EmissionsKeeper().GetNetworkInferences(s.Ctx(), topic, nonce)
		require.NoError(err)

		return bundle.CombinedValue[0].Value
	}

	singleA := getSingleResult("A")
	singleB := getSingleResult("B")

	testutil2.InEpsilon5(
		s.T(),
		multiCombined["A"],
		singleA.String(),
	)

	testutil2.InEpsilon5(
		s.T(),
		multiCombined["B"],
		singleB.String(),
	)
}

func assertLabeledValues(
	require *require.Assertions,
	vals []*types.LabeledValue,
	wantLen int,
	wantNames []string,
	msg string,
) {
	require.Len(vals, wantLen, "%s: wrong labeled value count", msg)

	for i, v := range vals {
		require.NotNil(v, "%s: labeled value %d is nil", msg, i)
		require.Equal(uint32(i+1), v.LabelId, "%s: wrong label id at idx=%d", msg, i)
		require.Equal(wantNames[i], v.LabelName, "%s: wrong label name at idx=%d", msg, i)
	}
}

func assertWorkerInferences(
	require *require.Assertions,
	vals []*types.WorkerInference,
	wantWorkers int,
	wantLabels int,
	wantNames []string,
	msg string,
) {
	require.Len(vals, wantWorkers, "%s: wrong worker inference count", msg)

	for i, wi := range vals {
		require.NotNil(wi, "%s: worker inference %d is nil", msg, i)
		require.NotEmpty(wi.Worker, "%s: worker inference %d has empty worker", msg, i)
		assertLabeledValues(require, wi.Values, wantLabels, wantNames, msg)
	}
}

func assertOneOutInfererValues(
	require *require.Assertions,
	vals []*types.OneOutInfererValue,
	wantLabels int,
	wantNames []string,
	msg string,
) {
	for i, v := range vals {
		require.NotNil(v, "%s: one-out inferer value %d is nil", msg, i)
		require.NotEmpty(v.WithheldInferer, "%s: one-out inferer value %d has empty withheld inferer", msg, i)
		assertLabeledValues(require, v.CombinedInference, wantLabels, wantNames, msg)
	}
}

func assertOneOutForecasterValues(
	require *require.Assertions,
	vals []*types.OneOutForecasterValue,
	wantLabels int,
	wantNames []string,
	msg string,
) {
	for i, v := range vals {
		require.NotNil(v, "%s: one-out forecaster value %d is nil", msg, i)
		require.NotEmpty(v.WithheldForecaster, "%s: one-out forecaster value %d has empty withheld forecaster", msg, i)
		assertLabeledValues(require, v.CombinedInference, wantLabels, wantNames, msg)
	}
}

func assertOneInForecasterValues(
	require *require.Assertions,
	vals []*types.OneInForecasterValue,
	wantLabels int,
	wantNames []string,
	msg string,
) {
	for i, v := range vals {
		require.NotNil(v, "%s: one-in forecaster value %d is nil", msg, i)
		require.NotEmpty(v.Forecaster, "%s: one-in forecaster value %d has empty forecaster", msg, i)
		assertLabeledValues(require, v.CombinedInference, wantLabels, wantNames, msg)
	}
}

func assertOneOutInfererForecasterValues(
	require *require.Assertions,
	vals []*types.OneOutInfererForecasterValue,
	wantLabels int,
	wantNames []string,
	msg string,
) {
	for i, v := range vals {
		require.NotNil(v, "%s: one-out inferer forecaster value %d is nil", msg, i)
		require.NotEmpty(v.Forecaster, "%s: one-out inferer forecaster value %d has empty forecaster", msg, i)
		require.NotEmpty(v.WithheldInferer, "%s: one-out inferer forecaster value %d has empty withheld inferer", msg, i)
		assertLabeledValues(require, v.CombinedInference, wantLabels, wantNames, msg)
	}
}

func assertRegistryLabelNames(
	require *require.Assertions,
	registry *types.EpochLabelRegistry,
	want []string,
) {
	require.NotNil(registry)
	require.Len(registry.Labels, len(want))

	for i, lbl := range registry.Labels {
		require.NotNil(lbl, "registry label %d is nil", i)
		require.Equal(uint32(i+1), lbl.Id, "wrong label id at idx=%d", i)
		require.Equal(want[i], lbl.Name, "wrong label name at idx=%d", i)
	}
}

func assertLabeledValuesMatchNames(
	require *require.Assertions,
	vals []*types.LabeledValue,
	wantNames []string,
	msg string,
) {
	require.Len(vals, len(wantNames), "%s: wrong labeled value count", msg)

	for i, v := range vals {
		require.NotNil(v, "%s: labeled value %d is nil", msg, i)
		require.Equal(uint32(i+1), v.LabelId, "%s: wrong label id at idx=%d", msg, i)
		require.Equal(wantNames[i], v.LabelName, "%s: wrong label name at idx=%d", msg, i)
	}
}

func assertActiveInferencesMatchExpected(
	require *require.Assertions,
	infs *types.Inferences,
	want map[string][]string,
) {
	require.NotNil(infs)
	require.Len(infs.Inferences, len(want))

	gotByWorker := make(map[string]*types.Inference, len(infs.Inferences))
	for _, inf := range infs.Inferences {
		require.NotNil(inf)
		gotByWorker[inf.Inferer] = inf
	}

	for worker, expectedValues := range want {
		inf, ok := gotByWorker[worker]
		require.True(ok, "missing inference for worker %s", worker)
		require.Len(inf.Values, len(expectedValues), "wrong aligned value count for worker %s", worker)

		for i, expected := range expectedValues {
			require.Equal(
				expected,
				inf.Values[i].String(),
				"worker %s wrong value at idx=%d",
				worker,
				i,
			)
		}
	}
}

func assertWorkerInferenceBundleValuesMatchExpected(
	require *require.Assertions,
	vals []*types.WorkerInference,
	wantNames []string,
	want map[string][]string,
) {
	require.Len(vals, len(want))

	gotByWorker := make(map[string]*types.WorkerInference, len(vals))
	for _, wi := range vals {
		require.NotNil(wi)
		gotByWorker[wi.Worker] = wi
	}

	for worker, expectedValues := range want {
		wi, ok := gotByWorker[worker]
		require.True(ok, "missing worker inference bundle entry for worker %s", worker)

		assertLabeledValuesMatchNames(require, wi.Values, wantNames, "worker inference values")
		require.Len(wi.Values, len(expectedValues), "wrong worker inference vector length for worker %s", worker)

		for i, expected := range expectedValues {
			require.Equal(
				expected,
				wi.Values[i].Value.String(),
				"worker inference bundle wrong value for worker %s at idx=%d",
				worker,
				i,
			)
		}
	}
}

func labeledValuesToMap(vals []*types.LabeledValue) map[string]alloraMath.Dec {
	out := make(map[string]alloraMath.Dec, len(vals))
	for _, v := range vals {
		if v != nil {
			out[v.LabelName] = v.Value
		}
	}
	return out
}

func computeExpectedNaiveByLabel(
	require *require.Assertions,
	workerValues []testutil.TestWorkerValue,
	registryNames []string,
) map[string]alloraMath.Dec {
	sumByLabel := make(map[string]alloraMath.Dec, len(registryNames))
	for _, name := range registryNames {
		sumByLabel[name] = alloraMath.ZeroDec()
	}

	for _, w := range workerValues {
		byLabel := make(map[string]alloraMath.Dec, len(w.Values))
		for _, lv := range w.Values {
			byLabel[lv.Label] = alloraMath.MustNewDecFromString(lv.Value)
		}

		for _, name := range registryNames {
			v, ok := byLabel[name]
			if !ok {
				v = alloraMath.ZeroDec()
			}
			var err error
			sumByLabel[name], err = sumByLabel[name].Add(v)
			require.NoError(err)
		}
	}

	n := alloraMath.NewDecFromInt64(int64(len(workerValues)))
	out := make(map[string]alloraMath.Dec, len(registryNames))
	for _, name := range registryNames {
		v, err := sumByLabel[name].Quo(n)
		require.NoError(err)
		out[name] = v
	}
	return out
}
