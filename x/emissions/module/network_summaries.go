package module

import (
	"fmt"
	"math"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/**
 * This file contains the logic from Section 3 of the litepaper,
 * which churns current inferences, forecasts, previous losses and regrets
 * into current network losses and regrets
 */

const p = 2.0

const eta = 1

// Implements function phi prime from litepaper eq7
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

// Calculate the forecast-implied inferences at a given time using eq3-8 from the litepaper
// and the given inferences, forecasts and network losses.
//
// Forecast without inference => eq3,9 weight set to 0. Use latest available regret R_i-1,l
// Inference without forecast => eq3 weight only set to 0
func CalcForcastImpliedInferencesAtTime(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	inferences *emissions.Inferences,
	forecasts *emissions.Forecasts,
	networkLossBundle *emissions.LossBundle) ([]emissions.Inference, error) {
	// Map each worker to the inference they submitted
	inferenceByWorker := make(map[string]*emissions.Inference)
	for _, inference := range inferences.Inferences {
		inferenceByWorker[inference.Worker] = inference
	}
	// Possibly add a small value to previous network loss avoid infinite logarithm
	networkCombinedLoss := networkLossBundle.CombinedLoss.Uint64()
	if networkCombinedLoss == 0 {
		networkCombinedLoss = eta
	}

	// "k" here is the index of the forecaster's report among many reports
	// Forecast-implied inferences per forecaster; litepaper eq3
	I_i := make([]emissions.Inference, len(forecasts.Forecasts))
	for k, forecast := range forecasts.Forecasts {
		if len(forecast.ForecastElements) > 0 {
			// Filter away all forecast elements that do not have an associated inference (match by worker)
			// Will effectively set weight in eq3 and eq9 to 0 for forecasts without inferences
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
			maxjRijk := float64(forecastElementsWithInferences[0].Value.Uint64())
			for j, el := range forecastElementsWithInferences {
				// Calculate the approximate forecast regret of the network inference
				R_ik[j] = math.Log10(float64(networkCombinedLoss / el.Value.Uint64())) // eq4
				if R_ik[j] > maxjRijk {
					maxjRijk = R_ik[j]
				}
			}

			// Calculate normalized regrets per forecaster then weights per forecaster
			for j, _ := range forecastElementsWithInferences {
				R_ik[j] = R_ik[j] / maxjRijk       // eq8
				w_ijk, err := gradient(p, R_ik[j]) // eq5
				if err != nil {
					fmt.Println("Error calculating gradient: ", err)
					return nil, err
				}
				w_ik[j] = w_ijk
			}

			// Calculate the forecast implied inferences; eq3
			weightSum := 0.0
			weightInferenceDotProduct := 0.0
			for j, w_ijk := range w_ik {
				// Calculate the forecast implied inferences
				weightInferenceDotProduct += w_ijk * float64(inferenceByWorker[forecast.ForecastElements[j].Inferer].Value.Uint64())
				weightSum += w_ijk
			}
			I_i[k] = emissions.Inference{
				Worker: forecast.Forecaster,
				Value:  cosmosMath.NewUint(uint64(weightInferenceDotProduct / weightSum)),
			}
		}
	}

	// A value of 0 => no inference corresponded to any of the forecasts from a forecaster
	return I_i, nil
}

// Calculate the forecast-implied inferences at a given time using eq3-8 from the litepaper,
// inferences and forecasts from the worker update at or just after the given time, and
// losses from the loss update just before the given time.
func GetAndCalcForcastImpliedInferencesAtTime(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	time uint64) ([]emissions.Inference, error) {
	// Get inferences from worker update at or just after the given time
	inferences, err := k.GetInferencesAtOrAfterTime(ctx, topicId, time)
	if err != nil {
		fmt.Println("Error getting inferences: ", err)
		return nil, err
	}
	// Get forecasts from worker update at or just after the given time
	forecasts, err := k.GetForecastsAtOrAfterTime(ctx, topicId, time)
	if err != nil {
		fmt.Println("Error getting forecasts: ", err)
		return nil, err
	}
	// Get losses from loss update just before the given time
	networkLossBundle, err := k.GetNetworkLossBundleAtOrBeforeTime(ctx, topicId, time)
	if err != nil {
		fmt.Println("Error getting network losses: ", err)
		return nil, err
	}

	return CalcForcastImpliedInferencesAtTime(ctx, k, topicId, inferences, forecasts, networkLossBundle)
}

// Calculates eq9-11 from the litepaper using the given inferences, forecast-implied inferences, and network regrets
func CalcNetworkCombinedInference(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	inferences *emissions.Inferences,
	forecastImpliedInferences []emissions.Inference,
	regrets *emissions.WorkerRegrets) (float64, error) {
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

	// Find the maximum regret admitted by any worker for an inference or forecast task; used in eq11
	maxPreviousRegret := float32(eta) // averts div by 0 error
	for _, regret := range regrets.WorkerRegrets {
		// If inference regret is not null, use it to calculate the maximum regret
		if maxPreviousRegret < regret.InferenceRegret && regret.InferenceRegret > eta {
			maxPreviousRegret = regret.InferenceRegret
		}
		// If forecast regret is not null, use it to calculate the maximum regret
		if maxPreviousRegret < regret.ForecastRegret && regret.InferenceRegret > eta {
			maxPreviousRegret = regret.ForecastRegret
		}
	}

	// Calculate the network combined inference; eq9-11
	unnormalizedI_i := float64(0)
	sumWeights := 0.0
	for _, regret := range regrets.WorkerRegrets {
		weight, err := gradient(p, float64(regret.InferenceRegret/maxPreviousRegret)) // eq11 then eq10
		if err != nil {
			fmt.Println("Error calculating gradient: ", err)
			return float64(0), err
		}
		unnormalizedI_i += weight * float64(inferenceByWorker[regret.Worker].Value.Uint64()) // pt1/2 of eq9
		sumWeights += weight

		weight, err = gradient(p, float64(regret.InferenceRegret/maxPreviousRegret)) // eq11 then eq10
		if err != nil {
			fmt.Println("Error calculating gradient: ", err)
			return float64(0), err
		}
		unnormalizedI_i += weight * float64(forecastImpliedInferenceByWorker[regret.Worker].Value.Uint64()) // pt1/2 of eq9
		sumWeights += weight
	}

	// Normalize the network combined inference
	if sumWeights < eta {
		return 0, emissions.ErrSumWeightsLessThanEta
	}
	return unnormalizedI_i / sumWeights, nil // pt2/2 of eq9
}

// Calculates eq9-11 from the litepaper using the forecast-implied inferences as of the given time
// and the network regrets admitted by workers at or just before the given time.
func GetAndCalcNetworkCombinedInference(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	time uint64) (float64, error) {
	// Get inferences from worker update at or just after the given time
	inferences, err := k.GetInferencesAtOrAfterTime(ctx, topicId, time)
	if err != nil {
		fmt.Println("Error getting inferences: ", err)
		return float64(0), err
	}
	// Get regrets admitted by workers just before the given time
	regrets, err := k.GetNetworkRegretsAtOrBeforeTime(ctx, topicId, time)
	if err != nil {
		fmt.Println("Error getting network regrets: ", err)
		return float64(0), err
	}
	// Get forecast-implied inferences at the given time
	forecastImpliedInferences, err := GetAndCalcForcastImpliedInferencesAtTime(ctx, k, topicId, time)
	if err != nil {
		fmt.Println("Error getting forecast implied inferences: ", err)
		return float64(0), err
	}

	return CalcNetworkCombinedInference(ctx, k, topicId, inferences, forecastImpliedInferences, regrets)
}
