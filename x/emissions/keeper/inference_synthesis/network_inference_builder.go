package inferencesynthesis

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

// GetCombinedInferenceArgs holds inputs for GetCombinedInference.
type GetCombinedInferenceArgs struct {
	Logger                               log.Logger
	TopicId                              uint64
	Inferers                             []Worker
	InfererToInference                   map[Worker]*emissions.Inference
	InfererToRegret                      map[Worker]*alloraMath.Dec
	AllInferersAreNew                    bool
	Forecasters                          []Worker
	ForecasterToRegret                   map[Worker]*alloraMath.Dec
	ForecasterToForecastImpliedInference map[Worker]*emissions.Inference
	EpsilonTopic                         alloraMath.Dec
	EpsilonSafeDiv                       alloraMath.Dec
	PNorm                                alloraMath.Dec
	CNorm                                alloraMath.Dec
	RegretScalePlusEpsilon               alloraMath.Dec
	NumLabels                            int
}

// Calculates the network combined inference I_i, Equation 9
func GetCombinedInference(args GetCombinedInferenceArgs) (
	weights RegretInformedWeights, combinedInference emissions.InferenceValues, err error) {
	args.Logger.Debug("Calculating combined inference", "topicId", args.TopicId)

	weights, err = CalcWeightsGivenWorkers(
		CalcWeightsGivenWorkersArgs{
			Logger:                 args.Logger,
			Inferers:               args.Inferers,
			Forecasters:            args.Forecasters,
			InfererToRegret:        args.InfererToRegret,
			ForecasterToRegret:     args.ForecasterToRegret,
			EpsilonTopic:           args.EpsilonTopic,
			PNorm:                  args.PNorm,
			CNorm:                  args.CNorm,
			RegretScalePlusEpsilon: args.RegretScalePlusEpsilon,
		},
	)
	if err != nil {
		return RegretInformedWeights{}, emissions.InferenceValues{}, errorsmod.Wrap(err, "GetCombinedInference() error calculating weights for combined inference")
	}

	combinedInference, err = calcWeightedInference(calcWeightedInferenceArgs{
		logger:                               args.Logger,
		allInferersAreNew:                    args.AllInferersAreNew,
		inferers:                             args.Inferers,
		workerToInference:                    args.InfererToInference,
		infererToRegret:                      args.InfererToRegret,
		forecasters:                          args.Forecasters,
		forecasterToRegret:                   args.ForecasterToRegret,
		forecasterToForecastImpliedInference: args.ForecasterToForecastImpliedInference,
		weights:                              weights,
		epsilonSafeDiv:                       args.EpsilonSafeDiv,
		numLabels:                            args.NumLabels,
	})
	if err != nil {
		return RegretInformedWeights{}, emissions.InferenceValues{}, errorsmod.Wrap(err, "GetCombinedInference() error calculating combined inference")
	}

	args.Logger.Debug("Combined inference calculated", "topicId", args.TopicId)
	return weights, combinedInference, nil
}

// Map inferences to a WorkerAttributedValue array and set
func getInferences(
	inferers []Worker,
	infererToInference map[Worker]*emissions.Inference,
	reg *emissions.EpochLabelRegistry,
) (infererValues []*emissions.WorkerInference, _ error) {
	infererValues = make([]*emissions.WorkerInference, 0)
	for _, inferer := range inferers {
		inference, ok := infererToInference[inferer]
		if !ok {
			return []*emissions.WorkerInference{}, errorsmod.Wrapf(sdkerrors.ErrLogic, "missing inference for inferer: %s", inferer)
		}
		inferenceValues, err := emissions.ConvertInferenceValuesToLabeledValues(inference.Values, reg)
		if err != nil {
			return []*emissions.WorkerInference{}, errorsmod.Wrap(err, "failed to convert inference values")
		}

		infererValues = append(infererValues, &emissions.WorkerInference{
			Worker: inferer,
			Values: inferenceValues,
		})
	}
	return infererValues, nil
}

// Map forecast-implied inferences to a WorkerAttributedValue array and set
func getForecastImpliedInferences(
	logger log.Logger,
	forecasters []Worker,
	forecasterToForecastImpliedInference map[Worker]*emissions.Inference,
	reg *emissions.EpochLabelRegistry,
) (forecastImpliedInferences []*emissions.WorkerInference, _ error) {
	forecastImpliedInferences = make([]*emissions.WorkerInference, 0)
	for _, forecaster := range forecasters {
		if forecasterToForecastImpliedInference[forecaster] == nil {
			logger.Warn("getForecastImpliedInferences() no forecast-implied inference", "forecaster", forecaster)
			continue
		}
		inferenceValues, err := emissions.ConvertInferenceValuesToLabeledValues(forecasterToForecastImpliedInference[forecaster].Values, reg)
		if err != nil {
			return []*emissions.WorkerInference{}, errorsmod.Wrap(err, "failed to convert inference values")
		}
		forecastImpliedInferences = append(forecastImpliedInferences, &emissions.WorkerInference{
			Worker: forecaster,
			Values: inferenceValues,
		})
	}
	return forecastImpliedInferences, nil
}

type GetOneOutInfererForecastImpliedInferencesArgs struct {
	Ctx                    sdk.Context
	K                      emissionskeeper.Keeper
	Logger                 log.Logger
	TopicId                uint64
	TopicArity             emissions.TopicOutputArity
	Inferers               []Inferer
	InfererToInference     map[Inferer]*emissions.Inference
	InfererToRegret        map[Inferer]*Regret
	AllInferersAreNew      bool
	Forecasters            []Forecaster
	ForecasterToForecast   map[Forecaster]*emissions.Forecast
	ForecasterToRegret     map[Forecaster]*Regret
	NetworkCombinedLoss    *alloraMath.Dec
	EpsilonTopic           alloraMath.Dec
	PNorm                  alloraMath.Dec
	CNorm                  alloraMath.Dec
	RegretScalePlusEpsilon alloraMath.Dec
	LabelRegistry          *emissions.EpochLabelRegistry
	NumLabels              int
	LabelDefaultValue      alloraMath.Dec
}

