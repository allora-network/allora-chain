package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/types"
)

// return the uncirculating supply, i.e. tokens on a vesting schedule
// these tokens will be custodied by a centralized actor off chain.
// this function returns the circulating supply based off of what
// the agreements off chain say were supposed to happen for token lockup
func GetLockedVestingTokens(
	blocksPerMonth uint64,
	blockHeight math.Int,
	params types.Params,
) (total, preseedInvestors, investors, team math.Int) {
	// foundation is unlocked from genesis
	// participants are unlocked from genesis
	// investors and team tokens are locked on a 1 year cliff three year vesting schedule
	blocksInAYear := math.NewIntFromUint64(blocksPerMonth * 12)
	blocksInThreeYears := blocksInAYear.Mul(math.NewInt(3))
	maxSupply := params.MaxSupply.ToLegacyDec()
	percentInvestors := params.InvestorsPercentOfTotalSupply
	percentPreseedInvestors := params.InvestorsPreseedPercentOfTotalSupply
	percentTeam := params.TeamPercentOfTotalSupply
	fullInvestors := percentInvestors.Mul(maxSupply).TruncateInt()
	fullPreseedInvestors := percentPreseedInvestors.Mul(maxSupply).TruncateInt()
	fullTeam := percentTeam.Mul(maxSupply).TruncateInt()
	if blockHeight.LT(blocksInAYear) {
		// less than a year, completely locked
		investors = fullInvestors
		preseedInvestors = fullPreseedInvestors
		team = fullTeam
	} else if blockHeight.GTE(blocksInAYear) && blockHeight.LT(blocksInThreeYears) {
		// between 1 and 3 years, investors and team tokens are vesting and partially unlocked
		thirtySix := math.LegacyNewDec(36)
		monthsUnlocked := blockHeight.Quo(math.NewIntFromUint64(blocksPerMonth)).ToLegacyDec()
		monthsLocked := thirtySix.Sub(monthsUnlocked)
		investors = monthsLocked.Quo(thirtySix).Mul(fullInvestors.ToLegacyDec()).TruncateInt()
		preseedInvestors = monthsLocked.Quo(thirtySix).Mul(fullPreseedInvestors.ToLegacyDec()).TruncateInt()
		team = monthsLocked.Quo(thirtySix).Mul(fullTeam.ToLegacyDec()).TruncateInt()
	} else {
		// greater than 3 years, all investor, team tokens are unlocked
		investors = math.ZeroInt()
		preseedInvestors = math.ZeroInt()
		team = math.ZeroInt()
	}
	return preseedInvestors.Add(investors).Add(team), preseedInvestors, investors, team
}

// helper function to get the number of staked tokens on the network
// includes both tokens staked by cosmos validators (cosmos staking)
// and tokens staked by reputers (allora staking)
func GetNumStakedTokens(ctx context.Context, k types.MintKeeper) (math.Int, error) {
	cosmosValidatorsStaked, err := k.CosmosValidatorStakedSupply(ctx)
	if err != nil {
		return math.Int{}, err
	}
	reputersStaked, err := k.GetEmissionsKeeperTotalStake(ctx)
	if err != nil {
		return math.Int{}, err
	}
	return cosmosValidatorsStaked.Add(reputersStaked), nil
}

// return the circulating supply of tokens
func GetCirculatingSupply(
	ctx context.Context,
	k types.MintKeeper,
	params types.Params,
	blockHeight uint64,
	blocksPerMonth uint64,
) (
	circulatingSupply,
	totalSupply,
	lockedVestingTokens,
	ecosystemLocked math.Int,
	err error,
) {
	ecosystemBalance, err := k.GetEcosystemBalance(ctx, params.MintDenom)
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, math.Int{}, err
	}
	totalSupply = params.MaxSupply
	lockedVestingTokens, _, _, _ = GetLockedVestingTokens(
		blocksPerMonth,
		math.NewIntFromUint64(blockHeight),
		params,
	)
	ecosystemMintSupplyRemaining, err := k.GetEcosystemMintSupplyRemaining(ctx, params)
	if err != nil {
		return math.Int{}, math.Int{}, math.Int{}, math.Int{}, err
	}
	ecosystemLocked = ecosystemBalance.Add(ecosystemMintSupplyRemaining)
	circulatingSupply = totalSupply.Sub(lockedVestingTokens).Sub(ecosystemLocked)
	return circulatingSupply, totalSupply, lockedVestingTokens, ecosystemLocked, nil
}

