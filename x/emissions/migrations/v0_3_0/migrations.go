package v0_3_0

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	emissionsv1 "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
)

func MigrateStore(ctx sdk.Context, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 1 TO VERSION 2")

	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
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
	return nil
}

func MigrateTopics(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	topicStore := prefix.NewStore(store, emissionsv1.TopicsKey)
	iterator := topicStore.Iterator(nil, nil)

	for ; iterator.Valid(); iterator.Next() {
		var oldMsg emissionsv1.Topic
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

		newMsg := emissionsv1.Topic{
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

func MigrateOffchainNode(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	workerStore := prefix.NewStore(store, emissionsv1.WorkerNodesKey)
	iterator := workerStore.Iterator(nil, nil)

	for ; iterator.Valid(); iterator.Next() {
		var oldMsg emissionsv1.OffchainNode
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		newMsg := emissionsv1.OffchainNode{
			NodeAddress: oldMsg.NodeAddress,
			Owner:       oldMsg.Owner,
		}

		store.Delete(iterator.Key())
		store.Set(iterator.Key(), cdc.MustMarshal(&newMsg))
	}

	reputerStore := prefix.NewStore(store, emissionsv1.ReputerNodesKey)
	iterator = reputerStore.Iterator(nil, nil)

	for ; iterator.Valid(); iterator.Next() {
		var oldMsg emissionsv1.OffchainNode
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		newMsg := emissionsv1.OffchainNode{
			NodeAddress: oldMsg.NodeAddress,
			Owner:       oldMsg.Owner,
		}

		store.Delete(iterator.Key())
		store.Set(iterator.Key(), cdc.MustMarshal(&newMsg))
	}
	return nil
}

func MigrateNetworkLossBundles(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	networkLossBundlesStore := prefix.NewStore(store, emissionsv1.NetworkLossBundlesKey)
	iterator := networkLossBundlesStore.Iterator(nil, nil)

	for ; iterator.Valid(); iterator.Next() {
		var oldMsg emissionsv1.ValueBundle
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		newMsg := emissionsv1.ValueBundle{
			TopicId:                       oldMsg.TopicId,
			ReputerRequestNonce:           oldMsg.ReputerRequestNonce,
			Reputer:                       oldMsg.Reputer,
			ExtraData:                     oldMsg.ExtraData,
			CombinedValue:                 oldMsg.CombinedValue,
			InfererValues:                 oldMsg.InfererValues,
			ForecasterValues:              oldMsg.ForecasterValues,
			NaiveValue:                    oldMsg.NaiveValue,
			OneOutInfererForecasterValues: []*emissionsv1.OneOutInfererForecasterValues{},
			OneOutInfererValues:           oldMsg.OneOutInfererValues,
			OneOutForecasterValues:        oldMsg.OneOutForecasterValues,
			OneInForecasterValues:         oldMsg.OneInForecasterValues,
		}

		store.Delete(iterator.Key())
		store.Set(iterator.Key(), cdc.MustMarshal(&newMsg))
	}
	return nil
}

func MigrateAllLossBundles(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	allLossBundlesStore := prefix.NewStore(store, emissionsv1.AllLossBundlesKey)
	iterator := allLossBundlesStore.Iterator(nil, nil)

	for ; iterator.Valid(); iterator.Next() {
		var oldMsg emissionsv1.ReputerValueBundles
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		newMsg := emissionsv1.ReputerValueBundles{
			ReputerValueBundles: []*emissionsv1.ReputerValueBundle{},
		}

		for _, valueBundle := range oldMsg.ReputerValueBundles {
			newMsg.ReputerValueBundles = append(newMsg.ReputerValueBundles,
				&emissionsv1.ReputerValueBundle{
					ValueBundle: &emissionsv1.ValueBundle{
						TopicId:                       valueBundle.ValueBundle.TopicId,
						ReputerRequestNonce:           valueBundle.ValueBundle.ReputerRequestNonce,
						Reputer:                       valueBundle.ValueBundle.Reputer,
						ExtraData:                     valueBundle.ValueBundle.ExtraData,
						CombinedValue:                 valueBundle.ValueBundle.CombinedValue,
						InfererValues:                 valueBundle.ValueBundle.InfererValues,
						ForecasterValues:              valueBundle.ValueBundle.ForecasterValues,
						NaiveValue:                    valueBundle.ValueBundle.NaiveValue,
						OneOutInfererForecasterValues: []*emissionsv1.OneOutInfererForecasterValues{},
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

func MigrateAllRecordCommits(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	err := restoreAllRecordCommits(store, cdc, emissionsv1.TopicLastWorkerCommitKey)
	if err != nil {
		return err
	}
	err = restoreAllRecordCommits(store, cdc, emissionsv1.TopicLastReputerCommitKey)
	if err != nil {
		return err
	}
	err = restoreAllRecordCommits(store, cdc, emissionsv1.TopicLastWorkerPayloadKey)
	if err != nil {
		return err
	}
	err = restoreAllRecordCommits(store, cdc, emissionsv1.TopicLastReputerPayloadKey)
	if err != nil {
		return err
	}
	return nil
}

func restoreAllRecordCommits(store storetypes.KVStore, cdc codec.BinaryCodec, commitKey collections.Prefix) error {
	topicLastWorkerCommitStore := prefix.NewStore(store, commitKey)
	iterator := topicLastWorkerCommitStore.Iterator(nil, nil)

	for ; iterator.Valid(); iterator.Next() {
		var oldMsg emissionsv1.TimestampedActorNonce
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		newMsg := emissionsv1.TimestampedActorNonce{
			BlockHeight: oldMsg.BlockHeight,
			Nonce:       oldMsg.Nonce,
		}

		store.Delete(iterator.Key())
		store.Set(iterator.Key(), cdc.MustMarshal(&newMsg))
	}
	return nil
}
