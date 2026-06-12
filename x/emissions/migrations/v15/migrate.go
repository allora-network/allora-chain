package v15

import (
	"cosmossdk.io/collections"
	collcodec "cosmossdk.io/collections/codec"
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
// It backfills params and topic defaults required by the classification
// feature, then migrates stored network inference bundles to the new labeled
// bundle format.
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 14 TO VERSION 15")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	// Params must be backfilled before post-v15 code reads label-related caps;
	// pre-v15 stored Params decode zero for newly added label params.
	if err := MigrateParams(ctx, emissionsKeeper); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateParams() FROM VERSION 14 TO VERSION 15")
		return err
	}

	if err := MigrateTopics(ctx, emissionsKeeper, store, cdc); err != nil {
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

// MigrateParams backfills module params fields introduced for v15 label
// canonicalization. Only zero-valued fields are touched; all other params are
// left unchanged.
//
// Backfilled fields:
//   - MaxCanonicalLabelByteLength: defaults to the module-initial cap when
//     zero. A zero cap would reject every label after v15.
//   - MaxTopicLabelWhitelistSize: defaults to the module-initial cap when
//     zero. A zero cap would allow no topic whitelist entries.
//   - MaxEpochLabelRegistrySize: defaults to the module-initial cap when zero.
//     A zero cap would reject every new epoch label registration.
func MigrateParams(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	params, err := emissionsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "MIGRATION V15: failed to get existing params")
	}

	defaultParams := emissionstypes.DefaultParams()
	changed := false
	if params.MaxCanonicalLabelByteLength == 0 {
		params.MaxCanonicalLabelByteLength = defaultParams.MaxCanonicalLabelByteLength
		changed = true
	}
	if params.MaxTopicLabelWhitelistSize == 0 {
		params.MaxTopicLabelWhitelistSize = defaultParams.MaxTopicLabelWhitelistSize
		changed = true
	}
	if params.MaxEpochLabelRegistrySize == 0 {
		params.MaxEpochLabelRegistrySize = defaultParams.MaxEpochLabelRegistrySize
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
		"maxCanonicalLabelByteLength", params.MaxCanonicalLabelByteLength,
		"maxTopicLabelWhitelistSize", params.MaxTopicLabelWhitelistSize,
		"maxEpochLabelRegistrySize", params.MaxEpochLabelRegistrySize,
	)
	return nil
}

