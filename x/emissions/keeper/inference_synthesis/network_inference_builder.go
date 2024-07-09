package inference_synthesis

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type NetworkInferenceBuilder struct {
	ctx     sdk.Context
	logger  log.Logger
	palette SynthPalette
	// Network Inferences Properties
	inferences                 []*emissions.WorkerAttributedValue
	forecastImpliedInferences  []*emissions.WorkerAttributedValue
	weights                    RegretInformedWeights
	combinedInference          InferenceValue
	naiveInference             InferenceValue
	oneOutInfererInferences    []*emissions.WithheldWorkerAttributedValue
	oneOutForecasterInferences []*emissions.WithheldWorkerAttributedValue
	oneInInferences            []*emissions.WorkerAttributedValue
}

func NewNetworkInferenceBuilderFromSynthRequest(
	req SynthRequest,
) (*NetworkInferenceBuilder, error) {
	paletteFactory := SynthPaletteFactory{}
	palette, err := paletteFactory.BuildPaletteFromRequest(req)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error building palette from request")
	}
	return &NetworkInferenceBuilder{
		ctx:     req.Ctx,
		logger:  Logger(req.Ctx),
		palette: palette,
	}, nil
}

// Calculates the network combined inference I_i, Equation 9
func (b *NetworkInferenceBuilder) SetCombinedValue() *NetworkInferenceBuilder {
	b.logger.Debug(fmt.Sprintf("Calculating combined inference for topic %v", b.palette.TopicId))
	palette := b.palette.Clone()

	weights, err := palette.CalcWeightsGivenWorkers()
	if err != nil {
		b.logger.Warn(fmt.Sprintf("Error calculating weights for combined inference: %s", err.Error()))
		return b
	}

	combinedInference, err := palette.CalcWeightedInference(weights)
	if err != nil {
		b.logger.Warn(fmt.Sprintf("Error calculating combined inference: %s", err.Error()))
		return b
	}

	b.logger.Debug(fmt.Sprintf("Combined inference calculated for topic %v is %v", b.palette.TopicId, combinedInference))
	b.combinedInference = combinedInference
	b.weights = weights
	return b
}

// Map inferences to a WorkerAttributedValue array and set
func (b *NetworkInferenceBuilder) SetInfererValues() *NetworkInferenceBuilder {
	infererValues := make([]*emissions.WorkerAttributedValue, 0)
	for _, inferer := range b.palette.Inferers {
		infererValues = append(infererValues, &emissions.WorkerAttributedValue{
			Worker: inferer,
			Value:  b.palette.InferenceByWorker[inferer].Value,
		})
	}
	b.inferences = infererValues
	return b
}

// Map forecast-implied inferences to a WorkerAttributedValue array and set
func (b *NetworkInferenceBuilder) SetForecasterValues() *NetworkInferenceBuilder {
	forecastImpliedValues := make([]*emissions.WorkerAttributedValue, 0)
	for _, forecaster := range b.palette.Forecasters {
		if b.palette.ForecastImpliedInferenceByWorker[forecaster] == nil {
			b.logger.Warn(fmt.Sprintf("No forecast-implied inference for forecaster %s", forecaster))
			continue
		}
		forecastImpliedValues = append(forecastImpliedValues, &emissions.WorkerAttributedValue{
			Worker: b.palette.ForecastImpliedInferenceByWorker[forecaster].Inferer,
			Value:  b.palette.ForecastImpliedInferenceByWorker[forecaster].Value,
		})
	}
	b.forecastImpliedInferences = forecastImpliedValues
	return b
}

// Calculates the network naive inference I^-_i
func (b *NetworkInferenceBuilder) SetNaiveValue() *NetworkInferenceBuilder {
	b.logger.Debug(fmt.Sprintf("Calculating naive inference for topic %v", b.palette.TopicId))
	palette := b.palette.Clone()

	palette.Forecasters = nil
	palette.ForecasterRegrets = make(map[string]*StatefulRegret, 0)
	weights, err := palette.CalcWeightsGivenWorkers()
	if err != nil {
		b.logger.Warn(fmt.Sprintf("Error calculating weights for naive inference: %s", err.Error()))
		return b
	}

	naiveInference, err := palette.CalcWeightedInference(weights)
	if err != nil {
		b.logger.Warn(fmt.Sprintf("Error calculating naive inference: %s", err.Error()))
		return b
	}

	b.logger.Debug(fmt.Sprintf("Naive inference calculated for topic %v is %v", b.palette.TopicId, naiveInference))
	b.naiveInference = naiveInference
	return b
}

