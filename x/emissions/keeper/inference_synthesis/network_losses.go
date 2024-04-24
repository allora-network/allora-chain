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
// weigth format - logged value
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
	runningWeightedAvg.Loss, err = runningWeightedAvg.Loss.Add(weightFracTimesLog10NextValueMinusLoss)
	if err != nil {
		return WorkerRunningWeightedLoss{}, err
	}

	return *runningWeightedAvg, nil
}

// Convert and exponentiate the running weighted averages to WorkerAttributedValues
func convertAndExpMapOfRunningWeightedLossesToWorkerAttributedValue(
	runningWeightedLosses map[Worker]*WorkerRunningWeightedLoss,
) ([]*emissions.WorkerAttributedValue, error) {
	weightedLosses := make([]*emissions.WorkerAttributedValue, 0)
	for worker, loss := range runningWeightedLosses {
		expLoss, err := alloraMath.Exp10(loss.Loss)
		if err != nil {
			return nil, err
		}
		weightedLosses = append(weightedLosses, &emissions.WorkerAttributedValue{
			Worker: worker,
			Value:  expLoss,
		})
	}
	return weightedLosses, nil
}

// Convert and exponentiate the running weighted averages to WithheldWorkerAttributedValue
func convertAndExpMapOfRunningWeightedLossesToWithheldWorkerAttributedValue(
	runningWeightedLosses map[Worker]*WorkerRunningWeightedLoss,
) ([]*emissions.WithheldWorkerAttributedValue, error) {
	weightedLosses := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for worker, loss := range runningWeightedLosses {
		expLoss, err := alloraMath.Exp10(loss.Loss)
		if err != nil {
			return nil, err
		}
		weightedLosses = append(weightedLosses, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  expLoss,
		})
	}
	return weightedLosses, nil
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
			stakeAmount, err := alloraMath.NewDecFromSdkUint(stakesByReputer[report.ValueBundle.Reputer])
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
				if runningWeightedInfererLosses[loss.Worker] == nil {
					runningWeightedInfererLosses[loss.Worker] = &WorkerRunningWeightedLoss{
						SumWeight: alloraMath.MustNewDecFromString("0"),
						Loss:      alloraMath.MustNewDecFromString("0"),
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
						SumWeight: alloraMath.MustNewDecFromString("0"),
						Loss:      alloraMath.MustNewDecFromString("0"),
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
						SumWeight: alloraMath.MustNewDecFromString("0"),
						Loss:      alloraMath.MustNewDecFromString("0"),
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
						SumWeight: alloraMath.MustNewDecFromString("0"),
						Loss:      alloraMath.MustNewDecFromString("0"),
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
			for _, loss := range report.ValueBundle.OneOutForecasterValues {
				if runningWeightedOneInForecasterLosses[loss.Worker] == nil {
					runningWeightedOneInForecasterLosses[loss.Worker] = &WorkerRunningWeightedLoss{
						SumWeight: alloraMath.MustNewDecFromString("0"),
						Loss:      alloraMath.MustNewDecFromString("0"),
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

	// Convert the running weighted averages to WorkerAttributedValue for inferers and forecasters + exponentiate
	expRunningWeightedCombinedLoss, err := alloraMath.Exp10(runningWeightedCombinedLoss.Loss)
	if err != nil {
		return emissions.ValueBundle{}, err
	}
	expInfererLosses, err := convertAndExpMapOfRunningWeightedLossesToWorkerAttributedValue(runningWeightedInfererLosses)
	if err != nil {
		return emissions.ValueBundle{}, err
	}
	expForecasterLosses, err := convertAndExpMapOfRunningWeightedLossesToWorkerAttributedValue(runningWeightedForecasterLosses)
	if err != nil {
		return emissions.ValueBundle{}, err
	}
	expRunningWeightedNaiveLoss, err := alloraMath.Exp10(runningWeightedNaiveLoss.Loss)
	if err != nil {
		return emissions.ValueBundle{}, err
	}
	expOneOutInfererLosses, err := convertAndExpMapOfRunningWeightedLossesToWithheldWorkerAttributedValue(runningWeightedOneOutInfererLosses)
	if err != nil {
		return emissions.ValueBundle{}, err
	}
	expOneOutForecasterLosses, err := convertAndExpMapOfRunningWeightedLossesToWithheldWorkerAttributedValue(runningWeightedOneOutForecasterLosses)
	if err != nil {
		return emissions.ValueBundle{}, err
	}
	expOneInForecasterLosses, err := convertAndExpMapOfRunningWeightedLossesToWorkerAttributedValue(runningWeightedOneInForecasterLosses)
	if err != nil {
		return emissions.ValueBundle{}, err
	}
	output := emissions.ValueBundle{
		CombinedValue:          expRunningWeightedCombinedLoss,
		InfererValues:          expInfererLosses,
		ForecasterValues:       expForecasterLosses,
		NaiveValue:             expRunningWeightedNaiveLoss,
		OneOutInfererValues:    expOneOutInfererLosses,
		OneOutForecasterValues: expOneOutForecasterLosses,
		OneInForecasterValues:  expOneInForecasterLosses,
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
			stakeAmount, err := alloraMath.NewDecFromSdkUint(stakesByReputer[report.ValueBundle.Reputer])
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

	// Exponentiate
	expRunningWeightedCombinedLoss, err := alloraMath.Exp10(runningWeightedCombinedLoss.Loss)
	if err != nil {
		fmt.Println("Error exponentiating combined loss: ", err)
		return alloraMath.ZeroDec(), err
	}

	return expRunningWeightedCombinedLoss, nil
}
