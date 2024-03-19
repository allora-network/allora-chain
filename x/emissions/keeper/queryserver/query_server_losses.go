package queryserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetLatestNetworkLossBundle(ctx context.Context, req *types.QueryLatestNetworkLossBundleRequest) (*types.QueryLatestNetworkLossBundleResponse, error) {
	latestLoss, err := qs.k.GetLatestNetworkLossBundle(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.QueryLatestNetworkLossBundleResponse{LossBundle: latestLoss}, nil
}
