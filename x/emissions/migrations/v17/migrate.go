package v17

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
)

// MigrateStore migrates the emissions module from version 16 to version 17.
//
// Cutover onto scheduler-driven epochs:
//   - reconstruct in-flight epochs from unfulfilled worker/reputer nonces
//   - convert topic timing fields and MinEpochLength from blocks to seconds
//     using AssumedBlockTimeSeconds (~6s)
//   - enroll active topics onto periodic StartNewEpoch
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 16 TO VERSION 17")
	if err := emissionsKeeper.MigrateToSchedulerEpochs(ctx); err != nil {
		ctx.Logger().Error("ERROR INVOKING MigrateToSchedulerEpochs() FROM VERSION 16 TO VERSION 17", "error", err)
		return err
	}
	ctx.Logger().Info("MIGRATION EMISSIONS MODULE FROM VERSION 16 TO VERSION 17 COMPLETE")
	return nil
}
