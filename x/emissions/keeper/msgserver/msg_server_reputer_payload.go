package msgserver

import (
	"context"
	"fmt"
	"sort"
	"time"

	errorsmod "cosmossdk.io/errors"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/metrics"

	sdk "github.com/cosmos/cosmos-sdk/types"

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

	canSubmit, err := ms.k.CanSubmitReputerPayload(ctx, rvb.ValueBundle.TopicId, rvb.ValueBundle.Reputer)
	if err != nil {
		return nil, err
	} else if !canSubmit {
		return nil, types.ErrNotPermittedToSubmitReputerPayload
	}

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

	if err := ms.validateValueBundle(ctx, msg.ReputerValueBundle.ValueBundle, topicId, nonce.ReputerNonce.BlockHeight); err != nil {
		return nil, err
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

func (ms msgServer) validateValueBundle(ctx context.Context, valueBundle *types.InputValueBundle, topicId uint64, blockHeight int64) error {
	networkInferences, err := ms.k.GetNetworkInferences(ctx, topicId, blockHeight)
	if err != nil {
		return errorsmod.Wrapf(err, "error getting inferences")
	}

	if err := validateWorkerValues(
		types.ConvertToWorkerValues(networkInferences.InfererValues),
		types.ConvertToWorkerValues(valueBundle.InfererValues),
	); err != nil {
		return err
	}

	if err := validateWorkerValues(
		types.ConvertToWorkerValues(networkInferences.ForecasterValues),
		types.ConvertToWorkerValues(valueBundle.ForecasterValues),
	); err != nil {
		return err
	}

	if err := validateWorkerValues(
		types.ConvertToWorkerValues(networkInferences.OneOutInfererValues),
		types.ConvertToWorkerValues(valueBundle.OneOutInfererValues),
	); err != nil {
		return err
	}

	if err := validateWorkerValues(
		types.ConvertToWorkerValues(networkInferences.OneInForecasterValues),
		types.ConvertToWorkerValues(valueBundle.OneInForecasterValues),
	); err != nil {
		return err
	}

	if err := validateWorkerValues(
		types.ConvertToWorkerValues(networkInferences.OneOutForecasterValues),
		types.ConvertToWorkerValues(valueBundle.OneOutForecasterValues)); err != nil {
		return err
	}

	if err := validateWorkerValues(
		types.ConvertToWorkerValues(networkInferences.OneOutInfererForecasterValues),
		types.ConvertToWorkerValues(valueBundle.OneOutInfererForecasterValues)); err != nil {
		return err
	}

	return nil
}
func validateWorkerValues(workerValues, inputWorkerValues []*types.WorkerValue) error {
	if len(workerValues) != len(inputWorkerValues) {
		return fmt.Errorf("worker sets don't match - different unique workers")
	}

	sort.Slice(workerValues, func(i, j int) bool {
		return workerValues[i].Worker < workerValues[j].Worker
	})

	sort.Slice(inputWorkerValues, func(i, j int) bool {
		return inputWorkerValues[i].Worker < inputWorkerValues[j].Worker
	})

	for i := range workerValues {
		if workerValues[i].Worker != inputWorkerValues[i].Worker {
			return fmt.Errorf("worker mismatch: expected %s, got %s",
				workerValues[i].Worker, inputWorkerValues[i].Worker)
		}

		valList, ok := workerValues[i].Value.([]*types.WorkerValue)
		if !ok {
			continue
		}

		inValList, ok := inputWorkerValues[i].Value.([]*types.WorkerValue)
		if !ok {
			continue
		}

		if err := validateWorkerValues(valList, inValList); err != nil {
			return err
		}
	}

	return nil
}
