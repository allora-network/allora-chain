package types

import (
	"context"
	"fmt"
	"reflect"
	"time"

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

type TaskTypes []TaskType

// TaskType defines the specification for a task type that can be managed by the scheduler.
// These should be registered to allow the module to schedule and execute tasks of that type.
type TaskType struct {
	// Name is the unique identifier for the task type.
	Name string

	// DependsOn defines task type dependencies that must be executed before.
	DependsOn []string

	// ArgsType defines the expected arguments for this task type.
	ArgsType proto.Message

	// TaskHandler carries the logic associated with this task type.
	TaskHandler TaskHandler
}

// TaskHandler defines the required interface for handling tasks in the scheduler.
// For each task type, execution is split into two phases:
//   1. Arbitrage phase: where the task handler can decide whether to run the task, postpone it, skip it, etc...
//   2. Run phase: where the task handler executes the task with the provided arguments.
type TaskHandler interface {
	// Arbitrate is called once prior to the execution of tasks that are to be invoked at this time. Its purpose is to
	// determine the action to take for each task invocation: If no action is taken for a task, it'll be executed.
	Arbitrate(ctx context.Context, tasks []Invocation) ([]ArbitrageDecision, error)

	// Run is called to execute the task with the provided arguments. The runCount is incremented each time the task is
	// executed (i.e., it's always 1 for non-periodic tasks).
	Run(ctx context.Context, id TaskID, args proto.Message, runCount uint64) error
}

type Invocation struct {
	TaskID TaskID
	Args   proto.Message
}

type ArbitrageDecision struct {
	TaskID     TaskID
	Action     ArbitrageAction
	PostponeAt *time.Duration
}

func (s TaskType) ValidateArgs(args proto.Message) error {
	if s.ArgsType == nil {
		if args != nil {
			return fmt.Errorf("task spec '%s' has no args type defined, but args are provided", s.Name)
		}
		return nil
	}

	if reflect.TypeOf(args) != reflect.TypeOf(s.ArgsType) {
		return fmt.Errorf("task spec args type mismatch")
	}

	return nil
}

type fnTaskHandler struct {
	ArbitrateFunc func(ctx context.Context, tasks []Invocation) ([]ArbitrageDecision, error)
	RunFunc       func(ctx context.Context, id TaskID, args proto.Message, runCount uint64) error
}

func NewTaskHandlerFromFuncs(
	arbitrateFn func(ctx context.Context, tasks []Invocation) ([]ArbitrageDecision, error),
	runFn func(ctx context.Context, id TaskID, args proto.Message, runCount uint64) error,
) TaskHandler {
	return &fnTaskHandler{
		ArbitrateFunc: arbitrateFn,
		RunFunc:       runFn,
	}
}

func (f *fnTaskHandler) Arbitrate(ctx context.Context, tasks []Invocation) ([]ArbitrageDecision, error) {
	if f.ArbitrateFunc == nil {
		return nil, nil
	}
	return f.ArbitrateFunc(ctx, tasks)
}

func (f *fnTaskHandler) Run(ctx context.Context, id TaskID, args proto.Message, runCount uint64) error {
	if f.RunFunc == nil {
		return nil
	}
	return f.RunFunc(ctx, id, args, runCount)
}
