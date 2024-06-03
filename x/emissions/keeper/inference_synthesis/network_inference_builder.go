package inference_synthesis

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type NetworkInferenceBuilder struct {
	ctx     sdk.Context
	palette SynthPalette
	// Network Inferences Properties
	inferences                 []*emissions.WorkerAttributedValue
	forecastImpliedInferences  []*emissions.WorkerAttributedValue
	combinedInference          InferenceValue
	naiveInference             InferenceValue
	oneOutInfererInferences    []*emissions.WithheldWorkerAttributedValue
	oneOutForecasterInferences []*emissions.WithheldWorkerAttributedValue
	oneInInferences            []*emissions.WorkerAttributedValue
}

func NewNetworkInferenceBuilderFromSynthRequest(
	req SynthRequest,
) *NetworkInferenceBuilder {
	paletteFactory := SynthPaletteFactory{}
	palette := paletteFactory.BuildPaletteFromRequest(req)
	return &NetworkInferenceBuilder{
		ctx:     req.Ctx,
		palette: palette,
	}
}

// Calculates the network combined naive inference I_i
func (b *NetworkInferenceBuilder) SetCombinedValue() *NetworkInferenceBuilder {
	weights, err := b.palette.CalcWeightsGivenWorkers()
	if err != nil {
		b.ctx.Logger().Warn(fmt.Sprintf("Error calculating weights for combined inference: %s", err.Error()))
		return b
	}

	combinedInference, err := b.palette.CalcWeightedInference(weights)
	if err != nil {
		b.ctx.Logger().Warn(fmt.Sprintf("Error calculating combined inference: %s", err.Error()))
		return b
	}

	b.combinedInference = combinedInference
	return b
}

// Map inferences to a WorkerAttributedValue array and set
func (b *NetworkInferenceBuilder) SetInfererValues() *NetworkInferenceBuilder {
	infererValues := make([]*emissions.WorkerAttributedValue, 0)
	for _, inferer := range b.palette.inferers {
		infererValues = append(infererValues, &emissions.WorkerAttributedValue{
			Worker: inferer,
			Value:  b.palette.inferenceByWorker[inferer].Value,
		})
	}
	b.inferences = infererValues
	return b
}

// Map forecast-implied inferences to a WorkerAttributedValue array and set
func (b *NetworkInferenceBuilder) SetForecasterValues() *NetworkInferenceBuilder {
	forecastImpliedValues := make([]*emissions.WorkerAttributedValue, 0)
	for _, forecaster := range b.palette.forecasters {
		forecastImpliedValues = append(forecastImpliedValues, &emissions.WorkerAttributedValue{
			Worker: b.palette.forecastImpliedInferenceByWorker[forecaster].Inferer,
			Value:  b.palette.forecastImpliedInferenceByWorker[forecaster].Value,
		})
	}
	b.forecastImpliedInferences = forecastImpliedValues
	return b
}

// Calculates the network naive inference I^-_i
func (b *NetworkInferenceBuilder) SetNaiveValue() *NetworkInferenceBuilder {
	palette := b.palette.Clone()

	palette.forecasters = nil
	weights, err := palette.CalcWeightsGivenWorkers()
	if err != nil {
		b.ctx.Logger().Warn(fmt.Sprintf("Error calculating weights for naive inference: %s", err.Error()))
		return b
	}

	naiveInference, err := palette.CalcWeightedInference(weights)
	if err != nil {
		b.ctx.Logger().Warn(fmt.Sprintf("Error calculating naive inference: %s", err.Error()))
		return b
	}

	b.naiveInference = naiveInference
	return b
}

// Calculate the one-out inference given a withheld inferer
func (b *NetworkInferenceBuilder) calcOneOutInfererInference(withheldInferer Worker) (alloraMath.Dec, error) {
	palette := b.palette.Clone()

	// Remove the inferer from the palette's inferers
	remainingInferers := make([]Worker, 0)
	for _, inferer := range palette.inferers {
		if inferer != withheldInferer {
			remainingInferers = append(remainingInferers, inferer)
		}
	}
	palette.inferers = remainingInferers // Override the inferers in the palette

	// Recalculate the forecast-implied inferences without the worker's inference
	// This is necessary because the forecast-implied inferences are calculated based on the inferences of the inferers
	palette.UpdateForecastImpliedInferences()

	weights, err := palette.CalcWeightsGivenWorkers()
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for forecaster")
	}

	oneOutNetworkInferenceWithoutInferer, err := palette.CalcWeightedInference(weights)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for inferer")
	}

	return oneOutNetworkInferenceWithoutInferer, nil
}

// Set all one-out-inferer inferences that are possible given the provided input
// Assumed that there is at most 1 inference per inferer
// Loop over all inferences and withold one, then calculate the network inference less that witheld inference
// This involves recalculating the forecast-implied inferences for each withheld inferer
func (b *NetworkInferenceBuilder) SetOneOutInfererValues() *NetworkInferenceBuilder {
	// Calculate the one-out inferences per inferer
	oneOutInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for _, worker := range b.palette.inferers {
		oneOutInference, err := b.calcOneOutInfererInference(worker)
		if err != nil {
			b.palette.ctx.Logger().Warn(fmt.Sprintf("Error calculating one-out inferer inferences: %s", err.Error()))
			b.oneOutInfererInferences = make([]*emissions.WithheldWorkerAttributedValue, 0)
			return b
		}
		oneOutInferences = append(oneOutInferences, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  oneOutInference,
		})
	}

	b.oneOutInfererInferences = oneOutInferences
	return b
}

