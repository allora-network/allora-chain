package inference_synthesis

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// Calculate the forecast-implied inferences I_ik given inferences, forecasts and network losses.
// Calculates R_ijk, w_ijk, and I_ik for each forecast k and forecast element (forcast of worker loss) j
//
// Forecast without inference => weight in calculation of I_ik and I_i set to 0. Use latest available regret R_i-1,l
// Inference without forecast => only weight in calculation of I_ik set to 0
// A value of 0 => no inference corresponded to any of the forecasts from a forecaster
//
// Requires: forecasts, inferenceByWorker, allInferersAreNew, networkCombinedLoss, epsilon, pNorm, cNorm
// Updates: forecastImpliedInferenceByWorker
func (p *SynthPalette) CalcForecastImpliedInferences() (map[Worker]*emissionstypes.Inference, error) {
	// "k" here is the forecaster's address
	// For each forecast, and for each forecast element, calculate forecast-implied inferences I_ik
	I_i := make(map[Worker]*emissionstypes.Inference, len(p.Forecasters))
	for _, forecaster := range p.Forecasters {
		if len(p.ForecastByWorker[forecaster].ForecastElements) > 0 {
			// Filter away all forecast elements that do not have an associated inference (match by worker)
			// Will effectively set weight in formulas for forcast-implied inference I_ik and network inference I_i to 0 for forecasts without inferences
			// Map inferer -> forecast element => only one (latest in array) forecast element per inferer
			forecastElementsByInferer := make(map[Worker]*emissionstypes.ForecastElement, 0)
			sortedInferersInForecast := make([]Worker, 0)
			for _, el := range p.ForecastByWorker[forecaster].ForecastElements {
				if _, ok := p.InferenceByWorker[el.Inferer]; ok {
					// Check that there is an inference for the worker forecasted before including the forecast element
					// otherwise the max value below will be incorrect.
					forecastElementsByInferer[el.Inferer] = el
					sortedInferersInForecast = append(sortedInferersInForecast, el.Inferer)
				}
			}

			weightSum := alloraMath.ZeroDec()                 // denominator in calculation of forecast-implied inferences
			weightInferenceDotProduct := alloraMath.ZeroDec() // numerator in calculation of forecast-implied inferences
			err := error(nil)

			// Calculate the forecast-implied inferences I_ik
			if p.InferersNewStatus == InferersAllNew {
				// If all inferers are new, take regular average of inferences
				// This means that forecasters won't be able to influence the network inference when all inferers are new
				// However this seeds losses for forecasters for future rounds

				for _, inferer := range sortedInferersInForecast {
					if p.InferenceByWorker[inferer] != nil {
						weightInferenceDotProduct, err = weightInferenceDotProduct.Add(p.InferenceByWorker[inferer].Value)
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
				R_ik := make(map[Worker]*StatefulRegret, len(forecastElementsByInferer))
				// Forecast-regret-informed weights dot product with inferences to yield forecast-implied inferences
				// Map inferer -> weight
				w_ik := make(map[Worker]Weight, len(forecastElementsByInferer))

				// Define variable to store maximum regret for forecast k
				// `j` is the inferer id. The nomenclature of `j` comes from the corresponding regret formulas in the litepaper
				for _, j := range sortedInferersInForecast {
					// Calculate the approximate forecast regret of the network inference
					R_ijk, err := p.NetworkCombinedLoss.Sub(forecastElementsByInferer[j].Value)
					if err != nil {
						return nil, errorsmod.Wrapf(err, "error calculating network loss per value")
					}
					R_ik[j] = &StatefulRegret{regret: R_ijk, noPriorRegret: false}
				}

				if len(sortedInferersInForecast) > 0 {
					p.InfererRegrets = R_ik
					p.ForecasterRegrets = make(map[string]*StatefulRegret, 0)

					weights, err := p.CalcWeightsGivenWorkers()
					if err != nil {
						return nil, errorsmod.Wrapf(err, "error calculating normalized forecasted regrets")
					}
					w_ik = weights.inferers
				}

				// Calculate the forecast-implied inferences I_ik
				for _, j := range sortedInferersInForecast {
					w_ijk := w_ik[j]
					if p.InferenceByWorker[j] != nil && !(w_ijk.Equal(alloraMath.ZeroDec())) {
						thisDotProduct, err := w_ijk.Mul(p.InferenceByWorker[j].Value)
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
			forecastImpliedInference := emissionstypes.Inference{
				Inferer: forecaster,
				Value:   forecastValue,
			}
			I_i[forecaster] = &forecastImpliedInference
		}
	}

	return I_i, nil
}

// Calculate the forecast-implied inferences I_ik given inferences, forecasts and network losses.
// See docs of CalcForecastImpliedInferences for more details on calculation.
// This calculates and then sets the forecastImpliedInferenceByWorker property of the palette.
// Requires: forecasts, inferenceByWorker, allInferersAreNew, networkCombinedLoss, epsilon, pNorm, cNorm
// Updates: forecastImpliedInferenceByWorker
func (p *SynthPalette) UpdateForecastImpliedInferences() error {
	p.Logger.Debug(fmt.Sprintf("Calculating forecast-implied inferences for topic %v", p.TopicId))

	I_i, err := p.CalcForecastImpliedInferences()
	if err != nil {
		return errorsmod.Wrapf(err, "error calculating forecast-implied inferences")
	}

	p.Logger.Debug(fmt.Sprintf("Forecast-implied inferences: %v", I_i))

	p.ForecastImpliedInferenceByWorker = I_i
	return nil
}
