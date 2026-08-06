package keeper

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SpikeWindowPolicy selects how reconstructed Epoch wall-clock windows are derived.
// Historical BlockTime at the legacy nonce height is not in app state, so any policy
// approximates. See ENGN-9094 / .reports/engn-4253-epoch-reconstruct-spike.md.
type SpikeWindowPolicy int

const (
	// SpikeWindowPolicyNewEpochMath uses types.NewEpoch (reputer opens at GTL+ExtraLag).
	SpikeWindowPolicyNewEpochMath SpikeWindowPolicy = iota
	// SpikeWindowPolicyLegacyAligned matches BlockWithin*SubmissionWindowOfNonce when
	// assuming 1 second per block (reputer opens at GTL, closes at GTL+ExtraLag+EpochLength).
	SpikeWindowPolicyLegacyAligned
)

// ReconstructTopicInFlightEpochsSpike is a spike prototype (ENGN-9094): for each
// unfulfilled worker/reputer nonce on the topic, invent a NonceV2 Epoch with
// StartBlockHeight = legacy BlockHeight, infer FSM state, approximate windows with
// 1s/block from upgrade BlockTime, and schedule remaining lifecycle tasks.
//
// Does not re-run openWorkerWindow / openReputerWindow (nonces already exist).
// Marks the topic scheduler-managed via AllocateNextEpochNonce.
//
// Not production migration code.
func (k *Keeper) ReconstructTopicInFlightEpochsSpike(
	ctx context.Context,
	topicID TopicId,
	policy SpikeWindowPolicy,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
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

	// Deterministic order: ascending legacy height.
	heights := make([]int64, 0, len(legacyHeights))
	for h := range legacyHeights { //nolint:maprange // sorted below
		heights = append(heights, h)
	}
	sort.Slice(heights, func(i, j int) bool { return heights[i] < heights[j] })

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

		state := inferEpochStateSpike(topic, legacyHeight, currentHeight, workerOpen, reputerOpen)
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
		startAt := now.Add(-time.Duration(blocksElapsed) * time.Second)

		epoch := buildReconstructedEpochSpike(nonce, topic, topicID, legacyHeight, startAt, state, policy)
		if err := k.epochs.Set(ctx, epoch.Key(), epoch); err != nil {
			return err
		}
		if err := k.scheduleRemainingEpochLifecycleSpike(ctx, epoch); err != nil {
			return err
		}
	}
	return nil
}

func inferEpochStateSpike(
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
		// Overdue worker close; treat as waiting GT so close can still run via soft-fail path,
		// or as reputer if reputer nonce already present.
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

func buildReconstructedEpochSpike(
	nonce types.NonceV2,
	topic types.Topic,
	topicID TopicId,
	legacyHeight int64,
	startAt time.Time,
	state types.EpochState,
	policy SpikeWindowPolicy,
) types.Epoch {
	extraLag := types.TopicExtraLag(topic)
	var epoch types.Epoch
	switch policy {
	case SpikeWindowPolicyLegacyAligned:
		reputerOpenAt := startAt.Add(time.Duration(topic.GroundTruthLag) * time.Second)
		epoch = types.Epoch{
			Nonce:   nonce,
			TopicId: topicID,
			State:   state,
			WorkerSubmissionWindow: &types.Window{
				OpenAt:  startAt,
				CloseAt: startAt.Add(time.Duration(topic.WorkerSubmissionWindow) * time.Second),
			},
			ReputerSubmissionWindow: &types.Window{
				OpenAt:  reputerOpenAt,
				CloseAt: reputerOpenAt.Add(time.Duration(extraLag+topic.EpochLength) * time.Second),
			},
			Epsilon: topic.Epsilon,
		}
	default:
		epoch = types.NewEpoch(nonce, topic, startAt)
		epoch.TopicId = topicID
		epoch.State = state
	}
	epoch.StartBlockHeight = legacyHeight
	return epoch
}

// scheduleRemainingEpochLifecycleSpike schedules only transitions that still remain
// given the reconstructed Epoch.State (worker/reputer windows already open in legacy).
func (k *Keeper) scheduleRemainingEpochLifecycleSpike(ctx context.Context, epoch types.Epoch) error {
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

// CompareReconstructedWindowsSpike returns whether reconstructed windows contain `now`
// for worker and reputer, and whether legacy height math would also accept at currentHeight.
// Used by spike tests to quantify policy gaps.
func CompareReconstructedWindowsSpike(
	topic types.Topic,
	legacyHeight int64,
	currentHeight int64,
	now time.Time,
	epoch types.Epoch,
) (epochWorkerOK, epochReputerOK, heightWorkerOK, heightReputerOK bool, err error) {
	epochWorkerOK = epoch.State == types.EpochState_WORKER_SUBMISSION &&
		TimeWithinWindow(now, epoch.WorkerSubmissionWindow)
	epochReputerOK = epoch.State == types.EpochState_REPUTER_SUBMISSION &&
		TimeWithinWindow(now, epoch.ReputerSubmissionWindow)

	heightWorkerOK, err = BlockWithinWorkerSubmissionWindowOfNonce(
		topic, types.Nonce{BlockHeight: legacyHeight}, currentHeight,
	)
	if err != nil {
		return
	}
	heightReputerOK, err = BlockWithinReputerSubmissionWindowOfNonce(
		topic,
		types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: legacyHeight}},
		currentHeight,
	)
	return
}
