package msgserver

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// A tx function that accepts a individual loss and possibly returns an error
func (ms msgServer) InsertReputerPayload(ctx context.Context, msg *types.MsgServiceInsertReputerPayloadRequest) (*types.MsgServiceInsertReputerPayloadResponse, error) {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	err = checkInputLength(ctx, ms, msg)
	if err != nil {
		return nil, err
	}

	blockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()

	if err := msg.ReputerValueBundle.Validate(); err != nil {
		return nil, errorsmod.Wrapf(err,
			"Error validating reputer value bundle at block height: %d", blockHeight)
	}

	nonce := msg.ReputerValueBundle.ValueBundle.ReputerRequestNonce
	topicId := msg.ReputerValueBundle.ValueBundle.TopicId

	// Check if the topic exists
	topic, err := ms.k.GetTopic(ctx, topicId)
	if err != nil {
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

	// Check if the ground truth lag has passed: if blockheight > nonce.BlockHeight + topic.GroundTruthLag
	if !ms.k.BlockWithinReputerSubmissionWindowOfNonce(topic, *nonce, blockHeight) {
		return nil, types.ErrReputerNonceWindowNotAvailable
	}

	isRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, topicId, msg.ReputerValueBundle.ValueBundle.Reputer)
	if err != nil {
		return nil, err
	}
	if !isRegistered {
		return nil, errorsmod.Wrapf(types.ErrAddressNotRegistered, "reputer is not registered in this topic")
	}

	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for sender: %v", &msg.Sender)
	}

	// Check that the reputer enough stake in the topic
	stake, err := ms.k.GetStakeReputerAuthority(ctx, topicId, msg.ReputerValueBundle.ValueBundle.Reputer)
	if err != nil {
		return nil, err
	}
	if stake.LT(params.RequiredMinimumStake) {
		return nil, errorsmod.Wrapf(types.ErrInsufficientStake, "reputer does not have sufficient stake in the topic")
	}

	// Before accepting data, transfer fee amount from sender to ecosystem bucket
	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, topicId, params.DataSendingFee)
	if err != nil {
		return nil, err
	}

	err = ms.k.AppendReputerLoss(ctx, topic, blockHeight, nonce.ReputerNonce.BlockHeight, msg.ReputerValueBundle)
	if err != nil {
		return nil, err
	}

	return &types.MsgServiceInsertReputerPayloadResponse{}, nil
}
