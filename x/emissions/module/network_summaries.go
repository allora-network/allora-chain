package module

import (
	"fmt"
	"math"

	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

/**
 * This file contains the logic from the Inference Synthesis Section of the litepaper,
 * which combine current inferences, forecasts, previous losses and regrets
 * into current network losses and regrets.
 */

// Implements function phi prime from litepaper
// φ'_p(x) = p * (ln(1 + e^x))^(p-1) * e^x / (1 + e^x)
func Gradient(p float64, x float64) (float64, error) {
	if math.IsNaN(p) || math.IsInf(p, 0) || math.IsNaN(x) || math.IsInf(x, 0) {
		return 0, emissions.ErrPhiInvalidInput
	}
	eToTheX := math.Exp(x)
	onePlusEToTheX := 1 + eToTheX
	if math.IsInf(onePlusEToTheX, 0) {
		return 0, emissions.ErrEToTheXExponentiationIsInfinity
	}
	naturalLog := math.Log(onePlusEToTheX)
	result := math.Pow(naturalLog, p-1)
	if math.IsInf(result, 0) {
		return 0, emissions.ErrLnToThePExponentiationIsInfinity
	}
	// should theoretically never be possible with the above checks
	if math.IsNaN(result) {
		return 0, emissions.ErrPhiResultIsNaN
	}
	result = (p * result * eToTheX) / onePlusEToTheX
	return result, nil
}

func MakeMapFromWorkerToTheirWork(inferences []*emissions.Inference) map[string]*emissions.Inference {
	inferencesByWorker := make(map[string]*emissions.Inference)
	for _, inference := range inferences {
		inferencesByWorker[inference.Worker] = inference
	}
	return inferencesByWorker
}

// Calculate the forecast-implied inferences I_ik given inferences, forecasts and network losses.
// Calculates R_ijk, w_ijk, and I_ik for each forecast k and forecast element (forcast of worker loss) j
//
// Forecast without inference => weight in calculation of I_ik and I_i set to 0. Use latest available regret R_i-1,l
// Inference without forecast => only weight in calculation of I_ik set to 0
func CalcForcastImpliedInferences(
	inferenceByWorker map[string]*emissions.Inference,
	forecasts *emissions.Forecasts,
	networkCombinedLoss float64,
	epsilon float64,
	pInferenceSynthesis float64) (map[string]*emissions.Inference, error) {
	// Possibly add a small value to previous network loss avoid infinite logarithm
	if networkCombinedLoss == 0 {
		// Take max of epsilon and 1 to avoid division by 0
		networkCombinedLoss = math.Max(epsilon, 1)
	}

	// "k" here is the forecaster's address
	// For each forecast, and for each forecast element, calculate forecast-implied inferences I_ik
	I_i := make(map[string]*emissions.Inference, len(forecasts.Forecasts))
	for _, forecast := range forecasts.Forecasts {
		if len(forecast.ForecastElements) > 0 {
			// Filter away all forecast elements that do not have an associated inference (match by worker)
			// Will effectively set weight in formulas for forcast-implied inference I_ik and network inference I_i to 0 for forecasts without inferences
			forecastElementsWithInferences := make([]*emissions.ForecastElement, 0)
			// Track workers with inferences => We only add the first forecast element per forecast per inferer
			inferenceWorkers := make(map[string]bool)
			for _, el := range forecast.ForecastElements {
				if _, ok := inferenceByWorker[el.Inferer]; ok && !inferenceWorkers[el.Inferer] {
					forecastElementsWithInferences = append(forecastElementsWithInferences, el)
					inferenceWorkers[el.Inferer] = true
				}
			}

			// Approximate forecast regrets of the network inference
			R_ik := make([]float64, len(forecastElementsWithInferences))
			// Weights used to map inferences to forecast-implied inferences
			w_ik := make([]float64, len(forecastElementsWithInferences))

			// Define variable to store maximum regret for forecast k
			maxjRijk := forecastElementsWithInferences[0].Value
			for j, el := range forecastElementsWithInferences {
				// Calculate the approximate forecast regret of the network inference
				R_ik[j] = math.Log10(networkCombinedLoss / el.Value) // forecasted regrets R_ijk = log10(L_i / L_ijk)
				if R_ik[j] > maxjRijk {
					maxjRijk = R_ik[j]
				}
			}

			// Calculate normalized forecasted regrets per forecaster R_ijk then weights w_ijk per forecaster
			for j := range forecastElementsWithInferences {
				R_ik[j] = R_ik[j] / maxjRijk                         // \hatR_ijk = R_ijk / |max_{j'}(R_ijk)|
				w_ijk, err := Gradient(pInferenceSynthesis, R_ik[j]) // w_ijk = φ'_p(\hatR_ijk)
				if err != nil {
					fmt.Println("Error calculating gradient: ", err)
					return nil, err
				}
				w_ik[j] = w_ijk
			}

			// Calculate the forecast-implied inferences I_ik
			weightSum := 0.0
			weightInferenceDotProduct := 0.0
			for j, w_ijk := range w_ik {
				weightInferenceDotProduct += w_ijk * inferenceByWorker[forecast.ForecastElements[j].Inferer].Value
				weightSum += w_ijk
			}
			forecastImpliedInference := emissions.Inference{
				Worker: forecast.Forecaster,
				Value:  weightInferenceDotProduct / weightSum,
			}
			I_i[forecast.Forecaster] = &forecastImpliedInference
		}
	}

	// A value of 0 => no inference corresponded to any of the forecasts from a forecaster
	return I_i, nil
}

type RegretsByWorkerByType struct {
	InferenceRegrets *map[string]*float64
	ForecastRegrets  *map[string]*float64
}

func MakeMapFromWorkerToTheirRegret(regrets *emissions.WorkerRegrets) *RegretsByWorkerByType {
	inferenceRegrets := make(map[string]*float64)
	forecastRegrets := make(map[string]*float64)
	for _, regret := range regrets.WorkerRegrets {
		inferenceRegrets[regret.Worker] = &regret.InferenceRegret
		forecastRegrets[regret.Worker] = &regret.ForecastRegret
	}
	return &RegretsByWorkerByType{
		InferenceRegrets: &inferenceRegrets,
		ForecastRegrets:  &forecastRegrets,
	}
}

func FindMaxRegret(regrets *RegretsByWorkerByType, epsilon float64) float64 {
	// Find the maximum regret admitted by any worker for an inference or forecast task; used in calculating the network combined inference
	maxPreviousRegret := epsilon // averts div by 0 error
	for _, regret := range *regrets.InferenceRegrets {
		// If inference regret is not null, use it to calculate the maximum regret
		if maxPreviousRegret < *regret && *regret > epsilon {
			maxPreviousRegret = *regret
		}
	}
	for _, regret := range *regrets.ForecastRegrets {
		// If forecast regret is not null, use it to calculate the maximum regret
		if maxPreviousRegret < *regret && *regret > epsilon {
			maxPreviousRegret = *regret
		}
	}
	return maxPreviousRegret
}

// Calculates network combined inference I_i, network per worker regret R_i-1,l, and weights w_il from the litepaper:
// I_i = Σ_l w_il I_il / Σ_l w_il
// w_il = φ'_p(\hatR_i-1,l)
// \hatR_i-1,l = R_i-1,l / |max_{l'}(R_i-1,l')|
// given inferences, forecast-implied inferences, and network regrets
func CalcWeightedInference(
	inferenceByWorker map[string]*emissions.Inference,
	forecastImpliedInferenceByWorker map[string]*emissions.Inference,
	regrets *RegretsByWorkerByType,
	epsilon float64,
	pInferenceSynthesis float64) (float64, error) {
	// Find the maximum regret admitted by any worker for an inference or forecast task; used in calculating the network combined inference
	maxPreviousRegret := FindMaxRegret(regrets, epsilon)

	// Calculate the network combined inference and network worker regrets
	unnormalizedI_i := float64(0)
	sumWeights := 0.0
	for worker, regret := range *regrets.InferenceRegrets {
		// normalize worker regret then calculate gradient => weight per worker for network combined inference
		weight, err := Gradient(pInferenceSynthesis, *regret/maxPreviousRegret)
		if err != nil {
			fmt.Println("Error calculating gradient: ", err)
			return 0, err
		}
		unnormalizedI_i += weight * inferenceByWorker[worker].Value // numerator of network combined inference calculation
		sumWeights += weight
	}
	for worker, regret := range *regrets.ForecastRegrets {
		weight, err := Gradient(pInferenceSynthesis, *regret/maxPreviousRegret)
		if err != nil {
			fmt.Println("Error calculating gradient: ", err)
			return 0, err
		}
		unnormalizedI_i += weight * forecastImpliedInferenceByWorker[worker].Value // numerator of network combined inference calculation
		sumWeights += weight
	}

	// Normalize the network combined inference
	if sumWeights < epsilon {
		return 0, emissions.ErrSumWeightsLessThanEta
	}
	return unnormalizedI_i / sumWeights, nil // divide numerator by denominator to get network combined inference
}

func CalcNaiveInference(
	inferences map[string]*emissions.Inference,
	regrets *RegretsByWorkerByType,
	epsilon float64,
) (float64, error) {
	// Update regrets to remove forecast regrets
	newRegrets := RegretsByWorkerByType{
		InferenceRegrets: regrets.InferenceRegrets,
		ForecastRegrets:  nil,
	}
	return CalcWeightedInference(inferences, nil, &newRegrets, epsilon, 0)
}

// Returns all one-out inferences that are possible given the provided input
// Assumed that there is at most 1 inference per worker. Also that there is at most 1 forecast-implied inference per worker.
// Loop over all inferences and forecast-implied inferences and withold one inference. Then calculate the network inference less that witheld inference
// If an inference is held out => recalculate the forecast-implied inferences before calculating the network inference
func CalcOneOutInferences(
	inferenceByWorker map[string]*emissions.Inference,
	forecastImpliedInferenceByWorker map[string]*emissions.Inference,
	forecasts *emissions.Forecasts,
	regrets *RegretsByWorkerByType,
	networkCombinedInference float64,
	epsilon float64,
	pInferenceSynthesis float64,
) ([]*emissions.WithheldWorkerAttributedValue, []*emissions.WithheldWorkerAttributedValue, error) {
	// Loop over inferences and reclculate forecast-implied inferences before calculating the network inference
	oneOutInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for worker := range inferenceByWorker {
		// Remove the inference of the worker from the inferences
		inferencesWithoutWorker := make(map[string]*emissions.Inference)

		for workerOfInference, inference := range inferenceByWorker {
			if workerOfInference != worker {
				inferencesWithoutWorker[workerOfInference] = inference
			}
		}
		// Recalculate the forecast-implied inferences without the worker's inference
		forecastImpliedInferencesWithoutWorkerByWorker, err := CalcForcastImpliedInferences(inferencesWithoutWorker, forecasts, networkCombinedInference, epsilon, pInferenceSynthesis)
		if err != nil {
			fmt.Println("Error calculating forecast-implied inferences for held-out inference: ", err)
			return nil, nil, err
		}

		oneOutNetworkInferenceWithoutInferer, err := CalcWeightedInference(inferenceByWorker, forecastImpliedInferencesWithoutWorkerByWorker, regrets, epsilon, pInferenceSynthesis)
		if err != nil {
			fmt.Println("Error calculating one-out inference: ", err)
			return nil, nil, err
		}

		oneOutInferences = append(oneOutInferences, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  oneOutNetworkInferenceWithoutInferer,
		})
	}

	// Loop over forecast-implied inferences and set it as the only forecast-implied inference one at a time, then calculate the network inference given that one held out
	oneOutImpliedInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for worker := range forecastImpliedInferenceByWorker {
		// Remove the inference of the worker from the inferences
		impliedInferenceWithoutWorker := make(map[string]*emissions.Inference)

		for workerOfImpliedInference, inference := range inferenceByWorker {
			if workerOfImpliedInference != worker {
				impliedInferenceWithoutWorker[workerOfImpliedInference] = inference
			}
		}

		// Calculate the network inference without the worker's inference
		oneOutInference, err := CalcWeightedInference(inferenceByWorker, impliedInferenceWithoutWorker, regrets, epsilon, pInferenceSynthesis)
		if err != nil {
			fmt.Println("Error calculating one-out inference: ", err)
			return nil, nil, err
		}
		oneOutImpliedInferences = append(oneOutImpliedInferences, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  oneOutInference,
		})
	}

	return oneOutInferences, oneOutImpliedInferences, nil
}

