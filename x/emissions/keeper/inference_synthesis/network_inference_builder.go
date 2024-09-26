package inferencesynthesis

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

// Calculates the network combined inference I_i, Equation 9
func GetCombinedInference(palette SynthPalette) (
	weights RegretInformedWeights, combinedInference InferenceValue, err error) {
	palette.Logger.Debug(fmt.Sprintf("Calculating combined inference for topic %v", palette.TopicId))

	weights, err = calcWeightsGivenWorkers(palette)
	if err != nil {
		return RegretInformedWeights{}, InferenceValue{}, errorsmod.Wrap(err, "Error calculating weights for combined inference")
	}

	paletteCopy := palette.Clone()
	combinedInference, err = calcWeightedInference(paletteCopy, weights)
	if err != nil {
		return RegretInformedWeights{}, InferenceValue{}, errorsmod.Wrap(err, "Error calculating combined inference")
	}

	palette.Logger.Debug(fmt.Sprintf("Combined inference calculated for topic %v is %v", palette.TopicId, combinedInference))
	return weights, combinedInference, nil
}

// Map inferences to a WorkerAttributedValue array and set
func GetInferences(palette SynthPalette) (infererValues []*emissions.WorkerAttributedValue) {
	infererValues = make([]*emissions.WorkerAttributedValue, 0)
	for _, inferer := range palette.Inferers {
		infererValues = append(infererValues, &emissions.WorkerAttributedValue{
			Worker: inferer,
			Value:  palette.InferenceByWorker[inferer].Value,
		})
	}
	return infererValues
}

// Map forecast-implied inferences to a WorkerAttributedValue array and set
func GetForecastImpliedInferences(palette SynthPalette) (
	forecastImpliedInferences []*emissions.WorkerAttributedValue) {
	forecastImpliedInferences = make([]*emissions.WorkerAttributedValue, 0)
	for _, forecaster := range palette.Forecasters {
		if palette.ForecastImpliedInferenceByWorker[forecaster] == nil {
			palette.Logger.Warn(fmt.Sprintf("No forecast-implied inference for forecaster %s", forecaster))
			continue
		}
		forecastImpliedInferences = append(forecastImpliedInferences, &emissions.WorkerAttributedValue{
			Worker: palette.ForecastImpliedInferenceByWorker[forecaster].Inferer,
			Value:  palette.ForecastImpliedInferenceByWorker[forecaster].Value,
		})
	}
	return forecastImpliedInferences
}

// Calculates the network naive inference I^-_i
func GetNaiveInference(palette SynthPalette) (naiveInference alloraMath.Dec, err error) {
	palette.Logger.Debug(fmt.Sprintf("Calculating naive inference for topic %v", palette.TopicId))

	// Update the forecasters info to exclude all forecasters
	err = palette.UpdateForecastersInfo(make([]string, 0))
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "Error updating forecasters info for naive inference")
	}

	// Get inferer naive regrets
	palette.InfererRegrets = make(map[string]*alloraMath.Dec)
	for _, inferer := range palette.Inferers {
		regret, _, err := palette.K.GetNaiveInfererNetworkRegret(palette.Ctx, palette.TopicId, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting naive regret for inferer %s", inferer)
		}
		palette.InfererRegrets[inferer] = &regret.Value
	}

	weights, err := calcWeightsGivenWorkers(palette)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "Error calculating weights for naive inference")
	}

	naiveInference, err = calcWeightedInference(palette, weights)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "Error calculating naive inference")
	}

	palette.Logger.Debug(fmt.Sprintf("Naive inference calculated for topic %v is %v", palette.TopicId, naiveInference))
	return naiveInference, nil
}

