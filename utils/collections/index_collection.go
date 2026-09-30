package collections

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/codec"
	"cosmossdk.io/collections/indexes"
	"github.com/cosmos/cosmos-sdk/types/query"
)

var _ query.Collection[collections.Pair[any, any], collections.NoValue] = multiIndexCollectionWrapper[any, any, any]{} //nolint:exhaustruct

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

type pairKeyCodec[K1, K2 any] interface {
	KeyCodec1() codec.KeyCodec[K1]
	KeyCodec2() codec.KeyCodec[K2]
}

func (w multiIndexCollectionWrapper[R, P, V]) IterateRaw(ctx context.Context, start, end []byte, order collections.Order) (collections.Iterator[collections.Pair[R, P], collections.NoValue], error) {
	var it collections.Iterator[collections.Pair[R, P], collections.NoValue]
	ranger, err := w.decodeRange(start, end, order)
	if err != nil {
		return it, err
	}

	mit, err := w.i.Iterate(ctx, ranger)
	if err != nil {
		return it, err
	}
	return collections.Iterator[collections.Pair[R, P], collections.NoValue](mit), nil
}

func (w multiIndexCollectionWrapper[R, P, V]) KeyCodec() codec.KeyCodec[collections.Pair[R, P]] {
	return w.i.KeyCodec()
}

func (w multiIndexCollectionWrapper[R, P, V]) decodeRange(start, end []byte, order collections.Order) (collections.Ranger[collections.Pair[R, P]], error) {
	ranger := &collections.Range[collections.Pair[R, P]]{}

	if len(end) > 0 {
		// The end range is always a prefix and has its last byte incremented to make it just after the real prefix, so we decrement it
		alignedEnd := make([]byte, len(end))
		copy(alignedEnd, end)
		alignedEnd[len(alignedEnd)-1]--

		endKey, err := w.decodeKey(alignedEnd)
		if err != nil {
			return nil, err
		}

		ranger = ranger.Prefix(endKey)
	}

	startKey, err := w.decodeKey(start)
	if err != nil {
		return nil, err
	}
	ranger = ranger.StartInclusive(startKey)

	if order == collections.OrderDescending {
		ranger = ranger.Descending()
	}

	return ranger, nil
}

func (w multiIndexCollectionWrapper[R, P, V]) decodeKey(key []byte) (collections.Pair[R, P], error) {
	var p collections.Pair[R, P]
	if len(key) == 0 {
		return p, nil
	}

	pkc, ok := w.KeyCodec().(pairKeyCodec[R, P])
	if !ok {
		return p, fmt.Errorf("key codec is not a pair key codec")
	}
	pkc1 := pkc.KeyCodec1()
	pkc2 := pkc.KeyCodec2()

	read, k1, err := pkc1.DecodeNonTerminal(key)
	if err != nil {
		return p, nil
	}

	remainingKey := key[read:]
	if len(remainingKey) == 0 {
		return collections.PairPrefix[R, P](k1), nil
	}

	_, k2, err := pkc2.Decode(remainingKey)
	if err != nil {
		return p, nil
	}

	return collections.Join(k1, k2), nil
}
