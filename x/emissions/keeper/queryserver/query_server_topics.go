package queryserver

import (
	"context"
	"strconv"
	"time"

	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// NextTopicId is a monotonically increasing counter that is used to assign unique IDs to topics.
func (qs queryServer) GetNextTopicId(ctx context.Context, req *types.GetNextTopicIdRequest) (_ *types.GetNextTopicIdResponse, err error) {
	defer metrics.RecordMetrics("GetNextTopicId", time.Now(), &err, map[string]string{})
	nextTopicId, err := qs.k.GetNextTopicId(ctx)
	if err != nil {
		return nil, err
	}
	return &types.GetNextTopicIdResponse{NextTopicId: nextTopicId}, nil
}

// Topics defines the handler for the Get/Topics RPC method.
func (qs queryServer) GetTopic(ctx context.Context, req *types.GetTopicRequest) (_ *types.GetTopicResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetTopic", time.Now(), &err, labels)
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
		params.BlocksPerMonth,
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
func (qs queryServer) GetTopicLastWorkerCommitInfo(ctx context.Context, req *types.GetTopicLastWorkerCommitInfoRequest) (_ *types.GetTopicLastWorkerCommitInfoResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetTopicLastWorkerCommitInfo", time.Now(), &err, labels)
	lastCommit, err := qs.k.GetWorkerTopicLastCommit(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetTopicLastWorkerCommitInfoResponse{LastCommit: &lastCommit}, nil
}

// Return last payload timestamp & nonce by worker/reputer
func (qs queryServer) GetTopicLastReputerCommitInfo(ctx context.Context, req *types.GetTopicLastReputerCommitInfoRequest) (_ *types.GetTopicLastReputerCommitInfoResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetTopicLastReputerCommitInfo", time.Now(), &err, labels)
	lastCommit, err := qs.k.GetReputerTopicLastCommit(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetTopicLastReputerCommitInfoResponse{LastCommit: &lastCommit}, nil
}

func (qs queryServer) GetTopicRewardNonce(ctx context.Context, req *types.GetTopicRewardNonceRequest) (_ *types.GetTopicRewardNonceResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetTopicRewardNonce", time.Now(), &err, labels)
	nonce, err := qs.k.GetTopicRewardNonce(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetTopicRewardNonceResponse{Nonce: nonce}, nil
}

func (qs queryServer) GetPreviousTopicWeight(ctx context.Context, req *types.GetPreviousTopicWeightRequest) (_ *types.GetPreviousTopicWeightResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetPreviousTopicWeight", time.Now(), &err, labels)
	previousTopicWeight, notFound, err := qs.k.GetPreviousTopicWeight(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetPreviousTopicWeightResponse{Weight: previousTopicWeight, NotFound: notFound}, nil
}

func (qs queryServer) GetTotalSumPreviousTopicWeights(ctx context.Context, req *types.GetTotalSumPreviousTopicWeightsRequest) (_ *types.GetTotalSumPreviousTopicWeightsResponse, err error) {
	defer metrics.RecordMetrics("GetTotalSumPreviousTopicWeights", time.Now(), &err, map[string]string{})
	previousTopicWeight, err := qs.k.GetTotalSumPreviousTopicWeights(ctx)
	if err != nil {
		return nil, err
	}

	return &types.GetTotalSumPreviousTopicWeightsResponse{Weight: previousTopicWeight}, nil
}

func (qs queryServer) TopicExists(ctx context.Context, req *types.TopicExistsRequest) (_ *types.TopicExistsResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("TopicExists", time.Now(), &err, labels)
	exists, err := qs.k.TopicExists(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.TopicExistsResponse{Exists: exists}, nil
}

func (qs queryServer) IsTopicActive(ctx context.Context, req *types.IsTopicActiveRequest) (_ *types.IsTopicActiveResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("IsTopicActive", time.Now(), &err, labels)
	isActive, err := qs.k.IsTopicActive(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.IsTopicActiveResponse{IsActive: isActive}, nil
}

func (qs queryServer) GetTopicFeeRevenue(ctx context.Context, req *types.GetTopicFeeRevenueRequest) (_ *types.GetTopicFeeRevenueResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetTopicFeeRevenue", time.Now(), &err, labels)
	feeRevenue, err := qs.k.GetTopicFeeRevenue(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetTopicFeeRevenueResponse{FeeRevenue: feeRevenue}, nil
}

func (qs queryServer) GetActiveTopicsAtBlock(ctx context.Context, req *types.GetActiveTopicsAtBlockRequest) (_ *types.GetActiveTopicsAtBlockResponse, err error) {
	labels := map[string]string{
		"block_height": strconv.FormatInt(req.BlockHeight, 10),
	}
	defer metrics.RecordMetrics("GetActiveTopicsAtBlock", time.Now(), &err, labels)
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
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetNextChurningBlockByTopicId", time.Now(), &err, labels)
	blockHeight, _, err := qs.k.GetNextPossibleChurningBlockByTopicId(ctx, req.TopicId)
	if err != nil {
		return &types.GetNextChurningBlockByTopicIdResponse{BlockHeight: 0}, err
	}
	return &types.GetNextChurningBlockByTopicIdResponse{BlockHeight: blockHeight}, nil
}
