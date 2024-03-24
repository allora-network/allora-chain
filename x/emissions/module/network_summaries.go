package module

import (
	"fmt"
	"math"

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

// Calculate the forecast-implied inferences at a given time using eq3-8 from the litepaper,
// inferences and forecasts from the worker update at or just after the given time, and
// losses from the loss update just before the given time.
//
// Forecast without inference => eq3,9 weight set to 0. Use latest available regret R_i-1,l
// Inference without forecast => eq3 weight only set to 0
func CalcForcastImpliedInferencesAtTime(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	time uint64) ([]float64, error) {
	// Get inferences from worker update at or just after the given time
	inferences, err := k.GetInferencesAtOrAfterTime(ctx, topicId, time)
	if err != nil {
		fmt.Println("Error getting inferences: ", err)
		return nil, err
	}
	// Map each worker to the inference they submitted
	inferenceByWorker := make(map[string]*emissions.Inference)
	for _, inference := range inferences.Inferences {
		inferenceByWorker[inference.Worker] = inference
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
	// Possibly add a small value to previous network loss avoid infinite logarithm
	networkCombinedLoss := networkLossBundle.CombinedLoss.Uint64()
	if networkCombinedLoss == 0 {
		networkCombinedLoss = eta
	}

	// "k" here is the index of the forecaster's report among many reports
	// Forecast-implied inferences per forecaster; litepaper eq3
	I_i := make([]float64, len(forecasts.Forecasts))
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
				R_ik[j] = R_ik[j] / maxjRijk        // eq8
				w_ik[j], err = gradient(p, R_ik[j]) // eq5
				if err != nil {
					fmt.Println("Error calculating gradient: ", err)
					return nil, err
				}
			}

			// Calculate the forecast implied inferences; eq3
			weightSum := 0.0
			weightInferenceDotProduct := 0.0
			for j, w_ijk := range w_ik {
				// Calculate the forecast implied inferences
				weightInferenceDotProduct += w_ijk * float64(inferenceByWorker[forecast.ForecastElements[j].Inferer].Value.Uint64())
				weightSum += w_ijk
			}
			I_i[k] = I_i[k] / weightSum
		}
	}

	// A value of 0 => no inference corresponded to any of the forecasts from a forecaster
	return I_i, nil
}
