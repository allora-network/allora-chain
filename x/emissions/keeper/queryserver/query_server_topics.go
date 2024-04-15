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
	nextTopicId, err := qs.k.GetNumTopics(ctx)
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

// TODO paginate
// GetActiveTopics retrieves a list of active topics.
func (qs queryServer) GetActiveTopics(ctx context.Context, req *types.QueryActiveTopicsRequest) (*types.QueryActiveTopicsResponse, error) {
	activeTopics, err := qs.k.GetActiveTopics(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryActiveTopicsResponse{Topics: activeTopics}, nil
}

// TODO paginate
func (qs queryServer) GetAllTopics(ctx context.Context, req *types.QueryAllTopicsRequest) (*types.QueryAllTopicsResponse, error) {
	topics, err := qs.k.GetAllTopics(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllTopicsResponse{Topics: topics}, nil
}

// TODO paginate
// GetTopicsByCreator retrieves a list of topics created by a given address.
func (qs queryServer) GetTopicsByCreator(ctx context.Context, req *types.QueryGetTopicsByCreatorRequest) (*types.QueryGetTopicsByCreatorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	topics, err := qs.k.GetTopicsByCreator(ctx, req.Creator)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetTopicsByCreatorResponse{Topics: topics}, nil
}

func (qs queryServer) GetTopicUnmetDemand(ctx context.Context, req *types.QueryTopicUnmetDemandRequest) (*types.QueryTopicUnmetDemandResponse, error) {
	unmetDemand, err := qs.k.GetTopicUnmetDemand(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	return &types.QueryTopicUnmetDemandResponse{DemandLeft: unmetDemand}, nil
}
