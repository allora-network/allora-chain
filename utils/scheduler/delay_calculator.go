package scheduler

import (
	"math"
	"time"
)

// safeUint64ToInt64 safely converts uint64 to int64, capping at MaxInt64 if overflow would occur
func safeUint64ToInt64(val uint64) int64 {
	if val > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(val)
}

// CalculateEmissionDelayResult holds the results of emission delay calculation
type CalculateEmissionDelayResult struct {
	InitialDelay    time.Duration
	BlocksRemaining uint64
}

// CalculateEmissionScheduleDelay calculates the initial delay and blocks remaining
// for scheduling emission recalculation tasks based on the current block height
// and the configured blocks per month cycle.
//
// Parameters:
//   - blockHeight: The current block height
//   - blocksPerMonth: The number of blocks that constitute one emission cycle (month)
//
// Returns:
//   - CalculateEmissionDelayResult containing the calculated initial delay and blocks remaining
//
// The function aligns the scheduler with the existing emission cycle by calculating
// how many blocks remain until the next monthly emission checkpoint, then converts
// that to a time duration while handling fractional parts to avoid drift.
func CalculateEmissionScheduleDelay(blockHeight uint64, blocksPerMonth uint64) CalculateEmissionDelayResult {
	monthDuration := 30 * 24 * time.Hour

	// Default to full month duration if blocksPerMonth is 0 or invalid
	if blocksPerMonth == 0 {
		return CalculateEmissionDelayResult{
			InitialDelay:    monthDuration,
			BlocksRemaining: 0,
		}
	}

	var blocksElapsed uint64
	if blockHeight > 0 {
		blocksElapsed = (blockHeight - 1) % blocksPerMonth
	}

	// Remaining blocks until we hit the next monthly emission checkpoint.
	blocksRemaining := blocksPerMonth - blocksElapsed
	if blocksRemaining == 0 {
		// If we landed exactly on the boundary we still want the next run a full month later.
		blocksRemaining = blocksPerMonth
	}

	// Convert the remaining block count into real time so the scheduler can use a relative delay.
	monthNanoseconds := monthDuration.Nanoseconds()
	blocksPerMonthInt64 := safeUint64ToInt64(blocksPerMonth)
	blocksRemainingInt64 := safeUint64ToInt64(blocksRemaining)

	perBlockNanoseconds := monthNanoseconds / blocksPerMonthInt64
	remainingNanoseconds := perBlockNanoseconds * blocksRemainingInt64

	// Carry the fractional part of the division to avoid monthly drift.
	if remainder := monthNanoseconds % blocksPerMonthInt64; remainder > 0 {
		remainingNanoseconds += remainder * blocksRemainingInt64 / blocksPerMonthInt64
	}

	initialDelay := time.Duration(remainingNanoseconds)
	if initialDelay <= 0 {
		// Guard against rounding corner cases that would otherwise schedule in the past.
		initialDelay = time.Second
	}

	return CalculateEmissionDelayResult{
		InitialDelay:    initialDelay,
		BlocksRemaining: blocksRemaining,
	}
}