// Calculate the one-out inference given a withheld inferer
func (b *NetworkInferenceBuilder) calcOneOutInfererInference(withheldInferer Worker) (alloraMath.Dec, error) {
	b.logger.Debug(fmt.Sprintf("Calculating one-out inference for topic %v withheld inferer %s", b.palette.TopicId, withheldInferer))
	palette := b.palette.Clone()

	// Check if withheld inferer is new
	withheldInfererRegret, ok := palette.InfererRegrets[withheldInferer]
	if !ok || withheldInfererRegret.noPriorRegret {
		return alloraMath.NewNaN(), nil
	}

	// Remove the inferer from the palette's inferers
	remainingInferers := make([]Worker, 0)
	for _, inferer := range palette.Inferers {
		if inferer != withheldInferer {
			remainingInferers = append(remainingInferers, inferer)
		}
	}

	err := palette.UpdateInferersInfo(remainingInferers)
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
	weights, err := paletteCopy.CalcWeightsGivenWorkers()
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for forecaster")
	}

	oneOutNetworkInferenceWithoutInferer, err := paletteCopy.CalcWeightedInference(weights)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for inferer")
	}

	b.logger.Debug(fmt.Sprintf("One-out inference calculated for topic %v withheld inferer %s is %v", b.palette.TopicId, withheldInferer, oneOutNetworkInferenceWithoutInferer))
	return oneOutNetworkInferenceWithoutInferer, nil
}

// Set all one-out-inferer inferences that are possible given the provided input
// Assumed that there is at most 1 inference per inferer
// Loop over all inferences and withold one, then calculate the network inference less that witheld inference
// This involves recalculating the forecast-implied inferences for each withheld inferer
func (b *NetworkInferenceBuilder) SetOneOutInfererValues() *NetworkInferenceBuilder {
	b.logger.Debug(fmt.Sprintf("Calculating one-out inferer inferences for topic %v with %v inferers", b.palette.TopicId, len(b.palette.Inferers)))
	// Calculate the one-out inferences per inferer
	oneOutInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	// Check if there are enough not new inferers to calculate one-out inference
	if b.palette.InferersNewStatus == InferersAllNewExceptOne ||
		b.palette.InferersNewStatus == InferersAllNew {
		b.oneOutInfererInferences = oneOutInferences
		return b
	}
	for _, worker := range b.palette.Inferers {
		oneOutInference, err := b.calcOneOutInfererInference(worker)
		if err != nil {
			b.logger.Warn(fmt.Sprintf("Error calculating one-out inferer inferences: %s", err.Error()))
			b.oneOutInfererInferences = make([]*emissions.WithheldWorkerAttributedValue, 0)
			return b
		}

		oneOutInferences = append(oneOutInferences, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  oneOutInference,
		})
	}

	b.logger.Debug(fmt.Sprintf("One-out inferer inferences calculated for topic %v", b.palette.TopicId))
	b.oneOutInfererInferences = oneOutInferences
	return b
}

// Calculate the one-out inference given a withheld forecaster
func (b *NetworkInferenceBuilder) calcOneOutForecasterInference(withheldForecaster Worker) (alloraMath.Dec, error) {
	b.logger.Debug(fmt.Sprintf("Calculating one-out inference for topic %v withheld forecaster %s", b.palette.TopicId, withheldForecaster))

	palette := b.palette.Clone()

	// Remove the withheldForecaster from the palette's forecasters
	remainingForecasters := make([]Worker, 0)
	for _, forecaster := range palette.Forecasters {
		if forecaster != withheldForecaster {
			remainingForecasters = append(remainingForecasters, forecaster)
		}
	}

	err := palette.UpdateForecastersInfo(remainingForecasters)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error updating forecasters")
	}

	weights, err := palette.CalcWeightsGivenWorkers()
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for forecaster")
	}

	oneOutNetworkInferenceWithoutForecaster, err := palette.CalcWeightedInference(weights)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error calculating one-out inference for forecaster")
	}

	b.logger.Debug(fmt.Sprintf("One-out inference calculated for topic %v withheld forecaster %s is %v", b.palette.TopicId, withheldForecaster, oneOutNetworkInferenceWithoutForecaster))
	return oneOutNetworkInferenceWithoutForecaster, nil
}

