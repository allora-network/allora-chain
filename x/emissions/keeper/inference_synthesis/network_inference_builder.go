package inferencesynthesis

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Arguments for GetCombinedInference
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
	StdDevPlusEpsilon                    alloraMath.Dec
}

// Calculates the network combined inference I_i, Equation 9
func GetCombinedInference(args GetCombinedInferenceArgs) (
	weights RegretInformedWeights, combinedInference InferenceValue, err error) {
	args.Logger.Debug("Calculating combined inference", "topicId", args.TopicId)

	weights, err = CalcWeightsGivenWorkers(
		CalcWeightsGivenWorkersArgs{
			Logger:             args.Logger,
			Inferers:           args.Inferers,
			Forecasters:        args.Forecasters,
			InfererToRegret:    args.InfererToRegret,
			ForecasterToRegret: args.ForecasterToRegret,
			EpsilonTopic:       args.EpsilonTopic,
			PNorm:              args.PNorm,
			CNorm:              args.CNorm,
			StdDevPlusEpsilon:  args.StdDevPlusEpsilon,
		},
	)
	if err != nil {
		return RegretInformedWeights{}, InferenceValue{}, errorsmod.Wrap(err, "GetCombinedInference() error calculating weights for combined inference")
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
	})
	if err != nil {
		return RegretInformedWeights{}, InferenceValue{}, errorsmod.Wrap(err, "GetCombinedInference() error calculating combined inference")
	}

	args.Logger.Debug("Combined inference calculated", "topicId", args.TopicId)
	return weights, combinedInference, nil
}

// Map inferences to a WorkerAttributedValue array and set
func getInferences(
	inferers []Worker,
	infererToInference map[Worker]*emissions.Inference,
) (infererValues []*emissions.WorkerAttributedValue) {
	infererValues = make([]*emissions.WorkerAttributedValue, 0)
	for _, inferer := range inferers {
		infererValues = append(infererValues, &emissions.WorkerAttributedValue{
			Worker: inferer,
			Value:  infererToInference[inferer].Value,
		})
	}
	return infererValues
}

// Map forecast-implied inferences to a WorkerAttributedValue array and set
func getForecastImpliedInferences(
	logger log.Logger,
	forecasters []Worker,
	forecasterToForecastImpliedInference map[Worker]*emissions.Inference,
) (forecastImpliedInferences []*emissions.WorkerAttributedValue) {
	forecastImpliedInferences = make([]*emissions.WorkerAttributedValue, 0)
	for _, forecaster := range forecasters {
		if forecasterToForecastImpliedInference[forecaster] == nil {
			logger.Warn("getForecastImpliedInferences() no forecast-implied inference", "forecaster", forecaster)
			continue
		}
		forecastImpliedInferences = append(forecastImpliedInferences, &emissions.WorkerAttributedValue{
			Worker: forecasterToForecastImpliedInference[forecaster].Inferer,
			Value:  forecasterToForecastImpliedInference[forecaster].Value,
		})
	}
	return forecastImpliedInferences
}

type GetOneOutInfererForecastImpliedInferencesArgs struct {
	Ctx                  sdk.Context
	K                    emissionskeeper.Keeper
	Logger               log.Logger
	TopicId              uint64
	Inferers             []Inferer
	InfererToInference   map[Inferer]*emissions.Inference
	InfererToRegret      map[Inferer]*Regret
	AllInferersAreNew    bool
	Forecasters          []Forecaster
	ForecasterToForecast map[Forecaster]*emissions.Forecast
	ForecasterToRegret   map[Forecaster]*Regret
	NetworkCombinedLoss  *alloraMath.Dec
	EpsilonTopic         alloraMath.Dec
	PNorm                alloraMath.Dec
	CNorm                alloraMath.Dec
	StdDevPlusEpsilon    alloraMath.Dec
}

