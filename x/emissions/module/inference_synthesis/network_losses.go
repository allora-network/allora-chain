package inference_synthesis

import (
	"fmt"
	"math"

	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

type workerRunningWeightedLoss struct {
	SumWeight Weight
	Loss      Loss
}

// Update the running weighted loss for the worker
// Source: "Weighted mean" section of: https://fanf2.user.srcf.net/hermes/doc/antiforgery/stats.pdf
func runningWeightedAvgUpdate(
	runningWeightedAvg *workerRunningWeightedLoss,
	weight Weight,
	nextValue Weight,
	epsilon float64,
) (workerRunningWeightedLoss, error) {
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
	runningWeightedLosses map[Worker]*workerRunningWeightedLoss,
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
	runningWeightedLosses map[Worker]*workerRunningWeightedLoss,
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

func CalcNetworkLosses(
	stakesByReputer map[Worker]Stake,
	reputerReportedLosses *emissions.ReputerValueBundles,
	epsilon float64,
) (*emissions.ValueBundle, error) {
	// Make map from inferer to their running weighted-average loss
	runningWeightedCombinedLoss := workerRunningWeightedLoss{0, 0}
	runningWeightedInfererLosses := make(map[Worker]*workerRunningWeightedLoss)
	runningWeightedForecasterLosses := make(map[Worker]*workerRunningWeightedLoss)
	runningWeightedNaiveLoss := workerRunningWeightedLoss{0, 0}
	runningWeightedOneOutInfererLosses := make(map[Worker]*workerRunningWeightedLoss) // Withheld worker -> Forecaster -> Loss
	runningWeightedOneOutForecasterLosses := make(map[Worker]*workerRunningWeightedLoss)
	runningWeightedOneInForecasterLosses := make(map[Worker]*workerRunningWeightedLoss)

	for _, report := range reputerReportedLosses.ReputerValueBundles {
		if report.ValueBundle != nil {
			// Update combined loss
			nextCombinedLoss, err := runningWeightedAvgUpdate(&runningWeightedCombinedLoss, stakesByReputer[report.Reputer], report.ValueBundle.CombinedValue, epsilon)
			if err != nil {
				fmt.Println("Error updating running weighted average for combined loss: ", err)
				return &emissions.ValueBundle{}, err
			}
			runningWeightedCombinedLoss = nextCombinedLoss

			// Not all reputers may have reported losses on the same set of inferers => important that the code below doesn't assume that!
			// Update inferer losses
			for _, loss := range report.ValueBundle.InfererValues {
				nextAvg, err := runningWeightedAvgUpdate(runningWeightedInfererLosses[loss.Worker], stakesByReputer[report.Reputer], loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for inferer: ", err)
					return &emissions.ValueBundle{}, err
				}
				runningWeightedInfererLosses[loss.Worker] = &nextAvg
			}

			// Update forecaster losses
			for _, loss := range report.ValueBundle.ForecasterValues {
				nextAvg, err := runningWeightedAvgUpdate(runningWeightedForecasterLosses[loss.Worker], stakesByReputer[report.Reputer], loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for forecaster: ", err)
					return &emissions.ValueBundle{}, err
				}
				runningWeightedForecasterLosses[loss.Worker] = &nextAvg
			}

			// Update naive loss
			nextNaiveLoss, err := runningWeightedAvgUpdate(&runningWeightedNaiveLoss, stakesByReputer[report.Reputer], report.ValueBundle.NaiveValue, epsilon)
			if err != nil {
				fmt.Println("Error updating running weighted average for naive loss: ", err)
				return &emissions.ValueBundle{}, err
			}
			runningWeightedCombinedLoss = nextNaiveLoss

			// Update one-out inferer losses
			for _, loss := range report.ValueBundle.OneOutInfererValues {
				nextAvg, err := runningWeightedAvgUpdate(runningWeightedOneOutInfererLosses[loss.Worker], stakesByReputer[report.Reputer], loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for one-out inferer: ", err)
					return &emissions.ValueBundle{}, err
				}
				runningWeightedOneOutInfererLosses[loss.Worker] = &nextAvg
			}

			// Update one-out forecaster losses
			for _, loss := range report.ValueBundle.OneOutForecasterValues {
				nextAvg, err := runningWeightedAvgUpdate(runningWeightedOneOutForecasterLosses[loss.Worker], stakesByReputer[report.Reputer], loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for one-out forecaster: ", err)
					return &emissions.ValueBundle{}, err
				}
				runningWeightedOneOutForecasterLosses[loss.Worker] = &nextAvg
			}

			// Update one-in forecaster losses
			for _, loss := range report.ValueBundle.OneOutForecasterValues {
				nextAvg, err := runningWeightedAvgUpdate(runningWeightedOneInForecasterLosses[loss.Worker], stakesByReputer[report.Reputer], loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for one-in forecaster: ", err)
					return &emissions.ValueBundle{}, err
				}
				runningWeightedOneInForecasterLosses[loss.Worker] = &nextAvg
			}
		}
	}

	// Convert the running weighted averages to WorkerAttributedValue for inferers and forecasters
	output := &emissions.ValueBundle{
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