// GetOneOutInfererForecastImpliedInferences calculates what each forecaster's implied inference
// would be when each inferer is removed from the calculation
func GetOneOutInfererForecastImpliedInferences(args GetOneOutInfererForecastImpliedInferencesArgs) (
	oneOutInfererForecastImpliedValues []*emissions.OneOutInfererForecasterValue,
	err error,
) {
	oneOutInfererForecastImpliedValues = make([]*emissions.OneOutInfererForecasterValue, 0)

	// If NetworkCombinedLoss is nil, return empty slice immediately
	if args.NetworkCombinedLoss == nil {
		args.Logger.Debug("NetworkCombinedLoss is nil, returning empty one-out inferer forecast implied values", "topicId", args.TopicId)
		return oneOutInfererForecastImpliedValues, nil
	}

	// Only process if we have both inferers and forecasters
	if len(args.Inferers) <= 1 || len(args.Forecasters) == 0 {
		return oneOutInfererForecastImpliedValues, nil
	}

	if args.LabelRegistry == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "GetOneOutInfererForecastImpliedInferences: LabelRegistry is nil")
	}

	// Organize by forecaster first, then by withheld inferer
	for _, forecaster := range args.Forecasters {
		// Get this forecaster's forecast and filter out the withheld inferer
		forecast, ok := args.ForecasterToForecast[forecaster]
		if !ok {
			continue
		}

		for _, withheldInferer := range args.Inferers {
			// Filter out the inferer we want to withhold
			filteredInferers := make([]Inferer, 0, len(args.Inferers)-1)
			filteredInfererToInference := make(map[Inferer]*emissions.Inference, len(args.InfererToInference)-1)
			filteredInfererToRegret := make(map[Inferer]*Regret, len(args.InfererToRegret)-1)

			for _, inferer := range args.Inferers {
				if inferer != withheldInferer {
					filteredInferers = append(filteredInferers, inferer)
					if inference, ok := args.InfererToInference[inferer]; ok {
						filteredInfererToInference[inferer] = inference
					}
					if regret, ok := args.InfererToRegret[inferer]; ok {
						filteredInfererToRegret[inferer] = regret
					}
				}
			}

			filteredForecastElements := make([]*emissions.ForecastElement, 0)
			for _, element := range forecast.ForecastElements {
				if element.Inferer != withheldInferer {
					filteredForecastElements = append(filteredForecastElements, element)
				}
			}

			if len(filteredForecastElements) == 0 {
				continue
			}

			filteredForecast := &emissions.Forecast{
				TopicId:          forecast.TopicId,
				BlockHeight:      forecast.BlockHeight,
				Forecaster:       forecaster,
				ForecastElements: filteredForecastElements,
				ExtraData:        forecast.ExtraData,
			}

			// Create filtered maps with just this forecaster
			filteredForecasters := []Forecaster{forecaster}
			filteredForecasterToForecast := map[Forecaster]*emissions.Forecast{
				forecaster: filteredForecast,
			}
			filteredForecasterToRegret := map[Forecaster]*Regret{}
			if regret, ok := args.ForecasterToRegret[forecaster]; ok {
				filteredForecasterToRegret[forecaster] = regret
			}

			// Calculate forecast-implied inference with the filtered data
			forecastImpliedInferences, calcErr := CalcForecastImpliedInferences(
				CalcForecastImpliedInferencesArgs{
					Logger:                 args.Logger,
					TopicId:                args.TopicId,
					TopicArity:             args.TopicArity,
					AllInferersAreNew:      args.AllInferersAreNew,
					Inferers:               filteredInferers,
					InfererToInference:     filteredInfererToInference,
					InfererToRegret:        filteredInfererToRegret,
					Forecasters:            filteredForecasters,
					ForecasterToForecast:   filteredForecasterToForecast,
					ForecasterToRegret:     filteredForecasterToRegret,
					NetworkCombinedLoss:    args.NetworkCombinedLoss,
					EpsilonTopic:           args.EpsilonTopic,
					PNorm:                  args.PNorm,
					CNorm:                  args.CNorm,
					RegretScalePlusEpsilon: args.RegretScalePlusEpsilon,
					LabelRegistry:          args.LabelRegistry,
					NumLabels:              args.NumLabels,
					LabelDefaultValue:      args.LabelDefaultValue,
				},
			)
			if calcErr != nil {
				args.Logger.Warn("Error calculating forecast implied inference for:", "forecaster", forecaster, "withheldInferer", withheldInferer, "error", calcErr)
				continue
			}

			// Extract the implied inference for this forecaster
			forecastImpliedInference, ok := forecastImpliedInferences[forecaster]
			if !ok {
				continue
			}

			// Add to our results
			oneOutInfererValues, err := emissions.ConvertInferenceValuesToLabeledValues(forecastImpliedInference.Values, args.LabelRegistry)
			if err != nil {
				return nil, errorsmod.Wrap(err, "failed to convert forecast implied inference values")
			}

			oneOutInfererForecastImpliedValues = append(
				oneOutInfererForecastImpliedValues,
				&emissions.OneOutInfererForecasterValue{
					Forecaster:        forecaster,
					WithheldInferer:   withheldInferer,
					CombinedInference: oneOutInfererValues,
				},
			)
		}
	}

	return oneOutInfererForecastImpliedValues, nil
}

// Arguments for GetNaiveInference
type GetNaiveInferenceArgs struct {
	Ctx                                  context.Context
	Logger                               log.Logger
	TopicId                              uint64
	K                                    emissionskeeper.Keeper
	Inferers                             []Worker
	InfererToInference                   map[Worker]*emissions.Inference
	AllInferersAreNew                    bool
	Forecasters                          []Worker
	ForecasterToRegret                   map[Worker]*alloraMath.Dec
	ForecasterToForecastImpliedInference map[Worker]*emissions.Inference
	EpsilonTopic                         alloraMath.Dec
	EpsilonSafeDiv                       alloraMath.Dec
	PNorm                                alloraMath.Dec
	CNorm                                alloraMath.Dec
	RegretScalePlusEpsilon               alloraMath.Dec
	LabelRegistry                        *emissions.EpochLabelRegistry
	NumLabels                            int
}

// Calculates the network naive inference I^-_i
func GetNaiveInference(args GetNaiveInferenceArgs) (labeledNaiveInference []*emissions.LabeledValue, err error) {
	args.Logger.Debug("Calculating naive inference", "topicId", args.TopicId)

	if args.LabelRegistry == nil {
		return []*emissions.LabeledValue{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "GetNaiveInference: LabelRegistry is nil")
	}

	// Get inferer naive regrets
	infererToRegret := make(map[string]*alloraMath.Dec)
	for _, inferer := range args.Inferers {
		regret, _, err := args.K.GetRegretsKeeper().GetNaiveInfererNetworkRegret(args.Ctx, args.TopicId, inferer)
		if err != nil {
			return []*emissions.LabeledValue{}, errorsmod.Wrapf(err, "GetNaiveInference() error getting naive regret for inferer %s", inferer)
		}
		infererToRegret[inferer] = &regret.Value
	}

	weights, err := CalcWeightsGivenWorkers(
		CalcWeightsGivenWorkersArgs{
			Logger:                 args.Logger,
			Inferers:               args.Inferers,
			Forecasters:            args.Forecasters,
			InfererToRegret:        infererToRegret,
			ForecasterToRegret:     make(map[Worker]*alloraMath.Dec, 0),
			EpsilonTopic:           args.EpsilonTopic,
			PNorm:                  args.PNorm,
			CNorm:                  args.CNorm,
			RegretScalePlusEpsilon: args.RegretScalePlusEpsilon,
		},
	)
	if err != nil {
		return []*emissions.LabeledValue{}, errorsmod.Wrap(err, "GetNaiveInference() error calculating weights for naive inference")
	}

	naiveInference, err := calcWeightedInference(calcWeightedInferenceArgs{
		logger:                               args.Logger,
		allInferersAreNew:                    args.AllInferersAreNew,
		inferers:                             args.Inferers,
		workerToInference:                    args.InfererToInference,
		infererToRegret:                      infererToRegret,
		forecasters:                          args.Forecasters,
		forecasterToRegret:                   args.ForecasterToRegret,
		forecasterToForecastImpliedInference: args.ForecasterToForecastImpliedInference,
		weights:                              weights,
		epsilonSafeDiv:                       args.EpsilonSafeDiv,
		numLabels:                            args.NumLabels,
	})
	if err != nil {
		return []*emissions.LabeledValue{}, errorsmod.Wrap(err, "GetNaiveInference() error calculating naive inference")
	}

	args.Logger.Debug("Naive inference calculated", "topicId", args.TopicId)

	labeledNaiveInference, err = emissions.ConvertInferenceValuesToLabeledValues(naiveInference, args.LabelRegistry)
	if err != nil {
		return []*emissions.LabeledValue{}, errorsmod.Wrap(err, "GetNaiveInference() error converting naive inference")
	}

	return labeledNaiveInference, nil
}

