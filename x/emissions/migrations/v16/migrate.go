package v16

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// MigrateStore migrates the emissions module from version 15 to version 16.
// It backfills the per-topic Topic.MaxTopInferersToReward field introduced when
// the previously-global max_top_inferers_to_reward cap became a topic-level
// parameter. Existing topics decode this new field as zero and are backfilled
// with the current on-chain global value so their admission behavior is
// unchanged after the upgrade. It also backfills the new
// Params.MinTopInferersToReward floor and recomputes totalSumPreviousTopicWeights
// from the active topic set so any drift accumulated by the incremental
// bookkeeping is cleared.
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 15 TO VERSION 16")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	// Topics first: it repairs a degenerate zero ceiling, and persisting the floor
	// requires a sane ceiling because params validation rejects floor > ceiling.
	if err := MigrateTopics(ctx, emissionsKeeper, store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateTopics() FROM VERSION 15 TO VERSION 16")
		return err
	}

	if err := MigrateParams(ctx, emissionsKeeper); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateParams() FROM VERSION 15 TO VERSION 16")
		return err
	}

	if err := MigrateTotalSumPreviousTopicWeights(ctx, emissionsKeeper); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateTotalSumPreviousTopicWeights() FROM VERSION 15 TO VERSION 16")
		return err
	}

	ctx.Logger().Info("MIGRATION EMISSIONS MODULE FROM VERSION 15 TO VERSION 16 COMPLETE")
	return nil
}

// MigrateTotalSumPreviousTopicWeights recomputes totalSumPreviousTopicWeights as the
// sum of the stored previous weights of the topics in the active set. The accumulator
// is otherwise only ever adjusted incrementally, so an earlier bookkeeping error (such as
// subtracting an inactive topic's weight twice on stake removal) persists in state after
// the code is fixed. Recomputing from the active set is idempotent and a no-op when the
// accumulator is already consistent.
func MigrateTotalSumPreviousTopicWeights(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	topicKeeper := emissionsKeeper.GetTopicKeeper()

	activeTopicIds, err := topicKeeper.GetActiveTopicIds(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "MIGRATION V16: failed to get active topic ids")
	}

	recomputed := alloraMath.ZeroDec()
	for _, topicId := range activeTopicIds {
		weight, noPrior, err := topicKeeper.GetPreviousTopicWeight(ctx, topicId)
		if err != nil {
			return errorsmod.Wrapf(err, "MIGRATION V16: failed to get previous weight of topic %d", topicId)
		}
		if noPrior {
			continue
		}
		recomputed, err = recomputed.Add(weight)
		if err != nil {
			return errorsmod.Wrapf(err, "MIGRATION V16: failed to add previous weight of topic %d", topicId)
		}
	}

	current, err := topicKeeper.GetTotalSumPreviousTopicWeights(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "MIGRATION V16: failed to get total sum of previous topic weights")
	}
	if current.Equal(recomputed) {
		ctx.Logger().Info("MIGRATION V16: total sum of previous topic weights already consistent", "value", current.String())
		return nil
	}

	if err := topicKeeper.SetTotalSumPreviousTopicWeights(ctx, recomputed); err != nil {
		return errorsmod.Wrap(err, "MIGRATION V16: failed to set recomputed total sum of previous topic weights")
	}
	ctx.Logger().Info(
		"MIGRATION V16: total sum of previous topic weights recomputed from active topics",
		"previous", current.String(),
		"recomputed", recomputed.String(),
		"activeTopics", len(activeTopicIds),
	)
	return nil
}

// MigrateParams backfills Params.MinTopInferersToReward, which an existing chain
// decodes as zero. Zero means "no floor", so this installs one where there was
// none; it changes no admission outcome at upgrade time only because
// MigrateTopics has already raised every topic cap to the ceiling. A non-zero
// value is left alone: an unset field is indistinguishable from a deliberate
// zero, so only the zero case is backfilled.
func MigrateParams(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	params, err := emissionsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "MIGRATION V16: failed to get existing params")
	}
	if params.MinTopInferersToReward != 0 {
		ctx.Logger().Info(
			"MIGRATION V16: min_top_inferers_to_reward already set, skipping backfill",
			"value", params.MinTopInferersToReward,
		)
		return nil
	}

	// Defensive: clamp to the ceiling, so a chain whose ceiling is below the
	// default floor cannot fail params validation and halt the upgrade.
	params.MinTopInferersToReward = min(emissionstypes.DefaultMinTopInferersToReward, params.MaxTopInferersToReward)
	if err := params.Validate(); err != nil {
		return errorsmod.Wrap(err, "MIGRATION V16: backfilled params failed validation")
	}
	if err := emissionsKeeper.SetParams(ctx, params); err != nil {
		return errorsmod.Wrap(err, "MIGRATION V16: failed to persist backfilled params")
	}
	ctx.Logger().Info(
		"MIGRATION V16: params backfill completed",
		"minTopInferersToReward", params.MinTopInferersToReward,
	)
	return nil
}

// MigrateTopics backfills Topic.MaxTopInferersToReward, which topics persisted
// before the field existed decode as zero, with the current global
// Params.MaxTopInferersToReward. Topics already carrying a value are skipped, so
// the run is idempotent.
//
// A degenerate zero global is repaired to the module default first. That should
// be unreachable now that the params validator rejects zero, but it would leave
// every topic at zero and has already zeroed the scores.go retention window.
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

		// No per-topic re-validation here: the backfilled cap equals the
		// current global Params.MaxTopInferersToReward, which is exactly the
		// value Topic.Validate treats as the ceiling, so the new field is
		// valid by construction (>= 1 and <= the global) regardless of any
		// other field on the topic. Running full Topic.Validate would also
		// re-check unrelated, param-dependent fields (e.g. ground-truth-lag
		// vs epoch-length bounds) that a pre-existing dormant topic could
		// violate after some other param was tightened over time; halting the
		// whole chain upgrade for a reason unrelated to this backfill is
		// avoided by deliberately skipping that broader check.

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
