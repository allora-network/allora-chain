package inferencesynthesis

import (
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
) (*GetNetworkInferencesResult, error) {
	// Retrieve the requested inferences (either latest or specified, depending on inferencesNonce)
	inferences, inferenceBlockHeight, err := getRequestedInferences(ctx, k, topicId, inferencesNonce)
	if err != nil {
		return nil, errorsmod.Wrap(err, "while getting inferences")
	}

	if len(inferences.Inferences) > 1 {
		// If we have multiple inferences:
		// 1. Try to get latest network loss
		networkLosses, err := k.GetLatestNetworkLossBundle(ctx, topicId) // TODO(spook): why latest?  why not use `inferenceBlockHeight`?
		if errors.Is(err, emissions.ErrNotFound) {
			// 2a. If we have no network losses, fallback to using the median of the inferences.
			return calcNetworkInferencesMultipleByMedian(ctx, topicId, inferences, inferenceBlockHeight)
		} else if err != nil {
			return nil, errorsmod.Wrap(err, "while getting latest network loss bundle")
		}

		// 2b. Otherwise, calculate the normal way.
		return calcNetworkInferencesMultiple(ctx, k, topicId, inferences, inferenceBlockHeight, networkLosses)
	} else if len(inferences.Inferences) == 1 {
		// If we only have a single inference, simply return it as is.
		return calcNetworkInferencesSingle(ctx, inferenceBlockHeight, topicId, inferences)
	} else {
		return nil, errors.Wrap(emissions.ErrNotFound, "no inferences found")
	}
}

// Decide whether to use the latest inferences or inferences at a specific block height
func getRequestedInferences(
	ctx sdk.Context,
	k emissionskeeper.Keeper,
	topicId TopicId,
	inferencesNonce *BlockHeight,
) (*emissions.Inferences, int64, error) {
	if inferencesNonce == nil {
		inferences, inferenceBlockHeight, err := k.GetLatestTopicInferences(ctx, topicId)
		if err != nil {
			return nil, 0, err
		} else if len(inferences.Inferences) == 0 {
			return nil, 0, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "no inferences found for topic %v at latest block", topicId)
		}
		return inferences, inferenceBlockHeight, nil
	} else {
		inferences, err := k.GetInferencesAtBlock(ctx, topicId, *inferencesNonce)
		if err != nil {
			return nil, 0, err
		} else if len(inferences.Inferences) == 0 {
			return nil, 0, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "no inferences found for topic %v at block %v", topicId, *inferencesNonce)
		}
		return inferences, *inferencesNonce, nil
	}
}

func calcNetworkInferencesMultipleByMedian(
	ctx sdk.Context,
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
			ReputerNonce: &emissions.Nonce{BlockHeight: ctx.BlockHeight()},
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
	inferenceBlockHeight BlockHeight,
	networkLosses *emissions.ValueBundle,
) (*GetNetworkInferencesResult, error) {
	// TODO(spook): should the following fetches happen at the very top of the
	// call stack to ensure that the topic is real and the module params are present?
	// It would waste some i/o for the other cases (single, multiple+median), but
	// would be more correct and could surface issues more readily.

	// Retrieve forecasts
	forecasts, err := k.GetForecastsAtBlock(ctx, topicId, inferenceBlockHeight)
	if errors.Is(err, collections.ErrNotFound) {
		forecasts = &emissions.Forecasts{Forecasts: nil}
	} else if err != nil {
		return nil, errorsmod.Wrap(err, "while getting forecasts")
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
	Logger(ctx).Debug("Creating network inferences",
		"topic_id", topicId,
		"num_inferences", len(inferences.Inferences),
		"num_forecasts", len(forecasts.Forecasts),
	)

	calcArgs, err := GetCalcNetworkInferenceArgs(
		ctx,
		k,
		topicId,
		inferences,
		forecasts,
		topic,
		*networkLosses,
		moduleParams,
	)
	if err != nil {
		return nil, errorsmod.Wrap(err, "while getting network inference args")
	}

	networkInferences, weights, err := CalcNetworkInferences(calcArgs)
	if err != nil {
		return nil, errorsmod.Wrap(err, "while calculating network inferences")
	}

	// TODO(spook): GetCalcNetworkInferenceArgs already calls CalcForecastImpliedInferences.
	// Why were we calling it again here?

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
	ctx sdk.Context,
	inferenceBlockHeight BlockHeight,
	topicId TopicId,
	inferences *emissions.Inferences,
) (*GetNetworkInferencesResult, error) {
	singleInference := inferences.Inferences[0]

	networkInferences := &emissions.ValueBundle{
		TopicId: topicId,
		Reputer: "allo1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqas6usy",
		ReputerRequestNonce: &emissions.ReputerRequestNonce{
			ReputerNonce: &emissions.Nonce{
				BlockHeight: ctx.BlockHeight(),
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
	}, nil
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

		logger.Debug(fmt.Sprintf("Inferer %v has regret %v", inferer, regret.Value))
		infererToRegret[inferer] = &regret.Value
	}

	forecasterToRegret := make(map[string]*alloraMath.Dec)
	for _, forecaster := range sortedForecasters {
		regret, _, err := k.GetForecasterNetworkRegret(ctx, topicId, forecaster)
		if err != nil {
			return CalcNetworkInferencesArgs{}, errorsmod.Wrapf(err, "GetCalcNetworkInferenceArgs: error getting forecaster regret")
		}

		logger.Debug(fmt.Sprintf("Forecaster %v has regret %v", forecaster, regret.Value))
		forecasterToRegret[forecaster] = &regret.Value
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
			NetworkCombinedLoss:  networkLosses.CombinedValue,
			EpsilonTopic:         topic.Epsilon,
			PNorm:                topic.PNorm,
			CNorm:                moduleParams.CNorm,
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
