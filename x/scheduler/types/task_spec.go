package types

import (
	"context"
	"fmt"
	"reflect"

	"github.com/cosmos/gogoproto/proto"
)

type TaskID string

type ArbitrageAction int

const (
	// ArbitrageActionReject means the task shall not be executed.
	ArbitrageActionReject ArbitrageAction = iota
	// ArbitrageActionPause for a recurring task only, it means the task shall not be executed, and the recurring task
	// will be set on pause.
	ArbitrageActionPause
)

type TaskSpecs []TaskSpec

type TaskSpec struct {
	// Name is the unique identifier for the task type.
	Name string

	// Priority Defines the execution order, the lowest priorities are executed first.
	Priority uint32

	// ArgsType defines the expected arguments for this task type.
	ArgsType proto.Message

	// TaskHandler carries the logic associated with this task type.
	TaskHandler TaskHandler
}

type TaskHandler interface {
	Arbitrate(ctx context.Context, tasks []Invocation) ([]ArbitrageDecision, error)
	Run(ctx context.Context, id TaskID, args proto.Message, runCount uint64) error
}

type Invocation struct {
	TaskID TaskID
	Args   []proto.Message
}

type ArbitrageDecision struct {
	TaskID TaskID
	Action ArbitrageAction
}

func (s TaskSpec) ValidateArgs(args proto.Message) error {
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
