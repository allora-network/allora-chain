package queryserver

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	state "github.com/allora-network/allora-chain/x/emissions"
)

// NextTopicId is a monotonically increasing counter that is used to assign unique IDs to topics.
func (qs queryServer) GetNextTopicId(ctx context.Context, req *state.QueryNextTopicIdRequest) (*state.QueryNextTopicIdResponse, error) {
	nextTopicId, err := qs.k.nextTopicId.Peek(ctx)
	if err != nil {
		return nil, err
	}
	return &state.QueryNextTopicIdResponse{NextTopicId: nextTopicId}, nil
}

// Topics defines the handler for the Query/Topics RPC method.
func (qs queryServer) GetTopic(ctx context.Context, req *state.QueryTopicRequest) (*state.QueryTopicResponse, error) {
	topic, err := qs.k.topics.Get(ctx, req.TopicId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return &state.QueryTopicResponse{Topic: &topic}, nil
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	return &state.QueryTopicResponse{Topic: &topic}, nil
}

// GetActiveTopics retrieves a list of active topics.
func (qs queryServer) GetActiveTopics(ctx context.Context, req *state.QueryActiveTopicsRequest) (*state.QueryActiveTopicsResponse, error) {
	activeTopics, err := qs.k.GetActiveTopics(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &state.QueryActiveTopicsResponse{Topics: activeTopics}, nil
}

func (qs queryServer) GetAllTopics(ctx context.Context, req *state.QueryAllTopicsRequest) (*state.QueryAllTopicsResponse, error) {
	topics, err := qs.k.GetAllTopics(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &state.QueryAllTopicsResponse{Topics: topics}, nil
}

// GetTopicsByCreator retrieves a list of topics created by a given address.
func (qs queryServer) GetTopicsByCreator(ctx context.Context, req *state.QueryGetTopicsByCreatorRequest) (*state.QueryGetTopicsByCreatorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	topics, err := qs.k.GetTopicsByCreator(ctx, req.Creator)
	if err != nil {
		return nil, err
	}

	return &state.QueryGetTopicsByCreatorResponse{Topics: topics}, nil
}

func (qs queryServer) GetTopicUnmetDemand(ctx context.Context, req *state.QueryTopicUnmetDemandRequest) (*state.QueryTopicUnmetDemandResponse, error) {
	unmetDemand, err := qs.k.GetTopicUnmetDemand(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}
	return &state.QueryTopicUnmetDemandResponse{DemandLeft: unmetDemand}, nil
}