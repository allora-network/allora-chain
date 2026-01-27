package keeper

import (
	"context"

	"github.com/allora-network/allora-chain/x/emissions/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
)

// TaskHandlers returns all the task handlers related to the x/emissions module.
func (k *Keeper) TaskHandlers() schedulertypes.TaskHandlers {
	return schedulertypes.TaskHandlers{
		schedulertypes.NewTaskHandler(
			types.CloseEpochWorkerWindowTask,
			nil,
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