// Calculate the one-out inference given a withheld forecaster
func (b *NetworkInferenceBuilder) calcOneOutForecasterInference(withheldForecaster Worker) (alloraMath.Dec, error) {
	palette := b.palette.Clone()
	// Remove the withheldForecaster from the palette's forecasters
	remainingForecasters := make([]Worker, 0)
	for _, forecaster := range palette.forecasters {
		if forecaster != withheldForecaster {
			remainingForecasters = append(remainingForecasters, forecaster)
		}
	}
	palette.forecasters = remainingForecasters // Override the forecasters in the palette

	weights, err := palette.CalcWeightsGivenWorkers()
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for forecaster")
	}

	oneOutNetworkInferenceWithoutInferer, err := palette.CalcWeightedInference(weights)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for inferer")
	}

	return oneOutNetworkInferenceWithoutInferer, nil
}

// Set all one-out-forecaster inferences that are possible given the provided input
// Assume that there is at most 1 forecast-implied inference per forecaster
// Loop over all forecast-implied inferences and withold one, then calculate the network inference less that witheld value
func (b *NetworkInferenceBuilder) SetOneOutForecasterValues() *NetworkInferenceBuilder {
	// Calculate the one-out forecast-implied inferences per forecaster
	oneOutImpliedInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for _, worker := range b.palette.forecasters {
		oneOutInference, err := b.calcOneOutForecasterInference(worker)
		if err != nil {
			b.palette.ctx.Logger().Warn(fmt.Sprintf("Error calculating one-out forecaster inferences: %s", err.Error()))
			b.oneOutForecasterInferences = make([]*emissions.WithheldWorkerAttributedValue, 0)
			return b
		}
		oneOutImpliedInferences = append(oneOutImpliedInferences, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  oneOutInference,
		})
	}

	b.oneOutForecasterInferences = oneOutImpliedInferences
	return b
}

func (b *NetworkInferenceBuilder) calcOneInValue(oneInForecaster Worker) (alloraMath.Dec, error) {
	palette := b.palette.Clone()

	// In each loop, remove all forecast-implied inferences except one
	forecastImpliedInferencesWithForecaster := make(map[Worker]*emissions.Inference)
	forecastImpliedInferencesWithForecaster[oneInForecaster] = palette.forecastImpliedInferenceByWorker[oneInForecaster]
	palette.forecastImpliedInferenceByWorker = forecastImpliedInferencesWithForecaster
	palette.forecasters = []Worker{oneInForecaster}

	// Get one-in regrets for the forecaster and the inferers they provided forecasts for
	for _, inferer := range palette.inferers {
		regret, noPriorRegret, err := palette.k.GetOneInForecasterNetworkRegret(palette.ctx, palette.topicId, oneInForecaster, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-in forecaster regret")
		}
		palette.infererRegrets[inferer] = StatefulRegret{
			regret:        regret.Value,
			noPriorRegret: noPriorRegret,
		}
	}
	regret, noPriorRegret, err := palette.k.GetOneInForecasterSelfNetworkRegret(palette.ctx, palette.topicId, oneInForecaster)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-in forecaster regret")
	}
	palette.forecasterRegrets[oneInForecaster] = StatefulRegret{
		regret:        regret.Value,
		noPriorRegret: noPriorRegret,
	}

	weights, err := palette.CalcWeightsGivenWorkers()
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating weights for one-in inferences")
	}

	// Calculate the network inference with just this forecaster's forecast-implied inference
	oneInInference, err := palette.CalcWeightedInference(weights)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-in inference")
	}

	return oneInInference, nil
}

// Set all one-in inferences that are possible given the provided input
// Assumed that there is at most 1 inference per worker.
// Also assume that there is at most 1 forecast-implied inference per worker.
func (b *NetworkInferenceBuilder) SetOneInValues() *NetworkInferenceBuilder {
	// Loop over all forecast-implied inferences and set it as the only forecast-implied inference
	// one at a time, then calculate the network inference given that one held out
	oneInInferences := make([]*emissions.WorkerAttributedValue, 0)
	for _, oneInForecaster := range b.palette.forecasters {
		oneInValue, err := b.calcOneInValue(oneInForecaster)
		if err != nil {
			b.ctx.Logger().Warn(fmt.Sprintf("Error calculating one-in inferences: %s", err.Error()))
			oneInInferences = make([]*emissions.WorkerAttributedValue, 0)
			return b
		}
		oneInInferences = append(oneInInferences, &emissions.WorkerAttributedValue{
			Worker: oneInForecaster,
			Value:  oneInValue,
		})
	}

	b.oneInInferences = oneInInferences
	return b
}

func (b *NetworkInferenceBuilder) CalcAndSetNetworkInferences() *NetworkInferenceBuilder {
	return b.SetCombinedValue().
		SetInfererValues().
		SetForecasterValues().
		SetNaiveValue().
		SetOneOutInfererValues().
		SetOneOutForecasterValues().
		SetOneInValues()
}

// Calculates all network inferences in the set I_i given historical state (e.g. regrets)
// and data from workers (e.g. inferences, forecast-implied inferences).
// Could improve this with Builder pattern, as for other instances of generated ValueBundles.
func (b *NetworkInferenceBuilder) Build() *emissions.ValueBundle {
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
