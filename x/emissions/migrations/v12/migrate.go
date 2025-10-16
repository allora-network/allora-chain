package v12

import (
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MigrateStore migrates the store from consensus version 11 to version 12
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 11 TO VERSION 12")
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 11 TO VERSION 12")

	// TBD

	ctx.Logger().Info("MIGRATING EMISSIONS MODULE FROM VERSION 11 TO VERSION 12 COMPLETE")
	return nil
}
