package types

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestNewTask(t *testing.T) {
	args := &cosmostypes.Coin{Denom: "udenom", Amount: math.NewInt(12000)}
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
				NextRunAt:          &at,
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
				ScheduleEvery(interval),
				WithAbsoluteScheduling(),
			},
			expectError: false,
			expectTask: &Task{
				Id:                 "2",
				Typename:           "type2",
				Args:               nil,
				NextRunAt:          &at,
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
				NextRunAt:          &at,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
		},
		{
			name:         "wrong task with no defined schedule",
			id:           TaskID("1"),
			typename:     "type",
			args:         nil,
			scheduleOpts: []SchedulingOption{},
			expectError:  true,
			expectTask:   nil,
		},
		{
			name:     "wrong task with schedule defined in the past",
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

func TestNextRun(t *testing.T) {
	now := time.Now()
	at := now.Add(-5 * time.Second)
	nowMinus15Min := now.Add(-15 * time.Minute)
	interval := 10 * time.Minute
	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: now}, false, nil)

	testCases := []struct {
		name         string
		task         *Task
		expectedNext *time.Time
	}{
		{
			name: "non-periodic task should return nil",
			task: &Task{
				Id:                 "task1",
				Typename:           "type",
				Args:               nil,
				NextRunAt:          &at,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
			expectedNext: nil,
		},
		{
			name: "periodic task with relative scheduling is based on actual execution time",
			task: &Task{
				Id:                 "task2",
				Typename:           "type",
				Args:               nil,
				NextRunAt:          &at,
				Interval:           &interval,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_RELATIVE,
			},
			expectedNext: func() *time.Time { t := ctx.BlockTime().Add(10 * time.Minute); return &t }(),
		},
		{
			name: "periodic task with absolute scheduling is based on scheduled execution time",
			task: &Task{
				Id:                 "task2",
				Typename:           "type",
				Args:               nil,
				NextRunAt:          &at,
				Interval:           &interval,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_ABSOLUTE,
			},
			expectedNext: func() *time.Time { t := at.Add(10 * time.Minute); return &t }(),
		},
		{
			name: "periodic task with absolute scheduling should not schedule in the past",
			task: &Task{
				Id:                 "task2",
				Typename:           "type",
				Args:               nil,
				NextRunAt:          &nowMinus15Min,
				Interval:           &interval,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: SchedulingStrategy_ABSOLUTE,
			},
			expectedNext: func() *time.Time { t := now.Add(5 * time.Minute); return &t }(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nextRun := tc.task.NextRun(ctx)
			if tc.expectedNext == nil {
				require.Nil(t, nextRun)
			} else {
				require.NotNil(t, nextRun)
				require.Equal(t, *tc.expectedNext, *nextRun)
			}
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
		NextRunAt:          &now,
		Interval:           nil,
		LastRunAt:          nil,
		RunCount:           0,
		SchedulingStrategy: 0,
	}

	ScheduleAt(runAt)(ctx, task)

	require.Equal(t, runAt, *task.NextRunAt)
}

func TestScheduleIn(t *testing.T) {
	now := time.Now()
	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: now}, false, nil)
	in := 10 * time.Minute
	task := &Task{
		Id:                 "task",
		Typename:           "type",
		Args:               nil,
		NextRunAt:          &now,
		Interval:           nil,
		LastRunAt:          nil,
		RunCount:           0,
		SchedulingStrategy: 0,
	}

	ScheduleIn(in)(ctx, task)
	fmt.Println(now.Add(in))
	fmt.Println(task.NextRunAt)
	require.Equal(t, ctx.BlockTime().Add(in), *task.NextRunAt)
}

func TestScheduleEvery(t *testing.T) {
	now := time.Now()
	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: now}, false, nil)
	interval := 10 * time.Minute
	task := &Task{
		Id:                 "task",
		Typename:           "type",
		Args:               nil,
		NextRunAt:          &now,
		Interval:           nil,
		LastRunAt:          nil,
		RunCount:           0,
		SchedulingStrategy: 0,
	}

	ScheduleEvery(interval)(ctx, task)

	require.Equal(t, interval, *task.Interval)
}

func TestWithRelativeScheduling(t *testing.T) {
	now := time.Now()
	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: now}, false, nil)
	task := &Task{
		Id:                 "task",
		Typename:           "type",
		Args:               nil,
		NextRunAt:          &now,
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
		NextRunAt:          &now,
		Interval:           nil,
		LastRunAt:          nil,
		RunCount:           0,
		SchedulingStrategy: SchedulingStrategy_RELATIVE,
	}

	WithAbsoluteScheduling()(ctx, task)

	require.Equal(t, SchedulingStrategy_ABSOLUTE, task.SchedulingStrategy)
}
