package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
)

// TaskHandlers returns all the task handlers related to the x/emissions module.
func (k *Keeper) TaskHandlers() schedulertypes.TaskHandlers {
	return schedulertypes.TaskHandlers{
		schedulertypes.NewTaskHandler(
			types.OpenEpochWorkerWindowTask,
			nil,
			nil,
			func(ctx context.Context, task schedulertypes.Task, args *types.EpochTransitionTaskArgs) error {
				return k.applyEpochTransition(ctx, args.TopicId, args.Nonce, epochSymbolOpenWorkerWindow)
			},
		),
		schedulertypes.NewTaskHandler(
			types.CloseEpochWorkerWindowTask,
			[]string{types.OpenEpochWorkerWindowTask},
			nil,
			func(ctx context.Context, task schedulertypes.Task, args *types.EpochTransitionTaskArgs) error {
				return k.applyEpochTransition(ctx, args.TopicId, args.Nonce, epochSymbolCloseWorkerWindow)
			},
		),
		schedulertypes.NewTaskHandler(
			types.OpenEpochReputerWindowTask,
			[]string{types.CloseEpochWorkerWindowTask},
			nil,
			func(ctx context.Context, task schedulertypes.Task, args *types.EpochTransitionTaskArgs) error {
				return k.applyEpochTransition(ctx, args.TopicId, args.Nonce, epochSymbolOpenReputerWindow)
			},
		),
		schedulertypes.NewTaskHandler(
			types.CloseEpochReputerWindowTask,
			[]string{types.OpenEpochReputerWindowTask},
			nil,
			func(ctx context.Context, task schedulertypes.Task, args *types.EpochTransitionTaskArgs) error {
				return k.applyEpochTransition(ctx, args.TopicId, args.Nonce, epochSymbolCloseReputerWindow)
			},
		),
		schedulertypes.NewTaskHandler(
			types.CompleteEpochTask,
			[]string{types.CloseEpochReputerWindowTask},
			nil,
			func(ctx context.Context, task schedulertypes.Task, args *types.EpochTransitionTaskArgs) error {
				return k.applyEpochTransition(ctx, args.TopicId, args.Nonce, epochSymbolComplete)
			},
		),
	}
}

func (k *Keeper) scheduleEpochLifecycle(ctx context.Context, epoch types.Epoch) error {
	taskIDSuffix := fmt.Sprintf(":%d-%d", epoch.TopicId, epoch.Nonce)
	args := &types.EpochTransitionTaskArgs{
		TopicId: epoch.TopicId,
		Nonce:   epoch.Nonce,
	}

	if err := k.schedulerKeeper.ScheduleTask(
		ctx,
		types.OpenEpochWorkerWindowTask,
		schedulertypes.TaskID(types.OpenEpochWorkerWindowTask+taskIDSuffix),
		args,
		schedulertypes.ScheduleAt(epoch.WorkerSubmissionWindow.OpenAt),
	); err != nil {
		return err
	}

	if err := k.schedulerKeeper.ScheduleTask(
		ctx,
		types.CloseEpochWorkerWindowTask,
		schedulertypes.TaskID(types.CloseEpochWorkerWindowTask+taskIDSuffix),
		args,
		schedulertypes.ScheduleAt(epoch.WorkerSubmissionWindow.CloseAt),
	); err != nil {
		return err
	}

	if err := k.schedulerKeeper.ScheduleTask(
		ctx,
		types.OpenEpochReputerWindowTask,
		schedulertypes.TaskID(types.OpenEpochReputerWindowTask+taskIDSuffix),
		args,
		schedulertypes.ScheduleAt(epoch.ReputerSubmissionWindow.OpenAt),
	); err != nil {
		return err
	}

	if err := k.schedulerKeeper.ScheduleTask(
		ctx,
		types.CloseEpochReputerWindowTask,
		schedulertypes.TaskID(types.CloseEpochReputerWindowTask+taskIDSuffix),
		args,
		schedulertypes.ScheduleAt(epoch.ReputerSubmissionWindow.CloseAt),
	); err != nil {
		return err
	}

	return k.schedulerKeeper.ScheduleTask(
		ctx,
		types.CompleteEpochTask,
		schedulertypes.TaskID(types.CompleteEpochTask+taskIDSuffix),
		args,
		schedulertypes.ScheduleAt(epoch.ReputerSubmissionWindow.CloseAt),
	)
}

func (k *Keeper) unscheduleEpochLifecycle(ctx context.Context, epoch types.Epoch) error {
	taskIDSuffix := fmt.Sprintf(":%d-%d", epoch.TopicId, epoch.Nonce)
	for _, taskType := range []string{
		types.OpenEpochWorkerWindowTask,
		types.CloseEpochWorkerWindowTask,
		types.OpenEpochReputerWindowTask,
		types.CloseEpochReputerWindowTask,
		types.CompleteEpochTask,
	} {
		if err := k.schedulerKeeper.CancelTask(
			ctx, schedulertypes.TaskID(taskType+taskIDSuffix),
		); err != nil && !errors.Is(err, collections.ErrNotFound) {
			return err
		}
	}
	return nil
}
