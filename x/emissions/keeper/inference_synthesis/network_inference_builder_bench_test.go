package inferencesynthesis_test

import (
	"fmt"
	"math/rand"
	"testing"

	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	inferencesynthesis "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// buildOneOutForecastImpliedArgs builds keeper-independent arguments for
// GetOneOutInfererForecastImpliedInferences. The function reads only the provided
// slices, maps, and scalars, so a bare logger and zero-value keeper are sufficient.
// forecastSize controls how many distinct inferers each forecaster forecasts
// (sparse when much smaller than numInferers, dense when equal). All values are
// derived deterministically from seed.
func buildOneOutForecastImpliedArgs(
	numForecasters, numInferers, numLabels, forecastSize int,
	seed int64,
) inferencesynthesis.GetOneOutInfererForecastImpliedInferencesArgs {
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic fixture values, not crypto

	inferers := make([]string, numInferers)
	infererToInference := make(map[string]*emissionstypes.Inference, numInferers)
	infererToRegret := make(map[string]*alloraMath.Dec, numInferers)
	for i := 0; i < numInferers; i++ {
		inferer := fmt.Sprintf("inferer%05d", i)
		inferers[i] = inferer

		values := make([]alloraMath.Dec, numLabels)
		for l := 0; l < numLabels; l++ {
			values[l] = alloraMath.MustNewDecFromString(
				fmt.Sprintf("%d.%06d", 100+rng.Intn(400), rng.Intn(1000000)))
		}
		infererToInference[inferer] = &emissionstypes.Inference{
			TopicId:     1,
			BlockHeight: 100,
			Inferer:     inferer,
			ExtraData:   nil,
			Proof:       "",
			Values:      values,
		}
		regret := alloraMath.MustNewDecFromString(
			fmt.Sprintf("%d.%06d", rng.Intn(10), rng.Intn(1000000)))
		infererToRegret[inferer] = &regret
	}

	if forecastSize > numInferers {
		forecastSize = numInferers
	}

	forecasters := make([]string, numForecasters)
	forecasterToForecast := make(map[string]*emissionstypes.Forecast, numForecasters)
	forecasterToRegret := make(map[string]*alloraMath.Dec, numForecasters)
	for f := 0; f < numForecasters; f++ {
		forecaster := fmt.Sprintf("forecaster%05d", f)
		forecasters[f] = forecaster

		// Pick forecastSize distinct inferers deterministically.
		perm := rng.Perm(numInferers)
		elements := make([]*emissionstypes.ForecastElement, 0, forecastSize)
		for i := 0; i < forecastSize; i++ {
			elements = append(elements, &emissionstypes.ForecastElement{
				Inferer: inferers[perm[i]],
				Value: alloraMath.MustNewDecFromString(
					fmt.Sprintf("%d.%06d", 10+rng.Intn(90), rng.Intn(1000000))),
			})
		}
		forecasterToForecast[forecaster] = &emissionstypes.Forecast{
			TopicId:          1,
			BlockHeight:      100,
			Forecaster:       forecaster,
			ForecastElements: elements,
			ExtraData:        nil,
		}
		regret := alloraMath.MustNewDecFromString(
			fmt.Sprintf("%d.%06d", rng.Intn(10), rng.Intn(1000000)))
		forecasterToRegret[forecaster] = &regret
	}

	labels := make([]*emissionstypes.TopicLabel, numLabels)
	for l := 0; l < numLabels; l++ {
		//nolint:gosec // label ids are tiny sequential indexes, far below the uint32 range
		labels[l] = &emissionstypes.TopicLabel{Id: uint32(l + 1), Name: fmt.Sprintf("label%d", l)}
	}

	arity := emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE
	if numLabels > 1 {
		arity = emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI
	}

	// Forecast element values are forecasted inferer losses; keep them within an
	// order of magnitude of the network combined loss so normalized regrets stay
	// in the regime the end-block path actually sees (extreme spreads make apd
	// decimal addition realign coefficients across thousands of digits, which
	// would benchmark the bignum library rather than this function).
	networkCombinedLoss := alloraMath.MustNewDecFromString("100")

	return inferencesynthesis.GetOneOutInfererForecastImpliedInferencesArgs{
		// Ctx and K are never read by the function under test.
		Ctx:                    sdk.Context{},
		K:                      emissionskeeper.Keeper{},
		Logger:                 log.NewNopLogger(),
		TopicId:                1,
		TopicArity:             arity,
		Inferers:               inferers,
		InfererToInference:     infererToInference,
		InfererToRegret:        infererToRegret,
		AllInferersAreNew:      false,
		Forecasters:            forecasters,
		ForecasterToForecast:   forecasterToForecast,
		ForecasterToRegret:     forecasterToRegret,
		NetworkCombinedLoss:    &networkCombinedLoss,
		EpsilonTopic:           alloraMath.MustNewDecFromString("0.0001"),
		PNorm:                  alloraMath.MustNewDecFromString("2"),
		CNorm:                  alloraMath.MustNewDecFromString("0.75"),
		RegretScalePlusEpsilon: alloraMath.ZeroDec(),
		LabelRegistry: &emissionstypes.EpochLabelRegistry{
			TopicId: 1,
			EpochId: 1,
			Labels:  labels,
		},
		NumLabels:         numLabels,
		LabelDefaultValue: alloraMath.ZeroDec(),
	}
}

// sparseForecastSize mimics the common regime where a forecaster forecasts only a
// small subset of the inferer set (bounded by MaxElementsPerForecast, default 12).
func sparseForecastSize(numInferers int) int {
	size := numInferers / 4
	if size < 1 {
		size = 1
	}
	if size > 12 {
		size = 12
	}
	return size
}

var oneOutBenchSink []*emissionstypes.OneOutInfererForecasterValue

func BenchmarkGetOneOutInfererForecastImpliedInferences(b *testing.B) {
	sizes := []int{1, 8, 32, 128}
	patterns := []struct {
		name string
		size func(numInferers int) int
	}{
		{name: "sparse", size: sparseForecastSize},
		{name: "dense", size: func(numInferers int) int { return numInferers }},
	}

	for _, numForecasters := range sizes {
		for _, numInferers := range sizes {
			if numInferers < numForecasters {
				continue
			}
			for _, numLabels := range []int{1, 4} {
				for _, pattern := range patterns {
					forecastSize := pattern.size(numInferers)
					name := fmt.Sprintf("F=%d/I=%d/L=%d/%s(k=%d)",
						numForecasters, numInferers, numLabels, pattern.name, forecastSize)
					args := buildOneOutForecastImpliedArgs(
						numForecasters, numInferers, numLabels, forecastSize, 42)
					b.Run(name, func(b *testing.B) {
						b.ReportAllocs()
						b.ResetTimer()
						for n := 0; n < b.N; n++ {
							result, err := inferencesynthesis.GetOneOutInfererForecastImpliedInferences(args)
							if err != nil {
								b.Fatalf("unexpected error: %v", err)
							}
							oneOutBenchSink = result
						}
					})
				}
			}
		}
	}
}
