package mint

import (
	"context"

	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	"github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GetEmissionPerMonth(
	ctx sdk.Context,
	k types.MintKeeper,
	blockHeight uint64,
	blocksPerMonth uint64,
	params types.Params,
	ecosystemBalance math.Int,
	ecosystemMintSupplyRemaining math.Int,
	validatorsPercent math.LegacyDec,
) (
	emissionPerMonth math.Int,
	emissionPerUnitStakedToken math.LegacyDec,
	err error,
) {
	// Get the expected amount of emissions this block
	networkStaked, err := keeper.GetNumStakedTokens(ctx, k)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, err
	}
	totalSupply := params.MaxSupply
	lockedVestingTokens, _, _, _ := keeper.GetLockedVestingTokens(
		blocksPerMonth,
		math.NewIntFromUint64(blockHeight),
		params,
	)
	ecosystemLocked := ecosystemBalance.Add(ecosystemMintSupplyRemaining)
	circulatingSupply := totalSupply.Sub(lockedVestingTokens).Sub(ecosystemLocked)
	if circulatingSupply.IsNegative() {
		ctx.Logger().Error(
			"Negative circulating supply",
			"totalSupply", totalSupply.String(),
			"lockedVestingTokens", lockedVestingTokens.String(),
			"ecosystemLocked", ecosystemLocked.String(),
			"circulatingSupply", circulatingSupply.String(),
		)

		return math.Int{}, math.LegacyDec{}, types.ErrNegativeCirculatingSupply
	}
	// T_{total,i} = ecosystemLocked
	// N_{staked,i} = networkStaked
	// N_{circ,i} = circulatingSupply
	// N_{total,i} = totalSupply
	ctx.Logger().Info(
		"Emission Per Unit Staked Token Calculation\n"+
			"FEmission %s\n"+
			"ecosystemLocked %s\n"+
			"networkStaked %s\n"+
			"circulatingSupply %s\n"+
			"totalSupply %s\n"+
			"lockedVestingTokens %s\n",
		"ecosystemBalance%s\n",
		"ecosystemMintSupplyRemaining %s\n"+
			params.FEmission.String(),
		ecosystemLocked.String(),
		networkStaked.String(),
		circulatingSupply.String(),
		totalSupply.String(),
		lockedVestingTokens.String(),
		ecosystemBalance.String(),
		ecosystemMintSupplyRemaining.String(),
	)
	targetRewardEmissionPerUnitStakedToken,
		err := keeper.GetTargetRewardEmissionPerUnitStakedToken(
		params.FEmission,
		ecosystemLocked,
		networkStaked,
		circulatingSupply,
		params.MaxSupply,
	)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, err
	}
	reputersPercent, err := k.GetPreviousPercentageRewardToStakedReputers(ctx)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, err
	}
	maximumMonthlyEmissionPerUnitStakedToken := keeper.GetMaximumMonthlyEmissionPerUnitStakedToken(
		params.MaximumMonthlyPercentageYield,
		reputersPercent,
		validatorsPercent,
	)
	targetRewardEmissionPerUnitStakedToken = keeper.GetCappedTargetEmissionPerUnitStakedToken(
		targetRewardEmissionPerUnitStakedToken,
		maximumMonthlyEmissionPerUnitStakedToken,
	)
	var previousRewardEmissionPerUnitStakedToken math.LegacyDec
	// if this is the first month/time we're calculating the target emission...
	if blockHeight < blocksPerMonth {
		previousRewardEmissionPerUnitStakedToken = targetRewardEmissionPerUnitStakedToken
	} else {
		previousRewardEmissionPerUnitStakedToken, err = k.GetPreviousRewardEmissionPerUnitStakedToken(ctx)
		if err != nil {
			return math.Int{}, math.LegacyDec{}, err
		}
	}
	emissionPerUnitStakedToken = keeper.GetExponentialMovingAverage(
		targetRewardEmissionPerUnitStakedToken,
		params.OneMonthSmoothingDegree,
		previousRewardEmissionPerUnitStakedToken,
	)
	emissionPerMonth = keeper.GetTotalEmissionPerMonth(emissionPerUnitStakedToken, networkStaked)
	return emissionPerMonth, emissionPerUnitStakedToken, nil
}

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

	blockHeight := uint64(sdkCtx.BlockHeight())

	blockEmission, err := k.PreviousBlockEmission.Get(ctx)
	if err != nil {
		return err
	}
	ecosystemMintSupplyRemaining, err := k.GetEcosystemMintSupplyRemaining(sdkCtx, params)
	if err != nil {
		return err
	}
	updateEmission := false
	var e_i math.LegacyDec
	blocksPerMonth, err := k.GetParamsBlocksPerMonth(ctx)
	if err != nil {
		return err
	}
	vPercentADec, err := k.GetValidatorsVsAlloraPercentReward(ctx)
	if err != nil {
		return err
	}
	vPercent := vPercentADec.SdkLegacyDec()
	// every month on the first block of the month, update the emissions rate
	if blockHeight%blocksPerMonth == 1 { // easier to test when genesis starts at 1
		emissionPerMonth, emissionPerUnitStakedToken, err := GetEmissionPerMonth(
			sdkCtx,
			k,
			blockHeight,
			blocksPerMonth,
			params,
			ecosystemBalance,
			ecosystemMintSupplyRemaining,
			vPercent,
		)
		if err != nil {
			return err
		}
		// emission/block = (emission/month) / (block/month)
		blockEmission = emissionPerMonth.
			Quo(math.NewIntFromUint64(blocksPerMonth))
		e_i = emissionPerUnitStakedToken
		updateEmission = true
		k.Logger(ctx).Info("Emissions Update",
			"emissionPerUnitStakedToken", e_i.String(),
			"emissionPerMonth", emissionPerMonth.String(),
			"blockEmission", blockEmission.String(),
		)

	}
	// if the expected amount of emissions is greater than the balance of the ecosystem module account
	if blockEmission.GT(ecosystemBalance) {
		// mint the amount of tokens required to pay out the emissions
		tokensToMint := blockEmission.Sub(ecosystemBalance)
		if tokensToMint.GT(ecosystemMintSupplyRemaining) {
			tokensToMint = ecosystemMintSupplyRemaining
		}
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
