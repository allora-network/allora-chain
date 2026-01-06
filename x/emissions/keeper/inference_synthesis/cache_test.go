package inferencesynthesis_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	alloraMath "github.com/allora-network/allora-chain/math"
	synth "github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
)

type CacheTestSuite struct {
	testutil.TestSuite
}

func TestCacheTestSuite(t *testing.T) {
	suite.Run(t, &CacheTestSuite{
		testutil.NewTestSuite("inference_synthesis_cache"),
	})
}

func (s *CacheTestSuite) TestNewMathHelperCache() {
	cache := synth.NewMathHelperCache()
	s.Require().NotNil(cache)

	// Cache should be disabled by default
	a := alloraMath.MustNewDecFromString("1.0")
	b := alloraMath.MustNewDecFromString("2.0")

	// Should not be in cache (disabled)
	s.Require().False(cache.IsInCache(a, b))

	result1, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)

	// Still should not be in cache (disabled)
	s.Require().False(cache.IsInCache(a, b))

	// Second call should work but not use cache (disabled)
	result2, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result2))
}

func (s *CacheTestSuite) TestEnable() {
	cache := synth.NewMathHelperCache()

	a := alloraMath.MustNewDecFromString("1.0")
	b := alloraMath.MustNewDecFromString("2.0")

	// Initially disabled - should not be in cache
	s.Require().False(cache.IsInCache(a, b))

	_, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)

	// Still should not be in cache (disabled)
	s.Require().False(cache.IsInCache(a, b))

	// Enable cache
	cache.Enable()

	// Still not in cache yet (haven't called with cache enabled)
	s.Require().False(cache.IsInCache(a, b))

	// Now make a call - should cache
	result1, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)

	// Should now be in cache
	s.Require().True(cache.IsInCache(a, b))

	// Second call with same inputs should return same result (cached)
	result2, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result2))
	s.Require().True(cache.IsInCache(a, b))
}

func (s *CacheTestSuite) TestDisable() {
	cache := synth.NewMathHelperCache()
	cache.Enable()

	a := alloraMath.MustNewDecFromString("1.0")
	b := alloraMath.MustNewDecFromString("2.0")

	// Should not be in cache initially
	s.Require().False(cache.IsInCache(a, b))

	// Make a call to populate cache
	result1, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)

	// Should now be in cache
	s.Require().True(cache.IsInCache(a, b))

	// Verify it's cached by calling again
	result2, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result2))
	s.Require().True(cache.IsInCache(a, b))

	// Disable cache
	cache.Disable()

	// IsInCache should return false when disabled (even if entry exists)
	s.Require().False(cache.IsInCache(a, b))

	// Call again - should still work but not use cache
	result3, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result3))

	// Still should return false (disabled)
	s.Require().False(cache.IsInCache(a, b))

	// Re-enable cache
	cache.Enable()

	// Should still be in cache (Disable doesn't clear, just disables)
	s.Require().True(cache.IsInCache(a, b))

	// Should return cached value
	result4, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result4))
}

func (s *CacheTestSuite) TestClear() {
	cache := synth.NewMathHelperCache()
	cache.Enable()

	// Add some values to cache
	a1 := alloraMath.MustNewDecFromString("1.0")
	b1 := alloraMath.MustNewDecFromString("2.0")
	result1, err := cache.Exp1DivExp1(a1, b1)
	s.Require().NoError(err)
	s.Require().True(cache.IsInCache(a1, b1))

	a2 := alloraMath.MustNewDecFromString("3.0")
	b2 := alloraMath.MustNewDecFromString("4.0")
	_, err = cache.Exp1DivExp1(a2, b2)
	s.Require().NoError(err)
	s.Require().True(cache.IsInCache(a2, b2))

	// Verify both are cached by calling again
	result1Again, err := cache.Exp1DivExp1(a1, b1)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result1Again))

	// Clear cache
	cache.Clear()

	// Both should no longer be in cache
	s.Require().False(cache.IsInCache(a1, b1))
	s.Require().False(cache.IsInCache(a2, b2))

	// Cache should still be enabled after clear
	// But entries should be cleared - verify by making new calls
	// The results should still be correct, but cache was cleared
	result1AfterClear, err := cache.Exp1DivExp1(a1, b1)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result1AfterClear))

	// Now it should be cached again
	s.Require().True(cache.IsInCache(a1, b1))
	result1AgainAfterClear, err := cache.Exp1DivExp1(a1, b1)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result1AgainAfterClear))
}