// Calculate the one-out inference given a withheld inferer
func calcOneOutInfererInference(palette SynthPalette, withheldInferer Worker) (
	oneOutNetworkInferenceWithoutInferer alloraMath.Dec, err error) {
	palette.Logger.Debug(fmt.Sprintf(
		"Calculating one-out inference for topic %v withheld inferer %s", palette.TopicId, withheldInferer))
	totalInferers := palette.Inferers

	// Remove the inferer from the palette's inferers
	remainingInferers := make([]Worker, 0)
	for _, inferer := range palette.Inferers {
		if inferer != withheldInferer {
			remainingInferers = append(remainingInferers, inferer)
		}
	}

	err = palette.UpdateInferersInfo(remainingInferers)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error updating inferers")
	}

	paletteCopy := palette.Clone()

	// Recalculate the forecast-implied inferences without the worker's inference
	// This is necessary because the forecast-implied inferences are calculated based on the inferences of the inferers
	err = palette.UpdateForecastImpliedInferences()
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error recalculating forecast-implied inferences")
	}

	paletteCopy.ForecastImpliedInferenceByWorker = palette.ForecastImpliedInferenceByWorker

	// Get regrets for the remaining inferers
	paletteCopy.InfererRegrets = make(map[string]*alloraMath.Dec)
	for _, inferer := range totalInferers {
		regret, _, err := paletteCopy.K.GetOneOutInfererInfererNetworkRegret(paletteCopy.Ctx, paletteCopy.TopicId, withheldInferer, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-out inferer regret")
		}
		paletteCopy.InfererRegrets[inferer] = &regret.Value
	}
	// Get regrets for the forecasters
	paletteCopy.ForecasterRegrets = make(map[string]*alloraMath.Dec)
	for _, forecaster := range paletteCopy.Forecasters {
		regret, _, err := paletteCopy.K.GetOneOutInfererForecasterNetworkRegret(paletteCopy.Ctx, paletteCopy.TopicId, withheldInferer, forecaster)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-out forecaster regret")
		}
		paletteCopy.ForecasterRegrets[forecaster] = &regret.Value
	}

	weights, err := calcWeightsGivenWorkers(paletteCopy)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for forecaster")
	}

	oneOutNetworkInferenceWithoutInferer, err = calcWeightedInference(paletteCopy, weights)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for inferer")
	}

	palette.Logger.Debug(fmt.Sprintf("One-out inference calculated for topic %v withheld inferer %s is %v", palette.TopicId, withheldInferer, oneOutNetworkInferenceWithoutInferer))
	return oneOutNetworkInferenceWithoutInferer, nil
}

// Set all one-out-inferer inferences that are possible given the provided input
// Assumed that there is at most 1 inference per inferer
// Loop over all inferences and withhold one, then calculate the network inference less that withheld inference
// This involves recalculating the forecast-implied inferences for each withheld inferer
func GetOneOutInfererInferences(palette SynthPalette) (
	oneOutInfererInferences []*emissions.WithheldWorkerAttributedValue, err error) {
	palette.Logger.Debug(fmt.Sprintf("Calculating one-out inferer inferences for topic %v with %v inferers", palette.TopicId, len(palette.Inferers)))
	// Calculate the one-out inferences per inferer
	oneOutInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for _, worker := range palette.Inferers {
		calcPalette := palette.Clone()
		oneOutInference, err := calcOneOutInfererInference(calcPalette, worker)
		if err != nil {
			return []*emissions.WithheldWorkerAttributedValue{}, errorsmod.Wrapf(err, "Error calculating one-out inferer inferences")
		}

		oneOutInferences = append(oneOutInferences, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  oneOutInference,
		})
	}

	palette.Logger.Debug(fmt.Sprintf("One-out inferer inferences calculated for topic %v", palette.TopicId))
	return oneOutInferences, nil
}

