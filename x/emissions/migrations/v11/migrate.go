package v11

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
)

// MigrateStore migrates the store from consensus version 10 to version 11
// It initializes the new monthly rewards values to zero.
func MigrateStore(ctx sdk.Context, wk *keeper.WeightsKeeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 10 TO VERSION 11")
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 10 TO VERSION 11")

	ctx.Logger().Info("Initializing monthly rewards values")

	// Will set to zero both monthlyReputerRewards and monthlyTopicRewards
	err := wk.ResetMonthlyRewards(ctx)
	if err != nil {
		ctx.Logger().Error("Error setting monthly reputer rewards during migration", "error", err)
		return err
	}

	ctx.Logger().Info("MIGRATING EMISSIONS MODULE FROM VERSION 10 TO VERSION 11 COMPLETE")
	return nil
}
