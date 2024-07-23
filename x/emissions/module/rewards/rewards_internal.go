package rewards

import (
	"sort"

	"cosmossdk.io/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// flatten converts a double slice of alloraMath.Dec to a single slice of alloraMath.Dec
func flatten(arr [][]alloraMath.Dec) []alloraMath.Dec {
	var flat []alloraMath.Dec
	for _, row := range arr {
		flat = append(flat, row...)
	}
	return flat
}

// RewardFractions without multiplication against total rewards are used to calculate entropy
// note the use of lowercase u as opposed to capital
// u_ij = M(Tij) / ∑_j M(T_ij)
// v_ik = M(Tik) / ∑_k M(T_ik)
func GetScoreFractions(
	latestWorkerScores []alloraMath.Dec,
	latestTimeStepsScores []alloraMath.Dec,
	pReward alloraMath.Dec,
	cReward alloraMath.Dec,
	epsilon alloraMath.Dec,
) ([]alloraMath.Dec, error) {
	mappedValues, err := GetMappingFunctionValues(latestWorkerScores, latestTimeStepsScores, pReward, cReward, epsilon)
	if err != nil {
		return nil, errors.Wrapf(err, "error in GetMappingFunctionValue")
	}
	ret := make([]alloraMath.Dec, len(mappedValues))
	mappedSum, err := alloraMath.SumDecSlice(mappedValues)
	if err != nil {
		return nil, errors.Wrapf(err, "error in SumDecSlice:")
	}
	for i, mappedValue := range mappedValues {
		ret[i], err = mappedValue.Quo(mappedSum)
		if err != nil {
			return nil, errors.Wrapf(err, "error doing division of mappedValues")
		}
	}
	return ret, nil
}

// Mapping function used by score fraction calculation
// M(T) = φ_p (T / (σ(T) + ɛ))
// phi is the phi function
// sigma is NOT the sigma function but rather represents standard deviation
func GetMappingFunctionValues(
	latestWorkerScores []alloraMath.Dec, // T - latest scores from workers
	latestTimeStepsScores []alloraMath.Dec, // σ(T) - scores for stdDev (from multiple workers/time steps)
	pReward alloraMath.Dec, // p
	cReward alloraMath.Dec, // c
	epsilon alloraMath.Dec, // ɛ
) ([]alloraMath.Dec, error) {
	stdDev := alloraMath.ZeroDec()

	var err error
	if len(latestTimeStepsScores) > 1 {
		stdDev, err = alloraMath.StdDev(latestTimeStepsScores)
		if err != nil {
			return nil, errors.Wrapf(err, "err getting stdDev")
		}
		stdDev = stdDev.Abs()
	}

	ret := make([]alloraMath.Dec, len(latestWorkerScores))
	for i, score := range latestWorkerScores {
		if stdDev.Lt(epsilon) {
			// if standard deviation is smaller than epsilon, or zero,
			// then all scores are the very close to the same and losses are the same
			// therefore everyone should be paid the same, so we
			// return the plain value 1 for everybody
			ret[i] = alloraMath.OneDec()
		} else {
			stdDevPlusEpsilon, err := stdDev.Add(epsilon)
			if err != nil {
				return nil, errors.Wrapf(err, "err adding epsilon to stdDev")
			}
			scoreDividedByStdDevPlusEpsilon, err := score.Quo(stdDevPlusEpsilon)
			if err != nil {
				return nil, errors.Wrapf(err, "err dividing score by stdDevPlusEpsilon")
			}
			ret[i], err = alloraMath.Phi(pReward, cReward, scoreDividedByStdDevPlusEpsilon)
			if err != nil {
				return nil, errors.Wrapf(err, "err calculating phi")
			}
		}
	}
	return ret, nil
}

// CalculateReputerRewardFractions calculates the reward fractions for each reputer based on their stakes, scores, and preward parameter.
// W_im
func CalculateReputerRewardFractions(
	stakes []alloraMath.Dec,
	scores []alloraMath.Dec,
	preward alloraMath.Dec,
) ([]alloraMath.Dec, error) {
	if len(stakes) != len(scores) {
		return nil, types.ErrInvalidSliceLength
	}

	var err error
	// Calculate (stakes * scores)^preward and sum of all fractions
	var totalFraction alloraMath.Dec
	fractions := make([]alloraMath.Dec, len(stakes))
	for i, stake := range stakes {
		stakeTimesScores, err := stake.Mul(scores[i])
		if err != nil {
			return []alloraMath.Dec{}, err
		}
		fractions[i], err = alloraMath.Pow(stakeTimesScores, preward)
		if err != nil {
			return []alloraMath.Dec{}, err
		}
		totalFraction, err = totalFraction.Add(fractions[i])
		if err != nil {
			return []alloraMath.Dec{}, err
		}
	}

	// Normalize fractions
	for i := range fractions {
		if fractions[i].IsZero() {
			continue
		}
		fractions[i], err = fractions[i].Quo(totalFraction)
		if err != nil {
			return []alloraMath.Dec{}, err
		}
	}

	return fractions, nil
}

// GetfUniqueAgg calculates the unique value or impact of each forecaster.
// ƒ^+
func GetfUniqueAgg(numForecasters alloraMath.Dec) (alloraMath.Dec, error) {
	numForecastersMinusOne, err := numForecasters.Sub(alloraMath.OneDec())
	if err != nil {
		return alloraMath.Dec{}, err
	}
	twoToTheNumForecastersMinusOne, err := alloraMath.Pow(
		alloraMath.NewDecFromInt64(2),
		numForecastersMinusOne,
	)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	ret, err := alloraMath.OneDec().Quo(twoToTheNumForecastersMinusOne)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return ret, nil
}

// GetFinalWorkerScoreForecastTask calculates the worker score in forecast task.
// T_ik
func GetFinalWorkerScoreForecastTask(
	scoreOneIn,
	scoreOneOut,
	fUniqueAgg alloraMath.Dec,
) (alloraMath.Dec, error) {
	scoreInUnique, err := fUniqueAgg.Mul(scoreOneIn)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	oneMinusUnique, err := alloraMath.OneDec().Sub(fUniqueAgg)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	scoreOutUnique, err := oneMinusUnique.Mul(scoreOneOut)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	ret, err := scoreInUnique.Add(scoreOutUnique)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return ret, nil
}

// GetStakeWeightedLoss calculates the stake-weighted average loss.
// Consider the losses and the stake of each reputer to calculate the stake-weighted loss.
// The stake weighted loss is used to calculate the network-wide losses.
// L_i / L_ij / L_ik / L^-_i / L^-_il / L^+_ik
func GetStakeWeightedLoss(reputersStakes, reputersReportedLosses []alloraMath.Dec) (alloraMath.Dec, error) {
	if len(reputersStakes) != len(reputersReportedLosses) {
		return alloraMath.ZeroDec(), types.ErrInvalidSliceLength
	}
	var err error = nil

	totalStake := alloraMath.ZeroDec()
	for _, stake := range reputersStakes {
		totalStake, err = totalStake.Add(stake)
		if err != nil {
			return alloraMath.ZeroDec(), err
		}
	}

	stakeWeightedLoss := alloraMath.ZeroDec()
	for i, loss := range reputersReportedLosses {
		reputerStakesByLoss, err := reputersStakes[i].Mul(loss)
		if err != nil {
			return alloraMath.ZeroDec(), err
		}
		weightedLoss, err := reputerStakesByLoss.Quo(totalStake)
		if err != nil {
			return alloraMath.ZeroDec(), err
		}
		stakeWeightedLoss, err = stakeWeightedLoss.Add(weightedLoss)
		if err != nil {
			return alloraMath.ZeroDec(), err
		}
	}
	return stakeWeightedLoss, nil
}

// GetStakeWeightedLossMatrix calculates the stake-weighted
// geometric mean of the losses to generate the consensus vector.
// L_i - consensus loss vector
func GetStakeWeightedLossMatrix(
	reputersAdjustedStakes []alloraMath.Dec,
	reputersReportedLosses [][]alloraMath.Dec,
) ([]alloraMath.Dec, []alloraMath.Dec, error) {
	if len(reputersAdjustedStakes) == 0 || len(reputersReportedLosses) == 0 {
		return nil, nil, types.ErrInvalidSliceLength
	}
	var err error = nil

	// Ensure every loss array is non-empty and calculate geometric mean
	stakeWeightedLoss := make([]alloraMath.Dec, len(reputersReportedLosses[0]))
	mostDistantValues := make([]alloraMath.Dec, len(reputersReportedLosses[0]))
	for j := 0; j < len(reputersReportedLosses[0]); j++ {
		// Calculate total stake to consider
		// Skip stakes of reputers with NaN losses
		totalStakeToConsider := alloraMath.ZeroDec()
		for i, losses := range reputersReportedLosses {
			// Skip if loss is NaN
			if losses[j].IsNaN() {
				continue
			}

			totalStakeToConsider, err = totalStakeToConsider.Add(reputersAdjustedStakes[i])
			if err != nil {
				return nil, nil, err
			}
		}

		sum := alloraMath.ZeroDec()
		for i, losses := range reputersReportedLosses {
			// Skip if loss is NaN
			if losses[j].IsNaN() {
				continue
			}

			lossesTimesStake, err := losses[j].Mul(reputersAdjustedStakes[i])
			if err != nil {
				return nil, nil, err
			}
			lossesTimesStakeOverTotalStake, err := lossesTimesStake.Quo(totalStakeToConsider)
			if err != nil {
				return nil, nil, err
			}
			sum, err = sum.Add(lossesTimesStakeOverTotalStake)
			if err != nil {
				return nil, nil, err
			}
		}
		stakeWeightedLoss[j] = sum

		// Find most distant value from consensus value
		maxDistance, err := alloraMath.OneDec().Mul(alloraMath.MustNewDecFromString("-1")) // Initialize with an impossible value
		if err != nil {
			return nil, nil, err
		}
		for _, losses := range reputersReportedLosses {
			// Skip if loss is NaN
			if losses[j].IsNaN() {
				continue
			}

			distance, err := sum.Sub(losses[j])
			if err != nil {
				return nil, nil, err
			}
			if distance.Gt(maxDistance) {
				maxDistance = distance
				mostDistantValues[j] = losses[j]
			}
		}
	}

	return stakeWeightedLoss, mostDistantValues, nil
}

// GetConsensusScore calculates the proximity to consensus score for a reputer.
// T_im
func GetConsensusScore(
	reputerLosses,
	consensusLosses,
	mostDistantValues []alloraMath.Dec,
	epsilonReputer alloraMath.Dec,
	epsilon alloraMath.Dec,
) (alloraMath.Dec, error) {
	if len(reputerLosses) != len(consensusLosses) {
		return alloraMath.ZeroDec(), types.ErrInvalidSliceLength
	}

	var err error = nil
	var sumConsensusSquared alloraMath.Dec = alloraMath.ZeroDec()
	for _, cLoss := range consensusLosses {
		cLossSquared, err := cLoss.Mul(cLoss)
		if err != nil {
			return alloraMath.ZeroDec(), err
		}
		sumConsensusSquared, err = sumConsensusSquared.Add(cLossSquared)
		if err != nil {
			return alloraMath.ZeroDec(), err
		}
	}
	consensusNorm, err := sumConsensusSquared.Sqrt()
	if err != nil {
		return alloraMath.ZeroDec(), err
	}

	var distanceSquared alloraMath.Dec
	for i, rLoss := range reputerLosses {
		// Attribute most distant value if loss is NaN
		if rLoss.IsNaN() {
			rLoss = mostDistantValues[i]
		}
		if rLoss.IsZero() {
			rLoss = epsilon
		}
		if consensusLosses[i].IsZero() {
			consensusLosses[i] = epsilon
		}
		// We have the log losses and the identity: log(Loss_im / Loss_i) = log(Loss_im) - log(Loss_i)
		rLossLessConsensusLoss, err := rLoss.Sub(consensusLosses[i])
		if err != nil {
			return alloraMath.ZeroDec(), err
		}
		rLossLessCLossSquared, err := rLossLessConsensusLoss.Mul(rLossLessConsensusLoss) // == Pow(x,2)
		if err != nil {
			return alloraMath.ZeroDec(), err
		}
		distanceSquared, err = distanceSquared.Add(rLossLessCLossSquared)
		if err != nil {
			return alloraMath.ZeroDec(), err
		}
	}
	distance, err := distanceSquared.Sqrt()
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	distanceOverConsensusNorm, err := distance.Quo(consensusNorm)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	distanceOverConsensusNormPlusEpsilonReputer, err := distanceOverConsensusNorm.Add(epsilonReputer)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	score, err := alloraMath.OneDec().Quo(distanceOverConsensusNormPlusEpsilonReputer)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	return score, nil
}

// GetAllConsensusScores calculates the proximity to consensus score for all reputers.
// calculates:
// T_i - stake weighted total consensus
// returns:
// T_im - reputer score (proximity to consensus)
func GetAllConsensusScores(
	allLosses [][]alloraMath.Dec,
	stakes []alloraMath.Dec,
	allListeningCoefficients []alloraMath.Dec,
	numReputers int64,
	epsilonReputer alloraMath.Dec,
	epsilon alloraMath.Dec,
) ([]alloraMath.Dec, error) {
	// Get adjusted stakes
	var adjustedStakes []alloraMath.Dec
	for i, reputerStake := range stakes {
		adjustedStake, err := GetAdjustedStake(
			reputerStake,
			stakes,
			allListeningCoefficients[i],
			allListeningCoefficients,
			alloraMath.NewDecFromInt64(numReputers),
		)
		if err != nil {
			return nil, errors.Wrapf(err, "error in GetAdjustedStake")
		}
		adjustedStakes = append(adjustedStakes, adjustedStake)
	}

	// Get consensus loss vector and retrieve most distant values from
	consensus, mostDistantValues, err := GetStakeWeightedLossMatrix(adjustedStakes, allLosses)
	if err != nil {
		return nil, errors.Wrapf(err, "error in GetStakeWeightedLossMatrix")
	}

	// Get reputers scores
	scores := make([]alloraMath.Dec, numReputers)
	for i := int64(0); i < numReputers; i++ {
		losses := allLosses[i]
		scores[i], err = GetConsensusScore(losses, consensus, mostDistantValues, epsilonReputer, epsilon)
		if err != nil {
			return nil, errors.Wrapf(err, "error in GetConsensusScore")
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
func GetAllReputersOutput(
	allLosses [][]alloraMath.Dec,
	stakes []alloraMath.Dec,
	initialCoefficients []alloraMath.Dec,
	numReputers int64,
	learningRate alloraMath.Dec,
	gradientDescentMaxIters uint64,
	epsilonReputer alloraMath.Dec,
	epsilon alloraMath.Dec,
	minStakeFraction alloraMath.Dec,
	maxGradientThreshold alloraMath.Dec,
) ([]alloraMath.Dec, []alloraMath.Dec, error) {
	coefficients := make([]alloraMath.Dec, len(initialCoefficients))
	copy(coefficients, initialCoefficients)

	oldCoefficients := make([]alloraMath.Dec, numReputers)
	var i uint64 = 0
	var maxGradient alloraMath.Dec = alloraMath.OneDec()
	// finalScores := make([]alloraMath.Dec, numReputers)
	newScores := make([]alloraMath.Dec, numReputers)

	for maxGradient.Gte(maxGradientThreshold) && i < gradientDescentMaxIters {
		copy(oldCoefficients, coefficients)
		gradient := make([]alloraMath.Dec, numReputers)

		for l := range coefficients {
			dcoeff := alloraMath.MustNewDecFromString("0.001")
			if coefficients[l].Equal(alloraMath.OneDec()) {
				dcoeff = alloraMath.MustNewDecFromString("-0.001")
			}
			coeffs := make([]alloraMath.Dec, len(coefficients))
			copy(coeffs, coefficients)

			scores, err := GetAllConsensusScores(allLosses, stakes, coeffs, numReputers, epsilonReputer, epsilon)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "error in GetAllConsensusScores")
			}
			coeffs2 := make([]alloraMath.Dec, len(coeffs))
			copy(coeffs2, coeffs)
			coeffs2[l], err = coeffs2[l].Add(dcoeff)
			if err != nil {
				return nil, nil, err
			}

			scores2, err := GetAllConsensusScores(allLosses, stakes, coeffs2, numReputers, epsilonReputer, epsilon)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "error in GetAllConsensusScores")
			}
			weightedSumScores, err := sumWeighted(scores, stakes)
			if err != nil {
				return nil, nil, err
			}
			weightedSumScores2, err := sumWeighted(scores2, stakes)
			if err != nil {
				return nil, nil, err
			}
			sumScoresOverSumScores2, err := weightedSumScores.Quo(weightedSumScores2)
			if err != nil {
				return nil, nil, err
			}
			oneMinusSumScoresOverSumScores2, err := alloraMath.OneDec().Sub(sumScoresOverSumScores2)
			if err != nil {
				return nil, nil, err
			}
			gradient[l], err = oneMinusSumScoresOverSumScores2.Quo(dcoeff)
			if err != nil {
				return nil, nil, err
			}
			copy(newScores, scores)
		}

		newCoefficients := make([]alloraMath.Dec, len(coefficients))
		for j := range coefficients {
			learningRateTimesGradient, err := learningRate.Mul(gradient[j])
			if err != nil {
				return nil, nil, err
			}
			coefficientsPlusLearningRateTimesGradient, err := coefficients[j].Add(learningRateTimesGradient)
			if err != nil {
				return nil, nil, err
			}
			newCoefficients[j] = alloraMath.Min(
				alloraMath.Max(
					coefficientsPlusLearningRateTimesGradient,
					alloraMath.ZeroDec(),
				),
				alloraMath.OneDec(),
			)
		}

		sumStakes, err := alloraMath.SumDecSlice(stakes)
		if err != nil {
			return nil, nil, err
		}
		oldWeighted, err := sumWeighted(oldCoefficients, stakes)
		if err != nil {
			return nil, nil, err
		}
		listenedStakeFractionOld, err := oldWeighted.Quo(sumStakes)
		if err != nil {
			return nil, nil, err
		}
		newWeighted, err := sumWeighted(newCoefficients, stakes)
		if err != nil {
			return nil, nil, err
		}
		listenedStakeFraction, err := newWeighted.Quo(sumStakes)
		if err != nil {
			return nil, nil, err
		}
		if listenedStakeFraction.Lt(minStakeFraction) {
			for l := range coefficients {
				coeffDiff, err := coefficients[l].Sub(oldCoefficients[l])
				if err != nil {
					return nil, nil, err
				}
				listenedDiff, err := minStakeFraction.Sub(listenedStakeFractionOld)
				if err != nil {
					return nil, nil, err
				}
				stakedFracDiff, err := listenedStakeFraction.Sub(listenedStakeFractionOld)
				if err != nil {
					return nil, nil, err
				}
				if stakedFracDiff.IsZero() {
					i = gradientDescentMaxIters
				} else {
					coeffDiffTimesListenedDiff, err := coeffDiff.Mul(listenedDiff)
					if err != nil {
						return nil, nil, err
					}
					coefDiffTimesListenedDiffOverStakedFracDiff, err := coeffDiffTimesListenedDiff.Quo(stakedFracDiff)
					if err != nil {
						return nil, nil, err
					}
					coefficients[l], err = oldCoefficients[l].Add(coefDiffTimesListenedDiffOverStakedFracDiff)
					if err != nil {
						return nil, nil, err
					}
				}
			}
		} else {
			coefficients = newCoefficients
		}
		maxAbsDiffCoeff, err := maxAbsDifference(coefficients, oldCoefficients)
		if err != nil {
			return nil, nil, err
		}
		maxGradient, err = maxAbsDiffCoeff.Quo(learningRate)
		if err != nil {
			return nil, nil, err
		}

		i++
	}

	return newScores, coefficients, nil
}

