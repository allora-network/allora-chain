package keeper

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AssumedBlockTimeSeconds is the migration estimate for converting legacy
// block-height topic timing into wall-clock seconds. Same class of estimate
// used off-chain today when setting EpochLength in blocks.
const AssumedBlockTimeSeconds int64 = 6

// MigrateToSchedulerEpochs performs the epoch scheduler cutover migration:
//  1. Reconstruct in-flight epochs from unfulfilled nonces (legacy-aligned windows).
//  2. Convert topic timing fields (and MinEpochLength) from blocks to seconds.
//  3. Enroll active topics onto periodic StartNewEpoch (and start one if none live).
//
// Topics already scheduler-managed are skipped (idempotent / feature-branch safe).
func (k *Keeper) MigrateToSchedulerEpochs(ctx context.Context) error {
	return k.MigrateToSchedulerEpochsWithBlockTime(ctx, AssumedBlockTimeSeconds)
}

// MigrateToSchedulerEpochsWithBlockTime is the testable entry point for MigrateToSchedulerEpochs.
func (k *Keeper) MigrateToSchedulerEpochsWithBlockTime(ctx context.Context, assumedBlockSecs int64) error {
	if assumedBlockSecs <= 0 {
		return fmt.Errorf("assumed block time seconds must be positive: %d", assumedBlockSecs)
	}
	blockTime := time.Duration(assumedBlockSecs) * time.Second

	nextTopicID, err := k.topicKeeper.GetNextTopicId(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "migrate scheduler epochs: next topic id")
	}

	// Topics already on the scheduler path are left alone (params already wall-clock).
	var legacyTopicIDs []TopicId
	for topicID := uint64(1); topicID < nextTopicID; topicID++ {
		exists, err := k.topicKeeper.TopicExists(ctx, topicID)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}

		managed, err := k.IsTopicSchedulerManaged(ctx, topicID)
		if err != nil {
			return err
		}
		if managed {
			continue
		}
		legacyTopicIDs = append(legacyTopicIDs, topicID)

		active, err := k.topicKeeper.IsTopicActive(ctx, topicID)
		if err != nil {
			return err
		}
		// Only reconstruct live windows for active topics. Inactive leftovers stay
		// on the unmanaged path; converting their timing still happens below so
		// reactivation sees second-valued params.
		if !active {
			continue
		}
		if err := k.reconstructTopicInFlightEpochs(ctx, topicID, blockTime); err != nil {
			return errorsmod.Wrapf(err, "reconstruct in-flight epochs for topic %d", topicID)
		}
	}

	for _, topicID := range legacyTopicIDs {
		if err := k.convertTopicTimingToSeconds(ctx, topicID, assumedBlockSecs); err != nil {
			return errorsmod.Wrapf(err, "convert topic %d timing to seconds", topicID)
		}
	}
	// Convert MinEpochLength when any legacy topic remains, or when the chain has
	// no topics yet (nextTopicID == 1). Skip when every existing topic is already
	// scheduler-managed so a rerun does not scale the param twice.
	if len(legacyTopicIDs) > 0 || nextTopicID == 1 {
		if err := k.convertMinEpochLengthToSeconds(ctx, assumedBlockSecs); err != nil {
			return err
		}
	}

	for _, topicID := range legacyTopicIDs {
		active, err := k.topicKeeper.IsTopicActive(ctx, topicID)
		if err != nil {
			return err
		}
		if !active {
			continue
		}
		if err := k.enrollActiveTopicAfterMigration(ctx, topicID); err != nil {
			return errorsmod.Wrapf(err, "enroll active topic %d after migration", topicID)
		}
	}
	return nil
}

