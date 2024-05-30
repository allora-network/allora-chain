package inference_synthesis

import (
	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

// Calculate the forecast-implied inferences I_ik given inferences, forecasts and network losses.
// Calculates R_ijk, w_ijk, and I_ik for each forecast k and forecast element (forcast of worker loss) j
//
// Forecast without inference => weight in calculation of I_ik and I_i set to 0. Use latest available regret R_i-1,l
// Inference without forecast => only weight in calculation of I_ik set to 0
func CalcForecastImpliedInferences(
	inferenceByWorker map[Worker]*emissions.Inference,
	sortedWorkers []Worker,
	forecasts *emissions.Forecasts,
	networkCombinedLoss Loss,
	allInferersAreNew bool,
	epsilon alloraMath.Dec,
	pNorm alloraMath.Dec,
	cNorm alloraMath.Dec,
) (map[Worker]*emissions.Inference, error) {
	// Possibly add a small value to previous network loss avoid infinite logarithm
	if networkCombinedLoss.Equal(alloraMath.ZeroDec()) {
		// Take max of epsilon and 1 to avoid division by 0
		networkCombinedLoss = alloraMath.Max(epsilon, alloraMath.OneDec())
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
			sortedInferersInForecast := make([]Worker, 0)
			for _, el := range forecast.ForecastElements {
				for _, worker := range sortedWorkers {
					// Check that there is an inference for the worker forecasted before including the forecast element
					// otherwise the max value below will be incorrect.
					if el.Inferer == worker {
						forecastElementsByInferer[el.Inferer] = el
						sortedInferersInForecast = append(sortedInferersInForecast, el.Inferer)
						break
					}
				}
			}

			weightSum := alloraMath.ZeroDec()                 // denominator in calculation of forecast-implied inferences
			weightInferenceDotProduct := alloraMath.ZeroDec() // numerator in calculation of forecast-implied inferences
			err := error(nil)

			// Calculate the forecast-implied inferences I_ik
			if allInferersAreNew {
				// If all inferers are new, take regular average of inferences
				// This means that forecasters won't be able to influence the network inference when all inferers are new
				// However this seeds losses for forecasters for future rounds

				for _, inferer := range sortedInferersInForecast {
					if inferenceByWorker[inferer] != nil {
						weightInferenceDotProduct, err = weightInferenceDotProduct.Add(inferenceByWorker[inferer].Value)
						if err != nil {
							return nil, errorsmod.Wrapf(err, "error adding dot product")
						}
						weightSum, err = weightSum.Add(alloraMath.OneDec())
						if err != nil {
							return nil, errorsmod.Wrapf(err, "error adding weight")
						}
					}
				}
			} else {
				// If not all inferers are new, calculate forecast-implied inferences using the previous inferer regrets and previous network loss

				// Approximate forecast regrets of the network inference
				// Map inferer -> regret
				R_ik := make(map[Worker]Regret, len(forecastElementsByInferer))
				// Weights used to map inferences to forecast-implied inferences
				// Map inferer -> weight
				w_ik := make(map[Worker]Weight, len(forecastElementsByInferer))

				// Define variable to store maximum regret for forecast k
				var forecastedRegrets []alloraMath.Dec
				// `j` is the inferer id. The nomenclature of `j` comes from the corresponding regret formulas in the litepaper
				for _, j := range sortedInferersInForecast {
					// Calculate the approximate forecast regret of the network inference
					R_ik[j], err = networkCombinedLoss.Sub(forecastElementsByInferer[j].Value)
					if err != nil {
						return nil, errorsmod.Wrapf(err, "error calculating network loss per value")
					}
					forecastedRegrets = append(forecastedRegrets, R_ik[j])
				}

				var err error
				// Calc std dev of forecasted regrets + epsilon
				// σ(R_ijk) + ε
				stdDevForecastedRegrets, err := alloraMath.StdDev(forecastedRegrets)
				if err != nil {
					return nil, errorsmod.Wrapf(err, "error calculating standard deviation")
				}
				stdDevForecastedRegretsPlusEpsilon, err := stdDevForecastedRegrets.Add(epsilon)
				if err != nil {
					return nil, errorsmod.Wrapf(err, "error adding epsilon to standard deviation")
				}

				// Calculate normalized forecasted regrets per forecaster R_ijk then weights w_ijk per forecaster
				// `j` is the inferer id. The nomenclature of `j` comes from the corresponding regret formulas in the litepaper
				for _, j := range sortedInferersInForecast {
					R_ik[j], err = R_ik[j].Quo(stdDevForecastedRegretsPlusEpsilon) // \hatR_ijk = R_ijk / σ(R_ijk) + ε
					if err != nil {
						return nil, errorsmod.Wrapf(err, "error calculating normalized forecasted regrets")
					}
					w_ijk, err := alloraMath.Gradient(pNorm, cNorm, R_ik[j]) // w_ijk = φ'_p(\hatR_ijk)
					if err != nil {
						return nil, errorsmod.Wrapf(err, "error calculating gradient")
					}
					w_ik[j] = w_ijk
				}

				// Calculate the forecast-implied inferences I_ik
				for _, j := range sortedInferersInForecast {
					w_ijk := w_ik[j]
					if inferenceByWorker[j] != nil && !(w_ijk.Equal(alloraMath.ZeroDec())) {
						thisDotProduct, err := w_ijk.Mul(inferenceByWorker[j].Value)
						if err != nil {
							return nil, errorsmod.Wrapf(err, "error calculating dot product")
						}
						weightInferenceDotProduct, err = weightInferenceDotProduct.Add(thisDotProduct)
						if err != nil {
							return nil, errorsmod.Wrapf(err, "error adding dot product")
						}
						weightSum, err = weightSum.Add(w_ijk)
						if err != nil {
							return nil, errorsmod.Wrapf(err, "error adding weight")
						}
					}
				}
			}

			forecastValue, err := weightInferenceDotProduct.Quo(weightSum)
			if err != nil {
				return nil, errorsmod.Wrapf(err, "error calculating forecast value")
			}
			forecastImpliedInference := emissions.Inference{
				Inferer: forecast.Forecaster,
				Value:   forecastValue,
			}
			I_i[forecast.Forecaster] = &forecastImpliedInference
		}
	}

	// A value of 0 => no inference corresponded to any of the forecasts from a forecaster
	return I_i, nil
}
