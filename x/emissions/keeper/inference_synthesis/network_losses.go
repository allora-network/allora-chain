package inference_synthesis

import (
	"fmt"

	alloraMath "github.com/allora-network/allora-chain/math"

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
	epsilon alloraMath.Dec,
) (WorkerRunningWeightedLoss, error) {
	var err error
	// weightedAvg_n = weightedAvg_{n-1} + (weight_n / sumOfWeights_n) * (log10(val_n) - weightedAvg_{n-1})
	runningWeightedAvg.SumWeight, err = runningWeightedAvg.SumWeight.Add(weight)
	if err != nil {
		return WorkerRunningWeightedLoss{}, err
	}
	if runningWeightedAvg.SumWeight.Lt(epsilon) {
		return *runningWeightedAvg, emissions.ErrFractionDivideByZero
	}
	weightFrac, err := weight.Quo(runningWeightedAvg.SumWeight)
	if err != nil {
		return WorkerRunningWeightedLoss{}, err
	}
	log10NextValue, err := alloraMath.Log10(nextValue)
	if err != nil {
		return WorkerRunningWeightedLoss{}, err
	}
	log10NextValueMinusLoss, err := log10NextValue.Sub(runningWeightedAvg.Loss)
	if err != nil {
		return WorkerRunningWeightedLoss{}, err
	}
	weightFracTimesLog10NextValueMinusLoss, err := weightFrac.Mul(log10NextValueMinusLoss)
	if err != nil {
		return WorkerRunningWeightedLoss{}, err
	}
	runningLoss, err := runningWeightedAvg.Loss.Add(weightFracTimesLog10NextValueMinusLoss)
	if err != nil {
		return WorkerRunningWeightedLoss{}, err
	}
	runningWeightedAvg.Loss, err = runningWeightedAvg.Loss.Add(runningLoss)
	if err != nil {
		return WorkerRunningWeightedLoss{}, err
	}
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

func CalcNetworkLosses(
	stakesByReputer map[Worker]Stake,
	reputerReportedLosses emissions.ReputerValueBundles,
	epsilon alloraMath.Dec,
) (emissions.ValueBundle, error) {
	// Make map from inferer to their running weighted-average loss
	runningWeightedCombinedLoss := WorkerRunningWeightedLoss{alloraMath.ZeroDec(), alloraMath.ZeroDec()}
	runningWeightedInfererLosses := make(map[Worker]*WorkerRunningWeightedLoss)
	runningWeightedForecasterLosses := make(map[Worker]*WorkerRunningWeightedLoss)
	runningWeightedNaiveLoss := WorkerRunningWeightedLoss{alloraMath.ZeroDec(), alloraMath.ZeroDec()}
	runningWeightedOneOutInfererLosses := make(map[Worker]*WorkerRunningWeightedLoss) // Withheld worker -> Forecaster -> Loss
	runningWeightedOneOutForecasterLosses := make(map[Worker]*WorkerRunningWeightedLoss)
	runningWeightedOneInForecasterLosses := make(map[Worker]*WorkerRunningWeightedLoss)

	for _, report := range reputerReportedLosses.ReputerValueBundles {
		if report.ValueBundle != nil {
			stakeAmount, err := alloraMath.NewDecFromSdkUint(stakesByReputer[report.ValueBundle.Reputer].Amount)
			if err != nil {
				return emissions.ValueBundle{}, err
			}
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
	epsilon alloraMath.Dec,
) (Loss, error) {
	// Make map from inferer to their running weighted-average loss
	runningWeightedCombinedLoss := WorkerRunningWeightedLoss{alloraMath.ZeroDec(), alloraMath.ZeroDec()}

	for _, report := range reputerReportedLosses.ReputerValueBundles {
		if report.ValueBundle != nil {
			stakeAmount, err := alloraMath.NewDecFromSdkUint(stakesByReputer[report.ValueBundle.Reputer].Amount)
			if err != nil {
				fmt.Println("Error converting stake to Dec: ", err)
				return Loss{}, err
			}
			// Update combined loss with reputer reported loss and stake
			nextCombinedLoss, err := RunningWeightedAvgUpdate(
				&runningWeightedCombinedLoss,
				stakeAmount,
				report.ValueBundle.CombinedValue,
				epsilon,
			)
			if err != nil {
				fmt.Println("Error updating running weighted average for combined loss: ", err)
				return alloraMath.ZeroDec(), err
			}
			runningWeightedCombinedLoss = nextCombinedLoss
		}
	}

	return runningWeightedCombinedLoss.Loss, nil
}
