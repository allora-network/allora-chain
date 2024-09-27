package inferencesynthesis

import (
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func GetNetworkInferences(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	inferencesNonce *BlockHeight,
) (
	networkInferences *emissions.ValueBundle,
	forecasterToForecastImpliedInference map[string]*emissions.Inference,
	infererToWeight map[Inferer]Weight,
	forecasterToWeight map[Forecaster]Weight,
	inferenceBlockHeight int64,
	lossBlockHeight int64,
	err error,
) {

	// Decide whether to use the latest inferences or inferences at a specific block height
	var inferences *emissions.Inferences
	if inferencesNonce == nil {
		inferences, inferenceBlockHeight, err = k.GetLatestTopicInferences(ctx, topicId)
		if err != nil || len(inferences.Inferences) == 0 {
			if err != nil {
				Logger(ctx).Warn(fmt.Sprintf("Error getting inferences: %s", err.Error()))
			}
			return nil, nil, nil, nil, inferenceBlockHeight, lossBlockHeight, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "no inferences found for topic %v", topicId)
		}
	} else {
		inferences, err = k.GetInferencesAtBlock(ctx, topicId, *inferencesNonce)
		inferenceBlockHeight = *inferencesNonce
		if err != nil || len(inferences.Inferences) == 0 {
			return nil, nil, nil, nil, inferenceBlockHeight, lossBlockHeight, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "no inferences found for topic %v at block %v", topicId, *inferencesNonce)
		}
	}

	networkInferences = &emissions.ValueBundle{
		TopicId:          topicId,
		InfererValues:    make([]*emissions.WorkerAttributedValue, 0),
		ForecasterValues: make([]*emissions.WorkerAttributedValue, 0),
	}

	forecasterToForecastImpliedInference = make(map[string]*emissions.Inference, 0)

	// Add inferences to the bundle
	for _, inference := range inferences.Inferences {
		networkInferences.InfererValues = append(networkInferences.InfererValues, &emissions.WorkerAttributedValue{
			Worker: inference.Inferer,
			Value:  inference.Value,
		})
	}

	// Retrieve forecasts
	forecasts, err := k.GetForecastsAtBlock(ctx, topicId, inferenceBlockHeight)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			forecasts = &emissions.Forecasts{
				Forecasts: make([]*emissions.Forecast, 0),
			}
		} else {
			Logger(ctx).Warn(fmt.Sprintf("Error getting forecasts: %s", err.Error()))
			return networkInferences, nil, infererToWeight, forecasterToWeight, inferenceBlockHeight, lossBlockHeight, nil
		}
	}

	// Proceed with network inference calculations if more than one inference exists
	if len(inferences.Inferences) > 1 {
		moduleParams, err := k.GetParams(ctx)
		if err != nil {
			return networkInferences, nil, infererToWeight, forecasterToWeight, inferenceBlockHeight, lossBlockHeight, err
		}

		topic, err := k.GetTopic(ctx, topicId)
		if err != nil {
			Logger(ctx).Warn(fmt.Sprintf("Error getting topic: %s", err.Error()))
			return networkInferences, nil, infererToWeight, forecasterToWeight, inferenceBlockHeight, lossBlockHeight, nil
		}

		// Get latest network loss
		networkLosses, err := k.GetLatestNetworkLossBundle(ctx, topicId)
		if err != nil || networkLosses == nil {
			// Fallback to using the median of the inferences
			inferenceValues := make([]alloraMath.Dec, 0, len(inferences.Inferences))
			for _, inference := range inferences.Inferences {
				inferenceValues = append(inferenceValues, inference.Value)
			}

			medianValue, err := alloraMath.Median(inferenceValues)
			if err != nil {
				Logger(ctx).Warn(fmt.Sprintf("Error calculating median: %s", err.Error()))
				return networkInferences, nil, infererToWeight, forecasterToWeight, inferenceBlockHeight, lossBlockHeight, nil
			}

			networkInferences.CombinedValue = medianValue
			return networkInferences, nil, infererToWeight, forecasterToWeight, inferenceBlockHeight, lossBlockHeight, nil
		} else {
			Logger(ctx).Debug(fmt.Sprintf("Creating network inferences for topic %v with %v inferences and %v forecasts", topicId, len(inferences.Inferences), len(forecasts.Forecasts)))

			logger := Logger(ctx)
			sortedInferers, infererToInference, infererToRegret,
				allInferersAreNew, sortedForecasters, forecasterToForecast,
				forecasterToRegret, forecastImpliedInferencesByWorker, err := GetCalcNetworkInferenceArgs(
				ctx,
				k,
				logger,
				topicId,
				inferences,
				forecasts,
				topic,
				*networkLosses,
				moduleParams,
			)
			if err != nil {
				Logger(ctx).Warn(fmt.Sprintf("Error getting network inference args: %s", err.Error()))
				return networkInferences, nil, infererToWeight, forecasterToWeight, inferenceBlockHeight, lossBlockHeight, nil
			}

			var weights RegretInformedWeights
			networkInferences, weights, err = CalcNetworkInferences(
				ctx,
				k,
				logger,
				topicId,
				sortedInferers,
				infererToInference,
				infererToRegret,
				allInferersAreNew,
				sortedForecasters,
				forecasterToForecast,
				forecasterToRegret,
				forecastImpliedInferencesByWorker,
				networkLosses.CombinedValue,
				topic.Epsilon,
				moduleParams.EpsilonSafeDiv,
				topic.PNorm,
				moduleParams.CNorm,
			)
			if err != nil {
				Logger(ctx).Warn(fmt.Sprintf("Error calculating network inferences: %s", err.Error()))
				return networkInferences, nil, infererToWeight, forecasterToWeight, inferenceBlockHeight, lossBlockHeight, nil
			}
			infererToWeight = weights.inferers
			forecasterToWeight = weights.forecasters
			// Calculate the forecastImpliedInferencesByWorker
			forecasterToForecastImpliedInference, _, _, err = CalcForecastImpliedInferences(
				logger,
				topicId,
				allInferersAreNew,
				sortedInferers,
				infererToInference,
				infererToRegret,
				sortedForecasters,
				forecasterToForecast,
				forecasterToRegret,
				networkLosses.CombinedValue,
				topic.Epsilon,
				topic.PNorm,
				moduleParams.CNorm,
			)
		}
	} else {
		// Single valid inference case
		singleInference := inferences.Inferences[0]

		networkInferences = &emissions.ValueBundle{
			TopicId:       topicId,
			CombinedValue: singleInference.Value,
			InfererValues: []*emissions.WorkerAttributedValue{
				{
					Worker: singleInference.Inferer,
					Value:  singleInference.Value,
				},
			},
			ForecasterValues:       []*emissions.WorkerAttributedValue{},
			NaiveValue:             singleInference.Value,
			OneOutInfererValues:    []*emissions.WithheldWorkerAttributedValue{},
			OneOutForecasterValues: []*emissions.WithheldWorkerAttributedValue{},
			OneInForecasterValues:  []*emissions.WorkerAttributedValue{},
		}
	}

	return networkInferences, forecasterToForecastImpliedInference, infererToWeight, forecasterToWeight, inferenceBlockHeight, lossBlockHeight, nil
}

