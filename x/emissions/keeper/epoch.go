package keeper

import (
	"context"

	"github.com/allora-network/allora-chain/fsm"
	"github.com/allora-network/allora-chain/x/emissions/types"
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
				epochSymbolOpenWorkerWindow: {To: types.EpochState_WORKER_SUBMISSION, Action: k.openWorkerWindow},
				epochSymbolCancel:           {To: types.EpochState_CANCELLED, Action: k.cancelEpoch},
			},
			types.EpochState_WORKER_SUBMISSION: {
				epochSymbolCloseWorkerWindow: {To: types.EpochState_WAITING_GROUND_TRUTH, Action: k.closeWorkerWindow},
				epochSymbolCancel:            {To: types.EpochState_CANCELLED, Action: k.cancelEpoch},
			},
			types.EpochState_WAITING_GROUND_TRUTH: {
				epochSymbolOpenReputerWindow: {To: types.EpochState_REPUTER_SUBMISSION, Action: k.openReputerWindow},
				epochSymbolCancel:            {To: types.EpochState_CANCELLED, Action: k.cancelEpoch},
			},
			types.EpochState_REPUTER_SUBMISSION: {
				epochSymbolCloseReputerWindow: {To: types.EpochState_PENDING_COMPLETION, Action: k.closeReputerWindow},
				epochSymbolCancel:             {To: types.EpochState_CANCELLED, Action: k.cancelEpoch},
			},
			types.EpochState_PENDING_COMPLETION: {
				epochSymbolComplete: {To: types.EpochState_COMPLETED, Action: k.completeEpoch},
				epochSymbolCancel:   {To: types.EpochState_CANCELLED, Action: k.cancelEpoch},
			},
		},
	)
	if err != nil {
		panic("failed to setup epoch FSM engine: " + err.Error())
	}

	k.epochFSMEngine = epochFSMEngine
}

func (k *Keeper) applyEpochTransition(ctx context.Context, topicID TopicId, nonce types.NonceV2, symbol EpochFSMSymbol) error {
	// TODO: fetch epoch from state
	epoch := types.Epoch{}

	if err := k.epochFSMEngine.Consume(ctx, &epoch, symbol); err != nil {
		return err
	}

	if k.epochFSMEngine.Terminated(&epoch) {
		// TODO: delete epoch
		return nil
	} else {
		// TODO: save epoch
		return nil
	}
}

func (k *Keeper) openWorkerWindow(ctx context.Context, epoch *types.Epoch) error {
	return nil
}

func (k *Keeper) closeWorkerWindow(ctx context.Context, epoch *types.Epoch) error {
	return nil
}

func (k *Keeper) openReputerWindow(ctx context.Context, epoch *types.Epoch) error {
	return nil
}

func (k *Keeper) closeReputerWindow(ctx context.Context, epoch *types.Epoch) error {
	return nil
}

func (k *Keeper) completeEpoch(ctx context.Context, epoch *types.Epoch) error {
	return nil
}

func (k *Keeper) cancelEpoch(ctx context.Context, epoch *types.Epoch) error {
	return nil
}
