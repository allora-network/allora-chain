package queryserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetNetworkLossBundleAtBlock(ctx context.Context, req *types.QueryNetworkLossBundleAtBlockRequest) (*types.QueryNetworkLossBundleAtBlockResponse, error) {
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

	return &types.QueryNetworkLossBundleAtBlockResponse{LossBundle: networkLoss}, nil
}

func (qs queryServer) GetIsReputerNonceUnfulfilled(
	ctx context.Context,
	req *types.QueryIsReputerNonceUnfulfilledRequest,
) (
	*types.QueryIsReputerNonceUnfulfilledResponse,
	error,
) {
	isReputerNonceUnfulfilled, err :=
		qs.k.IsReputerNonceUnfulfilled(ctx, req.TopicId, &types.Nonce{BlockHeight: req.BlockHeight})

	return &types.QueryIsReputerNonceUnfulfilledResponse{IsReputerNonceUnfulfilled: isReputerNonceUnfulfilled}, err
}

func (qs queryServer) GetUnfulfilledReputerNonces(
	ctx context.Context,
	req *types.QueryUnfulfilledReputerNoncesRequest,
) (
	*types.QueryUnfulfilledReputerNoncesResponse,
	error,
) {
	unfulfilledNonces, err := qs.k.GetUnfulfilledReputerNonces(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.QueryUnfulfilledReputerNoncesResponse{Nonces: &unfulfilledNonces}, nil
}

func (qs queryServer) GetReputerLossBundlesAtBlock(
	ctx context.Context,
	req *types.QueryReputerLossBundlesAtBlockRequest,
) (
	*types.QueryReputerLossBundlesAtBlockResponse,
	error,
) {
	reputerLossBundles, err := qs.k.GetReputerLossBundlesAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.QueryReputerLossBundlesAtBlockResponse{LossBundles: reputerLossBundles}, nil
}
