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

	if err = msg.ReputerValueBundle.Validate(); err != nil {
		return nil, errorsmod.Wrap(err, "failed to validate reputer value bundle")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()

	moduleParams, err := ms.pk.GetParams(ctx)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "Error getting params for reputer: %v", &msg.ReputerValueBundle.ValueBundle.Reputer)
	}
	rvb, err := types.NewReputerValueBundleFromInput(msg.ReputerValueBundle)
	if err != nil {
		return nil, errorsmod.Wrapf(err,
			"Reputer bad data format for block: %d", blockHeight)
	}

	vb := rvb.GetValueBundle()

	canSubmit, err := ms.wlk.CanSubmitReputerPayload(ctx, vb.TopicId, vb.Reputer)
	if err != nil {
		return nil, err
	} else if !canSubmit {
		return nil, types.ErrNotPermittedToSubmitReputerPayload
	}

	err = checkInputLength(moduleParams.MaxSerializedMsgLength, msg)
	if err != nil {
		return nil, err
	}

	nonce := vb.ReputerRequestNonce
	topicId := vb.TopicId

	topic, err := ms.tk.GetTopic(ctx, topicId)
	if err != nil {
		return nil, types.ErrInvalidTopicId
	}

	workerNonceUnfulfilled, err := ms.nk.IsWorkerNonceUnfulfilled(ctx, topicId, nonce.ReputerNonce)
	if err != nil {
		return nil, err
	} else if workerNonceUnfulfilled {
		return nil, errorsmod.Wrapf(types.ErrNonceStillUnfulfilled, "worker nonce")
	}

	reputerNonceUnfulfilled, err := ms.nk.IsReputerNonceUnfulfilled(ctx, topicId, nonce.ReputerNonce)
	if err != nil {
		return nil, err
	} else if !reputerNonceUnfulfilled {
		return nil, errorsmod.Wrapf(types.ErrUnfulfilledNonceNotFound, "reputer nonce")
	}

	// Prefer Epoch wall-clock windows when a live scheduler epoch matches this legacy nonce.
	epochFound, err := ms.k.CheckReputerSubmissionWindow(ctx, topicId, *nonce.ReputerNonce)
	if err != nil {
		return nil, err
	}
	if !epochFound {
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
	}

	isRegistered, err := ms.rlk.IsReputerRegisteredInTopic(ctx, topicId, vb.Reputer)
	if err != nil {
		return nil, err
	} else if !isRegistered {
		return nil, errorsmod.Wrapf(types.ErrAddressNotRegistered, "reputer is not registered in this topic")
	}

	// Check that the reputer enough stake in the topic
	stake, err := ms.sk.GetStakeReputerAuthority(ctx, topicId, vb.Reputer)
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

	err = ms.rlk.AppendReputerLoss(sdkCtx, topic, moduleParams, nonce.ReputerNonce.BlockHeight, rvb)
	if err != nil {
		return nil, err
	}

	types.EmitNewInsertReputerPayloadEvent(ctx, vb)
	return &types.InsertReputerPayloadResponse{}, nil
}
