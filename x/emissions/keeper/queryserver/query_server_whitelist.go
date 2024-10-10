package queryserver

import (
	"context"
	"time"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/allora-network/allora-chain/x/emissions/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Params defines the handler for the Query/Params RPC method.
func (qs queryServer) IsWhitelistAdmin(ctx context.Context, req *types.IsWhitelistAdminRequest) (_ *types.IsWhitelistAdminResponse, err error) {
	defer metrics.RecordMetrics("IsWhitelistAdmin", time.Now(), &err == nil)
	if err := qs.k.ValidateStringIsBech32(req.Address); err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid address: %s", err)
	}

	isAdmin, err := qs.k.IsWhitelistAdmin(ctx, req.Address)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.IsWhitelistAdminResponse{IsAdmin: isAdmin}, nil
}