func (k *Keeper) reconstructTopicInFlightEpochs(
	ctx context.Context,
	topicID TopicId,
	blockTime time.Duration,
) error {
	topic, err := k.topicKeeper.GetTopic(ctx, topicID)
	if err != nil {
		return err
	}

	workerNonces, err := k.nonceKeeper.GetUnfulfilledWorkerNonces(ctx, topicID)
	if err != nil {
		return err
	}
	reputerNonces, err := k.nonceKeeper.GetUnfulfilledReputerNonces(ctx, topicID)
	if err != nil {
		return err
	}

	legacyHeights := make(map[int64]struct{})
	for _, n := range workerNonces.Nonces {
		if n != nil {
			legacyHeights[n.BlockHeight] = struct{}{}
		}
	}
	for _, n := range reputerNonces.Nonces {
		if n != nil && n.ReputerNonce != nil {
			legacyHeights[n.ReputerNonce.BlockHeight] = struct{}{}
		}
	}
	if len(legacyHeights) == 0 {
		return nil
	}

	heights := make([]int64, 0, len(legacyHeights))
	for h := range legacyHeights { //nolint:maprange // sorted below
		heights = append(heights, h)
	}
	sort.Slice(heights, func(i, j int) bool { return heights[i] < heights[j] })

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()
	currentHeight := sdkCtx.BlockHeight()

	for _, legacyHeight := range heights {
		workerOpen := false
		for _, n := range workerNonces.Nonces {
			if n != nil && n.BlockHeight == legacyHeight {
				workerOpen = true
				break
			}
		}
		reputerOpen := false
		for _, n := range reputerNonces.Nonces {
			if n != nil && n.ReputerNonce != nil && n.ReputerNonce.BlockHeight == legacyHeight {
				reputerOpen = true
				break
			}
		}

		// Topic timing fields are still in blocks here (conversion runs after reconstruct).
		state := inferMigratedEpochState(topic, legacyHeight, currentHeight, workerOpen, reputerOpen)
		if state == types.EpochState_COMPLETED || state == types.EpochState_CANCELLED {
			continue
		}

		nonce, err := k.AllocateNextEpochNonce(ctx, topicID)
		if err != nil {
			return err
		}

		blocksElapsed := currentHeight - legacyHeight
		if blocksElapsed < 0 {
			blocksElapsed = 0
		}
		startAt := now.Add(-time.Duration(blocksElapsed) * blockTime)
		epoch := buildLegacyAlignedMigratedEpoch(nonce, topic, topicID, legacyHeight, startAt, state, blockTime)
		if err := k.epochs.Set(ctx, epoch.Key(), epoch); err != nil {
			return err
		}
		if err := k.scheduleRemainingEpochLifecycle(ctx, epoch); err != nil {
			return err
		}
	}
	return nil
}

func inferMigratedEpochState(
	topic types.Topic,
	legacyHeight, currentHeight int64,
	workerOpen, reputerOpen bool,
) types.EpochState {
	extraLag := types.TopicExtraLag(topic)
	workerCloseHeight := legacyHeight + topic.WorkerSubmissionWindow
	reputerOpenHeight := legacyHeight + topic.GroundTruthLag
	reputerCloseHeight := reputerOpenHeight + extraLag + topic.EpochLength

	switch {
	case workerOpen && currentHeight <= workerCloseHeight:
		return types.EpochState_WORKER_SUBMISSION
	case workerOpen && currentHeight > workerCloseHeight:
		if reputerOpen {
			if currentHeight < reputerOpenHeight {
				return types.EpochState_WAITING_GROUND_TRUTH
			}
			if currentHeight <= reputerCloseHeight {
				return types.EpochState_REPUTER_SUBMISSION
			}
			return types.EpochState_PENDING_COMPLETION
		}
		return types.EpochState_WAITING_GROUND_TRUTH
	case reputerOpen && currentHeight < reputerOpenHeight:
		return types.EpochState_WAITING_GROUND_TRUTH
	case reputerOpen && currentHeight <= reputerCloseHeight:
		return types.EpochState_REPUTER_SUBMISSION
	case reputerOpen:
		return types.EpochState_PENDING_COMPLETION
	default:
		return types.EpochState_COMPLETED
	}
}

func buildLegacyAlignedMigratedEpoch(
	nonce types.NonceV2,
	topic types.Topic,
	topicID TopicId,
	legacyHeight int64,
	startAt time.Time,
	state types.EpochState,
	blockTime time.Duration,
) types.Epoch {
	extraLag := types.TopicExtraLag(topic)
	reputerOpenAt := startAt.Add(time.Duration(topic.GroundTruthLag) * blockTime)
	epoch := types.Epoch{
		Nonce:   nonce,
		TopicId: topicID,
		State:   state,
		WorkerSubmissionWindow: &types.Window{
			OpenAt:  startAt,
			CloseAt: startAt.Add(time.Duration(topic.WorkerSubmissionWindow) * blockTime),
		},
		ReputerSubmissionWindow: &types.Window{
			OpenAt:  reputerOpenAt,
			CloseAt: reputerOpenAt.Add(time.Duration(extraLag+topic.EpochLength) * blockTime),
		},
		Epsilon:          topic.Epsilon,
		StartBlockHeight: legacyHeight,
	}
	return epoch
}

