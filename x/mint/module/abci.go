package mint

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	"github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// the begin blocker function for the mint module
// it calculates the emission / inflation rate for the block,
// and sends inflationary rewards to the validators and reputers module accounts
func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	moduleParams, err := k.Params.Get(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "could not get module params")
	}
	// if emissions are not enabled, do nothing
	if !moduleParams.EmissionEnabled {
		return nil
	}
	// Get the balance of the "ecosystem" module account
	ecosystemBalance, err := k.GetEcosystemBalance(ctx, moduleParams.MintDenom)
	if err != nil {
		return errorsmod.Wrap(err, "could not get ecosystem balance")
	}

	blockHeight := uint64(sdkCtx.BlockHeight())
	// Use this to be removed to current block height.
	startingEmissionsBlockHeight, err := k.GetStartingEmissionsBlockHeight(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "could not get starting emissions block height")
	}
	blockHeightTGEAdjusted := blockHeight - (uint64(startingEmissionsBlockHeight))

	blockEmission, err := k.PreviousBlockEmission.Get(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "could not get previous block emission")
	}
	ecosystemMintSupplyRemaining, err := k.GetEcosystemMintSupplyRemaining(sdkCtx, moduleParams)
	if err != nil {
		return errorsmod.Wrap(err, "could not get ecosystem mint supply remaining")
	}
	blocksPerMonth, err := k.GetParamsBlocksPerMonth(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "could not get blocks per month")
	}
	vPercentADec, err := k.GetValidatorsVsAlloraPercentReward(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "could not get validators vs allora percent reward")
	}
	vPercent, err := vPercentADec.SdkLegacyDec()
	if err != nil {
		return errorsmod.Wrap(err, "could not convert validators vs allora percent reward to legacy dec")
	}
	// every month on the first block of the month, update the emissions rate
	if blockHeightTGEAdjusted%blocksPerMonth == 1 { // easier to test when genesis starts at 1
		// Recalculate the target emission for the block
		// WARNING: After Calling RecalculateTargetEmission,
		// PreviousRewardEmissionPerUnitStakedToken and PreviousBlockEmission
		// are set to new values. If later in begin blocker you need to use these
		// you should get them first before this function is called!!
		blockEmission, _, err = keeper.RecalculateTargetEmission(
			sdkCtx,
			k,
			blockHeightTGEAdjusted,
			blocksPerMonth,
			moduleParams,
			ecosystemBalance,
			ecosystemMintSupplyRemaining,
			vPercent,
		)
		if err != nil {
			return errorsmod.Wrap(err, "could not recalculate target emission")
		}
	}
	// if the expected amount of emissions is greater than the balance of the ecosystem module account
	if blockEmission.GT(ecosystemBalance) {
		// mint the amount of tokens required to pay out the emissions
		tokensToMint := blockEmission.Sub(ecosystemBalance)
		if tokensToMint.GT(ecosystemMintSupplyRemaining) {
			sdkCtx.Logger().Warn("Not enough tokens in the ecosystem mint supply to pay rewards",
				"blockEmission", blockEmission,
				"ecosystemBalance", ecosystemBalance,
				"ecosystemMintSupplyRemaining", ecosystemMintSupplyRemaining,
			)
			// Reduce the block emission as we don't have enough tokens
			blockEmission = blockEmission.Sub(tokensToMint).Add(ecosystemMintSupplyRemaining)
			tokensToMint = ecosystemMintSupplyRemaining
		}
		coins := sdk.NewCoins(sdk.NewCoin(moduleParams.MintDenom, tokensToMint))
		err = k.MintCoins(sdkCtx, coins)
		if err != nil {
			return errorsmod.Wrap(err, "could not mint coins")
		}
		err = k.MoveCoinsFromMintToEcosystem(sdkCtx, coins)
		if err != nil {
			return errorsmod.Wrap(err, "could not move coins from mint to ecosystem")
		}
		// then increment the recorded history of the amount of tokens minted
		err = k.AddEcosystemTokensMinted(ctx, tokensToMint)
		if err != nil {
			return errorsmod.Wrap(err, "could not add ecosystem tokens minted")
		}
		types.EmitNewEcosystemTokenMintSetEvent(sdkCtx, blockHeight, tokensToMint)
	}
	// pay out the computed block emissions from the ecosystem account
	// if it came from collected fees, great, if it came from minting, also fine
	// we pay both reputers and cosmos validators, so each payment should be
	// half as big (divide by two). Integer division truncates, and that's fine.
	validatorCut := vPercent.Mul(blockEmission.ToLegacyDec()).TruncateInt()
	coinsValidator := sdk.NewCoins(sdk.NewCoin(moduleParams.MintDenom, validatorCut))
	alloraRewardsCut := blockEmission.Sub(validatorCut)
	coinsAlloraRewards := sdk.NewCoins(sdk.NewCoin(moduleParams.MintDenom, alloraRewardsCut))
	err = k.PayValidatorsFromEcosystem(sdkCtx, coinsValidator)
	if err != nil {
		return errorsmod.Wrap(err, "could not pay validators from ecosystem")
	}
	err = k.PayAlloraRewardsFromEcosystem(sdkCtx, coinsAlloraRewards)
	if err != nil {
		return errorsmod.Wrap(err, "could not pay allora rewards from ecosystem")
	}
	types.EmitNewRewardCurrentBlockEmissionEvent(sdkCtx, blockHeight, alloraRewardsCut)

	// Emit EmissionInfo event with current emission data
	_, eventInfo, err := k.GetEmissionInfo(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "could not get emission info")
	}
	types.EmitNewEmissionInfoEvent(sdkCtx, *eventInfo)

	return nil
}
