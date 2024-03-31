package mint

import (
	"context"

	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	"github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	// Get the balance of the "ecosystem" module account
	ecosystemBalance, err := k.GetEcosystemBalance(ctx, params.MintDenom)
	if err != nil {
		return err
	}
	// Get the expected amount of emissions this block
	networkStaked, err := keeper.GetNumStakedTokens(ctx, k)
	if err != nil {
		return err
	}
	targetRewardEmissionPerUnitStakedToken, err := keeper.TargetRewardEmissionPerUnitStakedToken(
		sdkCtx,
		k,
		params.FEmission,
		params.FEmissionPrec,
		ecosystemBalance,
		networkStaked,
	)
	if err != nil {
		return err
	}
	smoothingDegreeNumerator, smoothingDegreeDenominator, err := keeper.SmoothingFactorPerBlock(
		sdkCtx,
		k,
		params.OneMonthSmoothingDegree,
		params.OneMonthSmoothingDegreePrec,
	)
	if err != nil {
		return err
	}
	previousRewardEmissionsPerUnitStakedToken, err := k.PreviousRewardEmissionsPerUnitStakedToken.Get(ctx)
	if err != nil {
		return err
	}
	e_i := keeper.RewardEmissionPerUnitStakedToken(
		targetRewardEmissionPerUnitStakedToken,
		smoothingDegreeNumerator,
		smoothingDegreeDenominator,
		previousRewardEmissionsPerUnitStakedToken,
	)
	blockEmissions := keeper.TotalEmissionPerTimestep(e_i, networkStaked)

	// if the expected amount of emissions is greater than the balance of the ecosystem module account
	if blockEmissions.GT(ecosystemBalance) {
		// check that you are allowed to mint more tokens and we haven't hit the max supply
		ecosystemTokensAlreadyMinted, err := k.EcosystemTokensMinted.Get(ctx)
		if err != nil {
			return err
		}
		ecosystemMaxSupply := math.LegacyNewDecFromBigInt(params.MaxSupply.BigInt()).
			Mul(math.LegacyNewDecWithPrec(
				keeper.EcosystemTreasuryPercentOfTotalSupply,
				keeper.EcosystemTreasuryPercentOfTotalSupplyPrecision)).
			TruncateInt()
		if ecosystemTokensAlreadyMinted.Add(blockEmissions).GT(ecosystemMaxSupply) {
			return types.ErrMaxSupplyReached
		}
		// mint the amount of tokens required to pay out the emissions
		tokensToMint := blockEmissions.Sub(ecosystemBalance)
		coins := sdk.NewCoins(sdk.NewCoin(params.MintDenom, tokensToMint))
		err = k.MintCoins(sdkCtx, coins)
		if err != nil {
			return err
		}
		err = k.MoveCoinsFromMintToEcosystem(sdkCtx, coins)
		if err != nil {
			return err
		}
		// then increment the recorded history of the amount of tokens minted
		err = k.EcosystemTokensMinted.Set(ctx, ecosystemTokensAlreadyMinted.Add(tokensToMint))
		if err != nil {
			return err
		}
	}
	// pay out the computed block emissions from the ecosystem account
	// if it came from collected fees, great, if it came from minting, also fine
	coins := sdk.NewCoins(sdk.NewCoin(params.MintDenom, blockEmissions))
	err = k.PayEmissionsFromEcosystemAccount(sdkCtx, coins)
	if err != nil {
		return err
	}
	// set the previous emissions to this block's emissions
	// todo use int without truncation, control for precision in math
	k.PreviousRewardEmissionsPerUnitStakedToken.Set(ctx, e_i)
	return nil
}
