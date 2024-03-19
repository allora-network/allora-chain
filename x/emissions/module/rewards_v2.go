package module

import (
	"fmt"
	"math"
)

// GetWorkerScore calculates the worker score based on the losses and lossesCut.
func GetWorkerScore(losses, lossesCut float64) float64 {
	deltaLogLoss := math.Log10(lossesCut) - math.Log10(losses)
	return deltaLogLoss
}

// GetStakeWeightedLoss calculates the stake-weighted average loss.
// L_i / L_ij / L_ik / L_i- / L_il- / L_ik+
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

	var totalWeightedLoss float64 = 0
	for i, loss := range reputersReportedLosses {
		if loss <= 0 {
			return 0, fmt.Errorf("loss values must be greater than zero")
		}
		weightedLoss := (reputersStakes[i] / totalStake) * math.Log10(loss)
		totalWeightedLoss += weightedLoss
	}

	stakeWeightedLoss := math.Pow(10, totalWeightedLoss)

	return stakeWeightedLoss, nil
}