// GetOneOutInfererForecastImpliedInferences calculates what each forecaster's implied inference
// would be when each inferer is removed from the calculation
func GetOneOutInfererForecastImpliedInferences(args GetOneOutInfererForecastImpliedInferencesArgs) (
	oneOutInfererForecastImpliedValues []*emissions.OneOutInfererForecasterValues,
	err error,
) {
	oneOutInfererForecastImpliedValues = make([]*emissions.OneOutInfererForecasterValues, 0)

	// If NetworkCombinedLoss is nil, return empty slice immediately
	// if args.NetworkCombinedLoss == nil {
	// 	args.Logger.Debug("NetworkCombinedLoss is nil, returning empty one-out inferer forecast implied values", "topicId", args.TopicId)
	// 	return oneOutInfererForecastImpliedValues, nil
	// }

	// Only process if we have both inferers and forecasters
	if len(args.Inferers) <= 1 || len(args.Forecasters) == 0 {
		return oneOutInfererForecastImpliedValues, nil
	}

	// Organize by forecaster first, then by withheld inferer
	for _, forecaster := range args.Forecasters {
		// For each forecaster, we'll calculate what their forecast-implied inference would be
		// if each inferer was removed one at a time
		oneOutInfererValues := make([]*emissions.WithheldWorkerAttributedValue, 0)

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
			forecastImpliedInferences, _, _, calcErr := CalcForecastImpliedInferences(
				CalcForecastImpliedInferencesArgs{
					Logger:               args.Logger,
					TopicId:              args.TopicId,
					AllInferersAreNew:    args.AllInferersAreNew,
					Inferers:             filteredInferers,
					InfererToInference:   filteredInfererToInference,
					InfererToRegret:      filteredInfererToRegret,
					Forecasters:          filteredForecasters,
					ForecasterToForecast: filteredForecasterToForecast,
					ForecasterToRegret:   filteredForecasterToRegret,
					NetworkCombinedLoss:  args.NetworkCombinedLoss,
					EpsilonTopic:         args.EpsilonTopic,
					PNorm:                args.PNorm,
					CNorm:                args.CNorm,
					StdDevPlusEpsilon:    args.StdDevPlusEpsilon,
				},
			)
			if calcErr != nil {
				args.Logger.Debug("Error calculating forecast implied inference for forecaster", forecaster, "withheldInferer", withheldInferer, "error", calcErr)
				continue
			}

			// Extract the implied inference for this forecaster
			forecastImpliedInference, ok := forecastImpliedInferences[forecaster]
			if !ok {
				continue
			}

			// Add to our results
			oneOutInfererValues = append(oneOutInfererValues, &emissions.WithheldWorkerAttributedValue{
				Worker: withheldInferer,
				Value:  forecastImpliedInference.Value,
			})
		}

		// Only add this forecaster's results if we calculated values for at least one withheld inferer
		if len(oneOutInfererValues) > 0 {
			oneOutInfererForecastImpliedValues = append(
				oneOutInfererForecastImpliedValues,
				&emissions.OneOutInfererForecasterValues{
					Forecaster:          forecaster,
					OneOutInfererValues: oneOutInfererValues,
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
	StdDevPlusEpsilon                    alloraMath.Dec
}

// Calculates the network naive inference I^-_i
func GetNaiveInference(args GetNaiveInferenceArgs) (naiveInference alloraMath.Dec, err error) {
	args.Logger.Debug("Calculating naive inference", "topicId", args.TopicId)

	// Get inferer naive regrets
	infererToRegret := make(map[string]*alloraMath.Dec)
	for _, inferer := range args.Inferers {
		regret, _, err := args.K.GetNaiveInfererNetworkRegret(args.Ctx, args.TopicId, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "GetNaiveInference() error getting naive regret for inferer %s", inferer)
		}
		infererToRegret[inferer] = &regret.Value
	}

	weights, err := CalcWeightsGivenWorkers(
		CalcWeightsGivenWorkersArgs{
			Logger:             args.Logger,
			Inferers:           args.Inferers,
			Forecasters:        args.Forecasters,
			InfererToRegret:    infererToRegret,
			ForecasterToRegret: make(map[Worker]*alloraMath.Dec, 0),
			EpsilonTopic:       args.EpsilonTopic,
			PNorm:              args.PNorm,
			CNorm:              args.CNorm,
			StdDevPlusEpsilon:  args.StdDevPlusEpsilon,
		},
	)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "GetNaiveInference() error calculating weights for naive inference")
	}

	naiveInference, err = calcWeightedInference(calcWeightedInferenceArgs{
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
	})
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrap(err, "GetNaiveInference() error calculating naive inference")
	}

	args.Logger.Debug("Naive inference calculated", "topicId", args.TopicId)
	return naiveInference, nil
}

