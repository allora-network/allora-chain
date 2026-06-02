package inferencesynthesis_test

import (
	"fmt"
	"sort"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// TestGetNetworkInferencesAtBlockMultilabel is the multilabel (classification)
// counterpart of TestGetNetworkInferencesAtBlock. It feeds the simulator's
// classification fixture through GetNetworkInferences and asserts the produced
// NetworkInferenceBundle against the simulator's network_*_label_<k> columns.
//
// The simulator fixture is the source of truth: the chain's inference
// synthesis must reproduce the Python simulator's per-class outputs.
// TestGetNetworkInferencesAtBlockMultilabel runs the multilabel network
// inference synthesis check against every consecutive epoch pair in the
// simulator fixture: (0,1), (1,2), ... (N-1,N). For each pair the earlier
// epoch supplies previous-epoch losses and regrets, and the later epoch
// supplies the current inferences/forecasts whose synthesised outputs are
// checked against the simulator's network_*_label_<k> columns.
//
// Each pair runs as an isolated subtest with a fresh keeper state, so state
// written for one pair (inferences, labels, regrets) cannot leak into the next.
func (s *InferenceSynthesisTestSuite) TestGetNetworkInferencesAtBlockMultilabel() {
	const labelCount = 3
	// Label names registered for the epoch. Order matters: label index k in the
	// CSV columns (network_inference_label_<k>) corresponds to the k-th label
	// registered, which RegisterEpochLabels assigns id k+1.
	labelNames := []string{"label_0", "label_1", "label_2"}

	epochGet := testutil.GetSimulatedValuesGetterForMultilabelEpochs(s.T())

	// Collect and sort the epoch ids so the pairing is deterministic regardless
	// of map iteration order.
	epochIds := make([]int, 0, len(epochGet))
	for id := range epochGet {
		epochIds = append(epochIds, id)
	}
	sort.Ints(epochIds)
	s.Require().GreaterOrEqual(len(epochIds), 2,
		"multilabel fixture must contain at least two epochs to form a pair")

	for i := 0; i+1 < len(epochIds); i++ {
		prevEpoch := epochIds[i]
		currEpoch := epochIds[i+1]

		s.Run(fmt.Sprintf("epoch_%d_%d", prevEpoch, currEpoch), func() {
			// Fresh keeper state for this pair.
			s.SetupTest()

			require := s.Require()

			epochPrevGet := epochGet[prevEpoch]
			epochCurrGet := epochGet[currEpoch]
			require.NotNil(epochPrevGet, "missing previous epoch fixture")
			require.NotNil(epochCurrGet, "missing current epoch fixture")

			topicId := uint64(1)
			blockHeight := int64(300)
			blockHeightPreviousLosses := int64(200)

			simpleNonce := emissionstypes.Nonce{BlockHeight: blockHeight}

			topic := s.MockTopic()
			// Multilabel topic configuration. The simulator's
			// network_inference_label_* rows sum to 1.0, so unity is required.
			topic.OutputArity = emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
			topic.RequireUnity = true
			topic.UnityTolerance = alloraMath.MustNewDecFromString("0.0001")
			err := s.TopicKeeper().SetTopic(s.Ctx(), topicId, topic)
			require.NoError(err)

			inferer0 := s.AddrsStr(0)
			inferer1 := s.AddrsStr(1)
			inferer2 := s.AddrsStr(2)
			inferer3 := s.AddrsStr(3)
			inferer4 := s.AddrsStr(4)
			infererAddresses := []string{inferer0, inferer1, inferer2, inferer3, inferer4}

			forecaster0 := s.AddrsStr(5)
			forecaster1 := s.AddrsStr(6)
			forecaster2 := s.AddrsStr(7)
			forecasterAddresses := []string{forecaster0, forecaster1, forecaster2}

			// getLabeled extracts the labelCount-long ordered value slice from a
			// []*LabeledValue, verifying it has exactly one entry per registered
			// label.
			getLabeled := func(vals []*emissionstypes.LabeledValue) []alloraMath.Dec {
				require.Len(vals, labelCount)
				out := make([]alloraMath.Dec, labelCount)
				seen := make([]bool, labelCount)
				for _, lv := range vals {
					idx := int(lv.LabelId) - 1
					require.GreaterOrEqual(idx, 0)
					require.Less(idx, labelCount)
					require.False(seen[idx], "duplicate LabelId %d in labeled values", lv.LabelId)
					seen[idx] = true
					out[idx] = lv.Value
				}
				return out
			}

			// assertLabeledEquals checks a []*LabeledValue against simulator
			// columns "<prefix>_label_<k>" for k in [0, labelCount).
			assertLabeledEquals := func(
				vals []*emissionstypes.LabeledValue,
				epochGetter func(string) alloraMath.Dec,
				prefix string,
			) {
				got := getLabeled(vals)
				for k := 0; k < labelCount; k++ {
					valueName := fmt.Sprintf("%s_label_%d", prefix, k)
					want := epochGetter(valueName)
					s.T().Logf("comparing value %s", valueName)
					testutil.InEpsilon5(s.T(), got[k], want.String())
				}
			}

			// Current-epoch inferences (multilabel) and forecasts.
			inferences, err := testutil.GetMultilabelInferencesFromCsv(
				topicId, blockHeight, infererAddresses, labelCount, epochCurrGet,
			)
			require.NoError(err)

			infererValues := s.ConvertInferencesToWorkerAttributedValues(inferences)

			// Previous network losses (scalar, unchanged shape from regression).
			lossBundlePrevious := s.mockEmptyLossBundle(epochPrevGet("network_loss"), infererValues)
			err = s.ReputerLossKeeper().InsertNetworkLossBundleAtBlock(
				s.Ctx(), topicId, blockHeightPreviousLosses, lossBundlePrevious,
			)
			require.NoError(err)

			err = s.WorkerKeeper().InsertActiveInferences(
				s.Ctx(), topicId, simpleNonce.BlockHeight, inferences,
			)
			require.NoError(err)

			// Forecasted losses are scalar even in the multilabel case, so the
			// existing single-label forecast helper is reused unchanged.
			forecasts, err := testutil.GetForecastsFromCsv(
				topicId,
				blockHeight,
				infererAddresses,
				forecasterAddresses,
				epochCurrGet,
			)
			require.NoError(err)

			// Register the class labels for this epoch. In the regression test a
			// single placeholder "y" label is registered; here we register one
			// per class so label lookups during synthesis resolve to the correct
			// class index.
			params, err := s.ParamsKeeper().GetParams(s.Ctx())
			require.NoError(err)
			_, _, err = s.TopicKeeper().RegisterEpochLabels(
				s.Ctx(),
				topicId,
				topic.LabelCaseSensitive,
				simpleNonce.BlockHeight,
				labelNames,
				params.MaxCanonicalLabelByteLength,
				params.MaxEpochLabelRegistrySize,
			)
			require.NoError(err)

			err = s.WorkerKeeper().InsertActiveForecasts(
				s.Ctx(), topicId, simpleNonce.BlockHeight, forecasts,
			)
			require.NoError(err)

			// Regrets from the previous epoch.
			err = testutil.SetRegretsFromPreviousEpoch(
				s.Ctx(),
				*s.EmissionsKeeper(),
				topicId,
				blockHeight,
				infererAddresses,
				forecasterAddresses,
				epochPrevGet,
			)
			require.NoError(err)

			// Calculate and set regretScalePlusEpsilon.
			regrets := make([]alloraMath.Dec, 0, len(infererAddresses)+len(forecasterAddresses))
			for _, inf := range infererAddresses {
				infReg, _, err := s.RegretsKeeper().GetInfererNetworkRegret(s.Ctx(), topicId, inf)
				require.NoError(err)
				regrets = append(regrets, infReg.Value)
			}
			for _, forc := range forecasterAddresses {
				forcReg, _, err := s.RegretsKeeper().GetForecasterNetworkRegret(s.Ctx(), topicId, forc)
				require.NoError(err)
				regrets = append(regrets, forcReg.Value)
			}

			regretScalePlusEpsilon, err := inferencesynthesis.CalcRegretScalePlusEpsilon(regrets, topic.Epsilon)
			require.NoError(err)

			err = s.WeightsKeeper().SetLatestRegretScale(s.Ctx(), topicId, regretScalePlusEpsilon)
			require.NoError(err)

			// Calculate.
			result, err := inferencesynthesis.GetNetworkInferences(
				s.Ctx(),
				*s.EmissionsKeeper(),
				topicId,
				&blockHeight,
				&inferences,
				&forecasts,
			)
			require.NoError(err)
			require.Equal(result.LossBlockHeight, blockHeightPreviousLosses)

			bundle := result.NetworkInferences

			// Per-inferer inferences: each worker's values must match the
			// inference_<i> columns it was seeded from.
			require.Len(bundle.InfererValues, 5)
			for _, inference := range inferences.Inferences {
				found := false
				for _, infererValue := range bundle.InfererValues {
					if inference.Inferer == infererValue.Worker {
						found = true
						require.Len(infererValue.Values, labelCount)
						got := getLabeled(infererValue.Values)
						require.Len(inference.Values, labelCount)
						for k := 0; k < labelCount; k++ {
							require.True(inference.Values[k].Equal(got[k]),
								"inferer %s label %d mismatch", inference.Inferer, k)
						}
					}
				}
				require.True(found, "Inference not found for %s", inference.Inferer)
			}

			// Combined and naive network inferences, per class.
			assertLabeledEquals(bundle.NaiveValue, epochCurrGet, "network_naive_inference")
			assertLabeledEquals(bundle.CombinedValue, epochCurrGet, "network_inference")

			// Forecast-implied inferences, per forecaster, per class.
			require.Len(bundle.ForecasterValues, 3)
			for _, forecasterValue := range bundle.ForecasterValues {
				switch forecasterValue.Worker {
				case forecaster0:
					assertLabeledEquals(forecasterValue.Values, epochCurrGet, "forecast_implied_inference_0")
				case forecaster1:
					assertLabeledEquals(forecasterValue.Values, epochCurrGet, "forecast_implied_inference_1")
				case forecaster2:
					assertLabeledEquals(forecasterValue.Values, epochCurrGet, "forecast_implied_inference_2")
				default:
					require.Fail("Unexpected forecaster", forecasterValue.Worker)
				}
			}

			// One-out inferer/forecaster combined inferences, per class.
			require.Len(bundle.OneOutInfererForecasterValues, 15)
			for _, oo := range bundle.OneOutInfererForecasterValues {
				fIdx, ok := indexOf(forecasterAddresses, oo.Forecaster)
				require.True(ok, "Unexpected forecaster %v", oo.Forecaster)
				iIdx, ok := indexOf(infererAddresses, oo.WithheldInferer)
				require.True(ok, "Unexpected withheld inferer %v", oo.WithheldInferer)
				prefix := fmt.Sprintf("forecast_implied_inference_%d_oneout_%d", fIdx, iIdx)
				assertLabeledEquals(oo.CombinedInference, epochCurrGet, prefix)
			}

			// One-out inferer network inferences, per class.
			require.Len(bundle.OneOutInfererValues, 5)
			for _, oo := range bundle.OneOutInfererValues {
				iIdx, ok := indexOf(infererAddresses, oo.WithheldInferer)
				require.True(ok, "Unexpected withheld inferer %v", oo.WithheldInferer)
				prefix := fmt.Sprintf("network_inference_oneout_%d", iIdx)
				assertLabeledEquals(oo.CombinedInference, epochCurrGet, prefix)
			}

			// One-out forecaster network inferences, per class. Forecaster
			// one-out columns continue the oneout index space after the
			// inferers (5, 6, 7).
			require.Len(bundle.OneOutForecasterValues, 3)
			for _, oo := range bundle.OneOutForecasterValues {
				fIdx, ok := indexOf(forecasterAddresses, oo.WithheldForecaster)
				require.True(ok, "Unexpected withheld forecaster %v", oo.WithheldForecaster)
				prefix := fmt.Sprintf("network_inference_oneout_%d", fIdx+len(infererAddresses))
				assertLabeledEquals(oo.CombinedInference, epochCurrGet, prefix)
			}

			// One-in forecaster naive network inferences, per class.
			require.Len(bundle.OneInForecasterValues, 3)
			for _, oi := range bundle.OneInForecasterValues {
				fIdx, ok := indexOf(forecasterAddresses, oi.Forecaster)
				require.True(ok, "Unexpected forecaster %v", oi.Forecaster)
				prefix := fmt.Sprintf("network_naive_inference_onein_%d", fIdx)
				assertLabeledEquals(oi.CombinedInference, epochCurrGet, prefix)
			}
		})
	}
}

// indexOf returns the position of target in addrs, and whether it was found.
func indexOf(addrs []string, target string) (int, bool) {
	for i, a := range addrs {
		if a == target {
			return i, true
		}
	}
	return 0, false
}
