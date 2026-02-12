package v0_16_0 //nolint:revive // var-naming: don't use an underscore in package name

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/allora-network/allora-chain/app/keepers"
	"github.com/allora-network/allora-chain/app/upgrades"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
)

const (
	UpgradeName = "v0.16.0"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{Added: []string{schedulertypes.StoreKey}, Renamed: nil, Deleted: nil},
}

func CreateUpgradeHandler(
	moduleManager *module.Manager,
	configurator module.Configurator,
	_ *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info("RUN MIGRATIONS")
		vm, err := moduleManager.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		sdkCtx.Logger().Info("MIGRATIONS COMPLETED")
		return vm, nil
	}
}
