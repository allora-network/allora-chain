package module

import (
	"math"

	errors "cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// SmoothAbsoluteX calculates the smooth absolute function.
func SmoothAbsoluteX(x []float64, p float64) ([]float64, error) {
	result := make([]float64, len(x))
	for i, v := range x {
		expV := math.Exp(v)
		logExpV := math.Log(1 + expV)
		result[i] = math.Pow(logExpV, p)
	}
	return result, nil
}

// StdDev calculates the standard deviation of a slice of float64.
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

	smoothedScores, err := SmoothAbsoluteX(normalizedScores, preward)
	if err != nil {
		return nil, err
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
		sumLogConsensusSquared += math.Pow(cLoss, 2) //!
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
func GetAllReputersOutput(allLosses [][]float64, stakes []float64, initialCoefficients []float64, numReputers int, enableListening bool) ([]float64, []float64, error) {
	learningRate := 0.01
	coefficients := make([]float64, len(initialCoefficients))
	copy(coefficients, initialCoefficients)

	if enableListening {
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
	} else {
		coefficients = initialCoefficients
		scores, err := GetAllConsensusScores(allLosses, stakes, coefficients, numReputers)
		if err != nil {
			return nil, nil, err
		}
		return scores, coefficients, nil
	}
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
// This function expects that allStakes
// and allListeningCoefficients are slices of the same length
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
	// renaming variables just to be more legible with the formula
	S_im := stake
	a_im := listeningCoefficient
	N_r := numReputers

	denominator := 0.0
	for i, s := range allStakes {
		a := allListeningCoefficients[i]
		denominator += (a * s)
	}
	numerator := N_r * a_im * S_im
	stakeFraction := numerator / denominator
	stakeFraction = stakeFraction - 1
	stakeFraction = stakeFraction * -20 // eta = 20

	phi_1_stakeFraction, err := phi(1, stakeFraction)
	if err != nil {
		return 0, err
	}
	phi_1_Eta, err := phi(1, 20)
	if err != nil {
		return 0, err
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

// Used by Rewards fraction functions,
// all the exponential moving average functions take the form
// x_average=α*x_current + (1-α)*x_previous
//
// this covers the equations
// Uij = αUij + (1 − α)Ui−1,j
// ̃Vik = αVik + (1 − α)Vi−1,k
// ̃Wim = αWim + (1 − α)Wi−1,m
func exponentialMovingAverage(alpha float64, current float64, previous float64) (float64, error) {
	if math.IsNaN(alpha) || math.IsInf(alpha, 0) {
		return 0, errors.Wrapf(types.ErrExponentialMovingAverageInvalidInput, "alpha: %f", alpha)
	}
	if math.IsNaN(current) || math.IsInf(current, 0) {
		return 0, errors.Wrapf(types.ErrExponentialMovingAverageInvalidInput, "current: %f", current)
	}
	if math.IsNaN(previous) || math.IsInf(previous, 0) {
		return 0, errors.Wrapf(types.ErrExponentialMovingAverageInvalidInput, "previous: %f", previous)
	}

	// THE ONLY LINE OF CODE IN THIS FUNCTION
	// THAT ISN'T ERROR CHECKING IS HERE
	ret := alpha*current + (1-alpha)*previous

	if math.IsInf(ret, 0) {
		return 0, types.ErrExponentialMovingAverageIsInfinity
	}
	if math.IsNaN(ret) {
		return 0, types.ErrExponentialMovingAverageIsNaN
	}
	return ret, nil
}

// f_ij, f_ik, and f_im are all reward fractions
// that require computing the ratio of one participant to all participants
// yes this is extremely simple math
// yes we write a separate function for it anyway. The compiler can inline it if necessary
// normalizeToArray = value / sum(allValues)
// this covers equations
// f_ij =  (̃U_ij) / ∑_j(̃Uij)
// f_ik = (̃Vik) / ∑_k(̃Vik)
// fim =  (̃Wim) / ∑_m(̃Wim)
func normalizeAgainstSlice(value float64, allValues []float64) (float64, error) {
	if len(allValues) == 0 {
		return 0, types.ErrFractionInvalidSliceLength
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, errors.Wrapf(types.ErrFractionInvalidInput, "value: %f", value)
	}
	sumValues := 0.0
	for i, v := range allValues {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, errors.Wrapf(types.ErrFractionInvalidInput, "allValues[%d]: %f", i, v)
		}
		sumValues += v
	}
	if sumValues == 0 {
		return 0, types.ErrFractionDivideByZero
	}
	ret := value / sumValues
	if math.IsInf(ret, 0) {
		return 0, types.ErrFractionIsInfinity
	}
	if math.IsNaN(ret) {
		return 0, types.ErrFractionIsNaN
	}

	return ret, nil
}

// We define a modified entropy for each class
// ({F_i, G_i, H_i} for the inference, forecasting, and reputer tasks, respectively
// Fi = - ∑_j( f_ij * ln(f_ij) * (N_{i,eff} / N_i)^β )
// Gi = - ∑_k( f_ik * ln(f_ik) * (N_{f,eff} / N_f)^β )
// Hi = - ∑_m( f_im * ln(f_im) * (N_{r,eff} / N_r)^β )
// we use beta = 0.25 as a fiducial value
func entropy(allFs []float64, N_eff float64, numParticipants float64, beta float64) (float64, error) {
	if math.IsInf(N_eff, 0) ||
		math.IsNaN(N_eff) ||
		math.IsInf(numParticipants, 0) ||
		math.IsNaN(numParticipants) ||
		math.IsInf(beta, 0) ||
		math.IsNaN(beta) {
		return 0, errors.Wrapf(
			types.ErrEntropyInvalidInput,
			"N_eff: %f, numParticipants: %f, beta: %f",
			N_eff,
			numParticipants,
			beta,
		)
	}
	// simple variable rename to look more like the equations,
	// hopefully compiler is smart enough to inline it
	N := numParticipants

	multiplier := N_eff / N
	multiplier = math.Pow(multiplier, beta)

	sum := 0.0
	for i, f := range allFs {
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return 0, errors.Wrapf(types.ErrEntropyInvalidInput, "allFs[%d]: %f", i, f)
		}
		sum += f * math.Log(f)
	}

	ret := -1 * sum * multiplier
	if math.IsInf(ret, 0) {
		return 0, errors.Wrapf(
			types.ErrEntropyIsInfinity,
			"sum of f: %f, multiplier: %f",
			sum,
			multiplier,
		)
	}
	if math.IsNaN(ret) {
		return 0, errors.Wrapf(
			types.ErrEntropyIsNaN,
			"sum of f: %f, multiplier: %f",
			sum,
			multiplier,
		)
	}
	return ret, nil
}

// The number ratio term captures the number of participants in the network
// to prevent sybil attacks in the rewards distribution
// This function captures
// N_{i,eff} = 1 / ∑_j( f_ij^2 )
// N_{f,eff} = 1 / ∑_k( f_ik^2 )
// N_{r,eff} = 1 / ∑_m( f_im^2 )
func numberRatio(rewardFractions []float64) (float64, error) {
	if len(rewardFractions) == 0 {
		return 0, types.ErrNumberRatioInvalidSliceLength
	}
	sum := 0.0
	for i, f := range rewardFractions {
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, errors.Wrapf(types.ErrNumberRatioInvalidInput, "rewardFractions[%d]: %f", i, f)
		}
		sum += f * f
	}
	if sum == 0 {
		return 0, types.ErrNumberRatioDivideByZero
	}
	ret := 1 / sum
	if math.IsInf(ret, 0) {
		return 0, types.ErrNumberRatioIsInfinity
	}
	if math.IsNaN(ret) {
		return 0, types.ErrNumberRatioIsNaN
	}
	return ret, nil
}

// inference rewards calculation
// U_i = ((1 - χ) * γ * F_i * E_i ) / (F_i + G_i + H_i)
func inferenceRewards(
	chi float64,
	gamma float64,
	entropyInference float64,
	entropyForecasting float64,
	entropyReputer float64,
	timeStep float64,
) (float64, error) {
	if math.IsNaN(chi) || math.IsInf(chi, 0) ||
		math.IsNaN(gamma) || math.IsInf(gamma, 0) ||
		math.IsNaN(entropyInference) || math.IsInf(entropyInference, 0) ||
		math.IsNaN(entropyForecasting) || math.IsInf(entropyForecasting, 0) ||
		math.IsNaN(entropyReputer) || math.IsInf(entropyReputer, 0) ||
		math.IsNaN(timeStep) || math.IsInf(timeStep, 0) {
		return 0, errors.Wrapf(
			types.ErrInferenceRewardsInvalidInput,
			"chi: %f, gamma: %f, entropyInference: %f, entropyForecasting: %f, entropyReputer: %f, timeStep: %f",
			chi,
			gamma,
			entropyInference,
			entropyForecasting,
			entropyReputer,
			timeStep,
		)
	}
	ret := ((1 - chi) * gamma * entropyInference * timeStep) / (entropyInference + entropyForecasting + entropyReputer)
	if math.IsInf(ret, 0) {
		return 0, types.ErrInferenceRewardsIsInfinity
	}
	if math.IsNaN(ret) {
		return 0, types.ErrInferenceRewardsIsNaN
	}
	return ret, nil
}

// forecaster rewards calculation
// V_i = (χ * γ * G_i * E_i) / (F_i + G_i + H_i)
func forecastingRewards(
	chi float64,
	gamma float64,
	entropyInference float64,
	entropyForecasting float64,
	entropyReputer float64,
	timeStep float64,
) (float64, error) {
	if math.IsNaN(chi) || math.IsInf(chi, 0) ||
		math.IsNaN(gamma) || math.IsInf(gamma, 0) ||
		math.IsNaN(entropyInference) || math.IsInf(entropyInference, 0) ||
		math.IsNaN(entropyForecasting) || math.IsInf(entropyForecasting, 0) ||
		math.IsNaN(entropyReputer) || math.IsInf(entropyReputer, 0) ||
		math.IsNaN(timeStep) || math.IsInf(timeStep, 0) {
		return 0, errors.Wrapf(
			types.ErrForecastingRewardsInvalidInput,
			"chi: %f, gamma: %f, entropyInference: %f, entropyForecasting: %f, entropyReputer: %f, timeStep: %f",
			chi,
			gamma,
			entropyInference,
			entropyForecasting,
			entropyReputer,
			timeStep,
		)
	}
	ret := (chi * gamma * entropyForecasting * timeStep) / (entropyInference + entropyForecasting + entropyReputer)
	if math.IsInf(ret, 0) {
		return 0, types.ErrForecastingRewardsIsInfinity
	}
	if math.IsNaN(ret) {
		return 0, types.ErrForecastingRewardsIsNaN
	}
	return ret, nil
}

// reputer rewards calculation
// W_i = (H_i * E_i) / (F_i + G_i + H_i)
func reputerRewards(
	entropyInference float64,
	entropyForecasting float64,
	entropyReputer float64,
	timeStep float64,
) (float64, error) {
	if math.IsNaN(entropyInference) || math.IsInf(entropyInference, 0) ||
		math.IsNaN(entropyForecasting) || math.IsInf(entropyForecasting, 0) ||
		math.IsNaN(entropyReputer) || math.IsInf(entropyReputer, 0) ||
		math.IsNaN(timeStep) || math.IsInf(timeStep, 0) {
		return 0, errors.Wrapf(
			types.ErrReputerRewardsInvalidInput,
			"entropyInference: %f, entropyForecasting: %f, entropyReputer: %f, timeStep: %f",
			entropyInference,
			entropyForecasting,
			entropyReputer,
			timeStep,
		)
	}
	ret := (entropyReputer * timeStep) / (entropyInference + entropyForecasting + entropyReputer)
	if math.IsInf(ret, 0) {
		return 0, types.ErrReputerRewardsIsInfinity
	}
	if math.IsNaN(ret) {
		return 0, types.ErrReputerRewardsIsNaN
	}
	return ret, nil
}

// The performance score of the entire forecasting task T_i
// is positive if the removal of the forecasting task would
// increase the network loss, and is negative if its removal
// would decrease the network loss
// We subtract the log-loss of the complete network inference
// (L_i) from that of the naive network (L_i^-), which is
// obtained by omitting all forecast-implied inferences
// T_i = log L_i^- - log L_i
func forecastingPerformanceScore(
	naiveNetworkInferenceLoss float64,
	networkInferenceLoss float64,
) (float64, error) {
	if math.IsNaN(networkInferenceLoss) || math.IsInf(networkInferenceLoss, 0) ||
		math.IsNaN(naiveNetworkInferenceLoss) || math.IsInf(naiveNetworkInferenceLoss, 0) {
		return 0, errors.Wrapf(
			types.ErrForecastingPerformanceScoreInvalidInput,
			"networkInferenceLoss: %f, naiveNetworkInferenceLoss: %f",
			networkInferenceLoss,
			naiveNetworkInferenceLoss,
		)
	}
	ret := math.Log10(naiveNetworkInferenceLoss) - math.Log10(networkInferenceLoss)

	if math.IsInf(ret, 0) {
		return 0, types.ErrForecastingPerformanceScoreIsInfinity
	}
	if math.IsNaN(ret) {
		return 0, types.ErrForecastingPerformanceScoreIsNaN
	}
	return ret, nil
}

// sigmoid function
// σ(x) = 1/(1+e^{-x}) = e^x/(1+e^x)
func sigmoid(x float64) (float64, error) {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0, types.ErrSigmoidInvalidInput
	}
	ret := math.Exp(x) / (1 + math.Exp(x))
	if math.IsInf(ret, 0) {
		return 0, types.ErrSigmoidIsInfinity
	}
	if math.IsNaN(ret) {
		return 0, types.ErrSigmoidIsNaN
	}
	return ret, nil
}

