package msgserver

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (ms msgServer) AddToWhitelistAdmin(ctx context.Context, msg *types.MsgAddToWhitelistAdmin) (*types.MsgAddToWhitelistAdminResponse, error) {
	// Check that sender is also a whitelist admin
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, msg.Sender)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, types.ErrNotWhitelistAdmin
	}
	// Add the address to the whitelist
	err = ms.k.AddWhitelistAdmin(ctx, msg.Address)
	if err != nil {
		return nil, err
	}
	return &types.MsgAddToWhitelistAdminResponse{}, nil
}

func (ms msgServer) RemoveFromWhitelistAdmin(ctx context.Context, msg *types.MsgRemoveFromWhitelistAdmin) (*types.MsgRemoveFromWhitelistAdminResponse, error) {
	// Check that sender is also a whitelist admin
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, msg.Sender)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, types.ErrNotWhitelistAdmin
	}
	// Remove the address from the whitelist
	err = ms.k.RemoveWhitelistAdmin(ctx, msg.Address)
	if err != nil {
		return nil, err
	}
	return &types.MsgRemoveFromWhitelistAdminResponse{}, nil
}
