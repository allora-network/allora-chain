package queryserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	state "github.com/allora-network/allora-chain/x/emissions"
)

// Params defines the handler for the Query/Params RPC method.
func (qs queryServer) Params(ctx context.Context, req *state.QueryParamsRequest) (*state.QueryParamsResponse, error) {
	params, err := qs.k.GetParams(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &state.QueryParamsResponse{Params: params}, nil
}
