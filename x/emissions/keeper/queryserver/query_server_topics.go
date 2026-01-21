package queryserver

import (
	"context"
	"time"

	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// NextTopicId is a monotonically increasing counter that is used to assign unique IDs to topics.
func (qs queryServer) GetNextTopicId(ctx context.Context, req *types.GetNextTopicIdRequest) (_ *types.GetNextTopicIdResponse, err error) {
	defer metrics.RecordMetrics("GetNextTopicId", time.Now(), &err)
	nextTopicId, err := qs.k.GetNextTopicId(ctx)
	if err != nil {
		return nil, err
	}
	return &types.GetNextTopicIdResponse{NextTopicId: nextTopicId}, nil
}

// Topics defines the handler for the Get/Topics RPC method.
func (qs queryServer) GetTopic(ctx context.Context, req *types.GetTopicRequest) (_ *types.GetTopicResponse, err error) {
	defer metrics.RecordMetrics("GetTopic", time.Now(), &err)
	topic, err := qs.k.GetTopic(ctx, req.TopicId)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting topic")
	}

	params, err := qs.k.GetParams(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting params")
	}

	currentTopicWeight, currentTopicRevenue, currentTopicStake, err := qs.k.GetCurrentTopicWeight(
		ctx,
		req.TopicId,
		topic.EpochLength,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
		params.BlocksPerMonth,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting current topic weight")
	}

	return &types.GetTopicResponse{
		Topic:            &topic,
		Weight:           currentTopicWeight.String(),
		EffectiveRevenue: currentTopicRevenue.String(),
		TopicStake:       currentTopicStake.String(),
	}, nil
}

