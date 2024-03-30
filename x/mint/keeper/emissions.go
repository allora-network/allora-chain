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
// 0.3675
const EcosystemTreasuryPercentOfTotalSupply = 3675
const EcosystemTreasuryPercentOfTotalSupplyPrecision = 4

// return the uncirculating supply, i.e. tokens on a vesting schedule
// latest discussion on how these tokens should be handled lives in ORA-1111
// probably thee tokens will be custodied off chain and this function will
// just return the circulating supply based off of what the agreements off chain
// were supposed to be at time of chain-genesis
func numberLockedTokens() math.Int {
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
func TotalEmissionPerTimestep(
	rewardEmissionPerUnitStakedToken math.Int,
	numStakedTokens math.Int,
) math.Int {
	return rewardEmissionPerUnitStakedToken.Mul(numStakedTokens)
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
	fEmission math.Int,
	fEmissionPrec uint64,
	ecosystemBalance math.Int,
	networkStaked math.Int,
) (math.Int, error) {
	// T_{total,i} = ecosystemBalance
	// N_{circ,i}, N_{total,i}
	lockedSupply := numberLockedTokens()
	totalSupply := k.GetSupply(ctx).Amount
	circulatingSupply := totalSupply.Sub(lockedSupply)
	if circulatingSupply.IsNegative() {
		return math.Int{}, errors.Wrapf(
			types.ErrNegativeCirculatingSupply,
			"circulating supply is negative: %s | %s | %s",
			circulatingSupply.String(),
			totalSupply.String(),
			lockedSupply.String(),
		)
	}

	// N_{staked,i} = networkStaked
	// ^e_i
	// avoid truncation error
	numerator := fEmission.Mul(ecosystemBalance).Mul(circulatingSupply)
	denominator := networkStaked.Mul(totalSupply).Mul(math.NewIntFromUint64(fEmissionPrec))
	targetEmissionPerToken := numerator.Quo(denominator)
	if targetEmissionPerToken.IsNegative() {
		return math.Int{}, errors.Wrapf(
			types.ErrNegativeTargetEmissionPerToken,
			"target emission per token is negative: %s | %s | %s | %s | %s",
			targetEmissionPerToken.String(),
			fEmission.String(),
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
// alpha emission is represented as a numerator and denominator
// separately because cosmos math doesn't have a non deprecated float type.
// So α_e can be written as α_e_numerator (a_en) / α_e_denominator (a_ed)
// also to reduce confusion with exponentiation since latex doesn't translate
// to ascii comments well we write ^e_i from the paper as e'_i
// which makes our formula:
// e_i = a_en/a_ed * e'_i + (1 − a_en/a_ed)*e_{i−1}
// e_i = a_en*e'_i / a_ed + ( (a_ed - a_en) / a_ed )*e_{i−1}
// e_i = a_en*e'_i / a_ed + (a_ed - a_en) * e_{i−1} / a_ed
// e_i = ( a_en*e'_i + (a_ed - a_en)*e_{i−1} ) / a_ed
// and here we truncate after the division, to end with a math.int
func RewardEmissionPerUnitStakedToken(
	targetRewardEmissionPerUnitStakedToken math.Int,
	alphaEmissionNumerator math.Int,
	alphaEmissionDenominator math.Int,
	previousRewardEmissionPerUnitStakedToken math.Int,
) math.Int {
	// a_en * e'_i
	a_en_Mul_e_prime := alphaEmissionNumerator.Mul(targetRewardEmissionPerUnitStakedToken)
	// a_ed - a_en
	a_ed_Sub_a_en := alphaEmissionDenominator.Sub(alphaEmissionNumerator)
	// (a_ed - a_en)*e_{i−1}
	a_ed_Sub_a_en_Mul_e_prev := a_ed_Sub_a_en.Mul(previousRewardEmissionPerUnitStakedToken)
	// ( a_en*e'_i + (a_ed - a_en)*e_{i−1} )
	numerator := a_en_Mul_e_prime.Add(a_ed_Sub_a_en_Mul_e_prev)
	// e_i = ( a_en*e'_i + (a_ed - a_en)*e_{i−1} ) / a_ed
	return numerator.Quo(alphaEmissionDenominator)
}

// a_e needs to be set to the correct value for the timestep in question
// a_e has a fiduciary value of 0.1 but that's for a one-month timestep
// so it must be corrected for the block timestep
// default block time is 6311520 blocks per year aka 5 seconds per block
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
func SmoothingFactorPerBlock(
	ctx sdk.Context,
	k Keeper,
	a_en math.Int,
	a_ed uint64,
) (math.Int, math.Int, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return math.Int{}, math.Int{}, err
	}
	Dt := bn.NewInt(0).SetUint64(params.BlocksPerYear / 12)
	A_en := a_en.BigInt()
	A_ed := math.NewIntFromUint64(a_ed).BigInt()

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
	return numerator, denominator, nil
}
