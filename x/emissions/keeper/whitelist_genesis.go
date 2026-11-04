package keeper

import (
	"context"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (k *WhitelistsKeeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	// CoreTeamAddresses, WhitelistAdmins []string
	// This allows us to add core team addresses to the whitelist during a genesis import
	// while still keeping the original core team addresses in the genesis file
	if len(data.CoreTeamAddresses) != 0 || len(data.WhitelistAdmins) != 0 {
		for _, address := range append(data.CoreTeamAddresses, data.WhitelistAdmins...) {
			_, err := sdk.AccAddressFromBech32(address)
			if err != nil {
				return errors.Wrap(err, "error converting core team address from bech32")
			}
			err = k.AddWhitelistAdmin(ctx, address)
			if err != nil {
				return errors.Wrap(err, "error adding core team addresses to whitelists")
			}
		}
	}

	// globalWhitelist
	for _, address := range data.GlobalWhitelist {
		if err := k.AddToGlobalWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting globalWhitelist")
		}
	}

	// globalWorkerWhitelist
	for _, address := range data.GlobalWorkerWhitelist {
		if err := k.AddToGlobalWorkerWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting globalWorkerWhitelist")
		}
	}

	// globalReputerWhitelist
	for _, address := range data.GlobalReputerWhitelist {
		if err := k.AddToGlobalReputerWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting globalReputerWhitelist")
		}
	}

	// globalAdminWhitelist
	for _, address := range data.GlobalAdminWhitelist {
		if err := k.AddToGlobalAdminWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting globalAdminWhitelist")
		}
	}

	// topicCreatorWhitelist
	for _, address := range data.TopicCreatorWhitelist {
		if err := k.AddToTopicCreatorWhitelist(ctx, address); err != nil {
			return errors.Wrap(err, "error setting topicCreatorWhitelist")
		}
	}

	// topicWorkerWhitelist
	for _, topicAndActorId := range data.TopicWorkerWhitelist {
		if topicAndActorId != nil {
			if err := k.AddToTopicWorkerWhitelist(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicWorkerWhitelist")
			}
		}
	}

	// topicReputerWhitelist
	for _, topicAndActorId := range data.TopicReputerWhitelist {
		if topicAndActorId != nil {
			if err := k.AddToTopicReputerWhitelist(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicReputerWhitelist")
			}
		}
	}

	// topicWorkerWhitelistEnabled
	for _, topicId := range data.TopicWorkerWhitelistEnabled {
		if err := k.EnableTopicWorkerWhitelist(ctx, topicId); err != nil {
			return errors.Wrap(err, "error setting topicWorkerWhitelistEnabled")
		}
	}

	// topicReputerWhitelistEnabled
	for _, topicId := range data.TopicReputerWhitelistEnabled {
		if err := k.EnableTopicReputerWhitelist(ctx, topicId); err != nil {
			return errors.Wrap(err, "error setting topicReputerWhitelistEnabled")
		}
	}

	return nil
}

