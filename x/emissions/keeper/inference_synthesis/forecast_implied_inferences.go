package inference_synthesis

import (
	"fmt"

	alloraMath "github.com/allora-network/allora-chain/math"

	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

// Implements function phi prime from litepaper
// φ'_p(x) = p * (ln(1 + e^x))^(p-1) * e^x / (1 + e^x)
func Gradient(p alloraMath.Dec, x Regret) (Weight, error) {
	eToTheX, err := alloraMath.Exp(x)
	if err != nil {
		return Weight{}, err
	}
	onePlusEToTheX, err := alloraMath.OneDec().Add(eToTheX)
	if err != nil {
		return Weight{}, err
	}
	naturalLog, err := alloraMath.Ln(onePlusEToTheX)
	if err != nil {
		return Weight{}, err
	}
	pMinusOne, err := p.Sub(alloraMath.OneDec())
	if err != nil {
		return Weight{}, err
	}
	result, err := alloraMath.Pow(naturalLog, pMinusOne)
	if err != nil {
		return Weight{}, err
	}
	pResult, err := p.Mul(result)
	if err != nil {
		return Weight{}, err
	}
	numerator, err := pResult.Mul(eToTheX)
	if err != nil {
		return Weight{}, err
	}
	ret, err := numerator.Quo(onePlusEToTheX)
	if err != nil {
		return Weight{}, err
	}
	return ret, nil
}

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
	pInferenceSynthesis alloraMath.Dec,
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
							fmt.Println("Error adding dot product: ", err)
							return nil, err
						}
						weightSum, err = weightSum.Add(alloraMath.OneDec())
						if err != nil {
							fmt.Println("Error adding weight: ", err)
							return nil, err
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
				first := true
				var maxjRijk alloraMath.Dec
				// `j` is the inferer id. The nomenclature of `j` comes from the corresponding regret formulas in the litepaper
				for _, j := range sortedInferersInForecast {
					// Calculate the approximate forecast regret of the network inference
					networkLossPerValue, err := networkCombinedLoss.Quo(forecastElementsByInferer[j].Value)
					if err != nil {
						fmt.Println("Error calculating network loss per value: ", err)
						return nil, err
					}
					R_ik[j], err = alloraMath.Log10(networkLossPerValue) // forecasted regrets R_ijk = log10(L_i / L_ijk)
					if err != nil {
						fmt.Println("Error calculating forecasted regrets: ", err)
						return nil, err
					}
					if first {
						maxjRijk = R_ik[j]
						first = false
					} else {
						if R_ik[j].Gt(maxjRijk) {
							maxjRijk = R_ik[j]
						}
					}
				}

				// Calculate normalized forecasted regrets per forecaster R_ijk then weights w_ijk per forecaster
				var err error
				// `j` is the inferer id. The nomenclature of `j` comes from the corresponding regret formulas in the litepaper
				for _, j := range sortedInferersInForecast {
					R_ik[j], err = R_ik[j].Quo(maxjRijk.Abs()) // \hatR_ijk = R_ijk / |max_{j'}(R_ijk)|
					if err != nil {
						fmt.Println("Error calculating normalized forecasted regrets: ", err)
						return nil, err
					}
					w_ijk, err := Gradient(pInferenceSynthesis, R_ik[j]) // w_ijk = φ'_p(\hatR_ijk)
					if err != nil {
						fmt.Println("Error calculating gradient: ", err)
						return nil, err
					}
					w_ik[j] = w_ijk
				}

				// Calculate the forecast-implied inferences I_ik
				for _, j := range sortedInferersInForecast {
					w_ijk := w_ik[j]
					if inferenceByWorker[j] != nil && !(w_ijk.Equal(alloraMath.ZeroDec())) {
						thisDotProduct, err := w_ijk.Mul(inferenceByWorker[j].Value)
						if err != nil {
							fmt.Println("Error calculating dot product: ", err)
							return nil, err
						}
						weightInferenceDotProduct, err = weightInferenceDotProduct.Add(thisDotProduct)
						if err != nil {
							fmt.Println("Error adding dot product: ", err)
							return nil, err
						}
						weightSum, err = weightSum.Add(w_ijk)
						if err != nil {
							fmt.Println("Error adding weight: ", err)
							return nil, err
						}
					}
				}
			}

			forecastValue, err := weightInferenceDotProduct.Quo(weightSum)
			if err != nil {
				fmt.Println("Error calculating forecast value: ", err)
				return nil, err
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
