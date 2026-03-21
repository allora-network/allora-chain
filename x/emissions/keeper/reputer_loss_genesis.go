package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// InitGenesis initializes the ReputerLossKeeper state from a genesis state.
func (k *ReputerLossKeeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	// TopicReputers []*TopicAndActorId
	for _, topicAndActorId := range data.TopicReputers {
		if topicAndActorId != nil {
			if err := types.ValidateTopicId(topicAndActorId.TopicId); err != nil {
				return errors.Wrap(err, "error setting topicReputers")
			}
			if err := types.ValidateBech32(topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicReputers")
			}
			if err := k.SetTopicReputer(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting topicReputers")
			}
		}
	}

	// Reputers []*LibP2PKeyAndOffchainNode
	for _, libP2PKeyAndOffchainNode := range data.Reputers {
		if libP2PKeyAndOffchainNode != nil {
			if libP2PKeyAndOffchainNode.OffchainNode == nil {
				return errors.Wrap(types.ErrInvalidValue, "reputer info cannot be nil")
			}
			if err := libP2PKeyAndOffchainNode.OffchainNode.Validate(); err != nil {
				return errors.Wrap(err, "reputer info validation failed")
			}
			if err := k.reputers.Set(
				ctx,
				libP2PKeyAndOffchainNode.LibP2PKey,
				*libP2PKeyAndOffchainNode.OffchainNode); err != nil {
				return errors.Wrap(err, "error setting reputers")
			}
		}
	}

	// AllLossBundles []*TopicIdBlockHeightReputerValueBundles
	for _, topicIdBlockHeightReputerValueBundles := range data.AllLossBundles {
		if topicIdBlockHeightReputerValueBundles != nil {
			if topicIdBlockHeightReputerValueBundles.ReputerValueBundles == nil {
				return errors.Wrap(types.ErrInvalidValue, "all loss bundles cannot be nil")
			}
			if err := topicIdBlockHeightReputerValueBundles.ReputerValueBundles.Validate(); err != nil {
				return errors.Wrap(err, "reputer value bundles validation failed")
			}
			if err := k.allLossBundles.Set(ctx,
				collections.Join(topicIdBlockHeightReputerValueBundles.TopicId, topicIdBlockHeightReputerValueBundles.BlockHeight),
				*topicIdBlockHeightReputerValueBundles.ReputerValueBundles); err != nil {
				return errors.Wrap(err, "error setting allLossBundles")
			}
		}
	}

	// NetworkLossBundles []*TopicIdBlockHeightValueBundles
	for _, topicIdBlockHeightValueBundles := range data.NetworkLossBundles {
		if topicIdBlockHeightValueBundles != nil {
			if topicIdBlockHeightValueBundles.ValueBundle == nil {
				return errors.Wrap(types.ErrInvalidValue, "network loss bundle cannot be nil")
			}
			if err := topicIdBlockHeightValueBundles.ValueBundle.Validate(); err != nil {
				return errors.Wrap(err, "value bundle validation failed")
			}
			if err := k.networkLossBundles.Set(ctx,
				collections.Join(topicIdBlockHeightValueBundles.TopicId, topicIdBlockHeightValueBundles.BlockHeight),
				*topicIdBlockHeightValueBundles.ValueBundle); err != nil {
				return errors.Wrap(err, "error setting networkLossBundles")
			}
		}
	}

	// ActiveReputers []*TopicAndActorId
	for _, topicAndActorId := range data.ActiveReputers {
		if topicAndActorId != nil {
			if err := k.AddActiveReputer(ctx, topicAndActorId.TopicId, topicAndActorId.ActorId); err != nil {
				return errors.Wrap(err, "error setting activeReputers")
			}
		}
	}

	// LossBundles []*TopicIdReputerReputerValueBundle
	for _, bundle := range data.LossBundles {
		if bundle != nil {
			if bundle.ReputerValueBundle == nil {
				return errors.Wrap(types.ErrInvalidValue, "loss bundle cannot be nil")
			}
			key := collections.Join(bundle.TopicId, bundle.Reputer)
			if err := k.lossBundles.Set(ctx, key, *bundle.ReputerValueBundle); err != nil {
				return errors.Wrap(err, "error setting loss bundle")
			}
		}
	}

	return nil
}

