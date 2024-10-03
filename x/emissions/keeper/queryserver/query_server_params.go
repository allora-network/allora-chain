package queryserver

import (
	"context"
	"time"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// Params defines the handler for the Query/Params RPC method.
func (qs queryServer) GetParams(ctx context.Context, req *types.GetParamsRequest,
) (
	_ *types.GetParamsResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("GetParams", "rpc", time.Now(), returnErr == nil)
	params, err := qs.k.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetParamsResponse{Params: params}, nil
}