// Returns all one-in inferences that are possible given the provided input
// Assumed that there is at most 1 inference per worker. Also that there is at most 1 forecast-implied inference per worker.
func CalcOneInInferences(
	inferences map[string]*emissions.Inference,
	forecastImpliedInferences map[string]*emissions.Inference,
	regrets *RegretsByWorkerByType,
	epsilon float64,
	pInferenceSynthesis float64,
) ([]*emissions.WorkerAttributedValue, error) {
	// Loop over all forecast-implied inferences and set it as the only forecast-implied inference one at a time, then calculate the network inference given that one held out
	oneInInferences := make([]*emissions.WorkerAttributedValue, 0)
	for worker := range forecastImpliedInferences {
		// In each loop, remove all forecast-implied inferences except one
		forecastImpliedInferencesWithoutWorker := make(map[string]*emissions.Inference)
		for k, v := range forecastImpliedInferences {
			if k == worker {
				forecastImpliedInferencesWithoutWorker[k] = v
				break
			}
		}
		// Calculate the network inference without the worker's forecast-implied inference
		oneInInference, err := CalcWeightedInference(inferences, forecastImpliedInferencesWithoutWorker, regrets, epsilon, pInferenceSynthesis)
		if err != nil {
			fmt.Println("Error calculating one-in inference: ", err)
			return nil, err
		}
		oneInInferences = append(oneInInferences, &emissions.WorkerAttributedValue{
			Worker: worker,
			Value:  oneInInference,
		})
	}
	return oneInInferences, nil
}

