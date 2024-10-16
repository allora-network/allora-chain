package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPool(t *testing.T) {
	newFunc := func() int {
		return 42
	}
	resetFunc := func(x int) int {
		return x + 1
	}
	pool := NewPool(newFunc, resetFunc, nil)
	require.NotNil(t, pool, "Expected pool to be initialized")

	x := pool.Get()
	require.Equal(t, 42, x, "Expected Get() to return 42")
}

func TestPool_GetPut(t *testing.T) {
	pool := NewPool(func() string { return "initial" }, nil, nil)

	// Get a new item
	s := pool.Get()
	require.Equal(t, "initial", s, "Expected 'initial'")

	// Put back a different string
	pool.Put("updated")

	// Get should return the updated string
	s = pool.Get()
	require.Equal(t, "updated", s, "Expected 'updated'")
}

func TestNewBytesPool(t *testing.T) {
	size := 1024
	bytesPool := NewBytesPool(size, 0)
	require.NotNil(t, bytesPool, "Expected bytes pool to be initialized")

	b := bytesPool.Get()
	require.Equal(t, size, cap(b), "Expected byte slice with capacity %d", size)

	// Modify and put back
	b = append(b, 1, 2, 3)
	bytesPool.Put(b)

	// Get again and check reset
	b = bytesPool.Get()
	require.Empty(t, b, "Expected byte slice length 0 after reset")

	// Change the size of b
	b = make([]byte, size+1)
	bytesPool.Put(b)
	b = bytesPool.Get()
	require.Equal(t, size+1, cap(b), "Expected byte slice with capacity %d", size+1)

	bytesPool = NewBytesPool(size, size+1)
	b = make([]byte, size+2)
	bytesPool.Put(b)
	b = bytesPool.Get()
	require.Equal(t, size, cap(b), "Expected byte slice with capacity %d", size)

	bytesPool = NewBytesPool(size, 0)
	b = nil
	bytesPool.Put(b)
	b = bytesPool.Get()
	require.Equal(t, size, cap(b), "Expected byte slice with capacity %d", size)
}
