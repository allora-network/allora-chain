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
// nextValue format - raw value
// weight format - logged value
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
	if err != nil {
		return WorkerRunningWeightedLoss{}, err
	}
	nextValueMinusLoss, err := nextValue.Sub(runningWeightedAvg.Loss)
	if err != nil {
		return WorkerRunningWeightedLoss{}, err
	}
	weightFracTimesNextValueMinusLoss, err := weightFrac.Mul(nextValueMinusLoss)
	if err != nil {
		return WorkerRunningWeightedLoss{}, err
	}
	runningWeightedAvg.Loss, err = runningWeightedAvg.Loss.Add(weightFracTimesNextValueMinusLoss)
	if err != nil {
		return WorkerRunningWeightedLoss{}, err
	}
	return *runningWeightedAvg, nil
}

// Convert and exponentiate the running weighted averages to WorkerAttributedValues
func convertMapOfRunningWeightedLossesToWorkerAttributedValue[T emissions.WorkerAttributedValue | emissions.WithheldWorkerAttributedValue](
	runningWeightedLosses map[Worker]*WorkerRunningWeightedLoss,
	sortedWorkers []Worker,
) []*T {
	weightedLosses := make([]*T, 0)
	for _, worker := range sortedWorkers {
		runningLoss, ok := runningWeightedLosses[worker]
		if !ok {
			continue
		}
		weightedLosses = append(weightedLosses, &T{
			Worker: worker,
			Value:  runningLoss.Loss,
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
			stakeAmount, err := alloraMath.NewDecFromSdkInt(stakesByReputer[report.ValueBundle.Reputer])
			if err != nil {
				return emissions.ValueBundle{}, err
			}

			// Update combined loss with reputer reported loss and stake
			nextCombinedLoss, err := RunningWeightedAvgUpdate(&runningWeightedCombinedLoss, stakeAmount, report.ValueBundle.CombinedValue, epsilon)
			if err != nil {
				fmt.Println("Error updating running weighted average for next combined loss: ", err)
				return emissions.ValueBundle{}, err
			}
			// If the combined loss is zero, set it to epsilon to avoid divide by zero
			if nextCombinedLoss.Loss.IsZero() {
				nextCombinedLoss.Loss = epsilon
			}
			runningWeightedCombinedLoss = nextCombinedLoss

			// Not all reputers may have reported losses on the same set of inferers => important that the code below doesn't assume that!
			// Update inferer losses
			for _, loss := range report.ValueBundle.InfererValues {
				if runningWeightedInfererLosses[loss.Worker] == nil {
					runningWeightedInfererLosses[loss.Worker] = &WorkerRunningWeightedLoss{
						SumWeight: alloraMath.ZeroDec(),
						Loss:      alloraMath.ZeroDec(),
					}
				}

				nextAvg, err := RunningWeightedAvgUpdate(runningWeightedInfererLosses[loss.Worker], stakeAmount, loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for inferer: ", err)
					return emissions.ValueBundle{}, err
				}
				runningWeightedInfererLosses[loss.Worker] = &nextAvg
			}

			// Update forecaster losses
			for _, loss := range report.ValueBundle.ForecasterValues {
				if runningWeightedForecasterLosses[loss.Worker] == nil {
					runningWeightedForecasterLosses[loss.Worker] = &WorkerRunningWeightedLoss{
						SumWeight: alloraMath.ZeroDec(),
						Loss:      alloraMath.ZeroDec(),
					}
				}

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
			runningWeightedNaiveLoss = nextNaiveLoss

			// Update one-out inferer losses
			for _, loss := range report.ValueBundle.OneOutInfererValues {
				if runningWeightedOneOutInfererLosses[loss.Worker] == nil {
					runningWeightedOneOutInfererLosses[loss.Worker] = &WorkerRunningWeightedLoss{
						SumWeight: alloraMath.ZeroDec(),
						Loss:      alloraMath.ZeroDec(),
					}
				}
				nextAvg, err := RunningWeightedAvgUpdate(runningWeightedOneOutInfererLosses[loss.Worker], stakeAmount, loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for one-out inferer: ", err)
					return emissions.ValueBundle{}, err
				}
				runningWeightedOneOutInfererLosses[loss.Worker] = &nextAvg
			}

			// Update one-out forecaster losses
			for _, loss := range report.ValueBundle.OneOutForecasterValues {
				if runningWeightedOneOutForecasterLosses[loss.Worker] == nil {
					runningWeightedOneOutForecasterLosses[loss.Worker] = &WorkerRunningWeightedLoss{
						SumWeight: alloraMath.ZeroDec(),
						Loss:      alloraMath.ZeroDec(),
					}
				}

				nextAvg, err := RunningWeightedAvgUpdate(runningWeightedOneOutForecasterLosses[loss.Worker], stakeAmount, loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for one-out forecaster: ", err)
					return emissions.ValueBundle{}, err
				}
				runningWeightedOneOutForecasterLosses[loss.Worker] = &nextAvg
			}

			// Update one-in forecaster losses
			for _, loss := range report.ValueBundle.OneInForecasterValues {
				if runningWeightedOneInForecasterLosses[loss.Worker] == nil {
					runningWeightedOneInForecasterLosses[loss.Worker] = &WorkerRunningWeightedLoss{
						SumWeight: alloraMath.ZeroDec(),
						Loss:      alloraMath.ZeroDec(),
					}
				}

				nextAvg, err := RunningWeightedAvgUpdate(runningWeightedOneInForecasterLosses[loss.Worker], stakeAmount, loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for one-in forecaster: ", err)
					return emissions.ValueBundle{}, err
				}
				runningWeightedOneInForecasterLosses[loss.Worker] = &nextAvg
			}
		}
	}

	sortedInferers := alloraMath.GetSortedKeys(runningWeightedInfererLosses)
	sortedForecasters := alloraMath.GetSortedKeys(runningWeightedForecasterLosses)
	// Convert the running weighted averages to WorkerAttributedValue/WithheldWorkerAttributedValue for inferers and forecasters
	infererLosses := convertMapOfRunningWeightedLossesToWorkerAttributedValue[emissions.WorkerAttributedValue](runningWeightedInfererLosses, sortedInferers)
	forecasterLosses := convertMapOfRunningWeightedLossesToWorkerAttributedValue[emissions.WorkerAttributedValue](runningWeightedForecasterLosses, sortedForecasters)
	oneOutInfererLosses := convertMapOfRunningWeightedLossesToWorkerAttributedValue[emissions.WithheldWorkerAttributedValue](runningWeightedOneOutInfererLosses, sortedInferers)
	oneOutForecasterLosses := convertMapOfRunningWeightedLossesToWorkerAttributedValue[emissions.WithheldWorkerAttributedValue](runningWeightedOneOutForecasterLosses, sortedForecasters)
	oneInForecasterLosses := convertMapOfRunningWeightedLossesToWorkerAttributedValue[emissions.WorkerAttributedValue](runningWeightedOneInForecasterLosses, sortedForecasters)

	output := emissions.ValueBundle{
		CombinedValue:          runningWeightedCombinedLoss.Loss,
		InfererValues:          infererLosses,
		ForecasterValues:       forecasterLosses,
		NaiveValue:             runningWeightedNaiveLoss.Loss,
		OneOutInfererValues:    oneOutInfererLosses,
		OneOutForecasterValues: oneOutForecasterLosses,
		OneInForecasterValues:  oneInForecasterLosses,
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
			stakeAmount, err := alloraMath.NewDecFromSdkInt(stakesByReputer[report.ValueBundle.Reputer])
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
				return Loss{}, err
			}
			runningWeightedCombinedLoss = nextCombinedLoss
		}
	}
	// If the combined loss is zero, set it to epsilon to avoid divide by zero
	if runningWeightedCombinedLoss.Loss.IsZero() {
		runningWeightedCombinedLoss.Loss = epsilon
	}
	return runningWeightedCombinedLoss.Loss, nil
}
