package queryserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetLatestNetworkValueBundle(ctx context.Context, req *types.QueryLatestNetworkValueBundleRequest) (*types.QueryLatestNetworkValueBundleResponse, error) {
	latestLoss, err := qs.k.GetLatestNetworkValueBundle(ctx, req.TopicId)
	if err != nil {
		return nil, err
	}

	return &types.QueryLatestNetworkValueBundleResponse{ValueBundle: latestLoss}, nil
}