// Arguments for calcOneOutInfererInference
type CalcOneOutInfererInferenceArgs struct {
	Ctx                  sdk.Context
	K                    emissionskeeper.Keeper
	Logger               log.Logger
	TopicId              uint64
	Inferers             []Inferer
	InfererToInference   map[Inferer]*emissions.Inference
	InfererToRegret      map[Inferer]*Regret
	AllInferersAreNew    bool
	Forecasters          []Forecaster
	ForecasterToForecast map[Forecaster]*emissions.Forecast
	ForecasterToRegret   map[Forecaster]*Regret
	NetworkCombinedLoss  *alloraMath.Dec
	EpsilonTopic         alloraMath.Dec
	EpsilonSafeDiv       alloraMath.Dec
	PNorm                alloraMath.Dec
	CNorm                alloraMath.Dec
	WithheldInferer      Inferer
	StdDevPlusEpsilon    alloraMath.Dec
}

// Calculate the one-out inference given a withheld inferer
func calcOneOutInfererInference(args CalcOneOutInfererInferenceArgs) (
	oneOutNetworkInferenceWithoutInferer alloraMath.Dec,
	err error,
) {
	args.Logger.Debug("calcOneOutInfererInference() calculating one-out inference", "topicId", args.TopicId, "withheldInferer", args.WithheldInferer)

	// To calculate one out, remove the inferer from the list of inferers
	remainingInferers := make([]Worker, 0)
	remainingInfererToInference := make(map[Worker]*emissions.Inference)
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
		}

		//over every inferer
		regret, _, err := args.K.GetOneOutInfererInfererNetworkRegret(args.Ctx, args.TopicId, args.WithheldInferer, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "calcOneOutInfererInference() error getting one-out inferer regret")
		}
		remainingInfererRegrets[inferer] = &regret.Value
	}

	remainingForecasterRegrets := make(map[string]*alloraMath.Dec)
	forecasterToForecastImpliedInference := make(map[string]*emissions.Inference)
	if args.NetworkCombinedLoss != nil {
		// Recalculate the forecast-implied inferences without the worker's inference
		// This is necessary because the forecast-implied inferences are calculated based on the inferences of the inferers
		forecasterToForecastImpliedInference, _, _, err = CalcForecastImpliedInferences(
			CalcForecastImpliedInferencesArgs{
				Logger:               args.Logger,
				TopicId:              args.TopicId,
				AllInferersAreNew:    args.AllInferersAreNew,
				Inferers:             remainingInferers,
				InfererToInference:   args.InfererToInference,
				InfererToRegret:      args.InfererToRegret,
				Forecasters:          args.Forecasters,
				ForecasterToForecast: args.ForecasterToForecast,
				ForecasterToRegret:   args.ForecasterToRegret,
				NetworkCombinedLoss:  args.NetworkCombinedLoss,
				EpsilonTopic:         args.EpsilonTopic,
				PNorm:                args.PNorm,
				CNorm:                args.CNorm,
				StdDevPlusEpsilon:    args.StdDevPlusEpsilon,
			},
		)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "calcOneOutInfererInference() error recalculating forecast-implied inferences")
		}

		// Get regrets for the forecasters
		for _, forecaster := range args.Forecasters {
			regret, _, err := args.K.GetOneOutInfererForecasterNetworkRegret(args.Ctx, args.TopicId, args.WithheldInferer, forecaster)
			if err != nil {
				return alloraMath.Dec{}, errorsmod.Wrapf(err, "calcOneOutInfererInference() error getting one-out forecaster regret")
			}
			remainingForecasterRegrets[forecaster] = &regret.Value
		}
	}
	weights, err := CalcWeightsGivenWorkers(
		CalcWeightsGivenWorkersArgs{
			Logger:             args.Logger,
			Inferers:           remainingInferers,
			Forecasters:        args.Forecasters,
			InfererToRegret:    remainingInfererRegrets,
			ForecasterToRegret: remainingForecasterRegrets,
			EpsilonTopic:       args.EpsilonTopic,
			PNorm:              args.PNorm,
			CNorm:              args.CNorm,
			StdDevPlusEpsilon:  args.StdDevPlusEpsilon,
		},
	)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "calcOneOutInfererInference() error calculating one-out inference for forecaster")
	}

	oneOutNetworkInferenceWithoutInferer, err = calcWeightedInference(calcWeightedInferenceArgs{
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
	})
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "calcOneOutInfererInference() error calculating one-out inference for inferer")
	}

	args.Logger.Debug("One-out inference calculated", "topicId", args.TopicId, "withheldInferer", args.WithheldInferer, "oneOutNetworkInferenceWithoutInferer", oneOutNetworkInferenceWithoutInferer)
	return oneOutNetworkInferenceWithoutInferer, nil
}