// Arguments for calcOneOutInfererInference
type CalcOneOutInfererInferenceArgs struct {
	Ctx                    sdk.Context
	K                      emissionskeeper.Keeper
	Logger                 log.Logger
	TopicId                uint64
	TopicArity             emissions.TopicOutputArity
	Inferers               []Inferer
	InfererToInference     map[Inferer]*emissions.Inference
	InfererToRegret        map[Inferer]*Regret
	AllInferersAreNew      bool
	Forecasters            []Forecaster
	ForecasterToForecast   map[Forecaster]*emissions.Forecast
	ForecasterToRegret     map[Forecaster]*Regret
	NetworkCombinedLoss    *alloraMath.Dec
	EpsilonTopic           alloraMath.Dec
	EpsilonSafeDiv         alloraMath.Dec
	PNorm                  alloraMath.Dec
	CNorm                  alloraMath.Dec
	WithheldInferer        Inferer
	RegretScalePlusEpsilon alloraMath.Dec
	LabelRegistry          *emissions.EpochLabelRegistry
	NumLabels              int
	LabelDefaultValue      alloraMath.Dec
}

// Calculate the one-out inference given a withheld inferer
func calcOneOutInfererInference(args CalcOneOutInfererInferenceArgs) (
	oneOutNetworkInferencesWithoutInferer alloraMath.DecArray,
	err error,
) {
	args.Logger.Debug("calcOneOutInfererInference() calculating one-out inference", "topicId", args.TopicId, "withheldInferer", args.WithheldInferer)

	// To calculate one out, remove the inferer from the list of inferers
	remainingInferers := make([]Worker, 0)
	remainingInfererToInference := make(map[Worker]*emissions.Inference)
	remainingInfererToRegret := make(map[Inferer]*Regret, len(args.InfererToRegret))
	remainingInfererRegrets := make(map[string]*alloraMath.Dec)
	for _, inferer := range args.Inferers {
		// over just the remaining inferers
		if inferer != args.WithheldInferer {
			remainingInferers = append(remainingInferers, inferer)
			inference, ok := args.InfererToInference[inferer]
			if !ok {
				args.Logger.Debug("calcOneOutInfererInference() cannot find inferer in InferenceByWorker in args.InfererToInference", "inferer", inferer)
				continue
			}
			remainingInfererToInference[inferer] = inference
			if regret, ok := args.InfererToRegret[inferer]; ok {
				remainingInfererToRegret[inferer] = regret
			}
		}

		// over every inferer
		regret, _, err := args.K.GetRegretsKeeper().GetOneOutInfererInfererNetworkRegret(args.Ctx, args.TopicId, args.WithheldInferer, inferer)
		if err != nil {
			return alloraMath.DecArray{}, errorsmod.Wrapf(err, "calcOneOutInfererInference() error getting one-out inferer regret")
		}
		remainingInfererRegrets[inferer] = &regret.Value
	}

	remainingForecasterRegrets := make(map[string]*alloraMath.Dec)
	forecasterToForecastImpliedInference := make(map[string]*emissions.Inference)
	if args.NetworkCombinedLoss != nil {
		// Strip the withheld inferer from forecast elements before recomputation,
		// mirroring GetOneOutInfererForecastImpliedInferences.
		remainingForecasterToForecast := make(map[Forecaster]*emissions.Forecast, len(args.ForecasterToForecast))
		for forecaster, forecast := range args.ForecasterToForecast {
			filteredForecastElements := make([]*emissions.ForecastElement, 0, len(forecast.ForecastElements))
			for _, element := range forecast.ForecastElements {
				if element.Inferer != args.WithheldInferer {
					filteredForecastElements = append(filteredForecastElements, element)
				}
			}
			if len(filteredForecastElements) == 0 {
				continue
			}
			remainingForecasterToForecast[forecaster] = &emissions.Forecast{
				TopicId:          forecast.TopicId,
				BlockHeight:      forecast.BlockHeight,
				Forecaster:       forecast.Forecaster,
				ForecastElements: filteredForecastElements,
				ExtraData:        forecast.ExtraData,
			}
		}

		// Recalculate the forecast-implied inferences without the worker's inference
		// This is necessary because the forecast-implied inferences are calculated based on the inferences of the inferers
		forecasterToForecastImpliedInference, err = CalcForecastImpliedInferences(
			CalcForecastImpliedInferencesArgs{
				Logger:                 args.Logger,
				TopicId:                args.TopicId,
				TopicArity:             args.TopicArity,
				AllInferersAreNew:      args.AllInferersAreNew,
				Inferers:               remainingInferers,
				InfererToInference:     remainingInfererToInference,
				InfererToRegret:        remainingInfererToRegret,
				Forecasters:            args.Forecasters,
				ForecasterToForecast:   remainingForecasterToForecast,
				ForecasterToRegret:     args.ForecasterToRegret,
				NetworkCombinedLoss:    args.NetworkCombinedLoss,
				EpsilonTopic:           args.EpsilonTopic,
				PNorm:                  args.PNorm,
				CNorm:                  args.CNorm,
				RegretScalePlusEpsilon: args.RegretScalePlusEpsilon,
				LabelRegistry:          args.LabelRegistry,
				NumLabels:              args.NumLabels,
				LabelDefaultValue:      args.LabelDefaultValue,
			},
		)
		if err != nil {
			return alloraMath.DecArray{}, errorsmod.Wrapf(err, "calcOneOutInfererInference() error recalculating forecast-implied inferences")
		}

		// Get regrets for the forecasters
		for _, forecaster := range args.Forecasters {
			regret, _, err := args.K.GetRegretsKeeper().GetOneOutInfererForecasterNetworkRegret(args.Ctx, args.TopicId, args.WithheldInferer, forecaster)
			if err != nil {
				return alloraMath.DecArray{}, errorsmod.Wrapf(err, "calcOneOutInfererInference() error getting one-out forecaster regret")
			}
			remainingForecasterRegrets[forecaster] = &regret.Value
		}
	}
	weights, err := CalcWeightsGivenWorkers(
		CalcWeightsGivenWorkersArgs{
			Logger:                 args.Logger,
			Inferers:               remainingInferers,
			Forecasters:            args.Forecasters,
			InfererToRegret:        remainingInfererRegrets,
			ForecasterToRegret:     remainingForecasterRegrets,
			EpsilonTopic:           args.EpsilonTopic,
			PNorm:                  args.PNorm,
			CNorm:                  args.CNorm,
			RegretScalePlusEpsilon: args.RegretScalePlusEpsilon,
		},
	)
	if err != nil {
		return alloraMath.DecArray{}, errorsmod.Wrapf(err, "calcOneOutInfererInference() error calculating one-out inference for forecaster")
	}

	oneOutNetworkInferencesWithoutInferer, err = calcWeightedInference(calcWeightedInferenceArgs{
		logger:                               args.Logger,
		allInferersAreNew:                    args.AllInferersAreNew,
		inferers:                             remainingInferers,
		workerToInference:                    remainingInfererToInference,
		infererToRegret:                      remainingInfererRegrets,
		forecasters:                          args.Forecasters,
		forecasterToRegret:                   remainingForecasterRegrets,
		forecasterToForecastImpliedInference: forecasterToForecastImpliedInference,
		weights:                              weights,
		epsilonSafeDiv:                       args.EpsilonSafeDiv,
		numLabels:                            args.NumLabels,
	})
	if err != nil {
		return alloraMath.DecArray{}, errorsmod.Wrapf(err, "calcOneOutInfererInference() error calculating one-out inference for inferer")
	}

	args.Logger.Debug("One-out inference calculated", "topicId", args.TopicId, "withheldInferer", args.WithheldInferer, "oneOutNetworkInferencesWithoutInferer", oneOutNetworkInferencesWithoutInferer)
	return oneOutNetworkInferencesWithoutInferer, nil
}

