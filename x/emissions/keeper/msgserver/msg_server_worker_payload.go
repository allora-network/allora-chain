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

	blockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()

	if err := msg.WorkerDataBundle.Validate(); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidWorkerData,
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
		if inference.TopicId != msg.WorkerDataBundle.TopicId {
			return nil, errorsmod.Wrapf(err,
				"Error inferer not use same topic")
		}
		isInfererRegistered, err := ms.k.IsWorkerRegisteredInTopic(ctx, topicId, inference.Inferer)
		if err != nil {
			return nil, errorsmod.Wrapf(err,
				"Error inferer address is not registered in this topic")
		}
		if !isInfererRegistered {
			return nil, errorsmod.Wrapf(err,
				"Error inferer address is not registered in this topic")
		}
		err = ms.k.AppendInference(ctx, topicId, *nonce, inference)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "Error appending inference")
		}
	}

	// Append this individual inference to all inferences
	if msg.WorkerDataBundle.InferenceForecastsBundle.Forecast != nil {
		forecast := msg.WorkerDataBundle.InferenceForecastsBundle.Forecast
		if forecast.TopicId != msg.WorkerDataBundle.TopicId {
			return nil, errorsmod.Wrapf(err,
				"Error forecaster not use same topic")
		}
		isForecasterRegistered, err := ms.k.IsWorkerRegisteredInTopic(ctx, topicId, forecast.Forecaster)
		if err != nil {
			return nil, errorsmod.Wrapf(err,
				"Error forecaster address is not registered in this topic")
		}
		if !isForecasterRegistered {
			return nil, errorsmod.Wrapf(err,
				"Error forecaster address is not registered in this topic")
		}
		err = ms.k.AppendForecast(ctx, topicId, *nonce, forecast)
		if err != nil {
			return nil, errorsmod.Wrapf(err,
				"Error appending forecast")
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
