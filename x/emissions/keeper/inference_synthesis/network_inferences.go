package inference_synthesis

import (
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Calculates all network inferences in the set I_i given historical state (e.g. regrets)
// and data from workers (e.g. inferences, forecast-implied inferences)
// as of a specified block height
func GetNetworkInferencesAtBlock(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	inferencesNonce BlockHeight,
	previousLossNonce BlockHeight,
) (
	*emissions.ValueBundle,
	map[string]*emissions.Inference,
	map[string]alloraMath.Dec,
	map[string]alloraMath.Dec,
	error,
) {
	Logger(ctx).Debug(fmt.Sprintf("Calculating network inferences for topic %v at inference nonce %v with previous loss nonce %v", topicId, inferencesNonce, previousLossNonce))

	networkInferences := &emissions.ValueBundle{
		TopicId:          topicId,
		InfererValues:    make([]*emissions.WorkerAttributedValue, 0),
		ForecasterValues: make([]*emissions.WorkerAttributedValue, 0),
	}

	forecastImpliedInferencesByWorker := make(map[string]*emissions.Inference, 0)
	var infererWeights map[string]alloraMath.Dec
	var forecasterWeights map[string]alloraMath.Dec

	inferences, err := k.GetInferencesAtBlock(ctx, topicId, inferencesNonce)
	if err != nil {
		return nil, nil, infererWeights, forecasterWeights, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "no inferences found for topic %v at block %v", topicId, inferencesNonce)
	}
	// Add inferences in the bundle -> this bundle will be used as a fallback in case of error
	for _, infererence := range inferences.Inferences {
		networkInferences.InfererValues = append(networkInferences.InfererValues, &emissions.WorkerAttributedValue{
			Worker: infererence.Inferer,
			Value:  infererence.Value,
		})
	}

	forecasts, err := k.GetForecastsAtBlock(ctx, topicId, inferencesNonce)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			forecasts = &emissions.Forecasts{
				Forecasts: make([]*emissions.Forecast, 0),
			}
		} else {
			return nil, nil, infererWeights, forecasterWeights, err
		}
	}

	if len(inferences.Inferences) > 1 {
		moduleParams, err := k.GetParams(ctx)
		if err != nil {
			return nil, nil, infererWeights, forecasterWeights, err
		}

		networkLosses, err := k.GetNetworkLossBundleAtBlock(ctx, topicId, previousLossNonce)
		if err != nil {
			Logger(ctx).Warn(fmt.Sprintf("Error getting network losses: %s", err.Error()))
			return networkInferences, nil, infererWeights, forecasterWeights, nil
		}

		topic, err := k.GetTopic(ctx, topicId)
		if err != nil {
			Logger(ctx).Warn(fmt.Sprintf("Error getting topic: %s", err.Error()))
			return networkInferences, nil, infererWeights, forecasterWeights, nil
		}

		Logger(ctx).Debug(fmt.Sprintf("Creating network inferences for topic %v with %v inferences and %v forecasts", topicId, len(inferences.Inferences), len(forecasts.Forecasts)))
		networkInferenceBuilder, err := NewNetworkInferenceBuilderFromSynthRequest(
			SynthRequest{
				Ctx:                 ctx,
				K:                   k,
				TopicId:             topicId,
				Inferences:          inferences,
				Forecasts:           forecasts,
				NetworkCombinedLoss: networkLosses.CombinedValue,
				Epsilon:             moduleParams.Epsilon,
				PNorm:               topic.PNorm,
				CNorm:               moduleParams.CNorm,
			},
		)
		if err != nil {
			Logger(ctx).Warn(fmt.Sprintf("Error constructing network inferences builder topic: %s", err.Error()))
			return nil, nil, infererWeights, forecasterWeights, err
		}
		networkInferences = networkInferenceBuilder.CalcAndSetNetworkInferences().Build()

		forecastImpliedInferencesByWorker = networkInferenceBuilder.palette.ForecastImpliedInferenceByWorker
		infererWeights = networkInferenceBuilder.weights.inferers
		forecasterWeights = networkInferenceBuilder.weights.forecasters
	} else {
		// If there is only one valid inference, then the network inference is the same as the single inference
		// For the forecasts to be meaningful, there should be at least 2 inferences
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

	return networkInferences, forecastImpliedInferencesByWorker, infererWeights, forecasterWeights, nil
}

