package queryserver

import (
	"context"
	"strconv"
	"time"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (qs queryServer) GetLatestRegretStdNorm(ctx context.Context, req *types.GetLatestRegretStdNormRequest) (_ *types.GetLatestRegretStdNormResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
	}
	defer metrics.RecordMetrics("GetLatestRegretStdNorm", time.Now(), &err, labels)

	if err := types.ValidateTopicId(req.TopicId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	stdNorm, err := qs.k.GetLatestRegretStdNorm(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetLatestRegretStdNormResponse{
		Value: stdNorm,
	}, nil
}

func (qs queryServer) GetLatestInfererWeight(ctx context.Context, req *types.GetLatestInfererWeightRequest) (_ *types.GetLatestInfererWeightResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
		"address":  req.ActorId,
	}
	defer metrics.RecordMetrics("GetLatestInfererWeight", time.Now(), &err, labels)

	if err := types.ValidateTopicId(req.TopicId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := types.ValidateBech32(req.ActorId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	weight, err := qs.k.GetLatestInfererWeight(ctx, req.TopicId, req.ActorId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetLatestInfererWeightResponse{
		Weight: weight,
	}, nil
}

func (qs queryServer) GetLatestForecasterWeight(ctx context.Context, req *types.GetLatestForecasterWeightRequest) (_ *types.GetLatestForecasterWeightResponse, err error) {
	labels := map[string]string{
		"topic_id": strconv.FormatUint(req.TopicId, 10),
		"address":  req.ActorId,
	}
	defer metrics.RecordMetrics("GetLatestForecasterWeight", time.Now(), &err, labels)

	if err := types.ValidateTopicId(req.TopicId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := types.ValidateBech32(req.ActorId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	weight, err := qs.k.GetLatestForecasterWeight(ctx, req.TopicId, req.ActorId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetLatestForecasterWeightResponse{
		Weight: weight,
	}, nil
}
