package msgserver

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// A tx function that accepts a individual loss and possibly returns an error
func (ms msgServer) InsertReputerPayload(ctx context.Context, msg *types.InsertReputerPayloadRequest) (_ *types.InsertReputerPayloadResponse, err error) {
	defer metrics.RecordMetrics("InsertReputerPayload", time.Now(), &err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()

	moduleParams, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for reputer: %v", &msg.ReputerValueBundle.ValueBundle.Reputer)
	}
	rvb, err := types.NewInputReputerValueBundleFromInput(msg.ReputerValueBundle)
	if err != nil {
		return nil, errorsmod.Wrapf(err,
			"Reputer bad data format for block: %d", blockHeight)
	}

	canSubmit, err := ms.k.CanSubmitReputerPayload(ctx, rvb.TopicId, rvb.Reputer)
	if err != nil {
		return nil, err
	} else if !canSubmit {
		return nil, types.ErrNotPermittedToSubmitReputerPayload
	}

	err = checkInputLength(moduleParams.MaxSerializedMsgLength, msg)
	if err != nil {
		return nil, err
	}

	nonce := rvb.ReputerRequestNonce
	topicId := rvb.TopicId

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

	isRegistered, err := ms.k.IsReputerRegisteredInTopic(ctx, topicId, rvb.Reputer)
	if err != nil {
		return nil, err
	} else if !isRegistered {
		return nil, errorsmod.Wrapf(types.ErrAddressNotRegistered, "reputer is not registered in this topic")
	}

	// Check that the reputer enough stake in the topic
	stake, err := ms.k.GetStakeReputerAuthority(ctx, topicId, rvb.Reputer)
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

	types.EmitNewInsertReputerPayloadEvent(ctx, rvb)
	return &types.InsertReputerPayloadResponse{}, nil
}