// sumWeighted calculates the weighted sum of values based on the given weights.
// The length of weights and values must be the same.
func sumWeighted(weights, values []alloraMath.Dec) (alloraMath.Dec, error) {
	var sum alloraMath.Dec
	for i, weight := range weights {
		var err error = nil
		weightTimesValue, err := weight.Mul(values[i])
		if err != nil {
			return alloraMath.Dec{}, err
		}
		sum, err = sum.Add(weightTimesValue)
		if err != nil {
			return alloraMath.Dec{}, err
		}
	}
	return sum, nil
}

// maxAbsDifference calculates the maximum absolute difference value
// between every pair of values in two slices of alloraMath.Dec.
// it assumes that a and b are of the same length
func maxAbsDifference(a, b []alloraMath.Dec) (alloraMath.Dec, error) {
	maxDiff := alloraMath.ZeroDec()
	for i := range a {
		subtraction, err := a[i].Sub(b[i])
		if err != nil {
			return alloraMath.Dec{}, err
		}
		diff := subtraction.Abs()
		if diff.Gt(maxDiff) {
			maxDiff = diff
		}
	}
	return maxDiff, nil
}

// Adjusted stake for calculating consensus S hat
// ^S_im = min((N_r * a_im * S_im)/(Σ_m(a_im * S_im)), 1)
// INPUTS:
// This function expects that allStakes (S_im)
// and allListeningCoefficients are slices of the same length (a_im)
// and the index to each slice corresponds to the same reputer
func GetAdjustedStake(
	stake alloraMath.Dec,
	allStakes []alloraMath.Dec,
	listeningCoefficient alloraMath.Dec,
	allListeningCoefficients []alloraMath.Dec,
	numReputers alloraMath.Dec,
) (alloraMath.Dec, error) {
	if len(allStakes) != len(allListeningCoefficients) ||
		len(allStakes) == 0 ||
		len(allListeningCoefficients) == 0 {
		return alloraMath.ZeroDec(), types.ErrAdjustedStakeInvalidSliceLength
	}
	denominator, err := sumWeighted(allListeningCoefficients, allStakes)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	numReputersTimesListeningCoefficent, err := numReputers.Mul(listeningCoefficient)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	numerator, err := numReputersTimesListeningCoefficent.Mul(stake)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	stakeFraction, err := numerator.Quo(denominator)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	ret := alloraMath.Min(stakeFraction, alloraMath.OneDec())
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
// f_im =  (̃Wim) / ∑_m(̃Wim)
func ModifiedRewardFractions(rewardFractions []alloraMath.Dec) ([]alloraMath.Dec, error) {
	sumValues, err := alloraMath.SumDecSlice(rewardFractions)
	if err != nil {
		return nil, err
	}
	ret := make([]alloraMath.Dec, len(rewardFractions))
	for i, value := range rewardFractions {
		ret[i], err = value.Quo(sumValues)
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

// We define a modified entropy for each class
// ({F_i, G_i, H_i} for the inference, forecasting, and reputer tasks, respectively
// Fi = - ∑_j( f_ij * ln(f_ij) * (N_{i,eff} / N_i)^β )
// Gi = - ∑_k( f_ik * ln(f_ik) * (N_{f,eff} / N_f)^β )
// Hi = - ∑_m( f_im * ln(f_im) * (N_{r,eff} / N_r)^β )
// we use beta = 0.25 as a fiducial value
func Entropy(
	rewardFractionsPerActor []alloraMath.Dec, // an array of every f_{ij}, f_{ik}, or f_{im}
	numberRatio alloraMath.Dec, // N_{i, eff}, N_{f,eff} or N_{r,eff}
	numParticipants alloraMath.Dec, // N_i
	beta alloraMath.Dec, // β
) (alloraMath.Dec, error) {
	multiplier, err := numberRatio.Quo(numParticipants)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	multiplier, err = alloraMath.Pow(multiplier, beta)
	if err != nil {
		return alloraMath.Dec{}, err
	}

	sum := alloraMath.ZeroDec()
	for _, f := range rewardFractionsPerActor {
		lnF, err := alloraMath.Ln(f)
		if err != nil {
			return alloraMath.Dec{}, err
		}
		fLnF, err := f.Mul(lnF)
		if err != nil {
			return alloraMath.Dec{}, err
		}
		sum, err = sum.Add(fLnF)
		if err != nil {
			return alloraMath.Dec{}, err
		}
	}

	inverseSum, err := sum.Mul(alloraMath.NewDecFromInt64(-1))
	if err != nil {
		return alloraMath.Dec{}, err
	}
	ret, err := inverseSum.Mul(multiplier)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return ret, nil
}

// If there's only one worker, entropy should be the default number of 0.173286795139986
func EntropyForSingleParticipant() (alloraMath.Dec, error) {
	return alloraMath.MustNewDecFromString("0.173286795139986"), nil
}

// The number ratio term captures the number of participants in the network
// to prevent sybil attacks in the rewards distribution
// This function captures
// N_{i,eff} = 1 / ∑_j( f_ij^2 )
// N_{f,eff} = 1 / ∑_k( f_ik^2 )
// N_{r,eff} = 1 / ∑_m( f_im^2 )
func NumberRatio(rewardFractions []alloraMath.Dec) (alloraMath.Dec, error) {
	if len(rewardFractions) == 0 {
		return alloraMath.Dec{}, types.ErrNumberRatioInvalidSliceLength
	}
	sum := alloraMath.ZeroDec()
	for _, f := range rewardFractions {
		fSquared, err := f.Mul(f)
		if err != nil {
			return alloraMath.Dec{}, err
		}
		sum, err = sum.Add(fSquared)
		if err != nil {
			return alloraMath.Dec{}, err
		}
	}
	if sum.Equal(alloraMath.ZeroDec()) {
		return alloraMath.Dec{}, types.ErrNumberRatioDivideByZero
	}
	ret, err := alloraMath.OneDec().Quo(sum)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return ret, nil
}

// sigmoid function
// σ(x) = 1/(1+e^{-x}) = e^x/(1+e^x)
func Sigmoid(x alloraMath.Dec) (alloraMath.Dec, error) {
	expX, err := alloraMath.Exp(x)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	onePlusExpX, err := alloraMath.OneDec().Add(expX)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	ret, nil := expX.Quo(onePlusExpX)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return ret, nil
}

// ExtractValues extracts all alloraMath.Dec values from a ValueBundle.
func ExtractValues(bundle *types.ValueBundle) []alloraMath.Dec {
	var values []alloraMath.Dec

	// Extract direct alloraMath.Dec values
	values = append(values, bundle.CombinedValue, bundle.NaiveValue)

	// Sort and Extract values from slices of ValueBundle
	sort.Slice(bundle.InfererValues, func(i, j int) bool {
		return bundle.InfererValues[i].Worker < bundle.InfererValues[j].Worker
	})
	for _, v := range bundle.InfererValues {
		values = append(values, v.Value)
	}
	sort.Slice(bundle.ForecasterValues, func(i, j int) bool {
		return bundle.ForecasterValues[i].Worker < bundle.ForecasterValues[j].Worker
	})
	for _, v := range bundle.ForecasterValues {
		values = append(values, v.Value)
	}
	sort.Slice(bundle.OneOutInfererValues, func(i, j int) bool {
		return bundle.OneOutInfererValues[i].Worker < bundle.OneOutInfererValues[j].Worker
	})
	for _, v := range bundle.OneOutInfererValues {
		values = append(values, v.Value)
	}
	sort.Slice(bundle.OneOutForecasterValues, func(i, j int) bool {
		return bundle.OneOutForecasterValues[i].Worker < bundle.OneOutForecasterValues[j].Worker
	})
	for _, v := range bundle.OneOutForecasterValues {
		values = append(values, v.Value)
	}
	sort.Slice(bundle.OneInForecasterValues, func(i, j int) bool {
		return bundle.OneInForecasterValues[i].Worker < bundle.OneInForecasterValues[j].Worker
	})
	for _, v := range bundle.OneInForecasterValues {
		values = append(values, v.Value)
	}

	return values
}
