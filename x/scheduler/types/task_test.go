//nolint:exhaustruct
package types

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestNewTask(t *testing.T) {
	args := &sdk.Coin{Denom: "udenom", Amount: math.NewInt(12000)}
	packedArgs, err := codectypes.NewAnyWithValue(args)
	require.NoError(t, err)

	now := time.Now()
	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: now}, false, nil)
	interval := 10 * time.Minute
	at := ctx.BlockTime().Add(interval)

	testCases := []struct {
		name         string
		id           TaskID
		typename     string
		args         *codectypes.Any
		scheduleOpts []SchedulingOption
		expectError  bool
		expectTask   *Task
	}{
		{
			name:     "valid task with simple scheduling",
			id:       TaskID("1"),
			typename: "type",
			args:     nil,
			scheduleOpts: []SchedulingOption{
				ScheduleAt(at),
			},
			expectError: false,
			expectTask: &Task{
				Id:                 "1",
				Typename:           "type",
				Args:               nil,
				ScheduledFor:       &at,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
		},
		{
			name:     "valid task with periodic absolute scheduling",
			id:       TaskID("2"),
			typename: "type2",
			args:     nil,
			scheduleOpts: []SchedulingOption{
				ScheduleIn(interval),
				ScheduleEvery(&interval),
				WithAbsoluteScheduling(),
			},
			expectError: false,
			expectTask: &Task{
				Id:                 "2",
				Typename:           "type2",
				Args:               nil,
				ScheduledFor:       &at,
				Interval:           &interval,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_ABSOLUTE,
			},
		},
		{
			name:     "valid task with args",
			id:       TaskID("1"),
			typename: "type",
			args:     packedArgs,
			scheduleOpts: []SchedulingOption{
				ScheduleAt(at),
			},
			expectError: false,
			expectTask: &Task{
				Id:                 "1",
				Typename:           "type",
				Args:               packedArgs,
				ScheduledFor:       &at,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
		},
		{
			name:     "wrong task with invalid scheduling",
			id:       TaskID("1"),
			typename: "type",
			args:     nil,
			scheduleOpts: []SchedulingOption{
				ScheduleAt(now.Add(-interval)),
			},
			expectError: true,
			expectTask:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			task, err := NewTask(ctx, tc.id, tc.typename, tc.args, tc.scheduleOpts...)
			if tc.expectError {
				require.Error(t, err)
				require.Nil(t, task)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectTask, task)
			}
		})
	}
}

