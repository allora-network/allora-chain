package msgserver

import (
	"context"
	"github.com/allora-network/allora-chain/x/emissions/types"
	mintTypes "github.com/allora-network/allora-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// A tx function that accepts a individual loss and possibly returns an error
func (ms msgServer) InsertReputerPayload(ctx context.Context, msg *types.MsgInsertReputerPayload) (*types.MsgInsertReputerPayloadResponse, error) {
	err := checkInputLength(ctx, ms, msg)
	if err != nil {
		return nil, err
	}

	nonce := msg.ReputerRequestNonce
	topicId := msg.TopicId

	// Check if the topic exists
	topicExists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil || !topicExists {
		return nil, types.ErrInvalidTopicId
	}

	// Check if the nonce is unfulfilled
	nonceUnfulfilled, err := ms.k.IsWorkerNonceUnfulfilled(ctx, topicId, nonce.ReputerNonce)
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

	// Check if the ground truth lag has passed: if blockheight > nonce.BlockHeight + topic.GroundTruthLag
	if blockHeight <= nonce.ReputerNonce.BlockHeight+topic.GroundTruthLag {
		return nil, types.ErrWorkerNonceWindowNotAvailable
	}

	if err := msg.ReputerValueBundle.Validate(); err != nil {
		return nil, types.ErrInvalidWorkerData
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

	err = ms.k.AppendReputerLossAtBlock(ctx, topicId, nonce.ReputerNonce.BlockHeight, msg.ReputerValueBundle)
	if err != nil {
		return nil, err
	}

	return &types.MsgInsertReputerPayloadResponse{}, nil
}