// Calculate the one-out inference given a withheld forecaster
func calcOneOutForecasterInference(palette SynthPalette, withheldForecaster Worker) (
	oneOutNetworkInferenceWithoutInferer alloraMath.Dec, err error) {
	palette.Logger.Debug(fmt.Sprintf("Calculating one-out inference for topic %v withheld forecaster %s", palette.TopicId, withheldForecaster))
	totalForecasters := palette.Forecasters

	// Remove the withheldForecaster from the palette's forecasters
	remainingForecasters := make([]Worker, 0)
	for _, forecaster := range palette.Forecasters {
		if forecaster != withheldForecaster {
			remainingForecasters = append(remainingForecasters, forecaster)
		}
	}

	err = palette.UpdateForecastersInfo(remainingForecasters)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error updating forecasters")
	}

	// Get regrets for the remaining inferers
	palette.InfererRegrets = make(map[string]*alloraMath.Dec)
	for _, inferer := range palette.Inferers {
		regret, _, err := palette.K.GetOneOutForecasterInfererNetworkRegret(palette.Ctx, palette.TopicId, withheldForecaster, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-out inferer regret")
		}
		palette.InfererRegrets[inferer] = &regret.Value
	}
	// Get regrets for the forecasters
	palette.ForecasterRegrets = make(map[string]*alloraMath.Dec)
	for _, forecaster := range totalForecasters {
		regret, _, err := palette.K.GetOneOutForecasterForecasterNetworkRegret(palette.Ctx, palette.TopicId, withheldForecaster, forecaster)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-out forecaster regret")
		}
		palette.ForecasterRegrets[forecaster] = &regret.Value
	}

	weights, err := calcWeightsGivenWorkers(palette)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for forecaster")
	}

	oneOutNetworkInferenceWithoutInferer, err = calcWeightedInference(palette, weights)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for inferer")
	}

	palette.Logger.Debug(fmt.Sprintf("One-out inference calculated for topic %v withheld forecaster %s is %v", palette.TopicId, withheldForecaster, oneOutNetworkInferenceWithoutInferer))
	return oneOutNetworkInferenceWithoutInferer, nil
}

// Set all one-out-forecaster inferences that are possible given the provided input
// Assume that there is at most 1 forecast-implied inference per forecaster
// Loop over all forecast-implied inferences and withhold one, then calculate the network inference less that withheld value
func GetOneOutForecasterInferences(palette SynthPalette) (
	oneOutForecasterInferences []*emissions.WithheldWorkerAttributedValue, err error) {
	palette.Logger.Debug(fmt.Sprintf("Calculating one-out forecaster inferences for topic %v with %v forecasters", palette.TopicId, len(palette.Forecasters)))
	// Calculate the one-out forecast-implied inferences per forecaster
	oneOutForecasterInferences = make([]*emissions.WithheldWorkerAttributedValue, 0)
	// If there is only one forecaster, there's no need to calculate one-out inferences
	if len(palette.Forecasters) > 1 {
		for _, worker := range palette.Forecasters {
			calcPalette := palette.Clone()
			oneOutInference, err := calcOneOutForecasterInference(calcPalette, worker)
			if err != nil {
				return []*emissions.WithheldWorkerAttributedValue{}, errorsmod.Wrapf(err, "Error calculating one-out forecaster inferences")
			}
			oneOutForecasterInferences = append(oneOutForecasterInferences, &emissions.WithheldWorkerAttributedValue{
				Worker: worker,
				Value:  oneOutInference,
			})
		}
		palette.Logger.Debug(fmt.Sprintf("One-out forecaster inferences calculated for topic %v", palette.TopicId))
	}
	return oneOutForecasterInferences, nil
}

