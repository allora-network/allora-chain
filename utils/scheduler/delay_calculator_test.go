package scheduler

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateEmissionScheduleDelay(t *testing.T) {
	oneMonth := 30 * 24 * time.Hour
	blocksPerMonth := uint64(43200) // Example: ~1 month worth of blocks

	t.Run("zero blocks per month returns default duration", func(t *testing.T) {
		result := CalculateEmissionScheduleDelay(100, 0)

		assert.Equal(t, oneMonth, result.InitialDelay)
		assert.Equal(t, uint64(0), result.BlocksRemaining)
	})

	t.Run("zero block height", func(t *testing.T) {
		result := CalculateEmissionScheduleDelay(0, blocksPerMonth)

		// With blockHeight = 0, blocksElapsed = 0, so blocksRemaining = blocksPerMonth
		assert.Equal(t, blocksPerMonth, result.BlocksRemaining)

		// Should return the full month duration
		expectedNanoseconds := oneMonth.Nanoseconds()
		assert.Equal(t, time.Duration(expectedNanoseconds), result.InitialDelay)
	})

	t.Run("block height 1 - start of first cycle", func(t *testing.T) {
		result := CalculateEmissionScheduleDelay(1, blocksPerMonth)

		// blocksElapsed = (1-1) % blocksPerMonth = 0
		// blocksRemaining = blocksPerMonth - 0 = blocksPerMonth
		assert.Equal(t, blocksPerMonth, result.BlocksRemaining)
		assert.Equal(t, oneMonth, result.InitialDelay)
	})

	t.Run("middle of cycle", func(t *testing.T) {
		blockHeight := uint64(21600) // Half way through the cycle
		result := CalculateEmissionScheduleDelay(blockHeight, blocksPerMonth)

		// blocksElapsed = (21600-1) % 43200 = 21599
		// blocksRemaining = 43200 - 21599 = 21601
		expectedBlocksRemaining := blocksPerMonth - (blockHeight-1)%blocksPerMonth
		assert.Equal(t, expectedBlocksRemaining, result.BlocksRemaining)

		// Should be roughly half a month
		halfMonth := oneMonth / 2
		assert.InDelta(t, halfMonth.Nanoseconds(), result.InitialDelay.Nanoseconds(), float64(oneMonth.Nanoseconds())*0.01)
	})

	t.Run("exactly on cycle boundary", func(t *testing.T) {
		// Block height that would result in blocksRemaining = 0
		blockHeight := blocksPerMonth + 1 // This should land exactly on boundary
		result := CalculateEmissionScheduleDelay(blockHeight, blocksPerMonth)

		// When blocksRemaining would be 0, it should be set to blocksPerMonth
		assert.Equal(t, blocksPerMonth, result.BlocksRemaining)
		assert.Equal(t, oneMonth, result.InitialDelay)
	})

	t.Run("second cycle start", func(t *testing.T) {
		// Test a block height in the second cycle
		blockHeight := blocksPerMonth + 100
		result := CalculateEmissionScheduleDelay(blockHeight, blocksPerMonth)

		// blocksElapsed = (blocksPerMonth + 100 - 1) % blocksPerMonth = 99
		// blocksRemaining = blocksPerMonth - 99
		expectedBlocksRemaining := blocksPerMonth - 99
		assert.Equal(t, expectedBlocksRemaining, result.BlocksRemaining)

		// Should be close to full month minus a small amount
		expectedRatio := float64(expectedBlocksRemaining) / float64(blocksPerMonth)
		expectedDelay := time.Duration(float64(oneMonth.Nanoseconds()) * expectedRatio)
		assert.InDelta(t, expectedDelay.Nanoseconds(), result.InitialDelay.Nanoseconds(), float64(oneMonth.Nanoseconds())*0.001)
	})

	t.Run("fractional handling verification", func(t *testing.T) {
		// Use numbers that will create fractional parts
		blocksPerMonth := uint64(7) // Small number to ensure fractional parts
		blockHeight := uint64(4)

		result := CalculateEmissionScheduleDelay(blockHeight, blocksPerMonth)

		// blocksElapsed = (4-1) % 7 = 3
		// blocksRemaining = 7 - 3 = 4
		assert.Equal(t, uint64(4), result.BlocksRemaining)

		// Verify the calculation includes fractional handling
		monthNanos := oneMonth.Nanoseconds()

		// Use safe conversion function
		blocksPerMonthInt64 := safeUint64ToInt64(blocksPerMonth)
		blocksRemainingInt64 := safeUint64ToInt64(result.BlocksRemaining)

		perBlockNanos := monthNanos / blocksPerMonthInt64
		expectedNanos := perBlockNanos * blocksRemainingInt64

		// Add fractional part
		if remainder := monthNanos % blocksPerMonthInt64; remainder > 0 {
			expectedNanos += remainder * blocksRemainingInt64 / blocksPerMonthInt64
		}

		assert.Equal(t, time.Duration(expectedNanos), result.InitialDelay)
	})

	t.Run("large block height consistency", func(t *testing.T) {
		// Test with very large block heights to ensure no overflow issues
		largeBlockHeight := uint64(1000000000) // 1 billion blocks
		result := CalculateEmissionScheduleDelay(largeBlockHeight, blocksPerMonth)

		// Should still calculate correctly
		assert.Positive(t, result.InitialDelay)
		assert.Positive(t, result.BlocksRemaining)
		assert.LessOrEqual(t, result.BlocksRemaining, blocksPerMonth)
	})

	t.Run("various block heights in same cycle", func(t *testing.T) {
		// Test multiple block heights within the same cycle to ensure monotonic behavior
		baseBlockHeight := uint64(10000)
		var previousDelay time.Duration

		for i := uint64(0); i < 10; i++ {
			blockHeight := baseBlockHeight + i
			result := CalculateEmissionScheduleDelay(blockHeight, blocksPerMonth)

			if i > 0 {
				// As we progress through the cycle, delay should decrease
				assert.Less(t, result.InitialDelay, previousDelay, "delay should decrease as we progress through cycle")
			}
			previousDelay = result.InitialDelay
		}
	})

	t.Run("integer overflow handling", func(t *testing.T) {
		// Test with values that would overflow int64 to ensure safe conversion
		maxUint64 := uint64(math.MaxUint64)
		overflowBlocksPerMonth := maxUint64

		// Should handle overflow gracefully without panic
		result := CalculateEmissionScheduleDelay(1000, overflowBlocksPerMonth)

		// Should return valid results
		assert.Positive(t, result.InitialDelay)
		assert.Positive(t, result.BlocksRemaining)

		// Test with large but valid blocksRemaining
		largeBlockHeight := uint64(math.MaxInt64) + 1000
		normalBlocksPerMonth := uint64(43200)

		result2 := CalculateEmissionScheduleDelay(largeBlockHeight, normalBlocksPerMonth)
		assert.Positive(t, result2.InitialDelay)
		assert.Positive(t, result2.BlocksRemaining)
		assert.LessOrEqual(t, result2.BlocksRemaining, normalBlocksPerMonth)
	})
}

func TestCalculateEmissionDelayResult(t *testing.T) {
	t.Run("result struct contains expected fields", func(t *testing.T) {
		result := CalculateEmissionDelayResult{
			InitialDelay:    time.Hour,
			BlocksRemaining: 100,
		}

		assert.Equal(t, time.Hour, result.InitialDelay)
		assert.Equal(t, uint64(100), result.BlocksRemaining)
	})
}

// Benchmark tests to ensure the function is performant
func BenchmarkCalculateEmissionScheduleDelay(b *testing.B) {
	blocksPerMonth := uint64(43200)
	blockHeight := uint64(21600)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateEmissionScheduleDelay(blockHeight, blocksPerMonth)
	}
}
