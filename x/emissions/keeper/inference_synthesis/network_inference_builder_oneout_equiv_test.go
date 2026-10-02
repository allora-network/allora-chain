package inferencesynthesis_test

import (
	"fmt"
	"math/rand"
	"testing"

	"cosmossdk.io/log"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// getOneOutInfererForecastImpliedInferencesReference is the pre-optimization
// implementation of GetOneOutInfererForecastImpliedInferences, retained here so the
// equivalence tests can assert the optimized production implementation returns
// identical output. It recomputes the full forecast-implied pipeline for every
// (forecaster, withheld inferer) pair; treat any change to it as a change to the
// equivalence claim itself.
func getOneOutInfererForecastImpliedInferencesReference(
	args inferencesynthesis.GetOneOutInfererForecastImpliedInferencesArgs,
) ([]*emissionstypes.OneOutInfererForecasterValue, error) {
	oneOutInfererForecastImpliedValues := make([]*emissionstypes.OneOutInfererForecasterValue, 0)

	// If NetworkCombinedLoss is nil, return empty slice immediately
	if args.NetworkCombinedLoss == nil {
		args.Logger.Debug("NetworkCombinedLoss is nil, returning empty one-out inferer forecast implied values", "topicId", args.TopicId)
		return oneOutInfererForecastImpliedValues, nil
	}

	// Only process if we have both inferers and forecasters
	if len(args.Inferers) <= 1 || len(args.Forecasters) == 0 {
		return oneOutInfererForecastImpliedValues, nil
	}

	if args.LabelRegistry == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "GetOneOutInfererForecastImpliedInferences: LabelRegistry is nil")
	}

	// Organize by forecaster first, then by withheld inferer
	for _, forecaster := range args.Forecasters {
		// Get this forecaster's forecast and filter out the withheld inferer
		forecast, ok := args.ForecasterToForecast[forecaster]
		if !ok {
			continue
		}

		for _, withheldInferer := range args.Inferers {
			// Filter out the inferer we want to withhold
			filteredInferers := make([]string, 0, len(args.Inferers)-1)
			filteredInfererToInference := make(map[string]*emissionstypes.Inference, len(args.InfererToInference)-1)
			filteredInfererToRegret := make(map[string]*alloraMath.Dec, len(args.InfererToRegret)-1)

			for _, inferer := range args.Inferers {
				if inferer != withheldInferer {
					filteredInferers = append(filteredInferers, inferer)
					if inference, ok := args.InfererToInference[inferer]; ok {
						filteredInfererToInference[inferer] = inference
					}
					if regret, ok := args.InfererToRegret[inferer]; ok {
						filteredInfererToRegret[inferer] = regret
					}
				}
			}

			filteredForecastElements := make([]*emissionstypes.ForecastElement, 0)
			for _, element := range forecast.ForecastElements {
				if element.Inferer != withheldInferer {
					filteredForecastElements = append(filteredForecastElements, element)
				}
			}

			if len(filteredForecastElements) == 0 {
				continue
			}

			filteredForecast := &emissionstypes.Forecast{
				TopicId:          forecast.TopicId,
				BlockHeight:      forecast.BlockHeight,
				Forecaster:       forecaster,
				ForecastElements: filteredForecastElements,
				ExtraData:        forecast.ExtraData,
			}

			// Create filtered maps with just this forecaster
			filteredForecasters := []string{forecaster}
			filteredForecasterToForecast := map[string]*emissionstypes.Forecast{
				forecaster: filteredForecast,
			}
			filteredForecasterToRegret := map[string]*alloraMath.Dec{}
			if regret, ok := args.ForecasterToRegret[forecaster]; ok {
				filteredForecasterToRegret[forecaster] = regret
			}

			// Calculate forecast-implied inference with the filtered data
			forecastImpliedInferences, calcErr := inferencesynthesis.CalcForecastImpliedInferences(
				inferencesynthesis.CalcForecastImpliedInferencesArgs{
					Logger:                 args.Logger,
					TopicId:                args.TopicId,
					TopicArity:             args.TopicArity,
					AllInferersAreNew:      args.AllInferersAreNew,
					Inferers:               filteredInferers,
					InfererToInference:     filteredInfererToInference,
					InfererToRegret:        filteredInfererToRegret,
					Forecasters:            filteredForecasters,
					ForecasterToForecast:   filteredForecasterToForecast,
					ForecasterToRegret:     filteredForecasterToRegret,
					NetworkCombinedLoss:    args.NetworkCombinedLoss,
					EpsilonTopic:           args.EpsilonTopic,
					PNorm:                  args.PNorm,
					CNorm:                  args.CNorm,
					RegretScalePlusEpsilon: args.RegretScalePlusEpsilon,
					LabelRegistry:          args.LabelRegistry,
					NumLabels:              args.NumLabels,
					LabelDefaultValue:      args.LabelDefaultValue,
				},
			)
			if calcErr != nil {
				args.Logger.Warn("Error calculating forecast implied inference for:", "forecaster", forecaster, "withheldInferer", withheldInferer, "error", calcErr)
				continue
			}

			// Extract the implied inference for this forecaster
			forecastImpliedInference, ok := forecastImpliedInferences[forecaster]
			if !ok {
				continue
			}

			// Add to our results
			oneOutInfererValues, err := emissionstypes.ConvertInferenceValuesToLabeledValues(forecastImpliedInference.Values, args.LabelRegistry)
			if err != nil {
				return nil, errorsmod.Wrap(err, "failed to convert forecast implied inference values")
			}

			oneOutInfererForecastImpliedValues = append(
				oneOutInfererForecastImpliedValues,
				&emissionstypes.OneOutInfererForecasterValue{
					Forecaster:        forecaster,
					WithheldInferer:   withheldInferer,
					CombinedInference: oneOutInfererValues,
				},
			)
		}
	}

	return oneOutInfererForecastImpliedValues, nil
}

