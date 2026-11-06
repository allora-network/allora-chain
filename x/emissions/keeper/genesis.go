package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// InitGenesis initializes the module state from a genesis state.
func (k *Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	// ensure the module account exists
	stakingModuleAccount := k.authKeeper.GetModuleAccount(ctx, types.AlloraStakingAccountName)
	k.authKeeper.SetModuleAccount(ctx, stakingModuleAccount)
	alloraRewardsModuleAccount := k.authKeeper.GetModuleAccount(ctx, types.AlloraRewardsAccountName)
	k.authKeeper.SetModuleAccount(ctx, alloraRewardsModuleAccount)
	alloraPendingRewardsModuleAccount := k.authKeeper.GetModuleAccount(ctx, types.AlloraPendingRewardForDelegatorAccountName)
	k.authKeeper.SetModuleAccount(ctx, alloraPendingRewardsModuleAccount)

	if err := k.paramsKeeper.InitGenesis(ctx, data); err != nil {
		return errors.Wrap(err, "params keeper init genesis")
	}

	if err := k.topicKeeper.InitGenesis(ctx, data); err != nil {
		return errors.Wrap(err, "topic keeper init genesis")
	}

	if err := k.workerKeeper.InitGenesis(ctx, data); err != nil {
		return errors.Wrap(err, "worker keeper init genesis")
	}

	if err := k.reputerLossKeeper.InitGenesis(ctx, data); err != nil {
		return errors.Wrap(err, "reputer loss keeper init genesis")
	}

	if err := k.scoresKeeper.InitGenesis(ctx, data); err != nil {
		return errors.Wrap(err, "scores keeper init genesis")
	}

	if err := k.stakingKeeper.InitGenesis(ctx, data); err != nil {
		return errors.Wrap(err, "staking keeper init genesis")
	}

	if err := k.nonceKeeper.InitGenesis(ctx, data); err != nil {
		return errors.Wrap(err, "nonce keeper init genesis")
	}

	if err := k.regretsKeeper.InitGenesis(ctx, data); err != nil {
		return errors.Wrap(err, "regrets keeper init genesis")
	}

	if err := k.weightsKeeper.InitGenesis(ctx, data); err != nil {
		return errors.Wrap(err, "weights keeper init genesis")
	}

	if err := k.whitelistsKeeper.InitGenesis(ctx, data); err != nil {
		return errors.Wrap(err, "whitelists keeper init genesis")
	}

	if err := k.initGenesisKeeperFields(ctx, data); err != nil {
		return errors.Wrap(err, "keeper-level fields init genesis")
	}

	return nil
}

