package inferencesynthesis

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/utils/fn"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

type GetNetworkInferencesResult struct {
	NetworkInferences    *emissions.ValueBundle
	InfererToWeight      map[Inferer]Weight
	ForecasterToWeight   map[Forecaster]Weight
	InferenceBlockHeight int64
	LossBlockHeight      int64
}

func GetNetworkInferences(
	ctx sdk.Context,
	k emissionskeeper.Keeper,
	topicId TopicId,
	inferencesNonce *BlockHeight,
	inferences *emissions.Inferences,
	forecasts *emissions.Forecasts,
	outlierResistant bool,
) (*GetNetworkInferencesResult, error) {
	if inferencesNonce == nil {
		return nil, errors.Wrap(emissions.ErrNotFound, "no inferences nonce provided")
	}
	if inferences == nil {
		return nil, errors.Wrap(emissions.ErrNotFound, "no inferences found")
	}

	// Enable gradient cache for this function's scope
	enableGradientCache()

	// Disable gradient cache and clear cache when function exits
	defer func() {
		disableGradientCache()
		clearGradientCache()
		ctx.Logger().Debug("Gradient cache cleared after network inference calculation")
	}()

	if len(inferences.Inferences) > 1 {
		// If we have multiple inferences:
		// 1. Try to get latest network loss
		networkLosses, err := k.GetLatestNetworkLossBundle(ctx, topicId)
		if errors.Is(err, emissions.ErrNotFound) {
			// 2a. If we have no network losses, fallback to using the median of the inferences.
			return calcNetworkInferencesMultipleByMedian(topicId, inferences, *inferencesNonce)
		} else if err != nil {
			return nil, errorsmod.Wrap(err, "while getting latest network loss bundle")
		}

		// 2b. Otherwise, calculate the normal way.
		return calcNetworkInferencesMultiple(ctx, k, topicId, inferences, forecasts, *inferencesNonce, networkLosses)
	} else if len(inferences.Inferences) == 1 {
		// If we only have a single inference, simply return it as is.
		return calcNetworkInferencesSingle(*inferencesNonce, topicId, inferences), nil
	} else {
		return nil, errors.Wrap(emissions.ErrNotFound, "no inferences found")
	}
}

func calcNetworkInferencesMultipleByMedian(
	topicId TopicId,
	inferences *emissions.Inferences,
	inferenceBlockHeight BlockHeight,
) (*GetNetworkInferencesResult, error) {
	inferenceValues := fn.Map(inferences.Inferences, func(inf *emissions.Inference) alloraMath.Dec { return inf.Value })

	medianValue, err := alloraMath.Median(inferenceValues)
	if err != nil {
		return nil, errorsmod.Wrap(err, "while calculating median")
	}

	networkInferences := &emissions.ValueBundle{
		TopicId:   topicId,
		ExtraData: nil,
		ReputerRequestNonce: &emissions.ReputerRequestNonce{
			ReputerNonce: &emissions.Nonce{BlockHeight: inferenceBlockHeight},
		},
		Reputer:       "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
		CombinedValue: medianValue,
		InfererValues: fn.Map(inferences.Inferences, func(inf *emissions.Inference) *emissions.WorkerAttributedValue {
			return &emissions.WorkerAttributedValue{Worker: inf.Inferer, Value: inf.Value}
		}),
		ForecasterValues:              nil,
		NaiveValue:                    alloraMath.ZeroDec(),
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	return &GetNetworkInferencesResult{
		NetworkInferences:    networkInferences,
		InfererToWeight:      nil,
		ForecasterToWeight:   nil,
		InferenceBlockHeight: inferenceBlockHeight,
		LossBlockHeight:      0,
	}, nil
}

func calcNetworkInferencesMultiple(
	ctx sdk.Context,
	k emissionskeeper.Keeper,
	topicId TopicId,
	inferences *emissions.Inferences,
	forecasts *emissions.Forecasts,
	inferenceBlockHeight BlockHeight,
	networkLosses *emissions.ValueBundle,
) (*GetNetworkInferencesResult, error) {
	// Set forecasts to nil if there are no forecasts
	if forecasts == nil {
		forecasts = &emissions.Forecasts{
			Forecasts: make([]*emissions.Forecast, 0),
		}
	}

	// Retrieve module params
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "while getting params")
	}

	// Retrieve topic
	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return nil, errorsmod.Wrap(err, "while getting topic")
	}

	// Otherwise, go ahead and calculate the inferences in the more complex way
	calcArgs, err := GetCalcNetworkInferenceArgs(
		ctx,
		k,
		topicId,
		inferences,
		forecasts,
		topic,
		*networkLosses,
		moduleParams,
		inferenceBlockHeight,
	)
	if err != nil {
		return nil, errorsmod.Wrap(err, "while getting network inference args")
	}

	networkInferences, weights, err := CalcNetworkInferences(calcArgs)
	if err != nil {
		return nil, errorsmod.Wrap(err, "while calculating network inferences")
	}

	return &GetNetworkInferencesResult{
		NetworkInferences:    networkInferences,
		InfererToWeight:      weights.Inferers,
		ForecasterToWeight:   weights.Forecasters,
		InferenceBlockHeight: inferenceBlockHeight,
		LossBlockHeight:      networkLosses.ReputerRequestNonce.ReputerNonce.BlockHeight,
	}, nil
}

