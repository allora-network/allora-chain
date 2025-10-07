package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RecalculateEmission recomputes the target emission and updates the
// PreviousBlockEmission value stored in state. A zero value is returned when
// emissions are disabled or the recalculation results in no emission for the
// current block.
func (k Keeper) RecalculateEmission(ctx context.Context) (math.Int, error) {
	moduleParams, err := k.Params.Get(ctx)
	if err != nil {
		return math.Int{}, errorsmod.Wrap(err, "failed to get module params")
	}

	if !moduleParams.EmissionEnabled {
		return math.ZeroInt(), nil
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	ecosystemBalance, err := k.GetEcosystemBalance(ctx, moduleParams.MintDenom)
	if err != nil {
		return math.Int{}, errorsmod.Wrap(err, "failed to get ecosystem balance")
	}

	ecosystemMintSupplyRemaining, err := k.GetEcosystemMintSupplyRemaining(ctx, moduleParams)
	if err != nil {
		return math.Int{}, errorsmod.Wrap(err, "failed to get ecosystem mint supply remaining")
	}

	blocksPerMonth, err := k.GetParamsBlocksPerMonth(ctx)
	if err != nil {
		return math.Int{}, errorsmod.Wrap(err, "failed to get blocks per month")
	}

	vPercentADec, err := k.GetValidatorsVsAlloraPercentReward(ctx)
	if err != nil {
		return math.Int{}, errorsmod.Wrap(err, "failed to get validators vs allora percent reward")
	}

	vPercent, err := vPercentADec.SdkLegacyDec()
	if err != nil {
		return math.Int{}, errorsmod.Wrap(err, "failed to convert validators vs allora percent reward")
	}

	blockEmission, _, err := RecalculateTargetEmission(
		sdkCtx,
		k,
		uint64(sdkCtx.BlockHeight()),
		blocksPerMonth,
		moduleParams,
		ecosystemBalance,
		ecosystemMintSupplyRemaining,
		vPercent,
	)
	if err != nil {
		return math.Int{}, errorsmod.Wrap(err, "failed to recalculate target emission")
	}

	return blockEmission, nil
}