func calcOneInValue(palette SynthPalette, oneInForecaster Worker) (
	oneInInference alloraMath.Dec, err error) {
	palette.Logger.Debug(fmt.Sprintf("Calculating one-in inference for forecaster: %s", oneInForecaster))

	// In each loop, remove all forecast-implied inferences except one
	forecastImpliedInferencesWithForecaster := make(map[Worker]*emissions.Inference)
	forecastImpliedInferencesWithForecaster[oneInForecaster] = palette.ForecastImpliedInferenceByWorker[oneInForecaster]
	palette.ForecastImpliedInferenceByWorker = forecastImpliedInferencesWithForecaster

	// Get self regret for the forecaster
	regret, _, err := palette.K.GetOneInForecasterNetworkRegret(palette.Ctx, palette.TopicId, oneInForecaster, oneInForecaster)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-in forecaster regret")
	}

	palette.ForecasterRegrets[oneInForecaster] = &regret.Value

	remainingForecaster := []Worker{oneInForecaster}
	err = palette.UpdateForecastersInfo(remainingForecaster)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error updating forecasters")
	}

	// Get one-in regrets for the forecaster and the inferers they provided forecasts for
	for _, inferer := range palette.Inferers {
		regret, _, err := palette.K.GetOneInForecasterNetworkRegret(palette.Ctx, palette.TopicId, oneInForecaster, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-in forecaster regret")
		}

		palette.InfererRegrets[inferer] = &regret.Value
	}

	err = palette.UpdateInferersInfo(palette.Inferers)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error updating inferers")
	}

	weights, err := calcWeightsGivenWorkers(palette)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating weights for one-in inferences")
	}
	// Calculate the network inference with just this forecaster's forecast-implied inference
	oneInInference, err = calcWeightedInference(palette, weights)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-in inference")
	}

	return oneInInference, nil
}

// Set all one-in inferences that are possible given the provided input
// Assumed that there is at most 1 inference per worker.
// Also assume that there is at most 1 forecast-implied inference per worker.
func GetOneInForecasterInferences(palette SynthPalette) (oneInInferences []*emissions.WorkerAttributedValue, err error) {
	// Loop over all forecast-implied inferences and set it as the only forecast-implied inference
	// one at a time, then calculate the network inference given that one held out
	oneInInferences = make([]*emissions.WorkerAttributedValue, 0)
	// If there is only one forecaster, thre's no need to calculate one-in inferences
	if len(palette.Forecasters) > 1 {
		for _, oneInForecaster := range palette.Forecasters {
			calcPalette := palette.Clone()
			oneInValue, err := calcOneInValue(calcPalette, oneInForecaster)
			if err != nil {
				return []*emissions.WorkerAttributedValue{}, errorsmod.Wrapf(err, "Error calculating one-in inferences")
			}
			oneInInferences = append(oneInInferences, &emissions.WorkerAttributedValue{
				Worker: oneInForecaster,
				Value:  oneInValue,
			})
		}
	}
	return oneInInferences, err
}

// Calculates all network inferences in the set I_i given historical state (e.g. regrets)
// and data from workers (e.g. inferences, forecast-implied inferences).
// Could improve this with Builder pattern, as for other instances of generated ValueBundles.
func CalcNetworkInferences(
	palette SynthPalette,
) (inferenceBundle *emissions.ValueBundle, weights RegretInformedWeights, err error) {
	weights, combinedInference, err := GetCombinedInference(palette)
	if err != nil {
		return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "Error calculating combined inference")
	}
	inferences := GetInferences(palette)
	forecastImpliedInferences := GetForecastImpliedInferences(palette)
	naiveInferencePalette := palette.Clone()
	naiveInference, err := GetNaiveInference(naiveInferencePalette)
	if err != nil {
		return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "Error calculating naive inference")
	}
	oneOutInfererInferences, err := GetOneOutInfererInferences(palette)
	if err != nil {
		return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "Error calculating one-out inferer inferences")
	}
	oneOutForecasterInferences, err := GetOneOutForecasterInferences(palette)
	if err != nil {
		return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "Error calculating one-out forecaster inferences")
	}
	oneInForecasterInferences, err := GetOneInForecasterInferences(palette)
	if err != nil {
		return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "Error calculating one-in inferences")
	}

	// Build value bundle to return all the calculated inferences
	return &emissions.ValueBundle{
		TopicId:                palette.TopicId,
		CombinedValue:          combinedInference,
		InfererValues:          inferences,
		ForecasterValues:       forecastImpliedInferences,
		NaiveValue:             naiveInference,
		OneOutInfererValues:    oneOutInfererInferences,
		OneOutForecasterValues: oneOutForecasterInferences,
		OneInForecasterValues:  oneInForecasterInferences,
	}, weights, err
}
