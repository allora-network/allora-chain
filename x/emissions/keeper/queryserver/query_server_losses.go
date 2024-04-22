package queryserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetNetworkLossBundleAtBlock(ctx context.Context, req *types.QueryNetworkLossBundleAtBlockRequest) (*types.QueryNetworkLossBundleAtBlockResponse, error) {
	if req == nil {
		return nil, types.ErrReceivedNilRequest
	}

	networkLoss, err := qs.k.GetNetworkLossBundleAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.QueryNetworkLossBundleAtBlockResponse{LossBundle: networkLoss}, nil
}