// args for GetOneOutInfererInferences
type GetOneOutInfererInferencesArgs struct {
	Ctx                    sdk.Context
	K                      emissionskeeper.Keeper
	Logger                 log.Logger
	TopicId                uint64
	TopicArity             emissions.TopicOutputArity
	Inferers               []Inferer
	InfererToInference     map[Inferer]*emissions.Inference
	InfererToRegret        map[Inferer]*Regret
	AllInferersAreNew      bool
	Forecasters            []Forecaster
	ForecasterToForecast   map[Forecaster]*emissions.Forecast
	ForecasterToRegret     map[Forecaster]*Regret
	NetworkCombinedLoss    *alloraMath.Dec
	EpsilonTopic           alloraMath.Dec
	EpsilonSafeDiv         alloraMath.Dec
	PNorm                  alloraMath.Dec
	CNorm                  alloraMath.Dec
	RegretScalePlusEpsilon alloraMath.Dec
	LabelRegistry          *emissions.EpochLabelRegistry
	NumLabels              int
	LabelDefaultValue      alloraMath.Dec
}

// Set all one-out-inferer inferences that are possible given the provided input
// Assumed that there is at most 1 inference per inferer
// Loop over all inferences and withhold one, then calculate the network inference less that withheld inference
// This involves recalculating the forecast-implied inferences for each withheld inferer
func GetOneOutInfererInferences(args GetOneOutInfererInferencesArgs) (
	oneOutInfererInferences []*emissions.OneOutInfererValue,
	err error,
) {
	args.Logger.Debug("Calculating one-out inferer inferences", "topicId", args.TopicId, "numInferers", len(args.Inferers))

	if args.LabelRegistry == nil {
		return []*emissions.OneOutInfererValue{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "GetOneOutInfererInferences: LabelRegistry is nil")
	}

	// Calculate the one-out inferences per inferer
	oneOutInferences := make([]*emissions.OneOutInfererValue, 0)
	for _, worker := range args.Inferers {
		oneOutInference, err := calcOneOutInfererInference(
			CalcOneOutInfererInferenceArgs{
				Ctx:                    args.Ctx,
				K:                      args.K,
				Logger:                 args.Logger,
				TopicId:                args.TopicId,
				TopicArity:             args.TopicArity,
				Inferers:               args.Inferers,
				InfererToInference:     args.InfererToInference,
				InfererToRegret:        args.InfererToRegret,
				AllInferersAreNew:      args.AllInferersAreNew,
				Forecasters:            args.Forecasters,
				ForecasterToForecast:   args.ForecasterToForecast,
				ForecasterToRegret:     args.ForecasterToRegret,
				NetworkCombinedLoss:    args.NetworkCombinedLoss,
				EpsilonTopic:           args.EpsilonTopic,
				EpsilonSafeDiv:         args.EpsilonSafeDiv,
				PNorm:                  args.PNorm,
				CNorm:                  args.CNorm,
				WithheldInferer:        worker,
				RegretScalePlusEpsilon: args.RegretScalePlusEpsilon,
				LabelRegistry:          args.LabelRegistry,
				NumLabels:              args.NumLabels,
				LabelDefaultValue:      args.LabelDefaultValue,
			})
		if err != nil {
			return []*emissions.OneOutInfererValue{}, errorsmod.Wrapf(err, "GetOneOutInfererInferences() error calculating one-out inferer inferences")
		}

		combinedInference, err := emissions.ConvertInferenceValuesToLabeledValues(oneOutInference, args.LabelRegistry)
		if err != nil {
			return []*emissions.OneOutInfererValue{}, errorsmod.Wrapf(err, "GetOneOutForecasterInferences() error converting one-out inferer inferences")
		}
		oneOutInferences = append(oneOutInferences, &emissions.OneOutInfererValue{
			WithheldInferer:   worker,
			CombinedInference: combinedInference,
		})
	}

	args.Logger.Debug("One-out inferer inferences calculated", "topicId", args.TopicId)
	return oneOutInferences, nil
}

// Arguments for calcOneOutForecasterInferences
type CalcOneOutForecasterInferenceArgs struct {
	Ctx                                  sdk.Context
	K                                    emissionskeeper.Keeper
	Logger                               log.Logger
	TopicId                              uint64
	Inferers                             []Inferer
	InfererToInference                   map[Inferer]*emissions.Inference
	InfererToRegret                      map[Inferer]*Regret
	AllInferersAreNew                    bool
	Forecasters                          []Forecaster
	ForecasterToForecast                 map[Forecaster]*emissions.Forecast
	ForecasterToForecastImpliedInference map[Forecaster]*emissions.Inference
	EpsilonTopic                         alloraMath.Dec
	EpsilonSafeDiv                       alloraMath.Dec
	PNorm                                alloraMath.Dec
	CNorm                                alloraMath.Dec
	WithheldForecaster                   Forecaster
	RegretScalePlusEpsilon               alloraMath.Dec
	NumLabels                            int
}