// initGenesisKeeperFields handles fields that live directly on Keeper (not on any sub-keeper).
func (k *Keeper) initGenesisKeeperFields(ctx context.Context, data *types.GenesisState) error {
	// CountInfererInclusionsInTopicActiveSet
	for _, topicIdInfererCount := range data.CountInfererInclusionsInTopicActiveSet {
		if topicIdInfererCount != nil {
			if err := k.countInfererInclusionsInTopicActiveSet.Set(
				ctx,
				collections.Join(topicIdInfererCount.TopicId, topicIdInfererCount.ActorId),
				topicIdInfererCount.Uint64,
			); err != nil {
				return errors.Wrap(err, "error setting countInfererInclusionsInTopicActiveSet")
			}
		}
	}

	// CountForecasterInclusionsInTopicActiveSet
	for _, topicIdForecasterCount := range data.CountForecasterInclusionsInTopicActiveSet {
		if topicIdForecasterCount != nil {
			if err := k.countForecasterInclusionsInTopicActiveSet.Set(
				ctx,
				collections.Join(topicIdForecasterCount.TopicId, topicIdForecasterCount.ActorId),
				topicIdForecasterCount.Uint64,
			); err != nil {
				return errors.Wrap(err, "error setting countForecasterInclusionsInTopicActiveSet")
			}
		}
	}

	// RewardCurrentBlockEmission
	if data.RewardCurrentBlockEmission.GT(cosmosMath.ZeroInt()) {
		if err := k.SetRewardCurrentBlockEmission(ctx, data.RewardCurrentBlockEmission); err != nil {
			return errors.Wrap(err, "error setting RewardCurrentBlockEmission")
		}
	} else {
		if err := k.SetRewardCurrentBlockEmission(ctx, cosmosMath.ZeroInt()); err != nil {
			return errors.Wrap(err, "error setting RewardCurrentBlockEmission to zero int")
		}
	}

	// NetworkInferenceBundle
	for _, networkInference := range data.NetworkInferenceBundle {
		if networkInference != nil {
			if err := k.InsertNetworkInferenceBundle(ctx, networkInference.TopicId, networkInference.BlockHeight, *networkInference.NetworkInferenceBundle); err != nil {
				return errors.Wrap(err, "error setting network inference bundle")
			}
		}
	}

	// OutlierResistantNetworkInferenceBundle
	for _, outlierResistantNetworkInference := range data.OutlierResistantNetworkInferenceBundle {
		if outlierResistantNetworkInference != nil {
			if err := k.InsertOutlierResistantNetworkInferenceBundle(ctx, outlierResistantNetworkInference.TopicId, outlierResistantNetworkInference.BlockHeight, *outlierResistantNetworkInference.NetworkInferenceBundle); err != nil {
				return errors.Wrap(err, "error setting outlier resistant network inference")
			}
		}
	}

	// TopicLastEpochNonces
	for _, topicLastEpochNonce := range data.TopicLastEpochNonces {
		if topicLastEpochNonce != nil {
			if err := k.topicLastEpochNonce.Set(ctx, topicLastEpochNonce.TopicId, topicLastEpochNonce.Nonce); err != nil {
				return errors.Wrap(err, "error setting topicLastEpochNonce")
			}
		}
	}

	return nil
}

// ExportGenesis exports the module state to a genesis state.
func (k *Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	data := types.NewGenesisState()

	if err := k.paramsKeeper.ExportGenesis(ctx, data); err != nil {
		return nil, errors.Wrap(err, "params keeper export genesis")
	}

	if err := k.topicKeeper.ExportGenesis(ctx, data); err != nil {
		return nil, errors.Wrap(err, "topic keeper export genesis")
	}

	if err := k.workerKeeper.ExportGenesis(ctx, data); err != nil {
		return nil, errors.Wrap(err, "worker keeper export genesis")
	}

	if err := k.reputerLossKeeper.ExportGenesis(ctx, data); err != nil {
		return nil, errors.Wrap(err, "reputer loss keeper export genesis")
	}

	if err := k.scoresKeeper.ExportGenesis(ctx, data); err != nil {
		return nil, errors.Wrap(err, "scores keeper export genesis")
	}

	if err := k.stakingKeeper.ExportGenesis(ctx, data); err != nil {
		return nil, errors.Wrap(err, "staking keeper export genesis")
	}

	if err := k.nonceKeeper.ExportGenesis(ctx, data); err != nil {
		return nil, errors.Wrap(err, "nonce keeper export genesis")
	}

	if err := k.regretsKeeper.ExportGenesis(ctx, data); err != nil {
		return nil, errors.Wrap(err, "regrets keeper export genesis")
	}

	if err := k.weightsKeeper.ExportGenesis(ctx, data); err != nil {
		return nil, errors.Wrap(err, "weights keeper export genesis")
	}

	if err := k.whitelistsKeeper.ExportGenesis(ctx, data); err != nil {
		return nil, errors.Wrap(err, "whitelists keeper export genesis")
	}

	if err := k.exportGenesisKeeperFields(ctx, data); err != nil {
		return nil, errors.Wrap(err, "keeper-level fields export genesis")
	}

	return data, nil
}

