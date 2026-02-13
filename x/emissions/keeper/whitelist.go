package keeper

import (
	"context"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func NewWhitelistsKeeper(
	sb *collections.SchemaBuilder,
	paramsKeeper *ParamsKeeper,
	topicKeeper *TopicKeeper,
) *WhitelistsKeeper {
	return &WhitelistsKeeper{
		whitelistAdmins:              collections.NewKeySet(sb, types.WhitelistAdminsKey, "whitelist_admins", collections.StringKey),
		globalWhitelist:              collections.NewKeySet(sb, types.GlobalWhitelistKey, "global_whitelist", collections.StringKey),
		globalWorkerWhitelist:        collections.NewKeySet(sb, types.GlobalWorkerWhitelistKey, "global_worker_whitelist", collections.StringKey),
		globalReputerWhitelist:       collections.NewKeySet(sb, types.GlobalReputerWhitelistKey, "global_reputer_whitelist", collections.StringKey),
		globalAdminWhitelist:         collections.NewKeySet(sb, types.GlobalAdminWhitelistKey, "global_admin_whitelist", collections.StringKey),
		topicCreatorWhitelist:        collections.NewKeySet(sb, types.TopicCreatorWhitelistKey, "topic_creator_whitelist", collections.StringKey),
		topicWorkerWhitelist:         collections.NewKeySet(sb, types.TopicWorkerWhitelistKey, "topic_worker_whitelist", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
		topicReputerWhitelist:        collections.NewKeySet(sb, types.TopicReputerWhitelistKey, "topic_reputer_whitelist", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
		topicWorkerWhitelistEnabled:  collections.NewKeySet(sb, types.TopicWorkerWhitelistEnabledKey, "topic_worker_whitelist_enabled", collections.Uint64Key),
		topicReputerWhitelistEnabled: collections.NewKeySet(sb, types.TopicReputerWhitelistEnabledKey, "topic_reputer_whitelist_enabled", collections.Uint64Key),
		paramsKeeper:                 paramsKeeper,
		topicKeeper:                  topicKeeper,
	}
}

type WhitelistsKeeper struct {
	// can change parameters and decide who can be included in any whitelist
	whitelistAdmins collections.KeySet[ActorId]
	// actors who can create topics and participate in all topics as workers and reputers
	globalWhitelist collections.KeySet[ActorId]
	// global workers can participate in all topics as workers
	globalWorkerWhitelist collections.KeySet[ActorId]
	// global reputers can participate in all topics as reputers
	globalReputerWhitelist collections.KeySet[ActorId]
	// global admins can add global workers, global reputers, topic actors, topic reputers
	globalAdminWhitelist collections.KeySet[ActorId]
	// whitelist of actors who can create topics
	topicCreatorWhitelist collections.KeySet[ActorId]
	// map from topic id to whitelist of workers
	topicWorkerWhitelist collections.KeySet[collections.Pair[TopicId, ActorId]]
	// map from topic id to whitelist of reputers
	topicReputerWhitelist collections.KeySet[collections.Pair[TopicId, ActorId]]
	// map from topic id to whether the whitelist is enabled for topic workers
	topicWorkerWhitelistEnabled collections.KeySet[TopicId]
	// map from topic id to whether the whitelist is enabled for topic reputers
	topicReputerWhitelistEnabled collections.KeySet[TopicId]
	// params keeper
	paramsKeeper *ParamsKeeper
	// topic keeper
	topicKeeper *TopicKeeper
}

// SETTERS - Functions that update whitelists

func (k *WhitelistsKeeper) AddWhitelistAdmin(ctx context.Context, admin ActorId) error {
	if err := types.ValidateBech32(admin); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.whitelistAdmins.Set(ctx, admin)
}

func (k *WhitelistsKeeper) RemoveWhitelistAdmin(ctx context.Context, admin ActorId) error {
	if err := types.ValidateBech32(admin); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.whitelistAdmins.Remove(ctx, admin)
}

func (k *WhitelistsKeeper) EnableTopicWorkerWhitelist(ctx context.Context, topicId TopicId) error {
	return k.topicWorkerWhitelistEnabled.Set(ctx, topicId)
}

func (k *WhitelistsKeeper) DisableTopicWorkerWhitelist(ctx context.Context, topicId TopicId) error {
	return k.topicWorkerWhitelistEnabled.Remove(ctx, topicId)
}

func (k *WhitelistsKeeper) EnableTopicReputerWhitelist(ctx context.Context, topicId TopicId) error {
	return k.topicReputerWhitelistEnabled.Set(ctx, topicId)
}

func (k *WhitelistsKeeper) DisableTopicReputerWhitelist(ctx context.Context, topicId TopicId) error {
	return k.topicReputerWhitelistEnabled.Remove(ctx, topicId)
}

func (k *WhitelistsKeeper) AddToGlobalWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalWhitelist.Set(ctx, actor)
}

func (k *WhitelistsKeeper) RemoveFromGlobalWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalWhitelist.Remove(ctx, actor)
}

func (k *WhitelistsKeeper) AddToGlobalWorkerWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalWorkerWhitelist.Set(ctx, actor)
}