// Single valid inference case
func calcNetworkInferencesSingle(
	inferenceBlockHeight BlockHeight,
	topicId TopicId,
	inferences *emissions.Inferences,
) *GetNetworkInferencesResult {
	singleInference := inferences.Inferences[0]

	networkInferences := &emissions.ValueBundle{
		TopicId: topicId,
		Reputer: "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
		ReputerRequestNonce: &emissions.ReputerRequestNonce{
			ReputerNonce: &emissions.Nonce{
				BlockHeight: inferenceBlockHeight,
			},
		},
		ExtraData:     nil,
		CombinedValue: singleInference.Value,
		InfererValues: []*emissions.WorkerAttributedValue{
			{
				Worker: singleInference.Inferer,
				Value:  singleInference.Value,
			},
		},
		ForecasterValues:              nil,
		NaiveValue:                    singleInference.Value,
		OneOutInfererValues:           nil,
		OneOutForecasterValues:        nil,
		OneInForecasterValues:         nil,
		OneOutInfererForecasterValues: nil,
	}
	return &GetNetworkInferencesResult{
		NetworkInferences:    networkInferences,
		InfererToWeight:      nil,
		ForecasterToWeight:   nil,
		InferenceBlockHeight: inferenceBlockHeight,
		LossBlockHeight:      0, // Loss data may actually be available but is not needed to calculate network inference in this case
	}
}

