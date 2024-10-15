package utils

import (
	"testing"
)

func TestNewPool(t *testing.T) {
	newFunc := func() int {
		return 42
	}
	resetFunc := func(x int) int {
		return x + 1
	}
	pool := NewPool(newFunc, resetFunc, nil)

	if pool == nil {
		t.Fatal("Expected pool to be initialized")
	}

	x := pool.Get()
	if x != 42 {
		t.Errorf("Expected Get() to return 42, got %d", x)
	}
}

func TestPool_GetPut(t *testing.T) {
	pool := NewPool(func() string { return "initial" }, nil, nil)

	// Get a new item
	s := pool.Get()
	if s != "initial" {
		t.Errorf("Expected 'initial', got '%s'", s)
	}

	// Put back a different string
	pool.Put("updated")

	// Get should return the updated string
	s = pool.Get()
	if s != "updated" {
		t.Errorf("Expected 'updated', got '%s'", s)
	}
}

func TestNewBytesPool(t *testing.T) {
	size := 1024
	bytesPool := NewBytesPool(size, 0)
	if bytesPool == nil {
		t.Fatal("Expected bytes pool to be initialized")
	}

	b := bytesPool.Get()
	if cap(b) != size {
		t.Errorf("Expected byte slice with capacity %d, got %d", size, cap(b))
	}

	// Modify and put back
	b = append(b, 1, 2, 3)
	bytesPool.Put(b)

	// Get again and check reset
	b = bytesPool.Get()
	if len(b) != 0 {
		t.Errorf("Expected byte slice length 0 after reset, got %d", len(b))
	}

	// Change the size of b
	b = make([]byte, size+1)
	bytesPool.Put(b)
	b = bytesPool.Get()
	if cap(b) != size+1 {
		t.Errorf("Expected byte slice with capacity %d, got %d", size+1, cap(b))
	}

	bytesPool = NewBytesPool(size, size+1)
	b = make([]byte, size+2)
	bytesPool.Put(b)
	b = bytesPool.Get()
	if cap(b) != size {
		t.Errorf("Expected byte slice with capacity %d, got %d", size, cap(b))
	}

	bytesPool = NewBytesPool(size, 0)
	b = nil
	bytesPool.Put(b)
	b = bytesPool.Get()
	if cap(b) != size {
		t.Errorf("Expected byte slice with capacity %d, got %d", size, cap(b))
	}
}