func TestApplySchedulingOpts(t *testing.T) {
	args := &sdk.Coin{Denom: "udenom", Amount: math.NewInt(12000)}
	packedArgs, err := codectypes.NewAnyWithValue(args)
	require.NoError(t, err)

	now := time.Now()
	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: now}, false, nil)
	interval := 10 * time.Minute
	negInterval := -interval
	zeroInterval := time.Duration(0)
	at := ctx.BlockTime().Add(interval)

	testCases := []struct {
		name         string
		inputTask    *Task
		scheduleOpts []SchedulingOption
		expectError  bool
		expectTask   *Task
	}{
		{
			name: "simple scheduling",
			inputTask: &Task{
				Id:                 "1",
				Typename:           "type",
				Args:               nil,
				ScheduledFor:       nil,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
			scheduleOpts: []SchedulingOption{
				ScheduleAt(at),
			},
			expectError: false,
			expectTask: &Task{
				Id:                 "1",
				Typename:           "type",
				Args:               nil,
				ScheduledFor:       &at,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
		},
		{
			name: "periodic absolute scheduling",
			inputTask: &Task{
				Id:                 "2",
				Typename:           "type2",
				Args:               nil,
				ScheduledFor:       nil,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_ABSOLUTE,
			},
			scheduleOpts: []SchedulingOption{
				ScheduleIn(interval),
				ScheduleEvery(&interval),
				WithAbsoluteScheduling(),
			},
			expectError: false,
			expectTask: &Task{
				Id:                 "2",
				Typename:           "type2",
				Args:               nil,
				ScheduledFor:       &at,
				Interval:           &interval,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_ABSOLUTE,
			},
		},
		{
			name: "schedule defined in the past",
			inputTask: &Task{
				Id:                 "1",
				Typename:           "type",
				Args:               packedArgs,
				ScheduledFor:       &at,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
			scheduleOpts: []SchedulingOption{
				ScheduleAt(now.Add(-interval)),
			},
			expectError: true,
			expectTask:  nil,
		},
		{
			name: "negative interval",
			inputTask: &Task{
				Id:                 "1",
				Typename:           "type",
				Args:               packedArgs,
				ScheduledFor:       &at,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
			scheduleOpts: []SchedulingOption{
				ScheduleEvery(&negInterval),
			},
			expectError: true,
			expectTask:  nil,
		},
		{
			name: "zero interval",
			inputTask: &Task{
				Id:                 "1",
				Typename:           "type",
				Args:               packedArgs,
				ScheduledFor:       &at,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
			scheduleOpts: []SchedulingOption{
				ScheduleEvery(&zeroInterval),
			},
			expectError: true,
			expectTask:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			task := *tc.inputTask
			err := task.ApplySchedulingOpts(ctx, tc.scheduleOpts...)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectTask, &task)
			}
		})
	}
}

func TestComputeNextRun(t *testing.T) {
	now := time.Now()
	at := now.Add(-5 * time.Second)
	nowMinus15Min := now.Add(-15 * time.Minute)
	interval := 10 * time.Minute

	testCases := []struct {
		name           string
		task           *Task
		expectedTask   *Task
		expectedResult bool
	}{
		{
			name: "non-periodic task should return false",
			task: &Task{
				Id:                 "task1",
				Typename:           "type",
				Args:               nil,
				ScheduledFor:       &at,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
			expectedTask: &Task{
				Id:                 "task1",
				Typename:           "type",
				Args:               nil,
				ScheduledFor:       &at,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
			expectedResult: false,
		},
		{
			name: "periodic task with relative scheduling is based on actual execution time",
			task: &Task{
				Id:                 "task2",
				Typename:           "type",
				Args:               nil,
				ScheduledFor:       &at,
				Interval:           &interval,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
			expectedTask: &Task{
				Id:                 "task2",
				Typename:           "type",
				Args:               nil,
				ScheduledFor:       func() *time.Time { t := now.Add(10 * time.Minute); return &t }(),
				Interval:           &interval,
				LastRunAt:          &now,
				RunCount:           1,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
			expectedResult: true,
		},
		{
			name: "periodic task with absolute scheduling is based on scheduled execution time",
			task: &Task{
				Id:                 "task2",
				Typename:           "type",
				Args:               nil,
				ScheduledFor:       &at,
				Interval:           &interval,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_ABSOLUTE,
			},
			expectedTask: &Task{
				Id:                 "task2",
				Typename:           "type",
				Args:               nil,
				ScheduledFor:       func() *time.Time { t := at.Add(10 * time.Minute); return &t }(),
				Interval:           &interval,
				LastRunAt:          &now,
				RunCount:           1,
				SchedulingStrategy: SchedulingStrategy_ABSOLUTE,
			},
			expectedResult: true,
		},
		{
			name: "periodic task with absolute scheduling should not schedule in the past",
			task: &Task{
				Id:                 "task2",
				Typename:           "type",
				Args:               nil,
				ScheduledFor:       &nowMinus15Min,
				Interval:           &interval,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_ABSOLUTE,
			},
			expectedTask: &Task{
				Id:                 "task2",
				Typename:           "type",
				Args:               nil,
				ScheduledFor:       func() *time.Time { t := now.Add(5 * time.Minute); return &t }(),
				Interval:           &interval,
				LastRunAt:          &now,
				RunCount:           1,
				SchedulingStrategy: SchedulingStrategy_ABSOLUTE,
			},
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expectedResult, tc.task.ComputeNextRun(now))
			require.Equal(t, tc.expectedTask, tc.task)
		})
	}
}

func TestScheduleAt(t *testing.T) {
	now := time.Now()
	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: now}, false, nil)
	runAt := now.Add(10 * time.Minute)
	task := &Task{
		Id:                 "task",
		Typename:           "type",
		Args:               nil,
		ScheduledFor:       &now,
		Interval:           nil,
		LastRunAt:          nil,
		RunCount:           0,
		SchedulingStrategy: 0,
	}

	ScheduleAt(runAt)(ctx, task)

	require.Equal(t, runAt, *task.ScheduledFor)
}

func TestScheduleIn(t *testing.T) {
	now := time.Now()
	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: now}, false, nil)
	in := 10 * time.Minute
	task := &Task{
		Id:                 "task",
		Typename:           "type",
		Args:               nil,
		ScheduledFor:       &now,
		Interval:           nil,
		LastRunAt:          nil,
		RunCount:           0,
		SchedulingStrategy: 0,
	}

	ScheduleIn(in)(ctx, task)
	fmt.Println(now.Add(in))
	fmt.Println(task.ScheduledFor)
	require.Equal(t, ctx.BlockTime().Add(in), *task.ScheduledFor)
}

func TestScheduleEvery(t *testing.T) {
	now := time.Now()
	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: now}, false, nil)
	interval := 10 * time.Minute
	task := &Task{
		Id:                 "task",
		Typename:           "type",
		Args:               nil,
		ScheduledFor:       &now,
		Interval:           nil,
		LastRunAt:          nil,
		RunCount:           0,
		SchedulingStrategy: 0,
	}

	ScheduleEvery(&interval)(ctx, task)
	require.Equal(t, interval, *task.Interval)

	ScheduleEvery(nil)(ctx, task)
	require.Nil(t, task.Interval)
}

func TestUnschedule(t *testing.T) {
	now := time.Now()
	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: now}, false, nil)
	task := &Task{
		Id:                 "task",
		Typename:           "type",
		Args:               nil,
		ScheduledFor:       &now,
		Interval:           nil,
		LastRunAt:          nil,
		RunCount:           0,
		SchedulingStrategy: 0,
	}

	Unschedule()(ctx, task)

	require.Nil(t, task.ScheduledFor)
}

func TestWithRelativeScheduling(t *testing.T) {
	now := time.Now()
	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: now}, false, nil)
	task := &Task{
		Id:                 "task",
		Typename:           "type",
		Args:               nil,
		ScheduledFor:       &now,
		Interval:           nil,
		LastRunAt:          nil,
		RunCount:           0,
		SchedulingStrategy: SchedulingStrategy_ABSOLUTE,
	}

	WithRelativeScheduling()(ctx, task)

	require.Equal(t, SchedulingStrategy_RELATIVE, task.SchedulingStrategy)
}

func TestWithAbsoluteScheduling(t *testing.T) {
	now := time.Now()
	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: now}, false, nil)
	task := &Task{
		Id:                 "task",
		Typename:           "type",
		Args:               nil,
		ScheduledFor:       &now,
		Interval:           nil,
		LastRunAt:          nil,
		RunCount:           0,
		SchedulingStrategy: SchedulingStrategy_RELATIVE,
	}

	WithAbsoluteScheduling()(ctx, task)

	require.Equal(t, SchedulingStrategy_ABSOLUTE, task.SchedulingStrategy)
}