// Calculates all network inferences in I_i given inferences, forecast implied inferences, and network combined inference.
// I_ij are the inferences of worker j and already given as an argument.
func CalcNetworkInferences(
	inferences *emissions.Inferences,
	forecasts *emissions.Forecasts,
	regrets *emissions.WorkerRegrets,
	networkCombinedLoss float64,
	epsilon float64,
	pInferenceSynthesis float64,
) (*emissions.ValueBundle, error) {
	// Map each worker to their inference
	inferenceByWorker := MakeMapFromWorkerToTheirWork(inferences.Inferences)
	// Calculate forecast-implied inferences I_ik
	forecastImpliedInferenceByWorker, err := CalcForcastImpliedInferences(inferenceByWorker, forecasts, networkCombinedLoss, epsilon, pInferenceSynthesis)
	if err != nil {
		fmt.Println("Error calculating forecast-implied inferences: ", err)
		return nil, err
	}

	// Map each worker to their regrets
	regretsWorkerByType := MakeMapFromWorkerToTheirRegret(regrets)

	// Calculate the combined network inference I_i
	combinedNetworkInference, err := CalcWeightedInference(inferenceByWorker, forecastImpliedInferenceByWorker, regretsWorkerByType, epsilon, pInferenceSynthesis)
	if err != nil {
		fmt.Println("Error calculating network combined inference: ", err)
		return nil, err
	}

	// Calculate the naive inference I^-_i
	naiveInference, err := CalcNaiveInference(inferenceByWorker, regretsWorkerByType, epsilon)
	if err != nil {
		fmt.Println("Error calculating naive inference: ", err)
		return nil, err
	}

	// Calculate the one-out inference I^-_li
	oneOutInferences, oneOutImpliedInferences, err := CalcOneOutInferences(inferenceByWorker, forecastImpliedInferenceByWorker, forecasts, regretsWorkerByType, combinedNetworkInference, epsilon, pInferenceSynthesis)
	if err != nil {
		fmt.Println("Error calculating one-out inferences: ", err)
		return nil, err
	}

	// Calculate the one-in inference I^+_ki
	oneInInferences, err := CalcOneInInferences(inferenceByWorker, forecastImpliedInferenceByWorker, regretsWorkerByType, epsilon, pInferenceSynthesis)
	if err != nil {
		fmt.Println("Error calculating one-in inferences: ", err)
		return nil, err
	}

	// Build value bundle to return all the calculated inferences
	// Shouldn't need inferences nor forecasts because given from context (input arguments)
	return &emissions.ValueBundle{
		CombinedValue:          combinedNetworkInference,
		NaiveValue:             naiveInference,
		OneOutInfererValues:    oneOutInferences,
		OneOutForecasterValues: oneOutImpliedInferences,
		OneInForecasterValues:  oneInInferences,
	}, nil
}

