package rewards_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

const (
	presenceForecaster = "allo10ljt8c3nuky75nnca4gnt9sqaxrmmyaz4d520a"
	presenceWorker1    = "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy"
	presenceWorker2    = "allo13kenskkx7e0v253m3kcgwfc67cmx00fgwpgj6h"
)

func withheldValuesByWorker(values []*types.WithheldWorkerAttributedValue) map[string]alloraMath.Dec {
	byWorker := make(map[string]alloraMath.Dec, len(values))
	for _, v := range values {
		byWorker[v.Worker] = v.Value
	}
	return byWorker
}

// Note from v0.18.0 onwards, these tests (first-seen deduping) will be exercised over previously-stored legacy data,
// since validations are stricter and duplicates are disallowed from that point forward.
func TestEnsureWorkerPresenceKeepsFirstForecasterRow(t *testing.T) {
	bundle := &types.LossBundle{ //nolint:exhaustruct // only the one-out-inferer forecaster rows are under test
		OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{
			{
				Forecaster: presenceForecaster,
				OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
					{Worker: presenceWorker1, Value: alloraMath.NewDecFromInt64(100)},
				},
			},
			{
				Forecaster: presenceForecaster,
				OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
					{Worker: presenceWorker2, Value: alloraMath.NewDecFromInt64(200)},
				},
			},
		},
	}

	updated := rewards.EnsureWorkerPresence(types.LossBundles{bundle})

	require.Len(t, updated[0].OneOutInfererForecasterValues, 1)
	row := updated[0].OneOutInfererForecasterValues[0]
	require.Equal(t, presenceForecaster, row.Forecaster)
	// The dropped row contributes nothing: its worker is absent, not even NaN.
	require.Len(t, row.OneOutInfererValues, 1)
	require.Equal(t, presenceWorker1, row.OneOutInfererValues[0].Worker)
	require.True(t, alloraMath.NewDecFromInt64(100).Equal(row.OneOutInfererValues[0].Value))

	// A second pass is a no-op.
	again := rewards.EnsureWorkerPresence(updated)
	require.Len(t, again[0].OneOutInfererForecasterValues, 1)
	require.Len(t, again[0].OneOutInfererForecasterValues[0].OneOutInfererValues, 1)
	require.Equal(t, presenceWorker1, again[0].OneOutInfererForecasterValues[0].OneOutInfererValues[0].Worker)
	require.True(t, alloraMath.NewDecFromInt64(100).Equal(again[0].OneOutInfererForecasterValues[0].OneOutInfererValues[0].Value))
}

func TestEnsureWorkerPresenceAlignsDroppedWorkerWithNaN(t *testing.T) {
	first := &types.LossBundle{ //nolint:exhaustruct // only the one-out-inferer forecaster rows are under test
		OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{
			{
				Forecaster: presenceForecaster,
				OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
					{Worker: presenceWorker1, Value: alloraMath.NewDecFromInt64(100)},
				},
			},
			{
				Forecaster: presenceForecaster,
				OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
					{Worker: presenceWorker2, Value: alloraMath.NewDecFromInt64(200)},
				},
			},
		},
	}
	second := &types.LossBundle{ //nolint:exhaustruct // only the one-out-inferer forecaster rows are under test
		OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{
			{
				Forecaster: presenceForecaster,
				OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
					{Worker: presenceWorker1, Value: alloraMath.NewDecFromInt64(300)},
					{Worker: presenceWorker2, Value: alloraMath.NewDecFromInt64(400)},
				},
			},
		},
	}

	updated := rewards.EnsureWorkerPresence(types.LossBundles{first, second})

	require.Len(t, updated[0].OneOutInfererForecasterValues, 1)
	firstValues := withheldValuesByWorker(updated[0].OneOutInfererForecasterValues[0].OneOutInfererValues)
	require.Len(t, firstValues, 2)
	require.True(t, alloraMath.NewDecFromInt64(100).Equal(firstValues[presenceWorker1]))
	// The dropped row kept its worker out of the union; the second bundle
	// carries it, so alignment backfills NaN and the dropped 200 is gone.
	require.True(t, firstValues[presenceWorker2].IsNaN())

	require.Len(t, updated[1].OneOutInfererForecasterValues, 1)
	secondValues := withheldValuesByWorker(updated[1].OneOutInfererForecasterValues[0].OneOutInfererValues)
	require.True(t, alloraMath.NewDecFromInt64(300).Equal(secondValues[presenceWorker1]))
	require.True(t, alloraMath.NewDecFromInt64(400).Equal(secondValues[presenceWorker2]))
}

func TestEnsureWorkerPresenceEmptyRowDoesNotShadowLaterRow(t *testing.T) {
	bundle := &types.LossBundle{ //nolint:exhaustruct // only the one-out-inferer forecaster rows are under test
		OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{
			{
				Forecaster:          presenceForecaster,
				OneOutInfererValues: []*types.WithheldWorkerAttributedValue{},
			},
			{
				Forecaster: presenceForecaster,
				OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
					{Worker: presenceWorker1, Value: alloraMath.NewDecFromInt64(100)},
				},
			},
		},
	}

	updated := rewards.EnsureWorkerPresence(types.LossBundles{bundle})

	require.Len(t, updated[0].OneOutInfererForecasterValues, 1)
	row := updated[0].OneOutInfererForecasterValues[0]
	require.Len(t, row.OneOutInfererValues, 1)
	require.Equal(t, presenceWorker1, row.OneOutInfererValues[0].Worker)
	require.True(t, alloraMath.NewDecFromInt64(100).Equal(row.OneOutInfererValues[0].Value))
}
