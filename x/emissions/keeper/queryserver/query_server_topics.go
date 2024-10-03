package queryserver

import (
	"context"
	"time"

	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// NextTopicId is a monotonically increasing counter that is used to assign unique IDs to topics.
func (qs queryServer) GetNextTopicId(ctx context.Context, req *types.GetNextTopicIdRequest,
) (
	_ *types.GetNextTopicIdResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetNextTopicId", "rpc", time.Now(), returnErr == nil)
	nextTopicId, err := qs.k.GetNextTopicId(ctx)
	if err != nil {
		return nil, err
	}
	return &types.GetNextTopicIdResponse{NextTopicId: nextTopicId}, nil
}

// Topics defines the handler for the Get/Topics RPC method.
func (qs queryServer) GetTopic(ctx context.Context, req *types.GetTopicRequest,
) (
	_ *types.GetTopicResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetTopic", "rpc", time.Now(), returnErr == nil)
	topic, err := qs.k.GetTopic(ctx, req.TopicId)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting topic")
	}

	params, err := qs.k.GetParams(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting params")
	}

	currentTopicWeight, currentTopicRevenue, err := qs.k.GetCurrentTopicWeight(
		ctx,
		req.TopicId,
		topic.EpochLength,
		params.TopicRewardAlpha,
		params.TopicRewardStakeImportance,
		params.TopicRewardFeeRevenueImportance,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting current topic weight")
	}

	return &types.GetTopicResponse{
		Topic:            &topic,
		Weight:           currentTopicWeight.String(),
		EffectiveRevenue: currentTopicRevenue.String(),
	}, nil
}

// Return last payload timestamp & nonce by worker/reputer
func (qs queryServer) GetTopicLastWorkerCommitInfo(ctx context.Context, req *types.GetTopicLastWorkerCommitInfoRequest,
) (
	_ *types.GetTopicLastWorkerCommitInfoResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetTopicLastWorkerCommitInfo", "rpc", time.Now(), returnErr == nil)
	lastCommit, err := qs.k.GetWorkerTopicLastCommit(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetTopicLastWorkerCommitInfoResponse{LastCommit: &lastCommit}, nil
}

// Return last payload timestamp & nonce by worker/reputer
func (qs queryServer) GetTopicLastReputerCommitInfo(ctx context.Context, req *types.GetTopicLastReputerCommitInfoRequest,
) (
	_ *types.GetTopicLastReputerCommitInfoResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetTopicLastReputerCommitInfo", "rpc", time.Now(), returnErr == nil)
	lastCommit, err := qs.k.GetReputerTopicLastCommit(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetTopicLastReputerCommitInfoResponse{LastCommit: &lastCommit}, nil
}

func (qs queryServer) GetTopicRewardNonce(
	ctx context.Context,
	req *types.GetTopicRewardNonceRequest,
) (
	_ *types.GetTopicRewardNonceResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetTopicRewardNonce", "rpc", time.Now(), returnErr == nil)
	nonce, err := qs.k.GetTopicRewardNonce(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetTopicRewardNonceResponse{Nonce: nonce}, nil
}

func (qs queryServer) GetPreviousTopicWeight(
	ctx context.Context,
	req *types.GetPreviousTopicWeightRequest,
) (
	_ *types.GetPreviousTopicWeightResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetPreviousTopicWeight", "rpc", time.Now(), returnErr == nil)
	previousTopicWeight, notFound, err := qs.k.GetPreviousTopicWeight(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetPreviousTopicWeightResponse{Weight: previousTopicWeight, NotFound: notFound}, nil
}

func (qs queryServer) TopicExists(
	ctx context.Context,
	req *types.TopicExistsRequest,
) (
	_ *types.TopicExistsResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("TopicExists", "rpc", time.Now(), returnErr == nil)
	exists, err := qs.k.TopicExists(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.TopicExistsResponse{Exists: exists}, nil
}

func (qs queryServer) IsTopicActive(
	ctx context.Context,
	req *types.IsTopicActiveRequest,
) (
	_ *types.IsTopicActiveResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("IsTopicActive", "rpc", time.Now(), returnErr == nil)
	isActive, err := qs.k.IsTopicActive(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.IsTopicActiveResponse{IsActive: isActive}, nil
}

func (qs queryServer) GetTopicFeeRevenue(
	ctx context.Context,
	req *types.GetTopicFeeRevenueRequest,
) (
	_ *types.GetTopicFeeRevenueResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetTopicFeeRevenue", "rpc", time.Now(), returnErr == nil)
	feeRevenue, err := qs.k.GetTopicFeeRevenue(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetTopicFeeRevenueResponse{FeeRevenue: feeRevenue}, nil
}

func (qs queryServer) GetActiveTopicsAtBlock(
	ctx context.Context,
	req *types.GetActiveTopicsAtBlockRequest,
) (_ *types.GetActiveTopicsAtBlockResponse, returnErr error) {
	defer metrics.RecordMetrics("GetActiveTopicsAtBlock", "rpc", time.Now(), returnErr == nil)
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

func (qs queryServer) GetNextChurningBlockByTopicId(
	ctx context.Context,
	req *types.GetNextChurningBlockByTopicIdRequest,
) (_ *types.GetNextChurningBlockByTopicIdResponse, returnErr error) {
	defer metrics.RecordMetrics("GetNextChurningBlockByTopicId", "rpc", time.Now(), returnErr == nil)
	blockHeight, _, err := qs.k.GetNextPossibleChurningBlockByTopicId(ctx, req.TopicId)
	if err != nil {
		return &types.GetNextChurningBlockByTopicIdResponse{BlockHeight: 0}, err
	}
	return &types.GetNextChurningBlockByTopicIdResponse{BlockHeight: blockHeight}, nil
}
