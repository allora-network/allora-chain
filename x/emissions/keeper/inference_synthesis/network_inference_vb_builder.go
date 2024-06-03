package inference_synthesis

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

// Vb = ValueBundle
type NetworkInferenceVbBuilder struct {
	palette                    SynthPalette
	combinedInference          InferenceValue
	inferences                 []*emissions.WorkerAttributedValue
	forecastImpliedInferences  []*emissions.WorkerAttributedValue
	naiveInference             InferenceValue
	oneOutInfererInferences    []*emissions.WithheldWorkerAttributedValue
	oneOutForecasterInferences []*emissions.WithheldWorkerAttributedValue
	oneInInferences            []*emissions.WorkerAttributedValue
}

func newNetworkInferenceVbBuilderFromSynthRequest(
	req SynthRequest,
) *NetworkInferenceVbBuilder {
	paletteFactory := SynthPaletteFactory{}
	palette := paletteFactory.BuildPaletteFromRequest(req)
	return &NetworkInferenceVbBuilder{palette: palette}
}

// Calculates the network combined naive inference I_i
func (b *NetworkInferenceVbBuilder) setCombinedValue() {
	weights, err := b.palette.CalcWeightsGivenWorkers()
	if err != nil {
		b.palette.ctx.Logger().Warn(fmt.Sprintf("Error calculating weights for combined inference: %s", err.Error()))
		return
	}

	combinedInference, err := b.palette.CalcWeightedInference(weights)
	if err != nil {
		b.palette.ctx.Logger().Warn(fmt.Sprintf("Error calculating combined inference: %s", err.Error()))
		return
	}

	b.combinedInference = combinedInference
}

// Map inferences to a WorkerAttributedValue array and set
func (b *NetworkInferenceVbBuilder) setInfererValues() {
	infererValues := make([]*emissions.WorkerAttributedValue, 0)
	for _, inferer := range b.palette.inferers {
		infererValues = append(infererValues, &emissions.WorkerAttributedValue{
			Worker: inferer,
			Value:  b.palette.inferenceByWorker[inferer].Value,
		})
	}
	b.inferences = infererValues
}

// Map forecast-implied inferences to a WorkerAttributedValue array and set
func (b *NetworkInferenceVbBuilder) setForecasterValues() {
	forecastImpliedValues := make([]*emissions.WorkerAttributedValue, 0)
	for _, forecaster := range b.palette.forecasters {
		forecastImpliedValues = append(forecastImpliedValues, &emissions.WorkerAttributedValue{
			Worker: b.palette.forecastImpliedInferenceByWorker[forecaster].Inferer,
			Value:  b.palette.forecastImpliedInferenceByWorker[forecaster].Value,
		})
	}
	b.forecastImpliedInferences = forecastImpliedValues
}

// Calculates the network naive inference I^-_i
func (b *NetworkInferenceVbBuilder) setNaiveValue() {
	b.palette.forecasters = nil
	weights, err := b.palette.CalcWeightsGivenWorkers()
	if err != nil {
		b.palette.ctx.Logger().Warn(fmt.Sprintf("Error calculating weights for naive inference: %s", err.Error()))
		return
	}

	naiveInference, err := b.palette.CalcWeightedInference(weights)
	if err != nil {
		b.palette.ctx.Logger().Warn(fmt.Sprintf("Error calculating naive inference: %s", err.Error()))
		return
	}

	b.naiveInference = naiveInference
}

// Calculate the one-out inference given a withheld inferer
func (b *NetworkInferenceVbBuilder) calcOneOutInfererInference(withheldInferer Worker) (alloraMath.Dec, error) {
	// Remove the inferer from the palette's inferers
	remainingInferers := make([]Worker, 0)
	for _, inferer := range b.palette.inferers {
		if inferer != withheldInferer {
			remainingInferers = append(remainingInferers, inferer)
		}
	}
	b.palette.inferers = remainingInferers // Override the inferers in the palette

	// Recalculate the forecast-implied inferences without the worker's inference
	// This is necessary because the forecast-implied inferences are calculated based on the inferences of the inferers
	b.palette.UpdateForecastImpliedInferences()

	weights, err := b.palette.CalcWeightsGivenWorkers()
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for forecaster")
	}

	oneOutNetworkInferenceWithoutInferer, err := b.palette.CalcWeightedInference(weights)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for inferer")
	}

	return oneOutNetworkInferenceWithoutInferer, nil
}

// Set all one-out-inferer inferences that are possible given the provided input
// Assumed that there is at most 1 inference per inferer
// Loop over all inferences and withold one, then calculate the network inference less that witheld inference
// This involves recalculating the forecast-implied inferences for each withheld inferer
func (b *NetworkInferenceVbBuilder) setOneOutInfererValues() {
	// Calculate the one-out inferences per inferer
	oneOutInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for _, worker := range b.palette.inferers {
		oneOutInference, err := b.calcOneOutInfererInference(worker)
		if err != nil {
			b.palette.ctx.Logger().Warn(fmt.Sprintf("Error calculating one-out inferer inferences: %s", err.Error()))
			b.oneOutInfererInferences = make([]*emissions.WithheldWorkerAttributedValue, 0)
			return
		}
		oneOutInferences = append(oneOutInferences, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  oneOutInference,
		})
	}

	b.oneOutInfererInferences = oneOutInferences
}

