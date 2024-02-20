package module

import (
	"math/rand"
	"sort"
)

// A structure to hold the original value and a random tiebreaker
type Item[T any] struct {
	Value      any
	Weight     uint64
	Tiebreaker float64
}

// RandSortDesc sorts the given slice of integers in descending order according to their corresponding weight, using randomness as tiebreaker
// e.g. RandSortDesc([]uint64{1, 2, 3}, map[uint64]uint64{1: 2, 2: 2, 3: 3}, 0) -> [3, 1, 2] or [3, 2, 1]
func SortDescWithRandomTiebreaker[T any](valsToSort []T, weights map[uint64]uint64, randSeed uint64) []T {
	// Convert the slice of integers to a slice of Items, each with a random tiebreaker
	r := rand.New(rand.NewSource(int64(randSeed)))
	items := make([]Item[T], len(valsToSort))
	for i, v := range valsToSort {
		items[i] = Item[T]{v, weights[uint64(i)], r.Float64()}
	}

	// Sort the slice of Items
	// If the values are equal, the tiebreaker will decide their order
	sort.Slice(items, func(i, j int) bool {
		if items[i].Value == items[j].Value {
			return items[i].Tiebreaker > items[j].Tiebreaker
		}
		return items[i].Weight > items[j].Weight
	})

	// Extract and print the sorted values to demonstrate the sorting
	sortedValues := make([]T, len(valsToSort))
	for i, item := range items {
		sortedValues[i] = item.Value.(T)
	}
	return sortedValues
}
