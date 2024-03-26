package module

import (
	"math"

	errors "cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// StdDev calculates the standard deviation of a slice of float64.
// stdDev = sqrt((Σ(x - μ))^2/ N)
// where μ is mean and N is number of elements
func StdDev(data []float64) float64 {
	var mean, sd float64
	for _, v := range data {
		mean += v
	}
	mean /= float64(len(data))
	for _, v := range data {
		sd += math.Pow(v-mean, 2)
	}
	sd = math.Sqrt(sd / float64(len(data)))
	return sd
}

// flatten converts a double slice of float64 to a single slice of float64
func flatten(arr [][]float64) []float64 {
	var flat []float64
	for _, row := range arr {
		flat = append(flat, row...)
	}
	return flat
}

// GetWorkerRewardFractions calculates the reward fractions for workers for forecast and inference tasks
// U_ij / V_ik
func GetWorkerRewardFractions(scores [][]float64, preward float64) ([]float64, error) {
	lastScores := make([][]float64, len(scores))
	for i, workerScores := range scores {
		end := len(workerScores)
		start := end - 10
		if start < 0 {
			start = 0
		}
		lastScores[i] = workerScores[start:end]
	}

	stdDev := StdDev(flatten(lastScores))
	var normalizedScores []float64
	for _, score := range lastScores {
		normalizedScores = append(normalizedScores, score[len(score)-1]/stdDev)
	}

	smoothedScores := make([]float64, len(normalizedScores))
	for i, v := range normalizedScores {
		res, err := phi(preward, v)
		if err != nil {
			return nil, err
		}
		smoothedScores[i] = res
	}

	total := 0.0
	for _, score := range smoothedScores {
		total += score
	}

	var rewardFractions []float64
	for _, score := range smoothedScores {
		rewardFraction := score / total
		rewardFractions = append(rewardFractions, rewardFraction)
	}

	return rewardFractions, nil
}

// GetReputerRewardFractions calculates the reward fractions for each reputer based on their stakes, scores, and preward parameter.
// W_im
func GetReputerRewardFractions(stakes, scores []float64, preward float64) ([]float64, error) {
	if len(stakes) != len(scores) {
		return nil, types.ErrInvalidSliceLength
	}

	// Calculate (stakes * scores)^preward and sum of all fractions
	var totalFraction float64
	fractions := make([]float64, len(stakes))
	for i, stake := range stakes {
		fractions[i] = math.Pow(stake*scores[i], preward)
		totalFraction += fractions[i]
	}

	// Normalize fractions
	for i := range fractions {
		fractions[i] /= totalFraction
	}

	return fractions, nil
}

// GetfUniqueAgg calculates the unique value or impact of each forecaster.
// ƒ^+
func GetfUniqueAgg(numForecasters float64) float64 {
	return 1.0 / math.Pow(2.0, (numForecasters-1.0))
}

// GetFinalWorkerScoreForecastTask calculates the worker score in forecast task.
// T_ik
func GetFinalWorkerScoreForecastTask(scoreOneIn, scoreOneOut, fUniqueAgg float64) float64 {
	return fUniqueAgg*scoreOneIn + (1-fUniqueAgg)*scoreOneOut
}

// GetWorkerScore calculates the worker score based on the losses and lossesCut.
// Consider the staked weighted inference loss and one-out loss to calculate the worker score.
// T_ij / T^-_ik / T^+_ik
func GetWorkerScore(losses, lossesOneOut float64) float64 {
	deltaLogLoss := math.Log10(lossesOneOut) - math.Log10(losses)
	return deltaLogLoss
}

// GetStakeWeightedLoss calculates the stake-weighted average loss.
// Consider the losses and the stake of each reputer to calculate the stake-weighted loss.
// The stake weighted loss is used to calculate the network-wide losses.
// L_i / L_ij / L_ik / L^-_i / L^-_il / L^+_ik
func GetStakeWeightedLoss(reputersStakes, reputersReportedLosses []float64) (float64, error) {
	if len(reputersStakes) != len(reputersReportedLosses) {
		return 0, types.ErrInvalidSliceLength
	}

	totalStake := 0.0
	for _, stake := range reputersStakes {
		totalStake += stake
	}

	var stakeWeightedLoss float64 = 0
	for i, loss := range reputersReportedLosses {
		weightedLoss := (reputersStakes[i] * math.Log10(loss)) / totalStake
		stakeWeightedLoss += weightedLoss
	}

	return stakeWeightedLoss, nil
}

// GetStakeWeightedLossMatrix calculates the stake-weighted geometric mean of the losses to generate the consensus vector.
// L_i - consensus loss vector
func GetStakeWeightedLossMatrix(reputersAdjustedStakes []float64, reputersReportedLosses [][]float64) ([]float64, error) {
	if len(reputersAdjustedStakes) == 0 || len(reputersReportedLosses) == 0 {
		return nil, types.ErrInvalidSliceLength
	}

	// Calculate total stake for normalization
	totalStake := 0.0
	for _, stake := range reputersAdjustedStakes {
		totalStake += stake
	}

	// Ensure every loss array is non-empty and calculate geometric mean
	stakeWeightedLoss := make([]float64, len(reputersReportedLosses[0]))
	for j := 0; j < len(reputersReportedLosses[0]); j++ {
		logSum := 0.0
		for i, losses := range reputersReportedLosses {
			logSum += (math.Log10(losses[j]) * reputersAdjustedStakes[i]) / totalStake
		}
		stakeWeightedLoss[j] = logSum
	}

	return stakeWeightedLoss, nil
}

// GetConsensusScore calculates the proximity to consensus score for a reputer.
// T_im
func GetConsensusScore(reputerLosses, consensusLosses []float64) (float64, error) {
	fTolerance := 0.01
	if len(reputerLosses) != len(consensusLosses) {
		return 0, types.ErrInvalidSliceLength
	}

	var sumLogConsensusSquared float64
	for _, cLoss := range consensusLosses {
		sumLogConsensusSquared += math.Pow(cLoss, 2)
	}
	consensusNorm := math.Sqrt(sumLogConsensusSquared)

	var distanceSquared float64
	for i, rLoss := range reputerLosses {
		distanceSquared += math.Pow(math.Log10(rLoss/consensusLosses[i]), 2)
	}
	distance := math.Sqrt(distanceSquared)

	score := 1 / (distance/consensusNorm + fTolerance)
	return score, nil
}

// GetAllConsensusScores calculates the proximity to consensus score for all reputers.
// calculates:
// T_i - stake weighted total consensus
// returns:
// T_im - reputer score (proximity to consensus)
func GetAllConsensusScores(allLosses [][]float64, stakes []float64, allListeningCoefficients []float64, numReputers int) ([]float64, error) {
	// Get adjusted stakes
	var adjustedStakes []float64
	for i, reputerStake := range stakes {
		adjustedStake, err := GetAdjustedStake(reputerStake, stakes, allListeningCoefficients[i], allListeningCoefficients, float64(numReputers))
		if err != nil {
			return nil, err
		}
		adjustedStakes = append(adjustedStakes, adjustedStake)
	}

	// Get consensus loss vector
	consensus, err := GetStakeWeightedLossMatrix(adjustedStakes, allLosses)
	if err != nil {
		return nil, err
	}

	// Get reputers scores
	scores := make([]float64, numReputers)
	for i := 0; i < numReputers; i++ {
		losses := allLosses[i]
		scores[i], err = GetConsensusScore(losses, consensus)
		if err != nil {
			return nil, err
		}
	}

	return scores, nil
}

// GetAllReputersOutput calculates the final scores and adjusted listening coefficients for all reputers.
// This function iteratively adjusts the listening coefficients based on a gradient descent method to minimize
// the difference between each reputer's losses and the consensus losses, taking into account each reputer's stake.
// returns:
// T_im - reputer score (proximity to consensus)
// a_im - listening coefficients
func GetAllReputersOutput(allLosses [][]float64, stakes []float64, initialCoefficients []float64, numReputers int) ([]float64, []float64, error) {
	learningRate := types.DefaultParamsLearningRate()
	coefficients := make([]float64, len(initialCoefficients))
	copy(coefficients, initialCoefficients)

	oldCoefficients := make([]float64, numReputers)
	maxGradientThreshold := 0.001
	imax := int(math.Round(1.0 / learningRate))
	minStakeFraction := 0.5
	var i int
	var maxGradient float64 = 1
	finalScores := make([]float64, numReputers)

	for maxGradient > maxGradientThreshold && i < imax {
		i++
		copy(oldCoefficients, coefficients)
		gradient := make([]float64, numReputers)
		newScores := make([]float64, numReputers)

		for l := range coefficients {
			dcoeff := 0.001
			if coefficients[l] == 1 {
				dcoeff = -0.001
			}
			coeffs := make([]float64, len(coefficients))
			copy(coeffs, coefficients)

			scores, err := GetAllConsensusScores(allLosses, stakes, coeffs, numReputers)
			if err != nil {
				return nil, nil, err
			}
			coeffs2 := make([]float64, len(coeffs))
			copy(coeffs2, coeffs)
			coeffs2[l] += dcoeff

			scores2, err := GetAllConsensusScores(allLosses, stakes, coeffs2, numReputers)
			if err != nil {
				return nil, nil, err
			}
			gradient[l] = (1.0 - sum(scores)/sum(scores2)) / dcoeff
			copy(newScores, scores)
		}

		newCoefficients := make([]float64, len(coefficients))
		for j := range coefficients {
			newCoefficients[j] = math.Min(math.Max(coefficients[j]+learningRate*gradient[j], 0), 1)
		}

		listenedStakeFractionOld := sumWeighted(oldCoefficients, stakes) / sum(stakes)
		listenedStakeFraction := sumWeighted(newCoefficients, stakes) / sum(stakes)
		if listenedStakeFraction < minStakeFraction {
			for l := range coefficients {
				coefficients[l] = oldCoefficients[l] + (coefficients[l]-oldCoefficients[l])*(minStakeFraction-listenedStakeFractionOld)/(listenedStakeFraction-listenedStakeFractionOld)
			}
		} else {
			coefficients = newCoefficients
		}
		maxGradient = maxAbsDifference(coefficients, oldCoefficients) / learningRate

		copy(finalScores, newScores)
	}

	return finalScores, coefficients, nil
}

func sum(slice []float64) float64 {
	total := 0.0
	for _, v := range slice {
		total += v
	}
	return total
}

// sumWeighted calculates the weighted sum of values based on the given weights.
// The length of weights and values must be the same.
func sumWeighted(weights, values []float64) float64 {
	var sum float64
	for i, weight := range weights {
		sum += weight * values[i]
	}

	return sum
}

func maxAbsDifference(a, b []float64) float64 {
	maxDiff := 0.0
	for i := range a {
		diff := math.Abs(a[i] - b[i])
		if diff > maxDiff {
			maxDiff = diff
		}
	}
	return maxDiff
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
		return 0, types.ErrPhiInvalidInput
	}
	eToTheX := math.Exp(x)
	onePlusEToTheX := 1 + eToTheX
	if math.IsInf(onePlusEToTheX, 0) {
		return 0, types.ErrEToTheXExponentiationIsInfinity
	}
	naturalLog := math.Log(onePlusEToTheX)
	result := math.Pow(naturalLog, p)
	if math.IsInf(result, 0) {
		return 0, types.ErrLnToThePExponentiationIsInfinity
	}
	// should theoretically never be possible with the above checks
	if math.IsNaN(result) {
		return 0, types.ErrPhiResultIsNaN
	}
	return result, nil
}

