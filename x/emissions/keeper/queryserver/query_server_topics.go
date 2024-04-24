package queryserver

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
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
		if errors.Is(err, collections.ErrNotFound) {
			return &types.QueryTopicResponse{Topic: &topic}, nil
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTopicResponse{Topic: &topic}, nil
}

// Retrieves a list of active topics. Paginated.
func (qs queryServer) GetActiveTopics(ctx context.Context, req *types.QueryActiveTopicsRequest) (*types.QueryActiveTopicsResponse, error) {
	activeTopics, pageRes, err := qs.k.GetActiveTopics(ctx, req.Pagination)
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

func (qs queryServer) GetTopicUnmetDemand(ctx context.Context, req *types.QueryTopicUnmetDemandRequest) (*types.QueryTopicUnmetDemandResponse, error) {
	unmetDemand, err := qs.k.GetTopicUnmetDemand(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	return &types.QueryTopicUnmetDemandResponse{DemandLeft: unmetDemand}, nil
}
