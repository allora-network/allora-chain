package migutils_test

import (
	"fmt"
	rand "math/rand/v2"
	"strings"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/codec"
	"github.com/stretchr/testify/require"

	"github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/utils/migutils"
)

func TestSafelyClearWholeMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		keys        int
		maxPageSize uint64
	}{
		{30, 5},
		{30, 30},
		{1, 1},
		{1, 4},
		{4, 1},
	}

	for _, tc := range tests {
		name := fmt.Sprintf("keys:%d maxPageSize:%d", tc.keys, tc.maxPageSize)

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			runSafelyClearWholeMapCase(t, collections.StringKey, collections.Uint16Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.StringKey, collections.Uint32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.StringKey, collections.Uint64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.StringKey, collections.Int32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.StringKey, collections.Int64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.StringKey, collections.BoolValue, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.StringKey, collections.BytesValue, tc.keys, tc.maxPageSize)

			runSafelyClearWholeMapCase(t, collections.BoolKey, collections.Uint16Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.BoolKey, collections.Uint32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.BoolKey, collections.Uint64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.BoolKey, collections.Int32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.BoolKey, collections.Int64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.BoolKey, collections.BoolValue, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.BoolKey, collections.BytesValue, tc.keys, tc.maxPageSize)

			runSafelyClearWholeMapCase(t, collections.BytesKey, collections.Uint16Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.BytesKey, collections.Uint32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.BytesKey, collections.Uint64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.BytesKey, collections.Int32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.BytesKey, collections.Int64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.BytesKey, collections.BoolValue, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.BytesKey, collections.BytesValue, tc.keys, tc.maxPageSize)

			runSafelyClearWholeMapCase(t, collections.Uint16Key, collections.Uint16Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint16Key, collections.Uint32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint16Key, collections.Uint64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint16Key, collections.Int32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint16Key, collections.Int64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint16Key, collections.BoolValue, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint16Key, collections.BytesValue, tc.keys, tc.maxPageSize)

			runSafelyClearWholeMapCase(t, collections.Uint32Key, collections.Uint16Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint32Key, collections.Uint32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint32Key, collections.Uint64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint32Key, collections.Int32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint32Key, collections.Int64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint32Key, collections.BoolValue, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint32Key, collections.BytesValue, tc.keys, tc.maxPageSize)

			runSafelyClearWholeMapCase(t, collections.Uint64Key, collections.Uint16Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint64Key, collections.Uint32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint64Key, collections.Uint64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint64Key, collections.Int32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint64Key, collections.Int64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint64Key, collections.BoolValue, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Uint64Key, collections.BytesValue, tc.keys, tc.maxPageSize)

			runSafelyClearWholeMapCase(t, collections.Int32Key, collections.Uint16Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Int32Key, collections.Uint32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Int32Key, collections.Uint64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Int32Key, collections.Int32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Int32Key, collections.Int64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Int32Key, collections.BoolValue, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Int32Key, collections.BytesValue, tc.keys, tc.maxPageSize)

			runSafelyClearWholeMapCase(t, collections.Int64Key, collections.Uint16Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Int64Key, collections.Uint32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Int64Key, collections.Uint64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Int64Key, collections.Int32Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Int64Key, collections.Int64Value, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Int64Key, collections.BoolValue, tc.keys, tc.maxPageSize)
			runSafelyClearWholeMapCase(t, collections.Int64Key, collections.BytesValue, tc.keys, tc.maxPageSize)
		})
	}
}

func runSafelyClearWholeMapCase[K, V any](
	t *testing.T,
	keyType codec.KeyCodec[K],
	valType codec.ValueCodec[V],
	numKeys int,
	maxPageSize uint64,
) {
	t.Helper()

	testDB := testutil.NewTestDB(t)

	prefix := collections.NewPrefix(rand.IntN(255))
	mapName := testutil.MustRandomString(rand.IntN(32))
	m := collections.NewMap(testDB.SB, prefix, mapName, keyType, valType)

	var keys []K
	for range numKeys {
		key := testutil.RandomValueOfType[K](t)
		val := testutil.RandomValueOfType[V](t)

		err := m.Set(testDB.TestCtx.Ctx, key, val)
		require.NoError(t, err)

		keys = append(keys, key)
	}

	err := migutils.SafelyClearWholeMap(testDB.TestCtx.Ctx, testDB.Store, prefix, maxPageSize)
	require.NoError(t, err)

	for _, key := range keys {
		_, err := m.Get(testDB.TestCtx.Ctx, key)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "collections: not found"))
	}
}
