package queryserver

import (
	"context"

	"cosmossdk.io/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// NextTopicId is a monotonically increasing counter that is used to assign unique IDs to topics.
func (qs queryServer) GetNextTopicId(ctx context.Context, req *types.GetNextTopicIdRequest) (*types.GetNextTopicIdResponse, error) {
	nextTopicId, err := qs.k.GetNextTopicId(ctx)
	if err != nil {
		return nil, err
	}
	return &types.GetNextTopicIdResponse{NextTopicId: nextTopicId}, nil
}

// Topics defines the handler for the Get/Topics RPC method.
func (qs queryServer) GetTopic(ctx context.Context, req *types.GetTopicRequest) (*types.GetTopicResponse, error) {
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
func (qs queryServer) GetTopicLastWorkerCommitInfo(ctx context.Context, req *types.GetTopicLastWorkerCommitInfoRequest) (*types.GetTopicLastWorkerCommitInfoResponse, error) {
	lastCommit, err := qs.k.GetWorkerTopicLastCommit(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetTopicLastWorkerCommitInfoResponse{LastCommit: &lastCommit}, nil
}

// Return last payload timestamp & nonce by worker/reputer
func (qs queryServer) GetTopicLastReputerCommitInfo(ctx context.Context, req *types.GetTopicLastReputerCommitInfoRequest) (*types.GetTopicLastReputerCommitInfoResponse, error) {
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
	*types.GetTopicRewardNonceResponse,
	error,
) {
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
	*types.GetPreviousTopicWeightResponse,
	error,
) {
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
	*types.TopicExistsResponse,
	error,
) {
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
	*types.IsTopicActiveResponse,
	error,
) {
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
	*types.GetTopicFeeRevenueResponse,
	error,
) {
	feeRevenue, err := qs.k.GetTopicFeeRevenue(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetTopicFeeRevenueResponse{FeeRevenue: feeRevenue}, nil
}

func (qs queryServer) GetActiveTopicsAtBlock(
	ctx context.Context,
	req *types.GetActiveTopicsAtBlockRequest,
) (*types.GetActiveTopicsAtBlockResponse, error) {
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

	return &types.GetActiveTopicsAtBlockResponse{Topics: topics}, nil
}

func (qs queryServer) GetNextChurningBlockByTopicId(
	ctx context.Context,
	req *types.GetNextChurningBlockByTopicIdRequest,
) (*types.GetNextChurningBlockByTopicIdResponse, error) {
	blockHeight, _, err := qs.k.GetNextPossibleChurningBlockByTopicId(ctx, req.TopicId)
	if err != nil {
		return &types.GetNextChurningBlockByTopicIdResponse{BlockHeight: 0}, err
	}
	return &types.GetNextChurningBlockByTopicIdResponse{BlockHeight: blockHeight}, nil
}
