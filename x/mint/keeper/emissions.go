package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// return the uncirculating supply, i.e. tokens on a vesting schedule
// latest discussion on how these tokens should be handled lives in ORA-1111
// probably thee tokens will be custodied off chain and this function will
// just return the circulating supply based off of what the agreements off chain
// were supposed to be at time of chain-genesis
func numberLockedTokens() math.Int {
	return math.ZeroInt()
}

// The total amount of tokens emitted as rewards at timestep i
// E_i = e_i*N_{staked,i}
// where e_i is the emission per unit staked token
// and N_{staked,i} is the total amount of tokens staked at timestep i
func TotalEmissionPerTimestep(
	rewardEmissionPerUnitStakedToken math.LegacyDec,
	numStakedTokens math.Int,
) math.LegacyDec {
	return rewardEmissionPerUnitStakedToken.MulInt(numStakedTokens)
}

// Target Reward Emission Per Unit Staked Token
// controls the inflation rate of the token issuance
//
// ^e_i = (f_e*T_{total,i}) / N_{staked,i}) * (N_{circ,i} / N_{total,i})
//
// where T_{total,i} is the total number of tokens held by the ecosystem
// treasury, N_{total,i} is the total token supply, N_{circ,i} is the
// circulating supply, and N_{staked,i} is the staked supply. The
// factor f_e = 0.015 month^{−1} represents the fraction of the
// ecosystem treasury that would ideally be emitted per unit time.
func TargetRewardEmissionPerUnitStakedToken(
	ctx sdk.Context,
	k Keeper,
	f_emission math.LegacyDec,
) (math.LegacyDec, error) {
	// T_{total,i}
	ecosystemAddr := k.accountKeeper.GetModuleAddress(types.EcosystemModuleName)
	params, err := k.Params.Get(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}
	ecosystemBalance := k.bankKeeper.GetBalance(ctx, ecosystemAddr, params.MintDenom).Amount

	// N_{circ,i}, N_{total,i}
	lockedSupply := numberLockedTokens()
	totalSupply := k.GetSupply(ctx).Amount
	circulatingSupply := totalSupply.Sub(lockedSupply)
	if circulatingSupply.IsNegative() {
		return math.LegacyDec{}, errors.Wrapf(
			types.ErrNegativeCirculatingSupply,
			"circulating supply is negative: %s | %s | %s",
			circulatingSupply.String(),
			totalSupply.String(),
			lockedSupply.String(),
		)
	}

	// N_{staked,i}
	cosmosValidatorsStaked, err := k.StakingTokenSupply(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}
	reputersStaked, err := k.emissionsKeeper.GetTotalStake(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}
	networkStaked := cosmosValidatorsStaked.Add(math.NewIntFromBigInt(reputersStaked.BigInt()))

	// ^e_i
	targetEmissionPerToken := f_emission.MulInt(ecosystemBalance).QuoInt(networkStaked).MulInt(circulatingSupply).QuoInt(totalSupply)
	if targetEmissionPerToken.IsNegative() {
		return math.LegacyDec{}, errors.Wrapf(
			types.ErrNegativeTargetEmissionPerToken,
			"target emission per token is negative: %s | %s | %s | %s | %s",
			targetEmissionPerToken.String(),
			f_emission.String(),
			ecosystemBalance.String(),
			networkStaked.String(),
			circulatingSupply.String(),
		)
	}
	return targetEmissionPerToken, nil
}

// Reward Emission Per Unit Staked Token is an exponential moving
// average over the Target Reward Emission Per Unit Staked Token
// e_i = α_e * ^e_i + (1 − α_e)*e_{i−1}
func RewardEmissionPerUnitStakedToken(
	targetRewardEmissionPerUnitStakedToken math.LegacyDec,
	alphaEmission math.LegacyDec,
	previousRewardEmissionPerUnitStakedToken math.LegacyDec,
) math.LegacyDec {
	return alphaEmission.Mul(targetRewardEmissionPerUnitStakedToken).Add(
		math.LegacyOneDec().Sub(alphaEmission).Mul(previousRewardEmissionPerUnitStakedToken),
	)
}

// a_e needs to be set to the correct value for the timestep in question
// a_e has a fiduciary value of 0.1 but that's for a one-month timestep
// so it must be corrected for the block timestep
// default block time is 6311520 blocks per year aka 5 seconds per block
// ^α_e = 1 − (1 − α_e)^(∆t/month)
// where ˆαe is the recalibrated form of α_e appropriate for an update time step ∆t
func smoothingFactorPerBlock(
	ctx sdk.Context,
	k Keeper,
	oneMonthSmoothingFactor math.LegacyDec,
) (math.LegacyDec, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}
	deltaTimestepPerMonth := params.BlocksPerYear / 12
	return math.LegacyOneDec().Sub(
		math.LegacyOneDec().Sub(oneMonthSmoothingFactor).Power(deltaTimestepPerMonth),
	), nil
}
