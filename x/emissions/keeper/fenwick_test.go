package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/colltest"

	alloraMath "github.com/allora-network/allora-chain/math"
	keeper "github.com/allora-network/allora-chain/x/emissions/keeper"
)

func newTestFenwickTree(t *testing.T) (keeper.FenwickTree, context.Context) {
	t.Helper()
	sk, ctx := colltest.MockStore()
	sb := collections.NewSchemaBuilder(sk)
	return keeper.NewFenwickTree(sb, collections.NewPrefix(0), "fenwick"), ctx
}

func requireRangeSum(t *testing.T, ctx context.Context, f keeper.FenwickTree, start, end, expected int64) {
	t.Helper()
	sum, err := f.RangeSum(ctx, start, end)
	require.NoError(t, err)
	require.True(t, alloraMath.NewDecFromInt64(expected).Equal(sum),
		"RangeSum(%d, %d) = %s, expected %d", start, end, sum.String(), expected)
}

func TestFenwickTreeRangeSum(t *testing.T) {
	f, ctx := newTestFenwickTree(t)

	elements := map[int64]int64{3: 100, 4: 20, 5: 14, 7: 20, 10: 100, 11: 123}
	for _, idx := range []int64{3, 4, 5, 7, 10, 11} {
		require.NoError(t, f.Append(ctx, idx, alloraMath.NewDecFromInt64(elements[idx])))
	}

	requireRangeSum(t, ctx, f, 1, 1000, 377)
	requireRangeSum(t, ctx, f, 6, 11, 120)
	requireRangeSum(t, ctx, f, 8, 10, 0)

	// single-element ranges recover the appended values, with skipped indices counting as zero
	for idx := int64(1); idx <= 13; idx++ {
		requireRangeSum(t, ctx, f, idx, idx+1, elements[idx])
	}

	// both bounds clamp to the last appended index
	requireRangeSum(t, ctx, f, 11, 5000, 123)
	requireRangeSum(t, ctx, f, 20, 30, 0)
}

func TestFenwickTreeEmpty(t *testing.T) {
	f, ctx := newTestFenwickTree(t)

	sum, err := f.RangeSum(ctx, 1, 100)
	require.NoError(t, err)
	require.True(t, sum.IsZero())
}

func TestFenwickTreeInvalidArguments(t *testing.T) {
	f, ctx := newTestFenwickTree(t)

	// indices must be positive
	require.Error(t, f.Append(ctx, 0, alloraMath.OneDec()))
	require.Error(t, f.Append(ctx, -5, alloraMath.OneDec()))

	require.NoError(t, f.Append(ctx, 5, alloraMath.OneDec()))

	// indices must be strictly increasing
	require.Error(t, f.Append(ctx, 5, alloraMath.OneDec()))
	require.Error(t, f.Append(ctx, 4, alloraMath.OneDec()))

	// values must be valid Decs
	require.Error(t, f.Append(ctx, 6, alloraMath.NewNaN()))

	// range start must be positive and no greater than end
	_, err := f.RangeSum(ctx, 0, 5)
	require.Error(t, err)
	_, err = f.RangeSum(ctx, -1, 5)
	require.Error(t, err)
	_, err = f.RangeSum(ctx, 5, 4)
	require.Error(t, err)
}

// TestFenwickTreeAgainstNaiveSum appends a sequence with gaps and non-integer
// values and checks every range sum against a naively computed one.
func TestFenwickTreeAgainstNaiveSum(t *testing.T) {
	f, ctx := newTestFenwickTree(t)

	const first, last int64 = 1000, 1128
	values := make(map[int64]alloraMath.Dec)
	for idx := first; idx <= last; idx++ {
		// skip some indices to exercise the gap-filling logic
		if idx%7 == 3 || idx%11 == 5 {
			continue
		}
		value, err := alloraMath.NewDecFromInt64(idx - 1064).Quo(alloraMath.NewDecFromInt64(16))
		require.NoError(t, err)
		values[idx] = value
		require.NoError(t, f.Append(ctx, idx, value))
	}

	for start := first - 8; start <= last+8; start++ {
		naive := alloraMath.ZeroDec()
		for end := start; end <= last+9; end++ {
			sum, err := f.RangeSum(ctx, start, end)
			require.NoError(t, err)
			require.True(t, naive.Equal(sum),
				"RangeSum(%d, %d) = %s, expected %s", start, end, sum.String(), naive.String())
			if value, ok := values[end]; ok {
				naive, err = naive.Add(value)
				require.NoError(t, err)
			}
		}
	}
}