// requireEqualOneOutForecasterValues asserts the two outputs are identical: same
// length, same (forecaster, withheld inferer) ordering, and Dec-equal values per label.
func requireEqualOneOutForecasterValues(
	t *testing.T,
	expected, actual []*emissionstypes.OneOutInfererForecasterValue,
) {
	t.Helper()
	require.Equal(t, len(expected), len(actual), "output length mismatch")
	for i := range expected {
		require.Equal(t, expected[i].Forecaster, actual[i].Forecaster,
			"forecaster mismatch at output index %d", i)
		require.Equal(t, expected[i].WithheldInferer, actual[i].WithheldInferer,
			"withheld inferer mismatch at output index %d", i)
		expectedValues := expected[i].GetCombinedInference()
		actualValues := actual[i].GetCombinedInference()
		require.Equal(t, len(expectedValues), len(actualValues),
			"combined inference length mismatch at output index %d", i)
		for j := range expectedValues {
			require.Equal(t, expectedValues[j].LabelId, actualValues[j].LabelId,
				"label id mismatch at output index %d label %d", i, j)
			require.Equal(t, expectedValues[j].LabelName, actualValues[j].LabelName,
				"label name mismatch at output index %d label %d", i, j)
			require.True(t, expectedValues[j].Value.Equal(actualValues[j].Value),
				"value mismatch at output index %d label %d: expected %s, got %s",
				i, j, expectedValues[j].Value.String(), actualValues[j].Value.String())
		}
	}
}

// oneOutArgsSpec describes a fully keeper-independent input set for
// GetOneOutInfererForecastImpliedInferences. Cases start from defaultOneOutArgsSpec
// and override only the fields they exercise.
type oneOutArgsSpec struct {
	inferers               []string
	infererValues          map[string][]string // inferer -> label values; missing key = no inference
	infererRegrets         map[string]string   // nil = no regrets at all
	allInferersAreNew      bool
	forecasters            []string
	forecasts              map[string][][2]string // forecaster -> (inferer, forecasted loss) elements
	forecasterRegrets      map[string]string
	networkCombinedLoss    *alloraMath.Dec
	regretScalePlusEpsilon *alloraMath.Dec // nil = zero (MAD-derived scale)
	nilLabelRegistry       bool
	numLabels              int
}

