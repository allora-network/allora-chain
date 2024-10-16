package app

import (
	storetypes "cosmossdk.io/store/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	feemarket "github.com/skip-mev/feemarket/x/feemarket"
	feemarketkeeper "github.com/skip-mev/feemarket/x/feemarket/keeper"
	feemarkettypes "github.com/skip-mev/feemarket/x/feemarket/types"
)

// registerFeeMarketModule registers the feemarket module.
func (app *AlloraApp) registerFeeMarketModule() {
	if err := app.RegisterStores(
		storetypes.NewKVStoreKey(feemarkettypes.StoreKey),
	); err != nil {
		panic(err)
	}

	app.FeeMarketKeeper = feemarketkeeper.NewKeeper(
		app.appCodec,
		app.GetKey(feemarkettypes.StoreKey),
		app.AccountKeeper,
		&feemarkettypes.TestDenomResolver{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	if err := app.RegisterModules(
		feemarket.NewAppModule(app.appCodec, *app.FeeMarketKeeper),
	); err != nil {
		panic(err)
	}

	app.FeeMarketKeeper.SetDenomResolver(&feemarkettypes.TestDenomResolver{})
}