// helper function for getting the args needed for calcNetworkInferences
// we have to convert the inferences and forecasts to maps and sort the inferers and forecasters
// so that GetNetworkInference can use them
func GetCalcNetworkInferenceArgs(
	ctx sdk.Context,
	k emissionskeeper.Keeper,
	topicId uint64,
	inferences *emissions.Inferences,
	forecasts *emissions.Forecasts,
	topic emissions.Topic,
	networkLosses emissions.ValueBundle,
	moduleParams emissions.Params,
	inferenceBlockHeight BlockHeight,
) (
	calcArgs CalcNetworkInferencesArgs,
	err error,
) {
	infererToInference := MakeMapFromInfererToTheirInference(inferences.Inferences)
	forecasterToForecast := MakeMapFromForecasterToTheirForecast(forecasts.Forecasts)
	sortedInferers := alloraMath.GetSortedKeys(infererToInference)
	sortedForecasters := alloraMath.GetSortedKeys(forecasterToForecast)
	allInferersAreNew := topic.InitialRegret.Equal(alloraMath.ZeroDec()) // If initial regret is 0, all inferers are new
	logger := Logger(ctx)

	infererToRegret := make(map[string]*alloraMath.Dec)
	for _, inferer := range sortedInferers {
		regret, _, err := k.GetInfererNetworkRegret(ctx, topicId, inferer)
		if err != nil {
			return CalcNetworkInferencesArgs{}, errorsmod.Wrapf(err, "GetCalcNetworkInferenceArgs: error getting inferer regret")
		}

		logger.Debug("Inferer has regret", "inferer", inferer, "regret", regret.Value)
		infererToRegret[inferer] = &regret.Value

		// Emit event for inferer regret used in network inference calculation
		emissions.EmitNetworkInferenceInfererRegretUsedEvent(ctx, topicId, inferenceBlockHeight, inferer, regret.Value)
	}

	forecasterToRegret := make(map[string]*alloraMath.Dec)
	for _, forecaster := range sortedForecasters {
		regret, _, err := k.GetForecasterNetworkRegret(ctx, topicId, forecaster)
		if err != nil {
			return CalcNetworkInferencesArgs{}, errorsmod.Wrapf(err, "GetCalcNetworkInferenceArgs: error getting forecaster regret")
		}

		logger.Debug("Forecaster has regret", "forecaster", forecaster, "regret", regret.Value)
		forecasterToRegret[forecaster] = &regret.Value

		// Emit event for forecaster regret used in network inference calculation
		emissions.EmitNetworkInferenceForecasterRegretUsedEvent(ctx, topicId, inferenceBlockHeight, forecaster, regret.Value)
	}

	// Get the latest regret stdnorm from the keeper. If zero, it will recalculate with provided data.
	stdDevPlusEpsilon, err := k.GetLatestRegretStdNorm(ctx, topicId)
	if err != nil {
		return CalcNetworkInferencesArgs{}, errorsmod.Wrap(err, "CalcNetworkInferences() error getting latest regret stdnorm")
	}
	logger.Info("GetCalcNetworkInferenceArgs: StdDevPlusEpsilon", "stdDevPlusEpsilon", stdDevPlusEpsilon)

	forecastImpliedInferencesByWorker, _, _, err := CalcForecastImpliedInferences(
		CalcForecastImpliedInferencesArgs{
			Logger:               logger,
			TopicId:              topicId,
			AllInferersAreNew:    allInferersAreNew,
			Inferers:             sortedInferers,
			InfererToInference:   infererToInference,
			InfererToRegret:      infererToRegret,
			Forecasters:          sortedForecasters,
			ForecasterToForecast: forecasterToForecast,
			ForecasterToRegret:   forecasterToRegret,
			NetworkCombinedLoss:  networkLosses.CombinedValue,
			EpsilonTopic:         topic.Epsilon,
			PNorm:                topic.PNorm,
			CNorm:                moduleParams.CNorm,
			StdDevPlusEpsilon:    stdDevPlusEpsilon,
		},
	)
	if err != nil {
		return CalcNetworkInferencesArgs{}, errorsmod.Wrapf(err, "GetCalcNetworkInferenceArgs: error calculating forecast implied inferences")
	}

	calcArgs = CalcNetworkInferencesArgs{
		Ctx:                                  ctx,
		K:                                    k,
		Logger:                               logger,
		TopicId:                              topicId,
		Inferers:                             sortedInferers,
		InfererToInference:                   infererToInference,
		InfererToRegret:                      infererToRegret,
		AllInferersAreNew:                    allInferersAreNew,
		Forecasters:                          make([]Forecaster, 0),
		ForecasterToForecast:                 make(map[Forecaster]*emissions.Forecast, 0),
		ForecasterToRegret:                   make(map[Forecaster]*alloraMath.Dec, 0),
		ForecasterToForecastImpliedInference: make(map[Forecaster]*emissions.Inference, 0),
		NetworkCombinedLoss:                  networkLosses.CombinedValue,
		EpsilonTopic:                         topic.Epsilon,
		EpsilonSafeDiv:                       moduleParams.EpsilonSafeDiv,
		PNorm:                                topic.PNorm,
		CNorm:                                moduleParams.CNorm,
		StdDevPlusEpsilon:                    stdDevPlusEpsilon,
		InferenceBlockHeight:                 inferenceBlockHeight,
	}

	// If there are forecast-implied inferences, add forecasters info
	// It will not have available forecast-implied inferences if the forecasters
	// didn't make any forecasts for the existing inferers
	if len(forecastImpliedInferencesByWorker) > 0 {
		for _, forecaster := range sortedForecasters {
			if forecastImpliedInference, ok := forecastImpliedInferencesByWorker[forecaster]; ok {
				calcArgs.Forecasters = append(calcArgs.Forecasters, forecaster)
				calcArgs.ForecasterToForecast[forecaster] = forecasterToForecast[forecaster]
				calcArgs.ForecasterToRegret[forecaster] = forecasterToRegret[forecaster]
				calcArgs.ForecasterToForecastImpliedInference[forecaster] = forecastImpliedInference
			}
		}
	}

	return calcArgs, nil
}
