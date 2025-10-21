//nolint:exhaustruct
package collections_test

import (
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	storetypes "cosmossdk.io/store/types"
	collutils "github.com/allora-network/allora-chain/utils/collections"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
)

type thingsIndexes struct {
	ByType *indexes.Multi[string, int64, string]
}

func (i thingsIndexes) IndexesList() []collections.Index[int64, string] {
	return []collections.Index[int64, string]{
		i.ByType,
	}
}

func newThingsIndexes(sb *collections.SchemaBuilder) thingsIndexes {
	return thingsIndexes{
		ByType: indexes.NewMulti(
			sb, collections.NewPrefix(1), "things_by_thing", collections.StringKey, collections.Int64Key,
			func(_ int64, t string) (string, error) {
				return t, nil
			},
		),
	}
}

func TestMultiIndexCollectionWrapper(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey("test")
	storeService := runtime.NewKVStoreService(storeKey)
	sb := collections.NewSchemaBuilder(storeService)

	things := collections.NewIndexedMap[int64, string, thingsIndexes](
		sb,
		collections.NewPrefix(0),
		"things",
		collections.Int64Key,
		collections.StringValue,
		newThingsIndexes(sb),
	)

	_, err := sb.Build()
	require.NoError(t, err)

	ctx := testutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{"test": storeKey},
		map[string]*storetypes.TransientStoreKey{"transient_test": storetypes.NewTransientStoreKey("transient_test")},
		nil,
	)
	require.NoError(t, things.Set(ctx, 1, "a"))
	require.NoError(t, things.Set(ctx, 2, "a"))
	require.NoError(t, things.Set(ctx, 3, "a"))
	require.NoError(t, things.Set(ctx, 4, "b"))
	require.NoError(t, things.Set(ctx, 5, "b"))
	require.NoError(t, things.Set(ctx, 6, "b"))
	require.NoError(t, things.Set(ctx, 7, "c"))
	require.NoError(t, things.Set(ctx, 8, "c"))
	require.NoError(t, things.Set(ctx, 9, "c"))

	indexKeyCodec := collections.PairKeyCodec[string, int64](collections.StringKey, collections.Int64Key)
	encodePairKey := func(s string, i int64) []byte {
		key := collections.Join[string, int64](s, i)
		buf := make([]byte, indexKeyCodec.Size(key))
		_, err := indexKeyCodec.Encode(buf, key)
		require.NoError(t, err)
		return buf
	}
	encodePairKey2 := func(k2 int64) []byte {
		buf := make([]byte, collections.Int64Key.Size(k2))
		_, err := collections.Int64Key.Encode(buf, k2)
		require.NoError(t, err)
		return buf
	}

	wrapper := collutils.WrapMultiIndexToCollection(things.Indexes.ByType)
	require.Equal(t, things.Indexes.ByType.KeyCodec(), wrapper.KeyCodec())

	testCases := []struct {
		name        string
		page        *query.PageRequest
		opts        []func(o *query.CollectionsPaginateOptions[collections.Pair[string, int64]])
		expectItems []int64
		expectPage  *query.PageResponse
	}{
		{
			name:        "no pair prefix should get all elements",
			page:        &query.PageRequest{Limit: 100, CountTotal: true},
			opts:        nil,
			expectItems: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9},
			expectPage:  &query.PageResponse{Total: 9},
		},
		{
			name:        "no pair prefix with count total and limit",
			page:        &query.PageRequest{Limit: 5, CountTotal: true},
			opts:        nil,
			expectItems: []int64{1, 2, 3, 4, 5},
			expectPage:  &query.PageResponse{NextKey: encodePairKey("b", 6), Total: 9},
		},
		{
			name:        "no pair prefix reverse",
			page:        &query.PageRequest{Limit: 100, Reverse: true},
			opts:        nil,
			expectItems: []int64{9, 8, 7, 6, 5, 4, 3, 2, 1},
			expectPage:  &query.PageResponse{},
		},
		{
			name: "pair prefix opt should filter by index key (first elements)",
			page: &query.PageRequest{Limit: 100, CountTotal: true},
			opts: []func(o *query.CollectionsPaginateOptions[collections.Pair[string, int64]]){
				query.WithCollectionPaginationPairPrefix[string, int64]("a"),
			},
			expectItems: []int64{1, 2, 3},
			expectPage:  &query.PageResponse{Total: 3},
		},
		{
			name: "pair prefix opt should filter by index key (middle elements)",
			page: &query.PageRequest{Limit: 100},
			opts: []func(o *query.CollectionsPaginateOptions[collections.Pair[string, int64]]){
				query.WithCollectionPaginationPairPrefix[string, int64]("b"),
			},
			expectItems: []int64{4, 5, 6},
			expectPage:  &query.PageResponse{},
		},
		{
			name: "pair prefix opt should filter by index key (end elements)",
			page: &query.PageRequest{Limit: 100},
			opts: []func(o *query.CollectionsPaginateOptions[collections.Pair[string, int64]]){
				query.WithCollectionPaginationPairPrefix[string, int64]("c"),
			},
			expectItems: []int64{7, 8, 9},
			expectPage:  &query.PageResponse{},
		},
		{
			name: "pair prefix opt with count total and limit",
			page: &query.PageRequest{Limit: 2, CountTotal: true},
			opts: []func(o *query.CollectionsPaginateOptions[collections.Pair[string, int64]]){
				query.WithCollectionPaginationPairPrefix[string, int64]("a"),
			},
			expectItems: []int64{1, 2},
			expectPage:  &query.PageResponse{NextKey: encodePairKey2(3), Total: 3},
		},
		{
			name: "pair prefix reverse (middle elements)",
			page: &query.PageRequest{Limit: 100, Reverse: true},
			opts: []func(o *query.CollectionsPaginateOptions[collections.Pair[string, int64]]){
				query.WithCollectionPaginationPairPrefix[string, int64]("b"),
			},
			expectItems: []int64{6, 5, 4},
			expectPage:  &query.PageResponse{},
		},
		{
			name:        "pagination by offset without pair prefix, first page",
			page:        &query.PageRequest{Offset: 0, Limit: 4},
			opts:        nil,
			expectItems: []int64{1, 2, 3, 4},
			expectPage:  &query.PageResponse{NextKey: encodePairKey("b", 5)},
		},
		{
			name:        "pagination by offset without pair prefix, second page",
			page:        &query.PageRequest{Offset: 4, Limit: 4},
			opts:        nil,
			expectItems: []int64{5, 6, 7, 8},
			expectPage:  &query.PageResponse{NextKey: encodePairKey("c", 9)},
		},
		{
			name:        "pagination by offset without pair prefix reverse",
			page:        &query.PageRequest{Offset: 3, Limit: 4, Reverse: true},
			opts:        nil,
			expectItems: []int64{6, 5, 4, 3},
			expectPage:  &query.PageResponse{NextKey: encodePairKey("a", 2)},
		},
		{
			name:        "pagination by key without pair prefix",
			page:        &query.PageRequest{Key: encodePairKey("c", 8), Limit: 4},
			opts:        nil,
			expectItems: []int64{8, 9},
			expectPage:  &query.PageResponse{},
		},
		{
			name:        "pagination by key without pair prefix reverse",
			page:        &query.PageRequest{Key: encodePairKey("c", 7), Limit: 4, Reverse: true},
			opts:        nil,
			expectItems: []int64{7, 6, 5, 4},
			expectPage:  &query.PageResponse{NextKey: encodePairKey("a", 3)},
		},
		{
			name: "pagination by offset with pair prefix, first page",
			page: &query.PageRequest{Offset: 0, Limit: 2},
			opts: []func(o *query.CollectionsPaginateOptions[collections.Pair[string, int64]]){
				query.WithCollectionPaginationPairPrefix[string, int64]("b"),
			},
			expectItems: []int64{4, 5},
			expectPage:  &query.PageResponse{NextKey: encodePairKey2(6)},
		},
		{
			name: "pagination by offset with pair prefix, second page",
			page: &query.PageRequest{Offset: 2, Limit: 2},
			opts: []func(o *query.CollectionsPaginateOptions[collections.Pair[string, int64]]){
				query.WithCollectionPaginationPairPrefix[string, int64]("b"),
			},
			expectItems: []int64{6},
			expectPage:  &query.PageResponse{},
		},
		{
			name: "pagination by offset with pair prefix reverse",
			page: &query.PageRequest{Offset: 1, Limit: 2, Reverse: true},
			opts: []func(o *query.CollectionsPaginateOptions[collections.Pair[string, int64]]){
				query.WithCollectionPaginationPairPrefix[string, int64]("b"),
			},
			expectItems: []int64{5, 4},
			expectPage:  &query.PageResponse{},
		},
		{
			name: "pagination by key with pair prefix, second page",
			page: &query.PageRequest{Key: encodePairKey2(6), Limit: 2},
			opts: []func(o *query.CollectionsPaginateOptions[collections.Pair[string, int64]]){
				query.WithCollectionPaginationPairPrefix[string, int64]("b"),
			},
			expectItems: []int64{6},
			expectPage:  &query.PageResponse{},
		},
		{
			name: "pagination by key with pair prefix reverse",
			page: &query.PageRequest{Key: encodePairKey2(5), Limit: 2, Reverse: true},
			opts: []func(o *query.CollectionsPaginateOptions[collections.Pair[string, int64]]){
				query.WithCollectionPaginationPairPrefix[string, int64]("b"),
			},
			expectItems: []int64{5, 4},
			expectPage:  &query.PageResponse{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			items, page, err := query.CollectionPaginate(
				ctx,
				wrapper,
				tc.page,
				func(key collections.Pair[string, int64], _ collections.NoValue) (int64, error) {
					return key.K2(), nil
				},
				tc.opts...,
			)

			require.NoError(t, err)
			require.Equal(t, tc.expectItems, items)
			require.Equal(t, tc.expectPage, page)
		})
	}
}
