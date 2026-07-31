package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"github.com/allora-network/allora-chain/fsm"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type EpochFSMSymbol int

const (
	// Epoch finite state machine alphabet
	epochSymbolOpenWorkerWindow EpochFSMSymbol = iota
	epochSymbolCloseWorkerWindow
	epochSymbolOpenReputerWindow
	epochSymbolCloseReputerWindow
	epochSymbolComplete
	epochSymbolCancel
)

func (s EpochFSMSymbol) Name() string {
	switch s {
	case epochSymbolOpenWorkerWindow:
		return "OpenWorkerWindow"
	case epochSymbolCloseWorkerWindow:
		return "CloseWorkerWindow"
	case epochSymbolOpenReputerWindow:
		return "OpenReputerWindow"
	case epochSymbolCloseReputerWindow:
		return "CloseReputerWindow"
	case epochSymbolComplete:
		return "Complete"
	case epochSymbolCancel:
		return "Cancel"
	default:
		return "Unknown"
	}
}

func (k *Keeper) setupEpochFSMEngine() {
	epochFSMEngine, err := fsm.NewEngine[*types.Epoch](
		types.EpochState_INIT,
		[]fsm.State{types.EpochState_COMPLETED, types.EpochState_CANCELLED},
		fsm.TransitionsTable[*types.Epoch]{
			types.EpochState_INIT: {
				epochSymbolOpenWorkerWindow: {To: types.EpochState_WORKER_SUBMISSION, Action: wrapEpochTransitionFn(k.openWorkerWindow)},
				epochSymbolCancel:           {To: types.EpochState_CANCELLED, Action: wrapEpochTransitionFn(k.cancelEpoch)},
			},
			types.EpochState_WORKER_SUBMISSION: {
				epochSymbolCloseWorkerWindow: {To: types.EpochState_WAITING_GROUND_TRUTH, Action: wrapEpochTransitionFn(k.closeWorkerWindow)},
				epochSymbolCancel:            {To: types.EpochState_CANCELLED, Action: wrapEpochTransitionFn(k.cancelEpoch)},
			},
			types.EpochState_WAITING_GROUND_TRUTH: {
				epochSymbolOpenReputerWindow: {To: types.EpochState_REPUTER_SUBMISSION, Action: wrapEpochTransitionFn(k.openReputerWindow)},
				epochSymbolCancel:            {To: types.EpochState_CANCELLED, Action: wrapEpochTransitionFn(k.cancelEpoch)},
			},
			types.EpochState_REPUTER_SUBMISSION: {
				epochSymbolCloseReputerWindow: {To: types.EpochState_PENDING_COMPLETION, Action: wrapEpochTransitionFn(k.closeReputerWindow)},
				epochSymbolCancel:             {To: types.EpochState_CANCELLED, Action: wrapEpochTransitionFn(k.cancelEpoch)},
			},
			types.EpochState_PENDING_COMPLETION: {
				epochSymbolComplete: {To: types.EpochState_COMPLETED, Action: wrapEpochTransitionFn(k.completeEpoch)},
				epochSymbolCancel:   {To: types.EpochState_CANCELLED, Action: wrapEpochTransitionFn(k.cancelEpoch)},
			},
		},
	)

	if err != nil {
		panic(err)
	}

	k.epochFSMEngine = epochFSMEngine
}

// GetTopicLastEpochNonce returns the last allocated epoch nonce for a topic.
// found is false when no epoch nonce has been allocated for the topic yet.
func (k *Keeper) GetTopicLastEpochNonce(ctx context.Context, topicID TopicId) (nonce types.NonceV2, found bool, err error) {
	nonce, err = k.topicLastEpochNonce.Get(ctx, topicID)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.ZeroNonce(), false, nil
		}
		return types.NonceV2(0), false, err
	}
	return nonce, true, nil
}

// AllocateNextEpochNonce returns the next NonceV2 for the topic and persists it as the topic's last epoch nonce.
// The first allocation for a topic is ZeroNonce().NextNonce() (version V1, payload 1).
func (k *Keeper) AllocateNextEpochNonce(ctx context.Context, topicID TopicId) (types.NonceV2, error) {
	lastNonce, found, err := k.GetTopicLastEpochNonce(ctx, topicID)
	if err != nil {
		return types.NonceV2(0), err
	}
	if !found {
		lastNonce = types.ZeroNonce()
	}

	nextNonce := lastNonce.NextNonce()
	if err := k.topicLastEpochNonce.Set(ctx, topicID, nextNonce); err != nil {
		return types.NonceV2(0), err
	}
	return nextNonce, nil
}

