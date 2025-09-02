package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlock processes due tasks at the beginning of each block.
// It iterates through registered task handlers in their predefined order.
// For each task type, it retrieves tasks that are due to run at the current block time.
// It then invokes the handler's Arbitrate method to determine if any arbitrage actions are needed.
// If no arbitrage action is taken, it proceeds to execute the task using the handler's Run method.
func (k *Keeper) BeginBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	for _, taskType := range k.handlersOrder {
		handler := k.handlersByTypename[taskType]
		taskIDs, err := k.GetDueTasksAt(ctx, taskType, sdkCtx.BlockTime())
		if err != nil {
			return err
		}

		tasks := make([]types.Task, 0, len(taskIDs))
		for _, id := range taskIDs {
			task, err := k.tasks.Get(ctx, id)
			if err != nil {
				return err
			}
			tasks = append(tasks, task)
		}

		decisions, err := handler.Arbitrate(ctx, k.cdc, tasks)
		if err != nil {
			return errors.Wrapf(types.ErrTaskArbitrage, "arbitrate func failed for type '%s': %s", taskType, err)
		}

		for _, task := range tasks {
			if decision, ok := decisions[task.Id]; ok {
				if err := k.applyArbitrageDecision(ctx, task.Id, decision); err != nil {
					return errors.Wrapf(types.ErrTaskArbitrage, "could not apply arbitrage decision for task '%s' of type '%s': %s", task.Id, taskType, err)
				}
				continue
			}

			if err := k.runTask(ctx, task, handler); err != nil {
				return err
			}
		}
	}

	return nil
}

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

	// If the task is periodic, update its last run time and next run time, and reschedule it.
	nextRunTime := task.NextRun(sdkCtx)
	if nextRunTime != nil {
		if err := k.tasksSchedule.Remove(ctx, collections.Join3(task.Typename, *task.NextRunAt, task.Id)); err != nil {
			return errors.Wrapf(types.ErrTaskExecution, "couldn't reschedule periodic task '%s' of type '%s': %s", task.Id, handler.Typename(), err)
		}

		runTime := sdkCtx.BlockTime()
		task.LastRunAt = &runTime
		task.NextRunAt = nextRunTime
		if err := k.tasks.Set(ctx, task.Id, task); err != nil {
			return errors.Wrapf(types.ErrTaskExecution, "couldn't reschedule periodic task '%s' of type '%s': %s", task.Id, handler.Typename(), err)
		}

		if err := k.tasksSchedule.Set(ctx, collections.Join3(task.Typename, *task.NextRunAt, task.Id)); err != nil {
			return errors.Wrapf(types.ErrTaskExecution, "couldn't reschedule periodic task '%s' of type '%s': %s", task.Id, handler.Typename(), err)
		}
	} else {
		if err := k.CancelTask(ctx, task.Id); err != nil {
			return errors.Wrapf(types.ErrTaskExecution, "couldn't remove task '%s' of type '%s': %s", task.Id, handler.Typename(), err)
		}
	}

	return nil
}
