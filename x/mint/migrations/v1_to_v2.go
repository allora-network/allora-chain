package migrations

import (
	"cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	"github.com/allora-network/allora-chain/x/mint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
)

// V1ToV2 migrates the x/mint state from version 1 to version 2.
func V1ToV2(ctx sdk.Context, mintKeeper keeper.Keeper) error {
	ctx.Logger().Info("Migrating x/mint state from version 1 to version 2")

	storageService := mintKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := mintKeeper.GetBinaryCodec()

	if err := migrateParameters(store, cdc); err != nil {
		return err
	}
	return nil
}

// migrates the parameters from v1 to v2. The only change is the addition
// of the InvestorsPreseedPercentOfTotalSupply parameter.
// the InvestorsPercentOfTotalSupply now refers to the Investors SEED Percent of Total Supply
// as oppposed to preseed.
func migrateParameters(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	// Read the old parameters
	oldParams := types.Params{}
	oldParamsBytes := store.Get(types.ParamsKey)
	if oldParamsBytes == nil {
		return errors.Wrapf(types.ErrNotFound, "old parameters not found")
	}
	err := proto.Unmarshal(oldParamsBytes, &oldParams)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal old parameters")
	}

	defaultParams := types.DefaultParams()

	// Convert the old parameters to the new parameters
	newParams := types.Params{
		// old parameters
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
		// new parameters
		InvestorsPreseedPercentOfTotalSupply: defaultParams.InvestorsPercentOfTotalSupply,
	}
	store.Delete(types.ParamsKey)
	store.Set(types.ParamsKey, cdc.MustMarshal(&newParams))

	return nil
}
