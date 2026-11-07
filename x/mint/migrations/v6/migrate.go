package v6

import (
	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/mint/keeper"
	oldtypes "github.com/allora-network/allora-chain/x/mint/migrations/v6/oldtypes"
	"github.com/allora-network/allora-chain/x/mint/types"
)

// MigrateStore migrates the mint store from version 6 to version 7.
func MigrateStore(ctx sdk.Context, mintKeeper keeper.Keeper) error {
	ctx.Logger().Info("MIGRATING MINT MODULE FROM VERSION 6 TO VERSION 7")
	if err := migrateParams(ctx, mintKeeper); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER migrateParams() FROM VERSION 6 TO VERSION 7")
		return err
	}
	ctx.Logger().Info("MIGRATING MINT MODULE FROM VERSION 6 TO VERSION 7 COMPLETE")
	return nil
}

// migrateParams adds the investors and team vesting delay param. Existing
// networks start with no delay; governance or an upgrade handler sets it.
func migrateParams(ctx sdk.Context, mintKeeper keeper.Keeper) error {
	store := runtime.KVStoreAdapter(mintKeeper.GetStorageService().OpenKVStore(ctx))
	cdc := mintKeeper.GetBinaryCodec()

	oldBytes := store.Get(types.ParamsKey)
	if oldBytes == nil {
		return errors.Wrap(types.ErrNotFound, "mint params not found in store")
	}
	var oldParams oldtypes.Params
	if err := cdc.Unmarshal(oldBytes, &oldParams); err != nil {
		return errors.Wrap(err, "failed to unmarshal old mint params")
	}

	newParams := types.Params{
		MintDenom:                              oldParams.MintDenom,
		MaxSupply:                              oldParams.MaxSupply,
		FEmission:                              oldParams.FEmission,
		OneMonthSmoothingDegree:                oldParams.OneMonthSmoothingDegree,
		EcosystemTreasuryPercentOfTotalSupply:  oldParams.EcosystemTreasuryPercentOfTotalSupply,
		FoundationTreasuryPercentOfTotalSupply: oldParams.FoundationTreasuryPercentOfTotalSupply,
		ParticipantsPercentOfTotalSupply:       oldParams.ParticipantsPercentOfTotalSupply,
		InvestorsPercentOfTotalSupply:          oldParams.InvestorsPercentOfTotalSupply,
		TeamPercentOfTotalSupply:               oldParams.TeamPercentOfTotalSupply,
		MaximumMonthlyPercentageYield:          oldParams.MaximumMonthlyPercentageYield,
		InvestorsPreseedPercentOfTotalSupply:   oldParams.InvestorsPreseedPercentOfTotalSupply,
		EmissionEnabled:                        oldParams.EmissionEnabled,
		InvestorsTeamVestingDelayMonths:        0,
	}
	if err := mintKeeper.Params.Set(ctx, newParams); err != nil {
		return errors.Wrap(err, "failed to set migrated mint params")
	}
	return nil
}