// Return last payload timestamp & nonce by worker/reputer
func (qs queryServer) GetTopicLastWorkerCommitInfo(ctx context.Context, req *types.GetTopicLastWorkerCommitInfoRequest) (_ *types.GetTopicLastWorkerCommitInfoResponse, err error) {
	defer metrics.RecordMetrics("GetTopicLastWorkerCommitInfo", time.Now(), &err)
	lastCommit, err := qs.k.GetWorkerTopicLastCommit(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetTopicLastWorkerCommitInfoResponse{LastCommit: &lastCommit}, nil
}

// Return last payload timestamp & nonce by worker/reputer
func (qs queryServer) GetTopicLastReputerCommitInfo(ctx context.Context, req *types.GetTopicLastReputerCommitInfoRequest) (_ *types.GetTopicLastReputerCommitInfoResponse, err error) {
	defer metrics.RecordMetrics("GetTopicLastReputerCommitInfo", time.Now(), &err)
	lastCommit, err := qs.k.GetReputerTopicLastCommit(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetTopicLastReputerCommitInfoResponse{LastCommit: &lastCommit}, nil
}

func (qs queryServer) GetTopicRewardNonce(ctx context.Context, req *types.GetTopicRewardNonceRequest) (_ *types.GetTopicRewardNonceResponse, err error) {
	defer metrics.RecordMetrics("GetTopicRewardNonce", time.Now(), &err)
	nonce, err := qs.k.GetTopicRewardNonce(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetTopicRewardNonceResponse{Nonce: nonce}, nil
}

func (qs queryServer) GetPreviousTopicWeight(ctx context.Context, req *types.GetPreviousTopicWeightRequest) (_ *types.GetPreviousTopicWeightResponse, err error) {
	defer metrics.RecordMetrics("GetPreviousTopicWeight", time.Now(), &err)
	previousTopicWeight, notFound, err := qs.k.GetPreviousTopicWeight(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetPreviousTopicWeightResponse{Weight: previousTopicWeight, NotFound: notFound}, nil
}

func (qs queryServer) GetTotalSumPreviousTopicWeights(ctx context.Context, req *types.GetTotalSumPreviousTopicWeightsRequest) (_ *types.GetTotalSumPreviousTopicWeightsResponse, err error) {
	defer metrics.RecordMetrics("GetTotalSumPreviousTopicWeights", time.Now(), &err)
	previousTopicWeight, err := qs.k.GetTotalSumPreviousTopicWeights(ctx)
	if err != nil {
		return nil, err
	}

	return &types.GetTotalSumPreviousTopicWeightsResponse{Weight: previousTopicWeight}, nil
}

func (qs queryServer) TopicExists(ctx context.Context, req *types.TopicExistsRequest) (_ *types.TopicExistsResponse, err error) {
	defer metrics.RecordMetrics("TopicExists", time.Now(), &err)
	exists, err := qs.k.TopicExists(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.TopicExistsResponse{Exists: exists}, nil
}

func (qs queryServer) IsTopicActive(ctx context.Context, req *types.IsTopicActiveRequest) (_ *types.IsTopicActiveResponse, err error) {
	defer metrics.RecordMetrics("IsTopicActive", time.Now(), &err)
	isActive, err := qs.k.IsTopicActive(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.IsTopicActiveResponse{IsActive: isActive}, nil
}

func (qs queryServer) GetTopicFeeRevenue(ctx context.Context, req *types.GetTopicFeeRevenueRequest) (_ *types.GetTopicFeeRevenueResponse, err error) {
	defer metrics.RecordMetrics("GetTopicFeeRevenue", time.Now(), &err)
	feeRevenue, err := qs.k.GetTopicFeeRevenue(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetTopicFeeRevenueResponse{FeeRevenue: feeRevenue}, nil
}

func (qs queryServer) GetActiveTopicsAtBlock(ctx context.Context, req *types.GetActiveTopicsAtBlockRequest) (_ *types.GetActiveTopicsAtBlockResponse, err error) {
	defer metrics.RecordMetrics("GetActiveTopicsAtBlock", time.Now(), &err)
	activeTopicIds, err := qs.k.GetActiveTopicIdsAtBlock(ctx, req.BlockHeight)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	topics := make([]*types.Topic, 0)
	for _, topicId := range activeTopicIds.TopicIds {
		topic, err := qs.k.GetTopic(ctx, topicId)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		topics = append(topics, &topic)
	}

	return &types.GetActiveTopicsAtBlockResponse{Topics: topics, Pagination: nil}, nil
}

func (qs queryServer) GetNextChurningBlockByTopicId(ctx context.Context, req *types.GetNextChurningBlockByTopicIdRequest) (_ *types.GetNextChurningBlockByTopicIdResponse, err error) {
	defer metrics.RecordMetrics("GetNextChurningBlockByTopicId", time.Now(), &err)
	blockHeight, _, err := qs.k.GetNextPossibleChurningBlockByTopicId(ctx, req.TopicId)
	if err != nil {
		return &types.GetNextChurningBlockByTopicIdResponse{BlockHeight: 0}, err
	}
	return &types.GetNextChurningBlockByTopicIdResponse{BlockHeight: blockHeight}, nil
}

func (qs queryServer) GetWorkerSubmissionWindowStatus(ctx context.Context, req *types.GetWorkerSubmissionWindowStatusRequest) (_ *types.GetWorkerSubmissionWindowStatusResponse, err error) {
	defer metrics.RecordMetrics("GetWorkerSubmissionWindowStatus", time.Now(), &err)

	// Get current block height
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()

	// Get topic
	topic, err := qs.k.GetTopic(ctx, req.TopicId)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting topic")
	}

	// Get unfulfilled worker nonces to check current window
	nonces, err := qs.k.GetUnfulfilledWorkerNonces(ctx, req.TopicId)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting unfulfilled worker nonces")
	}

	response := &types.GetWorkerSubmissionWindowStatusResponse{
		IsOpen:                  false,
		CurrentNonce: 0,
		WindowStartBlock:        0,
		WindowEndBlock:          0,
		NextWindowStartBlock:    0,
		NextWindowEndBlock:      0,
		IsRegistered:            false,
		IsWhitelisted:           false,
		IsTopicActive:           false,
	}

	// Check registration and whitelist status if address is provided
	if req.Address != "" {
		// Validate the address
		if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
			return nil, errors.Wrapf(err, "invalid address format")
		}

		// Check if worker is registered in the topic
		isRegistered, err := qs.k.IsWorkerRegisteredInTopic(ctx, req.TopicId, req.Address)
		if err != nil {
			return nil, errors.Wrapf(err, "error checking worker registration")
		}
		response.IsRegistered = isRegistered

		// Check if worker can submit payload (whitelist check)
		isWhitelisted, err := qs.k.CanSubmitWorkerPayload(ctx, req.TopicId, req.Address)
		if err != nil {
			return nil, errors.Wrapf(err, "error checking worker whitelist status")
		}
		response.IsWhitelisted = isWhitelisted
	}

	// Check if there's an active worker window
	for _, nonce := range nonces.Nonces {
		windowStart := nonce.BlockHeight
		windowEnd := nonce.BlockHeight + topic.WorkerSubmissionWindow

		if currentBlockHeight >= windowStart && currentBlockHeight <= windowEnd {
			response.IsOpen = true
			response.CurrentNonce = nonce.BlockHeight
			response.WindowStartBlock = windowStart
			response.WindowEndBlock = windowEnd
			break
		}
	}

	// Calculate next window timing based on topic epoch
	nextChurningBlock, isActive, err := qs.k.GetNextPossibleChurningBlockByTopicId(ctx, req.TopicId)
	if err != nil {
		return nil, errors.Wrapf(err, "error checking topic active status")
	}
	if isActive {
		response.NextWindowStartBlock = nextChurningBlock
		response.NextWindowEndBlock = nextChurningBlock + topic.WorkerSubmissionWindow
		response.IsTopicActive = isActive
	}

	return response, nil
}

func (qs queryServer) GetReputerSubmissionWindowStatus(ctx context.Context, req *types.GetReputerSubmissionWindowStatusRequest) (_ *types.GetReputerSubmissionWindowStatusResponse, err error) {
	defer metrics.RecordMetrics("GetReputerSubmissionWindowStatus", time.Now(), &err)

	// Get current block height
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlockHeight := sdkCtx.BlockHeight()

	// Get topic
	topic, err := qs.k.GetTopic(ctx, req.TopicId)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting topic")
	}

	// Get unfulfilled reputer nonces to check current window
	nonces, err := qs.k.GetUnfulfilledReputerNonces(ctx, req.TopicId)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting unfulfilled reputer nonces")
	}

	response := &types.GetReputerSubmissionWindowStatusResponse{
		IsOpen:                  false,
		CurrentNonce: 0,
		WindowStartBlock:        0,
		WindowEndBlock:          0,
		NextWindowStartBlock:    0,
		NextWindowEndBlock:      0,
		IsRegistered:            false,
		IsWhitelisted:           false,
		IsTopicActive:           false,
	}

	// Check registration and whitelist status if address is provided
	if req.Address != "" {
		// Validate the address
		if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
			return nil, errors.Wrapf(err, "invalid address format")
		}

		// Check if reputer is registered in the topic
		isRegistered, err := qs.k.IsReputerRegisteredInTopic(ctx, req.TopicId, req.Address)
		if err != nil {
			return nil, errors.Wrapf(err, "error checking reputer registration")
		}
		response.IsRegistered = isRegistered

		// Check if reputer can submit payload (whitelist check)
		isWhitelisted, err := qs.k.CanSubmitReputerPayload(ctx, req.TopicId, req.Address)
		if err != nil {
			return nil, errors.Wrapf(err, "error checking reputer whitelist status")
		}
		response.IsWhitelisted = isWhitelisted
	}

	var latestActiveNonce *types.ReputerRequestNonce
	var earliestFutureNonce *types.ReputerRequestNonce
	extraLag := topic.GroundTruthLag % topic.EpochLength
	if extraLag != 0 {
		extraLag = topic.EpochLength - extraLag
	}

	for _, nonce := range nonces.Nonces {
		windowStart := nonce.ReputerNonce.BlockHeight + topic.GroundTruthLag
		windowEnd := windowStart + extraLag + topic.EpochLength

		if currentBlockHeight >= windowStart && currentBlockHeight <= windowEnd {
			// Current active window: find the most recent active nonce
			if latestActiveNonce == nil || nonce.ReputerNonce.BlockHeight > latestActiveNonce.ReputerNonce.BlockHeight {
				latestActiveNonce = nonce
				response.IsOpen = true
				response.CurrentNonce = nonce.ReputerNonce.BlockHeight
				response.WindowStartBlock = windowStart
				response.WindowEndBlock = windowEnd
			}
		} else if windowStart > currentBlockHeight {
			// Next future window: find the earliest future nonce
			if earliestFutureNonce == nil || nonce.ReputerNonce.BlockHeight < earliestFutureNonce.ReputerNonce.BlockHeight {
				earliestFutureNonce = nonce
			}
		}

		// Early exit: if we found both current and next nonce
		if latestActiveNonce != nil && earliestFutureNonce != nil {
			break
		}
	}

	// Calculate next window from the earliest future reputer nonce found
	if earliestFutureNonce != nil {
		nextReputerStart := earliestFutureNonce.ReputerNonce.BlockHeight + topic.GroundTruthLag
		nextReputerEnd := nextReputerStart + extraLag + topic.EpochLength
		response.NextWindowStartBlock = nextReputerStart
		response.NextWindowEndBlock = nextReputerEnd
	}

	// Get topic active status from GetNextPossibleChurningBlockByTopicId
	_, isActive, err := qs.k.GetNextPossibleChurningBlockByTopicId(ctx, req.TopicId)
	if err != nil {
		return nil, errors.Wrapf(err, "error checking topic active status")
	}
	response.IsTopicActive = isActive

	return response, nil
}
