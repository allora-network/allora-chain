package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func TestNetworkInferenceBundleToLossBundleSkeletonGroupsForecasterRows(t *testing.T) {
	const (
		forecasterA = "allo1forecastera"
		forecasterB = "allo1forecasterb"
		worker1     = "allo1worker1"
		worker2     = "allo1worker2"
	)

	bundle := &types.NetworkInferenceBundle{ //nolint:exhaustruct // only the one-out-inferer forecaster values are under test
		OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValue{
			{
				Forecaster:        forecasterA,
				WithheldInferer:   worker1,
				CombinedInference: []*types.LabeledValue{{Value: alloraMath.NewDecFromInt64(11)}},
			},
			{
				Forecaster:        forecasterB,
				WithheldInferer:   worker1,
				CombinedInference: []*types.LabeledValue{{Value: alloraMath.NewDecFromInt64(21)}},
			},
			{
				Forecaster:        forecasterA,
				WithheldInferer:   worker2,
				CombinedInference: []*types.LabeledValue{{Value: alloraMath.NewDecFromInt64(12)}},
			},
		},
	}

	skeleton := NetworkInferenceBundleToLossBundleSkeleton(bundle)

	// One row per forecaster, in first-seen order, with every withheld
	// inferer of that forecaster inside the row.
	require.Len(t, skeleton.OneOutInfererForecasterValues, 2)

	first := skeleton.OneOutInfererForecasterValues[0]
	require.Equal(t, forecasterA, first.Forecaster)
	require.Len(t, first.OneOutInfererValues, 2)
	require.Equal(t, worker1, first.OneOutInfererValues[0].Worker)
	require.True(t, alloraMath.NewDecFromInt64(11).Equal(first.OneOutInfererValues[0].Value))
	require.Equal(t, worker2, first.OneOutInfererValues[1].Worker)
	require.True(t, alloraMath.NewDecFromInt64(12).Equal(first.OneOutInfererValues[1].Value))

	second := skeleton.OneOutInfererForecasterValues[1]
	require.Equal(t, forecasterB, second.Forecaster)
	require.Len(t, second.OneOutInfererValues, 1)
	require.Equal(t, worker1, second.OneOutInfererValues[0].Worker)
	require.True(t, alloraMath.NewDecFromInt64(21).Equal(second.OneOutInfererValues[0].Value))
}
