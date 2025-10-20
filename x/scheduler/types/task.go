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

	if err := task.ApplySchedulingOpts(ctx, scheduleOpts...); err != nil {
		return nil, err
	}

	return task, nil
}

// ApplySchedulingOpts applies the provided scheduling options to the task, updating its scheduling parameters.
// It validates the resulting scheduling configuration to ensure it is valid.
func (t *Task) ApplySchedulingOpts(ctx context.Context, scheduleOpts ...SchedulingOption) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for _, opt := range scheduleOpts {
		opt(sdkCtx, t)
	}

	if t.NextRunAt != nil && sdkCtx.BlockTime().After(*t.NextRunAt) {
		return errors.Wrapf(ErrInvalidTask, "cannot schedule task '%s' for a time in the past: '%s'", t.Id, *t.NextRunAt)
	}

	if t.Interval != nil && *t.Interval <= 0 {
		return errors.Wrapf(ErrInvalidTask, "cannot schedule task '%s' with a non-positive interval: %s", t.Id, *t.Interval)
	}

	return nil
}

// ComputeNextRun computes and configures the next run time for the task based on its scheduling strategy, interval and
// the provided last run time. It updates the LastRunAt, NextRunAt and RunCount fields of the task.
//
// It returns true if the next run time was computed and set, false otherwise (e.g., if the task is not periodic).
func (t *Task) ComputeNextRun(lastRun time.Time) bool {
	if t.Interval == nil {
		return false
	}

	var refTime time.Time
	if t.SchedulingStrategy == SchedulingStrategy_ABSOLUTE {
		elapsed := lastRun.Sub(*t.NextRunAt)
		missed := elapsed / *t.Interval
		refTime = t.NextRunAt.Add(missed * (*t.Interval)) //nolint:durationcheck
	} else {
		refTime = lastRun
	}
	nextRun := refTime.Add(*t.Interval)

	t.RunCount += 1
	t.LastRunAt = &lastRun
	t.NextRunAt = &nextRun

	return true
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

// ScheduleEvery sets the task's interval between executions, if nil the task will execute once, otherwise the task is periodic.
func ScheduleEvery(interval *time.Duration) SchedulingOption {
	return func(_ sdk.Context, t *Task) {
		t.Interval = interval
	}
}

// Unschedule sets the task’s next execution to unscheduled. The task remains stored for future use but is considered paused.
func Unschedule() SchedulingOption {
	return func(_ sdk.Context, t *Task) {
		t.NextRunAt = nil
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
