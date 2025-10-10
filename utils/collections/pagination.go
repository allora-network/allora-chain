package collections

import (
	"cosmossdk.io/collections"
	"github.com/cosmos/cosmos-sdk/types/query"
)

// WithCollectionPaginationTriplePrefix applies a prefix to a collection, whose key is a collection.Triple,
// being paginated that needs prefixing.
func WithCollectionPaginationTriplePrefix[K1, K2, K3 any](prefix K1) func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
	return func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
		prefix := collections.TriplePrefix[K1, K2, K3](prefix)
		o.Prefix = &prefix
	}
}

// WithCollectionPaginationTripleSuperPrefix applies a super prefix to a collection, whose key is a collection.Triple,
// being paginated that needs prefixing.
func WithCollectionPaginationTripleSuperPrefix[K1, K2, K3 any](prefix1 K1, prefix2 K2) func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
	return func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
		prefix := collections.TripleSuperPrefix[K1, K2, K3](prefix1, prefix2)
		o.Prefix = &prefix
	}
}
