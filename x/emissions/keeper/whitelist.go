package keeper

import (
	"context"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

/// SETTERS - Functions that update whitelists

func (k *Keeper) AddWhitelistAdmin(ctx context.Context, admin ActorId) error {
	if err := types.ValidateBech32(admin); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.whitelistAdmins.Set(ctx, admin)
}

func (k *Keeper) RemoveWhitelistAdmin(ctx context.Context, admin ActorId) error {
	if err := types.ValidateBech32(admin); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.whitelistAdmins.Remove(ctx, admin)
}

func (k *Keeper) EnableTopicWorkerWhitelist(ctx context.Context, topicId TopicId) error {
	return k.topicWorkerWhitelistEnabled.Set(ctx, topicId)
}

func (k *Keeper) DisableTopicWorkerWhitelist(ctx context.Context, topicId TopicId) error {
	return k.topicWorkerWhitelistEnabled.Remove(ctx, topicId)
}

func (k *Keeper) EnableTopicReputerWhitelist(ctx context.Context, topicId TopicId) error {
	return k.topicReputerWhitelistEnabled.Set(ctx, topicId)
}

func (k *Keeper) DisableTopicReputerWhitelist(ctx context.Context, topicId TopicId) error {
	return k.topicReputerWhitelistEnabled.Remove(ctx, topicId)
}

func (k *Keeper) AddToGlobalWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalWhitelist.Set(ctx, actor)
}

func (k *Keeper) RemoveFromGlobalWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalWhitelist.Remove(ctx, actor)
}

func (k *Keeper) AddToGlobalWorkerWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalWorkerWhitelist.Set(ctx, actor)
}

func (k *Keeper) RemoveFromGlobalWorkerWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalWorkerWhitelist.Remove(ctx, actor)
}

func (k *Keeper) AddToGlobalReputerWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalReputerWhitelist.Set(ctx, actor)
}

func (k *Keeper) RemoveFromGlobalReputerWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalReputerWhitelist.Remove(ctx, actor)
}

func (k *Keeper) AddToGlobalAdminWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalAdminWhitelist.Set(ctx, actor)
}

func (k *Keeper) RemoveFromGlobalAdminWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalAdminWhitelist.Remove(ctx, actor)
}

func (k *Keeper) AddToTopicCreatorWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.topicCreatorWhitelist.Set(ctx, actor)
}

func (k *Keeper) RemoveFromTopicCreatorWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.topicCreatorWhitelist.Remove(ctx, actor)
}

func (k *Keeper) AddToTopicWorkerWhitelist(ctx context.Context, topicId TopicId, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	key := collections.Join(topicId, actor)
	return k.topicWorkerWhitelist.Set(ctx, key)
}

func (k *Keeper) RemoveFromTopicWorkerWhitelist(ctx context.Context, topicId TopicId, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	key := collections.Join(topicId, actor)
	return k.topicWorkerWhitelist.Remove(ctx, key)
}

func (k *Keeper) AddToTopicReputerWhitelist(ctx context.Context, topicId TopicId, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	key := collections.Join(topicId, actor)
	return k.topicReputerWhitelist.Set(ctx, key)
}

func (k *Keeper) RemoveFromTopicReputerWhitelist(ctx context.Context, topicId TopicId, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	key := collections.Join(topicId, actor)
	return k.topicReputerWhitelist.Remove(ctx, key)
}

/// GETTERS - Functions that retrieve information about whitelists

// An actor is a whitelist admin if they are in the whitelistAdmins keyset
func (k Keeper) IsWhitelistAdmin(ctx context.Context, admin ActorId) (bool, error) {
	return k.whitelistAdmins.Has(ctx, admin)
}

// A topic is whitelist enabled if the topicWhitelistEnabled keyset has the topicId
func (k *Keeper) IsTopicWorkerWhitelistEnabled(ctx context.Context, topicId TopicId) (bool, error) {
	return k.topicWorkerWhitelistEnabled.Has(ctx, topicId)
}

// A topic is whitelist enabled if the topicWhitelistEnabled keyset has the topicId
func (k *Keeper) IsTopicReputerWhitelistEnabled(ctx context.Context, topicId TopicId) (bool, error) {
	return k.topicReputerWhitelistEnabled.Has(ctx, topicId)
}

func (k *Keeper) IsWhitelistedTopicCreator(ctx context.Context, actor ActorId) (bool, error) {
	return k.topicCreatorWhitelist.Has(ctx, actor)
}

