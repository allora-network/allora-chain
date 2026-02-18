package inferencesynthesis

import (
	"errors"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
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
		return nil, errorsmod.Wrap(emissions.ErrNotFound, "no inferences nonce provided")
	}
	if inferences == nil {
		return nil, errorsmod.Wrap(emissions.ErrNotFound, "no inferences found")
	}

	// Enable math helper cache for this function's scope
	enableMathCache()

	// Disable math helper cache and clear cache when function exits
	defer func() {
		disableMathCache()
		clearMathCache()
		ctx.Logger().Debug("Math helper cache cleared after network inference calculation")
	}()

	if len(inferences.Inferences) > 1 {
		// If we have multiple inferences:
		return calcNetworkInferencesMultiple(ctx, k, topicId, inferences, forecasts, *inferencesNonce)
	} else if len(inferences.Inferences) == 1 {
		// If we only have a single inference, simply return it as is.
		return calcNetworkInferencesSingle(*inferencesNonce, topicId, inferences), nil
	} else {
		return nil, errorsmod.Wrap(emissions.ErrNotFound, "no inferences found")
	}
}

func calcNetworkInferencesMultiple(
	ctx sdk.Context,
	k emissionskeeper.Keeper,
	topicId TopicId,
	inferences *emissions.Inferences,
	forecasts *emissions.Forecasts,
	inferenceBlockHeight BlockHeight,
) (*GetNetworkInferencesResult, error) {
	// Set forecasts to nil if there are no forecasts
	if forecasts == nil {
		forecasts = &emissions.Forecasts{
			Forecasts: make([]*emissions.Forecast, 0),
		}
	}

	// Retrieve module params
	moduleParams, err := k.GetParamsKeeper().GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "while getting params")
	}

	// Retrieve topic
	topic, err := k.GetTopicKeeper().GetTopic(ctx, topicId)
	if err != nil {
		return nil, errorsmod.Wrap(err, "while getting topic")
	}

	var previousNetworkCombinedLoss *alloraMath.Dec
	networkLosses, err := k.GetReputerLossKeeper().GetLatestNetworkLossBundle(ctx, topicId)
	if err != nil && !errors.Is(err, emissions.ErrNotFound) {
		return nil, errorsmod.Wrap(err, "while getting latest network loss bundle")
	} else {
		if networkLosses != nil {
			previousNetworkCombinedLoss = &networkLosses.CombinedValue
		}
	}

	// Otherwise, go ahead and calculate the inferences in the more complex way
	calcArgs, err := GetCalcNetworkInferenceArgs(
		ctx,
		k,
		topicId,
		inferences,
		forecasts,
		topic,
		previousNetworkCombinedLoss,
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

	lossBlockHeight := int64(0)
	if networkLosses != nil {
		lossBlockHeight = networkLosses.ReputerRequestNonce.ReputerNonce.BlockHeight
	}

	return &GetNetworkInferencesResult{
		NetworkInferences:    networkInferences,
		InfererToWeight:      weights.Inferers,
		ForecasterToWeight:   weights.Forecasters,
		InferenceBlockHeight: inferenceBlockHeight,
		LossBlockHeight:      lossBlockHeight,
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
	previousLossesCombinedValue *alloraMath.Dec,
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
	infererAddresses := make([]string, 0, len(sortedInferers))
	infererRegrets := make([]alloraMath.Dec, 0, len(sortedInferers))
	for _, inferer := range sortedInferers {
		regret, _, err := k.GetRegretsKeeper().GetInfererNetworkRegret(ctx, topicId, inferer)
		if err != nil {
			return CalcNetworkInferencesArgs{}, errorsmod.Wrapf(err, "GetCalcNetworkInferenceArgs: error getting inferer regret")
		}

		logger.Debug("Inferer has regret", "inferer", inferer, "regret", regret.Value)
		infererToRegret[inferer] = &regret.Value

		// Collect for set event emission
		infererAddresses = append(infererAddresses, inferer)
		infererRegrets = append(infererRegrets, regret.Value)
	}

	// Get the latest regret stdnorm from the keeper. If zero, it will recalculate with provided data.
	stdDevPlusEpsilon, err := k.GetWeightsKeeper().GetLatestRegretStdNorm(ctx, topicId)
	if err != nil {
		return CalcNetworkInferencesArgs{}, errorsmod.Wrap(err, "CalcNetworkInferences() error getting latest regret stdnorm")
	}
	logger.Info("GetCalcNetworkInferenceArgs: StdDevPlusEpsilon", "stdDevPlusEpsilon", stdDevPlusEpsilon)

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
		NetworkCombinedLoss:                  previousLossesCombinedValue,
		EpsilonTopic:                         topic.Epsilon,
		EpsilonSafeDiv:                       moduleParams.EpsilonSafeDiv,
		PNorm:                                topic.PNorm,
		CNorm:                                topic.CNorm,
		StdDevPlusEpsilon:                    stdDevPlusEpsilon,
		InferenceBlockHeight:                 inferenceBlockHeight,
	}

	if previousLossesCombinedValue != nil {
		forecasterToRegret := make(map[string]*alloraMath.Dec)
		forecasterAddresses := make([]string, 0, len(sortedForecasters))
		forecasterRegrets := make([]alloraMath.Dec, 0, len(sortedForecasters))
		for _, forecaster := range sortedForecasters {
			regret, _, err := k.GetRegretsKeeper().GetForecasterNetworkRegret(ctx, topicId, forecaster)
			if err != nil {
				return CalcNetworkInferencesArgs{}, errorsmod.Wrapf(err, "GetCalcNetworkInferenceArgs: error getting forecaster regret")
			}

			logger.Debug("Forecaster has regret", "forecaster", forecaster, "regret", regret.Value)
			forecasterToRegret[forecaster] = &regret.Value

			// Collect for set event emission
			forecasterAddresses = append(forecasterAddresses, forecaster)
			forecasterRegrets = append(forecasterRegrets, regret.Value)
		}

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
				NetworkCombinedLoss:  previousLossesCombinedValue,
				EpsilonTopic:         topic.Epsilon,
				PNorm:                topic.PNorm,
				CNorm:                topic.CNorm,
				StdDevPlusEpsilon:    stdDevPlusEpsilon,
			},
		)
		if err != nil {
			return CalcNetworkInferencesArgs{}, errorsmod.Wrapf(err, "GetCalcNetworkInferenceArgs: error calculating forecast implied inferences")
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

		if len(forecasterAddresses) > 0 {
			emissions.EmitNewNetworkInferenceForecasterRegretsUsedSetEvent(ctx, topicId, inferenceBlockHeight, forecasterAddresses, forecasterRegrets)
		}
	}

	// Emit set events for regrets used in network inference calculation
	if len(infererAddresses) > 0 {
		emissions.EmitNewNetworkInferenceInfererRegretsUsedSetEvent(ctx, topicId, inferenceBlockHeight, infererAddresses, infererRegrets)
	}

	return calcArgs, nil
}
