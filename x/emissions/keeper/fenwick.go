package keeper

import (
	"context"
	"errors"
	"math/bits"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// lsb returns the least significant set bit of x, i.e. the largest power of 2 dividing x.
func lsb(x int64) int64 {
	return x & -x
}

// bitLength returns the number of bits needed to represent x,
// i.e. the position of the highest set bit plus one. x must be non-negative.
func bitLength(x int64) int {
	return bits.Len64(uint64(x)) //nolint:gosec // G115: x is non-negative by the callers' invariants
}

// FenwickTree stores a sequence of Dec values under strictly increasing positive
// int64 indices (e.g. block heights) and computes sums over index ranges in
// logarithmic time.
//
// The indices do not have to be consecutive: values can be appended under any
// index greater than the last one, and indices that were skipped count as zero.
// Tree nodes missing from the backing map are also treated as zero, so range
// sums degrade gracefully (returning partial sums) instead of failing if parts
// of the history are removed from state.
type FenwickTree struct {
	// backing storage: the node at index i holds the sum of the elements
	// with indices in (i - lsb(i), i]; missing nodes count as zero
	x collections.Map[int64, alloraMath.Dec]
}

// NewFenwickTree creates a Fenwick tree backed by a collections map registered
// under the given prefix and name.
func NewFenwickTree(sb *collections.SchemaBuilder, prefix collections.Prefix, name string) FenwickTree {
	return FenwickTree{
		x: collections.NewMap(sb, prefix, name, collections.Int64Key, alloraMath.DecValue),
	}
}

// lastIdx returns the largest index present in the tree,
// with found = false if the tree is empty.
func (f FenwickTree) lastIdx(ctx context.Context) (last int64, found bool, err error) {
	rng := &collections.Range[int64]{}
	iter, err := f.x.Iterate(ctx, rng.Descending())
	if err != nil {
		return 0, false, err
	}
	defer iter.Close()
	if !iter.Valid() {
		return 0, false, nil
	}
	key, err := iter.Key()
	if err != nil {
		return 0, false, err
	}
	return key, true, nil
}

// prefixSumWithCap computes the sum of all elements with indices > idx - idx % cap and <= idx.
// cap is assumed to be a power of 2. Nodes missing from the backing map count as zero.
func (f FenwickTree) prefixSumWithCap(ctx context.Context, idx int64, cap int64) (alloraMath.Dec, error) {
	set := idx % cap
	offset := idx - set
	result := alloraMath.ZeroDec()
	for lsb(set) != 0 {
		value, err := f.x.Get(ctx, offset+set)
		if err == nil {
			result, err = result.Add(value)
			if err != nil {
				return alloraMath.Dec{}, err
			}
		} else if !errors.Is(err, collections.ErrNotFound) {
			return alloraMath.Dec{}, err
		}
		set -= lsb(set)
	}
	return result, nil
}

// Append inserts a value under the given index, which must be positive and
// strictly greater than every index appended so far.
func (f FenwickTree) Append(ctx context.Context, idx int64, value alloraMath.Dec) error {
	if idx <= 0 {
		return errorsmod.Wrap(types.ErrInvalidValue, "fenwick tree index must be positive")
	}
	if err := types.ValidateDec(value); err != nil {
		return errorsmod.Wrap(err, "fenwick tree value validation failed")
	}

	// if indices were skipped, fill the nonzero tree nodes in between
	last, found, err := f.lastIdx(ctx)
	if err != nil {
		return err
	}
	if found {
		if idx <= last {
			return errorsmod.Wrap(types.ErrInvalidValue, "fenwick tree indices must be strictly increasing")
		}
		missingIdx := last + lsb(last)
		for missingIdx < idx {
			sum, err := f.prefixSumWithCap(ctx, missingIdx-1, lsb(missingIdx))
			if err != nil {
				return err
			}
			if err := f.x.Set(ctx, missingIdx, sum); err != nil {
				return err
			}
			missingIdx += lsb(missingIdx)
		}
	}

	prefixSum, err := f.prefixSumWithCap(ctx, idx-1, lsb(idx))
	if err != nil {
		return err
	}
	nodeValue, err := value.Add(prefixSum)
	if err != nil {
		return err
	}
	return f.x.Set(ctx, idx, nodeValue)
}

// RangeSum computes the sum of the elements with indices in [start, end).
// Both bounds are clamped to the last appended index, so querying past the
// end of the sequence sums up to and including the last element.
func (f FenwickTree) RangeSum(ctx context.Context, start int64, end int64) (alloraMath.Dec, error) {
	if start <= 0 || end < start {
		return alloraMath.Dec{}, errorsmod.Wrap(types.ErrInvalidValue, "fenwick tree range start must be positive and no greater than end")
	}
	last, found, err := f.lastIdx(ctx)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	if !found {
		return alloraMath.ZeroDec(), nil
	}

	// convert the [start, end) convention to (start, end]
	if start < last+1 {
		start--
	} else {
		start = last
	}
	if end < last+1 {
		end--
	} else {
		end = last
	}

	cap := int64(1) << bitLength(start^end)
	endSum, err := f.prefixSumWithCap(ctx, end, cap)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	startSum, err := f.prefixSumWithCap(ctx, start, cap)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return endSum.Sub(startSum)
}
