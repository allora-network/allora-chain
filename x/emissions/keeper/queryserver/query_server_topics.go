package queryserver

import (
	"context"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
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
		cosmosMath.ZeroInt(),
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

// Retrieves a list of active topics. Paginated.
func (qs queryServer) GetActiveTopics(ctx context.Context, req *types.QueryActiveTopicsRequest) (*types.QueryActiveTopicsResponse, error) {
	activeTopics, pageRes, err := qs.k.GetIdsOfActiveTopics(ctx, req.Pagination)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	topics := make([]*types.Topic, 0)
	for _, topicId := range activeTopics {
		topic, err := qs.k.GetTopic(ctx, topicId)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		topics = append(topics, &topic)
	}

	return &types.QueryActiveTopicsResponse{Topics: topics, Pagination: pageRes}, nil
}

// Return last payload timestamp & nonce by worker/reputer
func (qs queryServer) GetTopicLastWorkerCommitInfo(ctx context.Context, req *types.QueryTopicLastCommitRequest) (*types.QueryTopicLastCommitResponse, error) {
	lastCommit, err := qs.k.GetTopicLastCommit(ctx, req.TopicId, types.ActorType_INFERER)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTopicLastCommitResponse{LastCommit: &lastCommit}, nil
}

// Return last payload timestamp & nonce by worker/reputer
func (qs queryServer) GetTopicLastReputerCommitInfo(ctx context.Context, req *types.QueryTopicLastCommitRequest) (*types.QueryTopicLastCommitResponse, error) {
	lastCommit, err := qs.k.GetTopicLastCommit(ctx, req.TopicId, types.ActorType_REPUTER)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTopicLastCommitResponse{LastCommit: &lastCommit}, nil
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
