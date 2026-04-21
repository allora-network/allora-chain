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

	if err := MigrateInferencesToWorkerLatestInputInferences(ctx, store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateInferencesToWorkerLatestInputInferences() FROM VERSION 14 TO VERSION 15")
		return err
	}

	if err := MigrateParams(ctx, emissionsKeeper); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateParams() FROM VERSION 14 TO VERSION 15")
		return err
	}

	ctx.Logger().Info("MIGRATION EMISSIONS MODULE FROM VERSION 14 TO VERSION 15 COMPLETE")
	return nil
}

// MigrateInferencesToWorkerLatestInputInferences drains the legacy per-worker
// inferences store (prefix 11) into the v2 workerLatestInputInferences store
// (prefix 109). Each legacy `Inference` is converted to an `InputInference`:
//
//   - SINGLE-arity topics carry the single Dec value on InputInference.Value
//     with Values left nil. The output arity is determined from the topic
//     (prefix 3); a missing topic is treated as SINGLE for safety.
//   - MULTI-arity topics reverse-map Inference.Values against the
//     EpochLabelRegistry for (topicId, blockHeight) at prefix 106: each
//     position i becomes an InputLabeledValue with the canonical label name
//     from registry.Labels[i]. If the registry is missing or shorter than
//     Inference.Values, the inference is dropped (source entry still deleted)
//     because there is no consistent way to reconstruct labels.
//
// Every source entry is deleted after processing so the legacy store is
// empty post-migration.
func MigrateInferencesToWorkerLatestInputInferences(
	ctx sdk.Context,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
) error {
	// The v15 migration is precisely the point at which we drain the
	// legacy InferencesKey store; referencing the deprecated prefix here
	// is intentional.
	sourceStore := prefix.NewStore(store, emissionstypes.InferencesKey) //nolint:staticcheck // SA1019: draining legacy prefix is the point of the migration
	destStore := prefix.NewStore(store, emissionstypes.WorkerLatestInputInferenceKey)
	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	registryStore := prefix.NewStore(store, emissionstypes.TopicLabelRegistryKey)

	type kv struct {
		key   []byte
		value []byte
	}
	writes := make([]kv, 0)
	deletes := make([][]byte, 0)

	iterator := sourceStore.Iterator(nil, nil)
	defer iterator.Close()

	topicArityCache := make(map[uint64]emissionstypes.TopicOutputArity)
	lookupArity := func(topicId uint64) emissionstypes.TopicOutputArity {
		if arity, ok := topicArityCache[topicId]; ok {
			return arity
		}
		topicKey, err := collections.EncodeKeyWithPrefix(nil, collections.Uint64Key, topicId)
		if err != nil {
			topicArityCache[topicId] = emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE
			return emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE
		}
		raw := topicStore.Get(topicKey)
		if raw == nil {
			topicArityCache[topicId] = emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE
			return emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE
		}
		var topic emissionstypes.Topic
		if err := cdc.Unmarshal(raw, &topic); err != nil {
			topicArityCache[topicId] = emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE
			return emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE
		}
		if topic.OutputArity == emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_UNSPECIFIED {
			topic.OutputArity = emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE
		}
		topicArityCache[topicId] = topic.OutputArity
		return topic.OutputArity
	}

	lookupRegistry := func(topicId uint64, blockHeight int64) (*emissionstypes.EpochLabelRegistry, error) {
		key, err := collections.EncodeKeyWithPrefix(
			nil,
			collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key),
			collections.Join(topicId, blockHeight),
		)
		if err != nil {
			return nil, err
		}
		raw := registryStore.Get(key)
		if raw == nil {
			// Sentinel: a nil *EpochLabelRegistry with a nil error means
			// "no registry at this (topicId, blockHeight)". Callers must
			// treat it as a drop signal, not as an error. We deliberately
			// return (nil, nil) here rather than a typed error to keep the
			// migration drain fast and simple on the common missing path.
			//nolint:nilnil
			return nil, nil
		}
		var reg emissionstypes.EpochLabelRegistry
		if err := cdc.Unmarshal(raw, &reg); err != nil {
			return nil, err
		}
		return &reg, nil
	}

	dropped := 0
	converted := 0

	for ; iterator.Valid(); iterator.Next() {
		var inf emissionstypes.Inference
		if err := cdc.Unmarshal(iterator.Value(), &inf); err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal legacy inference during v15 migration")
		}

		input := emissionstypes.InputInference{
			TopicId:     inf.TopicId,
			BlockHeight: inf.BlockHeight,
			Inferer:     inf.Inferer,
			ExtraData:   inf.ExtraData,
			Proof:       inf.Proof,
			Value:       alloraMath.BoundedExp40Dec{},
			Values:      nil,
		}

		arity := lookupArity(inf.TopicId)
		switch arity {
		case emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE:
			var scalar alloraMath.Dec
			switch {
			case len(inf.Values) == 0:
				scalar = alloraMath.ZeroDec()
			case len(inf.Values) == 1:
				scalar = inf.Values[0]
			default:
				scalar = inf.Values[0]
				ctx.Logger().Warn(
					"MIGRATION V15: SINGLE inference had multiple values; keeping Values[0]",
					"topicId", inf.TopicId,
					"inferer", inf.Inferer,
					"length", len(inf.Values),
				)
			}
			bounded, err := alloraMath.NewBoundedExp40Dec(scalar)
			if err != nil {
				ctx.Logger().Warn(
					"MIGRATION V15: inference scalar out of BoundedExp40 range; clamping to zero",
					"topicId", inf.TopicId,
					"inferer", inf.Inferer,
					"error", err,
				)
				bounded = alloraMath.MustNewBoundedExp40Dec(alloraMath.ZeroDec())
			}
			input.Value = bounded
		case emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_MULTI:
			reg, err := lookupRegistry(inf.TopicId, inf.BlockHeight)
			if err != nil {
				return errorsmod.Wrapf(err, "failed to read epoch label registry for topic %d nonce %d", inf.TopicId, inf.BlockHeight)
			}
			if reg == nil || len(reg.Labels) < len(inf.Values) {
				dropped++
				deletes = append(deletes, append([]byte(nil), iterator.Key()...))
				ctx.Logger().Warn(
					"MIGRATION V15: MULTI inference dropped; registry missing or shorter than values",
					"topicId", inf.TopicId,
					"inferer", inf.Inferer,
					"blockHeight", inf.BlockHeight,
					"valuesLen", len(inf.Values),
				)
				continue
			}
			labeled := make([]*emissionstypes.InputLabeledValue, 0, len(inf.Values))
			for i, v := range inf.Values {
				label := reg.Labels[i].GetName()
				bounded, err := alloraMath.NewBoundedExp40Dec(v)
				if err != nil {
					bounded = alloraMath.MustNewBoundedExp40Dec(alloraMath.ZeroDec())
				}
				labeled = append(labeled, &emissionstypes.InputLabeledValue{
					Label: label,
					Value: bounded,
				})
			}
			input.Value = alloraMath.MustNewBoundedExp40Dec(alloraMath.ZeroDec())
			input.Values = labeled
		default:
			// Unknown output arity: drop, since we cannot safely project.
			dropped++
			deletes = append(deletes, append([]byte(nil), iterator.Key()...))
			continue
		}

		raw, err := input.Marshal()
		if err != nil {
			return errorsmod.Wrapf(err, "failed to marshal migrated InputInference for topic %d inferer %s", inf.TopicId, inf.Inferer)
		}
		writes = append(writes, kv{
			key:   append([]byte(nil), iterator.Key()...),
			value: raw,
		})
		converted++
	}

	for _, w := range writes {
		destStore.Set(w.key, w.value)
		sourceStore.Delete(w.key)
	}
	for _, k := range deletes {
		sourceStore.Delete(k)
	}

	ctx.Logger().Info(
		"MIGRATION V15: legacy inferences → workerLatestInputInferences migration completed",
		"converted", converted,
		"dropped", dropped,
	)
	return nil
}

