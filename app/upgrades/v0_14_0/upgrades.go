package v0_14_0 //nolint:revive // var-naming: don't use an underscore in package name

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/allora-network/allora-chain/app/keepers"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/app/upgrades"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const (
	UpgradeName = "v0.14.0"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{Added: nil, Renamed: nil, Deleted: nil},
}

func CreateUpgradeHandler(
	moduleManager *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info("RUN MIGRATIONS")
		vm, err := moduleManager.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		foundationAddr, err := sdk.AccAddressFromBech32("allo1av8zu7ss8lajw9v0lrs75jn679tvrty2qavsfk")
		if err != nil {
			return nil, err
		}

		amount, ok := math.NewIntFromString("145000000000000000000000000")
		if !ok {
			return nil, fmt.Errorf("could not convert amount to int")
		}
		coins := sdk.NewCoins(sdk.NewCoin(params.BaseCoinUnit, amount))

		sdkCtx.Logger().Info("Minting new coins", "coins", coins)
		if err := keepers.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins); err != nil {
			return nil, err
		}

		sdkCtx.Logger().Info(
			"Transferring newly minted coins to foundation wallet",
			"coins", coins,
			"address", foundationAddr,
		)
		if err := keepers.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, foundationAddr, coins); err != nil {
			return nil, err
		}

		sdkCtx.Logger().Info("MIGRATIONS COMPLETED")
		return vm, nil
	}
}
