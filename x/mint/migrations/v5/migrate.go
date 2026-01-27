package v5

import (
	"github.com/allora-network/allora-chain/x/mint/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// migrate the store from version 4 to version 5
func MigrateStore(ctx sdk.Context, mintKeeper keeper.Keeper) error {
	ctx.Logger().Info("MIGRATING MINT MODULE FROM VERSION 4 TO VERSION 5")
	if err := migrateParams(ctx, mintKeeper); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER migrateParams() FROM VERSION 4 TO VERSION 5")
		return err
	}
	ctx.Logger().Info("MIGRATING MINT MODULE FROM VERSION 4 TO VERSION 5 COMPLETE")
	return nil
}

// We add an additional boolean param that controls
// whether or not emissions is turned on
// For an already running network it should just be turned on
func migrateParams(ctx sdk.Context, mintKeeper keeper.Keeper) error {
	ctx.Logger().Info("MIGRATING MINT MODULE PARAMS FROM VERSION 4 TO VERSION 5")

	params, err := mintKeeper.Params.Get(ctx)
	if err != nil {
		ctx.Logger().Error("failed to get current params from keeper", "error", err)
		return err
	}

	// set the emission enabled param to true by default
	params.EmissionEnabled = true

	err = mintKeeper.Params.Set(ctx, params)
	if err != nil {
		ctx.Logger().Error("failed to set updated params in keeper", "error", err)
		return err
	}

	ctx.Logger().Info("MIGRATING MINT MODULE PARAMS FROM VERSION 4 TO VERSION 5 COMPLETE")
	return nil
}
