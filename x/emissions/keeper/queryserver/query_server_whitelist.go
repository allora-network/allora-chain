package queryserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Params defines the handler for the Query/Params RPC method.
func (qs queryServer) IsWhitelistAdmin(ctx context.Context, req *types.QueryIsWhitelistAdminRequest) (*types.QueryIsWhitelistAdminResponse, error) {
	address, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	isAdmin, err := qs.k.IsWhitelistAdmin(ctx, address)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryIsWhitelistAdminResponse{IsAdmin: isAdmin}, nil
}
