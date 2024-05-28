package inference_synthesis

import (
	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

type RunningWeightedLoss struct {
	UnnormalizedWeightedLoss Loss
	SumWeight                Weight
}

// Update the running information needed to calculate weighted loss per worker
func RunningWeightedAvgUpdate(
	runningWeightedAvg *RunningWeightedLoss,
	nextWeight Weight,
	nextValue Weight,
) (RunningWeightedLoss, error) {
	nextValTimesWeight, err := nextValue.Mul(nextWeight)
	if err != nil {
		return RunningWeightedLoss{}, err
	}
	newUnnormalizedWeightedLoss, err := runningWeightedAvg.UnnormalizedWeightedLoss.Add(nextValTimesWeight)
	if err != nil {
		return RunningWeightedLoss{}, err
	}
	newSumWeight, err := runningWeightedAvg.SumWeight.Add(nextWeight)
	if err != nil {
		return RunningWeightedLoss{}, err
	}
	return RunningWeightedLoss{
		UnnormalizedWeightedLoss: newUnnormalizedWeightedLoss,
		SumWeight:                newSumWeight,
	}, nil
}

// Convert the running weighted average objects to WorkerAttributedValues
func convertMapOfRunningWeightedLossesToWorkerAttributedValue[T emissions.WorkerAttributedValue | emissions.WithheldWorkerAttributedValue](
	runningWeightedLosses map[Worker]*RunningWeightedLoss,
	sortedWorkers []Worker,
	epsilon alloraMath.Dec,
) []*T {
	weightedLosses := make([]*T, 0)
	for _, worker := range sortedWorkers {
		runningLoss, ok := runningWeightedLosses[worker]
		if !ok {
			continue
		}
		normalizedWeightedLoss, err := normalizeWeightedLoss(runningLoss, epsilon)
		if err != nil {
			continue
		}
		weightedLosses = append(weightedLosses, &T{
			Worker: worker,
			Value:  normalizedWeightedLoss,
		})
	}
	return weightedLosses
}

// Assumes stakes are all positive
func CalcNetworkLosses(
	stakesByReputer map[Worker]Stake,
	reputerReportedLosses emissions.ReputerValueBundles,
	epsilon alloraMath.Dec,
) (emissions.ValueBundle, error) {
	// Make map from inferer to their running weighted-average loss
	runningWeightedCombinedLoss := RunningWeightedLoss{alloraMath.ZeroDec(), alloraMath.ZeroDec()}
	runningWeightedInfererLosses := make(map[Worker]*RunningWeightedLoss)
	runningWeightedForecasterLosses := make(map[Worker]*RunningWeightedLoss)
	runningWeightedNaiveLoss := RunningWeightedLoss{alloraMath.ZeroDec(), alloraMath.ZeroDec()}
	runningWeightedOneOutInfererLosses := make(map[Worker]*RunningWeightedLoss) // Withheld worker -> Forecaster -> Loss
	runningWeightedOneOutForecasterLosses := make(map[Worker]*RunningWeightedLoss)
	runningWeightedOneInForecasterLosses := make(map[Worker]*RunningWeightedLoss)

	for _, report := range reputerReportedLosses.ReputerValueBundles {
		if report.ValueBundle != nil {
			stakeAmount, err := alloraMath.NewDecFromSdkInt(stakesByReputer[report.ValueBundle.Reputer])
			if err != nil {
				return emissions.ValueBundle{}, err
			}

			// Update combined loss with reputer reported loss and stake
			runningWeightedCombinedLoss, err = RunningWeightedAvgUpdate(&runningWeightedCombinedLoss, stakeAmount, report.ValueBundle.CombinedValue)
			if err != nil {
				return emissions.ValueBundle{}, errorsmod.Wrapf(err, "Error updating running weighted average for next combined loss")
			}

			// Not all reputers may have reported losses on the same set of inferers => important that the code below doesn't assume that!
			// Update inferer losses
			for _, loss := range report.ValueBundle.InfererValues {
				if runningWeightedInfererLosses[loss.Worker] == nil {
					runningWeightedInfererLosses[loss.Worker] = &RunningWeightedLoss{
						UnnormalizedWeightedLoss: alloraMath.ZeroDec(),
						SumWeight:                alloraMath.ZeroDec(),
					}
				}

				nextAvg, err := RunningWeightedAvgUpdate(runningWeightedInfererLosses[loss.Worker], stakeAmount, loss.Value)
				if err != nil {
					return emissions.ValueBundle{}, errorsmod.Wrapf(err, "Error updating running weighted average for inferer")
				}
				runningWeightedInfererLosses[loss.Worker] = &nextAvg
			}

			// Update forecaster losses
			for _, loss := range report.ValueBundle.ForecasterValues {
				if runningWeightedForecasterLosses[loss.Worker] == nil {
					runningWeightedForecasterLosses[loss.Worker] = &RunningWeightedLoss{
						UnnormalizedWeightedLoss: alloraMath.ZeroDec(),
						SumWeight:                alloraMath.ZeroDec(),
					}
				}

				nextAvg, err := RunningWeightedAvgUpdate(runningWeightedForecasterLosses[loss.Worker], stakeAmount, loss.Value)
				if err != nil {
					return emissions.ValueBundle{}, errorsmod.Wrapf(err, "Error updating running weighted average for forecaster")
				}
				runningWeightedForecasterLosses[loss.Worker] = &nextAvg
			}

			// Update naive loss
			runningWeightedNaiveLoss, err = RunningWeightedAvgUpdate(&runningWeightedNaiveLoss, stakeAmount, report.ValueBundle.NaiveValue)
			if err != nil {
				return emissions.ValueBundle{}, errorsmod.Wrapf(err, "Error updating running weighted average for naive loss: ")
			}

			// Update one-out inferer losses
			for _, loss := range report.ValueBundle.OneOutInfererValues {
				if runningWeightedOneOutInfererLosses[loss.Worker] == nil {
					runningWeightedOneOutInfererLosses[loss.Worker] = &RunningWeightedLoss{
						UnnormalizedWeightedLoss: alloraMath.ZeroDec(),
						SumWeight:                alloraMath.ZeroDec(),
					}
				}
				nextAvg, err := RunningWeightedAvgUpdate(runningWeightedOneOutInfererLosses[loss.Worker], stakeAmount, loss.Value)
				if err != nil {
					return emissions.ValueBundle{}, errorsmod.Wrapf(err, "Error updating running weighted average for one-out inferer")
				}
				runningWeightedOneOutInfererLosses[loss.Worker] = &nextAvg
			}

			// Update one-out forecaster losses
			for _, loss := range report.ValueBundle.OneOutForecasterValues {
				if runningWeightedOneOutForecasterLosses[loss.Worker] == nil {
					runningWeightedOneOutForecasterLosses[loss.Worker] = &RunningWeightedLoss{
						UnnormalizedWeightedLoss: alloraMath.ZeroDec(),
						SumWeight:                alloraMath.ZeroDec(),
					}
				}

				nextAvg, err := RunningWeightedAvgUpdate(runningWeightedOneOutForecasterLosses[loss.Worker], stakeAmount, loss.Value)
				if err != nil {
					return emissions.ValueBundle{}, errorsmod.Wrapf(err, "Error updating running weighted average for one-out forecaster")
				}
				runningWeightedOneOutForecasterLosses[loss.Worker] = &nextAvg
			}

			// Update one-in forecaster losses
			for _, loss := range report.ValueBundle.OneInForecasterValues {
				if runningWeightedOneInForecasterLosses[loss.Worker] == nil {
					runningWeightedOneInForecasterLosses[loss.Worker] = &RunningWeightedLoss{
						UnnormalizedWeightedLoss: alloraMath.ZeroDec(),
						SumWeight:                alloraMath.ZeroDec(),
					}
				}

				nextAvg, err := RunningWeightedAvgUpdate(runningWeightedOneInForecasterLosses[loss.Worker], stakeAmount, loss.Value)
				if err != nil {
					return emissions.ValueBundle{}, errorsmod.Wrapf(err, "Error updating running weighted average for one-in forecaster")
				}
				runningWeightedOneInForecasterLosses[loss.Worker] = &nextAvg
			}
		}
	}

	sortedInferers := alloraMath.GetSortedKeys(runningWeightedInfererLosses)
	sortedForecasters := alloraMath.GetSortedKeys(runningWeightedForecasterLosses)

	// Normalize the combined loss
	combinedValue, err := normalizeWeightedLoss(&runningWeightedCombinedLoss, epsilon)
	if err != nil {
		return emissions.ValueBundle{}, errorsmod.Wrapf(err, "Error normalizing combined loss")
	}

	// Normalize the naive loss
	naiveValue, err := normalizeWeightedLoss(&runningWeightedNaiveLoss, epsilon)
	if err != nil {
		return emissions.ValueBundle{}, errorsmod.Wrapf(err, "Error normalizing naive loss")
	}

	// Convert the running weighted averages to WorkerAttributedValue/WithheldWorkerAttributedValue for inferers and forecasters
	infererLosses := convertMapOfRunningWeightedLossesToWorkerAttributedValue[emissions.WorkerAttributedValue](runningWeightedInfererLosses, sortedInferers, epsilon)
	forecasterLosses := convertMapOfRunningWeightedLossesToWorkerAttributedValue[emissions.WorkerAttributedValue](runningWeightedForecasterLosses, sortedForecasters, epsilon)
	oneOutInfererLosses := convertMapOfRunningWeightedLossesToWorkerAttributedValue[emissions.WithheldWorkerAttributedValue](runningWeightedOneOutInfererLosses, sortedInferers, epsilon)
	oneOutForecasterLosses := convertMapOfRunningWeightedLossesToWorkerAttributedValue[emissions.WithheldWorkerAttributedValue](runningWeightedOneOutForecasterLosses, sortedForecasters, epsilon)
	oneInForecasterLosses := convertMapOfRunningWeightedLossesToWorkerAttributedValue[emissions.WorkerAttributedValue](runningWeightedOneInForecasterLosses, sortedForecasters, epsilon)

	output := emissions.ValueBundle{
		CombinedValue:          combinedValue,
		InfererValues:          infererLosses,
		ForecasterValues:       forecasterLosses,
		NaiveValue:             naiveValue,
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
	runningWeightedCombinedLoss := RunningWeightedLoss{alloraMath.ZeroDec(), alloraMath.ZeroDec()}

	for _, report := range reputerReportedLosses.ReputerValueBundles {
		if report.ValueBundle != nil {
			stakeAmount, err := alloraMath.NewDecFromSdkInt(stakesByReputer[report.ValueBundle.Reputer])
			if err != nil {
				return Loss{}, errorsmod.Wrapf(err, "Error converting stake to Dec")
			}

			// Update combined loss with reputer reported loss and stake
			nextCombinedLoss, err := RunningWeightedAvgUpdate(
				&runningWeightedCombinedLoss,
				stakeAmount,
				report.ValueBundle.CombinedValue,
			)
			if err != nil {
				return Loss{}, errorsmod.Wrapf(err, "Error updating running weighted average for combined loss: ")
			}
			runningWeightedCombinedLoss = nextCombinedLoss
		}
	}

	combinedValue, err := normalizeWeightedLoss(&runningWeightedCombinedLoss, epsilon)
	if err != nil {
		return Loss{}, errorsmod.Wrapf(err, "Error normalizing combined loss")
	}

	return combinedValue, nil
}

func normalizeWeightedLoss(
	runningWeightedLossData *RunningWeightedLoss,
	epsilon alloraMath.Dec,
) (alloraMath.Dec, error) {
	if runningWeightedLossData.SumWeight.Lt(epsilon) {
		return alloraMath.Dec{}, errorsmod.Wrapf(emissions.ErrFractionDivideByZero, "Sum weight for combined naive loss is 0")
	}

	normalizedWeightedLoss, err := runningWeightedLossData.UnnormalizedWeightedLoss.Quo(runningWeightedLossData.SumWeight)
	if err != nil {
		return alloraMath.Dec{}, err
	}

	if normalizedWeightedLoss.IsZero() {
		normalizedWeightedLoss = epsilon
	}

	return normalizedWeightedLoss, nil
}
