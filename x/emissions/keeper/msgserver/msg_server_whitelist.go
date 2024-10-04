package msgserver

import (
	"context"
	"time"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (ms msgServer) AddToWhitelistAdmin(ctx context.Context, msg *types.AddToWhitelistAdminRequest,
) (
	_ *types.AddToWhitelistAdminResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("AddToWhitelistAdmin", time.Now(), returnErr == nil)
	// Validate the sender address
	err := ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	// Check that sender is also a whitelist admin
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !isAdmin {
		return nil, types.ErrNotWhitelistAdmin
	}
	// Validate the address
	if err := ms.k.ValidateStringIsBech32(msg.Address); err != nil {
		return nil, err
	}
	// Add the address to the whitelist
	err = ms.k.AddWhitelistAdmin(ctx, msg.Address)
	if err != nil {
		return nil, err
	}
	return &types.AddToWhitelistAdminResponse{}, nil
}

func (ms msgServer) RemoveFromWhitelistAdmin(ctx context.Context, msg *types.RemoveFromWhitelistAdminRequest,
) (
	_ *types.RemoveFromWhitelistAdminResponse,
	returnErr error,
) {
	defer metrics.RecordMetrics("RemoveFromWhitelistAdmin", time.Now(), returnErr == nil)
	// Validate the sender address
	err := ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	// Check that sender is also a whitelist admin
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !isAdmin {
		return nil, types.ErrNotWhitelistAdmin
	}
	// Validate the address
	if err := ms.k.ValidateStringIsBech32(msg.Address); err != nil {
		return nil, err
	}
	// Remove the address from the whitelist
	err = ms.k.RemoveWhitelistAdmin(ctx, msg.Address)
	if err != nil {
		return nil, err
	}
	return &types.RemoveFromWhitelistAdminResponse{}, nil
}
