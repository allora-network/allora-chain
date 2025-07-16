package types

import (
	"context"
	"fmt"
	"reflect"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/gogoproto/proto"
)

// TaskID denotes a unique identifier for a scheduled task, only one task with a given ID can be scheduled at a time,
// but an identifier may be reused later one.
type TaskID string

// ArbitrageAction defines the possible actions that can be taken by the task handler during the arbitrage step.
type ArbitrageAction int

const (
	// ArbitrageActionSkip means the task shall not be executed.
	ArbitrageActionSkip ArbitrageAction = iota

	// ArbitrageActionPostpone means the task shall execution shall be delayed at the provided time.
	ArbitrageActionPostpone

	// ArbitrageActionPause for a periodic task only, it means the task shall not be executed paused.
	ArbitrageActionPause
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
	Arbitrate(ctx context.Context, tasks []Task) ([]ArbitrageDecision, error)

	// Run is called to execute the task with the provided arguments. The runCount is incremented each time the task is
	// executed (i.e., it's always 1 for non-periodic tasks).
	Run(ctx context.Context, id TaskID, args *codectypes.Any, runCount uint64) error
}

// Invocation represents a scheduled task invocation.
type Invocation struct {
	// TaskID is the unique identifier of the task to be invoked.
	TaskID TaskID

	// Args are the arguments to be passed to the task handler when executing the task.
	Args proto.Message
}

// ArbitrageDecision defines the decision made by the task handler about a task during the arbitrage step.
type ArbitrageDecision struct {
	// Action defines the action to take for the task.
	Action ArbitrageAction

	// PostponeAt defines the time at which the task should be postponed if the action is ArbitrageActionPostpone.
	PostponeAt *time.Duration
}

func NewTaskHandler[T proto.Message](
	name string,
	dependsOn []string,
	arbitrateFn func(ctx context.Context, tasks []Invocation) ([]ArbitrageDecision, error),
	runFn func(ctx context.Context, id TaskID, args T, runCount uint64) error,
) TaskHandler {
	// if arbitrateFn is nil, we provide a no-op function.
	if arbitrateFn == nil {
		arbitrateFn = func(ctx context.Context, tasks []Invocation) ([]ArbitrageDecision, error) {
			return nil, nil
		}
	}

	// if runFn is nil, we provide a no-op function.
	if runFn == nil {
		runFn = func(ctx context.Context, id TaskID, args T, runCount uint64) error {
			return nil
		}
	}

	var zeroT T
	return taskHandler[T]{
		name:        name,
		dependsOn:   dependsOn,
		argsType:    zeroT,
		arbitrateFn: arbitrateFn,
		runFn:       runFn,
	}
}

func NewNoArgsTaskHandler(
	name string,
	dependsOn []string,
	arbitrateFn func(ctx context.Context, tasks []TaskID) ([]ArbitrageDecision, error),
	runFn func(ctx context.Context, id TaskID, runCount uint64) error,
) TaskHandler {
	var wrappedArbitrateFn func(ctx context.Context, tasks []Invocation) ([]ArbitrageDecision, error)
	var wrappedRunFn func(ctx context.Context, id TaskID, args proto.Message, runCount uint64) error

	if arbitrateFn != nil {
		wrappedArbitrateFn = func(ctx context.Context, tasks []Invocation) ([]ArbitrageDecision, error) {
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
	dependsOn   []string
	argsType    T
	arbitrateFn func(ctx context.Context, tasks []Invocation) ([]ArbitrageDecision, error)
	runFn       func(ctx context.Context, id TaskID, args T, runCount uint64) error
}

func (t taskHandler[T]) Typename() string {
	return t.name
}

func (t taskHandler[T]) DependsOn() []string {
	return t.dependsOn
}

func (t taskHandler[T]) PackArgs(args proto.Message) (*codectypes.Any, error) {
	if args == nil && t.argsType == nil {
		return nil, nil
	}

	if reflect.TypeOf(args) != reflect.TypeOf(t.argsType) {
		return nil, fmt.Errorf("task spec args type mismatch")
	}

	return codectypes.NewAnyWithValue(args)
}

func (t taskHandler[T]) UnpackArgs(packedArgs *codectypes.Any) (T, error) {
	var zeroArgs T
	if packedArgs == nil {
		if t.argsType == nil {
			return zeroArgs, nil
		}
		return zeroArgs, fmt.Errorf("task spec '%s' expects arguments of type '%T', but got nil", t.name, t.argsType)
	}

	if t.argsType == nil {
		return zeroArgs, fmt.Errorf("task spec '%s' does not expect any arguments, but got '%T'", t.name, packedArgs.GetCachedValue())
	}

	args, ok := packedArgs.GetCachedValue().(T)
	if !ok {
		return zeroArgs, fmt.Errorf("task spec '%s' expects arguments of type '%T', but got '%T'", t.name, t.argsType, packedArgs.GetCachedValue())
	}

	return args, nil
}

func (t taskHandler[T]) Arbitrate(ctx context.Context, tasks []Task) ([]ArbitrageDecision, error) {
	invocations := make([]Invocation, len(tasks))
	for i, task := range tasks {
		args, err := t.UnpackArgs(task.Args)
		if err != nil {
			return nil, err
		}
		invocations[i] = Invocation{
			TaskID: task.Id,
			Args:   args,
		}
	}

	return t.arbitrateFn(ctx, invocations)
}

func (t taskHandler[T]) Run(ctx context.Context, id TaskID, packedArgs *codectypes.Any, runCount uint64) error {
	args, err := t.UnpackArgs(packedArgs)
	if err != nil {
		return err
	}

	return t.runFn(ctx, id, args, runCount)
}