// Calculate the one-out inference given a withheld forecaster
func (b *NetworkInferenceVbBuilder) calcOneOutForecasterInference(withheldForecaster Worker) (alloraMath.Dec, error) {
	// Remove the withheldForecaster from the palette's forecasters
	remainingForecasters := make([]Worker, 0)
	for _, forecaster := range b.palette.forecasters {
		if forecaster != withheldForecaster {
			remainingForecasters = append(remainingForecasters, forecaster)
		}
	}
	b.palette.forecasters = remainingForecasters // Override the forecasters in the palette

	weights, err := b.palette.CalcWeightsGivenWorkers()
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for forecaster")
	}

	oneOutNetworkInferenceWithoutInferer, err := b.palette.CalcWeightedInference(weights)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for inferer")
	}

	return oneOutNetworkInferenceWithoutInferer, nil
}

// Set all one-out-forecaster inferences that are possible given the provided input
// Assume that there is at most 1 forecast-implied inference per forecaster
// Loop over all forecast-implied inferences and withold one, then calculate the network inference less that witheld value
func (b *NetworkInferenceVbBuilder) setOneOutForecasterValues() {
	// Calculate the one-out forecast-implied inferences per forecaster
	oneOutImpliedInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for _, worker := range b.palette.forecasters {
		oneOutInference, err := b.calcOneOutForecasterInference(worker)
		if err != nil {
			b.palette.ctx.Logger().Warn(fmt.Sprintf("Error calculating one-out forecaster inferences: %s", err.Error()))
			b.oneOutForecasterInferences = make([]*emissions.WithheldWorkerAttributedValue, 0)
			return
		}
		oneOutImpliedInferences = append(oneOutImpliedInferences, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  oneOutInference,
		})
	}

	b.oneOutForecasterInferences = oneOutImpliedInferences
}

func (b *NetworkInferenceVbBuilder) calcOneInValue(oneInForecaster Worker) (alloraMath.Dec, error) {
	// In each loop, remove all forecast-implied inferences except one
	forecastImpliedInferencesWithForecaster := make(map[Worker]*emissions.Inference)
	forecastImpliedInferencesWithForecaster[oneInForecaster] = b.palette.forecastImpliedInferenceByWorker[oneInForecaster]
	b.palette.forecastImpliedInferenceByWorker = forecastImpliedInferencesWithForecaster
	b.palette.forecasters = []Worker{oneInForecaster}

	// Get one-in regrets for the forecaster and the inferers they provided forecasts for
	for _, inferer := range b.palette.inferers {
		//
		// TODO MULTIPLY BY INFERENCES + 1-EXTRA FORECAST-IMPLIED INFERENCES
		//
		regret, noPriorRegret, err := b.palette.k.GetOneInForecasterNetworkRegret(b.palette.ctx, b.palette.topicId, oneInForecaster, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-in forecaster regret")
		}
		b.palette.forecasterRegrets[inferer] = StatefulRegret{
			regret:        regret.Value,
			noPriorRegret: noPriorRegret,
		}
	}

	weights, err := b.palette.CalcWeightsGivenWorkers()
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating weights for one-in inferences")
	}

	// Calculate the network inference with just this forecaster's forecast-implied inference
	oneInInference, err := b.palette.CalcWeightedInference(weights)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-in inference")
	}

	return oneInInference, nil
}

// Set all one-in inferences that are possible given the provided input
// Assumed that there is at most 1 inference per worker.
// Also assume that there is at most 1 forecast-implied inference per worker.
func (b *NetworkInferenceVbBuilder) setOneInValues() {
	// Loop over all forecast-implied inferences and set it as the only forecast-implied inference
	// one at a time, then calculate the network inference given that one held out
	oneInInferences := make([]*emissions.WorkerAttributedValue, 0)
	for _, oneInForecaster := range b.palette.forecasters {
		oneInValue, err := b.calcOneInValue(oneInForecaster)
		if err != nil {
			b.palette.ctx.Logger().Warn(fmt.Sprintf("Error calculating one-in inferences: %s", err.Error()))
			oneInInferences = make([]*emissions.WorkerAttributedValue, 0)
			return
		}
		oneInInferences = append(oneInInferences, &emissions.WorkerAttributedValue{
			Worker: oneInForecaster,
			Value:  oneInValue,
		})
	}

	b.oneInInferences = oneInInferences
}

// Calculates all network inferences in the set I_i given historical state (e.g. regrets)
// and data from workers (e.g. inferences, forecast-implied inferences).
// Could improve this with Builder pattern, as for other instances of generated ValueBundles.
func (b *NetworkInferenceVbBuilder) getNetworkValues() *emissions.ValueBundle {
	b.setCombinedValue()
	b.setInfererValues()
	b.setForecasterValues()
	b.setNaiveValue()
	b.setOneOutInfererValues()
	b.setOneOutForecasterValues()
	b.setOneInValues()

	// Build value bundle to return all the calculated inferences
	return &emissions.ValueBundle{
		TopicId:                b.palette.topicId,
		CombinedValue:          b.combinedInference,
		InfererValues:          b.inferences,
		ForecasterValues:       b.forecastImpliedInferences,
		NaiveValue:             b.naiveInference,
		OneOutInfererValues:    b.oneOutInfererInferences,
		OneOutForecasterValues: b.oneOutForecasterInferences,
		OneInForecasterValues:  b.oneInInferences,
	}
}