func StakeWeightedSumOfCombinedAndNaiveLosses(
	stakesByReputer map[string]float64,
	reputerReportedLosses *emissions.ReputerValueBundles,
) (float64, float64, error) {
	weightedCombinedSum := 0.0
	weightedNaiveSum := 0.0
	weightSum := 0.0
	for _, value := range reputerReportedLosses.ReputerValueBundles {
		if value.ValueBundle != nil {
			weight := stakesByReputer[value.Reputer]
			weightedCombinedSum += math.Log10(value.ValueBundle.CombinedValue) * stakesByReputer[value.Reputer]
			weightedNaiveSum += math.Log10(value.ValueBundle.NaiveValue) * stakesByReputer[value.Reputer]
			weightSum += weight
		}
	}
	if weightSum == 0 {
		return 0, 0, emissions.ErrFractionDivideByZero
	}
	combinedFraction := weightedCombinedSum / weightSum
	naiveFraction := weightedNaiveSum / weightSum
	return combinedFraction, naiveFraction, nil
}

type WorkerRunningWeightedLoss struct {
	SumWeight float64
	Loss      float64
}

// Update the running weighted loss for the worker
// Source: "Weighted mean" section of: https://fanf2.user.srcf.net/hermes/doc/antiforgery/stats.pdf
func runningWeightedAvgUpdate(
	runningWeightedAvg *WorkerRunningWeightedLoss,
	weight float64,
	nextValue float64,
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
func ConvertMapOfRunningWeightedLossesToWorkerAttributedValue(
	runningWeightedLosses map[string]*WorkerRunningWeightedLoss,
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
func ConvertMapOfRunningWeightedLossesToWithheldWorkerAttributedValue(
	runningWeightedLosses map[string]*WorkerRunningWeightedLoss,
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

type HigherOrderNetworkLosses struct {
	infererLosses          []*emissions.WorkerAttributedValue
	forecasterLosses       []*emissions.WorkerAttributedValue
	oneOutInfererLosses    []*emissions.WithheldWorkerAttributedValue
	oneOutForecasterLosses []*emissions.WithheldWorkerAttributedValue
	oneInForecasterLosses  []*emissions.WorkerAttributedValue
}

func StakeWeightedSumOfLogInfererLosses(
	stakesByReputer map[string]float64,
	reputerReportedLosses *emissions.ReputerValueBundles,
	epsilon float64,
) (HigherOrderNetworkLosses, error) {
	// Make map from inferer to their running weighted-average loss
	runningWeightedInfererLosses := make(map[string]*WorkerRunningWeightedLoss)
	runningWeightedForecasterLosses := make(map[string]*WorkerRunningWeightedLoss)
	runningWeightedOneOutInfererLosses := make(map[string]*WorkerRunningWeightedLoss) // Withheld worker -> Forecaster -> Loss
	runningWeightedOneOutForecasterLosses := make(map[string]*WorkerRunningWeightedLoss)
	runningWeightedOneInForecasterLosses := make(map[string]*WorkerRunningWeightedLoss)

	for _, report := range reputerReportedLosses.ReputerValueBundles {
		if report.ValueBundle != nil {
			// Not all reputers may have reported losses on the same set of inferers => important that the code below doesn't assume that!
			// Update inferer losses
			for _, loss := range report.ValueBundle.InfererValues {
				nextAvg, err := runningWeightedAvgUpdate(runningWeightedInfererLosses[loss.Worker], stakesByReputer[report.Reputer], loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for inferer: ", err)
					return HigherOrderNetworkLosses{}, err
				}
				runningWeightedInfererLosses[loss.Worker] = &nextAvg
			}

			// Update forecaster losses
			for _, loss := range report.ValueBundle.ForecasterValues {
				nextAvg, err := runningWeightedAvgUpdate(runningWeightedForecasterLosses[loss.Worker], stakesByReputer[report.Reputer], loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for forecaster: ", err)
					return HigherOrderNetworkLosses{}, err
				}
				runningWeightedForecasterLosses[loss.Worker] = &nextAvg
			}

			// Update one-out inferer losses
			for _, loss := range report.ValueBundle.OneOutInfererValues {
				nextAvg, err := runningWeightedAvgUpdate(runningWeightedOneOutInfererLosses[loss.Worker], stakesByReputer[report.Reputer], loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for one-out inferer: ", err)
					return HigherOrderNetworkLosses{}, err
				}
				runningWeightedOneOutInfererLosses[loss.Worker] = &nextAvg
			}

			// Update one-out forecaster losses
			for _, loss := range report.ValueBundle.OneOutForecasterValues {
				nextAvg, err := runningWeightedAvgUpdate(runningWeightedOneOutForecasterLosses[loss.Worker], stakesByReputer[report.Reputer], loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for one-out forecaster: ", err)
					return HigherOrderNetworkLosses{}, err
				}
				runningWeightedOneOutForecasterLosses[loss.Worker] = &nextAvg
			}

			// Update one-in forecaster losses
			for _, loss := range report.ValueBundle.OneOutForecasterValues {
				nextAvg, err := runningWeightedAvgUpdate(runningWeightedOneInForecasterLosses[loss.Worker], stakesByReputer[report.Reputer], loss.Value, epsilon)
				if err != nil {
					fmt.Println("Error updating running weighted average for one-in forecaster: ", err)
					return HigherOrderNetworkLosses{}, err
				}
				runningWeightedOneInForecasterLosses[loss.Worker] = &nextAvg
			}
		}
	}

	// Convert the running weighted averages to WorkerAttributedValue for inferers and forecasters
	output := HigherOrderNetworkLosses{
		infererLosses:          ConvertMapOfRunningWeightedLossesToWorkerAttributedValue(runningWeightedInfererLosses),
		forecasterLosses:       ConvertMapOfRunningWeightedLossesToWorkerAttributedValue(runningWeightedForecasterLosses),
		oneOutInfererLosses:    ConvertMapOfRunningWeightedLossesToWithheldWorkerAttributedValue(runningWeightedOneOutInfererLosses),
		oneOutForecasterLosses: ConvertMapOfRunningWeightedLossesToWithheldWorkerAttributedValue(runningWeightedOneOutForecasterLosses),
		oneInForecasterLosses:  ConvertMapOfRunningWeightedLossesToWorkerAttributedValue(runningWeightedOneInForecasterLosses),
	}

	return output, nil
}

func CalcNetworkLosses(
	stakesByReputer map[string]float64, //*cosmosMath.Uint,
	reputerReportedLosses *emissions.ReputerValueBundles,
	epsilon float64,
) (*emissions.ValueBundle, error) {
	combinedNetworkLoss, naiveNetworkLoss, err := StakeWeightedSumOfCombinedAndNaiveLosses(stakesByReputer, reputerReportedLosses)
	if err != nil {
		fmt.Println("Error calculating network losses: ", err)
		return nil, err
	}

	higherOrderLosses, err := StakeWeightedSumOfLogInfererLosses(stakesByReputer, reputerReportedLosses, epsilon)
	if err != nil {
		fmt.Println("Error calculating network losses: ", err)
		return nil, err
	}

	return &emissions.ValueBundle{
		CombinedValue:          combinedNetworkLoss,
		InfererValues:          higherOrderLosses.infererLosses,
		ForecasterValues:       higherOrderLosses.forecasterLosses,
		NaiveValue:             naiveNetworkLoss,
		OneOutInfererValues:    higherOrderLosses.oneOutInfererLosses,
		OneOutForecasterValues: higherOrderLosses.oneOutForecasterLosses,
		OneInForecasterValues:  higherOrderLosses.oneInForecasterLosses,
	}, nil
}

// // Build a value bundle of network regrets from the provided network losses
// func GetNetworkRegretsOfWorkersWithNetworkLosses(
// 	ctx sdk.Context,
// 	k keeper.Keeper,
// 	topicId string,
// 	networkLosses *emissions.ValueBundle,
// ) (*emissions.ValueBundle, error) {
// 	// Calculate the network regrets
// 	infererRegrets := make([]*emissions.WorkerAttributedValue, 0)
// 	forecasterRegrets := make([]*emissions.WorkerAttributedValue, 0)
// 	oneOutInfererRegrets := make([]*emissions.WorkerAttributedValue, 0)
// 	oneOutForecasterRegrets := make([]*emissions.WorkerAttributedValue, 0)
// 	oneInForecasterRegrets := make([]*emissions.WorkerAttributedValue, 0)

// 	// Get the network regrets for the workers represented in the inputted network losses
// 	for _, inferer := range networkLosses.InfererValues {
// 		lastRegret, err := k.GetInferenceRegret(ctx, topicId, inferer.Worker)
// 		if err != nil {
// 			fmt.Println("Error getting inference regret: ", err)
// 			return nil, err
// 		}
// 		infererRegrets = append(infererRegrets, &emissions.WorkerAttributedValue{
// 			Worker: inferer.Worker,
// 			Value:  lastRegret,
// 		})
// 	}

// 	for _, forecaster := range networkLosses.ForecasterValues {
// 		lastRegret, err := k.GetForecastRegret(ctx, topicId, forecaster.Worker)
// 		if err != nil {
// 			fmt.Println("Error getting forecast regret: ", err)
// 			return nil, err
// 		}
// 		forecasterRegrets = append(forecasterRegrets, &emissions.WorkerAttributedValue{
// 			Worker: forecaster.Worker,
// 			Value:  lastRegret,
// 		})
// 	}

// 	for _, oneOutInferer := range networkLosses.OneOutInfererValues {
// 		lastRegret, err := k.GetOneOutInferenceRegret(ctx, topicId, oneOutInferer.Worker)
// 		if err != nil {
// 			fmt.Println("Error getting one-out inference regret: ", err)
// 			return nil, err
// 		}
// 		oneOutInfererRegrets = append(oneOutInfererRegrets, &emissions.WorkerAttributedValue{
// 			Worker: oneOutInferer.Worker,
// 			Value:  lastRegret,
// 		})
// 	}

// 	for _, oneOutForecaster := range networkLosses.OneOutForecasterValues {
// 		lastRegret, err := k.GetOneOutForecastRegret(ctx, topicId, oneOutForecaster.Worker)
// 		if err != nil {
// 			fmt.Println("Error getting one-out forecast regret: ", err)
// 			return nil, err
// 		}
// 		oneOutForecasterRegrets = append(oneOutForecasterRegrets, &emissions.WorkerAttributedValue{
// 			Worker: oneOutForecaster.Worker,
// 			Value:  lastRegret,
// 		})
// 	}

// 	for _, oneInForecaster := range networkLosses.OneInNaiveValues {
// 		lastRegret, err := k.GetOneInForecastRegret(ctx, topicId, oneInForecaster.Worker)
// 		if err != nil {
// 			fmt.Println("Error getting one-in forecast regret: ", err)
// 			return nil, err
// 		}
// 		oneInForecasterRegrets = append(oneInForecasterRegrets, &emissions.WorkerAttributedValue{
// 			Worker: oneInForecaster.Worker,
// 			Value:  lastRegret,
// 		})
// 	}

// 	// Get the latest network regrets for the topic
// 	/**
// 	 * TODO
// 	 * Implement code to get network regrets per worker
// 	 * Implement keeper functions above
// 	 * Add extra loop to each of the above?
// 	 * Combine with function below to lower runtime
// 	 */

// 	return &emissions.ValueBundle{
// 		CombinedValue:          networkLosses.CombinedValue,
// 		InfererValues:          infererRegrets,
// 		ForecasterValues:       forecasterRegrets,
// 		NaiveValue:             networkLosses.NaiveValue,
// 		OneOutInfererValues:    oneOutInfererRegrets,
// 		OneOutForecasterValues: oneOutForecasterRegrets,
// 		OneInNaiveValues:       oneInForecasterRegrets,
// 	}, nil
// }

// // Calculate the new network regrets by taking EMAs between the previous network regrets
// // and the new regrets admitted by the inputted network losses
// func CalcNetworkRegrets(
// 	currentNetworkRegrets *emissions.ValueBundle,
// 	networkLosses *emissions.ValueBundle,
// 	alpha float64,
// ) (*emissions.ValueBundle, error) {
// 	// Calculate the network regrets
// 	infererRegrets := make([]*emissions.WorkerAttributedValue, 0)
// 	forecasterRegrets := make([]*emissions.WorkerAttributedValue, 0)
// 	oneOutInfererRegrets := make([]*emissions.WorkerAttributedValue, 0)
// 	oneOutForecasterRegrets := make([]*emissions.WorkerAttributedValue, 0)
// 	oneInForecasterRegrets := make([]*emissions.WorkerAttributedValue, 0)

// 	// Calculate the new network regrets
// 	for i, inferer := range currentNetworkRegrets.InfererValues {
// 		newRegret := (1.0-alpha)*inferer.Value + alpha*networkLosses.InfererValues[i].Value
// 		infererRegrets = append(infererRegrets, &emissions.WorkerAttributedValue{
// 			Worker: inferer.Worker,
// 			Value:  newRegret,
// 		})
// 	}

// 	for i, forecaster := range currentNetworkRegrets.ForecasterValues {
// 		newRegret := (1.0-alpha)*forecaster.Value + alpha*networkLosses.ForecasterValues[i].Value
// 		forecasterRegrets = append(forecasterRegrets, &emissions.WorkerAttributedValue{
// 			Worker: forecaster.Worker,
// 			Value:  newRegret,
// 		})
// 	}

// 	for i, oneOutInferer := range currentNetworkRegrets.OneOutInfererValues {
// 		newRegret := (1.0-alpha)*oneOutInferer.Value + alpha*networkLosses.OneOutInfererValues[i].Value
// 		oneOutInfererRegrets = append(oneOutInfererRegrets, &emissions.WorkerAttributedValue{
// 			Worker: oneOutInferer.Worker,
// 			Value:  newRegret,
// 		})
// 	}

// 	for i, oneOutForecaster := range currentNetworkRegrets.OneOutForecasterValues {
// 		newRegret := (1.0-alpha)*oneOutForecaster.Value + alpha*networkLosses.OneOutForecasterValues[i].Value
// 		oneOutForecasterRegrets = append(oneOutForecasterRegrets, &emissions.WorkerAttributedValue{
// 			Worker: oneOutForecaster.Worker,
// 			Value:  newRegret,
// 		})
// 	}

// 	for i, oneInForecaster := range currentNetworkRegrets.OneInNaiveValues {
// 		newRegret := (1.0-alpha)*oneInForecaster.Value + alpha*networkLosses.OneInNaiveValues[i].Value
// 		oneInForecasterRegrets = append(oneInForecasterRegrets, &emissions.WorkerAttributedValue{
// 			Worker: oneInForecaster.Worker,
// 			Value:  newRegret,
// 		})
// 	}

// 	return &emissions.ValueBundle{
// 		CombinedValue:          alpha*currentNetworkRegrets.CombinedValue + (1-alpha)*networkLosses.CombinedValue,
// 		InfererValues:          infererRegrets,
// 		ForecasterValues:       forecasterRegrets,
// 		NaiveValue:             alpha*currentNetworkRegrets.NaiveValue + (1-alpha)*networkLosses.NaiveValue,
// 		OneOutInfererValues:    oneOutInfererRegrets,
// 		OneOutForecasterValues: oneOutForecasterRegrets,
// 		OneInNaiveValues:       oneInForecasterRegrets,
// 	}, nil
// }

/// TODO ensure root functions check for uniqueness of inferer and forecaster workers in the input
/// => Every sub function need not care

/// !! TODO: Apply the following functions to the module + Revamp them as needed to exhibit the proper I/O interface !!

// // Gathers inferences and forecasts from the worker update at or just after the given time, and
// // losses from the loss update just before the given time.
// // Then invokes calculation of the forecast-implied inferences using the Inference Synthesis formula from the litepaper.
// func GetAndCalcForcastImpliedInferencesAtBlock(
// 	ctx sdk.Context,
// 	k keeper.Keeper,
// 	topicId TopicId,
// 	blockHeight BlockHeight) ([]*emissions.Inference, error) {
// 	// Get inferences from worker update at or just after the given time
// 	inferences, err := k.GetInferencesAtOrAfterBlock(ctx, topicId, blockHeight)
// 	if err != nil {
// 		fmt.Println("Error getting inferences: ", err)
// 		return nil, err
// 	}
// 	// Get forecasts from worker update at or just after the given time
// 	forecasts, err := k.GetForecastsAtOrAfterBlock(ctx, topicId, blockHeight)
// 	if err != nil {
// 		fmt.Println("Error getting forecasts: ", err)
// 		return nil, err
// 	}
// 	// Get losses from loss update just before the given time
// 	networkValueBundle, err := k.GetNetworkValueBundleAtOrBeforeBlock(ctx, topicId, blockHeight)
// 	if err != nil {
// 		fmt.Println("Error getting network losses: ", err)
// 		return nil, err
// 	}

// 	epsilon, err := k.GetParamsEpsilon(ctx)
// 	if err != nil {
// 		fmt.Println("Error getting epsilon: ", err)
// 		return nil, err
// 	}

// 	pInferenceSynthesis, err := k.GetParamsPInferenceSynthesis(ctx)
// 	if err != nil {
// 		fmt.Println("Error getting epsilon: ", err)
// 		return nil, err
// 	}

// 	return CalcForcastImpliedInferences(inferences, forecasts, networkValueBundle, epsilon, pInferenceSynthesis)
// }

// // Gathers inferences, forecast-implied inferences as of the given block
// // and the network regrets admitted by workers at or just before the given time,
// // then invokes calculation of network combined inference I_i, network per worker regret R_i-1,l, and weights w_il from the litepaper.
// func GetAndCalcNetworkCombinedInference(
// 	ctx sdk.Context,
// 	k keeper.Keeper,
// 	topicId TopicId,
// 	blockHeight BlockHeight) (float64, error) {
// 	// Get inferences from worker update at or just after the given block
// 	inferences, err := k.GetInferencesAtOrAfterBlock(ctx, topicId, blockHeight)
// 	if err != nil {
// 		fmt.Println("Error getting inferences: ", err)
// 		return 0, err
// 	}
// 	// Map each worker to their inference
// 	inferenceByWorker := MakeMapFromWorkerToTheirWork(inferences.Inferences)

// 	// Get regrets admitted by workers just before the given block
// 	regrets, err := k.GetNetworkRegretsAtOrBeforeBlock(ctx, topicId, blockHeight)
// 	if err != nil {
// 		fmt.Println("Error getting network regrets: ", err)
// 		return 0, err
// 	}
// 	// Get forecast-implied inferences at the given block
// 	forecastImpliedInferences, err := GetAndCalcForcastImpliedInferencesAtBlock(ctx, k, topicId, blockHeight)
// 	if err != nil {
// 		fmt.Println("Error getting forecast implied inferences: ", err)
// 		return 0, err
// 	}
// 	// Map each worker to their forecast-implied inference
// 	forecastImpliedInferenceByWorker := MakeMapFromWorkerToTheirWork(forecastImpliedInferences)

// 	epsilon, err := k.GetParamsEpsilon(ctx)
// 	if err != nil {
// 		fmt.Println("Error getting epsilon: ", err)
// 		return 0, err
// 	}

// 	pInferenceSynthesis, err := k.GetParamsPInferenceSynthesis(ctx)
// 	if err != nil {
// 		fmt.Println("Error getting epsilon: ", err)
// 		return 0, err
// 	}

// 	return CalcWeightedInference(inferenceByWorker, forecastImpliedInferenceByWorker, regrets, epsilon, pInferenceSynthesis)
// }
