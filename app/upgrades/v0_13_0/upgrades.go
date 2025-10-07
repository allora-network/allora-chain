package v0_13_0 //nolint:revive // var-naming: don't use an underscore in package name

import (
	"context"
	"strings"

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

			if err := appKeepers.MintKeeper.ScheduleEmissionRecalculationTask(ctx, appKeepers.SchedulerKeeper); err != nil {
				sdkCtx.Logger().Error("failed to schedule emission recalculation task", "err", err)
				return vm, err
			}

			sdkCtx.Logger().Info("scheduled emission recalculation task")
		}

		sdkCtx.Logger().Info("MIGRATIONS COMPLETED")
		return vm, nil
	}
}