// ExportGenesis exports the ReputerLossKeeper state into a genesis state.
func (k *ReputerLossKeeper) ExportGenesis(ctx context.Context, data *types.GenesisState) error {
	// topicReputers
	topicReputers := make([]*types.TopicAndActorId, 0)
	topicReputersIter, err := k.topicReputers.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topic reputers")
	}
	for ; topicReputersIter.Valid(); topicReputersIter.Next() {
		key, err := topicReputersIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: topicReputersIter")
		}
		topicIdAndActorId := types.TopicAndActorId{
			TopicId: key.K1(),
			ActorId: key.K2(),
		}
		topicReputers = append(topicReputers, &topicIdAndActorId)
	}
	data.TopicReputers = topicReputers

	// reputers
	reputers := make([]*types.LibP2PKeyAndOffchainNode, 0)
	reputersIter, err := k.reputers.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate reputers")
	}
	for ; reputersIter.Valid(); reputersIter.Next() {
		keyValue, err := reputersIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: reputersIter")
		}
		libP2PKeyAndOffchainNode := types.LibP2PKeyAndOffchainNode{
			LibP2PKey:    keyValue.Key,
			OffchainNode: &keyValue.Value,
		}
		reputers = append(reputers, &libP2PKeyAndOffchainNode)
	}
	data.Reputers = reputers

	// allLossBundles
	allLossBundles := make([]*types.TopicIdBlockHeightReputerValueBundles, 0)
	allLossBundlesIter, err := k.allLossBundles.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate all loss bundles")
	}
	for ; allLossBundlesIter.Valid(); allLossBundlesIter.Next() {
		keyValue, err := allLossBundlesIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: allLossBundlesIter")
		}
		value := keyValue.Value
		topicIdBlockHeightValueBundles := types.TopicIdBlockHeightReputerValueBundles{
			TopicId:             keyValue.Key.K1(),
			BlockHeight:         keyValue.Key.K2(),
			ReputerValueBundles: &value,
		}
		allLossBundles = append(allLossBundles, &topicIdBlockHeightValueBundles)
	}
	data.AllLossBundles = allLossBundles

	// networkLossBundles
	networkLossBundles := make([]*types.TopicIdBlockHeightValueBundles, 0)
	networkLossBundlesIter, err := k.networkLossBundles.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate network loss bundles")
	}
	for ; networkLossBundlesIter.Valid(); networkLossBundlesIter.Next() {
		keyValue, err := networkLossBundlesIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: networkLossBundlesIter")
		}
		value := keyValue.Value
		topicIdBlockHeightValueBundles := types.TopicIdBlockHeightValueBundles{
			TopicId:     keyValue.Key.K1(),
			BlockHeight: keyValue.Key.K2(),
			ValueBundle: &value,
		}
		networkLossBundles = append(networkLossBundles, &topicIdBlockHeightValueBundles)
	}
	data.NetworkLossBundles = networkLossBundles

	// activeReputers
	activeReputers := make([]*types.TopicAndActorId, 0)
	activeReputersIter, err := k.activeReputers.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate active reputers")
	}
	for ; activeReputersIter.Valid(); activeReputersIter.Next() {
		key, err := activeReputersIter.Key()
		if err != nil {
			return errors.Wrap(err, "failed to get key: activeReputersIter")
		}
		activeReputers = append(activeReputers, &types.TopicAndActorId{
			TopicId: key.K1(),
			ActorId: key.K2(),
		})
	}
	data.ActiveReputers = activeReputers

	// lossBundles
	lossBundles := make([]*types.TopicIdReputerReputerValueBundle, 0)
	lossBundlesIter, err := k.lossBundles.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate loss bundles")
	}
	for ; lossBundlesIter.Valid(); lossBundlesIter.Next() {
		keyValue, err := lossBundlesIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key-value: lossBundlesIter")
		}
		lossBundles = append(lossBundles, &types.TopicIdReputerReputerValueBundle{
			TopicId:            keyValue.Key.K1(),
			Reputer:            keyValue.Key.K2(),
			ReputerValueBundle: &keyValue.Value,
		})
	}
	data.LossBundles = lossBundles

	return nil
}
