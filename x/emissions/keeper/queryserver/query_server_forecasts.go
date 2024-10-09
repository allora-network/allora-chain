package queryserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetForecastsAtBlock(ctx context.Context, req *types.GetForecastsAtBlockRequest) (*types.GetForecastsAtBlockResponse, error) {
	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	forecasts, err := qs.k.GetForecastsAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetForecastsAtBlockResponse{Forecasts: forecasts}, nil
}

func (qs queryServer) GetActiveForecastersForTopic(ctx context.Context, req *types.GetActiveForecastersForTopicRequest) (*types.GetActiveForecastersForTopicResponse, error) {
	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	forecasters, err := qs.k.GetActiveForecastersForTopic(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetActiveForecastersForTopicResponse{Forecasters: forecasters}, nil
}
