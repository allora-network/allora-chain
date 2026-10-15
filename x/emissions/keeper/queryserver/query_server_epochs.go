package queryserver

import (
	"context"
	"errors"
	"time"

	"cosmossdk.io/collections"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetEpoch(ctx context.Context, req *types.GetEpochRequest) (_ *types.GetEpochResponse, err error) {
	defer metrics.RecordMetrics("GetEpoch", time.Now(), &err)

	epoch, err := qs.k.GetEpoch(ctx, req.TopicId, types.NonceV2(req.Nonce))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "epoch not found for topic %d nonce %d", req.TopicId, req.Nonce)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.GetEpochResponse{Epoch: &epoch}, nil
}

func (qs queryServer) GetTopicEpochs(ctx context.Context, req *types.GetTopicEpochsRequest) (_ *types.GetTopicEpochsResponse, err error) {
	defer metrics.RecordMetrics("GetTopicEpochs", time.Now(), &err)

	epochs, err := qs.k.GetTopicEpochs(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	out := make([]*types.Epoch, 0, len(epochs))
	for i := range epochs {
		out = append(out, &epochs[i])
	}
	return &types.GetTopicEpochsResponse{Epochs: out}, nil
}