func (s *CacheTestSuite) TestExp1DivExp1_DisabledCache() {
	cache := synth.NewMathHelperCache()
	// Don't enable cache

	a := alloraMath.MustNewDecFromString("1.0")
	b := alloraMath.MustNewDecFromString("2.0")

	// Should not be in cache (disabled)
	s.Require().False(cache.IsInCache(a, b))

	// First call
	result1, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.IsFinite())
	s.Require().True(result1.IsPositive())

	// Still should not be in cache (disabled)
	s.Require().False(cache.IsInCache(a, b))

	// Second call with same inputs - should recalculate (no cache)
	result2, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)

	// Results should be equal (same calculation)
	s.Require().True(result1.Equal(result2))
	s.Require().False(cache.IsInCache(a, b))
}

func (s *CacheTestSuite) TestExp1DivExp1_EnabledCache_CacheHit() {
	cache := synth.NewMathHelperCache()
	cache.Enable()

	a := alloraMath.MustNewDecFromString("1.0")
	b := alloraMath.MustNewDecFromString("2.0")

	// Should not be in cache initially
	s.Require().False(cache.IsInCache(a, b))

	// First call - should calculate and cache
	result1, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.IsFinite())
	s.Require().True(result1.IsPositive())

	// Should now be in cache
	s.Require().True(cache.IsInCache(a, b))

	// Second call with same inputs - should return cached value
	result2, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)

	// Results should be identical (from cache)
	s.Require().True(result1.Equal(result2))
	s.Require().True(cache.IsInCache(a, b))

	// Third call - should also use cache
	result3, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result3))
	s.Require().True(cache.IsInCache(a, b))
}

func (s *CacheTestSuite) TestExp1DivExp1_EnabledCache_DifferentInputs() {
	cache := synth.NewMathHelperCache()
	cache.Enable()

	// First call
	a1 := alloraMath.MustNewDecFromString("1.0")
	b1 := alloraMath.MustNewDecFromString("2.0")
	s.Require().False(cache.IsInCache(a1, b1))
	result1, err := cache.Exp1DivExp1(a1, b1)
	s.Require().NoError(err)
	s.Require().True(cache.IsInCache(a1, b1))

	// Second call with different inputs - should calculate new value
	a2 := alloraMath.MustNewDecFromString("3.0")
	b2 := alloraMath.MustNewDecFromString("4.0")
	s.Require().False(cache.IsInCache(a2, b2))
	result2, err := cache.Exp1DivExp1(a2, b2)
	s.Require().NoError(err)
	s.Require().True(cache.IsInCache(a2, b2))

	// Results should be different
	s.Require().False(result1.Equal(result2))

	// Verify both are cached by calling again
	result1Again, err := cache.Exp1DivExp1(a1, b1)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result1Again))
	s.Require().True(cache.IsInCache(a1, b1))

	result2Again, err := cache.Exp1DivExp1(a2, b2)
	s.Require().NoError(err)
	s.Require().True(result2.Equal(result2Again))
	s.Require().True(cache.IsInCache(a2, b2))
}

func (s *CacheTestSuite) TestExp1DivExp1_CacheKeyGeneration() {
	cache := synth.NewMathHelperCache()
	cache.Enable()

	// Test that cache keys are based on string representation
	// Same numeric value should produce same cache key
	a1 := alloraMath.MustNewDecFromString("1.0")
	b1 := alloraMath.MustNewDecFromString("2.0")

	a2 := alloraMath.MustNewDecFromString("1.0")
	b2 := alloraMath.MustNewDecFromString("2.0")

	// First call
	result1, err := cache.Exp1DivExp1(a1, b1)
	s.Require().NoError(err)

	// Second call with same values (even if different Dec instances)
	result2, err := cache.Exp1DivExp1(a2, b2)
	s.Require().NoError(err)

	// Results should be mathematically equal
	s.Require().True(result1.Equal(result2))

	// And should use same cache entry (verify by calling first again)
	result1Again, err := cache.Exp1DivExp1(a1, b1)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result1Again))
}

