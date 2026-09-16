package scheduler

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	"github.com/allora-network/allora-chain/x/scheduler/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: types.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Tasks",
					Use:       "tasks",
					Long: `Query the tasks registered, even if not scheduled.

Specify a 'typename' to filter results on a specific type of task.`,
					Short: "Query the tasks registered",
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"typename": {
							Usage: "Filter results on a specific type of task",
						},
					},
				},
				{
					RpcMethod: "Task",
					Use:       "task [task-id]",
					Short:     "Query a task",
					Long:      "Query details about a task given its id.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "task_id"},
					},
				},
				{
					RpcMethod: "ScheduledTasks",
					Use:       "scheduled-tasks [typename]",
					Short:     "Query the tasks scheduled per type",
					Long: `Query the tasks scheduled of a specific type, unscheduled tasks won't be returned.

Specify a 'from' to filter results from a specific scheduled time (and use the pagination order to filter before or after).`,
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "typename"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"from": {
							Usage: "Filter results from a specific scheduled time",
						},
					},
				},
				{
					RpcMethod: "Handlers",
					Use:       "handlers",
					Short:     "Query the registered task handlers",
					Long:      "Query the registered task handlers, returning their typename in the order of execution.",
				},
			},
			SubCommands:          nil,
			EnhanceCustomCommand: false,
			Short:                "Querying commands for the scheduler module",
		},
		Tx: nil,
	}
}
