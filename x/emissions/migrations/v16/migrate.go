package v16

import (
	"slices"

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

// MigrateTotalSumPreviousTopicWeights reconciles the stores that represent active topics,
// then recomputes totalSumPreviousTopicWeights from topics with a valid schedule and matching
// block-bucket entry. Topics with incomplete activity state are made fully inactive.
func MigrateTotalSumPreviousTopicWeights(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	topicKeeper := emissionsKeeper.GetTopicKeeper()

	scheduledTopicIds, err := topicKeeper.GetScheduledTopicIds(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "MIGRATION V16: failed to get scheduled topic ids")
	}
	scheduledBlocks := make(map[uint64]int64, len(scheduledTopicIds))
	for _, topicId := range scheduledTopicIds {
		block, err := topicKeeper.GetTopicSchedule(ctx, topicId)
		if err != nil {
			return errorsmod.Wrapf(err, "MIGRATION V16: failed to get schedule for topic %d", topicId)
		}
		scheduledBlocks[topicId] = block
	}

	blockBuckets, err := topicKeeper.GetActiveTopicIdsByBlock(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "MIGRATION V16: failed to get active topic block buckets")
	}
	validScheduled := make(map[uint64]struct{}, len(scheduledTopicIds))
	repairedBuckets := 0
	for _, bucket := range blockBuckets {
		existing := []uint64(nil)
		if bucket.TopicIds != nil {
			existing = bucket.TopicIds.TopicIds
		}
		filtered := make([]uint64, 0, len(existing))
		for _, topicId := range existing {
			scheduledBlock, isScheduled := scheduledBlocks[topicId]
			_, alreadyListed := validScheduled[topicId]
			if !isScheduled || scheduledBlock != bucket.BlockHeight || alreadyListed {
				continue
			}
			filtered = append(filtered, topicId)
			validScheduled[topicId] = struct{}{}
		}
		if slices.Equal(existing, filtered) {
			continue
		}
		if err := topicKeeper.SetBlockToActiveTopics(ctx, bucket.BlockHeight, emissionstypes.TopicIds{TopicIds: filtered}); err != nil {
			return errorsmod.Wrapf(err, "MIGRATION V16: failed to repair active topics at block %d", bucket.BlockHeight)
		}
		if err := topicKeeper.ResetLowestActiveTopicWeightAtBlock(ctx, bucket.BlockHeight); err != nil {
			return errorsmod.Wrapf(err, "MIGRATION V16: failed to reset lowest topic weight at block %d", bucket.BlockHeight)
		}
		repairedBuckets++
	}

	removedSchedules := 0
	for _, topicId := range scheduledTopicIds {
		if _, valid := validScheduled[topicId]; valid {
			continue
		}
		if err := topicKeeper.RemoveTopicSchedule(ctx, topicId); err != nil {
			return errorsmod.Wrapf(err, "MIGRATION V16: failed to remove invalid schedule for topic %d", topicId)
		}
		removedSchedules++
	}

	activeTopicIds, err := topicKeeper.GetActiveTopicIds(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "MIGRATION V16: failed to get active topic ids")
	}
	activeSet := make(map[uint64]struct{}, len(activeTopicIds))
	removedActiveTopics := 0
	for _, topicId := range activeTopicIds {
		if _, valid := validScheduled[topicId]; !valid {
			if err := topicKeeper.RemoveTopicFromActiveSet(ctx, topicId); err != nil {
				return errorsmod.Wrapf(err, "MIGRATION V16: failed to remove topic %d from active set", topicId)
			}
			removedActiveTopics++
			continue
		}
		activeSet[topicId] = struct{}{}
	}
	validTopicIds := make([]uint64, 0, len(validScheduled))
	for topicId := range validScheduled {
		validTopicIds = append(validTopicIds, topicId)
	}
	slices.Sort(validTopicIds)
	addedActiveTopics := 0
	for _, topicId := range validTopicIds {
		if _, active := activeSet[topicId]; active {
			continue
		}
		if err := topicKeeper.SetActiveTopics(ctx, topicId); err != nil {
			return errorsmod.Wrapf(err, "MIGRATION V16: failed to add topic %d to active set", topicId)
		}
		addedActiveTopics++
	}

	recomputed := alloraMath.ZeroDec()
	for _, topicId := range validTopicIds {
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
	if !current.Equal(recomputed) {
		if err := topicKeeper.SetTotalSumPreviousTopicWeights(ctx, recomputed); err != nil {
			return errorsmod.Wrap(err, "MIGRATION V16: failed to set recomputed total sum of previous topic weights")
		}
	}
	ctx.Logger().Info(
		"MIGRATION V16: topic activity state reconciled and total sum recomputed",
		"previous", current.String(),
		"recomputed", recomputed.String(),
		"scheduledTopics", len(validTopicIds),
		"repairedBuckets", repairedBuckets,
		"removedSchedules", removedSchedules,
		"removedActiveTopics", removedActiveTopics,
		"addedActiveTopics", addedActiveTopics,
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
