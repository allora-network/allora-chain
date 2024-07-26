package msgserver

import (
	"context"
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

	// Check if the window time has passed: if blockheight > topic.EpochLastEnded+topic.WorkerSubmissionWindow
	if blockHeight <= nonce.BlockHeight+topic.WorkerSubmissionWindow ||
		blockHeight > nonce.BlockHeight+topic.GroundTruthLag {
		return nil, types.ErrWorkerNonceWindowNotAvailable
	}

	if err := msg.WorkerDataBundle.Validate(); err != nil {
		return nil, types.ErrInvalidReputerData
	}

	hasEnoughBal, fee, err := ms.CheckBalanceForSendingDataFee(ctx, msg.Sender)
	if err != nil {
		return nil, err
	}
	if !hasEnoughBal {
		return nil, types.ErrDataSenderNotEnoughDenom
	}

	// Before creating topic, transfer fee amount from creator to ecosystem bucket
	err = ms.k.SendCoinsFromAccountToModule(ctx, msg.Sender, mintTypes.EcosystemModuleName, sdk.NewCoins(fee))
	if err != nil {
		return nil, err
	}

	// Append this individual inference to all inferences
	inference := msg.WorkerDataBundle.InferenceForecastsBundle.Inference

	err = ms.k.AppendInference(ctx, topicId, *nonce, inference)
	if err != nil {
		return nil, err
	}

	// Append this individual inference to all inferences
	forecast := msg.WorkerDataBundle.InferenceForecastsBundle.Forecast
	err = ms.k.AppendForecast(ctx, topicId, *nonce, forecast)
	if err != nil {
		return nil, err
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
