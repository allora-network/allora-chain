package types

import (
	"context"
	"fmt"
	"reflect"

	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/gogoproto/proto"
)

// ArbitrageAction defines the possible actions that can be taken by the task handler during the arbitrage step.
type ArbitrageAction int

const (
	// ArbitrageActionCancel means the task shall not be executed and is removed.
	ArbitrageActionCancel ArbitrageAction = iota

	// ArbitrageActionReschedule means the task execution shall be rescheduled using the provided scheduling options,
	// if any (i.e. if none provided it's postponed next block).
	ArbitrageActionReschedule
)

type TaskHandlers []TaskHandler

// TaskHandler defines carries the logic attached to a type of task so that it can be managed by the scheduler.
// Handlers must be registered to allow the module to schedule and execute tasks of that type.
type TaskHandler interface {
	// Typename returns the unique identifier for the task type.
	Typename() string

	// DependsOn defines the task type this type depends on. Tasks of dependent types must be executed before.
	DependsOn() []string

	// PackArgs is called to pack the provided arguments into a codectypes.Any. It also has the responsibility to
	// perform any necessary validation on the arguments.
	PackArgs(args proto.Message) (*codectypes.Any, error)

	// Arbitrate is called once prior to the execution of tasks that are to be invoked at this time. Its purpose is to
	// determine the action to take for each task invocation: If no action is taken for a task, it'll be executed.
	Arbitrate(ctx context.Context, cdc codec.Codec, tasks []Task) (map[TaskID]ArbitrageDecision, error)

	// Run is called to execute the task with the provided arguments. The runCount is incremented each time the task is
	// executed (i.e., it's always 1 for non-periodic tasks).
	Run(ctx context.Context, cdc codec.Codec, id TaskID, args *codectypes.Any, runCount uint64) error
}

// Invocation represents a scheduled task invocation.
type Invocation[T proto.Message] struct {
	// TaskID is the unique identifier of the task to be invoked.
	TaskID TaskID

	// Args are the arguments to be passed to the task handler when executing the task.
	Args T
}

// ArbitrageDecision defines the decision made by the task handler about a task during the arbitrage step.
type ArbitrageDecision struct {
	// Action defines the action to take for the task.
	Action ArbitrageAction

	// RescheduleOpts carries the rescheduling information if the action is ArbitrageActionReschedule.
	RescheduleOpts []SchedulingOption
}

func NewTaskHandler[T proto.Message](
	name string,
	dependsOn []string,
	arbitrateFn func(ctx context.Context, tasks []Invocation[T]) (map[TaskID]ArbitrageDecision, error),
	runFn func(ctx context.Context, id TaskID, args T, runCount uint64) error,
) TaskHandler {
	// if arbitrateFn is nil, we provide a no-op function.
	if arbitrateFn == nil {
		arbitrateFn = func(ctx context.Context, tasks []Invocation[T]) (map[TaskID]ArbitrageDecision, error) {
			return nil, nil
		}
	}

	// if runFn is nil, we provide a no-op function.
	if runFn == nil {
		runFn = func(ctx context.Context, id TaskID, args T, runCount uint64) error {
			return nil
		}
	}

	var zeroArgs T
	if any(zeroArgs) != nil {
		typ := reflect.TypeOf(zeroArgs)
		if typ.Kind() != reflect.Ptr {
			// Should never happen
			panic("task args must be a pointer type")
		}
	}

	return taskHandler[T]{
		name:        name,
		deps:        dependsOn,
		zeroArgs:    zeroArgs,
		arbitrateFn: arbitrateFn,
		runFn:       runFn,
	}
}