func defaultOneOutArgsSpec() oneOutArgsSpec {
	return oneOutArgsSpec{
		inferers:               nil,
		infererValues:          nil,
		infererRegrets:         nil,
		allInferersAreNew:      false,
		forecasters:            nil,
		forecasts:              nil,
		forecasterRegrets:      nil,
		networkCombinedLoss:    nil,
		regretScalePlusEpsilon: nil,
		nilLabelRegistry:       false,
		numLabels:              1,
	}
}

func buildOneOutArgsFromSpec(spec oneOutArgsSpec) inferencesynthesis.GetOneOutInfererForecastImpliedInferencesArgs {
	infererToInference := make(map[string]*emissionstypes.Inference, len(spec.infererValues))
	for inferer, values := range spec.infererValues {
		decValues := make([]alloraMath.Dec, len(values))
		for i, v := range values {
			decValues[i] = alloraMath.MustNewDecFromString(v)
		}
		infererToInference[inferer] = &emissionstypes.Inference{
			TopicId:     1,
			BlockHeight: 100,
			Inferer:     inferer,
			ExtraData:   nil,
			Proof:       "",
			Values:      decValues,
		}
	}

	infererToRegret := make(map[string]*alloraMath.Dec, len(spec.infererRegrets))
	for inferer, regret := range spec.infererRegrets {
		r := alloraMath.MustNewDecFromString(regret)
		infererToRegret[inferer] = &r
	}

	forecasterToForecast := make(map[string]*emissionstypes.Forecast, len(spec.forecasts))
	for forecaster, elements := range spec.forecasts {
		forecastElements := make([]*emissionstypes.ForecastElement, 0, len(elements))
		for _, element := range elements {
			forecastElements = append(forecastElements, &emissionstypes.ForecastElement{
				Inferer: element[0],
				Value:   alloraMath.MustNewDecFromString(element[1]),
			})
		}
		forecasterToForecast[forecaster] = &emissionstypes.Forecast{
			TopicId:          1,
			BlockHeight:      100,
			Forecaster:       forecaster,
			ForecastElements: forecastElements,
			ExtraData:        nil,
		}
	}

	forecasterToRegret := make(map[string]*alloraMath.Dec, len(spec.forecasterRegrets))
	for forecaster, regret := range spec.forecasterRegrets {
		r := alloraMath.MustNewDecFromString(regret)
		forecasterToRegret[forecaster] = &r
	}

	arity := emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE
	if spec.numLabels > 1 {
		arity = emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
	}

	var labelRegistry *emissionstypes.EpochLabelRegistry
	if !spec.nilLabelRegistry {
		labels := make([]*emissionstypes.TopicLabel, spec.numLabels)
		for l := 0; l < spec.numLabels; l++ {
			//nolint:gosec // label ids are tiny sequential indexes, far below the uint32 range
			labels[l] = &emissionstypes.TopicLabel{Id: uint32(l + 1), Name: fmt.Sprintf("label%d", l)}
		}
		labelRegistry = &emissionstypes.EpochLabelRegistry{TopicId: 1, EpochId: 1, Labels: labels}
	}

	regretScalePlusEpsilon := alloraMath.ZeroDec()
	if spec.regretScalePlusEpsilon != nil {
		regretScalePlusEpsilon = *spec.regretScalePlusEpsilon
	}

	return inferencesynthesis.GetOneOutInfererForecastImpliedInferencesArgs{
		// Ctx and K are never read by the function under test.
		Ctx:                    sdk.Context{},
		K:                      emissionskeeper.Keeper{},
		Logger:                 log.NewNopLogger(),
		TopicId:                1,
		TopicArity:             arity,
		Inferers:               spec.inferers,
		InfererToInference:     infererToInference,
		InfererToRegret:        infererToRegret,
		AllInferersAreNew:      spec.allInferersAreNew,
		Forecasters:            spec.forecasters,
		ForecasterToForecast:   forecasterToForecast,
		ForecasterToRegret:     forecasterToRegret,
		NetworkCombinedLoss:    spec.networkCombinedLoss,
		EpsilonTopic:           alloraMath.MustNewDecFromString("0.0001"),
		PNorm:                  alloraMath.MustNewDecFromString("2"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RegretScalePlusEpsilon: regretScalePlusEpsilon,
		LabelRegistry:          labelRegistry,
		NumLabels:              spec.numLabels,
		LabelDefaultValue:      alloraMath.ZeroDec(),
	}
}

// uniformInfererValues gives every inferer numLabels values around the given base.
func uniformInfererValues(inferers []string, numLabels, base int) map[string][]string {
	out := make(map[string][]string, len(inferers))
	for i, inferer := range inferers {
		values := make([]string, numLabels)
		for l := 0; l < numLabels; l++ {
			values[l] = fmt.Sprintf("%d", base+10*i+l)
		}
		out[inferer] = values
	}
	return out
}

func uniformInfererRegrets(inferers []string, regret string) map[string]string {
	out := make(map[string]string, len(inferers))
	for _, inferer := range inferers {
		out[inferer] = regret
	}
	return out
}

// spreadForecast assigns each forecaster the first forecastSize inferers, offset by
// the forecaster index so forecasters do not all cover the same subset.
func spreadForecast(forecasters, inferers []string, forecastSize int) map[string][][2]string {
	out := make(map[string][][2]string, len(forecasters))
	for f, forecaster := range forecasters {
		elements := make([][2]string, 0, forecastSize)
		for i := 0; i < forecastSize; i++ {
			inferer := inferers[(f+i)%len(inferers)]
			elements = append(elements, [2]string{inferer, fmt.Sprintf("%d", 10+i)})
		}
		out[forecaster] = elements
	}
	return out
}

func TestGetOneOutInfererForecastImpliedInferencesEquivalence(t *testing.T) {
	inferers4 := []string{"inferer0", "inferer1", "inferer2", "inferer3"}
	inferers6 := []string{"inferer0", "inferer1", "inferer2", "inferer3", "inferer4", "inferer5"}
	forecasters2 := []string{"forecaster0", "forecaster1"}
	forecasters3 := []string{"forecaster0", "forecaster1", "forecaster2"}

	cases := []struct {
		name   string
		mutate func(s *oneOutArgsSpec)
	}{
		{
			name: "single label sparse forecasts",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4, "0.1")
				s.forecasters = forecasters2
				s.forecasts = spreadForecast(forecasters2, inferers4, 2)
				s.forecasterRegrets = map[string]string{"forecaster0": "0.2", "forecaster1": "0.3"}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "multi label sparse forecasts",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers6
				s.infererValues = uniformInfererValues(inferers6, 3, 200)
				s.infererRegrets = uniformInfererRegrets(inferers6, "0.5")
				s.forecasters = forecasters3
				s.forecasts = spreadForecast(forecasters3, inferers6, 2)
				s.forecasterRegrets = map[string]string{"forecaster0": "0.2"}
				s.networkCombinedLoss = decPtr("500")
				s.numLabels = 3
			},
		},
		{
			name: "dense forecasts cover every inferer",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4, "0.1")
				s.forecasters = forecasters2
				s.forecasts = spreadForecast(forecasters2, inferers4, len(inferers4))
				s.forecasterRegrets = map[string]string{"forecaster0": "0.2"}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "forecaster forecasting a single inferer",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4[:3]
				s.infererValues = uniformInfererValues(inferers4[:3], 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4[:3], "0.1")
				s.forecasters = forecasters2[:1]
				s.forecasts = spreadForecast(forecasters2[:1], inferers4[:3], 1)
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "forecaster with empty forecast elements",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4, "0.1")
				s.forecasters = forecasters2
				s.forecasts = map[string][][2]string{
					"forecaster0": {},
					"forecaster1": {{"inferer0", "10"}, {"inferer1", "20"}},
				}
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "forecaster without a forecast entry",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4, "0.1")
				s.forecasters = forecasters2
				s.forecasts = map[string][][2]string{
					"forecaster1": {{"inferer0", "10"}},
				}
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "forecaster forecasting only inferers without inferences",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4, "0.1")
				s.forecasters = forecasters2[:1]
				s.forecasts = map[string][][2]string{
					"forecaster0": {{"ghost0", "10"}, {"ghost1", "20"}},
				}
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "missing inferer regrets",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = map[string]string{"inferer0": "0.1"}
				s.forecasters = forecasters2
				s.forecasts = spreadForecast(forecasters2, inferers4, 3)
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "no inferer regrets at all",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = nil
				s.forecasters = forecasters2
				s.forecasts = spreadForecast(forecasters2, inferers4, 3)
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "zero regrets everywhere",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4, "0")
				s.forecasters = forecasters2
				s.forecasts = spreadForecast(forecasters2, inferers4, 2)
				s.forecasterRegrets = map[string]string{"forecaster0": "0", "forecaster1": "0"}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "pinned regret scale",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4, "0.1")
				s.forecasters = forecasters2
				s.forecasts = spreadForecast(forecasters2, inferers4, 2)
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
				s.regretScalePlusEpsilon = decPtr("0.5")
			},
		},
		{
			name: "all inferers are new uses the median path",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers6
				s.infererValues = uniformInfererValues(inferers6, 1, 100)
				s.infererRegrets = nil
				s.allInferersAreNew = true
				s.forecasters = forecasters3
				s.forecasts = spreadForecast(forecasters3, inferers6, 3)
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "all inferers are new multi label",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 2, 100)
				s.infererRegrets = nil
				s.allInferersAreNew = true
				s.forecasters = forecasters2
				s.forecasts = spreadForecast(forecasters2, inferers4, 3)
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
				s.numLabels = 2
			},
		},
		{
			name: "single inferer early return",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4[:1]
				s.infererValues = uniformInfererValues(inferers4[:1], 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4[:1], "0.1")
				s.forecasters = forecasters2
				s.forecasts = spreadForecast(forecasters2, inferers4, 2)
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "zero forecasters",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4, "0.1")
				s.forecasters = []string{}
				s.forecasts = map[string][][2]string{}
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "nil network combined loss",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4, "0.1")
				s.forecasters = forecasters2
				s.forecasts = spreadForecast(forecasters2, inferers4, 2)
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = nil
			},
		},
		{
			name: "duplicate forecast elements for one inferer",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4, "0.1")
				s.forecasters = forecasters2[:1]
				s.forecasts = map[string][][2]string{
					"forecaster0": {{"inferer0", "10"}, {"inferer0", "15"}, {"inferer2", "20"}},
				}
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "inference registered for an unlisted inferer",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4[:3]
				values := uniformInfererValues(inferers4[:3], 1, 100)
				values["stranger"] = []string{"999"}
				s.infererValues = values
				s.infererRegrets = uniformInfererRegrets(inferers4[:3], "0.1")
				s.forecasters = forecasters2
				s.forecasts = spreadForecast(forecasters2, inferers4[:3], 2)
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "inference missing for a listed inferer",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				values := uniformInfererValues(inferers4, 1, 100)
				delete(values, "inferer1")
				s.infererValues = values
				s.infererRegrets = uniformInfererRegrets(inferers4, "0.1")
				s.forecasters = forecasters2
				s.forecasts = spreadForecast(forecasters2, inferers4, 3)
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "extreme regret spread with zero weights",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4, "0.1")
				s.forecasters = forecasters2
				s.forecasts = spreadForecast(forecasters2, inferers4, 3)
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100000000")
			},
		},
		{
			name: "zero labels makes every pair warn and skip",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4, "0.1")
				s.forecasters = forecasters2
				s.forecasts = spreadForecast(forecasters2, inferers4, 2)
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
				s.numLabels = 0
			},
		},
		{
			name: "duplicate inferer entries in the inferer list",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = []string{"inferer0", "inferer1", "inferer1", "inferer2"}
				s.infererValues = uniformInfererValues(inferers4[:3], 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4[:3], "0.1")
				s.forecasters = forecasters2[:1]
				s.forecasts = spreadForecast(forecasters2[:1], inferers4[:3], 2)
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
			},
		},
		{
			name: "nil label registry",
			mutate: func(s *oneOutArgsSpec) {
				s.inferers = inferers4
				s.infererValues = uniformInfererValues(inferers4, 1, 100)
				s.infererRegrets = uniformInfererRegrets(inferers4, "0.1")
				s.forecasters = forecasters2
				s.forecasts = spreadForecast(forecasters2, inferers4, 2)
				s.forecasterRegrets = map[string]string{}
				s.networkCombinedLoss = decPtr("100")
				s.nilLabelRegistry = true
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := defaultOneOutArgsSpec()
			tc.mutate(&spec)
			args := buildOneOutArgsFromSpec(spec)
			expected, expectedErr := getOneOutInfererForecastImpliedInferencesReference(args)
			actual, actualErr := inferencesynthesis.GetOneOutInfererForecastImpliedInferences(args)
			require.Equal(t, expectedErr == nil, actualErr == nil,
				"error presence mismatch: reference=%v optimized=%v", expectedErr, actualErr)
			if expectedErr != nil {
				require.Equal(t, expectedErr.Error(), actualErr.Error(), "error message mismatch")
				require.Nil(t, expected)
				require.Nil(t, actual)
				return
			}
			requireEqualOneOutForecasterValues(t, expected, actual)
		})
	}
}

