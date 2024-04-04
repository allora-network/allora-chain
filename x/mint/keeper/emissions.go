package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// set once at genesis and never changed after
// 0.3675 = 36.75%
const EcosystemTreasuryPercentOfTotalSupplyNumerator = 3675
const EcosystemTreasuryPercentOfTotalSupplyDenominator = 10000

// return the uncirculating supply, i.e. tokens on a vesting schedule
// latest discussion on how these tokens should be handled lives in ORA-1111
// probably thee tokens will be custodied off chain and this function will
// just return the circulating supply based off of what the agreements off chain
// were supposed to be at time of chain-genesis
func GetLockedTokenSupply() math.Int {
	return math.ZeroInt()
}

// helper function to get the number of staked tokens on the network
// includes both tokens staked by cosmos validators (cosmos staking)
// and tokens staked by reputers (allora staking)
func GetNumStakedTokens(ctx context.Context, k Keeper) (math.Int, error) {
	cosmosValidatorsStaked, err := k.StakingTokenSupply(ctx)
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
func TotalEmissionPerTimestep(
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
// where T_{total,i} is the total number of tokens held by the ecosystem
// treasury, N_{total,i} is the total token supply, N_{circ,i} is the
// circulating supply, and N_{staked,i} is the staked supply. The
// factor f_e = 0.015 month^{−1} represents the fraction of the
// ecosystem treasury that would ideally be emitted per unit time.
// pass f_e as a fractional value, numerator and denominator as separate args
func TargetRewardEmissionPerUnitStakedToken(
	fEmissionNumerator math.Int,
	fEmissionDenominator math.Int,
	ecosystemBalance math.Int,
	networkStaked math.Int,
	circulatingSupply math.Int,
	totalSupply math.Int,
) (math.LegacyDec, error) {
	if fEmissionDenominator.IsZero() ||
		networkStaked.IsZero() ||
		totalSupply.IsZero() {
		return math.LegacyDec{}, errors.Wrapf(
			types.ErrZeroDenominator,
			"denominator is zero: %s | %s",
			networkStaked.String(),
			fEmissionDenominator.String(),
		)
	}
	// T_{total,i} = ecosystemBalance
	// N_{staked,i} = networkStaked
	// N_{circ,i} = circulatingSupply
	// N_{total,i} = totalSupply
	ratioCirculating := circulatingSupply.ToLegacyDec().Quo(totalSupply.ToLegacyDec())
	ratioEcosystemToStaked := ecosystemBalance.ToLegacyDec().Quo(networkStaked.ToLegacyDec())
	ret := fEmissionNumerator.ToLegacyDec().
		Mul(ratioEcosystemToStaked).
		Mul(ratioCirculating).
		Quo(fEmissionDenominator.ToLegacyDec())
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
func RewardEmissionPerUnitStakedToken(
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
func SmoothingFactorPerTimestep(
	ctx sdk.Context,
	k Keeper,
	a_en math.Int,
	a_ed math.Int,
	dt uint64,
) math.LegacyDec {
	a_e := a_en.ToLegacyDec().Quo(a_ed.ToLegacyDec())
	base := math.OneInt().ToLegacyDec().Sub(a_e)
	secondTerm := base.Power(dt)
	return math.OneInt().ToLegacyDec().Sub(secondTerm)
}

// where ˆαe is the recalibrated form of α_e appropriate for an update time step ∆t
//
// due to using math.int rather than having a float like data structure available to us
// we actually encode α_e as α_e_numerator (a_en) and α_e_denominator (a_ed)
// ∆t/month is a pain in the butt to write in code so lets just call that dt
// now we have:
// ^α_e = 1 - (1 - a_en/a_ed)^dt
// ^α_e = 1 - (a_ed/a_ed - a_en/a_ed)^dt
// ^α_e = 1 - ((a_ed-a_en)/a_ed)^dt
// ^α_e = 1 - ((a_ed-a_en)^dt)/((a_ed)^dt)
// ^α_e = (a_ed)^dt)/((a_ed)^dt) - ((a_ed-a_en)^dt)/((a_ed)^dt)
// and the actual math we'll use in this function:
// ^α_e = ((a_ed)^dt - ((a_ed-a_en)^dt)) / ((a_ed)^dt)
/*
func SmoothingFactorPerTimestep(
	ctx sdk.Context,
	k Keeper,
	a_en math.Int,
	a_ed math.Int,
	dt uint64,
) (math.Int, math.Int) {
	Dt := bn.NewInt(0).SetUint64(dt)
	A_en := a_en.BigInt()
	A_ed := a_ed.BigInt()

	//(a_ed)^dt
	A_ed_Exp_Dt := bn.NewInt(0).Exp(A_ed, Dt, nil)

	//((a_ed-a_en)^dt))
	A_ed_Sub_A_en := bn.NewInt(0).Sub(A_ed, A_en)
	A_ed_Sub_A_en_Exp_Dt := bn.NewInt(0).Exp(A_ed_Sub_A_en, Dt, nil)

	//((a_ed)^dt - ((a_ed-a_en)^dt))
	numerator := math.NewIntFromBigInt(
		bn.NewInt(0).Sub(A_ed_Exp_Dt, A_ed_Sub_A_en_Exp_Dt),
	)
	//((a_ed)^dt)
	denominator := math.NewIntFromBigInt(A_ed_Exp_Dt)
	return numerator, denominator
}
*/
