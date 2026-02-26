package msgserver

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (ms msgServer) AddToWhitelistAdmin(ctx context.Context, msg *types.AddToWhitelistAdminRequest) (_ *types.AddToWhitelistAdminResponse, err error) {
	defer metrics.RecordMetrics("AddToWhitelistAdmin", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.wlk.CanUpdateAllGlobalWhitelists(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateWhitelistAdmins
	}

	// Add the address to the whitelist
	if err = ms.wlk.AddWhitelistAdmin(ctx, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "error adding whitelist admin")
	}

	types.EmitNewWhitelistAdminAddedEvent(ctx, msg.Address)
	return &types.AddToWhitelistAdminResponse{}, nil
}

func (ms msgServer) RemoveFromWhitelistAdmin(ctx context.Context, msg *types.RemoveFromWhitelistAdminRequest) (_ *types.RemoveFromWhitelistAdminResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromWhitelistAdmin", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.wlk.CanUpdateAllGlobalWhitelists(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateWhitelistAdmins
	}

	// Remove the address from the whitelist
	if err = ms.wlk.RemoveWhitelistAdmin(ctx, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "error removing whitelist admin")
	}

	types.EmitNewWhitelistAdminRemovedEvent(ctx, msg.Address)
	return &types.RemoveFromWhitelistAdminResponse{}, nil
}

func (ms msgServer) AddToGlobalWhitelist(ctx context.Context, msg *types.AddToGlobalWhitelistRequest) (_ *types.AddToGlobalWhitelistResponse, err error) {
	defer metrics.RecordMetrics("AddToGlobalWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.wlk.CanUpdateAllGlobalWhitelists(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Add the address to the whitelist
	if err = ms.wlk.AddToGlobalWhitelist(ctx, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "error adding to global whitelist")
	}

	types.EmitNewGlobalWhitelistAddedEvent(ctx, msg.Address)
	return &types.AddToGlobalWhitelistResponse{}, nil
}

func (ms msgServer) RemoveFromGlobalWhitelist(ctx context.Context, msg *types.RemoveFromGlobalWhitelistRequest) (_ *types.RemoveFromGlobalWhitelistResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromGlobalWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.wlk.CanUpdateAllGlobalWhitelists(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Remove the address from the whitelist
	if err = ms.wlk.RemoveFromGlobalWhitelist(ctx, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "error removing from global whitelist")
	}

	types.EmitNewGlobalWhitelistRemovedEvent(ctx, msg.Address)
	return &types.RemoveFromGlobalWhitelistResponse{}, nil
}

func (ms msgServer) AddToGlobalWorkerWhitelist(ctx context.Context, msg *types.AddToGlobalWorkerWhitelistRequest) (_ *types.AddToGlobalWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("AddToGlobalWorkerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.wlk.CanUpdateGlobalWorkerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Add the address to the whitelist
	if err = ms.wlk.AddToGlobalWorkerWhitelist(ctx, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "error adding to global worker whitelist")
	}

	types.EmitNewGlobalWorkerWhitelistAddedEvent(ctx, msg.Address)
	return &types.AddToGlobalWorkerWhitelistResponse{}, nil
}

func (ms msgServer) RemoveFromGlobalWorkerWhitelist(ctx context.Context, msg *types.RemoveFromGlobalWorkerWhitelistRequest) (_ *types.RemoveFromGlobalWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromGlobalWorkerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.wlk.CanUpdateGlobalWorkerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Remove the address from the whitelist
	if err = ms.wlk.RemoveFromGlobalWorkerWhitelist(ctx, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "error removing from global worker whitelist")
	}

	types.EmitNewGlobalWorkerWhitelistRemovedEvent(ctx, msg.Address)
	return &types.RemoveFromGlobalWorkerWhitelistResponse{}, err
}

