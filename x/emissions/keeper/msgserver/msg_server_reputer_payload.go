package msgserver

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/metrics"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// A tx function that accepts a individual loss and possibly returns an error
func (ms msgServer) InsertReputerPayload(ctx context.Context, msg *types.InsertReputerPayloadRequest) (_ *types.InsertReputerPayloadResponse, err error) {
	defer metrics.RecordMetrics("InsertReputerPayload", time.Now(), func() bool { return err == nil })

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()

	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	err = checkInputLength(ctx, ms, msg)
	if err != nil {
		return nil, err
	}

	err = msg.ReputerValueBundle.Validate()
	if err != nil {
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
		return nil, errorsmod.Wrapf(types.ErrNonceStillUnfulfilled, "worker nonce")
	}

	// Check if the reputer nonce is unfulfilled
	reputerNonceUnfulfilled, err := ms.k.IsReputerNonceUnfulfilled(ctx, topicId, nonce.ReputerNonce)
	if err != nil {
		return nil, err
	}
	// If the reputer nonce is already fulfilled, return an error
	if !reputerNonceUnfulfilled {
		return nil, errorsmod.Wrapf(types.ErrUnfulfilledNonceNotFound, "reputer nonce")
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

	moduleParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for sender: %v", &msg.Sender)
	}

	// Check that the reputer enough stake in the topic
	stake, err := ms.k.GetStakeReputerAuthority(ctx, topicId, msg.ReputerValueBundle.ValueBundle.Reputer)
	if err != nil {
		return nil, err
	}
	if stake.LT(moduleParams.RequiredMinimumStake) {
		return nil, errorsmod.Wrapf(types.ErrInsufficientStake, "reputer does not have sufficient stake in the topic")
	}

	// Before accepting data, transfer fee amount from sender to ecosystem bucket
	err = sendEffectiveRevenueActivateTopicIfWeightSufficient(ctx, ms, msg.Sender, topicId, moduleParams.DataSendingFee)
	if err != nil {
		return nil, err
	}

	err = ms.k.AppendReputerLoss(sdkCtx, topic, moduleParams, nonce.ReputerNonce.BlockHeight, msg.ReputerValueBundle)
	if err != nil {
		return nil, err
	}

	return &types.InsertReputerPayloadResponse{}, err
}
