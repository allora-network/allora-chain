package module

import (
	"fmt"
	"math"

	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

// GetWorkerScore calculates the worker score based on the losses and lossesCut.
// Consider the staked weighted inference loss and one-out loss to calculate the worker score.
// T_ij / T_ik / T^-_ik / T^+_ik
func GetWorkerScore(losses, lossesCut float64) float64 {
	deltaLogLoss := math.Log10(lossesCut) - math.Log10(losses)
	return deltaLogLoss
}

// GetStakeWeightedLoss calculates the stake-weighted average loss.
// Consider the losses and the stake of each reputer to calculate the stake-weighted loss.
// The stake weighted loss is used to calculate the network-wide losses.
// L_i / L_ij / L_ik / L^-_i / L^-_il / L^+_ik
func GetStakeWeightedLoss(reputersStakes, reputersReportedLosses []float64) (float64, error) {
	if len(reputersStakes) != len(reputersReportedLosses) {
		return 0, fmt.Errorf("slices must have the same length")
	}

	totalStake := 0.0
	for _, stake := range reputersStakes {
		totalStake += stake
	}

	if totalStake == 0 {
		return 0, fmt.Errorf("total stake cannot be zero")
	}

	var stakeWeightedLoss float64 = 0
	for i, loss := range reputersReportedLosses {
		if loss <= 0 {
			return 0, fmt.Errorf("loss values must be greater than zero")
		}
		weightedLoss := (reputersStakes[i] / totalStake) * math.Log10(loss)
		stakeWeightedLoss += weightedLoss
	}

	return stakeWeightedLoss, nil
}

// Implements the potential function phi for the module
// this is equation 6 from the litepaper:
// ϕ_p(x) = (ln(1 + e^x))^p
//
// error handling:
// float Inf can be generated for values greater than 1.7976931348623157e+308
// e^x can create Inf
// ln(blah)^p can create Inf for sufficiently large ln result
// NaN is impossible as 1+e^x is always positive no matter the value of x
// and pow only produces NaN for NaN input
// therefore we only return one type of error and that is if phi overflows.
func phi(p float64, x float64) (float64, error) {
	if math.IsNaN(p) || math.IsInf(p, 0) || math.IsNaN(x) || math.IsInf(x, 0) {
		return 0, emissions.ErrPhiInvalidInput
	}
	eToTheX := math.Exp(x)
	fmt.Println(eToTheX)
	onePlusEToTheX := 1 + eToTheX
	if math.IsInf(onePlusEToTheX, 0) {
		return 0, emissions.ErrEToTheXExponentiationIsInfinity
	}
	naturalLog := math.Log(onePlusEToTheX)
	fmt.Println(naturalLog)
	result := math.Pow(naturalLog, p)
	fmt.Println(result)
	if math.IsInf(result, 0) {
		return 0, emissions.ErrLnToThePExponentiationIsInfinity
	}
	// should theoretically never be possible with the above checks
	if math.IsNaN(result) {
		return 0, emissions.ErrPhiResultIsNaN
	}
	return result, nil
}

// Adjusted stake for calculating consensus S hat
// ^S_im = 1 - ϕ_1^−1(η) * ϕ1[ −η * (((N_r * a_im * S_im) / (Σ_m * a_im * S_im)) − 1 )]
// we use eta = 20
// phi_1 refers to the phi function with p = 1
func adjustedStake(stake float64, listeningCoefficient float64, numReputers float64) (float64, error) {
	return 0, nil
}
