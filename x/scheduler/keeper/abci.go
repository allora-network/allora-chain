package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (k *Keeper) runTask(ctx context.Context, task types.Task, handler types.TaskHandler) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	task.RunCount++
	if err := handler.Run(ctx, k.cdc, task.Id, task.Args, task.RunCount); err != nil {
		return errors.Wrapf(types.ErrTaskExecution, "run func failed for task '%s' of type '%s': %s", task.Id, handler.Typename(), err)
	}

	// if the task is periodic, update its last run time and next run time, and reschedule it.
	if task.Interval != nil {
		if err := k.tasksSchedule.Remove(ctx, collections.Join3(task.Typename, task.NextRunAt, task.Id)); err != nil {
			return errors.Wrapf(types.ErrTaskExecution, "couldn't reschedule periodic task '%s' of type '%s': %s", task.Id, handler.Typename(), err)
		}

		runTime := sdkCtx.BlockTime()
		task.LastRunAt = &runTime
		task.NextRunAt = runTime.Add(*task.Interval)
		if err := k.tasks.Set(ctx, task.Id, task); err != nil {
			return errors.Wrapf(types.ErrTaskExecution, "couldn't reschedule periodic task '%s' of type '%s': %s", task.Id, handler.Typename(), err)
		}

		if err := k.tasksSchedule.Set(ctx, collections.Join3(task.Typename, task.NextRunAt, task.Id)); err != nil {
			return errors.Wrapf(types.ErrTaskExecution, "couldn't reschedule periodic task '%s' of type '%s': %s", task.Id, handler.Typename(), err)
		}
	} else {
		if err := k.CancelTask(ctx, task.Id); err != nil {
			return errors.Wrapf(types.ErrTaskExecution, "couldn't remove task '%s' of type '%s': %s", task.Id, handler.Typename(), err)
		}
	}

	return nil
}
