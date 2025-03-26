package math

import (
	"sync"
)

// MathOperationsCacheKey represents a key for the mathematical operations cache
type MathOperationsCacheKey struct {
	Op   string
	Base string
}

// MathOperationsCache provides a cache for expensive mathematical operations
type MathOperationsCache struct {
	mu      sync.RWMutex
	cache   map[MathOperationsCacheKey]Dec
	enabled bool // Track if cache is enabled
}

var (
	// Global cache instance
	mathOperationsCache = NewMathOperationsCache()
)

// NewMathOperationsCache creates a new cache for expensive math operations
func NewMathOperationsCache() *MathOperationsCache {
	return &MathOperationsCache{
		cache:   make(map[MathOperationsCacheKey]Dec),
		enabled: false, // Disabled by default
	}
}

// Get retrieves a cached result
func (c *MathOperationsCache) Get(key MathOperationsCacheKey) (Dec, bool) {
	// Skip cache lookup if disabled
	if !c.enabled {
		return Dec{}, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.cache[key]
	return val, ok
}

// Set stores a result in the cache
func (c *MathOperationsCache) Set(key MathOperationsCacheKey, value Dec) {
	// Skip caching if disabled
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = value
}

// Clear empties the cache
func (c *MathOperationsCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[MathOperationsCacheKey]Dec)
}

// EnableCache turns on caching
func (c *MathOperationsCache) EnableCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = true
}

// DisableCache turns off caching
func (c *MathOperationsCache) DisableCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = false
}

// ClearMathOperationsCache clears the global math operations cache
func ClearMathOperationsCache() {
	mathOperationsCache.Clear()
}

// EnableMathOperationsCache enables the global math operations cache
func EnableMathOperationsCache() {
	mathOperationsCache.EnableCache()
}

// DisableMathOperationsCache disables the global math operations cache
func DisableMathOperationsCache() {
	mathOperationsCache.DisableCache()
}