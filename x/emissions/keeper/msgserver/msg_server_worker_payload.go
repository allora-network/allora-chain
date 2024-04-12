package msgserver

import (
	"context"
	"encoding/json"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
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

	// Verify nonce signature
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	pk := ms.k.AccountKeeper().GetAccount(ctx, senderAddr)
	stringNonce := strconv.FormatInt(msg.Nonce.Nonce, 10)
	nonceBytes := []byte(stringNonce)
	if pk == nil || !pk.GetPubKey().VerifySignature(nonceBytes, msg.Signature) {
		return nil, types.ErrSignatureVerificationFailed
	}

	for _, inference := range msg.Inferences {
		// Verify signature on every inference
		workerAddr, err := sdk.AccAddressFromBech32(inference.Worker)
		if err != nil {
			return nil, err
		}
		pk := ms.k.AccountKeeper().GetAccount(ctx, workerAddr)
		src, _ := json.Marshal(inference.Value)
		if pk == nil || !pk.GetPubKey().VerifySignature(src, inference.Signature) {
			return nil, types.ErrSignatureVerificationFailed
		}
		if inference.TopicId != msg.TopicId {
			return nil, types.ErrInvalidTopicId
		}
	}

	for _, forecast := range msg.Forecasts {
		// Verify signature on every forecast
		workerAddr, err := sdk.AccAddressFromBech32(forecast.Forecaster)
		if err != nil {
			return nil, err
		}
		pk := ms.k.AccountKeeper().GetAccount(ctx, workerAddr)
		src, _ := json.Marshal(forecast.ForecastElements)
		if !pk.GetPubKey().VerifySignature(src, forecast.Signature) {
			return nil, types.ErrSignatureVerificationFailed
		}

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