func (ms msgServer) AddToGlobalReputerWhitelist(ctx context.Context, msg *types.AddToGlobalReputerWhitelistRequest) (_ *types.AddToGlobalReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("AddToGlobalReputerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.wlk.CanUpdateGlobalReputerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Add the address to the whitelist
	if err = ms.wlk.AddToGlobalReputerWhitelist(ctx, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "error adding to global reputer whitelist")
	}

	types.EmitNewGlobalReputerWhitelistAddedEvent(ctx, msg.Address)
	return &types.AddToGlobalReputerWhitelistResponse{}, err
}

func (ms msgServer) RemoveFromGlobalReputerWhitelist(ctx context.Context, msg *types.RemoveFromGlobalReputerWhitelistRequest) (_ *types.RemoveFromGlobalReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromGlobalReputerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.wlk.CanUpdateGlobalReputerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Remove the address from the whitelist
	if err = ms.wlk.RemoveFromGlobalReputerWhitelist(ctx, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "error removing from global reputer whitelist")
	}

	types.EmitNewGlobalReputerWhitelistRemovedEvent(ctx, msg.Address)
	return &types.RemoveFromGlobalReputerWhitelistResponse{}, err
}

func (ms msgServer) AddToGlobalAdminWhitelist(ctx context.Context, msg *types.AddToGlobalAdminWhitelistRequest) (_ *types.AddToGlobalAdminWhitelistResponse, err error) {
	defer metrics.RecordMetrics("AddToGlobalAdminWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.wlk.CanUpdateAllGlobalWhitelists(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Add the address to the whitelist
	if err = ms.wlk.AddToGlobalAdminWhitelist(ctx, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "error adding to global admin whitelist")
	}

	types.EmitNewGlobalAdminWhitelistAddedEvent(ctx, msg.Address)
	return &types.AddToGlobalAdminWhitelistResponse{}, err
}

func (ms msgServer) RemoveFromGlobalAdminWhitelist(ctx context.Context, msg *types.RemoveFromGlobalAdminWhitelistRequest) (_ *types.RemoveFromGlobalAdminWhitelistResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromGlobalAdminWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.wlk.CanUpdateAllGlobalWhitelists(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Remove the address from the whitelist
	if err = ms.wlk.RemoveFromGlobalAdminWhitelist(ctx, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "error removing from global admin whitelist")
	}

	types.EmitNewGlobalAdminWhitelistRemovedEvent(ctx, msg.Address)
	return &types.RemoveFromGlobalAdminWhitelistResponse{}, err
}

func (ms msgServer) BulkAddToGlobalWorkerWhitelist(ctx context.Context, msg *types.BulkAddToGlobalWorkerWhitelistRequest) (_ *types.BulkAddToGlobalWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkAddToGlobalWorkerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.pk.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.wlk.CanUpdateGlobalWorkerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	for _, address := range msg.Addresses {
		if err = ms.wlk.AddToGlobalWorkerWhitelist(ctx, address); err != nil {
			return nil, err
		}
		types.EmitNewGlobalWorkerWhitelistAddedEvent(ctx, address)
	}

	return &types.BulkAddToGlobalWorkerWhitelistResponse{}, nil
}

func (ms msgServer) BulkRemoveFromGlobalWorkerWhitelist(ctx context.Context, msg *types.BulkRemoveFromGlobalWorkerWhitelistRequest) (_ *types.BulkRemoveFromGlobalWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkRemoveFromGlobalWorkerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.wlk.CanUpdateGlobalWorkerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.pk.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	for _, address := range msg.Addresses {
		if err = ms.wlk.RemoveFromGlobalWorkerWhitelist(ctx, address); err != nil {
			return nil, err
		}
		types.EmitNewGlobalWorkerWhitelistRemovedEvent(ctx, address)
	}

	return &types.BulkRemoveFromGlobalWorkerWhitelistResponse{}, nil
}

