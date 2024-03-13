package msgserver

import (
	"context"

	state "github.com/allora-network/allora-chain/x/emissions"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

///
/// WHITELIST
///

func (ms msgServer) AddToWhitelistAdmin(ctx context.Context, msg *state.MsgAddToWhitelistAdmin) (*state.MsgAddToWhitelistAdminResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Add the address to the whitelist
	err = ms.k.AddWhitelistAdmin(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgAddToWhitelistAdminResponse{}, nil
}

func (ms msgServer) RemoveFromWhitelistAdmin(ctx context.Context, msg *state.MsgRemoveFromWhitelistAdmin) (*state.MsgRemoveFromWhitelistAdminResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Remove the address from the whitelist
	err = ms.k.RemoveWhitelistAdmin(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgRemoveFromWhitelistAdminResponse{}, nil
}

func (ms msgServer) AddToTopicCreationWhitelist(ctx context.Context, msg *state.MsgAddToTopicCreationWhitelist) (*state.MsgAddToTopicCreationWhitelistResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Add the address to the whitelist
	err = ms.k.AddToTopicCreationWhitelist(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgAddToTopicCreationWhitelistResponse{}, nil
}

func (ms msgServer) RemoveFromTopicCreationWhitelist(ctx context.Context, msg *state.MsgRemoveFromTopicCreationWhitelist) (*state.MsgRemoveFromTopicCreationWhitelistResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Remove the address from the whitelist
	err = ms.k.RemoveFromTopicCreationWhitelist(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgRemoveFromTopicCreationWhitelistResponse{}, nil
}

func (ms msgServer) AddToReputerWhitelist(ctx context.Context, msg *state.MsgAddToReputerWhitelist) (*state.MsgAddToReputerWhitelistResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Add the address to the whitelist
	err = ms.k.AddToReputerWhitelist(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgAddToReputerWhitelistResponse{}, nil
}

func (ms msgServer) RemoveFromReputerWhitelist(ctx context.Context, msg *state.MsgRemoveFromReputerWhitelist) (*state.MsgRemoveFromReputerWhitelistResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Remove the address from the whitelist
	err = ms.k.RemoveFromReputerWhitelist(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgRemoveFromReputerWhitelistResponse{}, nil
}

func (ms msgServer) AddToFoundationWhitelist(ctx context.Context, msg *state.MsgAddToFoundationWhitelist) (*state.MsgAddToFoundationWhitelistResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Add the address to the foundation whitelist
	err = ms.k.AddToFoundationWhitelist(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgAddToFoundationWhitelistResponse{}, nil
}

func (ms msgServer) RemoveFromFoundationWhitelist(ctx context.Context, msg *state.MsgRemoveFromFoundationWhitelist) (*state.MsgRemoveFromFoundationWhitelistResponse, error) {
	// Check that sender is also a whitelist admin
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	targetAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return nil, err
	}
	isAdmin, err := ms.k.IsWhitelistAdmin(ctx, senderAddr)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, state.ErrNotWhitelistAdmin
	}
	// Remove the address from the foundation whitelist
	err = ms.k.RemoveFromFoundationWhitelist(ctx, targetAddr)
	if err != nil {
		return nil, err
	}
	return &state.MsgRemoveFromFoundationWhitelistResponse{}, nil
}