func (s *CacheTestSuite) TestExp1DivExp1_CacheAfterClear() {
	cache := synth.NewMathHelperCache()
	cache.Enable()

	a := alloraMath.MustNewDecFromString("1.0")
	b := alloraMath.MustNewDecFromString("2.0")

	// First call - cache it
	result1, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(cache.IsInCache(a, b))

	// Verify it's cached
	result1Again, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result1Again))

	// Clear cache
	cache.Clear()

	// Should no longer be in cache
	s.Require().False(cache.IsInCache(a, b))

	// Call again - should recalculate (not cached)
	result2, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)

	// Results should be equal (same calculation)
	s.Require().True(result1.Equal(result2))

	// But now it should be cached again
	s.Require().True(cache.IsInCache(a, b))
	result2Again, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result2.Equal(result2Again))
}

func (s *CacheTestSuite) TestExp1DivExp1_EnableAfterUse() {
	cache := synth.NewMathHelperCache()
	// Start with cache disabled

	a := alloraMath.MustNewDecFromString("1.0")
	b := alloraMath.MustNewDecFromString("2.0")

	// Should not be in cache (disabled)
	s.Require().False(cache.IsInCache(a, b))

	// Call with cache disabled
	result1, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().False(cache.IsInCache(a, b))

	// Enable cache
	cache.Enable()

	// Still not in cache (wasn't cached when disabled)
	s.Require().False(cache.IsInCache(a, b))

	// Call again - should calculate and cache this time
	result2, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result2))
	s.Require().True(cache.IsInCache(a, b))

	// Third call - should use cache
	result3, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result2.Equal(result3))
	s.Require().True(cache.IsInCache(a, b))
}

func (s *CacheTestSuite) TestExp1DivExp1_DisableAfterUse() {
	cache := synth.NewMathHelperCache()
	cache.Enable()

	a := alloraMath.MustNewDecFromString("1.0")
	b := alloraMath.MustNewDecFromString("2.0")

	// Call with cache enabled - should cache
	result1, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(cache.IsInCache(a, b))

	// Verify it's cached
	result1Again, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result1Again))

	// Disable cache
	cache.Disable()

	// IsInCache should return false when disabled
	s.Require().False(cache.IsInCache(a, b))

	// Call again - should recalculate (cache disabled)
	result2, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result2))
	s.Require().False(cache.IsInCache(a, b))

	// Re-enable cache
	cache.Enable()

	// Should still be in cache (Disable doesn't clear)
	s.Require().True(cache.IsInCache(a, b))

	// Call again - should use cached value
	result3, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result3))
	s.Require().True(cache.IsInCache(a, b))

	// Should still be cached
	result3Again, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result3.Equal(result3Again))
}

func (s *CacheTestSuite) TestCacheIsolation() {
	// Test that different cache instances don't interfere
	cache1 := synth.NewMathHelperCache()
	cache1.Enable()

	cache2 := synth.NewMathHelperCache()
	cache2.Enable()

	a := alloraMath.MustNewDecFromString("1.0")
	b := alloraMath.MustNewDecFromString("2.0")

	// Use cache1
	s.Require().False(cache1.IsInCache(a, b))
	result1, err := cache1.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(cache1.IsInCache(a, b))

	// cache2 should not have it
	s.Require().False(cache2.IsInCache(a, b))

	// Verify cache1 has it cached
	result1Again, err := cache1.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result1Again))

	// Use cache2 - should calculate independently
	result2, err := cache2.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result2))
	s.Require().True(cache2.IsInCache(a, b))

	// Verify cache2 now has it cached
	result2Again, err := cache2.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result2.Equal(result2Again))

	// Clear cache1 - cache2 should be unaffected
	cache1.Clear()
	s.Require().False(cache1.IsInCache(a, b))
	s.Require().True(cache2.IsInCache(a, b))

	// cache1 should recalculate
	result1AfterClear, err := cache1.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result1AfterClear))
	s.Require().True(cache1.IsInCache(a, b))

	// cache2 should still use cache
	result2AfterClear, err := cache2.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result2.Equal(result2AfterClear))
	s.Require().True(cache2.IsInCache(a, b))
}

