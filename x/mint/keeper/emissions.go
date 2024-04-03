package keeper

import (
	"context"

	bn "math/big"

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
	rewardEmissionPerUnitStakedTokenNumerator math.Int,
	rewardEmissionPerUnitStakedTokenDenominator math.Int,
	numStakedTokens math.Int,
) math.Int {
	return rewardEmissionPerUnitStakedTokenNumerator.Mul(numStakedTokens).Quo(rewardEmissionPerUnitStakedTokenDenominator)
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
) (math.Int, math.Int, error) {
	// T_{total,i} = ecosystemBalance
	// N_{staked,i} = networkStaked
	// N_{circ,i} = circulatingSupply
	// N_{total,i} = totalSupply
	numerator := fEmissionNumerator.Mul(ecosystemBalance).Mul(circulatingSupply)
	denominator := networkStaked.Mul(totalSupply).Mul(fEmissionDenominator)
	if numerator.IsNegative() || denominator.IsNegative() {
		return math.Int{}, math.Int{}, errors.Wrapf(
			types.ErrNegativeTargetEmissionPerToken,
			"target emission per token is negative: %s | %s",
			numerator.String(),
			denominator.String(),
		)
	}
	if denominator.IsZero() {
		return math.Int{}, math.Int{}, errors.Wrapf(
			types.ErrZeroDenominator,
			"denominator is zero: %s | %s",
			networkStaked.String(),
			fEmissionDenominator.String(),
		)
	}
	return numerator, denominator, nil
}

// Reward Emission Per Unit Staked Token is an exponential moving
// average over the Target Reward Emission Per Unit Staked Token
// e_i = α_e * ^e_i + (1 − α_e)*e_{i−1}
// all the terms are represented as a numerator and denominator
// So α_e can be written as α_e_numerator (a_en) / α_e_denominator (a_ed)
// and e_i can be written as e_i_numerator (e_in) / e_i_denominator (e_id), etc
// also to reduce confusion with exponentiation since latex doesn't translate
// to ascii comments well we write ^e_i from the paper as e'_i
// which makes our formula:
// e_in / e_id = a_en/a_ed * e'_in/e'_id + (1 − a_en/a_ed)*(e_{i−1}n / e_{i−1}d)
// e_in / e_id = (a_en*e'_in / a_ed*e'_id) + (((a_ed - a_en)*e_{i−1}n) / (a_ed * e_{i−1}d))
// e_in / e_id = (a_en*e'_in*e_{i-1}d + (a_ed - a_en)*e_{i−1}n*e'_id) / (a_ed * e'_id*e_{i−1}d)
// and we return the numerator and denominator separately
func RewardEmissionPerUnitStakedToken(
	targetRewardEmissionPerUnitStakedTokenNumerator math.Int,
	targetRewardEmissionPerUnitStakedTokenDenominator math.Int,
	alphaEmissionNumerator math.Int,
	alphaEmissionDenominator math.Int,
	previousRewardEmissionPerUnitStakedTokenNumerator math.Int,
	previousRewardEmissionPerUnitStakedTokenDenominator math.Int,
) (numerator math.Int, denominator math.Int) {
	// e_in = (a_en*e'_in*e_{i-1}d + (a_ed - a_en)*e_{i−1}n*e'_id)
	// first term
	// a_en*e'_in*e_{i-1}d
	firstTerm := alphaEmissionNumerator.
		Mul(targetRewardEmissionPerUnitStakedTokenNumerator).
		Mul(previousRewardEmissionPerUnitStakedTokenDenominator)
	// second term
	// (a_ed - a_en)*e_{i−1}n*e'_id)
	parens := alphaEmissionDenominator.Sub(alphaEmissionNumerator)
	secondTerm := parens.Mul(previousRewardEmissionPerUnitStakedTokenNumerator).
		Mul(targetRewardEmissionPerUnitStakedTokenDenominator)
	numerator = firstTerm.Add(secondTerm)
	// e_id = (a_ed * e'_id*e_{i−1}d)
	denominator = alphaEmissionDenominator.
		Mul(targetRewardEmissionPerUnitStakedTokenDenominator).
		Mul(previousRewardEmissionPerUnitStakedTokenDenominator)
	return numerator, denominator
}

// a_e needs to be set to the correct value for the timestep in question
// a_e has a fiduciary value of 0.1 but that's for a one-month timestep
// so it must be corrected for whatever timestep we actually use
// default block time is 6311520 blocks per year aka 5 seconds per block
// in this first version of the allora network we will use a "daily" timestep
// so the value for delta t should be 30 (assuming a perfect world of 30 day months)
// ^α_e = 1 - (1 - α_e)^(∆t/month)
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
