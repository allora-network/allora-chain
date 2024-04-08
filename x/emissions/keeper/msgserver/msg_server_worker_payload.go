package msgserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// A tx function that accepts a list of forecasts and possibly returns an error
// Need to call this once per forecaster per topic inference solicitation round because protobuf does not nested repeated fields
func (ms msgServer) InsertBulkWorkerPayload(ctx context.Context, msg *types.MsgInsertBulkWorkerPayload) (*types.MsgInsertBulkWorkerPayloadResponse, error) {
	// Check if the nonce is unfulfilled
	nonceUnfulfilled, err := ms.k.IsWorkerNonceUnfulfilled(ctx, msg.TopicId, msg.Nonce)
	if err != nil {
		return nil, err
	}
	if nonceUnfulfilled {
		return nil, types.ErrNonceNotUnfulfilled
	}

	for _, inference := range msg.Inferences {
		// TODO check signatures! throw if invalid!

		if inference.TopicId != msg.TopicId {
			return nil, types.ErrInvalidTopicId
		}
	}

	for _, forecast := range msg.Forecasts {
		// TODO check signatures! throw if invalid!

		if forecast.TopicId != msg.TopicId {
			return nil, types.ErrInvalidTopicId
		}
	}

	inferences := types.Inferences{
		Inferences: msg.Inferences,
	}
	err = ms.k.InsertInferences(ctx, msg.TopicId, *msg.Nonce, inferences)
	if err != nil {
		return nil, err
	}

	forecasts := types.Forecasts{
		Forecasts: msg.Forecasts,
	}
	err = ms.k.InsertForecasts(ctx, msg.TopicId, *msg.Nonce, forecasts)
	if err != nil {
		return nil, err
	}

	// Return an empty response as the operation was successful
	return &types.MsgInsertBulkWorkerPayloadResponse{}, nil
}