// helper function for getting the args needed for calcNetworkInferences
// we have to convert the inferences and forecasts to maps and sort the inferers and forecasters
// so that GetNetworkInference can use them
func GetCalcNetworkInferenceArgs(
	ctx sdk.Context,
	k emissionskeeper.Keeper,
	logger log.Logger,
	topicId uint64,
	inferences *emissions.Inferences,
	forecasts *emissions.Forecasts,
	topic emissions.Topic,
	networkLosses emissions.ValueBundle,
	moduleParams emissions.Params,
) (
	sortedInferers []Inferer,
	infererToInference map[string]*emissions.Inference,
	infererToRegret map[string]*alloraMath.Dec,
	allInferersAreNew bool,
	sortedForecasters []Forecaster,
	forecasterToForecast map[string]*emissions.Forecast,
	forecasterToRegret map[string]*alloraMath.Dec,
	forecastImpliedInferencesByWorker map[string]*emissions.Inference,
	err error,
) {
	infererToInference = MakeMapFromInfererToTheirInference(inferences.Inferences)
	forecasterToForecast = MakeMapFromForecasterToTheirForecast(forecasts.Forecasts)
	sortedInferers = alloraMath.GetSortedKeys(infererToInference)
	sortedForecasters = alloraMath.GetSortedKeys(forecasterToForecast)
	allInferersAreNew = topic.InitialRegret.Equal(alloraMath.ZeroDec()) // If initial regret is 0, all inferers are new

	infererToRegret = make(map[string]*alloraMath.Dec)
	for _, inferer := range sortedInferers {
		regret, _, err := k.GetInfererNetworkRegret(ctx, topicId, inferer)
		if err != nil {
			return nil, nil, nil, false, nil, nil, nil, nil, errorsmod.Wrapf(err, "Error getting inferer regret")
		}

		logger.Debug(fmt.Sprintf("Inferer %v has regret %v", inferer, regret.Value))
		infererToRegret[inferer] = &regret.Value
	}

	forecasterToRegret = make(map[string]*alloraMath.Dec)
	for _, forecaster := range sortedForecasters {
		regret, _, err := k.GetForecasterNetworkRegret(ctx, topicId, forecaster)
		if err != nil {
			return nil, nil, nil, false, nil, nil, nil, nil, errorsmod.Wrapf(err, "Error getting forecaster regret")
		}

		logger.Debug(fmt.Sprintf("Forecaster %v has regret %v", forecaster, regret.Value))
		forecasterToRegret[forecaster] = &regret.Value
	}

	forecastImpliedInferencesByWorker, _, _, err = CalcForecastImpliedInferences(
		logger,
		topicId,
		allInferersAreNew,
		sortedInferers,
		infererToInference,
		infererToRegret,
		sortedForecasters,
		forecasterToForecast,
		forecasterToRegret,
		networkLosses.CombinedValue,
		topic.Epsilon,
		topic.PNorm,
		moduleParams.CNorm,
	)
	return sortedInferers, infererToInference, infererToRegret,
		allInferersAreNew, sortedForecasters, forecasterToForecast,
		forecasterToRegret, forecastImpliedInferencesByWorker, err
}