// exportGenesisKeeperFields handles fields that live directly on Keeper (not on any sub-keeper).
func (k *Keeper) exportGenesisKeeperFields(ctx context.Context, data *types.GenesisState) error {
	// CountInfererInclusionsInTopicActiveSet
	countInfererInclusionsInTopicActiveSet := make([]*types.TopicIdActorIdUint64, 0)
	countInfererInclusionsInTopicIter, err := k.countInfererInclusionsInTopicActiveSet.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate count inferer inclusions in topic")
	}
	for ; countInfererInclusionsInTopicIter.Valid(); countInfererInclusionsInTopicIter.Next() {
		keyValue, err := countInfererInclusionsInTopicIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: countInfererInclusionsInTopicIter")
		}
		countInfererInclusionsInTopicActiveSet = append(countInfererInclusionsInTopicActiveSet, &types.TopicIdActorIdUint64{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Uint64:  keyValue.Value,
		})
	}
	data.CountInfererInclusionsInTopicActiveSet = countInfererInclusionsInTopicActiveSet

	// CountForecasterInclusionsInTopicActiveSet
	countForecasterInclusionsInTopicActiveSet := make([]*types.TopicIdActorIdUint64, 0)
	countForecasterInclusionsInTopicIter, err := k.countForecasterInclusionsInTopicActiveSet.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate count forecaster inclusions in topic")
	}
	for ; countForecasterInclusionsInTopicIter.Valid(); countForecasterInclusionsInTopicIter.Next() {
		keyValue, err := countForecasterInclusionsInTopicIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: countForecasterInclusionsInTopicIter")
		}
		countForecasterInclusionsInTopicActiveSet = append(countForecasterInclusionsInTopicActiveSet, &types.TopicIdActorIdUint64{
			TopicId: keyValue.Key.K1(),
			ActorId: keyValue.Key.K2(),
			Uint64:  keyValue.Value,
		})
	}
	data.CountForecasterInclusionsInTopicActiveSet = countForecasterInclusionsInTopicActiveSet

	// RewardCurrentBlockEmission
	rewardCurrentBlockEmission, err := k.GetRewardCurrentBlockEmission(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get reward current block emission")
	}
	data.RewardCurrentBlockEmission = rewardCurrentBlockEmission

	// Export network inferences
	networkInferenceBundle := make([]*types.TopicIdBlockHeightNetworkInferenceBundles, 0)

	networkInferencesBundleIter, err := k.networkInferenceBundle.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate network inference bundle")
	}
	for ; networkInferencesBundleIter.Valid(); networkInferencesBundleIter.Next() {
		keyValue, err := networkInferencesBundleIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: networkInferencesBundleIter")
		}
		networkInferenceBundle = append(networkInferenceBundle, &types.TopicIdBlockHeightNetworkInferenceBundles{
			TopicId:                keyValue.Key.K1(),
			BlockHeight:            keyValue.Key.K2(),
			NetworkInferenceBundle: &keyValue.Value,
		})
	}

	data.NetworkInferenceBundle = networkInferenceBundle

	// Outlier resistant network inferences
	outlierResistantNetworkInferenceBundle := make([]*types.TopicIdBlockHeightNetworkInferenceBundles, 0)

	outlierResistantNetworkInferenceBundleIter, err := k.outlierResistantNetworkInferenceBundle.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate outlier resistant network inference bundle")
	}
	for ; outlierResistantNetworkInferenceBundleIter.Valid(); outlierResistantNetworkInferenceBundleIter.Next() {
		keyValue, err := outlierResistantNetworkInferenceBundleIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: outlierResistantNetworkInferenceBundleIter")
		}
		outlierResistantNetworkInferenceBundle = append(outlierResistantNetworkInferenceBundle, &types.TopicIdBlockHeightNetworkInferenceBundles{
			TopicId:                keyValue.Key.K1(),
			BlockHeight:            keyValue.Key.K2(),
			NetworkInferenceBundle: &keyValue.Value,
		})
	}

	data.OutlierResistantNetworkInferenceBundle = outlierResistantNetworkInferenceBundle

	// TopicLastEpochNonces
	topicLastEpochNonces := make([]*types.TopicIdAndEpochNonce, 0)
	topicLastEpochNonceIter, err := k.topicLastEpochNonce.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate topic last epoch nonces")
	}
	for ; topicLastEpochNonceIter.Valid(); topicLastEpochNonceIter.Next() {
		keyValue, err := topicLastEpochNonceIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: topicLastEpochNonceIter")
		}
		topicLastEpochNonces = append(topicLastEpochNonces, &types.TopicIdAndEpochNonce{
			TopicId: keyValue.Key,
			Nonce:   keyValue.Value,
		})
	}
	data.TopicLastEpochNonces = topicLastEpochNonces

	return nil
}
