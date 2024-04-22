package queryserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (qs queryServer) GetForecastsAtBlock(ctx context.Context, req *types.QueryForecastsAtBlockRequest) (*types.QueryForecastsAtBlockResponse, error) {
	if req == nil {
		return nil, types.ErrReceivedNilRequest
	}

	forecasts, err := qs.k.GetForecastsAtBlock(ctx, req.TopicId, req.BlockHeight)
	if err != nil {
		return nil, err
	}

	return &types.QueryForecastsAtBlockResponse{Forecasts: forecasts}, nil
}
