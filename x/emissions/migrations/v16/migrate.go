package v16

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
)

// MigrateStore migrates the emissions module from version 15 to version 16.
//
// Cutover onto scheduler-driven epochs:
//   - reconstruct in-flight epochs from unfulfilled worker/reputer nonces
//   - convert topic timing fields and MinEpochLength from blocks to seconds
//     using AssumedBlockTimeSeconds (~6s)
//   - enroll active topics onto periodic StartNewEpoch
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 15 TO VERSION 16")
	if err := emissionsKeeper.MigrateToSchedulerEpochs(ctx); err != nil {
		ctx.Logger().Error("ERROR INVOKING MigrateToSchedulerEpochs() FROM VERSION 15 TO VERSION 16", "error", err)
		return err
	}
	ctx.Logger().Info("MIGRATION EMISSIONS MODULE FROM VERSION 15 TO VERSION 16 COMPLETE")
	return nil
}
