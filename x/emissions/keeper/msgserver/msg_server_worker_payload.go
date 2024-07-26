package msgserver

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/app/params"
	"github.com/allora-network/allora-chain/x/emissions/types"
	mintTypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// A tx function that accepts a individual inference and forecast and possibly returns an error
// Need to call this once per forecaster per topic inference solicitation round because protobuf does not nested repeated fields
func (ms msgServer) InsertWorkerPayload(ctx context.Context, msg *types.MsgInsertWorkerPayload) (*types.MsgInsertWorkerPayloadResponse, error) {
	err := checkInputLength(ctx, ms, msg)
	if err != nil {
		return nil, err
	}

	if msg.WorkerDataBundle == nil {
		return nil, errorsmod.Wrapf(types.ErrNoValidBundles,
			"Worker data bundle cannot be nil")
	}

	if msg.WorkerDataBundle.InferenceForecastsBundle == nil {
		return nil, errorsmod.Wrapf(types.ErrNoValidBundles,
			"Inference-Forecast bundle data bundle cannot be nil")
	}

	nonce := msg.Nonce
	topicId := msg.TopicId

	// Check if the topic exists
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
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

	blockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
	topic, err := ms.k.GetTopic(ctx, msg.TopicId)
	if err != nil {
		return nil, types.ErrInvalidTopicId
	}

	// Check if the window time is open
	if blockHeight <= nonce.BlockHeight ||
		blockHeight > nonce.BlockHeight+topic.WorkerSubmissionWindow {
		return nil, errorsmod.Wrapf(
			types.ErrWorkerNonceWindowNotAvailable,
			"Worker window not open for topic: %d, current block %d , nonce block height: %d , start window: %d, end window: %d",
			topicId, blockHeight, nonce.BlockHeight, nonce.BlockHeight+topic.WorkerSubmissionWindow, nonce.BlockHeight+topic.GroundTruthLag,
		)
	}

	if err := msg.WorkerDataBundle.Validate(); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidWorkerData,
			"Worker invalid data for block: %v", &nonce.BlockHeight)
	}

	hasEnoughBal, fee, err := ms.CheckBalanceForSendingDataFee(ctx, msg.Sender)
	if err != nil {
		return nil, errorsmod.Wrapf(err,
			"Error checking balance for sender: %v", &msg.Sender)
	}
	if !hasEnoughBal {
		return nil, types.ErrDataSenderNotEnoughDenom
	}

	// Before creating topic, transfer fee amount from creator to ecosystem bucket
	err = ms.k.SendCoinsFromAccountToModule(ctx, msg.Sender, mintTypes.EcosystemModuleName, sdk.NewCoins(fee))
	if err != nil {
		return nil, errorsmod.Wrapf(err,
			"Error sending coins for sender: %v", &msg.Sender)
	}

	if msg.WorkerDataBundle.InferenceForecastsBundle.Inference != nil {
		inference := msg.WorkerDataBundle.InferenceForecastsBundle.Inference
		err = ms.k.AppendInference(ctx, topicId, *nonce, inference)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "Error appending inference")
		}
	}

	// Append this individual inference to all inferences
	if msg.WorkerDataBundle.InferenceForecastsBundle.Forecast != nil {
		forecast := msg.WorkerDataBundle.InferenceForecastsBundle.Forecast
		err = ms.k.AppendForecast(ctx, topicId, *nonce, forecast)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "Error appending forecast")
		}
	}
	return &types.MsgInsertWorkerPayloadResponse{}, nil
}

func (ms msgServer) CheckBalanceForSendingDataFee(ctx context.Context, address string) (bool, sdk.Coin, error) {
	moduleParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return false, sdk.Coin{}, err
	}
	fee := sdk.NewCoin(params.DefaultBondDenom, moduleParams.DataSendingFee)
	accAddress, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return false, fee, err
	}
	balance := ms.k.GetBankBalance(ctx, accAddress, fee.Denom)
	return balance.IsGTE(fee), fee, nil
}
