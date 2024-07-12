package testcommon

import (
	"fmt"
	"math/rand"
)

type ValueIndex[V any] struct {
	v V
	i int
}

type RandomKeyMap[K comparable, V any] struct {
	rand *rand.Rand
	m    map[K]ValueIndex[V]
	s    []K
}

// RandomKeyMap is a map that is O(1) for insertion, deletion, and random key selection
func NewRandomKeyMap[K comparable, V any](r *rand.Rand) *RandomKeyMap[K, V] {
	return &RandomKeyMap[K, V]{
		rand: r,
		m:    make(map[K]ValueIndex[V]),
		s:    []K{},
	}
}

// Get returns an element from the map
func (rkm *RandomKeyMap[K, V]) Get(k K) (V, bool) {
	valueIndex, ok := rkm.m[k]
	return valueIndex.v, ok
}

// GetAll returns all elements from the map
func (rkm *RandomKeyMap[K, V]) GetAll() []V {
	values := make([]V, len(rkm.s))
	for i, k := range rkm.s {
		values[i] = rkm.m[k].v
	}
	return values
}

// Filter returns all elements from the map that satisfy the predicate
func (rkm *RandomKeyMap[K, V]) Filter(f func(K) bool) ([]K, []V) {
	keys := make([]K, 0)
	values := make([]V, 0)
	for _, k := range rkm.s {
		if f(k) {
			keys = append(keys, k)
			values = append(values, rkm.m[k].v)
		}
	}
	return keys, values
}

// Upsert element into the map
func (rkm *RandomKeyMap[K, V]) Upsert(k K, v V) {
	if valueIndex, ok := rkm.m[k]; ok {
		rkm.m[k] = ValueIndex[V]{v, valueIndex.i}
		return
	}
	rkm.m[k] = ValueIndex[V]{i: len(rkm.s), v: v}
	rkm.s = append(rkm.s, k)
}

// Remove element from the map by swapping in the last element in its place.
func (rkm *RandomKeyMap[K, V]) Delete(k K) {
	valueIndexOfK, ok := rkm.m[k]
	if !ok {
		return
	}
	indexOfK := valueIndexOfK.i
	lastElementKey := rkm.s[len(rkm.s)-1]
	lastElementValue := rkm.m[lastElementKey].v
	// set the slice position of the deleted element to the last element
	rkm.s[indexOfK] = lastElementKey
	// chop off the last element of the slice
	rkm.s = rkm.s[:len(rkm.s)-1]
	// update the index of the last element in the map to its new slice position
	rkm.m[lastElementKey] = ValueIndex[V]{i: indexOfK, v: lastElementValue}
	// delete the element from the map
	delete(rkm.m, k)
}

// Get a random key from the map
func (rkm *RandomKeyMap[K, V]) RandomKey() (*K, error) {
	if len(rkm.s) == 0 {
		return nil, fmt.Errorf("RandomKey called on empty M")
	}
	ret := rkm.s[rkm.rand.Intn(len(rkm.s))]
	return &ret, nil
}

// length of the map & slice
func (rkm *RandomKeyMap[K, V]) Len() int {
	return len(rkm.s)
}
