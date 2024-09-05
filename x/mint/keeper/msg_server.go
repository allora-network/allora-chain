package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/mint/types"
)

var _ types.MsgServiceServer = msgServiceServer{}

// msgServiceServer is a wrapper of Keeper.
type msgServiceServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the x/mint MsgServer interface.
func NewMsgServerImpl(k Keeper) types.MsgServiceServer {
	return &msgServiceServer{
		Keeper: k,
	}
}

// UpdateParams updates the params.
func (ms msgServiceServer) UpdateParams(ctx context.Context, msg *types.MsgServiceUpdateParamsRequest) (*types.MsgServiceUpdateParamsResponse, error) {
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

	return &types.MsgServiceUpdateParamsResponse{}, nil
}

func (ms msgServiceServer) RecalculateTargetEmission(ctx context.Context, msg *types.MsgServiceRecalculateTargetEmissionRequest) (*types.MsgServiceRecalculateTargetEmissionResponse, error) {
	return nil, nil
}
