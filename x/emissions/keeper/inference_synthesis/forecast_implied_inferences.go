package inference_synthesis

import (
	"fmt"
	"math"

	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

// Implements function phi prime from litepaper
// φ'_p(x) = p * (ln(1 + e^x))^(p-1) * e^x / (1 + e^x)
func Gradient(p float64, x Regret) (Weight, error) {
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
	inferenceByWorker map[Worker]*emissions.Inference,
	forecasts *emissions.Forecasts,
	networkCombinedLoss Loss,
	epsilon float64,
	pInferenceSynthesis float64,
) (map[Worker]*emissions.Inference, error) {
	// Possibly add a small value to previous network loss avoid infinite logarithm
	if networkCombinedLoss == 0 {
		// Take max of epsilon and 1 to avoid division by 0
		networkCombinedLoss = math.Max(epsilon, 1)
	}

	// "k" here is the forecaster's address
	// For each forecast, and for each forecast element, calculate forecast-implied inferences I_ik
	I_i := make(map[Worker]*emissions.Inference, len(forecasts.Forecasts))
	for _, forecast := range forecasts.Forecasts {
		if len(forecast.ForecastElements) > 0 {
			// Filter away all forecast elements that do not have an associated inference (match by worker)
			// Will effectively set weight in formulas for forcast-implied inference I_ik and network inference I_i to 0 for forecasts without inferences
			// Map inferer -> forecast element => only one (latest in array) forecast element per inferer
			forecastElementsByInferer := make(map[Worker]*emissions.ForecastElement, 0)
			for _, el := range forecast.ForecastElements {
				forecastElementsByInferer[el.Inferer] = el
			}

			// Approximate forecast regrets of the network inference
			// Map inferer -> regret
			R_ik := make(map[Worker]Regret, len(forecastElementsByInferer))
			// Weights used to map inferences to forecast-implied inferences
			// Map inferer -> weight
			w_ik := make(map[Worker]Weight, len(forecastElementsByInferer))

			// Define variable to store maximum regret for forecast k
			maxjRijk := float64(-1)
			for j, el := range forecastElementsByInferer {
				// Calculate the approximate forecast regret of the network inference
				R_ik[j] = math.Log10(networkCombinedLoss / el.Value) // forecasted regrets R_ijk = log10(L_i / L_ijk)
				if R_ik[j] > maxjRijk {
					maxjRijk = R_ik[j]
				}
			}

			// Calculate normalized forecasted regrets per forecaster R_ijk then weights w_ijk per forecaster
			for j := range forecastElementsByInferer {
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
				if inferenceByWorker[j] != nil && w_ijk != 0 {
					weightInferenceDotProduct += w_ijk * inferenceByWorker[j].Value
					weightSum += w_ijk
				}
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
