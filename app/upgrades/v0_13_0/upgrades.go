package v0_13_0 //nolint:revive // var-naming: don't use an underscore in package name

import (
	"context"
	"strings"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/allora-network/allora-chain/app/keepers"
	"github.com/allora-network/allora-chain/app/upgrades"
	"github.com/allora-network/allora-chain/utils/scheduler"
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
			// Check emissions are enabled to calculate when to start the scheduler
			moduleParams, err := appKeepers.MintKeeper.GetParams(ctx)
			if err != nil {
				sdkCtx.Logger().Error("failed to get module params", "err", err)
				return vm, err
			}

			// Check starting block is not set
			startingEmissionsBlockHeight, err := appKeepers.MintKeeper.GetStartingEmissionsBlockHeight(ctx)
			if err != nil {
				sdkCtx.Logger().Error("failed to get starting emissions block height", "err", err)
				return vm, err
			}
			// Only schedule the task if emissions are enabled
			if !moduleParams.EmissionEnabled && startingEmissionsBlockHeight == 0 {

				mintTaskHandlers := appKeepers.MintKeeper.TaskHandlers()
				if err := appKeepers.SchedulerKeeper.RegisterTaskHandlers(mintTaskHandlers); err != nil {
					if !errors.Is(err, schedulertypes.ErrInvalidTaskHandler) || !strings.Contains(err.Error(), "duplicate task handler") {
						sdkCtx.Logger().Error("failed to register mint task handlers", "err", err)
						return vm, err
					}
				}

				// Pull the currently configured cadence so we can align the scheduler with the pre-upgrade emission cycle.
				blocksPerMonth, err := appKeepers.MintKeeper.GetParamsBlocksPerMonth(ctx)
				if err != nil {
					sdkCtx.Logger().Error("failed to get blocks per month", "err", err)
					return vm, err
				}

				// Use the latest block height as the anchor for the alignment calculations.
				blockHeight := uint64(0)
				if sdkCtx.BlockHeight() > 0 {
					blockHeight = uint64(sdkCtx.BlockHeight())
				}

				// Calculate the emission schedule delay using the utility function
				result := scheduler.CalculateEmissionScheduleDelay(blockHeight, blocksPerMonth)
				initialDelay := result.InitialDelay
				blocksRemaining := result.BlocksRemaining

				sdkCtx.Logger().Info("calculated emission recalculation schedule",
					"initialDelay", initialDelay,
					"blocksRemaining", blocksRemaining,
				)

				if err := appKeepers.MintKeeper.ScheduleEmissionRecalculationTask(ctx, appKeepers.SchedulerKeeper, initialDelay); err != nil {
					sdkCtx.Logger().Error("failed to schedule emission recalculation task", "err", err)
					return vm, err
				}
				sdkCtx.Logger().Info("scheduled emission recalculation task")
			} else {
				sdkCtx.Logger().Info("Emissions are disabled - not scheduled emission recalculation task")
			}

		}

		sdkCtx.Logger().Info("MIGRATIONS COMPLETED")
		return vm, nil
	}
}
