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
func (ms msgServer) InsertWorkerPayload(ctx context.Context, msg *types.MsgInsertWorkerPayload) (*types.MsgInsertWorkerPayloadResponse, error) {
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

	// Check if the topic exists
	topicExists, err := ms.k.TopicExists(ctx, topicId)
	if err != nil || !topicExists {
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

	topic, err := ms.k.GetTopic(ctx, topicId)
	if err != nil {
		return nil, types.ErrInvalidTopicId
	}

	// Check if the window time is open
	if blockHeight < nonce.BlockHeight ||
		blockHeight > nonce.BlockHeight+topic.WorkerSubmissionWindow {
		return nil, errorsmod.Wrapf(
			types.ErrWorkerNonceWindowNotAvailable,
			"Worker window not open for topic: %d, current block %d , nonce block height: %d , start window: %d, end window: %d",
			topicId, blockHeight, nonce.BlockHeight, nonce.BlockHeight+topic.WorkerSubmissionWindow, nonce.BlockHeight+topic.GroundTruthLag,
		)
	}

	// Before creating topic, transfer fee amount from creator to ecosystem bucket
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
		isInfererRegistered, err := ms.k.IsWorkerRegisteredInTopic(ctx, topicId, inference.Inferer)
		if err != nil {
			return nil, errorsmod.Wrapf(err,
				"error checking if inferer address is registered in this topic")
		}
		if !isInfererRegistered {
			return nil, errorsmod.Wrapf(types.ErrAddressNotRegistered,
				"inferer address is not registered in this topic")
		}
		err = ms.k.AppendInference(ctx, topicId, *nonce, inference)
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
		isForecasterRegistered, err := ms.k.IsWorkerRegisteredInTopic(ctx, topicId, forecast.Forecaster)
		if err != nil {
			return nil, errorsmod.Wrapf(err,
				"error checking if forecaster address is registered in this topic")
		}
		if !isForecasterRegistered {
			return nil, errorsmod.Wrapf(types.ErrAddressNotRegistered,
				"forecaster address is not registered in this topic")
		}

		// LImit forecast elements for top inferers
		latestScoresForForecastedInferers := make([]types.Score, 0)
		for _, el := range forecast.ForecastElements {
			score, err := ms.k.GetLatestInfererScore(ctx, forecast.TopicId, el.Inferer)
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
			notAlreadySeen := !seenInferers[el.Inferer]
			_, isTopInferer := topNInferer[el.Inferer]
			if notAlreadySeen && isTopInferer {
				acceptedForecastElements = append(acceptedForecastElements, el)
				seenInferers[el.Inferer] = true
			}
		}

		if len(acceptedForecastElements) > 0 {
			forecast.ForecastElements = acceptedForecastElements
			err = ms.k.AppendForecast(ctx, topicId, *nonce, forecast)
			if err != nil {
				return nil, errorsmod.Wrapf(err,
					"Error appending forecast")
			}
		}
	}
	return &types.MsgInsertWorkerPayloadResponse{}, nil
}
