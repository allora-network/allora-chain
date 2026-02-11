package utils

import "sync"

type Pool[T any] struct {
	sync.Pool
	reset func(T) T
	valid func(T) bool
}

func (p *Pool[T]) Get() T {
	return p.Pool.Get().(T) //nolint:forcetypeassert // we know the type from the pool definition
}

func (p *Pool[T]) Put(x T) {
	// Check if the item is valid
	if p.valid != nil {
		if !p.valid(x) {
			return
		}
	}

	// Reset
	if p.reset != nil {
		x = p.reset(x)
	}

	// Put it back in the pool
	p.Pool.Put(x)
}

func NewPool[T any](newF func() T, reset func(T) T, valid func(T) bool) *Pool[T] {
	return &Pool[T]{
		Pool: sync.Pool{
			New: func() any {
				return newF()
			},
		},
		reset: reset,
		valid: valid,
	}
}

func NewBytesPool(size int, maxSize int) *Pool[[]byte] {
	return NewPool[[]byte](func() []byte {
		return make([]byte, 0, size)
	}, func(b []byte) []byte {
		return b[:0]
	}, func(b []byte) bool {
		if cap(b) < size {
			return false
		}

		if maxSize == 0 {
			return true
		}
		return cap(b) <= maxSize
	})
}
