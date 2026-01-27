package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/colltest"
	keeper "github.com/allora-network/allora-chain/x/emissions/keeper"
)

func TestQuadruple(t *testing.T) {
	kc := keeper.QuadrupleKeyCodec(
		collections.Int64Key,
		collections.Uint64Key,
		collections.StringKey,
		collections.BytesKey,
	)

	t.Run("conformance", func(t *testing.T) {
		colltest.TestKeyCodec(t, kc, keeper.Join4(int64(-1), uint64(1), "2", []byte("3")))
	})
}

func TestQuadrupleRange(t *testing.T) {
	sk, ctx := colltest.MockStore()
	schema := collections.NewSchemaBuilder(sk)
	// this is a key composed of 4 parts: int64, uint64, string, []byte
	kc := keeper.QuadrupleKeyCodec(
		collections.Int64Key,
		collections.Uint64Key,
		collections.StringKey,
		collections.BytesKey,
	)

	keySet := collections.NewKeySet(schema, collections.NewPrefix(0), "quadruple", kc)

	keys := []keeper.Quadruple[int64, uint64, string, []byte]{
		keeper.Join4(int64(-1), uint64(1), "A", []byte("1")),
		keeper.Join4(int64(-1), uint64(1), "A", []byte("2")),
		keeper.Join4(int64(-1), uint64(1), "B", []byte("3")),
		keeper.Join4(int64(-1), uint64(15), "B", []byte("4")),
		keeper.Join4(int64(256), uint64(12), "B", []byte("5")),
	}

	for _, k := range keys {
		require.NoError(t, keySet.Set(ctx, k))
	}

	// we prefix over (-1) we expect 4 results
	iter, err := keySet.Iterate(
		ctx,
		keeper.NewSinglePrefixedQuadrupleRange[int64, uint64, string, []byte](-1),
	)
	require.NoError(t, err)
	gotKeys, err := iter.Keys()
	require.NoError(t, err)
	require.Equal(t, keys[:4], gotKeys)

	// we double prefix over (-1, 1) we expect 3 results
	iter, err = keySet.Iterate(
		ctx,
		keeper.NewDoublePrefixedQuadrupleRange[int64, uint64, string, []byte](-1, 1),
	)
	require.NoError(t, err)
	gotKeys, err = iter.Keys()
	require.NoError(t, err)
	require.Equal(t, keys[:3], gotKeys)

	// we triple prefix over (-1, 1, "A") we expect 2 results
	iter, err = keySet.Iterate(
		ctx,
		keeper.NewTriplePrefixedQuadrupleRange[int64, uint64, string, []byte](-1, 1, "A"),
	)
	require.NoError(t, err)
	gotKeys, err = iter.Keys()
	require.NoError(t, err)
	require.Equal(t, keys[:2], gotKeys)
}
