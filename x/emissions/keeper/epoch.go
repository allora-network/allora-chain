package keeper

import (
	"context"

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

// StartEpoch initializes a new epoch for the given topic, and schedules its lifecycle tasks starting at the current block time.
func (k *Keeper) StartEpoch(ctx context.Context, topicID TopicId) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	topic, err := k.GetTopic(ctx, topicID)
	if err != nil {
		return err
	}

	// TODO: get last nonce from topic?
	lastNonce := types.ZeroNonce()
	nonce := lastNonce.NextNonce()

	epoch := types.NewEpoch(nonce, topic, sdkCtx.BlockTime())
	k.epochFSMEngine.Init(&epoch) // Unnecessary but more formal

	// TODO: update topic last nonce?
	// topic.LastNonce = nonce

	if err := k.epochs.Set(ctx, epoch.Key(), epoch); err != nil {
		return err
	}
	if err := k.topics.Set(ctx, topic.Id, topic); err != nil {
		return err
	}

	return k.scheduleEpochLifecycle(ctx, epoch)
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
	return nil
}

func (k *Keeper) closeWorkerWindow(ctx context.Context, epoch types.Epoch) error {
	return nil
}

func (k *Keeper) openReputerWindow(ctx context.Context, epoch types.Epoch) error {
	return nil
}

func (k *Keeper) closeReputerWindow(ctx context.Context, epoch types.Epoch) error {
	return nil
}

func (k *Keeper) completeEpoch(ctx context.Context, epoch types.Epoch) error {
	return nil
}

func (k *Keeper) cancelEpoch(ctx context.Context, epoch types.Epoch) error {
	return nil
}

// wrapEpochTransitionFn is used to map epoch transition functions that take epoch by value to the signature required by
// the FSM engine transitions table, that takes reference. This is a guard against side effects on the epoch state
// within transition functions.
func wrapEpochTransitionFn(transitionFn func(ctx context.Context, epoch types.Epoch) error) func(ctx context.Context, epoch *types.Epoch) error {
	return func(ctx context.Context, epoch *types.Epoch) error {
		return transitionFn(ctx, *epoch)
	}
}
