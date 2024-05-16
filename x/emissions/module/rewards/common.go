package rewards

import (
	"sort"
)

func GetSortedUint64Keys[T any](m map[uint64]T) []uint64 {
	keys := make([]uint64, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
