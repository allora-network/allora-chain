package msgserver

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// A tx function that accepts a individual loss and possibly returns an error
func (ms msgServer) InsertReputerPayload(ctx context.Context, msg *types.MsgInsertReputerPayload) (*types.MsgInsertReputerPayloadResponse, error) {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	err = checkInputLength(ctx, ms, msg)
	if err != nil {
		return nil, err
	}

	blockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()

	// Call the bundles self validation method
	if err := msg.ReputerValueBundle.Validate(); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidWorkerData,
			"Error validating reputer value bundle: %v", err)
	}

	nonce := msg.ReputerValueBundle.ValueBundle.ReputerRequestNonce
	topicId := msg.ReputerValueBundle.ValueBundle.TopicId
	reputer := msg.Sender

	// the reputer in the bundle must match the sender of this transaction
	if reputer != msg.ReputerValueBundle.ValueBundle.Reputer {
		return nil, errorsmod.Wrapf(types.ErrInvalidWorkerData,
			"Reputer cannot upload value bundle for another reputer")
	}

	// Check if the topic exists
	topicExists, err := ms.k.TopicExists(ctx, topicId)
	if err != nil || !topicExists {
		return nil, types.ErrInvalidTopicId
	}

	// Check that the reputer is registered in the topic
	isReputerRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, topicId, reputer)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error checking if reputer is registered in topic")
	}
	if !isReputerRegistered {
		return nil, errorsmod.Wrapf(types.ErrInvalidWorkerData,
			"Reputer not registered in topic")
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

	moduleParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for sender: %v", &msg.Sender)
	}

	// reputer must have minimum stake in order to participate in topic
	reputerStake, err := ms.k.GetStakeReputerAuthority(ctx, topicId, reputer)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting reputer stake for sender: %v", &msg.Sender)
	}
	if reputerStake.LT(moduleParams.RequiredMinimumStake) {
		return nil, errorsmod.Wrapf(types.ErrInvalidWorkerData,
			"Reputer must have minimum stake in order to participate in topic")
	}

	// Before activating topic, transfer fee amount from creator to ecosystem bucket
	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, topicId, moduleParams.DataSendingFee)
	if err != nil {
		return nil, err
	}

	err = ms.k.UpsertReputerLoss(ctx, topicId, nonce.ReputerNonce.BlockHeight, msg.ReputerValueBundle)
	if err != nil {
		return nil, err
	}

	return &types.MsgInsertReputerPayloadResponse{}, nil
}