func (ms msgServer) BulkAddToGlobalReputerWhitelist(ctx context.Context, msg *types.BulkAddToGlobalReputerWhitelistRequest) (_ *types.BulkAddToGlobalReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkAddToGlobalReputerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.wlk.CanUpdateGlobalReputerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.pk.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	for _, address := range msg.Addresses {
		if err = ms.wlk.AddToGlobalReputerWhitelist(ctx, address); err != nil {
			return nil, err
		}
		types.EmitNewGlobalReputerWhitelistAddedEvent(ctx, address)
	}

	return &types.BulkAddToGlobalReputerWhitelistResponse{}, nil
}

func (ms msgServer) BulkRemoveFromGlobalReputerWhitelist(ctx context.Context, msg *types.BulkRemoveFromGlobalReputerWhitelistRequest) (_ *types.BulkRemoveFromGlobalReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkRemoveFromGlobalReputerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.wlk.CanUpdateGlobalReputerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.pk.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	for _, address := range msg.Addresses {
		if err = ms.wlk.RemoveFromGlobalReputerWhitelist(ctx, address); err != nil {
			return nil, err
		}
		types.EmitNewGlobalReputerWhitelistRemovedEvent(ctx, address)
	}

	return &types.BulkRemoveFromGlobalReputerWhitelistResponse{}, nil
}

func (ms msgServer) BulkAddToTopicWorkerWhitelist(ctx context.Context, msg *types.BulkAddToTopicWorkerWhitelistRequest) (_ *types.BulkAddToTopicWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkAddToTopicWorkerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check that topic exists
	exists, err := ms.tk.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Check that sender is permitted to update topic whitelists
	canUpdate, err := ms.wlk.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.pk.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	for _, address := range msg.Addresses {
		// The main benefits of bulk operations are defeated if we do too much in-loop compute, and we validate addresses in layer below anyway
		// => no need to validate address here.

		if err = ms.wlk.AddToTopicWorkerWhitelist(ctx, msg.TopicId, address); err != nil {
			return nil, err
		}
		types.EmitNewTopicWorkerWhitelistAddedEvent(ctx, msg.TopicId, address)
	}

	return &types.BulkAddToTopicWorkerWhitelistResponse{}, nil
}

func (ms msgServer) BulkRemoveFromTopicWorkerWhitelist(ctx context.Context, msg *types.BulkRemoveFromTopicWorkerWhitelistRequest) (_ *types.BulkRemoveFromTopicWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkRemoveFromTopicWorkerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check that topic exists
	exists, err := ms.tk.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Check that sender is permitted to update topic whitelists
	canUpdate, err := ms.wlk.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.pk.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	for _, address := range msg.Addresses {
		// The main benefits of bulk operations are defeated if we do too much in-loop compute, and we validate addresses in layer below anyway
		// => no need to validate address here.

		if err = ms.wlk.RemoveFromTopicWorkerWhitelist(ctx, msg.TopicId, address); err != nil {
			return nil, err
		}
		types.EmitNewTopicWorkerWhitelistRemovedEvent(ctx, msg.TopicId, address)
	}

	return &types.BulkRemoveFromTopicWorkerWhitelistResponse{}, nil
}

func (ms msgServer) BulkAddToTopicReputerWhitelist(ctx context.Context, msg *types.BulkAddToTopicReputerWhitelistRequest) (_ *types.BulkAddToTopicReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkAddToTopicReputerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check that topic exists
	exists, err := ms.tk.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Check that sender is permitted to update topic whitelists
	canUpdate, err := ms.wlk.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.pk.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	for _, address := range msg.Addresses {
		// The main benefits of bulk operations are defeated if we do too much in-loop compute, and we validate addresses in layer below anyway
		// => no need to validate address here.

		if err = ms.wlk.AddToTopicReputerWhitelist(ctx, msg.TopicId, address); err != nil {
			return nil, err
		}
		types.EmitNewTopicReputerWhitelistAddedEvent(ctx, msg.TopicId, address)
	}

	return &types.BulkAddToTopicReputerWhitelistResponse{}, nil
}

