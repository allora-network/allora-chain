package module

import (
	"fmt"
	"math"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

// Calculate the forecast-implied inferences I_ik given inferences, forecasts and network losses.
// Calculates R_ijk, w_ijk, and I_ik for each forecast k and forecast element (forcast of worker loss) j
//
// Forecast without inference => weight in calculation of I_ik and I_i set to 0. Use latest available regret R_i-1,l
// Inference without forecast => only weight in calculation of I_ik set to 0
func CalcForcastImpliedInferencesAtTime(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	inferences *emissions.Inferences,
	forecasts *emissions.Forecasts,
	networkValueBundle *emissions.ValueBundle,
	epsilon float64,
	pInferenceSynthesis float64) ([]emissions.Inference, error) {
	// Map each worker to the inference they submitted
	inferenceByWorker := make(map[string]*emissions.Inference)
	for _, inference := range inferences.Inferences {
		inferenceByWorker[inference.Worker] = inference
	}
	// Possibly add a small value to previous network loss avoid infinite logarithm
	networkCombinedLoss := networkValueBundle.CombinedValue
	if networkCombinedLoss == 0 {
		// Take max of epsilon and 1 to avoid division by 0
		networkCombinedLoss = math.Max(epsilon, 1)
	}

	// "k" here is the index of the forecaster's report among many reports
	// For each forecast, and for each forecast element, calculate forecast-implied inferences I_ik
	I_i := make([]emissions.Inference, len(forecasts.Forecasts))
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
			I_i[k] = emissions.Inference{
				Worker: forecast.Forecaster,
				Value:  weightInferenceDotProduct / weightSum,
			}
		}
	}

	// A value of 0 => no inference corresponded to any of the forecasts from a forecaster
	return I_i, nil
}

// Gathers inferences and forecasts from the worker update at or just after the given time, and
// losses from the loss update just before the given time.
// Then invokes calculation of the forecast-implied inferences using the Inference Synthesis formula from the litepaper.
func GetAndCalcForcastImpliedInferencesAtBlock(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	blockHeight BlockHeight) ([]emissions.Inference, error) {
	// Get inferences from worker update at or just after the given time
	inferences, err := k.GetInferencesAtOrAfterBlock(ctx, topicId, blockHeight)
	if err != nil {
		fmt.Println("Error getting inferences: ", err)
		return nil, err
	}
	// Get forecasts from worker update at or just after the given time
	forecasts, err := k.GetForecastsAtOrAfterBlock(ctx, topicId, blockHeight)
	if err != nil {
		fmt.Println("Error getting forecasts: ", err)
		return nil, err
	}
	// Get losses from loss update just before the given time
	networkValueBundle, err := k.GetNetworkValueBundleAtOrBeforeBlock(ctx, topicId, blockHeight)
	if err != nil {
		fmt.Println("Error getting network losses: ", err)
		return nil, err
	}

	epsilon, err := k.GetParamsEpsilon(ctx)
	if err != nil {
		fmt.Println("Error getting epsilon: ", err)
		return nil, err
	}

	pInferenceSynthesis, err := k.GetParamsPInferenceSynthesis(ctx)
	if err != nil {
		fmt.Println("Error getting epsilon: ", err)
		return nil, err
	}

	return CalcForcastImpliedInferencesAtTime(ctx, k, topicId, inferences, forecasts, networkValueBundle, epsilon, pInferenceSynthesis)
}

