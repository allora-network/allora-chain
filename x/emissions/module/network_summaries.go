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
func gradient(p float64, x float64) (float64, error) {
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

// Calculate the forecast-implied inferences I_ik given inferences, forecasts and network losses.
// Calculates R_ijk, w_ijk, and I_ik for each forecast k and forecast element (forcast of worker loss) j
//
// Forecast without inference => weight in calculation of I_ik and I_i set to 0. Use latest available regret R_i-1,l
// Inference without forecast => only weight in calculation of I_ik set to 0
func CalcForcastImpliedInferences(
	inferences *emissions.Inferences,
	forecasts *emissions.Forecasts,
	networkCombinedLoss float64,
	epsilon float64,
	pInferenceSynthesis float64) ([]*emissions.Inference, error) {
	// Map each worker to the inference they submitted
	inferenceByWorker := make(map[string]*emissions.Inference)
	for _, inference := range inferences.Inferences {
		inferenceByWorker[inference.Worker] = inference
	}
	// Possibly add a small value to previous network loss avoid infinite logarithm
	if networkCombinedLoss == 0 {
		// Take max of epsilon and 1 to avoid division by 0
		networkCombinedLoss = math.Max(epsilon, 1)
	}

	// "k" here is the index of the forecaster's report among many reports
	// For each forecast, and for each forecast element, calculate forecast-implied inferences I_ik
	I_i := make([]*emissions.Inference, len(forecasts.Forecasts))
	for k, forecast := range forecasts.Forecasts {
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
				w_ijk, err := gradient(pInferenceSynthesis, R_ik[j]) // w_ijk = φ'_p(\hatR_ijk)
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
			forecast := emissions.Inference{
				Worker: forecast.Forecaster,
				Value:  weightInferenceDotProduct / weightSum,
			}
			I_i[k] = &forecast
		}
	}

	// A value of 0 => no inference corresponded to any of the forecasts from a forecaster
	return I_i, nil
}

func MakeMapFromWorkerToTheirWork(inferences []*emissions.Inference) map[string]*emissions.Inference {
	inferencesByWorker := make(map[string]*emissions.Inference)
	for _, inference := range inferences {
		inferencesByWorker[inference.Worker] = inference
	}
	return inferencesByWorker
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
		weight, err := gradient(pInferenceSynthesis, *regret/maxPreviousRegret)
		if err != nil {
			fmt.Println("Error calculating gradient: ", err)
			return 0, err
		}
		unnormalizedI_i += weight * inferenceByWorker[worker].Value // numerator of network combined inference calculation
		sumWeights += weight
	}
	for worker, regret := range *regrets.ForecastRegrets {
		weight, err := gradient(pInferenceSynthesis, *regret/maxPreviousRegret)
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
func CalcOneOutInferences(
	inferences map[string]*emissions.Inference,
	forecastImpliedInferences map[string]*emissions.Inference,
	regrets *RegretsByWorkerByType,
	pInferenceSynthesis float64,
) ([]*emissions.Inference, error) {
	// Loop over all inferences and forecast-implied inferences and remove one at a time, then calculate the network inference given that one held out
	oneOutInferences := make([]*emissions.Inference, 0)
	for worker := range inferences {
		// Remove the inference of the worker from the inferences
		inferencesWithoutWorker := make(map[string]*emissions.Inference)
		for k, v := range inferences {
			if k != worker {
				inferencesWithoutWorker[k] = v
			}
		}
		// Calculate the network inference without the worker's inference
		oneOutInference, err := CalcWeightedInference(inferencesWithoutWorker, forecastImpliedInferences, regrets, 0, pInferenceSynthesis)
		if err != nil {
			fmt.Println("Error calculating one-out inference: ", err)
			return nil, err
		}
		oneOutInferences = append(oneOutInferences, &emissions.Inference{
			Worker: worker,
			Value:  oneOutInference,
		})
	}
	return oneOutInferences, nil
}

// Returns all one-in inferences that are possible given the provided input
// Assumed that there is at most 1 inference per worker. Also that there is at most 1 forecast-implied inference per worker.
func CalcOneInInferences(
	inferences map[string]*emissions.Inference,
	forecastImpliedInferences map[string]*emissions.Inference,
	regrets *RegretsByWorkerByType,
	epsilon float64,
	pInferenceSynthesis float64,
) ([]*emissions.Inference, error) {
	// Loop over all forecast-implied inferences and set it as the only forecast-implied inference one at a time, then calculate the network inference given that one held out
	oneInInferences := make([]*emissions.Inference, 0)
	for worker := range forecastImpliedInferences {
		// Remove the forecast-implied inference of the worker from the forecast-implied inferences
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
		oneInInferences = append(oneInInferences, &emissions.Inference{
			Worker: worker,
			Value:  oneInInference,
		})
	}
	return oneInInferences, nil
}

// This is an intermediary type used to return all the calculated inferences given
// inferences, forecasts, regrets, network combined loss, from other actors or calculated on-chain
type NetworkInferences struct {
	CombinedValue float64
	// InfererValues    *emissions.Inferences // Not needed
	ForecasterValues *emissions.Forecasts
	NaiveValue       float64
	OneOutValues     []*emissions.Inference
	OneInValues      []*emissions.Inference
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
) (*NetworkInferences, error) {
	// Calculate forecast-implied inferences I_ik
	forecastImpliedInferences, err := CalcForcastImpliedInferences(inferences, forecasts, networkCombinedLoss, epsilon, pInferenceSynthesis)
	if err != nil {
		fmt.Println("Error calculating forecast-implied inferences: ", err)
		return nil, err
	}
	// Map each worker to their inference
	inferenceByWorker := MakeMapFromWorkerToTheirWork(inferences.Inferences)
	// Map each worker to their forecast-implied inference
	forecastImpliedInferenceByWorker := MakeMapFromWorkerToTheirWork(forecastImpliedInferences)

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
	oneOutInferences, err := CalcOneOutInferences(inferenceByWorker, forecastImpliedInferenceByWorker, regretsWorkerByType, pInferenceSynthesis)
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
	return &NetworkInferences{
		CombinedValue:    combinedNetworkInference,
		ForecasterValues: forecasts,
		NaiveValue:       naiveInference,
		OneOutValues:     oneOutInferences,
		OneInValues:      oneInInferences,
	}, nil
}

/// !! TODO: Apply the following functions to the module + Revamp as needed !!

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