func (k *Keeper) IsWhitelistedGlobalActor(ctx context.Context, actor ActorId) (bool, error) {
	return k.globalWhitelist.Has(ctx, actor)
}

func (k *Keeper) IsWhitelistedGlobalWorker(ctx context.Context, actor ActorId) (bool, error) {
	return k.globalWorkerWhitelist.Has(ctx, actor)
}

func (k *Keeper) IsWhitelistedGlobalReputer(ctx context.Context, actor ActorId) (bool, error) {
	return k.globalReputerWhitelist.Has(ctx, actor)
}

func (k *Keeper) IsWhitelistedGlobalAdmin(ctx context.Context, actor ActorId) (bool, error) {
	return k.globalAdminWhitelist.Has(ctx, actor)
}

func (k *Keeper) IsWhitelistedTopicWorker(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
	key := collections.Join(topicId, actor)
	return k.topicWorkerWhitelist.Has(ctx, key)
}

func (k *Keeper) IsWhitelistedTopicReputer(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
	key := collections.Join(topicId, actor)
	return k.topicReputerWhitelist.Has(ctx, key)
}

/// QUALIFIERS - Helper functions that are part of the same layer of abstraction as PERMISSIONS

// An actor is globally whitelisted if the GlobalWhitelistEnabled global parameter is false
// or (the parameter is true and the globalWhitelist keyset has the actor)
func (k *Keeper) IsEnabledGlobalActor(ctx context.Context, actor ActorId) (bool, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return false, err
	}
	if params.GlobalWhitelistEnabled {
		// If whitelist enabled check to see if actor is whitelisted
		return k.IsWhitelistedGlobalActor(ctx, actor)
	}
	return true, nil
}

func (k *Keeper) IsEnabledGlobalWorker(ctx context.Context, actor ActorId) (bool, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return false, err
	}
	if params.GlobalWorkerWhitelistEnabled {
		// If whitelist enabled check to see if actor is whitelisted
		return k.IsWhitelistedGlobalWorker(ctx, actor)
	}
	return true, nil
}

func (k *Keeper) IsEnabledGlobalReputer(ctx context.Context, actor ActorId) (bool, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return false, err
	}
	if params.GlobalReputerWhitelistEnabled {
		// If whitelist enabled check to see if actor is whitelisted
		return k.IsWhitelistedGlobalReputer(ctx, actor)
	}
	return true, nil
}

func (k *Keeper) IsEnabledGlobalAdmin(ctx context.Context, actor ActorId) (bool, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return false, err
	}
	if params.GlobalAdminWhitelistAppended {
		// If whitelist enabled check to see if actor is whitelisted
		return k.IsWhitelistedGlobalAdmin(ctx, actor)
	}
	return true, nil
}

// An actor is topic creator whitelisted if the TopicCreatorWhitelistEnabled global parameter is false
// or (the parameter is true and the topicCreatorWhitelist keyset has the actor)
func (k *Keeper) IsEnabledWhitelistedTopicCreator(ctx context.Context, actor ActorId) (bool, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return false, err
	}
	if params.TopicCreatorWhitelistEnabled {
		// If whitelist enabled check to see if actor is whitelisted
		return k.IsWhitelistedTopicCreator(ctx, actor)
	}
	return true, nil
}

// An actor is topic worker whitelisted if the topicWhitelistEnabled parameter is false
// or (the parameter is true and topicWorkerWhitelist keyset has the (topicId, actor) key)
func (k *Keeper) IsEnabledTopicWorker(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
	topicWhitelistEnabled, err := k.IsTopicWorkerWhitelistEnabled(ctx, topicId)
	if err != nil {
		return false, err
	}
	if topicWhitelistEnabled {
		// If whitelist enabled check to see if actor is whitelisted
		return k.IsWhitelistedTopicWorker(ctx, topicId, actor)
	}
	return true, nil
}

// An actor is topic reputer whitelisted if the topicWhitelistEnabled parameter is false
// or (the parameter is true and topicReputerWhitelist keyset has the (topicId, actor) key)
func (k *Keeper) IsEnabledTopicReputer(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
	topicWhitelistEnabled, err := k.IsTopicReputerWhitelistEnabled(ctx, topicId)
	if err != nil {
		return false, err
	}
	if topicWhitelistEnabled {
		// If whitelist enabled check to see if actor is whitelisted
		return k.IsWhitelistedTopicReputer(ctx, topicId, actor)
	}
	return true, nil
}

/// PERMISSIONS - Functions that determine if an actor has the ability to perform an action