// we apply a utility function to the forecasting performance score
// to let the forecasting task utility range from the interval [0.1, 0.5]
// χ = 0.1 + 0.4σ(a*T_i − b)
// sigma is the sigmoid function
// a has fiduciary value of 8
// b has fiduciary value of 0.5
func forecastingUtility(forecastingPerformanceScore float64, a float64, b float64) (float64, error) {
	if math.IsNaN(forecastingPerformanceScore) || math.IsInf(forecastingPerformanceScore, 0) ||
		math.IsNaN(a) || math.IsInf(a, 0) ||
		math.IsNaN(b) || math.IsInf(b, 0) {
		return 0, types.ErrForecastingUtilityInvalidInput
	}
	ret, err := sigmoid(a*forecastingPerformanceScore - b)
	if err != nil {
		return 0, err
	}
	ret = 0.1 + 0.4*ret
	if math.IsInf(ret, 0) {
		return 0, types.ErrForecastingUtilityIsInfinity
	}
	if math.IsNaN(ret) {
		return 0, types.ErrForecastingUtilityIsNaN
	}
	return ret, nil
}

// renormalize with a factor γ to ensure that the
// total reward allocated to workers (Ui + Vi)
// remains constant (otherwise, this would go at the expense of reputers)
// γ = (F_i + G_i) / ( (1 − χ)*F_i + χ*G_i)
func normalizationFactor(
	entropyInference float64,
	entropyForecasting float64,
	forecastingUtility float64,
) (float64, error) {
	if math.IsNaN(entropyInference) || math.IsInf(entropyInference, 0) ||
		math.IsNaN(entropyForecasting) || math.IsInf(entropyForecasting, 0) ||
		math.IsNaN(forecastingUtility) || math.IsInf(forecastingUtility, 0) {
		return 0, errors.Wrapf(
			types.ErrNormalizationFactorInvalidInput,
			"entropyInference: %f, entropyForecasting: %f, forecastingUtility: %f",
			entropyInference,
			entropyForecasting,
			forecastingUtility,
		)
	}
	numerator := entropyInference + entropyForecasting
	denominator := (1-forecastingUtility)*entropyInference + forecastingUtility*entropyForecasting
	ret := numerator / denominator
	if math.IsInf(ret, 0) {
		return 0, errors.Wrapf(
			types.ErrNormalizationFactorIsInfinity,
			"numerator: %f, denominator: %f entropyInference: %f, entropyForecasting: %f, forecastingUtility: %f",
			numerator,
			denominator,
			entropyInference,
			entropyForecasting,
			forecastingUtility,
		)
	}
	if math.IsNaN(ret) {
		return 0, errors.Wrapf(
			types.ErrNormalizationFactorIsNaN,
			"numerator: %f, denominator: %f entropyInference: %f, entropyForecasting: %f, forecastingUtility: %f",
			numerator,
			denominator,
			entropyInference,
			entropyForecasting,
			forecastingUtility,
		)
	}

	return ret, nil
}