// args for GetOneOutInfererInferences
type GetOneOutInfererInferencesArgs struct {
	Ctx                  sdk.Context
	K                    emissionskeeper.Keeper
	Logger               log.Logger
	TopicId              uint64
	Inferers             []Inferer
	InfererToInference   map[Inferer]*emissions.Inference
	InfererToRegret      map[Inferer]*Regret
	AllInferersAreNew    bool
	Forecasters          []Forecaster
	ForecasterToForecast map[Forecaster]*emissions.Forecast
	ForecasterToRegret   map[Forecaster]*Regret
	NetworkCombinedLoss  *alloraMath.Dec
	EpsilonTopic         alloraMath.Dec
	EpsilonSafeDiv       alloraMath.Dec
	PNorm                alloraMath.Dec
	CNorm                alloraMath.Dec
	StdDevPlusEpsilon    alloraMath.Dec
}

// Set all one-out-inferer inferences that are possible given the provided input
// Assumed that there is at most 1 inference per inferer
// Loop over all inferences and withhold one, then calculate the network inference less that withheld inference
// This involves recalculating the forecast-implied inferences for each withheld inferer
func GetOneOutInfererInferences(args GetOneOutInfererInferencesArgs) (
	oneOutInfererInferences []*emissions.WithheldWorkerAttributedValue,
	err error,
) {
	args.Logger.Debug("Calculating one-out inferer inferences", "topicId", args.TopicId, "numInferers", len(args.Inferers))

	// Calculate the one-out inferences per inferer
	oneOutInferences := make([]*emissions.WithheldWorkerAttributedValue, 0)
	for _, worker := range args.Inferers {
		oneOutInference, err := calcOneOutInfererInference(
			CalcOneOutInfererInferenceArgs{
				Ctx:                  args.Ctx,
				K:                    args.K,
				Logger:               args.Logger,
				TopicId:              args.TopicId,
				Inferers:             args.Inferers,
				InfererToInference:   args.InfererToInference,
				InfererToRegret:      args.InfererToRegret,
				AllInferersAreNew:    args.AllInferersAreNew,
				Forecasters:          args.Forecasters,
				ForecasterToForecast: args.ForecasterToForecast,
				ForecasterToRegret:   args.ForecasterToRegret,
				NetworkCombinedLoss:  args.NetworkCombinedLoss,
				EpsilonTopic:         args.EpsilonTopic,
				EpsilonSafeDiv:       args.EpsilonSafeDiv,
				PNorm:                args.PNorm,
				CNorm:                args.CNorm,
				WithheldInferer:      worker,
				StdDevPlusEpsilon:    args.StdDevPlusEpsilon,
			})
		if err != nil {
			return []*emissions.WithheldWorkerAttributedValue{}, errorsmod.Wrapf(err, "GetOneOutInfererInferences() error calculating one-out inferer inferences")
		}

		oneOutInferences = append(oneOutInferences, &emissions.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  oneOutInference,
		})
	}

	args.Logger.Debug("One-out inferer inferences calculated", "topicId", args.TopicId)
	return oneOutInferences, nil
}

// Arguments for calcOneOutForecasterInference
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
	StdDevPlusEpsilon                    alloraMath.Dec
}

