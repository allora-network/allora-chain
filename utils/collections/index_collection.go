package collections

import (
	"context"

	"cosmossdk.io/collections"
	collcodec "cosmossdk.io/collections/codec"
	"cosmossdk.io/collections/indexes"
	"github.com/cosmos/cosmos-sdk/types/query"
)

var _ query.Collection[collections.Pair[any, any], collections.NoValue] = multiIndexCollectionWrapper[any, any, any]{}

// WrapMultiIndexToCollection wraps a indexes.Multi as a query.Collection.
// The keys of the collection are pairs of (ReferenceKey, PrimaryKey) and the values are empty (NoValue).
// This allows the use of provided pagination primitives with an indexes.Multi.
func WrapMultiIndexToCollection[R, K, V any](i *indexes.Multi[R, K, V]) query.Collection[collections.Pair[R, K], collections.NoValue] {
	return multiIndexCollectionWrapper[R, K, V]{i: i}
}

// multiIndexCollectionWrapper implements query.Collection by wrapping an indexes.Multi.
type multiIndexCollectionWrapper[ReferenceKey, PrimaryKey, Value any] struct {
	i *indexes.Multi[ReferenceKey, PrimaryKey, Value]
}

func (w multiIndexCollectionWrapper[R, P, V]) IterateRaw(ctx context.Context, start, end []byte, order collections.Order) (collections.Iterator[collections.Pair[R, P], collections.NoValue], error) {
	var it collections.Iterator[collections.Pair[R, P], collections.NoValue]
	var err error
	var startKey, endKey collections.Pair[R, P]
	if len(start) > 0 {
		_, startKey, err = w.KeyCodec().Decode(start)
		if err != nil {
			return it, err
		}
	}
	if len(end) > 0 {
		_, endKey, err = w.KeyCodec().Decode(end)
		if err != nil {
			return it, err
		}
	}

	ranger := (&collections.Range[collections.Pair[R, P]]{}).
		StartInclusive(startKey).
		EndInclusive(endKey)
	if order == collections.OrderDescending {
		ranger = ranger.Descending()
	}

	mit, err := w.i.Iterate(ctx, ranger)
	if err != nil {
		return it, err
	}
	return collections.Iterator[collections.Pair[R, P], collections.NoValue](mit), nil
}

func (w multiIndexCollectionWrapper[R, K, V]) KeyCodec() collcodec.KeyCodec[collections.Pair[R, K]] {
	return w.i.KeyCodec()
}
