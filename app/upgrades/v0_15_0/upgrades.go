package v0_15_0 //nolint:revive // var-naming: don't use an underscore in package name

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/allora-network/allora-chain/app/keepers"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/app/upgrades"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const (
	UpgradeName = "v0.15.0"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{Added: []string{schedulertypes.StoreKey}, Renamed: nil, Deleted: nil},
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

		amount, ok := math.NewIntFromString("1000000000000000000000") // 1,000 ALLO
		if !ok {
			return nil, fmt.Errorf("could not convert amount to int")
		}
		coins := sdk.NewCoins(sdk.NewCoin(params.BaseCoinUnit, amount))

		sdkCtx.Logger().Info("Minting new coins", "coins", coins)
		if err := keepers.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins); err != nil {
			return nil, err
		}

		sdkCtx.Logger().Info(
			"Transferring newly minted coins to allorarewards module account",
			"coins", coins,
			"module", emissionstypes.AlloraPendingRewardForDelegatorAccountName,
		)
		if err := keepers.BankKeeper.SendCoinsFromModuleToModule(
			ctx,
			minttypes.ModuleName,
			emissionstypes.AlloraRewardsAccountName,
			coins,
		); err != nil {
			return nil, err
		}

		err = keepers.MintKeeper.AddEcosystemTokensMinted(ctx, coins.AmountOf(params.BaseCoinUnit))
		if err != nil {
			return nil, errorsmod.Wrap(err, "could not add ecosystem tokens minted")
		}

		sdkCtx.Logger().Info("MIGRATIONS COMPLETED")
		return vm, nil
	}
}
