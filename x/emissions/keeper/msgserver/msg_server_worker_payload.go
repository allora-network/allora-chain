package msgserver

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// A tx function that accepts a individual inference and forecast and possibly returns an error
// Need to call this once per forecaster per topic inference solicitation round because protobuf does not nested repeated fields
// Only 1 payload per registered worker is kept, ignore the rest. In particular, take the first payload from each
// registered worker and none from any unregistered actor.
// Signatures, anti-sybil procedures, and "skimming of only the top few workers by EMA score descending" should be done here.
func (ms msgServer) InsertWorkerPayload(ctx context.Context, msg *types.InsertWorkerPayloadRequest) (*types.InsertWorkerPayloadResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	err = checkInputLength(ctx, ms, msg)
	if err != nil {
		return nil, err
	}

	if err := msg.WorkerDataBundle.Validate(); err != nil {
		return nil, errorsmod.Wrapf(err,
			"Worker invalid data for block: %d", blockHeight)
	}

	nonce := msg.WorkerDataBundle.Nonce
	topicId := msg.WorkerDataBundle.TopicId

	// Check if the topic exists. Will throw if topic does not exist
	topic, err := ms.k.GetTopic(ctx, topicId)
	if err != nil {
		return nil, types.ErrInvalidTopicId
	}
	// Check if the nonce is unfulfilled
	nonceUnfulfilled, err := ms.k.IsWorkerNonceUnfulfilled(ctx, topicId, nonce)
	if err != nil {
		return nil, err
	}
	// If the nonce is already fulfilled, return an error
	if !nonceUnfulfilled {
		return nil, types.ErrUnfulfilledNonceNotFound
	}

	// Check if the window time is open
	if !ms.k.BlockWithinWorkerSubmissionWindowOfNonce(topic, *nonce, blockHeight) {
		return nil, errorsmod.Wrapf(
			types.ErrWorkerNonceWindowNotAvailable,
			"Worker window not open for topic: %d, current block %d , nonce block height: %d , start window: %d, end window: %d",
			topicId, blockHeight, nonce.BlockHeight, nonce.BlockHeight+topic.WorkerSubmissionWindow, nonce.BlockHeight+topic.GroundTruthLag,
		)
	}

	isInfererRegistered, err := ms.k.IsWorkerRegisteredInTopic(ctx, topicId, msg.WorkerDataBundle.Worker)
	if err != nil {
		return nil, err
	}
	if !isInfererRegistered {
		return nil, errorsmod.Wrapf(types.ErrAddressNotRegistered, "worker is not registered in this topic")
	}

	// Before accepting data, transfer fee amount from sender to ecosystem bucket
	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for sender: %v", &msg.Sender)
	}
	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, topicId, params.DataSendingFee)
	if err != nil {
		return nil, err
	}

	// Inferences
	if msg.WorkerDataBundle.InferenceForecastsBundle.Inference != nil {
		inference := msg.WorkerDataBundle.InferenceForecastsBundle.Inference
		if inference == nil {
			return nil, errorsmod.Wrapf(types.ErrNoValidInferences, "Inference not found")
		}
		if inference.TopicId != msg.WorkerDataBundle.TopicId {
			return nil, errorsmod.Wrapf(types.ErrInvalidTopicId,
				"inferer not using the same topic as bundle")
		}

		err = ms.k.AppendInference(ctx, topic, blockHeight, nonce.BlockHeight, inference)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "Error appending inference")
		}
	}

	// Forecasts
	if msg.WorkerDataBundle.InferenceForecastsBundle.Forecast != nil {
		forecast := msg.WorkerDataBundle.InferenceForecastsBundle.Forecast
		if len(forecast.ForecastElements) == 0 {
			return nil, errorsmod.Wrapf(types.ErrNoValidForecastElements, "No valid forecast elements found in Forecast")
		}
		if forecast.TopicId != msg.WorkerDataBundle.TopicId {
			return nil, errorsmod.Wrapf(types.ErrInvalidTopicId, "forecaster not using the same topic as bundle")
		}

		// Limit forecast elements to top inferers
		latestScoresForForecastedInferers := make([]types.Score, 0)
		for _, el := range forecast.ForecastElements {
			score, err := ms.k.GetInfererScoreEma(ctx, forecast.TopicId, el.Inferer)
			if err != nil {
				continue
			}
			latestScoresForForecastedInferers = append(latestScoresForForecastedInferers, score)
		}

		moduleParams, err := ms.k.GetParams(ctx)
		if err != nil {
			return nil, err
		}
		_, _, topNInferer := actorutils.FindTopNByScoreDesc(
			sdkCtx,
			moduleParams.MaxElementsPerForecast,
			latestScoresForForecastedInferers,
			forecast.BlockHeight,
		)

		// Remove duplicate forecast elements
		acceptedForecastElements := make([]*types.ForecastElement, 0)
		seenInferers := make(map[string]bool)
		for _, el := range forecast.ForecastElements {
			// Check if the forecasted inferer is registered in the topic
			isInfererRegistered, err := ms.k.IsWorkerRegisteredInTopic(ctx, topicId, el.Inferer)
			if err != nil {
				return nil, err
			}
			if !isInfererRegistered {
				return nil, errorsmod.Wrapf(err,
					"Error forecasted inferer address is not registered in this topic")
			}

			notAlreadySeen := !seenInferers[el.Inferer]
			_, isTopInferer := topNInferer[el.Inferer]
			if notAlreadySeen && isTopInferer {
				acceptedForecastElements = append(acceptedForecastElements, el)
				seenInferers[el.Inferer] = true
			}
		}

		if len(acceptedForecastElements) > 0 {
			forecast.ForecastElements = acceptedForecastElements
			err = ms.k.AppendForecast(ctx, topic, blockHeight, nonce.BlockHeight, forecast)
			if err != nil {
				return nil, errorsmod.Wrapf(err,
					"Error appending forecast")
			}
		}
	}
	return &types.InsertWorkerPayloadResponse{}, nil
}
