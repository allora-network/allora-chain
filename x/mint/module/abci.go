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
) (
	emissionPerTimestep math.Int,
	emissionPerUnitStakedToken math.LegacyDec,
	err error,
) {
	fmt.Println("Updating emission rate")
	// Get the expected amount of emissions this block
	networkStaked, err := keeper.GetNumStakedTokens(ctx, k)
	fmt.Println("Network staked", networkStaked)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, err
	}
	totalSupply := k.GetSupply(ctx).Amount
	lockedSupply := keeper.GetLockedTokenSupply()
	circulatingSupply := totalSupply.Sub(lockedSupply)
	if circulatingSupply.IsNegative() {
		return math.Int{}, math.LegacyDec{}, errors.Wrapf(
			types.ErrNegativeCirculatingSupply,
			"total supply %s, locked supply %s",
			totalSupply.String(),
			lockedSupply.String(),
		)
	}
	fmt.Println("Total supply", totalSupply)
	fmt.Println("Locked supply", lockedSupply)
	fmt.Println("Circulating supply", circulatingSupply)
	fmt.Println("FEmissionNumerator", params.FEmissionNumerator)
	fmt.Println("FEmissionDenominator", params.FEmissionDenominator)
	targetRewardEmissionPerUnitStakedToken,
		err := keeper.GetTargetRewardEmissionPerUnitStakedToken(
		params.FEmissionNumerator,
		params.FEmissionDenominator,
		ecosystemBalance,
		networkStaked,
		circulatingSupply,
		totalSupply,
	)
	fmt.Println("Target reward emission per unit staked token", targetRewardEmissionPerUnitStakedToken)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, err
	}
	smoothingDegree := keeper.GetSmoothingFactorPerTimestep(
		ctx,
		k,
		params.OneMonthSmoothingDegreeNumerator,
		params.OneMonthSmoothingDegreeDenominator,
		params.EmissionCalibrationsTimestepPerMonth,
	)
	fmt.Println("Smoothing degree", smoothingDegree)
	previousRewardEmissionPerUnitStakedToken, err := k.PreviousRewardEmissionPerUnitStakedToken.Get(ctx)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, err
	}
	fmt.Println("Previous reward emissions per unit staked token", previousRewardEmissionPerUnitStakedToken)
	emissionPerUnitStakedToken = keeper.GetRewardEmissionPerUnitStakedToken(
		targetRewardEmissionPerUnitStakedToken,
		smoothingDegree,
		previousRewardEmissionPerUnitStakedToken,
	)
	fmt.Println("E_i", emissionPerUnitStakedToken)
	emissionPerTimestep = keeper.GetTotalEmissionPerTimestep(emissionPerUnitStakedToken, networkStaked)
	return emissionPerTimestep, emissionPerUnitStakedToken, nil
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
	emissionRateUpdateCadence := params.BlocksPerMonth / params.EmissionCalibrationsTimestepPerMonth
	if emissionRateUpdateCadence == 0 {
		return errors.Wrapf(
			types.ErrZeroDenominator,
			"emissions rate update cadence is zero: %d | %d",
			params.BlocksPerMonth,
			params.EmissionCalibrationsTimestepPerMonth,
		)
	}
	blockHeight := sdkCtx.BlockHeight()

	blockEmission, err := k.PreviousBlockEmission.Get(ctx)
	if err != nil {
		return err
	}
	updateEmission := false
	var e_i math.LegacyDec
	// every emissionsRateUpdateCadence blocks, update the emissions rate
	fmt.Printf("Block Height %d | emissionRateUpdateCadence %d\n", blockHeight, emissionRateUpdateCadence)
	if uint64(blockHeight)%emissionRateUpdateCadence == 1 { // easier to test when genesis starts at 1
		emissionPerTimestep, emissionPerUnitStakedToken, err := UpdateEmissionRate(sdkCtx, k, params, ecosystemBalance)
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
		// check that you are allowed to mint more tokens and we haven't hit the max supply
		ecosystemTokensAlreadyMinted, err := k.EcosystemTokensMinted.Get(ctx)
		if err != nil {
			return err
		}
		ecosystemMaxSupply := params.MaxSupply.
			Mul(params.EcosystemTreasuryPercentOfTotalSupplyNumerator).
			Quo(params.EcosystemTreasuryPercentOfTotalSupplyDenominator)
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
		k.PreviousRewardEmissionPerUnitStakedToken.Set(ctx, e_i)
		k.PreviousBlockEmission.Set(ctx, blockEmission)
	}
	return nil
}
