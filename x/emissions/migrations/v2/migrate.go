package v2

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	oldtypes "github.com/allora-network/allora-chain/x/emissions/migrations/v2/oldtypes"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
)

func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 1 TO VERSION 2")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	removeOldKVStores(store)

	err := MigrateTopics(store, cdc)
	if err != nil {
		return err
	}
	err = MigrateOffchainNode(store, cdc)
	if err != nil {
		return err
	}
	err = MigrateNetworkLossBundles(store, cdc)
	if err != nil {
		return err
	}
	err = MigrateAllLossBundles(store, cdc)
	if err != nil {
		return err
	}
	err = MigrateAllRecordCommits(store, cdc)
	if err != nil {
		return err
	}

	err = MigrateParams(ctx, emissionsKeeper)
	if err != nil {
		return err
	}

	return nil
}

func MigrateParams(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	defaultParams := types.DefaultParams()
	err := emissionsKeeper.SetParams(ctx, defaultParams)
	if err != nil {
		return err
	}
	return nil
}

func MigrateTopics(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	topicStore := prefix.NewStore(store, types.TopicsKey)
	iterator := topicStore.Iterator(nil, nil)
	defer iterator.Close()

	valueToAdd := make(map[string]types.Topic, 0)
	for ; iterator.Valid(); iterator.Next() {
		var oldMsg oldtypes.Topic
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		// Make newWorkerSubmissionWindow to be the 10% of the epoch length
		newWorkerSubmissionWindow := oldMsg.EpochLength / 10
		// set min and max boundaries: max 60 blocks
		if newWorkerSubmissionWindow > 60 {
			newWorkerSubmissionWindow = 60
		} else if newWorkerSubmissionWindow == 0 {
			newWorkerSubmissionWindow = max(1, oldMsg.EpochLength/2)
		}

		newMsg := types.Topic{ //nolint: exhaustruct // not sure if safe to fix, also this upgrade has already happened.
			Id:                     oldMsg.Id,
			Creator:                oldMsg.Creator,
			Metadata:               oldMsg.Metadata,
			LossMethod:             "mse",
			EpochLastEnded:         oldMsg.EpochLastEnded, // Add default value
			EpochLength:            oldMsg.EpochLength,
			GroundTruthLag:         oldMsg.GroundTruthLag,
			PNorm:                  oldMsg.PNorm,
			AlphaRegret:            oldMsg.AlphaRegret,
			AllowNegative:          oldMsg.AllowNegative,
			Epsilon:                alloraMath.MustNewDecFromString("0.01"),
			InitialRegret:          alloraMath.MustNewDecFromString("0"),
			WorkerSubmissionWindow: newWorkerSubmissionWindow,
		}

		valueToAdd[string(iterator.Key())] = newMsg
	}
	iterator.Close()

	for key, value := range valueToAdd {
		topicStore.Set([]byte(key), cdc.MustMarshal(&value))
	}

	return nil
}

func MigrateOffchainStore(workerStore storetypes.KVStore, cdc codec.BinaryCodec) error {
	iterator := workerStore.Iterator(nil, nil)
	defer iterator.Close()

	keysToDelete := make([][]byte, 0)
	for ; iterator.Valid(); iterator.Next() {
		keysToDelete = append(keysToDelete, iterator.Key())
	}
	iterator.Close()

	// delete the old keys
	for _, key := range keysToDelete {
		oldNode := workerStore.Get(key)
		var oldMsg oldtypes.OffchainNode
		err := proto.Unmarshal(oldNode, &oldMsg)
		if err != nil {
			return err
		}
		newMsg := types.OffchainNode{
			NodeAddress: oldMsg.NodeAddress,
			Owner:       oldMsg.Owner,
		}
		workerStore.Set([]byte(oldMsg.NodeAddress), cdc.MustMarshal(&newMsg))
		workerStore.Delete(key)
	}
	return nil
}

func MigrateOffchainNode(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	workerStore := prefix.NewStore(store, types.WorkerNodesKey)
	err := MigrateOffchainStore(workerStore, cdc)
	if err != nil {
		return errors.Wrap(err, "error migrating worker store")
	}
	reputerStore := prefix.NewStore(store, types.ReputerNodesKey)
	err = MigrateOffchainStore(reputerStore, cdc)
	if err != nil {
		return errors.Wrap(err, "error migrating reputer store")
	}
	return nil
}

