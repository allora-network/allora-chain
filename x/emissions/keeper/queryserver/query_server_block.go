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
func (qs queryServer) GetStartingEmissionsBlockHeight(ctx context.Context, req *types.GetStartingEmissionsBlockHeightRequest) (_ *types.GetStartingEmissionsBlockHeightResponse, err error) {
	defer metrics.RecordMetrics("GetStartingEmissionsBlockHeight", time.Now(), &err)

	startingEmissionsBlockHeight, err := qs.k.GetStartingEmissionsBlockHeight(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.GetStartingEmissionsBlockHeightResponse{StartingEmissionsBlockHeight: startingEmissionsBlockHeight}, nil
}