// Whitelist admins can update global whitelists including adding/removing from the global actor and whitelist admin lists
func (k *Keeper) CanUpdateAllGlobalWhitelists(ctx context.Context, actor ActorId) (bool, error) {
	return k.IsWhitelistAdmin(ctx, actor)
}

// Whitelist admins and global admins can update global worker whitelist
func (k *Keeper) CanUpdateGlobalWorkerWhitelist(ctx context.Context, actor ActorId) (bool, error) {
	isWhitelistAdmin, err := k.IsEnabledGlobalAdmin(ctx, actor)
	if err != nil {
		return false, err
	}
	if isWhitelistAdmin {
		return true, nil
	}
	return k.IsWhitelistAdmin(ctx, actor)
}

// Whitelist admins and global admins can update global reputer whitelist
func (k *Keeper) CanUpdateGlobalReputerWhitelist(ctx context.Context, actor ActorId) (bool, error) {
	isWhitelistAdmin, err := k.IsEnabledGlobalAdmin(ctx, actor)
	if err != nil {
		return false, err
	}
	if isWhitelistAdmin {
		return true, nil
	}
	return k.IsWhitelistAdmin(ctx, actor)
}

// Whitelist admins can update global parameters
func (k *Keeper) CanUpdateParams(ctx context.Context, actor ActorId) (bool, error) {
	return k.IsWhitelistAdmin(ctx, actor)
}

// Whitelist admins and topic creators and global admins can update topic whitelists
// Updating the whitelist includes adding/removing from the whitelist and enabling/disabling the whitelist
func (k *Keeper) CanUpdateTopicWhitelist(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
	topic, err := k.GetTopic(ctx, topicId)
	if err != nil {
		return false, err
	}
	if topic.Creator == actor {
		return true, nil
	}
	isWhitelistAdmin, err := k.IsEnabledGlobalAdmin(ctx, actor)
	if err != nil {
		return false, err
	}
	if isWhitelistAdmin {
		return true, nil
	}
	return k.IsWhitelistAdmin(ctx, actor)
}

// An actor can create a topic if they are topic creator whitelisted
// or if they are globally whitelisted
func (k *Keeper) CanCreateTopic(ctx context.Context, actor ActorId) (bool, error) {
	isTopicCreator, err := k.IsEnabledWhitelistedTopicCreator(ctx, actor)
	if err != nil {
		return false, err
	}

	if isTopicCreator {
		return true, nil
	}

	return k.IsEnabledGlobalActor(ctx, actor)
}

// An actor can submit a worker payload if they are topic worker whitelisted, global actor, or global worker
func (k *Keeper) CanSubmitWorkerPayload(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
	isEnabledTopicWorker, err := k.IsEnabledTopicWorker(ctx, topicId, actor)
	if err != nil {
		return false, err
	}
	if isEnabledTopicWorker {
		return true, nil
	}
	isEnabledGlobalWorker, err := k.IsEnabledGlobalWorker(ctx, actor)
	if err != nil {
		return false, err
	}
	if isEnabledGlobalWorker {
		return true, nil
	}
	return k.IsEnabledGlobalActor(ctx, actor)
}

// An actor can submit a reputer payload if they are topic reputer whitelisted, global actor, or global reputer
func (k *Keeper) CanSubmitReputerPayload(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
	isEnabledTopicReputer, err := k.IsEnabledTopicReputer(ctx, topicId, actor)
	if err != nil {
		return false, err
	}
	if isEnabledTopicReputer {
		return true, nil
	}
	isEnabledGlobalReputer, err := k.IsEnabledGlobalReputer(ctx, actor)
	if err != nil {
		return false, err
	}
	if isEnabledGlobalReputer {
		return true, nil
	}
	return k.IsEnabledGlobalActor(ctx, actor)
}

// An actor can add reputer stake if they are topic reputer whitelisted, global actor, or global reputer
func (k *Keeper) CanAddReputerStake(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
	isEnabledTopicReputer, err := k.IsEnabledTopicReputer(ctx, topicId, actor)
	if err != nil {
		return false, err
	}
	if isEnabledTopicReputer {
		return true, nil
	}
	isEnabledGlobalReputer, err := k.IsEnabledGlobalReputer(ctx, actor)
	if err != nil {
		return false, err
	}
	if isEnabledGlobalReputer {
		return true, nil
	}
	return k.IsEnabledGlobalActor(ctx, actor)
}

// Whitelist admins can update the topic creator whitelist
func (k *Keeper) CanUpdateTopicCreatorWhitelist(ctx context.Context, actor ActorId) (bool, error) {
	return k.IsWhitelistAdmin(ctx, actor)
}
