package mint

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	"github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func UpdateEmissionRate(
	ctx sdk.Context,
	k keeper.Keeper,
	params types.Params,
	ecosystemBalance math.Int,
) (blockEmission math.Int, e_i_n math.Int, e_i_d math.Int, err error) {
	// Get the expected amount of emissions this block
	networkStaked, err := keeper.GetNumStakedTokens(ctx, k)
	fmt.Println("Network staked", networkStaked)
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, err
	}
	totalSupply := k.GetSupply(ctx).Amount
	lockedSupply := keeper.GetLockedTokenSupply()
	circulatingSupply := totalSupply.Sub(lockedSupply)
	if circulatingSupply.IsNegative() {
		return math.Int{}, math.Int{}, math.Int{}, errors.Wrapf(
			types.ErrNegativeCirculatingSupply,
			"total supply %s, locked supply %s",
			totalSupply.String(),
			lockedSupply.String(),
		)
	}
	targetRewardEmissionPerUnitStakedTokenNumerator, targetRewardEmissionPerUnitStakedTokenDenominator,
		err := keeper.TargetRewardEmissionPerUnitStakedToken(
		params.FEmissionNumerator,
		params.FEmissionDenominator,
		ecosystemBalance,
		networkStaked,
		circulatingSupply,
		totalSupply,
	)
	fmt.Println("Target reward emission per unit staked token numerator", targetRewardEmissionPerUnitStakedTokenNumerator)
	fmt.Println("Target reward emission per unit staked token denominator", targetRewardEmissionPerUnitStakedTokenDenominator)
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, err
	}
	smoothingDegreeNumerator, smoothingDegreeDenominator := keeper.SmoothingFactorPerTimestep(
		ctx,
		k,
		params.OneMonthSmoothingDegreeNumerator,
		params.OneMonthSmoothingDegreeDenominator,
		params.EmissionCalibrationsTimestepPerMonth,
	)
	fmt.Println("Smoothing degree numerator", smoothingDegreeNumerator)
	previousRewardEmissionPerUnitStakedTokenNumerator, err := k.PreviousRewardEmissionPerUnitStakedTokenNumerator.Get(ctx)
	fmt.Println("Previous reward emissions per unit staked token", previousRewardEmissionPerUnitStakedTokenNumerator)
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, err
	}
	previousRewardEmissionPerUnitStakedTokenDenominator, err := k.PreviousRewardEmissionPerUnitStakedTokenDenominator.Get(ctx)
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, err
	}
	// e_i_n stands for e_i numerator, d denominator
	e_i_n, e_i_d = keeper.RewardEmissionPerUnitStakedToken(
		targetRewardEmissionPerUnitStakedTokenNumerator,
		targetRewardEmissionPerUnitStakedTokenDenominator,
		smoothingDegreeNumerator,
		smoothingDegreeDenominator,
		previousRewardEmissionPerUnitStakedTokenNumerator,
		previousRewardEmissionPerUnitStakedTokenDenominator,
	)
	fmt.Println("E_i numerator", e_i_n)
	fmt.Println("E_i denominator", e_i_d)
	blockEmission = keeper.TotalEmissionPerTimestep(e_i_n, e_i_d, networkStaked)
	fmt.Println("Block emissions", blockEmission)
	return blockEmission, e_i_n, e_i_d, nil
}

func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}
	// Get the balance of the "ecosystem" module account
	ecosystemBalance, err := k.GetEcosystemBalance(ctx, params.MintDenom)
	fmt.Println("Ecosystem balance", ecosystemBalance)
	if err != nil {
		return err
	}

	// find out if we need to update the block emissions rate
	emissionsRateUpdateCadence := params.BlocksPerMonth / params.EmissionCalibrationsTimestepPerMonth
	if emissionsRateUpdateCadence == 0 {
		return errors.Wrapf(
			types.ErrZeroDenominator,
			"emissions rate update cadence is zero: %d | %d",
			params.BlocksPerMonth,
			params.EmissionCalibrationsTimestepPerMonth,
		)
	}
	blockHeight := sdkCtx.BlockHeight()

	updateEmission := false
	blockEmission, err := k.PreviousBlockEmission.Get(ctx)
	if err != nil {
		return err
	}
	var e_i_n, e_i_d math.Int
	// every emissionsRateUpdateCadence blocks, update the emissions rate
	if uint64(blockHeight)%emissionsRateUpdateCadence == 0 {
		blockEmission, e_i_n, e_i_d, err = UpdateEmissionRate(sdkCtx, k, params, ecosystemBalance)
		if err != nil {
			return err
		}
		updateEmission = true
	}
	// if the expected amount of emissions is greater than the balance of the ecosystem module account
	if blockEmission.GT(ecosystemBalance) {
		// check that you are allowed to mint more tokens and we haven't hit the max supply
		ecosystemTokensAlreadyMinted, err := k.EcosystemTokensMinted.Get(ctx)
		if err != nil {
			return err
		}
		ecosystemMaxSupply := params.MaxSupply.
			Mul(math.NewInt(keeper.EcosystemTreasuryPercentOfTotalSupplyNumerator)).
			Quo(math.NewInt(keeper.EcosystemTreasuryPercentOfTotalSupplyDenominator))
		if ecosystemTokensAlreadyMinted.Add(blockEmission).GT(ecosystemMaxSupply) {
			return types.ErrMaxSupplyReached
		}
		// mint the amount of tokens required to pay out the emissions
		tokensToMint := blockEmission.Sub(ecosystemBalance)
		coins := sdk.NewCoins(sdk.NewCoin(params.MintDenom, tokensToMint))
		fmt.Println("Minting tokensToMint", tokensToMint)
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
	// we pay both reputers and cosmos validators, so each payment should be
	// half as big (divide by two). Integer division truncates, and that's fine.
	coins := sdk.NewCoins(sdk.NewCoin(params.MintDenom, blockEmission.Quo(math.NewInt(2))))
	fmt.Println("Paying coins", coins)
	err = k.PayCosmosValidatorRewardFromEcosystemAccount(sdkCtx, coins)
	if err != nil {
		return err
	}
	err = k.PayReputerRewardFromEcosystemAccount(sdkCtx, coins)
	if err != nil {
		return err
	}
	if updateEmission {
		// set the previous emissions to this block's emissions
		k.PreviousRewardEmissionPerUnitStakedTokenNumerator.Set(ctx, e_i_n)
		k.PreviousRewardEmissionPerUnitStakedTokenDenominator.Set(ctx, e_i_d)
		k.PreviousBlockEmission.Set(ctx, blockEmission)
	}
	return nil
}
