package inference_synthesis

import (
	"fmt"
	"math"

	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

type WorkerRunningWeightedLoss struct {
	SumWeight Weight
	Loss      Loss
}

// Update the running weighted loss for the worker
// Source: "Weighted mean" section of: https://fanf2.user.srcf.net/hermes/doc/antiforgery/stats.pdf
func RunningWeightedAvgUpdate(
	runningWeightedAvg *WorkerRunningWeightedLoss,
	weight Weight,
	nextValue Weight,
	epsilon float64,
) (WorkerRunningWeightedLoss, error) {
	// weightedAvg_n = weightedAvg_{n-1} + (weight_n / sumOfWeights_n) * (log10(val_n) - weightedAvg_{n-1})
	runningWeightedAvg.SumWeight += weight
	if runningWeightedAvg.SumWeight < epsilon {
		return *runningWeightedAvg, emissions.ErrFractionDivideByZero
	}
	runningWeightedAvg.Loss += runningWeightedAvg.Loss + (weight/runningWeightedAvg.SumWeight)*(math.Log10(nextValue)-runningWeightedAvg.Loss)
	return *runningWeightedAvg, nil
}

// Convert the running weighted averages to WorkerAttributedValues
func convertMapOfRunningWeightedLossesToWorkerAttributedValue(
	runningWeightedLosses map[Worker]*WorkerRunningWeightedLoss,
) []*emissions.WorkerAttributedValue {
	weightedLosses := make([]*emissions.WorkerAttributedValue, 0)
	for worker, loss := range runningWeightedLosses {
		weightedLosses = append(weightedLosses, &emissions.WorkerAttributedValue{
			Worker: worker,
			Value:  loss.Loss,
		})
	}
	return weightedLosses
}

// Convert the running weighted averages to WithheldWorkerAttributedValue
func convertMapOfRunningWeightedLossesToWithheldWorkerAttributedValue(
	runningWeightedLosses map[Worker]*WorkerRunningWeightedLoss,
) []*emissions.WithheldWorkerAttributedValue {
	weightedLosses := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for worker, loss := range runningWeightedLosses {
		weightedLosses = append(weightedLosses, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  loss.Loss,
		})
	}
	return weightedLosses
}

func stakePlacementToFloat64(stake emissions.StakePlacement) float64 {
	return float64(stake.Amount.Uint64())
}