func (s *CacheTestSuite) TestExp1DivExp1_ErrorPropagation() {
	cache := synth.NewMathHelperCache()
	cache.Enable()

	// Test with NaN - should return error and not cache
	nan := alloraMath.NewNaN()
	valid := alloraMath.MustNewDecFromString("1.0")

	_, err := cache.Exp1DivExp1(nan, valid)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "NaN")

	// Should not be cached (errors aren't cached)
	s.Require().False(cache.IsInCache(nan, valid))

	// Try again - should still error (not cached)
	_, err = cache.Exp1DivExp1(nan, valid)
	s.Require().Error(err)
	s.Require().False(cache.IsInCache(nan, valid))

	// Valid call should still work
	result, err := cache.Exp1DivExp1(valid, valid)
	s.Require().NoError(err)
	s.Require().True(result.IsFinite())
	s.Require().True(cache.IsInCache(valid, valid))
}

func (s *CacheTestSuite) TestExp1DivExp1_MultipleCalls_Performance() {
	cache := synth.NewMathHelperCache()
	cache.Enable()

	// Test with multiple different inputs
	inputs := []struct {
		a, b string
	}{
		{"1.0", "2.0"},
		{"3.0", "4.0"},
		{"5.0", "6.0"},
		{"1.0", "2.0"}, // Duplicate - should use cache
		{"3.0", "4.0"}, // Duplicate - should use cache
		{"7.0", "8.0"},
		{"1.0", "2.0"}, // Another duplicate
	}

	results := make([]alloraMath.Dec, 0, len(inputs))
	for i, input := range inputs {
		a := alloraMath.MustNewDecFromString(input.a)
		b := alloraMath.MustNewDecFromString(input.b)
		result, err := cache.Exp1DivExp1(a, b)
		s.Require().NoError(err, "Failed on input %d", i)
		results = append(results, result)
	}

	// Verify duplicates return same results (cached)
	s.Require().True(results[0].Equal(results[3]), "First and fourth should be equal (cached)")
	s.Require().True(results[0].Equal(results[6]), "First and seventh should be equal (cached)")
	s.Require().True(results[1].Equal(results[4]), "Second and fifth should be equal (cached)")

	// Verify unique inputs produce different results
	s.Require().False(results[0].Equal(results[1]), "Different inputs should produce different results")
	s.Require().False(results[1].Equal(results[2]), "Different inputs should produce different results")
	s.Require().False(results[2].Equal(results[5]), "Different inputs should produce different results")
}

func (s *CacheTestSuite) TestExp1DivExp1_EdgeCaseValues() {
	cache := synth.NewMathHelperCache()
	cache.Enable()

	testCases := []struct {
		name string
		a    string
		b    string
	}{
		{"zero values", "0.0", "0.0"},
		{"very small values", "0.0001", "0.0002"},
		{"very large values", "100.0", "200.0"},
		{"negative values", "-1.0", "-2.0"},
		{"mixed signs", "-1.0", "1.0"},
		{"equal values", "5.0", "5.0"},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			a := alloraMath.MustNewDecFromString(tc.a)
			b := alloraMath.MustNewDecFromString(tc.b)

			// Should not be in cache initially
			s.Require().False(cache.IsInCache(a, b))

			// First call
			result1, err := cache.Exp1DivExp1(a, b)
			s.Require().NoError(err)
			s.Require().True(result1.IsFinite())
			s.Require().True(cache.IsInCache(a, b))

			// Second call - should use cache
			result2, err := cache.Exp1DivExp1(a, b)
			s.Require().NoError(err)
			s.Require().True(result1.Equal(result2))
			s.Require().True(cache.IsInCache(a, b))
		})
	}
}

func (s *CacheTestSuite) TestExp1DivExp1_ReEnableAfterDisable() {
	cache := synth.NewMathHelperCache()
	cache.Enable()

	a := alloraMath.MustNewDecFromString("1.0")
	b := alloraMath.MustNewDecFromString("2.0")

	// Call with cache enabled
	result1, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(cache.IsInCache(a, b))

	// Disable
	cache.Disable()
	s.Require().False(cache.IsInCache(a, b))

	// Call while disabled
	result2, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result2))
	s.Require().False(cache.IsInCache(a, b))

	// Re-enable
	cache.Enable()

	// Should still be in cache (Disable doesn't clear)
	s.Require().True(cache.IsInCache(a, b))

	// Call again - should use cached value
	result3, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result1.Equal(result3))
	s.Require().True(cache.IsInCache(a, b))

	// Should still be cached
	result3Again, err := cache.Exp1DivExp1(a, b)
	s.Require().NoError(err)
	s.Require().True(result3.Equal(result3Again))
	s.Require().True(cache.IsInCache(a, b))
}
