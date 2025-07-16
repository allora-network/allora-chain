# Scheduler Module

The `x/scheduler` module allows Cosmos SDK modules to schedule tasks for deferred execution at specific times.
It handles task scheduling while delegating the actual execution logic to modules through registered task type handlers.
Tasks are automatically executed in an order compliant with potential dependencies between tasks, and module can 
dynamically decide to execute, postpone, or reject its own tasks.

## Usage

This section aims to provide guidelines on how to leverage the `x/scheduler` module from another module.

### Registering a Task Type

Each task is of a certain type, so the first step is to register a new task type.

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

**3. Implement the task type in `x/my_module/keeper/tasks.go`:**

```go
func (k *Keeper) TaskHandlers() schedulertypes.TaskHandlers {
    return schedulertypes.TaskHandlers{
        schedulertypes.NewTaskHandler[*types.MyTaskArgs](
            types.TaskMyTask, // Task name
            nil,              // Dependencies, if any
            nil,              // Arbitrage func
            func(ctx context.Context, id schedulertypes.TaskID, args *types.MyTaskArgs, runCount uint64) error {
                return nil // Run func
            },
        ),
    }
}
```

**4. Register the task type in `x/my_module/module/depinject.go`:**

Update the module outputs to indicates there is task types to register:
```go
type ModuleOutputs struct {
	depinject.Out

	Module    appmodule.AppModule
	Keeper    keeper.Keeper
    TaskHandlers schedulertypes.TaskHandlers
}
```

And register the task types in the `ProvideModule` function:
```go
func ProvideModule(in ModuleInputs) ModuleOutputs {
	k := keeper.NewKeeper(...)
	m := NewAppModule(...)

	return ModuleOutputs{Module: m, Keeper: k, TaskHandlers: k.TaskHandlers(), Out: depinject.Out{}}
}
```