func (k *WhitelistsKeeper) RemoveFromGlobalWorkerWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalWorkerWhitelist.Remove(ctx, actor)
}

func (k *WhitelistsKeeper) AddToGlobalReputerWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalReputerWhitelist.Set(ctx, actor)
}

func (k *WhitelistsKeeper) RemoveFromGlobalReputerWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalReputerWhitelist.Remove(ctx, actor)
}

func (k *WhitelistsKeeper) AddToGlobalAdminWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalAdminWhitelist.Set(ctx, actor)
}

func (k *WhitelistsKeeper) RemoveFromGlobalAdminWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.globalAdminWhitelist.Remove(ctx, actor)
}

func (k *WhitelistsKeeper) AddToTopicCreatorWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.topicCreatorWhitelist.Set(ctx, actor)
}

func (k *WhitelistsKeeper) RemoveFromTopicCreatorWhitelist(ctx context.Context, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	return k.topicCreatorWhitelist.Remove(ctx, actor)
}

func (k *WhitelistsKeeper) AddToTopicWorkerWhitelist(ctx context.Context, topicId TopicId, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	key := collections.Join(topicId, actor)
	return k.topicWorkerWhitelist.Set(ctx, key)
}

func (k *WhitelistsKeeper) RemoveFromTopicWorkerWhitelist(ctx context.Context, topicId TopicId, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	key := collections.Join(topicId, actor)
	return k.topicWorkerWhitelist.Remove(ctx, key)
}

func (k *WhitelistsKeeper) AddToTopicReputerWhitelist(ctx context.Context, topicId TopicId, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	key := collections.Join(topicId, actor)
	return k.topicReputerWhitelist.Set(ctx, key)
}

func (k *WhitelistsKeeper) RemoveFromTopicReputerWhitelist(ctx context.Context, topicId TopicId, actor ActorId) error {
	if err := types.ValidateBech32(actor); err != nil {
		return errorsmod.Wrap(err, "error validating admin id")
	}
	key := collections.Join(topicId, actor)
	return k.topicReputerWhitelist.Remove(ctx, key)
}

// / GETTERS - Functions that retrieve information about whitelists

// An actor is a whitelist admin if they are in the whitelistAdmins keyset
func (k *WhitelistsKeeper) IsWhitelistAdmin(ctx context.Context, admin ActorId) (bool, error) {
	return k.whitelistAdmins.Has(ctx, admin)
}

// A topic is whitelist enabled if the topicWhitelistEnabled keyset has the topicId
func (k *WhitelistsKeeper) IsTopicWorkerWhitelistEnabled(ctx context.Context, topicId TopicId) (bool, error) {
	return k.topicWorkerWhitelistEnabled.Has(ctx, topicId)
}

// A topic is whitelist enabled if the topicWhitelistEnabled keyset has the topicId
func (k *WhitelistsKeeper) IsTopicReputerWhitelistEnabled(ctx context.Context, topicId TopicId) (bool, error) {
	return k.topicReputerWhitelistEnabled.Has(ctx, topicId)
}

func (k *WhitelistsKeeper) IsWhitelistedTopicCreator(ctx context.Context, actor ActorId) (bool, error) {
	return k.topicCreatorWhitelist.Has(ctx, actor)
}

func (k *WhitelistsKeeper) IsWhitelistedGlobalActor(ctx context.Context, actor ActorId) (bool, error) {
	return k.globalWhitelist.Has(ctx, actor)
}

func (k *WhitelistsKeeper) IsWhitelistedGlobalWorker(ctx context.Context, actor ActorId) (bool, error) {
	return k.globalWorkerWhitelist.Has(ctx, actor)
}

func (k *WhitelistsKeeper) IsWhitelistedGlobalReputer(ctx context.Context, actor ActorId) (bool, error) {
	return k.globalReputerWhitelist.Has(ctx, actor)
}