// Calculates network combined inference I_i, network per worker regret R_i-1,l, and weights w_il from the litepaper:
// I_i = Σ_l w_il I_il / Σ_l w_il
// w_il = φ'_p(\hatR_i-1,l)
// \hatR_i-1,l = R_i-1,l / |max_{l'}(R_i-1,l')|
// given inferences, forecast-implied inferences, and network regrets
func CalcNetworkCombinedInference(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	inferences *emissions.Inferences,
	forecastImpliedInferences []emissions.Inference,
	regrets *emissions.WorkerRegrets,
	epsilon float64,
	pInferenceSynthesis float64) (float64, error) {
	// Map each worker to their inference
	inferenceByWorker := make(map[string]*emissions.Inference)
	for _, inference := range inferences.Inferences {
		inferenceByWorker[inference.Worker] = inference
	}
	// Map each worker to their forecast-implied inference
	forecastImpliedInferenceByWorker := make(map[string]*emissions.Inference)
	for _, inference := range forecastImpliedInferences {
		forecastImpliedInferenceByWorker[inference.Worker] = &inference
	}

	// Find the maximum regret admitted by any worker for an inference or forecast task; used in calculating the network combined inference
	maxPreviousRegret := epsilon // averts div by 0 error
	for _, regret := range regrets.WorkerRegrets {
		// If inference regret is not null, use it to calculate the maximum regret
		if maxPreviousRegret < regret.InferenceRegret && float64(regret.InferenceRegret) > epsilon {
			maxPreviousRegret = regret.InferenceRegret
		}
		// If forecast regret is not null, use it to calculate the maximum regret
		if maxPreviousRegret < regret.ForecastRegret && float64(regret.ForecastRegret) > epsilon {
			maxPreviousRegret = regret.ForecastRegret
		}
	}

	// Calculate the network combined inference and network worker regrets
	unnormalizedI_i := float64(0)
	sumWeights := 0.0
	for _, regret := range regrets.WorkerRegrets {
		// normalize worker regret then calculate gradient => weight per worker for network combined inference
		weight, err := Gradient(pInferenceSynthesis, float64(regret.InferenceRegret/maxPreviousRegret))
		if err != nil {
			fmt.Println("Error calculating gradient: ", err)
			return float64(0), err
		}
		unnormalizedI_i += weight * inferenceByWorker[regret.Worker].Value // numerator of network combined inference calculation
		sumWeights += weight
	}

	// Normalize the network combined inference
	if sumWeights < epsilon {
		return 0, emissions.ErrSumWeightsLessThanEta
	}
	return unnormalizedI_i / sumWeights, nil // divide numerator by denominator to get network combined inference
}

// Gathers inferences, forecast-implied inferences as of the given block
// and the network regrets admitted by workers at or just before the given time,
// then invokes calculation of network combined inference I_i, network per worker regret R_i-1,l, and weights w_il from the litepaper.
func GetAndCalcNetworkCombinedInference(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	blockHeight BlockHeight) (float64, error) {
	// Get inferences from worker update at or just after the given block
	inferences, err := k.GetInferencesAtOrAfterBlock(ctx, topicId, blockHeight)
	if err != nil {
		fmt.Println("Error getting inferences: ", err)
		return float64(0), err
	}
	// Get regrets admitted by workers just before the given block
	regrets, err := k.GetNetworkRegretsAtOrBeforeBlock(ctx, topicId, blockHeight)
	if err != nil {
		fmt.Println("Error getting network regrets: ", err)
		return float64(0), err
	}
	// Get forecast-implied inferences at the given block
	forecastImpliedInferences, err := GetAndCalcForcastImpliedInferencesAtBlock(ctx, k, topicId, blockHeight)
	if err != nil {
		fmt.Println("Error getting forecast implied inferences: ", err)
		return float64(0), err
	}

	epsilon, err := k.GetParamsEpsilon(ctx)
	if err != nil {
		fmt.Println("Error getting epsilon: ", err)
		return float64(0), err
	}

	pInferenceSynthesis, err := k.GetParamsPInferenceSynthesis(ctx)
	if err != nil {
		fmt.Println("Error getting epsilon: ", err)
		return float64(0), err
	}

	return CalcNetworkCombinedInference(ctx, k, topicId, inferences, forecastImpliedInferences, regrets, epsilon, pInferenceSynthesis)
}

// Calculates all network inferences in I_i given inferences, forecast implied inferences, and network combined inference
func CalcNetworkInferences(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	inferences *emissions.Inferences,
	forecastImpliedInferences []emissions.Inference,
	networkCombinedInference float64,
) ([]emissions.Inference, error) {
	// TODO implement!
	return nil, nil
}
