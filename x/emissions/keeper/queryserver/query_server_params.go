package queryserver

import (
	"context"
	"time"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// GetParams defines the handler for the Query/Params RPC method.
func (qs queryServer) GetParams(ctx context.Context, req *types.GetParamsRequest) (_ *types.GetParamsResponse, err error) {
	defer metrics.RecordMetrics("GetParams", time.Now(), func() bool { return err == nil })

	params, err := qs.k.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.GetParamsResponse{Params: params}, nil
}
