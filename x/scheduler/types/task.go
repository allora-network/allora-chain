package types

import (
	"context"
	"time"

	"cosmossdk.io/errors"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewTask creates a new Task with the provided parameters and scheduling options.
func NewTask(ctx context.Context, id TaskID, typename string, args *codectypes.Any, scheduleOpts ...SchedulingOption) (*Task, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	task := &Task{
		Id:                 id,
		Typename:           typename,
		Args:               args,
		NextRunAt:          nil,
		Interval:           nil,
		LastRunAt:          nil,
		RunCount:           0,
		SchedulingStrategy: SchedulingStrategy_RELATIVE,
	}

	for _, opt := range scheduleOpts {
		opt(sdkCtx, task)
	}

	if task.NextRunAt == nil {
		return nil, errors.Wrapf(ErrInvalidTask, "task '%s' must have a scheduled time", id)
	}

	if sdkCtx.BlockTime().After(*task.NextRunAt) {
		return nil, errors.Wrapf(ErrInvalidTask, "cannot schedule task '%s' for a time in the past: '%s'", id, *task.NextRunAt)
	}

	return task, nil
}

// NextRun computes the next run time for the task based on its scheduling strategy and interval.
// If the task is not periodic (i.e., Interval is nil), it returns nil.
func (t *Task) NextRun(ctx sdk.Context) *time.Time {
	if t.Interval == nil {
		return nil
	}

	var refTime time.Time
	if t.SchedulingStrategy == SchedulingStrategy_ABSOLUTE {
		elapsed := ctx.BlockTime().Sub(*t.NextRunAt)
		missed := elapsed / *t.Interval
		refTime = t.NextRunAt.Add(missed * (*t.Interval))
	} else {
		refTime = ctx.BlockTime()
	}

	nextRun := refTime.Add(*t.Interval)
	return &nextRun
}

// SchedulingOption defines a function that modifies a Task to set its scheduling parameters.
type SchedulingOption func(sdk.Context, *Task)

// ScheduleAt sets the exact time when the task should be executed.
func ScheduleAt(at time.Time) SchedulingOption {
	return func(_ sdk.Context, t *Task) {
		t.NextRunAt = &at
	}
}

// ScheduleIn sets the task to be executed after the specified duration from the current block time.
func ScheduleIn(in time.Duration) SchedulingOption {
	return func(ctx sdk.Context, t *Task) {
		at := ctx.BlockTime().Add(in)
		t.NextRunAt = &at
	}
}

// ScheduleEvery sets the task to be periodic, with the specified interval between executions.
func ScheduleEvery(interval time.Duration) SchedulingOption {
	return func(_ sdk.Context, t *Task) {
		t.Interval = &interval
	}
}

// WithSchedulingStrategy sets the scheduling strategy for the task (relative or absolute).
func WithSchedulingStrategy(strategy SchedulingStrategy) SchedulingOption {
	return func(_ sdk.Context, t *Task) {
		t.SchedulingStrategy = strategy
	}
}

// WithRelativeScheduling sets the scheduling strategy to relative (default).
func WithRelativeScheduling() SchedulingOption {
	return WithSchedulingStrategy(SchedulingStrategy_RELATIVE)
}

// WithAbsoluteScheduling sets the scheduling strategy to absolute.
func WithAbsoluteScheduling() SchedulingOption {
	return WithSchedulingStrategy(SchedulingStrategy_ABSOLUTE)
}
