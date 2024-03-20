package module

import (
	"fmt"
	"math"
)

// GetfUniqueAgg calculates the unique value or impact of each forecaster.
// f^+
func GetfUniqueAgg(numForecasters float64) float64 {
	return  1.0 / math.Pow(2.0, (numForecasters - 1.0))
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
		weightedLoss := (reputersStakes[i] * math.Log10(loss)) / totalStake
		stakeWeightedLoss += weightedLoss
	}

	return stakeWeightedLoss, nil
}
