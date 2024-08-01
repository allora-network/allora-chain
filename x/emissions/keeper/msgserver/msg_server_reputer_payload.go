package msgserver

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// A tx function that accepts a individual loss and possibly returns an error
func (ms msgServer) InsertReputerPayload(ctx context.Context, msg *types.MsgInsertReputerPayload) (*types.MsgInsertReputerPayloadResponse, error) {
	err := checkInputLength(ctx, ms, msg)
	if err != nil {
		return nil, err
	}

	blockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()

	if err := msg.ReputerValueBundle.Validate(); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidWorkerData,
			"Worker invalid data for block: %d", blockHeight)
	}

	nonce := msg.ReputerValueBundle.ValueBundle.ReputerRequestNonce
	topicId := msg.ReputerValueBundle.ValueBundle.TopicId

	// Check if the topic exists
	topicExists, err := ms.k.TopicExists(ctx, topicId)
	if err != nil || !topicExists {
		return nil, types.ErrInvalidTopicId
	}

	// Check if the worker nonce is fulfilled
	workerNonceUnfulfilled, err := ms.k.IsWorkerNonceUnfulfilled(ctx, topicId, nonce.ReputerNonce)
	if err != nil {
		return nil, err
	}
	// Returns an error if unfulfilled worker nonce exists
	if workerNonceUnfulfilled {
		return nil, types.ErrNonceStillUnfulfilled
	}

	// Check if the reputer nonce is unfulfilled
	reputerNonceUnfulfilled, err := ms.k.IsReputerNonceUnfulfilled(ctx, topicId, nonce.ReputerNonce)
	if err != nil {
		return nil, err
	}
	// If the reputer nonce is already fulfilled, return an error
	if !reputerNonceUnfulfilled {
		return nil, types.ErrUnfulfilledNonceNotFound
	}

	topic, err := ms.k.GetTopic(ctx, topicId)
	if err != nil {
		return nil, types.ErrInvalidTopicId
	}

	// Check if the ground truth lag has passed: if blockheight > nonce.BlockHeight + topic.GroundTruthLag
	if blockHeight < nonce.ReputerNonce.BlockHeight+topic.GroundTruthLag {
		return nil, types.ErrReputerNonceWindowNotAvailable
	}

	// Before creating topic, transfer fee amount from creator to ecosystem bucket
	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for sender: %v", &msg.Sender)
	}
	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, topicId, params.DataSendingFee, "insert reputer payload")
	if err != nil {
		return nil, err
	}

	err = ms.k.AppendReputerLoss(ctx, topicId, nonce.ReputerNonce.BlockHeight, msg.ReputerValueBundle)
	if err != nil {
		return nil, err
	}

	return &types.MsgInsertReputerPayloadResponse{}, nil
}