// MigrateParams backfills module params fields introduced with Epoch Label
// Registry v2 that are not present in the pre-v15 stored `Params`. Only
// zero-valued fields are touched; all other params are left unchanged.
//
// Backfilled fields:
//   - MaxLabelsPerSubmission: defaults to DefaultMaxLabelsPerSubmission when
//     zero. This matches the constructor default; a pre-v15 stored Params
//     value proto-decodes zero for this new field and would otherwise fail
//     the post-v15 validator (which requires >= 1).
func MigrateParams(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	params, err := emissionsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "MIGRATION V15: failed to get existing params")
	}

	changed := false
	if params.MaxLabelsPerSubmission == 0 {
		params.MaxLabelsPerSubmission = emissionstypes.DefaultMaxLabelsPerSubmission
		changed = true
	}

	if !changed {
		ctx.Logger().Info("MIGRATION V15: params backfill skipped (all fields already populated)")
		return nil
	}

	if err := params.Validate(); err != nil {
		return errorsmod.Wrap(err, "MIGRATION V15: backfilled params failed validation")
	}
	if err := emissionsKeeper.SetParams(ctx, params); err != nil {
		return errorsmod.Wrap(err, "MIGRATION V15: failed to persist backfilled params")
	}

	ctx.Logger().Info(
		"MIGRATION V15: params backfill completed",
		"maxLabelsPerSubmission", params.MaxLabelsPerSubmission,
	)
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