// Calculate the one-out inference given a withheld forecaster
func calcOneOutForecasterInference(args CalcOneOutForecasterInferenceArgs) (
	oneOutNetworkInferenceWithoutInferer alloraMath.Dec,
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

			regret, _, err := args.K.GetOneOutForecasterForecasterNetworkRegret(args.Ctx, args.TopicId, args.WithheldForecaster, forecaster)
			if err != nil {
				return alloraMath.Dec{}, errorsmod.Wrapf(err, "calcOneOutForecasterInference() error getting one-out forecaster regret")
			}
			remainingForecasterRegrets[forecaster] = &regret.Value

			forecast, ok := args.ForecasterToForecast[forecaster]
			if !ok {
				return alloraMath.Dec{}, errorsmod.Wrapf(emissions.ErrNotFound, "calcOneOutForecasterInference() cannot find forecaster in ForecasterRegrets %v", forecaster)
			}
			remainingForecasterToForecast[forecaster] = forecast
		}
	}

	// Get regrets for the remaining inferers
	remainingInfererRegrets := make(map[Inferer]*Regret)
	for _, inferer := range args.Inferers {
		regret, _, err := args.K.GetOneOutForecasterInfererNetworkRegret(args.Ctx, args.TopicId, args.WithheldForecaster, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "calcOneOutForecasterInference() error getting one-out inferer regret")
		}
		remainingInfererRegrets[inferer] = &regret.Value
	}

	weights, err := CalcWeightsGivenWorkers(
		CalcWeightsGivenWorkersArgs{
			Logger:             args.Logger,
			Inferers:           args.Inferers,
			Forecasters:        remainingForecasters,
			InfererToRegret:    remainingInfererRegrets,
			ForecasterToRegret: remainingForecasterRegrets,
			EpsilonTopic:       args.EpsilonTopic,
			PNorm:              args.PNorm,
			CNorm:              args.CNorm,
			StdDevPlusEpsilon:  args.StdDevPlusEpsilon,
		},
	)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "calcOneOutForecasterInference() error calculating one-out inference for forecaster")
	}

	oneOutNetworkInferenceWithoutInferer, err = calcWeightedInference(calcWeightedInferenceArgs{
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
	})
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "calcOneOutForecasterInference() error calculating one-out inference for inferer")
	}

	args.Logger.Debug("One-out inference calculated", "topicId", args.TopicId, "withheldForecaster", args.WithheldForecaster, "oneOutNetworkInferenceWithoutInferer", oneOutNetworkInferenceWithoutInferer)
	return oneOutNetworkInferenceWithoutInferer, nil
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
	StdDevPlusEpsilon                    alloraMath.Dec
}

// Set all one-out-forecaster inferences that are possible given the provided input
// Assume that there is at most 1 forecast-implied inference per forecaster
// Loop over all forecast-implied inferences and withhold one, then calculate the network inference less that withheld value
func GetOneOutForecasterInferences(args GetOneOutForecasterInferencesArgs) (
	oneOutForecasterInferences []*emissions.WithheldWorkerAttributedValue,
	err error,
) {
	args.Logger.Debug("Calculating one-out forecaster inferences", "topicId", args.TopicId, "numForecasters", len(args.Forecasters))
	// Calculate the one-out forecast-implied inferences per forecaster
	oneOutForecasterInferences = make([]*emissions.WithheldWorkerAttributedValue, 0)
	// If there is only one forecaster, there's no need to calculate one-out inferences
	if len(args.Forecasters) > 1 {
		for _, worker := range args.Forecasters {
			oneOutInference, err := calcOneOutForecasterInference(
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
					StdDevPlusEpsilon:                    args.StdDevPlusEpsilon,
				})
			if err != nil {
				return []*emissions.WithheldWorkerAttributedValue{}, errorsmod.Wrapf(err, "GetOneOutForecasterInferences() error calculating one-out forecaster inferences")
			}
			oneOutForecasterInferences = append(oneOutForecasterInferences, &emissions.WithheldWorkerAttributedValue{
				Worker: worker,
				Value:  oneOutInference,
			})
		}
		args.Logger.Debug("One-out forecaster inferences calculated", "topicId", args.TopicId)
	}
	return oneOutForecasterInferences, nil
}

// Arguments to calcOneInValue
type calcOneInValueArgs struct {
	Ctx                                  sdk.Context
	K                                    emissionskeeper.Keeper
	Logger                               log.Logger
	TopicId                              uint64
	AllInferersAreNew                    bool
	Inferers                             []Inferer
	InfererToInference                   map[Inferer]*emissions.Inference
	ForecasterToForecast                 map[Forecaster]*emissions.Forecast
	ForecasterToForecastImpliedInference map[Forecaster]*emissions.Inference
	EpsilonTopic                         alloraMath.Dec
	EpsilonSafeDiv                       alloraMath.Dec
	PNorm                                alloraMath.Dec
	CNorm                                alloraMath.Dec
	OneInForecaster                      Forecaster
	StdDevPlusEpsilon                    alloraMath.Dec
}

