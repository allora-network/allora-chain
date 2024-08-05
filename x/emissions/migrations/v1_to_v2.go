package migrations

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
)

func V1ToV2(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 1 TO VERSION 2")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	err := migrateTopics(store, cdc)
	if err != nil {
		return err
	}
	err = migrateOffchainNode(store, cdc)
	if err != nil {
		return err
	}
	err = migrateNetworkLossBundles(store, cdc)
	if err != nil {
		return err
	}
	err = migrateAllLossBundles(store, cdc)
	if err != nil {
		return err
	}
	err = migrateAllRecordCommits(store, cdc)
	if err != nil {
		return err
	}

	defaultParams := types.DefaultParams()
	err = emissionsKeeper.SetParams(ctx, defaultParams)
	if err != nil {
		return err
	}

	return nil
}

func migrateTopics(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	topicStore := prefix.NewStore(store, types.TopicsKey)
	iterator := topicStore.Iterator(nil, nil)

	for ; iterator.Valid(); iterator.Next() {
		var oldMsg types.Topic
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		// Make newWorkerSubmissionWindow to be the 10% of the epoch length
		newWorkerSubmissionWindow := oldMsg.EpochLength / 10
		// set min and max boundaries: max 60 blocks
		if newWorkerSubmissionWindow > 60 {
			newWorkerSubmissionWindow = 60
		}

		newMsg := types.Topic{
			Id:                     oldMsg.Id,
			Creator:                oldMsg.Creator,
			Metadata:               oldMsg.Metadata,
			LossMethod:             "mse",
			EpochLastEnded:         oldMsg.EpochLastEnded,
			EpochLength:            oldMsg.EpochLength,
			GroundTruthLag:         oldMsg.GroundTruthLag,
			PNorm:                  oldMsg.PNorm,
			AlphaRegret:            oldMsg.AlphaRegret,
			AllowNegative:          oldMsg.AllowNegative,
			Epsilon:                oldMsg.Epsilon,
			InitialRegret:          oldMsg.InitialRegret,
			WorkerSubmissionWindow: newWorkerSubmissionWindow,
		}

		store.Delete(iterator.Key())
		store.Set(iterator.Key(), cdc.MustMarshal(&newMsg))
	}

	return nil
}

func migrateOffchainNode(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	workerStore := prefix.NewStore(store, types.WorkerNodesKey)
	iterator := workerStore.Iterator(nil, nil)

	for ; iterator.Valid(); iterator.Next() {
		var oldMsg types.OffchainNode
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		newMsg := types.OffchainNode{
			NodeAddress: oldMsg.NodeAddress,
			Owner:       oldMsg.Owner,
		}

		store.Delete(iterator.Key())
		store.Set([]byte(oldMsg.NodeAddress), cdc.MustMarshal(&newMsg))
	}

	reputerStore := prefix.NewStore(store, types.ReputerNodesKey)
	iterator = reputerStore.Iterator(nil, nil)

	for ; iterator.Valid(); iterator.Next() {
		var oldMsg types.OffchainNode
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		newMsg := types.OffchainNode{
			NodeAddress: oldMsg.NodeAddress,
			Owner:       oldMsg.Owner,
		}

		store.Delete(iterator.Key())
		store.Set([]byte(oldMsg.NodeAddress), cdc.MustMarshal(&newMsg))
	}
	return nil
}

func migrateNetworkLossBundles(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	networkLossBundlesStore := prefix.NewStore(store, types.NetworkLossBundlesKey)
	iterator := networkLossBundlesStore.Iterator(nil, nil)

	for ; iterator.Valid(); iterator.Next() {
		var oldMsg types.ValueBundle
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		newMsg := types.ValueBundle{
			TopicId:                       oldMsg.TopicId,
			ReputerRequestNonce:           oldMsg.ReputerRequestNonce,
			Reputer:                       oldMsg.Reputer,
			ExtraData:                     oldMsg.ExtraData,
			CombinedValue:                 oldMsg.CombinedValue,
			InfererValues:                 oldMsg.InfererValues,
			ForecasterValues:              oldMsg.ForecasterValues,
			NaiveValue:                    oldMsg.NaiveValue,
			OneOutInfererForecasterValues: []*types.OneOutInfererForecasterValues{},
			OneOutInfererValues:           oldMsg.OneOutInfererValues,
			OneOutForecasterValues:        oldMsg.OneOutForecasterValues,
			OneInForecasterValues:         oldMsg.OneInForecasterValues,
		}

		store.Delete(iterator.Key())
		store.Set(iterator.Key(), cdc.MustMarshal(&newMsg))
	}
	return nil
}

func migrateAllLossBundles(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	allLossBundlesStore := prefix.NewStore(store, types.AllLossBundlesKey)
	iterator := allLossBundlesStore.Iterator(nil, nil)

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

		store.Delete(iterator.Key())
		store.Set(iterator.Key(), cdc.MustMarshal(&newMsg))
	}
	return nil
}

func migrateAllRecordCommits(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	err := restoreAllRecordCommits(store, cdc, types.TopicLastWorkerCommitKey)
	if err != nil {
		return err
	}
	err = restoreAllRecordCommits(store, cdc, types.TopicLastReputerCommitKey)
	if err != nil {
		return err
	}
	err = restoreAllRecordCommits(store, cdc, types.TopicLastWorkerPayloadKey)
	if err != nil {
		return err
	}
	err = restoreAllRecordCommits(store, cdc, types.TopicLastReputerPayloadKey)
	if err != nil {
		return err
	}
	return nil
}

func restoreAllRecordCommits(store storetypes.KVStore, cdc codec.BinaryCodec, commitKey collections.Prefix) error {
	topicLastWorkerCommitStore := prefix.NewStore(store, commitKey)
	iterator := topicLastWorkerCommitStore.Iterator(nil, nil)

	for ; iterator.Valid(); iterator.Next() {
		var oldMsg types.TimestampedActorNonce
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		newMsg := types.TimestampedActorNonce{
			BlockHeight: oldMsg.BlockHeight,
			Nonce:       oldMsg.Nonce,
		}

		store.Delete(iterator.Key())
		store.Set(iterator.Key(), cdc.MustMarshal(&newMsg))
	}
	return nil
}

func removeOldKVStores(store storetypes.KVStore) {
	store.Delete(types.ChurnableTopicsKey)
}