func TestGetOneOutInfererForecastImpliedInferencesEquivalenceRandomized(t *testing.T) {
	rng := rand.New(rand.NewSource(20260913)) //nolint:gosec // deterministic test fixtures, not crypto

	for trial := 0; trial < 50; trial++ {
		numLabels := 1 + rng.Intn(4)
		numInferers := 2 + rng.Intn(9)
		numForecasters := 1 + rng.Intn(4)

		inferers := make([]string, numInferers)
		for i := range inferers {
			inferers[i] = fmt.Sprintf("inferer%02d", i)
		}
		forecasters := make([]string, numForecasters)
		for f := range forecasters {
			forecasters[f] = fmt.Sprintf("forecaster%02d", f)
		}

		spec := defaultOneOutArgsSpec()
		spec.numLabels = numLabels

		spec.infererValues = make(map[string][]string, numInferers)
		for i, inferer := range inferers {
			if rng.Intn(10) == 0 {
				// Some listed inferers have no inference.
				continue
			}
			values := make([]string, numLabels)
			for l := 0; l < numLabels; l++ {
				values[l] = fmt.Sprintf("%d.%06d", 1+rng.Intn(1000)+100*i, rng.Intn(1000000))
			}
			spec.infererValues[inferer] = values
		}

		if rng.Intn(4) != 0 {
			spec.infererRegrets = make(map[string]string, numInferers)
			for _, inferer := range inferers {
				switch rng.Intn(5) {
				case 0:
					spec.infererRegrets[inferer] = "0"
				case 1:
					// Missing regret entry.
				default:
					spec.infererRegrets[inferer] = fmt.Sprintf("%d.%06d", rng.Intn(5), rng.Intn(1000000))
				}
			}
		}

		spec.forecasts = make(map[string][][2]string, numForecasters)
		for _, forecaster := range forecasters {
			forecastSize := rng.Intn(numInferers + 1)
			if rng.Intn(6) == 0 {
				forecastSize = numInferers // dense case: every inferer participates
			}
			perm := rng.Perm(numInferers)
			elements := make([][2]string, 0, forecastSize)
			for i := 0; i < forecastSize; i++ {
				elements = append(elements, [2]string{
					inferers[perm[i]],
					fmt.Sprintf("%d.%06d", rng.Intn(100), rng.Intn(1000000)),
				})
			}
			spec.forecasts[forecaster] = elements
		}
		if rng.Intn(6) == 0 {
			// A forecaster without any forecast entry.
			delete(spec.forecasts, forecasters[0])
		}

		spec.forecasterRegrets = make(map[string]string, numForecasters)
		for _, forecaster := range forecasters {
			if rng.Intn(2) == 0 {
				spec.forecasterRegrets[forecaster] = fmt.Sprintf("%d.%06d", rng.Intn(3), rng.Intn(1000000))
			}
		}

		spec.inferers = inferers
		spec.forecasters = forecasters
		spec.allInferersAreNew = rng.Intn(4) == 0
		spec.networkCombinedLoss = decPtr(fmt.Sprintf("%d.%06d", 100+rng.Intn(900), rng.Intn(1000000)))
		if rng.Intn(3) == 0 {
			spec.regretScalePlusEpsilon = decPtr(fmt.Sprintf("0.%06d", 1+rng.Intn(999999)))
		}

		args := buildOneOutArgsFromSpec(spec)
		expected, expectedErr := getOneOutInfererForecastImpliedInferencesReference(args)
		actual, actualErr := inferencesynthesis.GetOneOutInfererForecastImpliedInferences(args)
		require.Equal(t, expectedErr == nil, actualErr == nil,
			"trial %d: error presence mismatch: reference=%v optimized=%v", trial, expectedErr, actualErr)
		if expectedErr != nil {
			require.Equal(t, expectedErr.Error(), actualErr.Error(), "trial %d: error message mismatch", trial)
			continue
		}
		requireEqualOneOutForecasterValues(t, expected, actual)
	}
}

