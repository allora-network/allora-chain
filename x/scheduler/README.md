# Scheduler Module

The `x/scheduler` module allows other modules to schedule tasks for deferred execution at specific times. It handles 
task scheduling while delegating the actual execution logic to modules through registered task type handlers.
Tasks are automatically executed in an order compliant with potential dependencies between tasks, and modules can 
dynamically decide to execute, postpone, or reject their own tasks.

## Usage

This section aims to provide guidelines on how to leverage the `x/scheduler` module from another module.

### Registering a Task Handler

Each task is of a certain type, so the first step is to register a task handler to manage this task type.

Assuming we want to create a task type `my_task` in the `x/my_module` module, we would do the following:

**1. Create the task type arguments proto message (i.e. in any):**

Create in your proto API a `tasks.proto` file with the needed message definition, for example:
```proto
message MyTaskArgs {
  string input_1 = 1;
  string input_2 = 2;
}
```

**2. Add the task type name in `x/my_module/types/keys.go`:**

```go
const (
	ModuleName = "my_module"
    TaskMyTask = ModuleName + ":my_task"
)
```

**3. Implement the task handler in `x/my_module/keeper/tasks.go`:**

```go
func (k *Keeper) TaskHandlers() schedulertypes.TaskHandlers {
    return schedulertypes.TaskHandlers{
        schedulertypes.NewTaskHandler[*types.MyTaskArgs](
            types.TaskMyTask, // Task type name
            nil,              // Dependencies, if any
			func(ctx context.Context, tasks []Invocation[*types.MyTaskArgs]) ([]ArbitrageDecision, error) {
				return nil, nil // Arbitrage func
            },
            func(ctx context.Context, id schedulertypes.TaskID, args *types.MyTaskArgs, runCount uint64) error {
                return nil // Run func
            },
        ),
    }
}
```

**4. Register the task handler in `x/my_module/module/depinject.go`:**

Update the module outputs to indicate there are task handlers to register:
```go
type ModuleOutputs struct {
	depinject.Out

	Module    appmodule.AppModule
	Keeper    keeper.Keeper
    TaskHandlers schedulertypes.TaskHandlers
}
```

And register the task handlers in the `ProvideModule` function:
```go
func ProvideModule(in ModuleInputs) ModuleOutputs {
	k := keeper.NewKeeper(...)
	m := NewAppModule(...)

	return ModuleOutputs{Module: m, Keeper: k, TaskHandlers: k.TaskHandlers(), Out: depinject.Out{}}
}
```

## Details

This section aims to provide more in-depth information about the `x/scheduler` internals.

### Entities

There are two main entities in the `x/scheduler` module:

- **Task**: The execution unit that is scheduled for deferred execution.
- **TaskHandler**: The logic that defines how a task of a specific type is executed.

#### Task Handler

A `TaskHandler` (i.e., anything implementing `types.TaskHandler`) is attached to a specific task type and 
provides both the logic to execute the task and the task type specifications.

A task type is specified by:
- **Typename**: A unique identifier for the task type;
- **Dependencies**: A list of task types it depends on, defining execution order;
- **Arguments**: A type used to provide arguments to the task execution logic, which must implement `proto.Message`;

The attached execution logic is defined by:
- **Arbitrage function**: A function called before task invocation with the set of tasks that are due at this moment, and allow to take arbitrage decisions on whether to execute specific tasks or not;
- **Run function**: The actual logic to execute the task;

The `TaskHandlers` must be registered in the `x/scheduler` module (i.e., see the usage section above), when registering 
the handlers in `keeper.Keeper#RegisterTaskHandlers` the related DAG is computed to determine the task types execution order.
All the `TaskHandler` are then kept in-memory in the `keeper.Keeper`.

#### Task

A `Task` is defined by:
- **ID**: A string identifier that must be unique across the different task types;
- **Type**: The type of the task, which must point to the registered `TaskHandler` to be used;
- **Arguments**: The arguments to be passed to the task execution logic, which must implement `proto.Message`;

In the task scheduling, we'll identify two kinds of tasks:
- **One-shot task**: Tasks that are due for a single execution at the specified time. They can be scheduled using `keeper.Keeper#ScheduleTask`;
- **Periodic task**: Tasks that are executed recurrently at each specified interval. They can be scheduled using `keeper.Keeper#SchedulePeriodicTask`;

These two types share the same underlying `types.Task` structure, the presence of the `interval` field will denote a periodic task.

Internally, the `x/scheduler` module keeps tracks of executions with information like the execution counter and the last 
execution time.

### State

#### Stores

The keeper currently uses two stores to keep track of the tasks and their scheduling.

**1. The `tasks` store:**

The `tasks` store holds all the tasks mapped by their `TaskID`.

This store is an `IndexedMap` as it also contains an index on the tasks types, allowing to retrieve all the tasks of a 
specific type. This could have been achieved using two separate stores, but the `IndexedMap` allows to maintain both 
stores upon insertions and deletions.

**2. The `tasks_schedule`**

The `tasks_schedule` store holds information about tasks scheduling.

This store index tasks on both their scheduled execution time and their type, allowing to retrieve all the tasks of a 
certain type that are due for execution at a specific time. To do so the store key is a triple `<type, time, id>` which
allow the usage of triple prefixes and super prefixes to efficiently query the due tasks for a specific time and type.

The store is not part of the `tasks` store indexes, as we do need to manipulate it independently. For instance, in the 
case of pausing a periodic task, the task stays in the `tasks` store but is not scheduled anymore and thus removed from
the `tasks_schedule` store.

#### Task Arguments

In the [task.proto](./proto/scheduler/v1/task.proto) the arguments are defined as a protobuf `Any`.

The `x/scheduler` module has no information related to the concrete type of arguments, it only requires it to be an 
implementation of `proto.Message`.

When scheduling a task, the validation and serialization are delegated to the `TaskHandler` associated with the task 
type. And the same applies to deserialization when executing the task.

When using the `types.NewTaskHandler` helper to create handlers, serialization, deserialization and type validations 
are automatically managed.

### Execution flow