func (k *WhitelistsKeeper) IsWhitelistedGlobalAdmin(ctx context.Context, actor ActorId) (bool, error) {
	return k.globalAdminWhitelist.Has(ctx, actor)
}

func (k *WhitelistsKeeper) IsWhitelistedTopicWorker(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
	key := collections.Join(topicId, actor)
	return k.topicWorkerWhitelist.Has(ctx, key)
}

func (k *WhitelistsKeeper) IsWhitelistedTopicReputer(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
	key := collections.Join(topicId, actor)
	return k.topicReputerWhitelist.Has(ctx, key)
}

// / QUALIFIERS - Helper functions that are part of the same layer of abstraction as PERMISSIONS

// An actor is globally whitelisted if the GlobalWhitelistEnabled global parameter is false
// or (the parameter is true and the globalWhitelist keyset has the actor)
func (k *WhitelistsKeeper) IsEnabledGlobalActor(ctx context.Context, actor ActorId) (bool, error) {
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return false, err
	}
	if params.GlobalWhitelistEnabled {
		// If whitelist enabled check to see if actor is whitelisted
		return k.IsWhitelistedGlobalActor(ctx, actor)
	}
	return true, nil
}

func (k *WhitelistsKeeper) IsEnabledGlobalWorker(ctx context.Context, actor ActorId) (bool, error) {
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return false, err
	}
	if params.GlobalWorkerWhitelistEnabled {
		// If whitelist enabled check to see if actor is whitelisted
		return k.IsWhitelistedGlobalWorker(ctx, actor)
	}
	return true, nil
}

func (k *WhitelistsKeeper) IsEnabledGlobalReputer(ctx context.Context, actor ActorId) (bool, error) {
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return false, err
	}
	if params.GlobalReputerWhitelistEnabled {
		// If whitelist enabled check to see if actor is whitelisted
		return k.IsWhitelistedGlobalReputer(ctx, actor)
	}
	return true, nil
}

func (k *WhitelistsKeeper) IsEnabledGlobalAdmin(ctx context.Context, actor ActorId) (bool, error) {
	params, err := k.paramsKeeper.GetParams(ctx)
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
func (k *WhitelistsKeeper) IsEnabledWhitelistedTopicCreator(ctx context.Context, actor ActorId) (bool, error) {
	params, err := k.paramsKeeper.GetParams(ctx)
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
func (k *WhitelistsKeeper) IsEnabledTopicWorker(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
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
func (k *WhitelistsKeeper) IsEnabledTopicReputer(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
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

// / PERMISSIONS - Functions that determine if an actor has the ability to perform an action

// Whitelist admins can update global whitelists including adding/removing from the global actor and whitelist admin lists
func (k *WhitelistsKeeper) CanUpdateAllGlobalWhitelists(ctx context.Context, actor ActorId) (bool, error) {
	return k.IsWhitelistAdmin(ctx, actor)
}

// Whitelist admins and global admins can update global worker whitelist
func (k *WhitelistsKeeper) CanUpdateGlobalWorkerWhitelist(ctx context.Context, actor ActorId) (bool, error) {
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
func (k *WhitelistsKeeper) CanUpdateGlobalReputerWhitelist(ctx context.Context, actor ActorId) (bool, error) {
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
func (k *WhitelistsKeeper) CanUpdateParams(ctx context.Context, actor ActorId) (bool, error) {
	return k.IsWhitelistAdmin(ctx, actor)
}

// Whitelist admins and topic creators and global admins can update topic whitelists
// Updating the whitelist includes adding/removing from the whitelist and enabling/disabling the whitelist
func (k *WhitelistsKeeper) CanUpdateTopicWhitelist(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
	topic, err := k.topicKeeper.GetTopic(ctx, topicId)
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
func (k *WhitelistsKeeper) CanCreateTopic(ctx context.Context, actor ActorId) (bool, error) {
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
func (k *WhitelistsKeeper) CanSubmitWorkerPayload(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
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
func (k *WhitelistsKeeper) CanSubmitReputerPayload(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
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
func (k *WhitelistsKeeper) CanAddReputerStake(ctx context.Context, topicId TopicId, actor ActorId) (bool, error) {
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
func (k *WhitelistsKeeper) CanUpdateTopicCreatorWhitelist(ctx context.Context, actor ActorId) (bool, error) {
	return k.IsWhitelistAdmin(ctx, actor)
}
