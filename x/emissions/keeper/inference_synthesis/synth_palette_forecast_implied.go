package inferencesynthesis

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
	I_i := make(map[Worker]*emissionstypes.Inference, len(p.Forecasters)) //nolint:revive // var-naming: don't use underscores in Go names
	for _, forecaster := range p.Forecasters {
		_, ok := p.ForecastByWorker[forecaster]
		if ok && len(p.ForecastByWorker[forecaster].ForecastElements) > 0 {
			// Filter away all forecast elements that do not have an associated inference (match by worker)
			// Will effectively set weight in formulas for forecast-implied inference I_ik and network inference I_i to 0 for forecasts without inferences
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

			// Calculate the forecast-implied inferences I_ik
			if p.AllInferersAreNew {
				// If all inferers are new, calculate the median of the inferences
				// This means that forecasters won't be able to influence the network inference when all inferers are new
				// However, this seeds losses for forecasters for future rounds

				inferenceValues := make([]alloraMath.Dec, 0, len(sortedInferersInForecast))
				for _, inferer := range sortedInferersInForecast {
					inference, ok := p.InferenceByWorker[inferer]
					if ok {
						inferenceValues = append(inferenceValues, inference.Value)
					}
				}

				medianValue, err := alloraMath.Median(inferenceValues)
				if err != nil {
					return nil, errorsmod.Wrapf(err, "error calculating median of inference values")
				}

				forecastImpliedInference := emissionstypes.Inference{
					Inferer:     forecaster,
					Value:       medianValue,
					TopicId:     p.TopicId,
					BlockHeight: p.Nonce.BlockHeight,
					ExtraData:   make([]byte, 0),
					Proof:       "",
				}
				I_i[forecaster] = &forecastImpliedInference
			} else {
				// If not all inferers are new, calculate forecast-implied inferences using the previous inferer regrets and previous network loss

				// Approximate forecast regrets of the network inference
				// Map inferer -> regret
				R_ik := make(map[Worker]*alloraMath.Dec, len(forecastElementsByInferer)) //nolint:revive // var-naming: don't use underscores in Go names
				// Forecast-regret-informed weights dot product with inferences to yield forecast-implied inferences
				// Map inferer -> weight
				w_ik := make(map[Worker]Weight, len(forecastElementsByInferer)) //nolint:revive // var-naming: don't use underscores in Go names

				// Define variable to store maximum regret for forecast k
				// `j` is the inferer id. The nomenclature of `j` comes from the corresponding regret formulas in the litepaper
				for _, j := range sortedInferersInForecast {
					// Calculate the approximate forecast regret of the network inference
					R_ijk, err := p.NetworkCombinedLoss.Sub(forecastElementsByInferer[j].Value) //nolint:revive // var-naming: don't use underscores in Go names
					if err != nil {
						return nil, errorsmod.Wrapf(err, "error calculating network loss per value")
					}
					R_ik[j] = &R_ijk
				}

				if len(sortedInferersInForecast) > 1 {
					p.InfererRegrets = R_ik
					p.ForecasterRegrets = make(map[string]*alloraMath.Dec, 0)

					weights, err := p.CalcWeightsGivenWorkers()
					if err != nil {
						return nil, errorsmod.Wrapf(err, "error calculating normalized forecasted regrets")
					}
					w_ik = weights.inferers
				} else if len(sortedInferersInForecast) == 1 {
					weights := make(map[Worker]Weight, 1)
					weights[sortedInferersInForecast[0]] = alloraMath.OneDec()
					w_ik = weights
				}

				// Calculate the forecast-implied inferences I_ik
				for _, j := range sortedInferersInForecast {
					w_ijk := w_ik[j] //nolint:revive // var-naming: don't use underscores in Go names
					_, ok := p.InferenceByWorker[j]
					if ok && !(w_ijk.Equal(alloraMath.ZeroDec())) {
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

				if !weightSum.Equal(alloraMath.ZeroDec()) {
					forecastValue, err := weightInferenceDotProduct.Quo(weightSum)
					if err != nil {
						return nil, errorsmod.Wrapf(err, "error calculating forecast value")
					}
					forecastImpliedInference := emissionstypes.Inference{
						Inferer:     forecaster,
						Value:       forecastValue,
						TopicId:     p.TopicId,
						BlockHeight: p.Nonce.BlockHeight,
						ExtraData:   make([]byte, 0),
						Proof:       "",
					}
					I_i[forecaster] = &forecastImpliedInference
				}
			}
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

	I_i, err := p.CalcForecastImpliedInferences() //nolint:revive // var-naming: don't use underscores in Go names
	if err != nil {
		return errorsmod.Wrapf(err, "error calculating forecast-implied inferences")
	}

	p.Logger.Debug(fmt.Sprintf("Forecast-implied inferences: %v", I_i))

	p.ForecastImpliedInferenceByWorker = I_i
	return nil
}
