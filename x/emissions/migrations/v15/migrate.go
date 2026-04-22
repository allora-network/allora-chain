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
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// MigrateStore migrates the emissions module from version 14 to version 15.
// It backfills topic defaults required by the classification feature and
// migrates stored network inference bundles to the new labeled bundle format.
// It also canonicalizes per-topic label whitelists under the stricter v15
// label rules (ASCII charset; optional lowercasing controlled by
// topic.LabelCaseSensitive, which proto-defaults to false for pre-v15 topics).
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 14 TO VERSION 15")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	// Params must be backfilled before any migration step that consults
	// label-related caps (MigrateTopicLabelWhitelists reads
	// MaxCanonicalLabelByteLength), otherwise pre-v15 stored Params would
	// decode zero for the new cap and reject every whitelist entry.
	if err := MigrateParams(ctx, emissionsKeeper); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateParams() FROM VERSION 14 TO VERSION 15")
		return err
	}

	if err := MigrateTopics(ctx, store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateTopics() FROM VERSION 14 TO VERSION 15")
		return err
	}

	if err := MigrateTopicLabelWhitelists(ctx, store, cdc, emissionsKeeper); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateTopicLabelWhitelists() FROM VERSION 14 TO VERSION 15")
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

	ctx.Logger().Info("MIGRATION EMISSIONS MODULE FROM VERSION 14 TO VERSION 15 COMPLETE")
	return nil
}

// MigrateTopicLabelWhitelists iterates all topics and re-canonicalizes
// LabelWhitelist entries under the v15 rules. The migration is idempotent:
// running it twice yields the same canonical whitelist because
// CanonicalizeLabelList is idempotent on already-canonical input.
//
// For each topic:
//   - LabelWhitelist is filtered to entries that canonicalize successfully
//     under topic.LabelCaseSensitive and Params.MaxCanonicalLabelByteLength.
//   - Duplicates post-canonicalization are collapsed (first occurrence wins).
//   - Labels that fail the new canonicalizer (for example, labels that carry
//     characters outside the ASCII charset) are dropped with a Warn log
//     rather than aborting the migration: on a live chain we prefer to clear
//     obviously-unusable whitelist entries than to halt the upgrade, because
//     a whitelisted label that cannot be submitted is already a silent dead
//     end.
//   - If the filtered whitelist differs from the stored one, the topic is
//     persisted.
//
// EpochLabelRegistry entries are left untouched: they are nonce-scoped and
// were already persisted under the previous rules.
func MigrateTopicLabelWhitelists(
	ctx sdk.Context,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	emissionsKeeper keeper.Keeper,
) error {
	params, err := emissionsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "MIGRATION V15: failed to get params for whitelist canonicalization")
	}
	maxBytes := params.MaxCanonicalLabelByteLength
	if maxBytes == 0 {
		// Defensive: MigrateParams should have backfilled this already to
		// the module-initial cap of 64 bytes.
		maxBytes = 64
	}

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	iterator := topicStore.Iterator(nil, nil)
	defer iterator.Close()

	type kv struct {
		key   []byte
		value []byte
	}

	updates := make([]kv, 0)
	topicsTouched := 0
	labelsDropped := 0

	for ; iterator.Valid(); iterator.Next() {
		var topic emissionstypes.Topic
		if err := cdc.Unmarshal(iterator.Value(), &topic); err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal topic during v15 migration")
		}

		if len(topic.LabelWhitelist) == 0 {
			continue
		}

		filtered := make([]string, 0, len(topic.LabelWhitelist))
		seen := make(map[string]struct{}, len(topic.LabelWhitelist))
		droppedForTopic := 0
		for _, raw := range topic.LabelWhitelist {
			canonical, err := emissionstypes.CanonicalLabelName(raw, maxBytes, topic.LabelCaseSensitive)
			if err != nil {
				droppedForTopic++
				ctx.Logger().Warn(
					"MIGRATION V15: dropping non-canonical whitelist entry",
					"topicId", topic.Id,
					"label", raw,
					"error", err.Error(),
				)
				continue
			}
			if _, dup := seen[canonical]; dup {
				droppedForTopic++
				continue
			}
			seen[canonical] = struct{}{}
			filtered = append(filtered, canonical)
		}

		labelsDropped += droppedForTopic
		if !labelWhitelistEqual(topic.LabelWhitelist, filtered) {
			topic.LabelWhitelist = filtered
			raw, err := cdc.Marshal(&topic)
			if err != nil {
				return errorsmod.Wrapf(err, "failed to marshal migrated topic %d", topic.Id)
			}
			updates = append(updates, kv{
				key:   append([]byte(nil), iterator.Key()...),
				value: raw,
			})
			topicsTouched++
		}
	}

	for _, u := range updates {
		topicStore.Set(u.key, u.value)
	}

	ctx.Logger().Info(
		"MIGRATION V15: topic label whitelist canonicalization completed",
		"topicsTouched", topicsTouched,
		"labelsDropped", labelsDropped,
	)
	return nil
}

// labelWhitelistEqual reports whether two ordered whitelists are byte-equal.
// Treated separately from the keeper's labelWhitelistChanged so the migration
// has no keeper-layer dependency.
func labelWhitelistEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// MigrateParams backfills module params fields introduced for v15 label
// canonicalization. Only zero-valued fields are touched; all other params are
// left unchanged.
//
// Backfilled fields:
//   - MaxCanonicalLabelByteLength: defaults to 64 (the module-initial cap)
//     when zero. Backfilled before MigrateTopicLabelWhitelists runs because
//     the whitelist canonicalizer reads this cap; a zero cap would reject
//     every label.
func MigrateParams(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	params, err := emissionsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "MIGRATION V15: failed to get existing params")
	}

	changed := false
	if params.MaxCanonicalLabelByteLength == 0 {
		params.MaxCanonicalLabelByteLength = 64
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
		if topic.MaxLabelsPerSubmission == 0 {
			topic.MaxLabelsPerSubmission = emissionstypes.DefaultMaxLabelsPerSubmission
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