// Adjusted stake for calculating consensus S hat
// ^S_im = 1 - ϕ_1^−1(η) * ϕ1[ −η * (((N_r * a_im * S_im) / (Σ_m(a_im * S_im))) − 1 )]
// we use eta = 20 as the fiducial value decided in the paper
// phi_1 refers to the phi function with p = 1
// INPUTS:
// This function expects that allStakes (S_im)
// and allListeningCoefficients are slices of the same length (a_im)
// and the index to each slice corresponds to the same reputer
func GetAdjustedStake(
	stake float64,
	allStakes []float64,
	listeningCoefficient float64,
	allListeningCoefficients []float64,
	numReputers float64,
) (float64, error) {
	if len(allStakes) != len(allListeningCoefficients) ||
		len(allStakes) == 0 ||
		len(allListeningCoefficients) == 0 {
		return 0, types.ErrAdjustedStakeInvalidSliceLength
	}
	eta := types.DefaultParamsSharpness()
	denominator := sumWeighted(allListeningCoefficients, allStakes)
	numerator := numReputers * listeningCoefficient * stake
	stakeFraction := numerator / denominator
	stakeFraction = stakeFraction - 1
	stakeFraction = stakeFraction * -eta

	phi_1_stakeFraction, err := phi(1, stakeFraction)
	if err != nil {
		return 0, err
	}
	phi_1_Eta, err := phi(1, eta)
	if err != nil {
		return 0, err
	}
	if phi_1_Eta == 0 {
		return 0, types.ErrPhiCannotBeZero
	}
	// phi_1_Eta is taken to the -1 power
	// and then multiplied by phi_1_stakeFraction
	// so we can just treat it as phi_1_stakeFraction / phi_1_Eta
	phiVal := phi_1_stakeFraction / phi_1_Eta
	ret := 1 - phiVal

	if math.IsInf(ret, 0) {
		return 0, errors.Wrapf(types.ErrAdjustedStakeIsInfinity, "stake: %f", stake)
	}
	if math.IsNaN(ret) {
		return 0, errors.Wrapf(types.ErrAdjustedStakeIsNaN, "stake: %f", stake)
	}
	return ret, nil
}