// StartNewEpoch initializes a new epoch for the given topic, opens the worker window immediately,
// and schedules the remaining lifecycle transition tasks.
func (k *Keeper) StartNewEpoch(ctx context.Context, topicID TopicId) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	topic, err := k.topicKeeper.GetTopic(ctx, topicID)
	if err != nil {
		return err
	}

	nonce, err := k.AllocateNextEpochNonce(ctx, topicID)
	if err != nil {
		return err
	}

	epoch := types.NewEpoch(nonce, topic, sdkCtx.BlockTime())
	// Prefer the topicID argument: stored Topic.Id can be stale when topics are inserted via SetTopic in tests.
	epoch.TopicId = topicID
	// Transitional bridge into height-keyed APIs (see Epoch.LegacyNonce).
	epoch.StartBlockHeight = sdkCtx.BlockHeight()
	k.epochFSMEngine.Init(&epoch) // Unnecessary but more formal

	if err := k.epochs.Set(ctx, epoch.Key(), epoch); err != nil {
		return err
	}

	// Open the worker window immediately; later transitions are scheduled as tasks.
	if err := k.applyEpochTransition(ctx, topicID, nonce, epochSymbolOpenWorkerWindow); err != nil {
		return err
	}

	epoch, err = k.epochs.Get(ctx, collections.Join(topicID, nonce))
	if err != nil {
		return err
	}

	return k.scheduleEpochLifecycle(ctx, epoch)
}

// StartEpoch is an alias for StartNewEpoch kept for callers that still use the older name.
func (k *Keeper) StartEpoch(ctx context.Context, topicID TopicId) error {
	return k.StartNewEpoch(ctx, topicID)
}

// GetEpoch returns the epoch for the given topic and nonce.
func (k *Keeper) GetEpoch(ctx context.Context, topicID TopicId, nonce types.NonceV2) (types.Epoch, error) {
	return k.epochs.Get(ctx, collections.Join(topicID, nonce))
}

// OnTopicActivated starts the first epoch and registers the periodic new-epoch task for the topic.
func (k *Keeper) OnTopicActivated(ctx context.Context, topicID TopicId) error {
	if k.schedulerKeeper == nil {
		return nil
	}
	if err := k.StartNewEpoch(ctx, topicID); err != nil {
		return err
	}
	return k.schedulePeriodicNewEpoch(ctx, topicID)
}

// OnTopicInactivated cancels the periodic new-epoch task for the topic.
// In-flight epoch lifecycle tasks are left alone; cancellation of open epochs is handled separately.
func (k *Keeper) OnTopicInactivated(ctx context.Context, topicID TopicId) error {
	if k.schedulerKeeper == nil {
		return nil
	}
	return k.unschedulePeriodicNewEpoch(ctx, topicID)
}

func (k *Keeper) applyEpochTransition(ctx context.Context, topicID TopicId, nonce types.NonceV2, symbol EpochFSMSymbol) error {
	epoch, err := k.epochs.Get(ctx, collections.Join(topicID, nonce))
	if err != nil {
		return err
	}

	if err := k.epochFSMEngine.Consume(ctx, &epoch, symbol); err != nil {
		return err
	}

	if k.epochFSMEngine.Terminated(&epoch) {
		return k.epochs.Remove(ctx, epoch.Key())
	}

	return k.epochs.Set(ctx, epoch.Key(), epoch)
}

func (k *Keeper) openWorkerWindow(ctx context.Context, epoch types.Epoch) error {
	// TODO: emit evt
	return nil
}

func (k *Keeper) closeWorkerWindow(ctx context.Context, epoch types.Epoch) error {
	// TODO: emit evt & compute network inference
	return nil
}

func (k *Keeper) openReputerWindow(ctx context.Context, epoch types.Epoch) error {
	// TODO: emit evt
	return nil
}

func (k *Keeper) closeReputerWindow(ctx context.Context, epoch types.Epoch) error {
	// TODO: emit evt
	return nil
}

func (k *Keeper) completeEpoch(ctx context.Context, epoch types.Epoch) error {
	// TODO: compute loss, distribute reward, prune epoch data
	return nil
}

func (k *Keeper) cancelEpoch(ctx context.Context, epoch types.Epoch) error {
	if epoch.State == types.EpochState_WORKER_SUBMISSION {
		// TODO: emit closing worker window evt
	} else if epoch.State == types.EpochState_REPUTER_SUBMISSION {
		// TODO: emit closing reputer window evt
	}

	// TODO: prune epoch data

	return k.unscheduleEpochLifecycle(ctx, epoch)
}

// wrapEpochTransitionFn is used to map epoch transition functions that take epoch by value to the signature required by
// the FSM engine transitions table, that takes reference. This is a guard against side effects on the epoch state
// within transition functions.
func wrapEpochTransitionFn(transitionFn func(ctx context.Context, epoch types.Epoch) error) func(ctx context.Context, epoch *types.Epoch) error {
	return func(ctx context.Context, epoch *types.Epoch) error {
		return transitionFn(ctx, *epoch)
	}
}
