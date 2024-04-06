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
	ecosystemMintSupplyRemaining math.Int,
) (
	emissionPerTimestep math.Int,
	emissionPerUnitStakedToken math.LegacyDec,
	err error,
) {
	fmt.Println("Updating emission rate")
	// Get the expected amount of emissions this block
	networkStaked, err := keeper.GetNumStakedTokens(ctx, k)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, err
	}
	totalSupply := k.GetTotalCurrTokenSupply(ctx).Amount
	lockedSupply := keeper.GetLockedTokenSupply(
		math.NewIntFromUint64(uint64(ctx.BlockHeight())),
		params,
	)
	circulatingSupply := totalSupply.Sub(lockedSupply)
	if circulatingSupply.IsNegative() {
		return math.Int{}, math.LegacyDec{}, errors.Wrapf(
			types.ErrNegativeCirculatingSupply,
			"total supply %s, locked supply %s",
			totalSupply.String(),
			lockedSupply.String(),
		)
	}
	fmt.Println("Supply minted", totalSupply)
	fmt.Println("Locked supply", lockedSupply)
	fmt.Println("Circulating supply", circulatingSupply)
	fmt.Println("FEmission", params.FEmission)
	fmt.Println("Max supply", params.MaxSupply)
	fmt.Println("Ecosystem mint supply remaining", ecosystemMintSupplyRemaining)
	fmt.Println("Network staked", networkStaked)
	targetRewardEmissionPerUnitStakedToken,
		err := keeper.GetTargetRewardEmissionPerUnitStakedToken(
		params.FEmission,
		ecosystemMintSupplyRemaining,
		networkStaked,
		circulatingSupply,
		params.MaxSupply,
	)
	fmt.Println("Target reward emission per unit staked token", targetRewardEmissionPerUnitStakedToken)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, err
	}
	smoothingDegree := keeper.GetSmoothingFactorPerTimestep(
		ctx,
		k,
		params.OneMonthSmoothingDegree,
		params.EmissionCalibrationsTimestepPerMonth,
	)
	fmt.Println("Smoothing degree", smoothingDegree)
	previousRewardEmissionPerUnitStakedToken, err := k.PreviousRewardEmissionPerUnitStakedToken.Get(ctx)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, err
	}
	fmt.Println("Previous reward emissions per unit staked token", previousRewardEmissionPerUnitStakedToken)
	emissionPerUnitStakedToken = keeper.GetExponentialMovingAverage(
		targetRewardEmissionPerUnitStakedToken,
		smoothingDegree,
		previousRewardEmissionPerUnitStakedToken,
	)
	fmt.Println("E_i", emissionPerUnitStakedToken)
	emissionPerTimestep = keeper.GetTotalEmissionPerTimestep(emissionPerUnitStakedToken, networkStaked)
	return emissionPerTimestep, emissionPerUnitStakedToken, nil
}

// How many tokens are left that the ecosystem bucket is allowed to mint?
func GetEcosystemMintSupplyRemaining(
	ctx sdk.Context,
	k keeper.Keeper,
	params types.Params,
) (math.Int, error) {
	// calculate how many tokens left the ecosystem account is allowed to mint
	ecosystemTokensAlreadyMinted, err := k.EcosystemTokensMinted.Get(ctx)
	if err != nil {
		return math.Int{}, err
	}
	// check that you are allowed to mint more tokens and we haven't hit the max supply
	ecosystemMaxSupply := math.LegacyNewDecFromInt(params.MaxSupply).
		Mul(params.EcosystemTreasuryPercentOfTotalSupply).TruncateInt()
	return ecosystemMaxSupply.Sub(ecosystemTokensAlreadyMinted), nil
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
	// EmissionsCalibrationsTimesepPerMonth can never be zero
	// validateEmissionCalibrationTimestepPerMonth prevents zero
	emissionRateUpdateCadence := params.BlocksPerMonth / params.EmissionCalibrationsTimestepPerMonth

	blockHeight := sdkCtx.BlockHeight()

	blockEmission, err := k.PreviousBlockEmission.Get(ctx)
	if err != nil {
		return err
	}
	ecosystemMintSupplyRemaining, err := GetEcosystemMintSupplyRemaining(sdkCtx, k, params)
	if err != nil {
		return err
	}
	fmt.Println("Ecosystem mint supply remaining", ecosystemMintSupplyRemaining)
	updateEmission := false
	var e_i math.LegacyDec
	// every emissionsRateUpdateCadence blocks, update the emissions rate
	fmt.Printf("Block Height %d | emissionRateUpdateCadence %d\n", blockHeight, emissionRateUpdateCadence)
	if uint64(blockHeight)%emissionRateUpdateCadence == 1 { // easier to test when genesis starts at 1
		emissionPerTimestep, emissionPerUnitStakedToken, err := UpdateEmissionRate(
			sdkCtx,
			k,
			params,
			ecosystemMintSupplyRemaining,
		)
		if err != nil {
			return err
		}
		// emission/block = (emission/timestep) * (timestep/month) / (block/month)
		blockEmission = emissionPerTimestep.
			Mul(math.NewIntFromUint64(params.EmissionCalibrationsTimestepPerMonth)).
			Quo(math.NewIntFromUint64(params.BlocksPerMonth))
		e_i = emissionPerUnitStakedToken
		updateEmission = true
	}
	fmt.Println("Block emissions", blockEmission)
	// if the expected amount of emissions is greater than the balance of the ecosystem module account
	if blockEmission.GT(ecosystemBalance) {
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
		err = k.AddEcosystemTokensMinted(ctx, tokensToMint)
		if err != nil {
			return err
		}
	}
	// pay out the computed block emissions from the ecosystem account
	// if it came from collected fees, great, if it came from minting, also fine
	// we pay both reputers and cosmos validators, so each payment should be
	// half as big (divide by two). Integer division truncates, and that's fine.
	validatorCut := params.ValidatorsVsAlloraPercentReward.Mul(blockEmission.ToLegacyDec()).TruncateInt()
	coinsValidator := sdk.NewCoins(sdk.NewCoin(params.MintDenom, validatorCut))
	alloraRewardsCut := blockEmission.Sub(validatorCut)
	coinsAlloraRewards := sdk.NewCoins(sdk.NewCoin(params.MintDenom, alloraRewardsCut))
	err = k.PayValidatorsFromEcosystem(sdkCtx, coinsValidator)
	if err != nil {
		return err
	}
	err = k.PayAlloraRewardsFromEcosystem(sdkCtx, coinsAlloraRewards)
	if err != nil {
		return err
	}
	if updateEmission {
		// set the previous emissions to this block's emissions
		k.PreviousRewardEmissionPerUnitStakedToken.Set(ctx, e_i)
		k.PreviousBlockEmission.Set(ctx, blockEmission)
	}
	return nil
}
