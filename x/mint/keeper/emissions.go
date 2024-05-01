package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// return the uncirculating supply, i.e. tokens on a vesting schedule
// these tokens will be custodied by a centralized actor off chain.
// this function returns the circulating supply based off of what
// the agreements off chain say were supposed to happen for token lockup
func GetLockedTokenSupply(
	blockHeight math.Int,
	params types.Params,
) math.Int {
	// foundation is unlocked from genesis
	// participants are unlocked from genesis
	//investors and team tokens are locked on a 1 year cliff three year vesting schedule
	blocksInAYear := math.NewIntFromUint64(params.BlocksPerMonth * 12)
	blocksInThreeYears := blocksInAYear.Mul(math.NewInt(3))
	maxSupply := params.MaxSupply.ToLegacyDec()
	percentInvestors := params.InvestorsPercentOfTotalSupply
	percentTeam := params.TeamPercentOfTotalSupply
	fullInvestors := percentInvestors.Mul(maxSupply).TruncateInt()
	fullTeam := percentTeam.Mul(maxSupply).TruncateInt()
	var investors, team math.Int
	if blockHeight.LT(blocksInAYear) {
		// less than a year, completely locked
		investors = fullInvestors
		team = fullTeam
	} else if blockHeight.GTE(blocksInAYear) && blockHeight.LT(blocksInThreeYears) {
		// between 1 and 3 years, investors and team tokens are vesting and partially unlocked
		thirtySix := math.LegacyNewDec(36)
		monthsUnlocked := blockHeight.Quo(math.NewIntFromUint64(params.BlocksPerMonth)).ToLegacyDec()
		monthsLocked := thirtySix.Sub(monthsUnlocked)
		investors = monthsLocked.Quo(thirtySix).Mul(fullInvestors.ToLegacyDec()).TruncateInt()
		team = monthsLocked.Quo(thirtySix).Mul(fullTeam.ToLegacyDec()).TruncateInt()
	} else {
		// greater than 3 years, all investor, team tokens are unlocked
		investors = math.ZeroInt()
		team = math.ZeroInt()
	}
	return investors.Add(team)
}

// helper function to get the number of staked tokens on the network
// includes both tokens staked by cosmos validators (cosmos staking)
// and tokens staked by reputers (allora staking)
func GetNumStakedTokens(ctx context.Context, k Keeper) (math.Int, error) {
	cosmosValidatorsStaked, err := k.CosmosValidatorStakedSupply(ctx)
	if err != nil {
		return math.Int{}, err
	}
	reputersStaked, err := k.emissionsKeeper.GetTotalStake(ctx)
	if err != nil {
		return math.Int{}, err
	}
	return cosmosValidatorsStaked.Add(math.NewIntFromBigInt(reputersStaked.BigInt())), nil
}

// The total amount of tokens emitted as rewards at timestep i
// E_i = e_i*N_{staked,i}
// where e_i is the emission per unit staked token
// and N_{staked,i} is the total amount of tokens staked at timestep i
// THIS FUNCTION TRUNCATES THE RESULT DIVISION TO AN INTEGER
func GetTotalEmissionPerTimestep(
	rewardEmissionPerUnitStakedToken math.LegacyDec,
	numStakedTokens math.Int,
) math.Int {
	return rewardEmissionPerUnitStakedToken.MulInt(numStakedTokens).TruncateInt()
}

// Target Reward Emission Per Unit Staked Token
// controls the inflation rate of the token issuance
//
// ^e_i = ((f_e*T_{total,i}) / N_{staked,i}) * (N_{circ,i} / N_{total,i})
//
// f_e is a global tuning constant, by default f_e = 0.015 month^{−1}
// represents the fraction of the ecosystem treasury that would ideally
// be emitted per unit time.
// T_{total,i} = number of tokens that the ecosystem bucket can still mint.
// The ecosystem bucket is capped to be able to mint by default 36.75% of the max supply,
// but as more tokens are minted the amount the ecosystem is permitted to mint decreases.
// N_{staked,i} is the total number of tokens staked on the network at timestep i
// N_{circ,i} is the number of tokens in circulation at timestep i
// N_{total,i} is the total number of tokens ever allowed to exist
func GetTargetRewardEmissionPerUnitStakedToken(
	fEmission math.LegacyDec,
	ecosystemMintableRemaining math.Int,
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
	// T_{total,i} = ecosystemMintableRemaining
	// N_{staked,i} = networkStaked
	// N_{circ,i} = circulatingSupply
	// N_{total,i} = totalSupply
	ratioCirculating := circulatingSupply.ToLegacyDec().Quo(maxSupply.ToLegacyDec())
	ratioEcosystemToStaked := ecosystemMintableRemaining.ToLegacyDec().Quo(networkStaked.ToLegacyDec())
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
	firstTerm := targetRewardEmissionPerUnitStakedToken.Mul(alphaEmission)
	secondTerm := math.OneInt().ToLegacyDec().Sub(alphaEmission).
		Mul(previousRewardEmissionPerUnitStakedToken)
	return firstTerm.Add(secondTerm)
}

// a_e needs to be set to the correct value for the timestep in question
// a_e has a fiduciary value of 0.1 but that's for a one-month timestep
// so it must be corrected for whatever timestep we actually use
// in this first version of the allora network we will use a "daily" timestep
// so the value for delta t should be 30 (assuming a perfect world of 30 day months)
// ^α_e = 1 - (1 - α_e)^(∆t/month)
func GetSmoothingFactorPerTimestep(
	ctx sdk.Context,
	k Keeper,
	a_e math.LegacyDec,
	dt uint64,
) math.LegacyDec {
	base := math.OneInt().ToLegacyDec().Sub(a_e)
	secondTerm := base.Power(dt)
	return math.OneInt().ToLegacyDec().Sub(secondTerm)
}
