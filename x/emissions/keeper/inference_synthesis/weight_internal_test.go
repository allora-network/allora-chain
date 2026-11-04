//nolint:exhaustruct
package inferencesynthesis

import (
	"testing"

	"cosmossdk.io/log"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

func infValues(vals ...string) []alloraMath.Dec {
	out := make([]alloraMath.Dec, len(vals))
	for i, v := range vals {
		out[i] = alloraMath.MustNewDecFromString(v)
	}
	return out
}

// The reviewer's worry was that a topic whose active inferers all compute a zero
// weight (and that has no forecasters) would hit the !usedAny hard error and halt.
// It does not: a zero weight still marks the worker as used, so the function
// normalizes to a zero-valued vector of length numLabels and returns no error.
func TestCalcWeightedInference_AllZeroWeightNoForecasters_ReturnsZeroVector(t *testing.T) {
	const L = 2
	a, b := "inferer-a", "inferer-b"
	zero := alloraMath.ZeroDec()

	out, err := calcWeightedInference(calcWeightedInferenceArgs{
		logger:            log.NewNopLogger(),
		allInferersAreNew: false,
		inferers:          []Worker{a, b},
		workerToInference: map[Worker]*emissionstypes.Inference{
			a: {Values: infValues("1", "2")},
			b: {Values: infValues("3", "4")},
		},
		infererToRegret:                      map[Worker]*alloraMath.Dec{a: &zero, b: &zero},
		forecasters:                          []Worker{},
		forecasterToRegret:                   map[Worker]*alloraMath.Dec{},
		forecasterToForecastImpliedInference: map[Worker]*emissionstypes.Inference{},
		weights: RegretInformedWeights{
			Inferers:    map[Worker]Weight{a: alloraMath.ZeroDec(), b: alloraMath.ZeroDec()},
			Forecasters: map[Worker]Weight{},
		},
		epsilonSafeDiv: alloraMath.MustNewDecFromString("0.0001"),
		numLabels:      L,
	})

	require.NoError(t, err)
	require.Len(t, out, L)
	for i, v := range out {
		require.True(t, v.Equal(alloraMath.ZeroDec()), "expected zero at idx %d, got %s", i, v.String())
	}
}

// A genuine first epoch routes through the all-inferers-are-new branch (weight = 1
// per inferer), which also marks workers as used and returns the simple average —
// never the !usedAny error.
func TestCalcWeightedInference_AllInferersAreNew_ReturnsAverage(t *testing.T) {
	const L = 2
	a, b := "inferer-a", "inferer-b"

	out, err := calcWeightedInference(calcWeightedInferenceArgs{
		logger:            log.NewNopLogger(),
		allInferersAreNew: true,
		inferers:          []Worker{a, b},
		workerToInference: map[Worker]*emissionstypes.Inference{
			a: {Values: infValues("2", "4")},
			b: {Values: infValues("4", "8")},
		},
		infererToRegret:                      map[Worker]*alloraMath.Dec{},
		forecasters:                          []Worker{},
		forecasterToRegret:                   map[Worker]*alloraMath.Dec{},
		forecasterToForecastImpliedInference: map[Worker]*emissionstypes.Inference{},
		weights:                              RegretInformedWeights{Inferers: map[Worker]Weight{}, Forecasters: map[Worker]Weight{}},
		epsilonSafeDiv:                       alloraMath.MustNewDecFromString("0.0001"),
		numLabels:                            L,
	})

	require.NoError(t, err)
	require.Len(t, out, L)
	require.True(t, out[0].Equal(alloraMath.MustNewDecFromString("3")), "idx0=%s", out[0].String())
	require.True(t, out[1].Equal(alloraMath.MustNewDecFromString("6")), "idx1=%s", out[1].String())
}

// The only way to reach the !usedAny error: a listed inferer is absent from the
// weight/regret maps (and there are no forecasters). Production never builds such
// inputs — the inferer list is the keyset of workerToInference and the maps are
// derived from it — so this stands as a guard against an internal invariant break.
func TestCalcWeightedInference_InfererMissingFromMaps_ReturnsErrLogic(t *testing.T) {
	const L = 2
	a := "inferer-a"

	out, err := calcWeightedInference(calcWeightedInferenceArgs{
		logger:            log.NewNopLogger(),
		allInferersAreNew: false,
		inferers:          []Worker{a},
		workerToInference: map[Worker]*emissionstypes.Inference{
			a: {Values: infValues("1", "2")},
		},
		infererToRegret:                      map[Worker]*alloraMath.Dec{},
		forecasters:                          []Worker{},
		forecasterToRegret:                   map[Worker]*alloraMath.Dec{},
		forecasterToForecastImpliedInference: map[Worker]*emissionstypes.Inference{},
		weights:                              RegretInformedWeights{Inferers: map[Worker]Weight{}, Forecasters: map[Worker]Weight{}},
		epsilonSafeDiv:                       alloraMath.MustNewDecFromString("0.0001"),
		numLabels:                            L,
	})

	require.Error(t, err)
	require.ErrorIs(t, err, sdkerrors.ErrLogic)
	require.Nil(t, out)
}
