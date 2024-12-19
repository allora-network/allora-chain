package msgserver

import (
	"context"
	"time"

	"github.com/allora-network/allora-chain/x/emissions/metrics"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (ms msgServer) AddToWhitelistAdmin(ctx context.Context, msg *types.AddToWhitelistAdminRequest) (_ *types.AddToWhitelistAdminResponse, err error) {
	defer metrics.RecordMetrics("AddToWhitelistAdmin", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.k.CanUpdateAllGlobalWhitelists(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateWhitelistAdmins
	}

	// Add the address to the whitelist
	return &types.AddToWhitelistAdminResponse{}, ms.k.AddWhitelistAdmin(ctx, msg.Address)
}

func (ms msgServer) RemoveFromWhitelistAdmin(ctx context.Context, msg *types.RemoveFromWhitelistAdminRequest) (_ *types.RemoveFromWhitelistAdminResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromWhitelistAdmin", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.k.CanUpdateAllGlobalWhitelists(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateWhitelistAdmins
	}

	// Remove the address from the whitelist
	return &types.RemoveFromWhitelistAdminResponse{}, ms.k.RemoveWhitelistAdmin(ctx, msg.Address)
}

func (ms msgServer) AddToGlobalWhitelist(ctx context.Context, msg *types.AddToGlobalWhitelistRequest) (_ *types.AddToGlobalWhitelistResponse, err error) {
	defer metrics.RecordMetrics("AddToGlobalWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.k.CanUpdateAllGlobalWhitelists(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Add the address to the whitelist
	return &types.AddToGlobalWhitelistResponse{}, ms.k.AddToGlobalWhitelist(ctx, msg.Address)
}

func (ms msgServer) RemoveFromGlobalWhitelist(ctx context.Context, msg *types.RemoveFromGlobalWhitelistRequest) (_ *types.RemoveFromGlobalWhitelistResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromGlobalWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.k.CanUpdateAllGlobalWhitelists(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Remove the address from the whitelist
	return &types.RemoveFromGlobalWhitelistResponse{}, ms.k.RemoveFromGlobalWhitelist(ctx, msg.Address)
}

func (ms msgServer) AddToGlobalWorkerWhitelist(ctx context.Context, msg *types.AddToGlobalWorkerWhitelistRequest) (_ *types.AddToGlobalWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("AddToGlobalWorkerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.k.CanUpdateGlobalWorkerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Add the address to the whitelist
	return &types.AddToGlobalWorkerWhitelistResponse{}, ms.k.AddToGlobalWorkerWhitelist(ctx, msg.Address)
}

func (ms msgServer) RemoveFromGlobalWorkerWhitelist(ctx context.Context, msg *types.RemoveFromGlobalWorkerWhitelistRequest) (_ *types.RemoveFromGlobalWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromGlobalWorkerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.k.CanUpdateGlobalWorkerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Remove the address from the whitelist
	return &types.RemoveFromGlobalWorkerWhitelistResponse{}, ms.k.RemoveFromGlobalWorkerWhitelist(ctx, msg.Address)
}

func (ms msgServer) AddToGlobalReputerWhitelist(ctx context.Context, msg *types.AddToGlobalReputerWhitelistRequest) (_ *types.AddToGlobalReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("AddToGlobalReputerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.k.CanUpdateGlobalReputerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Add the address to the whitelist
	return &types.AddToGlobalReputerWhitelistResponse{}, ms.k.AddToGlobalReputerWhitelist(ctx, msg.Address)
}

func (ms msgServer) RemoveFromGlobalReputerWhitelist(ctx context.Context, msg *types.RemoveFromGlobalReputerWhitelistRequest) (_ *types.RemoveFromGlobalReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromGlobalReputerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.k.CanUpdateGlobalReputerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Remove the address from the whitelist
	return &types.RemoveFromGlobalReputerWhitelistResponse{}, ms.k.RemoveFromGlobalReputerWhitelist(ctx, msg.Address)
}

func (ms msgServer) AddToGlobalAdminWhitelist(ctx context.Context, msg *types.AddToGlobalAdminWhitelistRequest) (_ *types.AddToGlobalAdminWhitelistResponse, err error) {
	defer metrics.RecordMetrics("AddToGlobalAdminWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.k.CanUpdateAllGlobalWhitelists(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Add the address to the whitelist
	return &types.AddToGlobalAdminWhitelistResponse{}, ms.k.AddToGlobalAdminWhitelist(ctx, msg.Address)
}

func (ms msgServer) RemoveFromGlobalAdminWhitelist(ctx context.Context, msg *types.RemoveFromGlobalAdminWhitelistRequest) (_ *types.RemoveFromGlobalAdminWhitelistResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromGlobalAdminWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.k.CanUpdateAllGlobalWhitelists(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Remove the address from the whitelist
	return &types.RemoveFromGlobalAdminWhitelistResponse{}, ms.k.RemoveFromGlobalAdminWhitelist(ctx, msg.Address)
}

func (ms msgServer) BulkAddToGlobalWorkerWhitelist(ctx context.Context, msg *types.BulkAddToGlobalWorkerWhitelistRequest) (_ *types.BulkAddToGlobalWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkAddToGlobalWorkerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.k.CanUpdateGlobalWorkerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	for _, address := range msg.Addresses {
		err := ms.k.AddToGlobalWorkerWhitelist(ctx, address)
		if err != nil {
			return nil, err
		}
	}

	return &types.BulkAddToGlobalWorkerWhitelistResponse{}, nil
}

func (ms msgServer) BulkRemoveFromGlobalWorkerWhitelist(ctx context.Context, msg *types.BulkRemoveFromGlobalWorkerWhitelistRequest) (_ *types.BulkRemoveFromGlobalWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkRemoveFromGlobalWorkerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.k.CanUpdateGlobalWorkerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	for _, address := range msg.Addresses {
		err := ms.k.RemoveFromGlobalWorkerWhitelist(ctx, address)
		if err != nil {
			return nil, err
		}
	}

	return &types.BulkRemoveFromGlobalWorkerWhitelistResponse{}, nil
}

func (ms msgServer) BulkAddToGlobalReputerWhitelist(ctx context.Context, msg *types.BulkAddToGlobalReputerWhitelistRequest) (_ *types.BulkAddToGlobalReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkAddToGlobalReputerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.k.CanUpdateGlobalReputerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	for _, address := range msg.Addresses {
		err := ms.k.AddToGlobalReputerWhitelist(ctx, address)
		if err != nil {
			return nil, err
		}
	}

	return &types.BulkAddToGlobalReputerWhitelistResponse{}, nil
}

func (ms msgServer) BulkRemoveFromGlobalReputerWhitelist(ctx context.Context, msg *types.BulkRemoveFromGlobalReputerWhitelistRequest) (_ *types.BulkRemoveFromGlobalReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkRemoveFromGlobalReputerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check that sender is permitted to update global whitelists
	canUpdate, err := ms.k.CanUpdateGlobalReputerWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateGlobalWhitelist
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	for _, address := range msg.Addresses {
		err := ms.k.RemoveFromGlobalReputerWhitelist(ctx, address)
		if err != nil {
			return nil, err
		}
	}

	return &types.BulkRemoveFromGlobalReputerWhitelistResponse{}, nil
}

func (ms msgServer) BulkAddToTopicWorkerWhitelist(ctx context.Context, msg *types.BulkAddToTopicWorkerWhitelistRequest) (_ *types.BulkAddToTopicWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkAddToTopicWorkerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check that topic exists
	exists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Check that sender is permitted to update topic whitelists
	canUpdate, err := ms.k.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	for _, address := range msg.Addresses {
		// The main benefits of bulk operations are defeated if we do too much in-loop compute, and we validate addresses in layer below anyway
		// => no need to validate address here.

		err := ms.k.AddToTopicWorkerWhitelist(ctx, msg.TopicId, address)
		if err != nil {
			return nil, err
		}
	}

	return &types.BulkAddToTopicWorkerWhitelistResponse{}, nil
}

func (ms msgServer) BulkRemoveFromTopicWorkerWhitelist(ctx context.Context, msg *types.BulkRemoveFromTopicWorkerWhitelistRequest) (_ *types.BulkRemoveFromTopicWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkRemoveFromTopicWorkerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check that topic exists
	exists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Check that sender is permitted to update topic whitelists
	canUpdate, err := ms.k.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	for _, address := range msg.Addresses {
		// The main benefits of bulk operations are defeated if we do too much in-loop compute, and we validate addresses in layer below anyway
		// => no need to validate address here.

		err := ms.k.RemoveFromTopicWorkerWhitelist(ctx, msg.TopicId, address)
		if err != nil {
			return nil, err
		}
	}

	return &types.BulkRemoveFromTopicWorkerWhitelistResponse{}, nil
}

func (ms msgServer) BulkAddToTopicReputerWhitelist(ctx context.Context, msg *types.BulkAddToTopicReputerWhitelistRequest) (_ *types.BulkAddToTopicReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkAddToTopicReputerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check that topic exists
	exists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Check that sender is permitted to update topic whitelists
	canUpdate, err := ms.k.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	for _, address := range msg.Addresses {
		// The main benefits of bulk operations are defeated if we do too much in-loop compute, and we validate addresses in layer below anyway
		// => no need to validate address here.

		err := ms.k.AddToTopicReputerWhitelist(ctx, msg.TopicId, address)
		if err != nil {
			return nil, err
		}
	}

	return &types.BulkAddToTopicReputerWhitelistResponse{}, nil
}

func (ms msgServer) BulkRemoveFromTopicReputerWhitelist(ctx context.Context, msg *types.BulkRemoveFromTopicReputerWhitelistRequest) (_ *types.BulkRemoveFromTopicReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("BulkRemoveFromTopicReputerWhitelist", time.Now(), &err)

	// Validate the sender address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Check that topic exists
	exists, err := ms.k.TopicExists(ctx, msg.TopicId)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, types.ErrTopicDoesNotExist
	}

	// Check that sender is permitted to update topic whitelists
	canUpdate, err := ms.k.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	// Check length of addresses to add using global max_whitelist_input_array_length
	params, err := ms.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if uint64(len(msg.Addresses)) > params.MaxWhitelistInputArrayLength {
		return nil, types.ErrMaxWhitelistInputArrayLengthExceeded
	}

	for _, address := range msg.Addresses {
		// The main benefits of bulk operations are defeated if we do too much in-loop compute, and we validate addresses in layer below anyway
		// => no need to validate address here.

		err := ms.k.RemoveFromTopicReputerWhitelist(ctx, msg.TopicId, address)
		if err != nil {
			return nil, err
		}
	}

	return &types.BulkRemoveFromTopicReputerWhitelistResponse{}, nil
}

func (ms msgServer) EnableTopicWorkerWhitelist(ctx context.Context, msg *types.EnableTopicWorkerWhitelistRequest) (_ *types.EnableTopicWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("EnableTopicWorkerWhitelist", time.Now(), &err)

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.k.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	return &types.EnableTopicWorkerWhitelistResponse{}, ms.k.EnableTopicWorkerWhitelist(ctx, msg.TopicId)
}

func (ms msgServer) DisableTopicWorkerWhitelist(ctx context.Context, msg *types.DisableTopicWorkerWhitelistRequest) (_ *types.DisableTopicWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("DisableTopicWorkerWhitelist", time.Now(), &err)

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.k.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	return &types.DisableTopicWorkerWhitelistResponse{}, ms.k.DisableTopicWorkerWhitelist(ctx, msg.TopicId)
}

func (ms msgServer) EnableTopicReputerWhitelist(ctx context.Context, msg *types.EnableTopicReputerWhitelistRequest) (_ *types.EnableTopicReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("EnableTopicReputerWhitelist", time.Now(), &err)

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.k.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	return &types.EnableTopicReputerWhitelistResponse{}, ms.k.EnableTopicReputerWhitelist(ctx, msg.TopicId)
}

func (ms msgServer) DisableTopicReputerWhitelist(ctx context.Context, msg *types.DisableTopicReputerWhitelistRequest) (_ *types.DisableTopicReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("DisableTopicReputerWhitelist", time.Now(), &err)

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.k.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWhitelist
	}

	return &types.DisableTopicReputerWhitelistResponse{}, ms.k.DisableTopicReputerWhitelist(ctx, msg.TopicId)
}

func (ms msgServer) AddToTopicCreatorWhitelist(ctx context.Context, msg *types.AddToTopicCreatorWhitelistRequest) (_ *types.AddToTopicCreatorWhitelistResponse, err error) {
	defer metrics.RecordMetrics("AddToTopicCreatorWhitelist", time.Now(), &err)

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.k.CanUpdateTopicCreatorWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicCreatorWhitelist
	}

	return &types.AddToTopicCreatorWhitelistResponse{}, ms.k.AddToTopicCreatorWhitelist(ctx, msg.Address)
}

func (ms msgServer) RemoveFromTopicCreatorWhitelist(ctx context.Context, msg *types.RemoveFromTopicCreatorWhitelistRequest) (_ *types.RemoveFromTopicCreatorWhitelistResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromTopicCreatorWhitelist", time.Now(), &err)

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.k.CanUpdateTopicCreatorWhitelist(ctx, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicCreatorWhitelist
	}

	return &types.RemoveFromTopicCreatorWhitelistResponse{}, ms.k.RemoveFromTopicCreatorWhitelist(ctx, msg.Address)
}

func (ms msgServer) AddToTopicWorkerWhitelist(ctx context.Context, msg *types.AddToTopicWorkerWhitelistRequest) (_ *types.AddToTopicWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("AddToTopicWorkerWhitelist", time.Now(), &err)

	// Validate the sender
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.k.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWorkerWhitelist
	}

	return &types.AddToTopicWorkerWhitelistResponse{}, ms.k.AddToTopicWorkerWhitelist(ctx, msg.TopicId, msg.Address)
}

func (ms msgServer) RemoveFromTopicWorkerWhitelist(ctx context.Context, msg *types.RemoveFromTopicWorkerWhitelistRequest) (_ *types.RemoveFromTopicWorkerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromTopicWorkerWhitelist", time.Now(), &err)

	// Validate the sender
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.k.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicWorkerWhitelist
	}

	return &types.RemoveFromTopicWorkerWhitelistResponse{}, ms.k.RemoveFromTopicWorkerWhitelist(ctx, msg.TopicId, msg.Address)
}

func (ms msgServer) AddToTopicReputerWhitelist(ctx context.Context, msg *types.AddToTopicReputerWhitelistRequest) (_ *types.AddToTopicReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("AddToTopicReputerWhitelist", time.Now(), &err)

	// Validate the sender
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.k.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicReputerWhitelist
	}

	return &types.AddToTopicReputerWhitelistResponse{}, ms.k.AddToTopicReputerWhitelist(ctx, msg.TopicId, msg.Address)
}

func (ms msgServer) RemoveFromTopicReputerWhitelist(ctx context.Context, msg *types.RemoveFromTopicReputerWhitelistRequest) (_ *types.RemoveFromTopicReputerWhitelistResponse, err error) {
	defer metrics.RecordMetrics("RemoveFromTopicReputerWhitelist", time.Now(), &err)

	// Validate the sender
	err = ms.k.ValidateStringIsBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// Validate the address
	err = ms.k.ValidateStringIsBech32(msg.Address)
	if err != nil {
		return nil, err
	}

	canUpdate, err := ms.k.CanUpdateTopicWhitelist(ctx, msg.TopicId, msg.Sender)
	if err != nil {
		return nil, err
	} else if !canUpdate {
		return nil, types.ErrNotPermittedToUpdateTopicReputerWhitelist
	}

	return &types.RemoveFromTopicReputerWhitelistResponse{}, ms.k.RemoveFromTopicReputerWhitelist(ctx, msg.TopicId, msg.Address)
}