// Calculate the one-out inference given a withheld forecaster
func calcOneOutForecasterInferences(args CalcOneOutForecasterInferenceArgs) (
	oneOutNetworkInferencesWithoutInferer alloraMath.DecArray,
	err error,
) {
	args.Logger.Debug("Calculating one-out inference", "topicId", args.TopicId, "withheldForecaster", args.WithheldForecaster)

	// To calculate one out, remove the withheldForecaster from the list of forecasters
	remainingForecasters := make([]Forecaster, 0)
	remainingForecasterToForecast := make(map[Forecaster]*emissions.Forecast)
	remainingForecasterRegrets := make(map[Forecaster]*Regret)
	for _, forecaster := range args.Forecasters {
		if forecaster != args.WithheldForecaster {
			remainingForecasters = append(remainingForecasters, forecaster)

			regret, _, err := args.K.GetRegretsKeeper().GetOneOutForecasterForecasterNetworkRegret(args.Ctx, args.TopicId, args.WithheldForecaster, forecaster)
			if err != nil {
				return alloraMath.DecArray{}, errorsmod.Wrapf(err, "calcOneOutForecasterInferences() error getting one-out forecaster regret")
			}
			remainingForecasterRegrets[forecaster] = &regret.Value

			forecast, ok := args.ForecasterToForecast[forecaster]
			if !ok {
				return alloraMath.DecArray{}, errorsmod.Wrapf(emissions.ErrNotFound, "calcOneOutForecasterInferences() cannot find forecaster in ForecasterRegrets %v", forecaster)
			}
			remainingForecasterToForecast[forecaster] = forecast
		}
	}

	// Get regrets for the remaining inferers
	remainingInfererRegrets := make(map[Inferer]*Regret)
	for _, inferer := range args.Inferers {
		regret, _, err := args.K.GetRegretsKeeper().GetOneOutForecasterInfererNetworkRegret(args.Ctx, args.TopicId, args.WithheldForecaster, inferer)
		if err != nil {
			return alloraMath.DecArray{}, errorsmod.Wrapf(err, "calcOneOutForecasterInferences() error getting one-out inferer regret")
		}
		remainingInfererRegrets[inferer] = &regret.Value
	}

	weights, err := CalcWeightsGivenWorkers(
		CalcWeightsGivenWorkersArgs{
			Logger:                 args.Logger,
			Inferers:               args.Inferers,
			Forecasters:            remainingForecasters,
			InfererToRegret:        remainingInfererRegrets,
			ForecasterToRegret:     remainingForecasterRegrets,
			EpsilonTopic:           args.EpsilonTopic,
			PNorm:                  args.PNorm,
			CNorm:                  args.CNorm,
			RegretScalePlusEpsilon: args.RegretScalePlusEpsilon,
		},
	)
	if err != nil {
		return alloraMath.DecArray{}, errorsmod.Wrapf(err, "calcOneOutForecasterInferences() error calculating one-out inference for forecaster")
	}

	oneOutNetworkInferencesWithoutInferer, err = calcWeightedInference(calcWeightedInferenceArgs{
		logger:                               args.Logger,
		allInferersAreNew:                    args.AllInferersAreNew,
		inferers:                             args.Inferers,
		workerToInference:                    args.InfererToInference,
		infererToRegret:                      args.InfererToRegret,
		forecasters:                          remainingForecasters,
		forecasterToRegret:                   remainingForecasterRegrets,
		forecasterToForecastImpliedInference: args.ForecasterToForecastImpliedInference,
		weights:                              weights,
		epsilonSafeDiv:                       args.EpsilonSafeDiv,
		numLabels:                            args.NumLabels,
	})
	if err != nil {
		return alloraMath.DecArray{}, errorsmod.Wrapf(err, "calcOneOutForecasterInferences() error calculating one-out inference for inferer")
	}

	args.Logger.Debug("One-out inference calculated", "topicId", args.TopicId, "withheldForecaster", args.WithheldForecaster, "oneOutNetworkInferencesWithoutInferer", oneOutNetworkInferencesWithoutInferer)
	return oneOutNetworkInferencesWithoutInferer, nil
}

// GetOneOutForecasterInferencesArgs is the set of arguments for the GetOneOutForecasterInferences function
type GetOneOutForecasterInferencesArgs struct {
	Ctx                                  sdk.Context
	K                                    emissionskeeper.Keeper
	Logger                               log.Logger
	TopicId                              uint64
	Inferers                             []Inferer
	InfererToInference                   map[Inferer]*emissions.Inference
	InfererToRegret                      map[Inferer]*Regret
	AllInferersAreNew                    bool
	Forecasters                          []Forecaster
	ForecasterToForecast                 map[Forecaster]*emissions.Forecast
	ForecasterToRegret                   map[Forecaster]*Regret
	ForecasterToForecastImpliedInference map[Forecaster]*emissions.Inference
	EpsilonTopic                         alloraMath.Dec
	EpsilonSafeDiv                       alloraMath.Dec
	PNorm                                alloraMath.Dec
	CNorm                                alloraMath.Dec
	RegretScalePlusEpsilon               alloraMath.Dec
	LabelRegistry                        *emissions.EpochLabelRegistry
	NumLabels                            int
}

// Set all one-out-forecaster inferences that are possible given the provided input
// Assume that there is at most 1 forecast-implied inference per forecaster
// Loop over all forecast-implied inferences and withhold one, then calculate the network inference less that withheld value
func GetOneOutForecasterInferences(args GetOneOutForecasterInferencesArgs) (
	oneOutForecasterInferences []*emissions.OneOutForecasterValue,
	err error,
) {
	args.Logger.Debug("Calculating one-out forecaster inferences", "topicId", args.TopicId, "numForecasters", len(args.Forecasters))

	if args.LabelRegistry == nil {
		return []*emissions.OneOutForecasterValue{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "GetOneOutForecasterInferences: LabelRegistry is nil")
	}

	// Calculate the one-out forecast-implied inferences per forecaster
	oneOutForecasterInferences = make([]*emissions.OneOutForecasterValue, 0)
	// If there is only one forecaster, there's no need to calculate one-out inferences
	if len(args.Forecasters) > 1 {
		for _, worker := range args.Forecasters {
			oneOutInferences, err := calcOneOutForecasterInferences(
				CalcOneOutForecasterInferenceArgs{
					Ctx:                                  args.Ctx,
					K:                                    args.K,
					Logger:                               args.Logger,
					TopicId:                              args.TopicId,
					Inferers:                             args.Inferers,
					InfererToInference:                   args.InfererToInference,
					InfererToRegret:                      args.InfererToRegret,
					AllInferersAreNew:                    args.AllInferersAreNew,
					Forecasters:                          args.Forecasters,
					ForecasterToForecast:                 args.ForecasterToForecast,
					ForecasterToForecastImpliedInference: args.ForecasterToForecastImpliedInference,
					EpsilonTopic:                         args.EpsilonTopic,
					EpsilonSafeDiv:                       args.EpsilonSafeDiv,
					PNorm:                                args.PNorm,
					CNorm:                                args.CNorm,
					WithheldForecaster:                   worker,
					RegretScalePlusEpsilon:               args.RegretScalePlusEpsilon,
					NumLabels:                            args.NumLabels,
				})
			if err != nil {
				return []*emissions.OneOutForecasterValue{}, errorsmod.Wrapf(err, "GetOneOutForecasterInferences() error calculating one-out forecaster inferences")
			}
			combinedInference, err := emissions.ConvertInferenceValuesToLabeledValues(oneOutInferences, args.LabelRegistry)
			if err != nil {
				return []*emissions.OneOutForecasterValue{}, errorsmod.Wrapf(err, "GetOneOutForecasterInferences() error converting one-out forecaster inferences")
			}
			oneOutForecasterInferences = append(oneOutForecasterInferences, &emissions.OneOutForecasterValue{
				WithheldForecaster: worker,
				CombinedInference:  combinedInference,
			})
		}
		args.Logger.Debug("One-out forecaster inferences calculated", "topicId", args.TopicId)
	}
	return oneOutForecasterInferences, nil
}

