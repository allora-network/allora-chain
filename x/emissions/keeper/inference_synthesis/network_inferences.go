package inferencesynthesis

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

func GetNetworkInferences(
	ctx sdk.Context,
	k keeper.Keeper,
	topicId TopicId,
	inferencesNonce *BlockHeight,
) (
	*emissions.ValueBundle,
	map[string]*emissions.Inference,
	map[string]alloraMath.Dec,
	map[string]alloraMath.Dec,
	int64,
	int64,
	error,
) {
	var (
		inferenceBlockHeight int64
		lossBlockHeight      int64
		err                  error
	)

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
		inferenceBlockHeight = int64(*inferencesNonce)
		if err != nil || len(inferences.Inferences) == 0 {
			return nil, nil, nil, nil, inferenceBlockHeight, lossBlockHeight, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "no inferences found for topic %v at block %v", topicId, *inferencesNonce)
		}
	}

	networkInferences := &emissions.ValueBundle{
		TopicId:          topicId,
		InfererValues:    make([]*emissions.WorkerAttributedValue, 0),
		ForecasterValues: make([]*emissions.WorkerAttributedValue, 0),
	}

	forecastImpliedInferencesByWorker := make(map[string]*emissions.Inference, 0)
	var infererWeights map[string]alloraMath.Dec
	var forecasterWeights map[string]alloraMath.Dec

	// Add inferences to the bundle
	for _, inference := range inferences.Inferences {
		networkInferences.InfererValues = append(networkInferences.InfererValues, &emissions.WorkerAttributedValue{
			Worker: inference.Inferer,
			Value:  inference.Value,
		})
	}

	// Retrieve forecasts
	forecasts, err := k.GetForecastsAtBlock(ctx, topicId, BlockHeight(inferenceBlockHeight))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			forecasts = &emissions.Forecasts{
				Forecasts: make([]*emissions.Forecast, 0),
			}
		} else {
			Logger(ctx).Warn(fmt.Sprintf("Error getting forecasts: %s", err.Error()))
			return networkInferences, nil, infererWeights, forecasterWeights, inferenceBlockHeight, lossBlockHeight, nil
		}
	}

	// Proceed with network inference calculations if more than one inference exists
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
				return networkInferences, nil, infererWeights, forecasterWeights, inferenceBlockHeight, lossBlockHeight, nil
			}

			networkInferences.CombinedValue = medianValue
			return networkInferences, nil, infererWeights, forecasterWeights, inferenceBlockHeight, lossBlockHeight, nil
		} else {
			Logger(ctx).Debug(fmt.Sprintf("Creating network inferences for topic %v with %v inferences and %v forecasts", topicId, len(inferences.Inferences), len(forecasts.Forecasts)))

			networkInferenceBuilder, err := NewNetworkInferenceBuilderFromSynthRequest(
				SynthRequest{
					Ctx:                 ctx,
					K:                   k,
					TopicId:             topicId,
					Inferences:          inferences,
					Forecasts:           forecasts,
					NetworkCombinedLoss: networkLosses.CombinedValue,
					EpsilonTopic:        topic.Epsilon,
					EpsilonSafeDiv:      moduleParams.EpsilonSafeDiv,
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

	return networkInferences, forecastImpliedInferencesByWorker, infererWeights, forecasterWeights, inferenceBlockHeight, lossBlockHeight, nil
}
