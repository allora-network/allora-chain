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
