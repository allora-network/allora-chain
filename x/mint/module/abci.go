package mint

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// the begin blocker function for the mint module
// it calculates the emission / inflation rate for the block,
// and sends inflationary rewards to the validators and reputers module accounts
func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	moduleParams, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}
	// Get the balance of the "ecosystem" module account
	ecosystemBalance, err := k.GetEcosystemBalance(ctx, moduleParams.MintDenom)
	if err != nil {
		return err
	}

	blockHeight := uint64(sdkCtx.BlockHeight())

	blockEmission, err := k.PreviousBlockEmission.Get(ctx)
	if err != nil {
		return err
	}
	ecosystemMintSupplyRemaining, err := k.GetEcosystemMintSupplyRemaining(sdkCtx, moduleParams)
	if err != nil {
		return err
	}
	blocksPerMonth, err := k.GetParamsBlocksPerMonth(ctx)
	if err != nil {
		return err
	}
	vPercentADec, err := k.GetValidatorsVsAlloraPercentReward(ctx)
	if err != nil {
		return err
	}
	vPercent, err := vPercentADec.SdkLegacyDec()
	if err != nil {
		return err
	}
	// every month on the first block of the month, update the emissions rate
	if blockHeight%blocksPerMonth == 1 { // easier to test when genesis starts at 1
		// Recalculate the target emission for the block
		// WARNING: After Calling RecalculateTargetEmission,
		// PreviousRewardEmissionPerUnitStakedToken and PreviousBlockEmission
		// are set to new values. If later in begin blocker you need to use these
		// you should get them first before this function is called!!
		blockEmission, _, err = keeper.RecalculateTargetEmission(
			sdkCtx,
			k,
			blockHeight,
			blocksPerMonth,
			moduleParams,
			ecosystemBalance,
			ecosystemMintSupplyRemaining,
			vPercent,
		)
		if err != nil {
			return errors.Wrap(err, "error recalculating target emission")
		}
	}
	// if the expected amount of emissions is greater than the balance of the ecosystem module account
	if blockEmission.GT(ecosystemBalance) {
		// mint the amount of tokens required to pay out the emissions
		tokensToMint := blockEmission.Sub(ecosystemBalance)
		if tokensToMint.GT(ecosystemMintSupplyRemaining) {
			tokensToMint = ecosystemMintSupplyRemaining
		}
		coins := sdk.NewCoins(sdk.NewCoin(moduleParams.MintDenom, tokensToMint))
		err = k.MintCoins(sdkCtx, coins)
		if err != nil {
			return err
		}
		err = k.MoveCoinsFromMintToEcosystem(sdkCtx, coins)
		if err != nil {
			return err
		}
		// then increment the recorded history of the amount of tokens minted
		err = k.AddEcosystemTokensMinted(ctx, tokensToMint)
		if err != nil {
			return err
		}
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
		return err
	}
	err = k.PayAlloraRewardsFromEcosystem(sdkCtx, coinsAlloraRewards)
	if err != nil {
		return err
	}
	return nil
}