// The total amount of tokens emitted for a full month
// \cal E_i = e_i*N_{staked,i}
// where e_i is the emission per unit staked token
// and N_{staked,i} is the total amount of tokens staked at timestep i
// THIS FUNCTION TRUNCATES THE RESULT DIVISION TO AN INTEGER
func GetTotalEmissionPerMonth(
	rewardEmissionPerUnitStakedToken math.LegacyDec,
	numStakedTokens math.Int,
) math.Int {
	return rewardEmissionPerUnitStakedToken.MulInt(numStakedTokens).TruncateInt()
}

// maximum monthly emission per unit staked token
// given a maximum monthly percentage yield
// ^e_{max,i} = Xi_max / f_{stakers}
// where Xi_{max} is the maximum MPY,
// and f_{stakers} is the fraction of the total token emission during
// the previous epoch that was paid to reward staking network participants,
// i.e. reputers and network validators
func GetMaximumMonthlyEmissionPerUnitStakedToken(
	maximumMonthlyPercentageYield math.LegacyDec,
	reputersPercentOfTopicRewards math.LegacyDec,
	validatorsPercent math.LegacyDec,
) math.LegacyDec {
	// the reputers percent is in terms of the emission to workers and reputers, not including validators
	// e.g. if 1/4 goes to validators, then of the 3/4 that goes to workers and reputers, reputers got 1/3
	// so you have to do 1/3 *3/4 = ACTUAL percent to reputers of the total emission
	reputersPercent := math.LegacyOneDec().Sub(validatorsPercent).Mul(reputersPercentOfTopicRewards)
	f_stakers := reputersPercent.Add(validatorsPercent) //nolint:revive // var-naming: don't use underscores in Go names
	return maximumMonthlyPercentageYield.Quo(f_stakers)
}

// Target Monthly Emission Per Unit Staked Token
// is the capped value of either the computed target for this
// timestep, or simply the maximum allowable emission per unit
// staked token
// ^e_i = min(^e_{target,i}, ^e_{max,i})
func GetCappedTargetEmissionPerUnitStakedToken(
	targetRewardEmissionPerUnitStakedToken math.LegacyDec,
	maximumMonthlyEmissionPerUnitStakedToken math.LegacyDec,
) math.LegacyDec {
	if targetRewardEmissionPerUnitStakedToken.GT(maximumMonthlyEmissionPerUnitStakedToken) {
		return maximumMonthlyEmissionPerUnitStakedToken
	}
	return targetRewardEmissionPerUnitStakedToken
}

// Target Reward Emission Per Unit Staked Token
// controls the inflation rate of the token issuance
//
// ^e_i = ((f_e*T_{total,i}) / N_{staked,i}) * (N_{circ,i} / N_{total,i})
//
// f_e is a global tuning constant, by default f_e = 0.015 month^{−1}
// represents the fraction of the ecosystem treasury that would ideally
// be emitted per unit time.
// T_{total,i} = number of tokens that the ecosystem bucket can still mint PLUS
// the current balance of the bucket.
// The ecosystem bucket is capped to be able to mint by default 36.75% of the max supply,
// but as more tokens are minted the amount the ecosystem is permitted to mint decreases.
// N_{staked,i} is the total number of tokens staked on the network at timestep i
// N_{circ,i} is the number of tokens in circulation at timestep i
// N_{total,i} is the total number of tokens ever allowed to exist
func GetTargetRewardEmissionPerUnitStakedToken(
	fEmission math.LegacyDec,
	ecosystemLocked math.Int,
	networkStaked math.Int,
	circulatingSupply math.Int,
	maxSupply math.Int,
) (math.LegacyDec, error) {
	if networkStaked.IsZero() ||
		maxSupply.IsZero() {
		return math.LegacyDec{}, errors.Wrapf(
			types.ErrZeroDenominator,
			"denominator is zero: %s | %s",
			networkStaked.String(),
			maxSupply.String(),
		)
	}
	// T_{total,i} = ecosystemLocked
	// N_{staked,i} = networkStaked
	// N_{circ,i} = circulatingSupply
	// N_{total,i} = totalSupply
	ratioCirculating := circulatingSupply.ToLegacyDec().Quo(maxSupply.ToLegacyDec())
	ratioEcosystemToStaked := ecosystemLocked.ToLegacyDec().Quo(networkStaked.ToLegacyDec())
	ret := fEmission.
		Mul(ratioEcosystemToStaked).
		Mul(ratioCirculating)
	if ret.IsNegative() {
		return math.LegacyDec{}, errors.Wrapf(
			types.ErrNegativeTargetEmissionPerToken,
			"target emission per token is negative: %s | %s | %s",
			ratioCirculating.String(),
			ratioEcosystemToStaked.String(),
			ret.String(),
		)
	}

	return ret, nil
}