// scheduleRemainingEpochLifecycle schedules only transitions that still remain given Epoch.State.
func (k *Keeper) scheduleRemainingEpochLifecycle(ctx context.Context, epoch types.Epoch) error {
	if k.schedulerKeeper == nil {
		return nil
	}
	taskIDSuffix := fmt.Sprintf(":%d-%d", epoch.TopicId, epoch.Nonce)
	args := &types.EpochTransitionTaskArgs{
		TopicId: epoch.TopicId,
		Nonce:   epoch.Nonce,
	}

	schedule := func(taskType string, at time.Time) error {
		return k.schedulerKeeper.ScheduleTask(
			ctx,
			taskType,
			schedulertypes.TaskID(taskType+taskIDSuffix),
			args,
			schedulertypes.ScheduleAt(at),
		)
	}

	switch epoch.State {
	case types.EpochState_WORKER_SUBMISSION:
		if err := schedule(types.CloseEpochWorkerWindowTask, epoch.WorkerSubmissionWindow.CloseAt); err != nil {
			return err
		}
		fallthrough
	case types.EpochState_WAITING_GROUND_TRUTH:
		if err := schedule(types.OpenEpochReputerWindowTask, epoch.ReputerSubmissionWindow.OpenAt); err != nil {
			return err
		}
		fallthrough
	case types.EpochState_REPUTER_SUBMISSION:
		if err := schedule(types.CloseEpochReputerWindowTask, epoch.ReputerSubmissionWindow.CloseAt); err != nil {
			return err
		}
		return schedule(types.CompleteEpochTask, epoch.ReputerSubmissionWindow.CloseAt)
	case types.EpochState_PENDING_COMPLETION:
		return schedule(types.CompleteEpochTask, sdk.UnwrapSDKContext(ctx).BlockTime())
	default:
		return nil
	}
}

func (k *Keeper) convertTopicTimingToSeconds(ctx context.Context, topicID TopicId, assumedBlockSecs int64) error {
	topic, err := k.topicKeeper.GetTopic(ctx, topicID)
	if err != nil {
		return err
	}
	converted, err := scaleTopicTimingBlocksToSeconds(topic, assumedBlockSecs)
	if err != nil {
		return err
	}
	return k.topicKeeper.SetTopic(ctx, topicID, converted)
}

func scaleTopicTimingBlocksToSeconds(topic types.Topic, assumedBlockSecs int64) (types.Topic, error) {
	epochLength, err := mulPositiveOrZero(topic.EpochLength, assumedBlockSecs)
	if err != nil {
		return topic, errorsmod.Wrap(err, "EpochLength")
	}
	gtl, err := mulPositiveOrZero(topic.GroundTruthLag, assumedBlockSecs)
	if err != nil {
		return topic, errorsmod.Wrap(err, "GroundTruthLag")
	}
	wsw, err := mulPositiveOrZero(topic.WorkerSubmissionWindow, assumedBlockSecs)
	if err != nil {
		return topic, errorsmod.Wrap(err, "WorkerSubmissionWindow")
	}
	topic.EpochLength = epochLength
	topic.GroundTruthLag = gtl
	topic.WorkerSubmissionWindow = wsw
	return topic, nil
}

func (k *Keeper) convertMinEpochLengthToSeconds(ctx context.Context, assumedBlockSecs int64) error {
	params, err := k.paramsKeeper.GetParams(ctx)
	if err != nil {
		return err
	}
	scaled, err := mulPositiveOrZero(params.MinEpochLength, assumedBlockSecs)
	if err != nil {
		return errorsmod.Wrap(err, "MinEpochLength")
	}
	params.MinEpochLength = scaled
	return k.paramsKeeper.SetParams(ctx, params)
}

func mulPositiveOrZero(value, factor int64) (int64, error) {
	if value == 0 {
		return 0, nil
	}
	if value < 0 || factor < 0 {
		return 0, fmt.Errorf("refusing to scale negative timing value %d by %d", value, factor)
	}
	if value > math.MaxInt64/factor {
		return 0, fmt.Errorf("scaling %d by %d would overflow int64", value, factor)
	}
	return value * factor, nil
}

func (k *Keeper) enrollActiveTopicAfterMigration(ctx context.Context, topicID TopicId) error {
	hasLive, err := k.topicHasLiveEpoch(ctx, topicID)
	if err != nil {
		return err
	}
	if !hasLive {
		if err := k.StartNewEpoch(ctx, topicID); err != nil {
			return err
		}
	}
	return k.schedulePeriodicNewEpoch(ctx, topicID)
}

func (k *Keeper) topicHasLiveEpoch(ctx context.Context, topicID TopicId) (bool, error) {
	rng := collections.NewPrefixedPairRange[TopicId, types.NonceV2](topicID)
	iter, err := k.epochs.Iterate(ctx, rng)
	if err != nil {
		return false, err
	}
	defer iter.Close()
	return iter.Valid(), nil
}
