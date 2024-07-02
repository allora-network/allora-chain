package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/mint/types"
)

var _ types.MsgServer = msgServer{}

// msgServer is a wrapper of Keeper.
type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the x/mint MsgServer interface.
func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{
		Keeper: k,
	}
}

// UpdateParams updates the params.
func (ms msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	isAdmin, err := ms.IsWhitelistAdmin(ctx, msg.Sender)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, errors.Wrapf(types.ErrUnauthorized, " %s not whitelist admin for mint update params", msg.Sender)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	if err := ms.Params.Set(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
