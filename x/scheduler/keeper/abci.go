package keeper

import (
	"context"

	"github.com/allora-network/allora-chain/x/scheduler/types"
)

func (k *Keeper) applyArbitrageDecision(ctx context.Context, task types.TaskID, decision types.ArbitrageDecision) (err error) {
	switch decision.Action {
	case types.ArbitrageActionCancel:
		err = k.CancelTask(ctx, task)
	case types.ArbitrageActionPostponeAt:
		// if no postponed time is given, we do nothing, and it'll be picked up again next block.
		if decision.PostponeAt != nil {
			err = k.RescheduleTaskAt(ctx, task, *decision.PostponeAt)
		}
	case types.ArbitrageActionPause:
		err = k.PausePeriodicTask(ctx, task)
	}
	return
}