func GetLatestNetworkInference(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
) (
	*emissions.ValueBundle,
	map[string]*emissions.Inference,
	map[string]alloraMath.Dec,
	map[string]alloraMath.Dec,
	int64,
	int64,
	error,
) {
	inferenceBlockHeight := int64(0)
	lossBlockHeight := int64(0)

	networkInferences := &emissions.ValueBundle{
		TopicId:          topicId,
		InfererValues:    make([]*emissions.WorkerAttributedValue, 0),
		ForecasterValues: make([]*emissions.WorkerAttributedValue, 0),
	}
	forecastImpliedInferencesByWorker := make(map[string]*emissions.Inference, 0)
	var infererWeights map[string]alloraMath.Dec
	var forecasterWeights map[string]alloraMath.Dec

	inferences, inferenceBlockHeight, err := k.GetLatestTopicInferences(ctx, topicId)
	if err != nil || len(inferences.Inferences) == 0 {
		if err != nil {
			Logger(ctx).Warn(fmt.Sprintf("Error getting inferences: %s", err.Error()))
		}
		return networkInferences, nil, infererWeights, forecasterWeights, inferenceBlockHeight, lossBlockHeight, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "no inferences found for topic %v", topicId)
	}
	for _, infererence := range inferences.Inferences {
		networkInferences.InfererValues = append(networkInferences.InfererValues, &emissions.WorkerAttributedValue{
			Worker: infererence.Inferer,
			Value:  infererence.Value,
		})
	}

	forecasts, err := k.GetForecastsAtBlock(ctx, topicId, inferenceBlockHeight)
	if err != nil {
		Logger(ctx).Warn(fmt.Sprintf("Error getting forecasts: %s", err.Error()))
		if errors.Is(err, collections.ErrNotFound) {
			forecasts = &emissions.Forecasts{
				Forecasts: make([]*emissions.Forecast, 0),
			}
		} else {
			return networkInferences, nil, infererWeights, forecasterWeights, inferenceBlockHeight, lossBlockHeight, nil
		}
	}

	if len(inferences.Inferences) > 1 {
		moduleParams, err := k.GetParams(ctx)
		if err != nil {
			return networkInferences, nil, infererWeights, forecasterWeights, inferenceBlockHeight, lossBlockHeight, err
		}
		topic, err := k.GetTopic(ctx, topicId)
		if err != nil {
			Logger(ctx).Warn(fmt.Sprintf("Error getting topic: %s", err.Error()))
			return networkInferences, nil, infererWeights, forecasterWeights, inferenceBlockHeight, lossBlockHeight, nil
		}
		lossBlockHeight = inferenceBlockHeight - topic.EpochLength
		if lossBlockHeight < 0 {
			Logger(ctx).Warn("Network inference is not available for the epoch yet")
			return networkInferences, nil, infererWeights, forecasterWeights, inferenceBlockHeight, lossBlockHeight, nil
		}
		networkLosses, err := k.GetNetworkLossBundleAtBlock(ctx, topicId, lossBlockHeight)

		if err != nil {
			Logger(ctx).Warn(fmt.Sprintf("Error getting network losses: %s", err.Error()))
			return networkInferences, nil, infererWeights, forecasterWeights, inferenceBlockHeight, lossBlockHeight, nil
		}
		networkInferenceBuilder, err := NewNetworkInferenceBuilderFromSynthRequest(
			SynthRequest{
				Ctx:                 ctx,
				K:                   k,
				TopicId:             topicId,
				Inferences:          inferences,
				Forecasts:           forecasts,
				NetworkCombinedLoss: networkLosses.CombinedValue,
				Epsilon:             moduleParams.Epsilon,
				PNorm:               topic.PNorm,
				CNorm:               moduleParams.CNorm,
			},
		)
		if err != nil {
			Logger(ctx).Warn(fmt.Sprintf("Error constructing network inferences builder topic: %s", err.Error()))
			return networkInferences, nil, infererWeights, forecasterWeights, inferenceBlockHeight, lossBlockHeight, err
		}
		networkInferences = networkInferenceBuilder.CalcAndSetNetworkInferences().Build()
		forecastImpliedInferencesByWorker = networkInferenceBuilder.palette.ForecastImpliedInferenceByWorker
		infererWeights = networkInferenceBuilder.weights.inferers
		forecasterWeights = networkInferenceBuilder.weights.forecasters
	} else {
		// If there is only one valid inference, then the network inference is the same as the single inference
		// For the forecasts to be meaningful, there should be at least 2 inferences
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

	return networkInferences, forecastImpliedInferencesByWorker, infererWeights, forecasterWeights, inferenceBlockHeight, lossBlockHeight, nil
}