// Arguments to calcOneInValues
type calcOneInValuesArgs struct {
	Ctx                                  sdk.Context
	K                                    emissionskeeper.Keeper
	Logger                               log.Logger
	TopicId                              uint64
	AllInferersAreNew                    bool
	Inferers                             []Inferer
	InfererToInference                   map[Inferer]*emissions.Inference
	ForecasterToForecastImpliedInference map[Forecaster]*emissions.Inference
	EpsilonTopic                         alloraMath.Dec
	EpsilonSafeDiv                       alloraMath.Dec
	PNorm                                alloraMath.Dec
	CNorm                                alloraMath.Dec
	OneInForecaster                      Forecaster
	RegretScalePlusEpsilon               alloraMath.Dec
	NumLabels                            int
}

// Calculate the one-in inference given a single included forecaster.
func calcOneInValues(args calcOneInValuesArgs) (alloraMath.DecArray, error) {
	args.Logger.Debug("Calculating one-in inference", "forecaster", args.OneInForecaster)

	// Need the forecaster's forecast-implied inference to do "one-in".
	fi, ok := args.ForecasterToForecastImpliedInference[args.OneInForecaster]
	if !ok || fi == nil {
		return alloraMath.DecArray{}, errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"missing forecast-implied inference for one-in forecaster %s",
			args.OneInForecaster,
		)
	}

	// Ensure we have a self-regret entry for this forecaster (weighting path expects it).
	selfRegret, _, err := args.K.GetRegretsKeeper().GetOneInForecasterNetworkRegret(args.Ctx, args.TopicId, args.OneInForecaster, args.OneInForecaster)
	if err != nil {
		return alloraMath.DecArray{}, errorsmod.Wrapf(err, "calcOneInValues: error getting one-in forecaster self regret")
	}
	singleForecasterRegret := map[Worker]*Regret{
		args.OneInForecaster: &selfRegret.Value,
	}

	// Build inferer regrets/inferences for this single-forecaster view.
	// Only include inferers that actually have an inference.
	infererToRegretForSingleForecaster := make(map[Inferer]*Regret, len(args.Inferers))
	infererToInferenceForSingleForecaster := make(map[Inferer]*emissions.Inference, len(args.Inferers))

	for _, inferer := range args.Inferers {
		inf, ok := args.InfererToInference[inferer]
		if !ok || inf == nil {
			args.Logger.Debug("calcOneInValues: missing inferer inference, skipping", "inferer", inferer)
			continue
		}
		infererToInferenceForSingleForecaster[inferer] = inf

		r, _, err := args.K.GetRegretsKeeper().GetOneInForecasterNetworkRegret(args.Ctx, args.TopicId, args.OneInForecaster, inferer)
		if err != nil {
			return alloraMath.DecArray{}, errorsmod.Wrapf(err, "calcOneInValues: error getting one-in forecaster regret for inferer %s", inferer)
		}
		infererToRegretForSingleForecaster[inferer] = &r.Value
	}

	singleForecasters := []Worker{args.OneInForecaster}

	weights, err := CalcWeightsGivenWorkers(
		CalcWeightsGivenWorkersArgs{
			Logger:                 args.Logger,
			Inferers:               args.Inferers,
			Forecasters:            singleForecasters,
			InfererToRegret:        infererToRegretForSingleForecaster,
			ForecasterToRegret:     singleForecasterRegret,
			EpsilonTopic:           args.EpsilonTopic,
			PNorm:                  args.PNorm,
			CNorm:                  args.CNorm,
			RegretScalePlusEpsilon: args.RegretScalePlusEpsilon,
		},
	)
	if err != nil {
		return alloraMath.DecArray{}, errorsmod.Wrapf(err, "calcOneInValues: error calculating weights for one-in inference")
	}

	// IMPORTANT: for true "one-in", pass ONLY this forecaster's forecast-implied inference map.
	singleForecasterToFI := map[Forecaster]*emissions.Inference{
		args.OneInForecaster: fi,
	}

	oneInInference, err := calcWeightedInference(
		calcWeightedInferenceArgs{
			logger:                               args.Logger,
			allInferersAreNew:                    args.AllInferersAreNew,
			inferers:                             args.Inferers,
			workerToInference:                    infererToInferenceForSingleForecaster,
			infererToRegret:                      infererToRegretForSingleForecaster,
			forecasters:                          singleForecasters,
			forecasterToRegret:                   singleForecasterRegret,
			forecasterToForecastImpliedInference: singleForecasterToFI,
			weights:                              weights,
			epsilonSafeDiv:                       args.EpsilonSafeDiv,
			numLabels:                            args.NumLabels,
		},
	)
	if err != nil {
		return alloraMath.DecArray{}, errorsmod.Wrapf(err, "calcOneInValues: error calculating one-in inference")
	}

	return oneInInference, nil
}

// Arguments to GetOneInForecasterInferences
type GetOneInForecasterInferencesArgs struct {
	Ctx                                  sdk.Context
	K                                    emissionskeeper.Keeper
	Logger                               log.Logger
	TopicId                              uint64
	Inferers                             []Inferer
	InfererToInference                   map[Inferer]*emissions.Inference
	AllInferersAreNew                    bool
	Forecasters                          []Forecaster
	ForecasterToForecastImpliedInference map[Forecaster]*emissions.Inference
	EpsilonTopic                         alloraMath.Dec
	EpsilonSafeDiv                       alloraMath.Dec
	PNorm                                alloraMath.Dec
	CNorm                                alloraMath.Dec
	RegretScalePlusEpsilon               alloraMath.Dec
	LabelRegistry                        *emissions.EpochLabelRegistry
	NumLabels                            int
}