// Calculate the one-in inference given a withheld forecaster
func calcOneInValue(args calcOneInValueArgs) (
	oneInInference alloraMath.Dec,
	err error,
) {
	args.Logger.Debug("Calculating one-in inference", "forecaster", args.OneInForecaster)

	// In each loop, remove all forecast-implied inferences except one
	singleForecastImpliedInference := make(map[Worker]*emissions.Inference, 1)
	singleForecastImpliedInference[args.OneInForecaster] = args.ForecasterToForecastImpliedInference[args.OneInForecaster]

	// Get self regret for the forecaster
	singleForecasterRegret := make(map[Worker]*Regret, 1)
	regret, _, err := args.K.GetOneInForecasterNetworkRegret(args.Ctx, args.TopicId, args.OneInForecaster, args.OneInForecaster)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "CalcOneInValue() error getting one-in forecaster regret")
	}
	singleForecasterRegret[args.OneInForecaster] = &regret.Value

	// get self forecast list
	singleForecaster := []Worker{args.OneInForecaster}

	// get map of Forecaster to their forecast for the single forecaster
	singleForecasterToForecast := make(map[Forecaster]*emissions.Forecast, 1)
	singleForecasterToForecast[args.OneInForecaster] = args.ForecasterToForecast[args.OneInForecaster]

	// Get one-in regrets for the forecaster and the inferers they provided forecasts for
	infererToRegretForSingleForecaster := make(map[Inferer]*Regret)
	infererToInferenceForSingleForecaster := make(map[Inferer]*emissions.Inference)
	for _, inferer := range args.Inferers {
		regret, _, err := args.K.GetOneInForecasterNetworkRegret(args.Ctx, args.TopicId, args.OneInForecaster, inferer)
		if err != nil {
			return alloraMath.Dec{}, errorsmod.Wrapf(err, "CalcOneInValue() error getting one-in forecaster regret")
		}
		infererToRegretForSingleForecaster[inferer] = &regret.Value

		inference, ok := args.InfererToInference[inferer]
		if !ok {
			args.Logger.Debug("CalcOneInValue() cannot find inferer in InferenceByWorker", "inferer", inferer)
			continue
		}
		infererToInferenceForSingleForecaster[inferer] = inference
	}

	weights, err := CalcWeightsGivenWorkers(
		CalcWeightsGivenWorkersArgs{
			Logger:             args.Logger,
			Inferers:           args.Inferers,
			Forecasters:        singleForecaster,
			InfererToRegret:    infererToRegretForSingleForecaster,
			ForecasterToRegret: singleForecasterRegret,
			EpsilonTopic:       args.EpsilonTopic,
			PNorm:              args.PNorm,
			CNorm:              args.CNorm,
			StdDevPlusEpsilon:  args.StdDevPlusEpsilon,
		},
	)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "CalcOneInValue() error calculating weights for one-in inferences")
	}
	// Calculate the network inference with just this forecaster's forecast-implied inference
	oneInInference, err = calcWeightedInference(calcWeightedInferenceArgs{
		logger:                               args.Logger,
		allInferersAreNew:                    args.AllInferersAreNew,
		inferers:                             args.Inferers,
		workerToInference:                    infererToInferenceForSingleForecaster,
		infererToRegret:                      infererToRegretForSingleForecaster,
		forecasters:                          singleForecaster,
		forecasterToRegret:                   singleForecasterRegret,
		forecasterToForecastImpliedInference: args.ForecasterToForecastImpliedInference,
		weights:                              weights,
		epsilonSafeDiv:                       args.EpsilonSafeDiv,
	})
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "CalcOneInValue() error calculating one-in inference")
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
	ForecasterToForecast                 map[Forecaster]*emissions.Forecast
	ForecasterToForecastImpliedInference map[Forecaster]*emissions.Inference
	EpsilonTopic                         alloraMath.Dec
	EpsilonSafeDiv                       alloraMath.Dec
	PNorm                                alloraMath.Dec
	CNorm                                alloraMath.Dec
	StdDevPlusEpsilon                    alloraMath.Dec
}

