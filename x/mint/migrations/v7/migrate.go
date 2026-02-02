package v7

import (
	"time"

	"github.com/allora-network/allora-chain/x/mint/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MigrateStore aligns the emission recalculation schedule when upgrading to v7.
func MigrateStore(ctx sdk.Context, mintKeeper keeper.Keeper) error {
	ctx.Logger().Info("MIGRATING MINT MODULE FROM VERSION 6 TO VERSION 7")

	blocksPerMonth, err := mintKeeper.GetParamsBlocksPerMonth(ctx)
	if err != nil {
		ctx.Logger().Error("failed to get blocks per month", "err", err)
		return err
	}

	oneMonthDuration := time.Hour * 24 * 30
	initialDelay := oneMonthDuration
	var blocksRemaining uint64

	if blocksPerMonth > 0 {
		blockHeight := uint64(0)
		if ctx.BlockHeight() > 0 {
			blockHeight = uint64(ctx.BlockHeight())
		}

		var blocksElapsed uint64
		if blockHeight > 0 {
			blocksElapsed = (blockHeight - 1) % blocksPerMonth
		}

		blocksRemaining = blocksPerMonth - blocksElapsed
		if blocksRemaining == 0 {
			blocksRemaining = blocksPerMonth
		}

		monthNanoseconds := oneMonthDuration.Nanoseconds()
		//nolint:gosec // blocksPerMonth is validated to be > 0 and within safe bounds
		perBlockNanoseconds := monthNanoseconds / int64(blocksPerMonth)
		//nolint:gosec // blocksRemaining <= blocksPerMonth, safe conversion
		remainingNanoseconds := perBlockNanoseconds * int64(blocksRemaining)
		//nolint:gosec // same as above
		if remainder := monthNanoseconds % int64(blocksPerMonth); remainder > 0 {
			//nolint:gosec // same as above
			remainingNanoseconds += remainder * int64(blocksRemaining) / int64(blocksPerMonth)
		}

		initialDelay = time.Duration(remainingNanoseconds)
		if initialDelay <= 0 {
			initialDelay = time.Second
		}
	}

	ctx.Logger().Info("calculated emission recalculation schedule",
		"interval", oneMonthDuration,
		"initialDelay", initialDelay,
		"blocksRemaining", blocksRemaining,
	)

	if err := mintKeeper.EnsureEmissionRecalculationTaskScheduled(ctx, initialDelay); err != nil {
		ctx.Logger().Error("failed to schedule emission recalculation task", "err", err)
		return err
	}

	ctx.Logger().Info("MIGRATING MINT MODULE FROM VERSION 6 TO VERSION 7 COMPLETE")
	return nil
}