func (ms msgServer) BulkRemoveFromTopicReputerWhitelist(ctx context.Context, msg *types.BulkRemoveFromTopicReputerWhitelistRequest) (_ *types.BulkRemoveFromTopicReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkRemoveFromTopicReputerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check that topic exists
	exists, err := ms.tk.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Check that sender is permitted to update topic whitelists
	canUpdate, err := ms.wlk.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.pk.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	for _, address := range msg.Addresses {
		// The main benefits of bulk operations are defeated if we do too much in-loop compute, and we validate addresses in layer below anyway
		// => no need to validate address here.

		if err = ms.wlk.RemoveFromTopicReputerWhitelist(ctx, msg.TopicId, address); err != nil {
			return nil, err
		}
		types.EmitNewTopicReputerWhitelistRemovedEvent(ctx, msg.TopicId, address)
	}

	return &types.BulkRemoveFromTopicReputerWhitelistResponse{}, nil
}

func (ms msgServer) EnableTopicWorkerWhitelist(ctx context.Context, msg *types.EnableTopicWorkerWhitelistRequest) (_ *types.EnableTopicWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("EnableTopicWorkerWhitelist", time.Now(), &err)

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.wlk.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	if err = ms.wlk.EnableTopicWorkerWhitelist(ctx, msg.TopicId); err != nil {
		return nil, errorsmod.Wrap(err, "unable to enable topic worker whitelist")
	}

	types.EmitNewTopicWorkerWhitelistEnabledEvent(ctx, msg.TopicId)
	return &types.EnableTopicWorkerWhitelistResponse{}, nil
}

func (ms msgServer) DisableTopicWorkerWhitelist(ctx context.Context, msg *types.DisableTopicWorkerWhitelistRequest) (_ *types.DisableTopicWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("DisableTopicWorkerWhitelist", time.Now(), &err)

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.wlk.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	if err = ms.wlk.DisableTopicWorkerWhitelist(ctx, msg.TopicId); err != nil {
		return nil, errorsmod.Wrap(err, "unable to disable topic worker whitelist")
	}

	types.EmitNewTopicWorkerWhitelistDisabledEvent(ctx, msg.TopicId)
	return &types.DisableTopicWorkerWhitelistResponse{}, nil
}

func (ms msgServer) EnableTopicReputerWhitelist(ctx context.Context, msg *types.EnableTopicReputerWhitelistRequest) (_ *types.EnableTopicReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("EnableTopicReputerWhitelist", time.Now(), &err)

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.wlk.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	if err = ms.wlk.EnableTopicReputerWhitelist(ctx, msg.TopicId); err != nil {
		return nil, errorsmod.Wrap(err, "unable to enable topic reputer whitelist")
	}

	types.EmitNewTopicReputerWhitelistEnabledEvent(ctx, msg.TopicId)
	return &types.EnableTopicReputerWhitelistResponse{}, nil
}

func (ms msgServer) DisableTopicReputerWhitelist(ctx context.Context, msg *types.DisableTopicReputerWhitelistRequest) (_ *types.DisableTopicReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("DisableTopicReputerWhitelist", time.Now(), &err)

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.wlk.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	if err = ms.wlk.DisableTopicReputerWhitelist(ctx, msg.TopicId); err != nil {
		return nil, errorsmod.Wrap(err, "unable to disable topic reputer whitelist")
	}

	types.EmitNewTopicReputerWhitelistDisabledEvent(ctx, msg.TopicId)
	return &types.DisableTopicReputerWhitelistResponse{}, nil
}