func CalcNetworkLosses(
	stakesByReputer map[Worker]Stake,
	reputerReportedLosses emissions.ReputerValueBundles,
	epsilon float64,
) (emissions.ValueBundle, error) {
	// Make map from inferer to their running weighted-average loss
	runningWeightedCombinedLoss := WorkerRunningWeightedLoss{0, 0}
	runningWeightedInfererLosses := make(map[Worker]*WorkerRunningWeightedLoss)
	runningWeightedForecasterLosses := make(map[Worker]*WorkerRunningWeightedLoss)
	runningWeightedNaiveLoss := WorkerRunningWeightedLoss{0, 0}
	runningWeightedOneOutInfererLosses := make(map[Worker]*WorkerRunningWeightedLoss) // Withheld worker -> Forecaster -> Loss
	runningWeightedOneOutForecasterLosses := make(map[Worker]*WorkerRunningWeightedLoss)
	runningWeightedOneInForecasterLosses := make(map[Worker]*WorkerRunningWeightedLoss)

	for _, report := range reputerReportedLosses.ReputerValueBundles {
		if report.ValueBundle != nil {
			stakeAmount := stakePlacementToFloat64(stakesByReputer[report.Reputer])
			// Update combined loss with reputer reported loss and stake
			nextCombinedLoss, err := RunningWeightedAvgUpdate(&runningWeightedCombinedLoss, stakeAmount, report.ValueBundle.CombinedValue, epsilon)
			if err != nil {
				fmt.Println("Error updating running weighted average for combined loss: ", err)
				return emissions.ValueBundle{}, err
			}
			runningWeightedCombinedLoss = nextCombinedLoss

			// Not all reputers may have reported losses on the same set of inferers => important that the code below doesn't assume that!
			// Update inferer losses
			for _, loss := range report.ValueBundle.InfererValues {
				nextAvg, err := RunningWeightedAvgUpdate(runningWeightedInfererLosses[loss.Worker], stakeAmount, loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for inferer: ", err)
					return emissions.ValueBundle{}, err
				}
				runningWeightedInfererLosses[loss.Worker] = &nextAvg
			}

			// Update forecaster losses
			for _, loss := range report.ValueBundle.ForecasterValues {
				nextAvg, err := RunningWeightedAvgUpdate(runningWeightedForecasterLosses[loss.Worker], stakeAmount, loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for forecaster: ", err)
					return emissions.ValueBundle{}, err
				}
				runningWeightedForecasterLosses[loss.Worker] = &nextAvg
			}

			// Update naive loss
			nextNaiveLoss, err := RunningWeightedAvgUpdate(&runningWeightedNaiveLoss, stakeAmount, report.ValueBundle.NaiveValue, epsilon)
			if err != nil {
				fmt.Println("Error updating running weighted average for naive loss: ", err)
				return emissions.ValueBundle{}, err
			}
			runningWeightedCombinedLoss = nextNaiveLoss

			// Update one-out inferer losses
			for _, loss := range report.ValueBundle.OneOutInfererValues {
				nextAvg, err := RunningWeightedAvgUpdate(runningWeightedOneOutInfererLosses[loss.Worker], stakeAmount, loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for one-out inferer: ", err)
					return emissions.ValueBundle{}, err
				}
				runningWeightedOneOutInfererLosses[loss.Worker] = &nextAvg
			}

			// Update one-out forecaster losses
			for _, loss := range report.ValueBundle.OneOutForecasterValues {
				nextAvg, err := RunningWeightedAvgUpdate(runningWeightedOneOutForecasterLosses[loss.Worker], stakeAmount, loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for one-out forecaster: ", err)
					return emissions.ValueBundle{}, err
				}
				runningWeightedOneOutForecasterLosses[loss.Worker] = &nextAvg
			}

			// Update one-in forecaster losses
			for _, loss := range report.ValueBundle.OneOutForecasterValues {
				nextAvg, err := RunningWeightedAvgUpdate(runningWeightedOneInForecasterLosses[loss.Worker], stakeAmount, loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for one-in forecaster: ", err)
					return emissions.ValueBundle{}, err
				}
				runningWeightedOneInForecasterLosses[loss.Worker] = &nextAvg
			}
		}
	}

	// Convert the running weighted averages to WorkerAttributedValue for inferers and forecasters
	output := emissions.ValueBundle{
		CombinedValue:          runningWeightedCombinedLoss.Loss,
		InfererValues:          convertMapOfRunningWeightedLossesToWorkerAttributedValue(runningWeightedInfererLosses),
		ForecasterValues:       convertMapOfRunningWeightedLossesToWorkerAttributedValue(runningWeightedForecasterLosses),
		NaiveValue:             runningWeightedNaiveLoss.Loss,
		OneOutInfererValues:    convertMapOfRunningWeightedLossesToWithheldWorkerAttributedValue(runningWeightedOneOutInfererLosses),
		OneOutForecasterValues: convertMapOfRunningWeightedLossesToWithheldWorkerAttributedValue(runningWeightedOneOutForecasterLosses),
		OneInForecasterValues:  convertMapOfRunningWeightedLossesToWorkerAttributedValue(runningWeightedOneInForecasterLosses),
	}

	return output, nil
}

// Same as CalcNetworkLosses() but just returns the combined loss
func CalcCombinedNetworkLoss(
	stakesByReputer map[Worker]Stake,
	reputerReportedLosses *emissions.ReputerValueBundles,
	epsilon float64,
) (Loss, error) {
	// Make map from inferer to their running weighted-average loss
	runningWeightedCombinedLoss := WorkerRunningWeightedLoss{0, 0}

	for _, report := range reputerReportedLosses.ReputerValueBundles {
		if report.ValueBundle != nil {
			stakeAmount := stakePlacementToFloat64(stakesByReputer[report.Reputer])
			// Update combined loss with reputer reported loss and stake
			nextCombinedLoss, err := RunningWeightedAvgUpdate(&runningWeightedCombinedLoss, stakeAmount, report.ValueBundle.CombinedValue, epsilon)
			if err != nil {
				fmt.Println("Error updating running weighted average for combined loss: ", err)
				return 0, err
			}
			runningWeightedCombinedLoss = nextCombinedLoss
		}
	}

	return runningWeightedCombinedLoss.Loss, nil
}
