package queryserver

import (
	"context"
	"time"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetNetworkLossBundleAtBlock(ctx context.Context, req *types.GetNetworkLossBundleAtBlockRequest) (_ *types.GetNetworkLossBundleAtBlockResponse, err error) {
	defer metrics.RecordMetrics("GetNetworkLossBundleAtBlock", time.Now(), &err)

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	networkLoss, err := qs.k.GetNetworkLossBundleAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetNetworkLossBundleAtBlockResponse{LossBundle: networkLoss}, nil
}

func (qs queryServer) IsReputerNonceUnfulfilled(ctx context.Context, req *types.IsReputerNonceUnfulfilledRequest) (_ *types.IsReputerNonceUnfulfilledResponse, err error) {
	defer metrics.RecordMetrics("IsReputerNonceUnfulfilled", time.Now(), &err)

	isReputerNonceUnfulfilled, err := qs.k.IsReputerNonceUnfulfilled(ctx, req.TopicId, &types.Nonce{BlockHeight: req.BlockHeight})
	if err != nil {
		return nil, err
	}

	return &types.IsReputerNonceUnfulfilledResponse{IsReputerNonceUnfulfilled: isReputerNonceUnfulfilled}, nil
}

func (qs queryServer) GetUnfulfilledReputerNonces(ctx context.Context, req *types.GetUnfulfilledReputerNoncesRequest) (_ *types.GetUnfulfilledReputerNoncesResponse, err error) {
	defer metrics.RecordMetrics("GetUnfulfilledReputerNonces", time.Now(), &err)

	unfulfilledNonces, err := qs.k.GetUnfulfilledReputerNonces(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.GetUnfulfilledReputerNoncesResponse{Nonces: &unfulfilledNonces}, nil
}

func (qs queryServer) GetReputerLossBundlesAtBlock(ctx context.Context, req *types.GetReputerLossBundlesAtBlockRequest) (_ *types.GetReputerLossBundlesAtBlockResponse, err error) {
	defer metrics.RecordMetrics("GetReputerLossBundlesAtBlock", time.Now(), &err)

	reputerLossBundles, err := qs.k.GetReputerLossBundlesAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.GetReputerLossBundlesAtBlockResponse{LossBundles: reputerLossBundles}, nil
}

func (qs queryServer) GetActiveReputersForTopic(ctx context.Context, req *types.GetActiveReputersForTopicRequest) (_ *types.GetActiveReputersForTopicResponse, err error) {
	defer metrics.RecordMetrics("GetActiveReputersForTopic", time.Now(), &err)

	topicExists, err := qs.k.TopicExists(ctx, req.TopicId)
	if !topicExists {
		return nil, status.Errorf(codes.NotFound, "topic %v not found", req.TopicId)
	} else if err != nil {
		return nil, err
	}

	reputers, err := qs.k.GetActiveReputersForTopic(ctx, req.TopicId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetActiveReputersForTopicResponse{Reputers: reputers}, nil
}
