package v15

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	"github.com/allora-network/allora-chain/x/emissions/migrations/v15/oldtypes"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// MigrateStore migrates the emissions module from version 14 to version 15.
// It backfills topic defaults required by the classification feature and
// migrates stored network inference bundles to the new labeled bundle format.
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 14 TO VERSION 15")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	if err := MigrateTopics(ctx, store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateTopics() FROM VERSION 14 TO VERSION 15")
		return err
	}

	if err := MigrateNetworkInferences(ctx, store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateNetworkInferences() FROM VERSION 14 TO VERSION 15")
		return err
	}

	if err := MigrateOutlierResistantNetworkInferences(ctx, store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateOutlierResistantNetworkInferences() FROM VERSION 14 TO VERSION 15")
		return err
	}

	if err := MigrateInferences(ctx, store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateInferences() FROM VERSION 14 TO VERSION 15")
		return err
	}

	if err := MigrateAllInferences(ctx, store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateAllInferences() FROM VERSION 14 TO VERSION 15")
		return err
	}

	ctx.Logger().Info("MIGRATION EMISSIONS MODULE FROM VERSION 14 TO VERSION 15 COMPLETE")
	return nil
}

// MigrateTopics backfills default topic settings introduced for the
// classification/multilabel feature without disturbing existing values.
func MigrateTopics(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	iterator := topicStore.Iterator(nil, nil)
	defer iterator.Close()

	type kv struct {
		key   []byte
		value []byte
	}

	updates := make([]kv, 0)

	for ; iterator.Valid(); iterator.Next() {
		var topic emissionstypes.Topic
		if err := cdc.Unmarshal(iterator.Value(), &topic); err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal topic")
		}

		changed := false
		if topic.TopicType == emissionstypes.TopicType_TOPIC_TYPE_UNSPECIFIED {
			topic.TopicType = emissionstypes.TopicType_TOPIC_TYPE_REGRESSION
			changed = true
		}
		if topic.OutputArity == emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_UNSPECIFIED {
			topic.OutputArity = emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE
			changed = true
		}
		if topic.UnityTolerance.IsNaN() {
			topic.UnityTolerance = alloraMath.ZeroDec()
			changed = true
		}

		if !changed {
			continue
		}

		updates = append(updates, kv{
			key:   append([]byte(nil), iterator.Key()...),
			value: cdc.MustMarshal(&topic),
		})
	}

	for _, u := range updates {
		topicStore.Set(u.key, u.value)
	}

	ctx.Logger().Info("MIGRATION V15: topic defaults migration completed", "topicsUpdated", len(updates))
	return nil
}

func MigrateNetworkInferences(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	return migrateInferenceBundles(
		ctx,
		store,
		cdc,
		emissionstypes.NetworkInferencesKey,
		emissionstypes.NetworkInferenceBundleKey,
		"network inferences",
	)
}

func MigrateOutlierResistantNetworkInferences(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	return migrateInferenceBundles(
		ctx,
		store,
		cdc,
		emissionstypes.OutlierResistantNetworkInferencesKey,
		emissionstypes.OutlierResistantNetworkInferenceBundleKey,
		"outlier resistant network inferences",
	)
}

func migrateInferenceBundles(
	ctx sdk.Context,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	sourcePrefix collections.Prefix,
	destPrefix collections.Prefix,
	logName string,
) error {
	sourceStore := prefix.NewStore(store, sourcePrefix)
	iterator := sourceStore.Iterator(nil, nil)
	defer iterator.Close()

	type kv struct {
		key   []byte
		value []byte
	}

	updates := make([]kv, 0)

	for ; iterator.Valid(); iterator.Next() {
		var oldNetworkInference emissionstypes.ValueBundle
		if err := cdc.Unmarshal(iterator.Value(), &oldNetworkInference); err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal %s", logName)
		}

		networkInferenceBundle := emissionstypes.ValueBundleToNetworkInferenceBundle(&oldNetworkInference)

		if err := networkInferenceBundle.Validate(); err != nil {
			return errorsmod.Wrapf(err, "failed to validate %s", logName)
		}

		updates = append(updates, kv{
			key:   append([]byte(nil), iterator.Key()...),
			value: cdc.MustMarshal(networkInferenceBundle),
		})
	}

	destStore := prefix.NewStore(store, destPrefix)
	for _, u := range updates {
		destStore.Set(u.key, u.value)
		sourceStore.Delete(u.key)
	}

	ctx.Logger().Info("MIGRATION V15: network inference bundle migration completed", "store", logName, "entriesUpdated", len(updates))
	return nil
}

