package v0_7_0 //nolint:revive // var-naming: don't use an underscore in package name

import (
	"context"
	"fmt"

	cosmosmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/allora-network/allora-chain/app/keepers"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	feemarkettypes "github.com/skip-mev/feemarket/x/feemarket/types"
)

const (
	UpgradeName = "v0.7.0"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: storetypes.StoreUpgrades{
		Added: []string{
			feegrant.StoreKey,
			feemarkettypes.StoreKey,
			evidencetypes.StoreKey,
		},
		Renamed: nil,
		Deleted: nil,
	},
}

func CreateUpgradeHandler(
	moduleManager *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		vm, err := moduleManager.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		sdkCtx.Logger().Info("CONFIGURING FEE MARKET MODULE")
		if err := ConfigureFeeMarketModule(sdkCtx, keepers); err != nil {
			return vm, err
		}
		sdkCtx.Logger().Info("FEE MARKET MODULE CONFIGURED")

		sdkCtx.Logger().Info("ADDING BURNER PERMISSION TO GOV MODULE")
		if err := AddBurnerPermissionToGovModule(sdkCtx, keepers.AccountKeeper); err != nil {
			return vm, err
		}
		sdkCtx.Logger().Info("BURNER PERMISSION ADDED TO GOV MODULE")

		return vm, nil
	}
}

func ConfigureFeeMarketModule(ctx sdk.Context, keepers *keepers.AppKeepers) error {
	ctx.Logger().Info("SETTING PARAMETERS FOR FEE MARKET MODULE")

	feeMarketParams, err := keepers.FeeMarketKeeper.GetParams(ctx)
	if err != nil {
		return err
	}

	feeMarketParams.Enabled = true
	feeMarketParams.FeeDenom = params.BaseCoinUnit
	feeMarketParams.DistributeFees = true
	feeMarketParams.MinBaseGasPrice = cosmosmath.LegacyMustNewDecFromStr("10")
	if err := keepers.FeeMarketKeeper.SetParams(ctx, feeMarketParams); err != nil {
		return err
	}
	ctx.Logger().Info("SETTING STATE FOR FEE MARKET MODULE")
	state, err := keepers.FeeMarketKeeper.GetState(ctx)
	if err != nil {
		return err
	}

	state.BaseGasPrice = cosmosmath.LegacyMustNewDecFromStr("10")

	return keepers.FeeMarketKeeper.SetState(ctx, state)
}

func AddBurnerPermissionToGovModule(ctx sdk.Context, ak authkeeper.AccountKeeper) error {
	govAccount := ak.GetModuleAccount(ctx, govtypes.ModuleName)
	macc, ok := govAccount.(*authtypes.ModuleAccount)
	if !ok {
		ctx.Logger().Error("FAILED TO GET MODULE ACCOUNT")
		return fmt.Errorf("failed to get module account")
	}

	// Check if the permission already exists to avoid duplicates
	for _, perm := range macc.Permissions {
		if perm == authtypes.Burner {
			return nil // Permission already exists, nothing to do
		}
	}

	macc.Permissions = append(macc.Permissions, authtypes.Burner)
	ak.SetModuleAccount(ctx, macc)
	return nil
}