// calcInvocationCountingLogger counts how many times CalcForecastImpliedInferences is
// entered, which it detects via the debug log line that function emits on entry.
type calcInvocationCountingLogger struct {
	log.Logger
	count *int
}

func (l calcInvocationCountingLogger) Debug(msg string, keyVals ...interface{}) {
	if msg == "Calculating forecast-implied inferences" {
		(*l.count)++
	}
	l.Logger.Debug(msg, keyVals...)
}

// TestGetOneOutInfererForecastImpliedInferencesInvocationCount pins down how many
// times each implementation enters CalcForecastImpliedInferences: the reference
// implementation enters it once per (forecaster, withheld inferer) pair, while the
// optimized implementation enters it once per forecast participant plus at most one
// base computation per forecaster.
func TestGetOneOutInfererForecastImpliedInferencesInvocationCount(t *testing.T) {
	run := func(args inferencesynthesis.GetOneOutInfererForecastImpliedInferencesArgs) (int, int) {
		referenceCount := 0
		args.Logger = calcInvocationCountingLogger{Logger: log.NewNopLogger(), count: &referenceCount}
		expected, err := getOneOutInfererForecastImpliedInferencesReference(args)
		require.NoError(t, err)

		optimizedCount := 0
		args.Logger = calcInvocationCountingLogger{Logger: log.NewNopLogger(), count: &optimizedCount}
		actual, err := inferencesynthesis.GetOneOutInfererForecastImpliedInferences(args)
		require.NoError(t, err)

		requireEqualOneOutForecasterValues(t, expected, actual)
		return referenceCount, optimizedCount
	}

	t.Run("dense: every inferer participates so the base is never computed", func(t *testing.T) {
		args := buildOneOutForecastImpliedArgs(2, 4, 1, 4, 7)
		referenceCount, optimizedCount := run(args)
		require.Equal(t, 2*4, referenceCount)
		require.Equal(t, 2*4, optimizedCount)
	})

	t.Run("sparse: one base plus one recompute per forecast participant", func(t *testing.T) {
		args := buildOneOutForecastImpliedArgs(2, 8, 1, 2, 7)
		referenceCount, optimizedCount := run(args)
		require.Equal(t, 2*8, referenceCount)
		require.Equal(t, 2*(2+1), optimizedCount)
	})

	t.Run("forecast over inferers without inferences emits no entries", func(t *testing.T) {
		args := buildOneOutForecastImpliedArgs(1, 4, 1, 2, 7)
		// Remove the inferences of the inferers this forecaster forecasted: the
		// withheld inferers still participate (their forecast elements are filtered
		// out per pair) but no forecast element has a matching inference, so the
		// pipeline produces no entry for any pair.
		for _, forecast := range args.ForecasterToForecast {
			for _, element := range forecast.ForecastElements {
				delete(args.InfererToInference, element.Inferer)
			}
		}
		referenceCount, optimizedCount := run(args)
		require.Equal(t, 1*4, referenceCount)
		require.Equal(t, 1*(2+1), optimizedCount)
	})
}