func MigrateInferences(
	ctx sdk.Context,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
) error {
	infStore := prefix.NewStore(store, emissionstypes.InferencesKey)
	labelStore := prefix.NewStore(store, emissionstypes.TopicLabelRegistryKey)
	lblKeyCodec := collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key)

	iterator := infStore.Iterator(nil, nil)
	defer iterator.Close()

	type kv struct {
		key   []byte
		value []byte
	}

	updates := make([]kv, 0)
	lblUpdates := make([]kv, 0)

	for ; iterator.Valid(); iterator.Next() {
		var oldInference oldtypes.Inference
		if err := cdc.Unmarshal(iterator.Value(), &oldInference); err != nil {
			return errorsmod.Wrap(err, "failed to unmarshal inferences")
		}

		inference := &emissionstypes.Inference{
			TopicId:     oldInference.TopicId,
			BlockHeight: oldInference.BlockHeight,
			Inferer:     oldInference.Inferer,
			Values:      []alloraMath.Dec{oldInference.Value},
			ExtraData:   oldInference.ExtraData,
			Proof:       oldInference.Proof,
		}

		if err := inference.Validate(); err != nil {
			return errorsmod.Wrap(err, "failed to validate inferences")
		}

		updates = append(updates, kv{
			key:   append([]byte(nil), iterator.Key()...),
			value: cdc.MustMarshal(inference),
		})

		lblKeyBytes := make([]byte, lblKeyCodec.Size(collections.Join(oldInference.TopicId, oldInference.BlockHeight)))
		if _, err := lblKeyCodec.Encode(lblKeyBytes, collections.Join(oldInference.TopicId, oldInference.BlockHeight)); err != nil {
			return errorsmod.Wrap(err, "failed to encode label store key")
		}
		if labelStore.Has(lblKeyBytes) {
			continue
		}
		registry := emissionstypes.EpochLabelRegistry{
			TopicId: oldInference.TopicId,
			EpochId: uint64(oldInference.BlockHeight),
			Labels: []*emissionstypes.TopicLabel{
				{Id: 1, Name: "y"},
			},
		}
		lblUpdates = append(lblUpdates, kv{
			key:   lblKeyBytes,
			value: cdc.MustMarshal(&registry),
		})
	}

	for _, u := range updates {
		infStore.Set(u.key, u.value)
	}
	for _, u := range lblUpdates {
		labelStore.Set(u.key, u.value)
	}

	ctx.Logger().Info("MIGRATION V15: inferences migration completed", "store", "entriesUpdated", len(updates))
	return nil
}

func MigrateAllInferences(
	ctx sdk.Context,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
) error {
	infStore := prefix.NewStore(store, emissionstypes.AllInferencesKey)
	iterator := infStore.Iterator(nil, nil)
	defer iterator.Close()

	type kv struct {
		key   []byte
		value []byte
	}

	updates := make([]kv, 0)

	for ; iterator.Valid(); iterator.Next() {
		var oldInferences oldtypes.Inferences
		if err := cdc.Unmarshal(iterator.Value(), &oldInferences); err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal all inferences")
		}

		allInferences := &emissionstypes.Inferences{
			Inferences: make([]*emissionstypes.Inference, len(oldInferences.Inferences)),
		}

		for i, oldInference := range oldInferences.Inferences {
			allInferences.Inferences[i] = &emissionstypes.Inference{
				TopicId:     oldInference.TopicId,
				BlockHeight: oldInference.BlockHeight,
				Inferer:     oldInference.Inferer,
				Values:      []alloraMath.Dec{oldInference.Value},
				ExtraData:   oldInference.ExtraData,
				Proof:       oldInference.Proof,
			}

			if err := allInferences.Inferences[i].Validate(); err != nil {
				return errorsmod.Wrapf(err, "failed to validate all inferences")
			}
		}

		updates = append(updates, kv{
			key:   append([]byte(nil), iterator.Key()...),
			value: cdc.MustMarshal(allInferences),
		})
	}

	for _, u := range updates {
		infStore.Set(u.key, u.value)
	}

	ctx.Logger().Info("MIGRATION V15: all inferences migration completed", "store", "entriesUpdated", len(updates))
	return nil
}