// Set all one-out-forecaster inferences that are possible given the provided input
// Assume that there is at most 1 forecast-implied inference per forecaster
// Loop over all forecast-implied inferences and withold one, then calculate the network inference less that witheld value
func (b *NetworkInferenceBuilder) SetOneOutForecasterValues() *NetworkInferenceBuilder {
	b.logger.Debug(fmt.Sprintf("Calculating one-out forecaster inferences for topic %v with %v forecasters", b.palette.TopicId, len(b.palette.Forecasters)))
	// Calculate the one-out forecast-implied inferences per forecaster
	oneOutImpliedInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	// Check if there are enough not new inferers to calculate one-out inferences
	if b.palette.InferersNewStatus == InferersAllNewExceptOne ||
		b.palette.InferersNewStatus == InferersAllNew ||
		b.palette.ForecastersNewStatus == ForecastersAllNewExceptOne ||
		b.palette.ForecastersNewStatus == ForecastersAllNew {
		b.oneOutForecasterInferences = oneOutImpliedInferences
		return b
	}
	for _, worker := range b.palette.Forecasters {
		oneOutInference, err := b.calcOneOutForecasterInference(worker)
		if err != nil {
			b.logger.Warn(fmt.Sprintf("Error calculating one-out forecaster inferences: %s", err.Error()))
			b.oneOutForecasterInferences = make([]*emissions.WithheldWorkerAttributedValue, 0)
			return b
		}
		oneOutImpliedInferences = append(oneOutImpliedInferences, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  oneOutInference,
		})
	}

	b.logger.Debug(fmt.Sprintf("One-out forecaster inferences calculated for topic %v", b.palette.TopicId))
	b.oneOutForecasterInferences = oneOutImpliedInferences
	return b
}

func (b *NetworkInferenceBuilder) calcOneInValue(oneInForecaster Worker) (alloraMath.Dec, error) {
	b.logger.Debug(fmt.Sprintf("Calculating one-in inference for forecaster: %s", oneInForecaster))
	palette := b.palette.Clone()

	// In each loop, remove all forecast-implied inferences except one
	forecastImpliedInferencesWithForecaster := make(map[Worker]*emissions.Inference)
	forecastImpliedInferencesWithForecaster[oneInForecaster] = palette.ForecastImpliedInferenceByWorker[oneInForecaster]
	palette.ForecastImpliedInferenceByWorker = forecastImpliedInferencesWithForecaster

	regret, noPriorRegret, err := palette.K.GetOneInForecasterSelfNetworkRegret(palette.Ctx, palette.TopicId, oneInForecaster)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-in forecaster regret")
	}

	if noPriorRegret {
		return alloraMath.NewNaN(), nil
	}

	palette.ForecasterRegrets[oneInForecaster] = &StatefulRegret{
		regret:        regret.Value,
		noPriorRegret: noPriorRegret,
	}

	remainingForecaster := []Worker{oneInForecaster}
	err = palette.UpdateForecastersInfo(remainingForecaster)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error updating forecasters")
	}

	// Get one-in regrets for the forecaster and the inferers they provided forecasts for
	for _, inferer := range palette.Inferers {
		regret, noPriorRegret, err := palette.K.GetOneInForecasterNetworkRegret(palette.Ctx, palette.TopicId, oneInForecaster, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error getting one-in forecaster regret")
		}

		palette.InfererRegrets[inferer] = &StatefulRegret{
			regret:        regret.Value,
			noPriorRegret: noPriorRegret,
		}
	}

	err = palette.UpdateInferersInfo(palette.Inferers)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "Error updating inferers")
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
	// Check if there are enough not new inferers to calculate one-in inferences
	if b.palette.InferersNewStatus == InferersAllNewExceptOne ||
		b.palette.InferersNewStatus == InferersAllNew ||
		b.palette.ForecastersNewStatus == ForecastersAllNewExceptOne ||
		b.palette.ForecastersNewStatus == ForecastersAllNew {
		b.oneInInferences = oneInInferences
		return b
	}
	for _, oneInForecaster := range b.palette.Forecasters {
		oneInValue, err := b.calcOneInValue(oneInForecaster)
		if err != nil {
			b.logger.Warn(fmt.Sprintf("Error calculating one-in inferences: %s", err.Error()))
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
		TopicId:                b.palette.TopicId,
		CombinedValue:          b.combinedInference,
		InfererValues:          b.inferences,
		ForecasterValues:       b.forecastImpliedInferences,
		NaiveValue:             b.naiveInference,
		OneOutInfererValues:    b.oneOutInfererInferences,
		OneOutForecasterValues: b.oneOutForecasterInferences,
		OneInForecasterValues:  b.oneInInferences,
	}
}