// Set all one-in inferences that are possible given the provided input
// Assumed that there is at most 1 inference per worker.
// Also assume that there is at most 1 forecast-implied inference per worker.
func GetOneInForecasterInferences(args GetOneInForecasterInferencesArgs) (
	oneInInferences []*emissions.WorkerAttributedValue,
	err error,
) {
	// Loop over all forecast-implied inferences and set it as the only forecast-implied inference
	// one at a time, then calculate the network inference given that one held out
	oneInInferences = make([]*emissions.WorkerAttributedValue, 0)
	// If there is only one forecaster, thre's no need to calculate one-in inferences
	if len(args.Forecasters) > 1 {
		for _, oneInForecaster := range args.Forecasters {
			oneInValue, err := calcOneInValue(
				calcOneInValueArgs{
					Ctx:                                  args.Ctx,
					K:                                    args.K,
					Logger:                               args.Logger,
					TopicId:                              args.TopicId,
					AllInferersAreNew:                    args.AllInferersAreNew,
					Inferers:                             args.Inferers,
					InfererToInference:                   args.InfererToInference,
					ForecasterToForecast:                 args.ForecasterToForecast,
					ForecasterToForecastImpliedInference: args.ForecasterToForecastImpliedInference,
					EpsilonTopic:                         args.EpsilonTopic,
					EpsilonSafeDiv:                       args.EpsilonSafeDiv,
					PNorm:                                args.PNorm,
					CNorm:                                args.CNorm,
					OneInForecaster:                      oneInForecaster,
					StdDevPlusEpsilon:                    args.StdDevPlusEpsilon,
				})
			if err != nil {
				return []*emissions.WorkerAttributedValue{}, errorsmod.Wrapf(err, "GetOneInForecasterInferences() error calculating one-in inferences")
			}
			oneInInferences = append(oneInInferences, &emissions.WorkerAttributedValue{
				Worker: oneInForecaster,
				Value:  oneInValue,
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
	StdDevPlusEpsilon                    alloraMath.Dec
	InferenceBlockHeight                 BlockHeight
}

// Calculates all network inferences in the set I_i given historical state (e.g. regrets)
// and data from workers (e.g. inferences, forecast-implied inferences).
// Could improve this with Builder pattern, as for other instances of generated ValueBundles.
func CalcNetworkInferences(
	args CalcNetworkInferencesArgs,
) (
	inferenceBundle *emissions.ValueBundle,
	weights RegretInformedWeights,
	err error,
) {
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
			StdDevPlusEpsilon:                    args.StdDevPlusEpsilon,
		})
	if err != nil {
		return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "CalcNetworkInferences() error calculating combined inference")
	}
	// get all the inferences which is all I_ij
	inferences := getInferences(args.Inferers, args.InfererToInference)
	// get all the forecast-implied inferences which is all I_ik
	forecastImpliedInferences := getForecastImpliedInferences(
		args.Logger,
		args.Forecasters,
		args.ForecasterToForecastImpliedInference,
	)

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
			StdDevPlusEpsilon:                    args.StdDevPlusEpsilon,
		})
	if err != nil {
		return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "CalcNetworkInferences() error calculating naive inference")
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
			Ctx:                  args.Ctx,
			K:                    args.K,
			Logger:               args.Logger,
			TopicId:              args.TopicId,
			Inferers:             args.Inferers,
			InfererToInference:   args.InfererToInference,
			InfererToRegret:      args.InfererToRegret,
			AllInferersAreNew:    args.AllInferersAreNew,
			Forecasters:          args.Forecasters,
			ForecasterToForecast: args.ForecasterToForecast,
			ForecasterToRegret:   args.ForecasterToRegret,
			NetworkCombinedLoss:  args.NetworkCombinedLoss,
			EpsilonTopic:         args.EpsilonTopic,
			EpsilonSafeDiv:       args.EpsilonSafeDiv,
			PNorm:                args.PNorm,
			CNorm:                args.CNorm,
			StdDevPlusEpsilon:    args.StdDevPlusEpsilon,
		})
	if err != nil {
		return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "CalcNetworkInferences() error calculating one-out inferer inferences")
	}

	// get the one-out forecaster inferences I^-_li over I_ik
	// The one-out network forecaster inferences represent an approximation of
	// Shapley (1953) values and are used to quantify the individual
	// contributions of forecasters to the network accuracy,
	// which in turn sets the reward distribution between forecasters.
	// The one-out network forecaster inferences are also used to
	// calculate confidence intervals on the network inference I_i
	var oneOutForecasterInferences []*emissions.WithheldWorkerAttributedValue
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
				StdDevPlusEpsilon:                    args.StdDevPlusEpsilon,
			})
		if err != nil {
			return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "CalcNetworkInferences() error calculating one-out forecaster inferences")
		}
	} else {
		oneOutForecasterInferences = []*emissions.WithheldWorkerAttributedValue{}
	}

	// get the one-in forecaster inferences I^+_ki
	// which adds only a single forecast-implied inference I_ik to the inferences
	// from the inference task I_ij . As such, it is used to quantify how the naive
	// network inference I^-_i changes with the addition of a single
	// forecast-implied inference, which in turn is used for setting the
	// reward distribution between workers for their forecasting tasks.
	var oneInForecasterInferences []*emissions.WorkerAttributedValue
	if args.NetworkCombinedLoss != nil {
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
				ForecasterToForecast:                 args.ForecasterToForecast,
				ForecasterToForecastImpliedInference: args.ForecasterToForecastImpliedInference,
				EpsilonTopic:                         args.EpsilonTopic,
				EpsilonSafeDiv:                       args.EpsilonSafeDiv,
				PNorm:                                args.PNorm,
				CNorm:                                args.CNorm,
				StdDevPlusEpsilon:                    args.StdDevPlusEpsilon,
			})
		if err != nil {
			return &emissions.ValueBundle{}, RegretInformedWeights{}, errorsmod.Wrap(err, "CalcNetworkInferences() error calculating one-in inferences")
		}
	} else {
		oneInForecasterInferences = []*emissions.WorkerAttributedValue{}
	}

	// Calculate one-out inferer forecast implied values
	var oneOutInfererForecastImpliedValues []*emissions.OneOutInfererForecasterValues
	if args.NetworkCombinedLoss != nil {
		oneOutInfererForecastImpliedValues, err = GetOneOutInfererForecastImpliedInferences(
			GetOneOutInfererForecastImpliedInferencesArgs{
				Ctx:                  args.Ctx,
				K:                    args.K,
				Logger:               args.Logger,
				TopicId:              args.TopicId,
				Inferers:             args.Inferers,
				InfererToInference:   args.InfererToInference,
				InfererToRegret:      args.InfererToRegret,
				AllInferersAreNew:    args.AllInferersAreNew,
				Forecasters:          args.Forecasters,
				ForecasterToForecast: args.ForecasterToForecast,
				ForecasterToRegret:   args.ForecasterToRegret,
				NetworkCombinedLoss:  args.NetworkCombinedLoss,
				EpsilonTopic:         args.EpsilonTopic,
				PNorm:                args.PNorm,
				CNorm:                args.CNorm,
				StdDevPlusEpsilon:    args.StdDevPlusEpsilon,
			},
		)
		if err != nil {
			return nil, RegretInformedWeights{}, errorsmod.Wrap(err, "while calculating one-out inferer forecast implied inferences")
		}
	} else {
		oneOutInfererForecastImpliedValues = []*emissions.OneOutInfererForecasterValues{}
	}

	// Build value bundle to return all the calculated inferences
	// ATTN: PROTO-2464
	return &emissions.ValueBundle{
		TopicId: args.TopicId,
		ReputerRequestNonce: &emissions.ReputerRequestNonce{
			ReputerNonce: &emissions.Nonce{BlockHeight: args.InferenceBlockHeight},
		},
		Reputer:                       "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
		ExtraData:                     nil,
		CombinedValue:                 combinedInference,
		InfererValues:                 inferences,
		ForecasterValues:              forecastImpliedInferences,
		NaiveValue:                    naiveInference,
		OneOutInfererValues:           oneOutInfererInferences,
		OneOutForecasterValues:        oneOutForecasterInferences,
		OneInForecasterValues:         oneInForecasterInferences,
		OneOutInfererForecasterValues: oneOutInfererForecastImpliedValues,
	}, weights, err
}