// Set all one-in inferences that are possible given the provided input
// Assumed that there is at most 1 inference per worker.
// Also assume that there is at most 1 forecast-implied inference per worker.
func GetOneInForecasterInferences(args GetOneInForecasterInferencesArgs) (
	oneInInferences []*emissions.OneInForecasterValue,
	err error,
) {
	if args.LabelRegistry == nil {
		return []*emissions.OneInForecasterValue{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "GetOneInForecasterInferences: LabelRegistry is nil")
	}

	// Loop over all forecast-implied inferences and set it as the only forecast-implied inference
	// one at a time, then calculate the network inference given that one held out
	oneInInferences = make([]*emissions.OneInForecasterValue, 0)
	// If there is only one forecaster, thre's no need to calculate one-in inferences
	if len(args.Forecasters) > 1 {
		for _, oneInForecaster := range args.Forecasters {
			oneInValues, err := calcOneInValues(
				calcOneInValuesArgs{
					Ctx:                                  args.Ctx,
					K:                                    args.K,
					Logger:                               args.Logger,
					TopicId:                              args.TopicId,
					AllInferersAreNew:                    args.AllInferersAreNew,
					Inferers:                             args.Inferers,
					InfererToInference:                   args.InfererToInference,
					ForecasterToForecastImpliedInference: args.ForecasterToForecastImpliedInference,
					EpsilonTopic:                         args.EpsilonTopic,
					EpsilonSafeDiv:                       args.EpsilonSafeDiv,
					PNorm:                                args.PNorm,
					CNorm:                                args.CNorm,
					OneInForecaster:                      oneInForecaster,
					RegretScalePlusEpsilon:               args.RegretScalePlusEpsilon,
					NumLabels:                            args.NumLabels,
				})
			if err != nil {
				return []*emissions.OneInForecasterValue{}, errorsmod.Wrapf(err, "GetOneInForecasterInferences() error calculating one-in inferences")
			}
			combinedInference, err := emissions.ConvertInferenceValuesToLabeledValues(oneInValues, args.LabelRegistry)
			if err != nil {
				return []*emissions.OneInForecasterValue{}, errorsmod.Wrapf(err, "GetOneInForecasterInferences() error converting one-in inferences")
			}
			oneInInferences = append(oneInInferences, &emissions.OneInForecasterValue{
				Forecaster:        oneInForecaster,
				CombinedInference: combinedInference,
			})
		}
	}
	return oneInInferences, err
}

// Arguments for CalcNetworkInferences
type CalcNetworkInferencesArgs struct {
	Ctx                                  sdk.Context
	K                                    emissionskeeper.Keeper
	Logger                               log.Logger
	TopicId                              uint64
	TopicArity                           emissions.TopicOutputArity
	Inferers                             []Inferer
	InfererToInference                   map[Inferer]*emissions.Inference
	InfererToRegret                      map[Inferer]*Regret
	AllInferersAreNew                    bool
	Forecasters                          []Forecaster
	ForecasterToForecast                 map[Forecaster]*emissions.Forecast
	ForecasterToRegret                   map[Forecaster]*Regret
	ForecasterToForecastImpliedInference map[Forecaster]*emissions.Inference
	NetworkCombinedLoss                  *alloraMath.Dec
	EpsilonTopic                         alloraMath.Dec
	EpsilonSafeDiv                       alloraMath.Dec
	PNorm                                alloraMath.Dec
	CNorm                                alloraMath.Dec
	RegretScalePlusEpsilon               alloraMath.Dec
	InferenceBlockHeight                 BlockHeight
	LabelRegistry                        *emissions.EpochLabelRegistry
	NumLabels                            int
	LabelDefaultValue                    alloraMath.Dec
}