// MigrateTopics backfills default topic settings introduced for the
// classification/multilabel feature without disturbing existing values.
func MigrateTopics(ctx sdk.Context, emissionsKeeper keeper.Keeper, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	iterator := topicStore.Iterator(nil, nil)
	defer iterator.Close()

	type kv struct {
		key   []byte
		value []byte
	}

	updates := make([]kv, 0)

	params, err := emissionsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "MIGRATION V15: failed to get existing params")
	}
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
		// Normalize any invalid decimal (NaN or non-finite) to zero so it
		// cannot survive the migration and fail later unity checks.
		if emissionstypes.ValidateDec(topic.UnityTolerance) != nil {
			topic.UnityTolerance = alloraMath.ZeroDec()
			changed = true
		}
		if topic.MaxLabelsPerSubmission == 0 {
			topic.MaxLabelsPerSubmission = emissionstypes.DefaultMaxLabelsPerSubmission
			changed = true
		}
		if emissionstypes.ValidateDec(topic.LabelDefaultValue) != nil {
			topic.LabelDefaultValue = alloraMath.ZeroDec()
			changed = true
		}

		if !changed {
			continue
		}

		// Safety check: validate the topic after backfill
		if err := topic.Validate(params); err != nil {
			return errorsmod.Wrapf(err, "v15 migration: topic %d failed validation after backfill", topic.Id)
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
		if networkInferenceBundle == nil {
			return errorsmod.Wrapf(
				emissionstypes.ErrInvalidValue,
				"converted %s bundle is nil; topicId: %d",
				logName,
				oldNetworkInference.TopicId,
			)
		}

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

// kvPair is a raw key/value pending write used by the v15 store migrations.
type kvPair struct {
	key   []byte
	value []byte
}

// labelRegistryPairKeyCodec encodes the (topicId, blockHeight) key used by the
// topic label registry store. It mirrors the codec the keeper registers for
// types.TopicLabelRegistryKey.
func labelRegistryPairKeyCodec() collcodec.KeyCodec[collections.Pair[uint64, int64]] {
	return collections.PairKeyCodec(collections.Uint64Key, collections.Int64Key)
}

// seedSingleArityRegistry returns the key/value for a {"y"} EpochLabelRegistry at
// (topicId, block) with ok=true, unless a registry already exists at that key (ok=false).
// It is the shared, idempotent seeding step used when backfilling registries for migrated
// single-arity inferences: the labelStore.Has guard makes re-running, or running after a
// sibling migration that already seeded the same key, a no-op.
func seedSingleArityRegistry(
	labelStore prefix.Store,
	lblKeyCodec collcodec.KeyCodec[collections.Pair[uint64, int64]],
	cdc codec.BinaryCodec,
	topicId uint64,
	block int64,
) (kvPair, bool, error) {
	pairKey := collections.Join(topicId, block)
	lblKeyBytes := make([]byte, lblKeyCodec.Size(pairKey))
	if _, err := lblKeyCodec.Encode(lblKeyBytes, pairKey); err != nil {
		return kvPair{}, false, errorsmod.Wrap(err, "failed to encode label store key") //nolint:exhaustruct // zero-value sentinel: ok=false, err set
	}
	if labelStore.Has(lblKeyBytes) {
		return kvPair{}, false, nil //nolint:exhaustruct // zero-value sentinel: ok=false signals "skip, already present"
	}
	registry := emissionstypes.EpochLabelRegistry{
		TopicId: topicId,
		EpochId: uint64(block), //nolint:gosec // block is a non-negative block height; cast is safe
		Labels: []*emissionstypes.TopicLabel{
			{Id: emissionstypes.SingleArityCanonicalLabelID, Name: emissionstypes.SingleArityCanonicalLabel},
		},
	}
	return kvPair{key: lblKeyBytes, value: cdc.MustMarshal(&registry)}, true, nil
}

func MigrateInferences(
	ctx sdk.Context,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
) error {
	infStore := prefix.NewStore(store, emissionstypes.InferencesKey)
	labelStore := prefix.NewStore(store, emissionstypes.TopicLabelRegistryKey)
	lblKeyCodec := labelRegistryPairKeyCodec()

	iterator := infStore.Iterator(nil, nil)
	defer iterator.Close()

	updates := make([]kvPair, 0)
	lblUpdates := make([]kvPair, 0)

	for ; iterator.Valid(); iterator.Next() {
		// Skip already-migrated records. In the new Inference type the value lives
		// in the repeated Values field (field 7) while the old Inference type
		// carries it in the scalar Value field (field 4, now reserved). Decoding
		// already-migrated bytes as the old type silently drops field 7 and would
		// rewrite the value as 0, which still passes Validate(). Guarding here keeps
		// any second pass (re-invocation, tooling) from zeroing migrated values.
		var maybeMigrated emissionstypes.Inference
		if err := cdc.Unmarshal(iterator.Value(), &maybeMigrated); err == nil && len(maybeMigrated.Values) > 0 {
			continue
		}

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

		updates = append(updates, kvPair{
			key:   append([]byte(nil), iterator.Key()...),
			value: cdc.MustMarshal(inference),
		})

		lbl, ok, err := seedSingleArityRegistry(labelStore, lblKeyCodec, cdc, oldInference.TopicId, oldInference.BlockHeight)
		if err != nil {
			return err
		}
		if ok {
			lblUpdates = append(lblUpdates, lbl)
		}
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
	labelStore := prefix.NewStore(store, emissionstypes.TopicLabelRegistryKey)
	lblKeyCodec := labelRegistryPairKeyCodec()

	iterator := infStore.Iterator(nil, nil)
	defer iterator.Close()

	updates := make([]kvPair, 0)
	lblUpdates := make([]kvPair, 0)

	for ; iterator.Valid(); iterator.Next() {
		// Skip already-migrated records. As in MigrateInferences, the per-inference
		// value moved from the old scalar Value field (field 4, now reserved) to the
		// new repeated Values field (field 7). If any inner inference already carries
		// Values, these bytes are new-format and re-decoding them as the old type
		// would silently zero every value while still passing Validate().
		var maybeMigrated emissionstypes.Inferences
		if err := cdc.Unmarshal(iterator.Value(), &maybeMigrated); err == nil &&
			len(maybeMigrated.Inferences) > 0 && len(maybeMigrated.Inferences[0].Values) > 0 {
			continue
		}

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

		updates = append(updates, kvPair{
			key:   append([]byte(nil), iterator.Key()...),
			value: cdc.MustMarshal(allInferences),
		})

		// Seed a {"y"} registry for this archived epoch so historical inferences can
		// be denormalized symmetrically with the latest-inference store (see
		// MigrateInferences). The archive is keyed by a unique (topic, block) and all
		// inferences in the entry share that block, so seed once per entry. The Has
		// guard makes it idempotent and a no-op for keys MigrateInferences already
		// seeded (it runs first). Writes are deferred until after iteration, matching
		// MigrateInferences, to avoid mutating the store while the iterator is open.
		if len(oldInferences.Inferences) > 0 {
			first := oldInferences.Inferences[0]
			lbl, ok, err := seedSingleArityRegistry(labelStore, lblKeyCodec, cdc, first.TopicId, first.BlockHeight)
			if err != nil {
				return err
			}
			if ok {
				lblUpdates = append(lblUpdates, lbl)
			}
		}
	}

	for _, u := range updates {
		infStore.Set(u.key, u.value)
	}
	for _, u := range lblUpdates {
		labelStore.Set(u.key, u.value)
	}

	ctx.Logger().Info(
		"MIGRATION V15: all inferences migration completed",
		"store", "entriesUpdated", len(updates), "registriesSeeded", len(lblUpdates),
	)
	return nil
}
