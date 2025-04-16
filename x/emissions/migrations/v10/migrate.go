package v10

import (
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MigrateStore migrates the store from consensus version 9 to version 10
// It does nothing as there are no changes to the store
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 9 TO VERSION 10")
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 9 TO VERSION 10")
	ctx.Logger().Info("MIGRATING EMISSIONS MODULE FROM VERSION 9 TO VERSION 10 COMPLETE")
	return nil
}