// Calculates all network inferences in the set I_i given historical state (e.g. regrets)
// and data from workers (e.g. inferences, forecast-implied inferences).
// Could improve this with Builder pattern, as for other instances of generated ValueBundles.
func CalcNetworkInferences(
	args CalcNetworkInferencesArgs,
) (
	inferenceBundle *emissions.NetworkInferenceBundle,
	weights RegretInformedWeights,
	err error,
) {
	if args.LabelRegistry == nil {
		return &emissions.NetworkInferenceBundle{}, RegretInformedWeights{}, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "CalcNetworkInferences: LabelRegistry is nil")
	}

	// first get the network combined inference I_i
	// which is the end result of all this work, the actual combined
	// inference from all of the inferers put together
	args.Logger.Info("CalcNetworkInferences() starting")
	weights, combinedInference, err := GetCombinedInference(
		GetCombinedInferenceArgs{
			Logger:                               args.Logger,
			TopicId:                              args.TopicId,
			Inferers:                             args.Inferers,
			InfererToInference:                   args.InfererToInference,
			InfererToRegret:                      args.InfererToRegret,
			AllInferersAreNew:                    args.AllInferersAreNew,
			Forecasters:                          args.Forecasters,
			ForecasterToRegret:                   args.ForecasterToRegret,
			ForecasterToForecastImpliedInference: args.ForecasterToForecastImpliedInference,
			EpsilonTopic:                         args.EpsilonTopic,
			EpsilonSafeDiv:                       args.EpsilonSafeDiv,
			PNorm:                                args.PNorm,
			CNorm:                                args.CNorm,
			RegretScalePlusEpsilon:               args.RegretScalePlusEpsilon,
			NumLabels:                            args.NumLabels,
		})
	if err != nil {
		return &emissions.NetworkInferenceBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "CalcNetworkInferences() error calculating combined inference")
	}
	// get all the inferences which is all I_ij
	inferences, err := getInferences(args.Inferers, args.InfererToInference, args.LabelRegistry)
	if err != nil {
		return &emissions.NetworkInferenceBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "CalcNetworkInferences() error calculating inferences")
	}
	// get all the forecast-implied inferences which is all I_ik
	forecastImpliedInferences, err := getForecastImpliedInferences(
		args.Logger,
		args.Forecasters,
		args.ForecasterToForecastImpliedInference,
		args.LabelRegistry,
	)
	if err != nil {
		return &emissions.NetworkInferenceBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "CalcNetworkInferences() error calculating forecast implied inferences")
	}

	// get the naive network inference I^-_i
	// The naive network inference is used to quantify the contribution of the
	// forecasting task to the network accuracy, which in turn sets the reward
	// distribution between the inference and forecasting tasks
	naiveInference, err := GetNaiveInference(
		GetNaiveInferenceArgs{
			Ctx:                                  args.Ctx,
			Logger:                               args.Logger,
			TopicId:                              args.TopicId,
			K:                                    args.K,
			Inferers:                             args.Inferers,
			InfererToInference:                   args.InfererToInference,
			AllInferersAreNew:                    args.AllInferersAreNew,
			Forecasters:                          args.Forecasters,
			ForecasterToRegret:                   args.ForecasterToRegret,
			ForecasterToForecastImpliedInference: args.ForecasterToForecastImpliedInference,
			EpsilonTopic:                         args.EpsilonTopic,
			EpsilonSafeDiv:                       args.EpsilonSafeDiv,
			PNorm:                                args.PNorm,
			CNorm:                                args.CNorm,
			RegretScalePlusEpsilon:               args.RegretScalePlusEpsilon,
			LabelRegistry:                        args.LabelRegistry,
			NumLabels:                            args.NumLabels,
		})
	if err != nil {
		return &emissions.NetworkInferenceBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "CalcNetworkInferences() error calculating naive inference")
	}

	// Get the one-out inferer inferences I^-_li over I_ij
	// The one-out network inferer inferences represent an approximation of
	// Shapley (1953) values and are used to quantify the individual
	// contributions of inferers to the network accuracy,
	// which in turn sets the reward distribution between inferers.
	// The one-out network inferer inferences are also used to
	// calculate confidence intervals on the network inference I_i
	oneOutInfererInferences, err := GetOneOutInfererInferences(
		GetOneOutInfererInferencesArgs{
			Ctx:                    args.Ctx,
			K:                      args.K,
			Logger:                 args.Logger,
			TopicId:                args.TopicId,
			TopicArity:             args.TopicArity,
			Inferers:               args.Inferers,
			InfererToInference:     args.InfererToInference,
			InfererToRegret:        args.InfererToRegret,
			AllInferersAreNew:      args.AllInferersAreNew,
			Forecasters:            args.Forecasters,
			ForecasterToForecast:   args.ForecasterToForecast,
			ForecasterToRegret:     args.ForecasterToRegret,
			NetworkCombinedLoss:    args.NetworkCombinedLoss,
			EpsilonTopic:           args.EpsilonTopic,
			EpsilonSafeDiv:         args.EpsilonSafeDiv,
			PNorm:                  args.PNorm,
			CNorm:                  args.CNorm,
			RegretScalePlusEpsilon: args.RegretScalePlusEpsilon,
			LabelRegistry:          args.LabelRegistry,
			NumLabels:              args.NumLabels,
			LabelDefaultValue:      args.LabelDefaultValue,
		})
	if err != nil {
		return &emissions.NetworkInferenceBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "CalcNetworkInferences() error calculating one-out inferer inferences")
	}

	// get the one-out forecaster inferences I^-_li over I_ik
	// The one-out network forecaster inferences represent an approximation of
	// Shapley (1953) values and are used to quantify the individual
	// contributions of forecasters to the network accuracy,
	// which in turn sets the reward distribution between forecasters.
	// The one-out network forecaster inferences are also used to
	// calculate confidence intervals on the network inference I_i
	var oneOutForecasterInferences []*emissions.OneOutForecasterValue
	var oneInForecasterInferences []*emissions.OneInForecasterValue
	var oneOutInfererForecastImpliedValues []*emissions.OneOutInfererForecasterValue
	if args.NetworkCombinedLoss != nil {
		oneOutForecasterInferences, err = GetOneOutForecasterInferences(
			GetOneOutForecasterInferencesArgs{
				Ctx:                                  args.Ctx,
				K:                                    args.K,
				Logger:                               args.Logger,
				TopicId:                              args.TopicId,
				Inferers:                             args.Inferers,
				InfererToInference:                   args.InfererToInference,
				InfererToRegret:                      args.InfererToRegret,
				AllInferersAreNew:                    args.AllInferersAreNew,
				Forecasters:                          args.Forecasters,
				ForecasterToForecast:                 args.ForecasterToForecast,
				ForecasterToRegret:                   args.ForecasterToRegret,
				ForecasterToForecastImpliedInference: args.ForecasterToForecastImpliedInference,
				EpsilonTopic:                         args.EpsilonTopic,
				EpsilonSafeDiv:                       args.EpsilonSafeDiv,
				PNorm:                                args.PNorm,
				CNorm:                                args.CNorm,
				RegretScalePlusEpsilon:               args.RegretScalePlusEpsilon,
				LabelRegistry:                        args.LabelRegistry,
				NumLabels:                            args.NumLabels,
			})
		if err != nil {
			return &emissions.NetworkInferenceBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "CalcNetworkInferences() error calculating one-out forecaster inferences")
		}
		// get the one-in forecaster inferences I^+_ki
		// which adds only a single forecast-implied inference I_ik to the inferences
		// from the inference task I_ij . As such, it is used to quantify how the naive
		// network inference I^-_i changes with the addition of a single
		// forecast-implied inference, which in turn is used for setting the
		// reward distribution between workers for their forecasting tasks.
		oneInForecasterInferences, err = GetOneInForecasterInferences(
			GetOneInForecasterInferencesArgs{
				Ctx:                                  args.Ctx,
				K:                                    args.K,
				Logger:                               args.Logger,
				TopicId:                              args.TopicId,
				Inferers:                             args.Inferers,
				InfererToInference:                   args.InfererToInference,
				AllInferersAreNew:                    args.AllInferersAreNew,
				Forecasters:                          args.Forecasters,
				ForecasterToForecastImpliedInference: args.ForecasterToForecastImpliedInference,
				EpsilonTopic:                         args.EpsilonTopic,
				EpsilonSafeDiv:                       args.EpsilonSafeDiv,
				PNorm:                                args.PNorm,
				CNorm:                                args.CNorm,
				RegretScalePlusEpsilon:               args.RegretScalePlusEpsilon,
				LabelRegistry:                        args.LabelRegistry,
				NumLabels:                            args.NumLabels,
			})
		if err != nil {
			return &emissions.NetworkInferenceBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "CalcNetworkInferences() error calculating one-in inferences")
		}

		// Calculate one-out inferer forecast implied values
		oneOutInfererForecastImpliedValues, err = GetOneOutInfererForecastImpliedInferences(
			GetOneOutInfererForecastImpliedInferencesArgs{
				Ctx:                    args.Ctx,
				K:                      args.K,
				Logger:                 args.Logger,
				TopicId:                args.TopicId,
				TopicArity:             args.TopicArity,
				Inferers:               args.Inferers,
				InfererToInference:     args.InfererToInference,
				InfererToRegret:        args.InfererToRegret,
				AllInferersAreNew:      args.AllInferersAreNew,
				Forecasters:            args.Forecasters,
				ForecasterToForecast:   args.ForecasterToForecast,
				ForecasterToRegret:     args.ForecasterToRegret,
				NetworkCombinedLoss:    args.NetworkCombinedLoss,
				EpsilonTopic:           args.EpsilonTopic,
				PNorm:                  args.PNorm,
				CNorm:                  args.CNorm,
				RegretScalePlusEpsilon: args.RegretScalePlusEpsilon,
				LabelRegistry:          args.LabelRegistry,
				NumLabels:              args.NumLabels,
				LabelDefaultValue:      args.LabelDefaultValue,
			},
		)
		if err != nil {
			return nil, RegretInformedWeights{}, errorsmod.Wrap(err, "while calculating one-out inferer forecast implied inferences")
		}
	}

	combinedInferenceLabeled, err := emissions.ConvertInferenceValuesToLabeledValues(combinedInference, args.LabelRegistry)
	if err != nil {
		return &emissions.NetworkInferenceBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "CalcNetworkInferences() error converting combined values")
	}

	// Build value bundle to return all the calculated inferences
	return &emissions.NetworkInferenceBundle{
		TopicId:                       args.TopicId,
		Nonce:                         args.InferenceBlockHeight,
		CombinedValue:                 combinedInferenceLabeled,
		InfererValues:                 inferences,
		ForecasterValues:              forecastImpliedInferences,
		NaiveValue:                    naiveInference,
		OneOutInfererValues:           oneOutInfererInferences,
		OneOutForecasterValues:        oneOutForecasterInferences,
		OneInForecasterValues:         oneInForecasterInferences,
		OneOutInfererForecasterValues: oneOutInfererForecastImpliedValues,
	}, weights, err
}
