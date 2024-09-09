package queryserver

import (
	"context"

	"cosmossdk.io/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// NextTopicId is a monotonically increasing counter that is used to assign unique IDs to topics.
func (qs queryServer) GetNextTopicId(ctx context.Context, req *types.QueryNextTopicIdRequest) (*types.QueryNextTopicIdResponse, error) {
	nextTopicId, err := qs.k.GetNextTopicId(ctx)
	if err != nil {
		return nil, err
	}
	return &types.QueryNextTopicIdResponse{NextTopicId: nextTopicId}, nil
}

// Topics defines the handler for the Query/Topics RPC method.
func (qs queryServer) GetTopic(ctx context.Context, req *types.QueryTopicRequest) (*types.QueryTopicResponse, error) {
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

	return &types.QueryTopicResponse{
		Topic:            &topic,
		Weight:           currentTopicWeight.String(),
		EffectiveRevenue: currentTopicRevenue.String(),
	}, nil
}

// Return last payload timestamp & nonce by worker/reputer
func (qs queryServer) GetTopicLastWorkerCommitInfo(ctx context.Context, req *types.QueryTopicLastWorkerCommitInfoRequest) (*types.QueryTopicLastWorkerCommitInfoResponse, error) {
	lastCommit, err := qs.k.GetWorkerTopicLastCommit(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTopicLastWorkerCommitInfoResponse{LastCommit: &lastCommit}, nil
}

// Return last payload timestamp & nonce by worker/reputer
func (qs queryServer) GetTopicLastReputerCommitInfo(ctx context.Context, req *types.QueryTopicLastReputerCommitInfoRequest) (*types.QueryTopicLastReputerCommitInfoResponse, error) {
	lastCommit, err := qs.k.GetReputerTopicLastCommit(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTopicLastReputerCommitInfoResponse{LastCommit: &lastCommit}, nil
}

func (qs queryServer) GetTopicRewardNonce(
	ctx context.Context,
	req *types.QueryTopicRewardNonceRequest,
) (
	*types.QueryTopicRewardNonceResponse,
	error,
) {
	nonce, err := qs.k.GetTopicRewardNonce(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTopicRewardNonceResponse{Nonce: nonce}, nil
}

func (qs queryServer) GetPreviousTopicWeight(
	ctx context.Context,
	req *types.QueryPreviousTopicWeightRequest,
) (
	*types.QueryPreviousTopicWeightResponse,
	error,
) {
	previousTopicWeight, notFound, err := qs.k.GetPreviousTopicWeight(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.QueryPreviousTopicWeightResponse{Weight: previousTopicWeight, NotFound: notFound}, nil
}

func (qs queryServer) TopicExists(
	ctx context.Context,
	req *types.QueryTopicExistsRequest,
) (
	*types.QueryTopicExistsResponse,
	error,
) {
	exists, err := qs.k.TopicExists(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.QueryTopicExistsResponse{Exists: exists}, nil
}

func (qs queryServer) IsTopicActive(
	ctx context.Context,
	req *types.QueryIsTopicActiveRequest,
) (
	*types.QueryIsTopicActiveResponse,
	error,
) {
	isActive, err := qs.k.IsTopicActive(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.QueryIsTopicActiveResponse{IsActive: isActive}, nil
}

func (qs queryServer) GetTopicFeeRevenue(
	ctx context.Context,
	req *types.QueryTopicFeeRevenueRequest,
) (
	*types.QueryTopicFeeRevenueResponse,
	error,
) {
	feeRevenue, err := qs.k.GetTopicFeeRevenue(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.QueryTopicFeeRevenueResponse{FeeRevenue: feeRevenue}, nil
}

func (qs queryServer) GetActiveTopicsAtBlock(
	ctx context.Context,
	req *types.QueryActiveTopicsAtBlockRequest,
) (*types.QueryActiveTopicsAtBlockResponse, error) {
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

	return &types.QueryActiveTopicsAtBlockResponse{Topics: topics}, nil
}

func (qs queryServer) GetNextChurningBlockByTopicId(
	ctx context.Context,
	req *types.QueryNextChurningBlockByTopicIdRequest,
) (*types.QueryNextChurningBlockByTopicIdResponse, error) {
	blockHeight, _, err := qs.k.GetNextPossibleChurningBlockByTopicId(ctx, req.TopicId)
	if err != nil {
		return &types.QueryNextChurningBlockByTopicIdResponse{BlockHeight: 0}, err
	}
	return &types.QueryNextChurningBlockByTopicIdResponse{BlockHeight: blockHeight}, nil
}