func MigrateNetworkLossBundles(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	networkLossBundlesStore := prefix.NewStore(store, types.NetworkLossBundlesKey)
	iterator := networkLossBundlesStore.Iterator(nil, nil)
	defer iterator.Close()

	valueToAdd := make(map[string]types.ValueBundle, 0)
	for ; iterator.Valid(); iterator.Next() {
		var oldMsg oldtypes.ValueBundle
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		newInfererValues := make([]*types.WorkerAttributedValue, 0)
		newForecastValues := make([]*types.WorkerAttributedValue, 0)
		newOneOutInfererValues := make([]*types.WithheldWorkerAttributedValue, 0)
		newOneOutForecasterValues := make([]*types.WithheldWorkerAttributedValue, 0)
		newOneInForecastValues := make([]*types.WorkerAttributedValue, 0)
		newMsg := types.ValueBundle{
			TopicId: oldMsg.TopicId,
			ReputerRequestNonce: &types.ReputerRequestNonce{
				ReputerNonce: &types.Nonce{
					BlockHeight: oldMsg.ReputerRequestNonce.ReputerNonce.BlockHeight,
				},
			},
			Reputer:                       oldMsg.Reputer,
			ExtraData:                     oldMsg.ExtraData,
			CombinedValue:                 oldMsg.CombinedValue,
			InfererValues:                 newInfererValues,
			ForecasterValues:              newForecastValues,
			NaiveValue:                    oldMsg.NaiveValue,
			OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{},
			OneOutInfererValues:           newOneOutInfererValues,
			OneOutForecasterValues:        newOneOutForecasterValues,
			OneInForecasterValues:         newOneInForecastValues,
		}

		for _, inference := range oldMsg.InfererValues {
			newInfererValues = append(newInfererValues, &types.WorkerAttributedValue{
				Worker: inference.Worker,
				Value:  inference.Value,
			})
		}
		for _, forecast := range oldMsg.ForecasterValues {
			newForecastValues = append(newForecastValues, &types.WorkerAttributedValue{
				Worker: forecast.Worker,
				Value:  forecast.Value,
			})
		}
		for _, inference := range oldMsg.OneOutInfererValues {
			newOneOutInfererValues = append(newOneOutInfererValues, &types.WithheldWorkerAttributedValue{
				Worker: inference.Worker,
				Value:  inference.Value,
			})
		}
		for _, forecast := range oldMsg.OneOutForecasterValues {
			newOneOutForecasterValues = append(newOneOutForecasterValues, &types.WithheldWorkerAttributedValue{
				Worker: forecast.Worker,
				Value:  forecast.Value,
			})
		}
		for _, forecast := range oldMsg.OneInForecasterValues {
			newOneInForecastValues = append(newOneInForecastValues, &types.WorkerAttributedValue{
				Worker: forecast.Worker,
				Value:  forecast.Value,
			})
		}

		newMsg.InfererValues = newInfererValues
		newMsg.ForecasterValues = newForecastValues
		newMsg.OneOutInfererValues = newOneOutInfererValues
		newMsg.OneOutForecasterValues = newOneOutForecasterValues
		newMsg.OneInForecasterValues = newOneInForecastValues

		valueToAdd[string(iterator.Key())] = newMsg
	}
	iterator.Close()

	for key, value := range valueToAdd {
		networkLossBundlesStore.Set([]byte(key), cdc.MustMarshal(&value))
	}

	return nil
}

func MigrateAllLossBundles(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	allLossBundlesStore := prefix.NewStore(store, types.AllLossBundlesKey)
	iterator := allLossBundlesStore.Iterator(nil, nil)
	defer iterator.Close()

	valuesToAdd := make(map[string]types.ReputerValueBundles, 0)
	for ; iterator.Valid(); iterator.Next() {
		var oldMsg types.ReputerValueBundles
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		newMsg := types.ReputerValueBundles{
			ReputerValueBundles: []*types.ReputerValueBundle{},
		}

		for _, valueBundle := range oldMsg.ReputerValueBundles {
			newMsg.ReputerValueBundles = append(newMsg.ReputerValueBundles,
				&types.ReputerValueBundle{
					ValueBundle: &types.ValueBundle{
						TopicId:                       valueBundle.ValueBundle.TopicId,
						ReputerRequestNonce:           valueBundle.ValueBundle.ReputerRequestNonce,
						Reputer:                       valueBundle.ValueBundle.Reputer,
						ExtraData:                     valueBundle.ValueBundle.ExtraData,
						CombinedValue:                 valueBundle.ValueBundle.CombinedValue,
						InfererValues:                 valueBundle.ValueBundle.InfererValues,
						ForecasterValues:              valueBundle.ValueBundle.ForecasterValues,
						NaiveValue:                    valueBundle.ValueBundle.NaiveValue,
						OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{},
						OneOutInfererValues:           valueBundle.ValueBundle.OneOutInfererValues,
						OneOutForecasterValues:        valueBundle.ValueBundle.OneOutForecasterValues,
						OneInForecasterValues:         valueBundle.ValueBundle.OneInForecasterValues,
					},
					Pubkey:    valueBundle.Pubkey,
					Signature: valueBundle.Signature,
				},
			)
		}
		valuesToAdd[string(iterator.Key())] = newMsg
	}
	iterator.Close()

	for key, value := range valuesToAdd {
		allLossBundlesStore.Set([]byte(key), cdc.MustMarshal(&value))
	}

	return nil
}

func MigrateAllRecordCommits(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	err := restoreAllRecordCommits(store, cdc, types.TopicLastWorkerCommitKey)
	if err != nil {
		return err
	}
	err = restoreAllRecordCommits(store, cdc, types.TopicLastReputerCommitKey)
	if err != nil {
		return err
	}
	return nil
}

func restoreAllRecordCommits(store storetypes.KVStore, cdc codec.BinaryCodec, commitKey collections.Prefix) error {
	topicLastWorkerCommitStore := prefix.NewStore(store, commitKey)
	iterator := topicLastWorkerCommitStore.Iterator(nil, nil)
	defer iterator.Close()

	valuesToAdd := make(map[string]types.TimestampedActorNonce, 0)
	for ; iterator.Valid(); iterator.Next() {
		var oldMsg oldtypes.TimestampedActorNonce
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		newMsg := types.TimestampedActorNonce{
			BlockHeight: oldMsg.BlockHeight,
			Nonce: &types.Nonce{
				BlockHeight: oldMsg.Nonce.BlockHeight,
			},
		}

		valuesToAdd[string(iterator.Key())] = newMsg
	}
	iterator.Close()

	for key, value := range valuesToAdd {
		topicLastWorkerCommitStore.Set([]byte(key), cdc.MustMarshal(&value))
	}

	return nil
}

func removeOldKVStores(store storetypes.KVStore) {
	store.Delete(types.ChurnableTopicsKey)
	store.Delete(types.TopicLastWorkerPayloadKey)
	store.Delete(types.TopicLastReputerPayloadKey)
}
