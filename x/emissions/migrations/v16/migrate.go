package v16

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// MigrateStore migrates the emissions module from version 15 to version 16.
// It backfills the per-topic Topic.MaxTopInferersToReward field introduced when
// the previously-global max_top_inferers_to_reward cap became a topic-level
// parameter. Existing topics decode this new field as zero and are backfilled
// with the current on-chain global value so their admission behavior is
// unchanged after the upgrade.
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 15 TO VERSION 16")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	if err := MigrateTopics(ctx, emissionsKeeper, store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateTopics() FROM VERSION 15 TO VERSION 16")
		return err
	}

	ctx.Logger().Info("MIGRATION EMISSIONS MODULE FROM VERSION 15 TO VERSION 16 COMPLETE")
	return nil
}

// MigrateTopics backfills Topic.MaxTopInferersToReward for existing topics.
// Only topics whose field is zero (i.e. persisted before the field existed) are
// touched; topics that already carry a value are left unchanged, so the
// migration is idempotent.
//
// The backfill value is the current on-chain global Params.MaxTopInferersToReward,
// which is also the per-topic ceiling enforced by Topic.Validate. As a defensive
// measure the migration first repairs a degenerate zero global (which should be
// impossible now that the params validator rejects zero, and which would already
// have zeroed the scores.go retention window) to the module default, so that
// every topic ends up with a concrete value >= 1 and topic validation cannot
// halt the upgrade on a bad ceiling.
func MigrateTopics(ctx sdk.Context, emissionsKeeper keeper.Keeper, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	params, err := emissionsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "MIGRATION V16: failed to get existing params")
	}

	if params.MaxTopInferersToReward == 0 {
		params.MaxTopInferersToReward = emissionstypes.DefaultParams().MaxTopInferersToReward
		if err := params.Validate(); err != nil {
			return errorsmod.Wrap(err, "MIGRATION V16: repaired params failed validation")
		}
		if err := emissionsKeeper.SetParams(ctx, params); err != nil {
			return errorsmod.Wrap(err, "MIGRATION V16: failed to persist repaired params")
		}
		ctx.Logger().Info(
			"MIGRATION V16: repaired zero global max_top_inferers_to_reward to module default",
			"value", params.MaxTopInferersToReward,
		)
	}
	backfillValue := params.MaxTopInferersToReward

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
			return errorsmod.Wrap(err, "MIGRATION V16: failed to unmarshal topic")
		}

		// Only backfill topics missing the value; leave already-set topics
		// untouched so re-runs and already-migrated topics are no-ops.
		if topic.MaxTopInferersToReward != 0 {
			continue
		}
		topic.MaxTopInferersToReward = backfillValue

		// Abort (do not skip) if the topic is invalid after backfill: a
		// half-migrated topic would leave state inconsistent for post-v16 code.
		// Halting is deterministic across validators.
		if err := topic.Validate(params); err != nil {
			return errorsmod.Wrapf(err, "MIGRATION V16: topic %d failed validation after backfill", topic.Id)
		}

		updates = append(updates, kv{
			key:   append([]byte(nil), iterator.Key()...),
			value: cdc.MustMarshal(&topic),
		})
	}

	for _, u := range updates {
		topicStore.Set(u.key, u.value)
	}

	ctx.Logger().Info(
		"MIGRATION V16: topic max_top_inferers_to_reward backfill completed",
		"topicsUpdated", len(updates),
		"backfillValue", backfillValue,
	)
	return nil
}
