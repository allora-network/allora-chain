package v0_13_0 //nolint:revive // var-naming: don't use an underscore in package name

import (
	"context"
	"strings"
	"time"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/allora-network/allora-chain/app/keepers"
	"github.com/allora-network/allora-chain/app/upgrades"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/pkg/errors"
)

const (
	UpgradeName = "v0.13.0"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{Added: []string{schedulertypes.StoreKey}, Renamed: nil, Deleted: nil},
}

func CreateUpgradeHandler(
	moduleManager *module.Manager,
	configurator module.Configurator,
	appKeepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info("RUN MIGRATIONS")
		vm, err := moduleManager.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		if appKeepers != nil && appKeepers.SchedulerKeeper != nil {
			mintTaskHandlers := appKeepers.MintKeeper.TaskHandlers()
			if err := appKeepers.SchedulerKeeper.RegisterTaskHandlers(mintTaskHandlers); err != nil {
				if !errors.Is(err, schedulertypes.ErrInvalidTaskHandler) || !strings.Contains(err.Error(), "duplicate task handler") {
					sdkCtx.Logger().Error("failed to register mint task handlers", "err", err)
					return vm, err
				}
			}

			oneMonthDuration := 30 * 24 * time.Hour

			// Pull the currently configured cadence so we can align the scheduler with the pre-upgrade emission cycle.
			blocksPerMonth, err := appKeepers.MintKeeper.GetParamsBlocksPerMonth(ctx)
			if err != nil {
				sdkCtx.Logger().Error("failed to get blocks per month", "err", err)
				return vm, err
			}

			initialDelay := oneMonthDuration
			var blocksRemaining uint64
			if blocksPerMonth > 0 {
				// Use the latest block height as the anchor for the alignment calculations.
				blockHeight := uint64(0)
				if sdkCtx.BlockHeight() > 0 {
					blockHeight = uint64(sdkCtx.BlockHeight())
				}

				var blocksElapsed uint64
				if blockHeight > 0 {
					blocksElapsed = (blockHeight - 1) % blocksPerMonth
				}

				// Remaining blocks until we hit the next monthly emission checkpoint.
				blocksRemaining = blocksPerMonth - blocksElapsed
				if blocksRemaining == 0 {
					// If we landed exactly on the boundary we still want the next run a full month later.
					blocksRemaining = blocksPerMonth
				}

				// Convert the remaining block count into real time so the scheduler can use a relative delay.
				monthNanoseconds := oneMonthDuration.Nanoseconds()
				perBlockNanoseconds := monthNanoseconds / int64(blocksPerMonth)
				remainingNanoseconds := perBlockNanoseconds * int64(blocksRemaining)

				// Carry the fractional part of the division to avoid monthly drift.
				if remainder := monthNanoseconds % int64(blocksPerMonth); remainder > 0 {
					remainingNanoseconds += remainder * int64(blocksRemaining) / int64(blocksPerMonth)
				}

				initialDelay = time.Duration(remainingNanoseconds)
				if initialDelay <= 0 {
					// Guard against rounding corner cases that would otherwise schedule in the past.
					initialDelay = time.Second
				}
			}

			sdkCtx.Logger().Info("calculated emission recalculation schedule",
				"interval", oneMonthDuration,
				"initialDelay", initialDelay,
				"blocksRemaining", blocksRemaining,
			)

			if err := appKeepers.MintKeeper.ScheduleEmissionRecalculationTask(ctx, appKeepers.SchedulerKeeper, initialDelay); err != nil {
				sdkCtx.Logger().Error("failed to schedule emission recalculation task", "err", err)
				return vm, err
			}

			sdkCtx.Logger().Info("scheduled emission recalculation task")
		}

		sdkCtx.Logger().Info("MIGRATIONS COMPLETED")
		return vm, nil
	}
}
