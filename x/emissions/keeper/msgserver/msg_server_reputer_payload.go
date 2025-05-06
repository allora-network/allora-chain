package msgserver

import (
	"context"
	"strconv"
	"time"

	errorsmod "cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/metrics"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// A tx function that accepts a individual loss and possibly returns an error
func (ms msgServer) InsertReputerPayload(ctx context.Context, msg *types.InsertReputerPayloadRequest) (_ *types.InsertReputerPayloadResponse, err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()

	moduleParams, err := ms.k.GetParams(ctx)
	if err != nil {
		defer metrics.RecordMetrics("InsertReputerPayload", time.Now(), &err, map[string]string{"error": err.Error()})
		return nil, errorsmod.Wrapf(err, "Error getting params for reputer: %v", &msg.ReputerValueBundle.ValueBundle.Reputer)
	}
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		defer metrics.RecordMetrics("InsertReputerPayload", time.Now(), &err, map[string]string{"error": err.Error()})
		return nil, errorsmod.Wrapf(err, "Error validating sender address")
	}
	// fast permission check before entering validations which are more expensive
	if msg.ReputerValueBundle != nil && msg.ReputerValueBundle.ValueBundle != nil {
		err = ms.k.ValidateStringIsBech32(msg.ReputerValueBundle.ValueBundle.Reputer)
		if err != nil {
			defer metrics.RecordMetrics("InsertReputerPayload", time.Now(), &err, map[string]string{"error": err.Error()})
			return nil, errorsmod.Wrapf(err, "Error validating reputer address")
		}
		canSubmit, err := ms.k.CanSubmitReputerPayload(ctx, msg.ReputerValueBundle.ValueBundle.TopicId, msg.ReputerValueBundle.ValueBundle.Reputer)
		if err != nil {
			defer metrics.RecordMetrics("InsertReputerPayload", time.Now(), &err, map[string]string{"error": err.Error()})
			return nil, err
		} else if !canSubmit {
			defer metrics.RecordMetrics("InsertReputerPayload", time.Now(), &err, map[string]string{"error": types.ErrNotPermittedToSubmitReputerPayload.Error()})
			return nil, types.ErrNotPermittedToSubmitReputerPayload
		}
	} else {
		defer metrics.RecordMetrics("InsertReputerPayload", time.Now(), &err, map[string]string{"error": types.ErrInvalidReputerData.Error()})
		return nil, types.ErrInvalidReputerData
	}

	// includes validation
	rvb, err := types.NewInputReputerValueBundleFromInput(msg.ReputerValueBundle)
	if err != nil {
		defer metrics.RecordMetrics("InsertReputerPayload", time.Now(), &err, map[string]string{"error": err.Error()})
		return nil, errorsmod.Wrapf(err,
			"Reputer bad data format for block: %d", blockHeight)
	}
	// if validated, record metrics with appropriate labels
	labels := map[string]string{
		"address":     msg.ReputerValueBundle.ValueBundle.Reputer,
		"topic_id":    strconv.FormatUint(msg.ReputerValueBundle.ValueBundle.TopicId, 10),
		"nonce":       strconv.FormatUint(uint64(msg.ReputerValueBundle.ValueBundle.ReputerRequestNonce.ReputerNonce.BlockHeight), 10),
		"blockHeight": strconv.FormatInt(blockHeight, 10),
	}
	defer metrics.RecordMetrics("InsertReputerPayload", time.Now(), &err, labels)

	err = checkInputLength(moduleParams.MaxSerializedMsgLength, msg)
	if err != nil {
		return nil, err
	}

	nonce := rvb.ValueBundle.ReputerRequestNonce
	topicId := rvb.ValueBundle.TopicId

	topic, err := ms.k.GetTopic(ctx, topicId)
	if err != nil {
		return nil, types.ErrInvalidTopicId
	}

	workerNonceUnfulfilled, err := ms.k.IsWorkerNonceUnfulfilled(ctx, topicId, nonce.ReputerNonce)
	if err != nil {
		return nil, err
	} else if workerNonceUnfulfilled {
		return nil, errorsmod.Wrapf(types.ErrNonceStillUnfulfilled, "worker nonce")
	}

	reputerNonceUnfulfilled, err := ms.k.IsReputerNonceUnfulfilled(ctx, topicId, nonce.ReputerNonce)
	if err != nil {
		return nil, err
	} else if !reputerNonceUnfulfilled {
		return nil, errorsmod.Wrapf(types.ErrUnfulfilledNonceNotFound, "reputer nonce")
	}

	withinWindow, err := keeper.BlockWithinReputerSubmissionWindowOfNonce(topic, *nonce, blockHeight)
	if err != nil {
		return nil, err
	} else if !withinWindow {
		return nil, errorsmod.Wrapf(
			types.ErrReputerNonceWindowNotAvailable,
			"Reputer window not open for topic: %d, current block %d, start window: %d, end window: %d",
			topicId, blockHeight, nonce.ReputerNonce.BlockHeight+topic.GroundTruthLag, nonce.ReputerNonce.BlockHeight+topic.GroundTruthLag+topic.EpochLength*2,
		)
	}

	isRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, topicId, rvb.ValueBundle.Reputer)
	if err != nil {
		return nil, err
	} else if !isRegistered {
		return nil, errorsmod.Wrapf(types.ErrAddressNotRegistered, "reputer is not registered in this topic")
	}

	// Check that the reputer enough stake in the topic
	stake, err := ms.k.GetStakeReputerAuthority(ctx, topicId, rvb.ValueBundle.Reputer)
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

	err = ms.k.AppendReputerLoss(sdkCtx, topic, moduleParams, nonce.ReputerNonce.BlockHeight, rvb)
	if err != nil {
		return nil, err
	}

	return &types.InsertReputerPayloadResponse{}, err
}