func (k *WhitelistsKeeper) ExportGenesis(ctx context.Context, data *types.GenesisState) error {
	coreTeamAddresses := make([]string, 0)
	coreTeamAddressesIter, err := k.whitelistAdmins.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate core team addresses")
	}
	for ; coreTeamAddressesIter.Valid(); coreTeamAddressesIter.Next() {
		key, err := coreTeamAddressesIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: coreTeamAddressesIter")
		}
		coreTeamAddresses = append(coreTeamAddresses, key)
	}
	data.CoreTeamAddresses = coreTeamAddresses

	whitelistAdmins := make([]string, 0)
	whitelistAdminsIter, err := k.whitelistAdmins.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate whitelist admins")
	}
	for ; whitelistAdminsIter.Valid(); whitelistAdminsIter.Next() {
		key, err := whitelistAdminsIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: whitelistAdminsIter")
		}
		whitelistAdmins = append(whitelistAdmins, key)
	}
	data.WhitelistAdmins = whitelistAdmins

	globalWhitelist := make([]string, 0)
	globalWhitelistIter, err := k.globalWhitelist.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate global whitelist")
	}
	for ; globalWhitelistIter.Valid(); globalWhitelistIter.Next() {
		key, err := globalWhitelistIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: globalWhitelistIter")
		}
		globalWhitelist = append(globalWhitelist, key)
	}
	data.GlobalWhitelist = globalWhitelist

	globalWorkerWhitelist := make([]string, 0)
	globalWorkerWhitelistIter, err := k.globalWorkerWhitelist.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate global worker whitelist")
	}
	for ; globalWorkerWhitelistIter.Valid(); globalWorkerWhitelistIter.Next() {
		key, err := globalWorkerWhitelistIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: globalWorkerWhitelistIter")
		}
		globalWorkerWhitelist = append(globalWorkerWhitelist, key)
	}
	data.GlobalWorkerWhitelist = globalWorkerWhitelist

	globalReputerWhitelist := make([]string, 0)
	globalReputerWhitelistIter, err := k.globalReputerWhitelist.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate global reputer whitelist")
	}
	for ; globalReputerWhitelistIter.Valid(); globalReputerWhitelistIter.Next() {
		key, err := globalReputerWhitelistIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: globalReputerWhitelistIter")
		}
		globalReputerWhitelist = append(globalReputerWhitelist, key)
	}
	data.GlobalReputerWhitelist = globalReputerWhitelist

	globalAdminWhitelist := make([]string, 0)
	globalAdminWhitelistIter, err := k.globalAdminWhitelist.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate global admin whitelist")
	}
	for ; globalAdminWhitelistIter.Valid(); globalAdminWhitelistIter.Next() {
		key, err := globalAdminWhitelistIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: globalAdminWhitelistIter")
		}
		globalAdminWhitelist = append(globalAdminWhitelist, key)
	}
	data.GlobalAdminWhitelist = globalAdminWhitelist

	topicCreatorWhitelist := make([]string, 0)
	topicCreatorWhitelistIter, err := k.topicCreatorWhitelist.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topic creator whitelist")
	}
	for ; topicCreatorWhitelistIter.Valid(); topicCreatorWhitelistIter.Next() {
		key, err := topicCreatorWhitelistIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: topicCreatorWhitelistIter")
		}
		topicCreatorWhitelist = append(topicCreatorWhitelist, key)
	}
	data.TopicCreatorWhitelist = topicCreatorWhitelist

	topicWorkerWhitelist := make([]*types.TopicAndActorId, 0)
	topicWorkerWhitelistIter, err := k.topicWorkerWhitelist.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topic worker whitelist")
	}
	for ; topicWorkerWhitelistIter.Valid(); topicWorkerWhitelistIter.Next() {
		keyValue, err := topicWorkerWhitelistIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: topicWorkerWhitelistIter")
		}
		topicWorkerWhitelist = append(topicWorkerWhitelist, &types.TopicAndActorId{
			TopicId: keyValue.K1(),
			ActorId: keyValue.K2(),
		})
	}
	data.TopicWorkerWhitelist = topicWorkerWhitelist

	topicReputerWhitelist := make([]*types.TopicAndActorId, 0)
	topicReputerWhitelistIter, err := k.topicReputerWhitelist.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topic reputer whitelist")
	}
	for ; topicReputerWhitelistIter.Valid(); topicReputerWhitelistIter.Next() {
		keyValue, err := topicReputerWhitelistIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: topicReputerWhitelistIter")
		}
		topicReputerWhitelist = append(topicReputerWhitelist, &types.TopicAndActorId{
			TopicId: keyValue.K1(),
			ActorId: keyValue.K2(),
		})
	}
	data.TopicReputerWhitelist = topicReputerWhitelist

	topicWorkerWhitelistEnabled := make([]uint64, 0)
	topicWorkerWhitelistEnabledIter, err := k.topicWorkerWhitelistEnabled.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topic whitelist enabled")
	}
	for ; topicWorkerWhitelistEnabledIter.Valid(); topicWorkerWhitelistEnabledIter.Next() {
		key, err := topicWorkerWhitelistEnabledIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: topicWhitelistEnabledIter")
		}
		topicWorkerWhitelistEnabled = append(topicWorkerWhitelistEnabled, key)
	}
	data.TopicWorkerWhitelistEnabled = topicWorkerWhitelistEnabled

	topicReputerWhitelistEnabled := make([]uint64, 0)
	topicReputerWhitelistEnabledIter, err := k.topicReputerWhitelistEnabled.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topic reputer whitelist enabled")
	}
	for ; topicReputerWhitelistEnabledIter.Valid(); topicReputerWhitelistEnabledIter.Next() {
		key, err := topicReputerWhitelistEnabledIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: topicReputerWhitelistEnabledIter")
		}
		topicReputerWhitelistEnabled = append(topicReputerWhitelistEnabled, key)
	}
	data.TopicReputerWhitelistEnabled = topicReputerWhitelistEnabled

	return nil
}
