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

	// Check if the topic exists
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil || !topicExists {
		return nil, types.ErrInvalidTopicId
	}
	// Check if the window time has passed: if blockheight > nonce.BlockHeight + topic.WorkerSubmissionWindow
	blockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
	nonce := msg.Nonce

	topicId := msg.TopicId
	topic, err := ms.k.GetTopic(ctx, msg.TopicId)
	if err != nil {
		return nil, types.ErrInvalidTopicId
	}

	if blockHeight > topic.EpochLastEnded+topic.WorkerSubmissionWindow {
		return nil, types.ErrWorkerNonceWindowNotAvailable
	}

	if err := msg.WorkerDataBundles.Validate(); err != nil {
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

	// Get all inferences from this topic, nonce
	inferences, err := ms.k.GetInferencesAtBlock(ctx, topicId, nonce.BlockHeight)
	if err != nil {
		return nil, err
	}
	// Append this individual inference to all inferences
	inference := msg.WorkerDataBundles.InferenceForecastsBundle.Inference

	inferencesToInsert := types.Inferences{
		Inferences: append(inferences.Inferences, inference),
	}
	err = ms.k.InsertInferences(ctx, topicId, *nonce, inferencesToInsert)
	if err != nil {
		return nil, err
	}

	// Get all forecasts from this topic, nonce
	forecasts, err := ms.k.GetForecastsAtBlock(ctx, topicId, nonce.BlockHeight)
	if err != nil {
		return nil, err
	}
	// Append this individual inference to all inferences
	forecast := msg.WorkerDataBundles.InferenceForecastsBundle.Forecast

	forecastsToInsert := types.Forecasts{
		Forecasts: append(forecasts.Forecasts, forecast),
	}
	err = ms.k.InsertForecasts(ctx, topicId, *nonce, forecastsToInsert)
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
