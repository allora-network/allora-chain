package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"github.com/allora-network/allora-chain/x/scheduler/types"
)

func (k *Keeper) InitGenesis(ctx context.Context, state types.GenesisState) {
	for _, task := range state.Tasks {
		if err := k.tasks.Set(ctx, task.Id, task); err != nil {
			panic(err)
		}
		if key := getTaskScheduleKey(task); key != nil {
			if err := k.tasksSchedule.Set(ctx, *key); err != nil {
				panic(err)
			}
		}
	}
}

func (k *Keeper) ExportGenesis(ctx context.Context) types.GenesisState {
	var tasks []types.Task
	it, err := k.tasks.Iterate(ctx, &collections.Range[types.TaskID]{})
	if err != nil {
		panic(err)
	}

	for ; it.Valid(); it.Next() {
		task, err := it.Value()
		if err != nil {
			panic(err)
		}

		tasks = append(tasks, task)
	}

	return types.NewGenesisState(tasks)
}