func (ms msgServer) AddToTopicCreatorWhitelist(ctx context.Context, msg *types.AddToTopicCreatorWhitelistRequest) (_ *types.AddToTopicCreatorWhitelistResponse, err error) {
	defer metrics.RecordMetrics("AddToTopicCreatorWhitelist", time.Now(), &err)

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.wlk.CanUpdateTopicCreatorWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicCreatorWhitelist
	}

	if err = ms.wlk.AddToTopicCreatorWhitelist(ctx, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "unable to add to topic whitelist")
	}

	types.EmitNewTopicCreatorWhitelistAddedEvent(ctx, msg.Address)
	return &types.AddToTopicCreatorWhitelistResponse{}, nil
}

func (ms msgServer) RemoveFromTopicCreatorWhitelist(ctx context.Context, msg *types.RemoveFromTopicCreatorWhitelistRequest) (_ *types.RemoveFromTopicCreatorWhitelistResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromTopicCreatorWhitelist", time.Now(), &err)

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.wlk.CanUpdateTopicCreatorWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicCreatorWhitelist
	}

	if err = ms.wlk.RemoveFromTopicCreatorWhitelist(ctx, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "unable to remove from topic whitelist")
	}

	types.EmitNewTopicCreatorWhitelistRemovedEvent(ctx, msg.Address)
	return &types.RemoveFromTopicCreatorWhitelistResponse{}, nil
}

func (ms msgServer) AddToTopicWorkerWhitelist(ctx context.Context, msg *types.AddToTopicWorkerWhitelistRequest) (_ *types.AddToTopicWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("AddToTopicWorkerWhitelist", time.Now(), &err)

	// Validate the sender
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.wlk.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWorkerWhitelist
	}

	if err = ms.wlk.AddToTopicWorkerWhitelist(ctx, msg.TopicId, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "unable to add to topic whitelist")
	}

	types.EmitNewTopicWorkerWhitelistAddedEvent(ctx, msg.TopicId, msg.Address)
	return &types.AddToTopicWorkerWhitelistResponse{}, nil
}

func (ms msgServer) RemoveFromTopicWorkerWhitelist(ctx context.Context, msg *types.RemoveFromTopicWorkerWhitelistRequest) (_ *types.RemoveFromTopicWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromTopicWorkerWhitelist", time.Now(), &err)

	// Validate the sender
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.wlk.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWorkerWhitelist
	}

	if err = ms.wlk.RemoveFromTopicWorkerWhitelist(ctx, msg.TopicId, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "unable to remove from topic whitelist")
	}

	types.EmitNewTopicWorkerWhitelistRemovedEvent(ctx, msg.TopicId, msg.Address)
	return &types.RemoveFromTopicWorkerWhitelistResponse{}, nil
}

func (ms msgServer) AddToTopicReputerWhitelist(ctx context.Context, msg *types.AddToTopicReputerWhitelistRequest) (_ *types.AddToTopicReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("AddToTopicReputerWhitelist", time.Now(), &err)

	// Validate the sender
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.wlk.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicReputerWhitelist
	}

	if err = ms.wlk.AddToTopicReputerWhitelist(ctx, msg.TopicId, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "unable to add to topic whitelist")
	}

	types.EmitNewTopicReputerWhitelistAddedEvent(ctx, msg.TopicId, msg.Address)
	return &types.AddToTopicReputerWhitelistResponse{}, nil
}

func (ms msgServer) RemoveFromTopicReputerWhitelist(ctx context.Context, msg *types.RemoveFromTopicReputerWhitelistRequest) (_ *types.RemoveFromTopicReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromTopicReputerWhitelist", time.Now(), &err)

	// Validate the sender
	err = keeper.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = keeper.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.wlk.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicReputerWhitelist
	}

	if err = ms.wlk.RemoveFromTopicReputerWhitelist(ctx, msg.TopicId, msg.Address); err != nil {
		return nil, errorsmod.Wrap(err, "unable to remove from topic whitelist")
	}

	types.EmitNewTopicReputerWhitelistRemovedEvent(ctx, msg.TopicId, msg.Address)

	return &types.RemoveFromTopicReputerWhitelistResponse{}, nil
}