func NewNoArgsTaskHandler(
	name string,
	dependsOn []string,
	arbitrateFn func(ctx context.Context, tasks []TaskID) (map[TaskID]ArbitrageDecision, error),
	runFn func(ctx context.Context, id TaskID, runCount uint64) error,
) TaskHandler {
	var wrappedArbitrateFn func(ctx context.Context, tasks []Invocation[proto.Message]) (map[TaskID]ArbitrageDecision, error)
	var wrappedRunFn func(ctx context.Context, id TaskID, args proto.Message, runCount uint64) error

	if arbitrateFn != nil {
		wrappedArbitrateFn = func(ctx context.Context, tasks []Invocation[proto.Message]) (map[TaskID]ArbitrageDecision, error) {
			taskIDs := make([]TaskID, len(tasks))
			for i, task := range tasks {
				taskIDs[i] = task.TaskID
			}
			return arbitrateFn(ctx, taskIDs)
		}
	}

	if runFn != nil {
		wrappedRunFn = func(ctx context.Context, id TaskID, args proto.Message, runCount uint64) error {
			return runFn(ctx, id, runCount)
		}
	}

	return NewTaskHandler(name, dependsOn, wrappedArbitrateFn, wrappedRunFn)
}

type taskHandler[T proto.Message] struct {
	name        string
	deps        []string
	zeroArgs    T
	arbitrateFn func(ctx context.Context, tasks []Invocation[T]) (map[TaskID]ArbitrageDecision, error)
	runFn       func(ctx context.Context, id TaskID, args T, runCount uint64) error
}

func (t taskHandler[T]) Typename() string {
	return t.name
}

func (t taskHandler[T]) DependsOn() []string {
	return t.deps
}

func (t taskHandler[T]) PackArgs(args proto.Message) (*codectypes.Any, error) {
	if args == nil && any(t.zeroArgs) == nil {
		return nil, nil
	}

	if reflect.TypeOf(args) != reflect.TypeOf(t.zeroArgs) {
		return nil, errors.Wrapf(ErrInvalidTaskArguments, "could not pack task '%s' arguments", t.name)
	}

	return codectypes.NewAnyWithValue(args)
}

func (t taskHandler[T]) UnpackArgs(cdc codec.Codec, packedArgs *codectypes.Any) (T, error) {
	var zeroArgs T
	if !t.HasArgs() {
		if packedArgs == nil {
			return zeroArgs, nil
		}
		return zeroArgs, errors.Wrapf(ErrInvalidTaskArguments, "could not unpack task '%s' arguments", t.name)
	}

	if packedArgs == nil {
		return zeroArgs, errors.Wrapf(ErrInvalidTaskArguments, "could not unpack task '%s' arguments", t.name)
	}

	typ := reflect.TypeOf(t.zeroArgs)
	val := reflect.New(typ.Elem())
	args, ok := val.Interface().(T)
	if !ok {
		// This should never happen.
		return zeroArgs, fmt.Errorf("failed to cast new task args")
	}

	if packedArgs.TypeUrl != codectypes.MsgTypeURL(args) {
		return zeroArgs, errors.Wrapf(ErrInvalidTaskArguments, "could not unpack task '%s' arguments", t.name)
	}

	return args, cdc.Unmarshal(packedArgs.Value, args)
}

func (t taskHandler[T]) Arbitrate(ctx context.Context, cdc codec.Codec, tasks []Task) (map[TaskID]ArbitrageDecision, error) {
	invocations := make([]Invocation[T], len(tasks))
	for i, task := range tasks {
		args, err := t.UnpackArgs(cdc, task.Args)
		if err != nil {
			return nil, err
		}
		invocations[i] = Invocation[T]{
			TaskID: task.Id,
			Args:   args,
		}
	}

	return t.arbitrateFn(ctx, invocations)
}

func (t taskHandler[T]) Run(ctx context.Context, cdc codec.Codec, id TaskID, packedArgs *codectypes.Any, runCount uint64) error {
	args, err := t.UnpackArgs(cdc, packedArgs)
	if err != nil {
		return err
	}

	return t.runFn(ctx, id, args, runCount)
}

func (t taskHandler[T]) HasArgs() bool {
	return any(t.zeroArgs) != nil
}