// Reward Emission Per Unit Staked Token is an exponential moving
// average over the Target Reward Emission Per Unit Staked Token
// e_i = α_e * ^e_i + (1 − α_e)*e_{i−1}
func GetExponentialMovingAverage(
	targetRewardEmissionPerUnitStakedToken math.LegacyDec,
	alphaEmission math.LegacyDec,
	previousRewardEmissionPerUnitStakedToken math.LegacyDec,
) math.LegacyDec {
	firstTerm := alphaEmission.Mul(targetRewardEmissionPerUnitStakedToken)
	inverseAlpha := math.OneInt().ToLegacyDec().Sub(alphaEmission)
	secondTerm := inverseAlpha.Mul(previousRewardEmissionPerUnitStakedToken)
	return firstTerm.Add(secondTerm)
}

// How many tokens are left that the ecosystem bucket is allowed to mint?
func (k Keeper) GetEcosystemMintSupplyRemaining(
	ctx context.Context,
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

// RecalculateTargetEmission recalculates the target emission for the network
// It writes the old emission rates to the store for
// PreviousRewardEmissionPerUnitStakedToken and PreviousBlockEmission
// Then it calculates the new target emission rate
// returns that, and the block emission we should mint/send for this block
func RecalculateTargetEmission(
	ctx sdk.Context,
	k keeper.Keeper,
	blockHeight uint64,
	blocksPerMonth uint64,
	moduleParams types.Params,
	ecosystemBalance math.Int,
	ecosystemMintSupplyRemaining math.Int,
	validatorsPercent math.LegacyDec,
) (
	blockEmission math.Int,
	emissionPerUnitStakedToke math.LegacyDec, // e_i in the whitepaper
	err error,
) {
	emissionPerMonth, emissionPerUnitStakedToken, err := GetEmissionPerMonth(
		ctx,
		k,
		blockHeight,
		blocksPerMonth,
		moduleParams,
		ecosystemBalance,
		ecosystemMintSupplyRemaining,
		validatorsPercent,
	)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, err
	}
	// emission/block = (emission/month) / (block/month)
	blockEmission = emissionPerMonth.
		Quo(math.NewIntFromUint64(blocksPerMonth))

	// set the previous emissions to this block's emissions
	err = k.PreviousRewardEmissionPerUnitStakedToken.Set(ctx, emissionPerUnitStakedToken)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, errors.Wrap(err, "error setting previous reward emission per unit staked token")
	}
	err = k.PreviousBlockEmission.Set(ctx, blockEmission)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, errors.Wrap(err, "error setting previous block emission")
	}

	k.Logger(ctx).Info("Emissions Update",
		"emissionPerUnitStakedToken", emissionPerUnitStakedToken.String(),
		"emissionPerMonth", emissionPerMonth.String(),
		"blockEmission", blockEmission.String(),
	)
	return blockEmission, emissionPerUnitStakedToken, nil
}

// GetEmissionPerMonth calculates the emission per month for the network
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
	circulatingSupply,
		totalSupply,
		lockedVestingTokens,
		ecosystemLocked,
		err := keeper.GetCirculatingSupply(ctx, k, params, blockHeight, blocksPerMonth)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, err
	}
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
