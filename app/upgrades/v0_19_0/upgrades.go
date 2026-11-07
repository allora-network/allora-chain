package v0_19_0 //nolint:revive // var-naming: don't use an underscore in package name

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/allora-network/allora-chain/app/keepers"
	"github.com/allora-network/allora-chain/app/upgrades"
	mintkeeper "github.com/allora-network/allora-chain/x/mint/keeper"
)

const (
	UpgradeName = "v0.19.0"

	// InvestorsTeamVestingDelayMonths is how far the investors and team
	// vesting schedules are pushed back on the live chain.
	InvestorsTeamVestingDelayMonths uint64 = 6
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        storetypes.StoreUpgrades{Added: nil, Renamed: nil, Deleted: nil},
}

func CreateUpgradeHandler(
	moduleManager *module.Manager,
	configurator module.Configurator,
	appKeepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info("RUN MIGRATIONS")
		vm, err := moduleManager.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}
		sdkCtx.Logger().Info("MIGRATIONS COMPLETED")

		if err := SetInvestorsTeamVestingDelay(ctx, appKeepers.MintKeeper, InvestorsTeamVestingDelayMonths); err != nil {
			return vm, err
		}
		sdkCtx.Logger().Info("INVESTORS AND TEAM VESTING DELAY SET", "months", InvestorsTeamVestingDelayMonths)
		return vm, nil
	}
}

// SetInvestorsTeamVestingDelay writes the investors and team vesting delay
// into the mint params, leaving every other param unchanged.
func SetInvestorsTeamVestingDelay(ctx context.Context, mintKeeper mintkeeper.Keeper, months uint64) error {
	params, err := mintKeeper.Params.Get(ctx)
	if err != nil {
		return err
	}
	params.InvestorsTeamVestingDelayMonths = months
	if err := params.Validate(); err != nil {
		return err
	}
	return mintKeeper.Params.Set(ctx, params)
}
