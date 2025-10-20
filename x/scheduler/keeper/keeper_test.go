//nolint:exhaustruct
package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/header"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

func TestRegisterTaskHandlers(t *testing.T) {
	testCases := []struct {
		name        string
		handlers    types.TaskHandlers
		expectError bool
		expectOrder []string
	}{
		{
			name:        "no handlers",
			handlers:    nil,
			expectError: false,
			expectOrder: nil,
		},
		{
			name: "a single handler",
			handlers: types.TaskHandlers{
				types.NewNoArgsTaskHandler("handler1", nil, nil, nil),
			},
			expectError: false,
			expectOrder: []string{"handler1"},
		},
		{
			name: "handler with empty name",
			handlers: types.TaskHandlers{
				types.NewNoArgsTaskHandler("", nil, nil, nil),
			},
			expectError: true,
			expectOrder: nil,
		},
		{
			name: "duplicate name",
			handlers: types.TaskHandlers{
				types.NewNoArgsTaskHandler("handler1", nil, nil, nil),
				types.NewNoArgsTaskHandler("handler1", nil, nil, nil),
			},
			expectError: true,
			expectOrder: nil,
		},
		{
			name: "handler without deps should be ordered alphanumerically",
			handlers: types.TaskHandlers{
				types.NewNoArgsTaskHandler("handlerB", nil, nil, nil),
				types.NewNoArgsTaskHandler("handlerA", nil, nil, nil),
				types.NewNoArgsTaskHandler("handlerC2", nil, nil, nil),
				types.NewNoArgsTaskHandler("handlerC1", nil, nil, nil),
			},
			expectError: false,
			expectOrder: []string{"handlerA", "handlerB", "handlerC1", "handlerC2"},
		},
		{
			name: "handlers with deps should be ordered respecting dependencies",
			handlers: types.TaskHandlers{
				types.NewNoArgsTaskHandler("handlerA", nil, nil, nil),
				types.NewNoArgsTaskHandler("handlerD", []string{"handlerA"}, nil, nil),
				types.NewNoArgsTaskHandler("handlerC", []string{"handlerB", "handlerD"}, nil, nil),
				types.NewNoArgsTaskHandler("handlerB", []string{"handlerA"}, nil, nil),
			},
			expectError: false,
			expectOrder: []string{"handlerA", "handlerB", "handlerD", "handlerC"},
		},
		{
			name: "handler with unexisting dependency",
			handlers: types.TaskHandlers{
				types.NewNoArgsTaskHandler("handlerA", []string{"handlerX"}, nil, nil),
			},
			expectError: true,
			expectOrder: nil,
		},
		{
			name: "handlers with circular dependency",
			handlers: types.TaskHandlers{
				types.NewNoArgsTaskHandler("handlerA", []string{"handlerC"}, nil, nil),
				types.NewNoArgsTaskHandler("handlerB", []string{"handlerA"}, nil, nil),
				types.NewNoArgsTaskHandler("handlerC", []string{"handlerB"}, nil, nil),
			},
			expectError: true,
			expectOrder: nil,
		},
	}

	storeService := runtime.NewKVStoreService(storetypes.NewKVStoreKey("emissions"))
	encCfg := moduletestutil.MakeTestEncodingConfig()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k := NewKeeper(storeService, encCfg.Codec)
			err := k.RegisterTaskHandlers(tc.handlers)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectOrder, k.handlersOrder)
				require.Equal(t, len(tc.expectOrder), len(k.handlersByTypename))
				for _, name := range tc.expectOrder {
					_, exists := k.handlersByTypename[name]
					require.True(t, exists)
				}
			}
		})
	}
}

func TestScheduleTask(t *testing.T) {
	now := time.Now()
	d10min := 10 * time.Minute
	d1Hour := 1 * time.Hour
	at := now.Add(10 * time.Minute).UTC()

	args := &cosmostypes.Coin{Denom: "udenom", Amount: math.NewInt(12000)}
	packedArgs, err := codectypes.NewAnyWithValue(args)
	args2 := &cosmostypes.DecCoin{Denom: "udenom", Amount: math.LegacyNewDec(12000)}
	require.NoError(t, err)

	testCases := []struct {
		name         string
		id           types.TaskID
		typename     string
		args         proto.Message
		scheduleOpts []types.SchedulingOption
		expectError  bool
		expectTask   *types.Task
	}{
		{
			name:     "ok without args",
			id:       "task1",
			typename: "noargs",
			args:     nil,
			scheduleOpts: []types.SchedulingOption{
				types.ScheduleAt(at),
			},
			expectError: false,
			expectTask: &types.Task{
				Id:                 "task1",
				Typename:           "noargs",
				Args:               nil,
				ScheduledFor:       &at,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: types.SchedulingStrategy_RELATIVE,
			},
		},
		{
			name:     "ok with args",
			id:       "task2",
			typename: "withargs",
			args:     args,
			scheduleOpts: []types.SchedulingOption{
				types.ScheduleAt(at),
			},
			expectError: false,
			expectTask: &types.Task{
				Id:                 "task2",
				Typename:           "withargs",
				Args:               packedArgs,
				ScheduledFor:       &at,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: types.SchedulingStrategy_RELATIVE,
			},
		},
		{
			name:     "ok with complex scheduling",
			id:       "task",
			typename: "withargs",
			args:     args,
			scheduleOpts: []types.SchedulingOption{
				types.ScheduleIn(d10min),
				types.ScheduleEvery(&d1Hour),
				types.WithAbsoluteScheduling(),
			},
			expectError: false,
			expectTask: &types.Task{
				Id:                 "task",
				Typename:           "withargs",
				Args:               packedArgs,
				ScheduledFor:       &at,
				Interval:           &d1Hour,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: types.SchedulingStrategy_ABSOLUTE,
			},
		},
		{
			name:     "unexisting handler",
			id:       "task",
			typename: "unexisting",
			args:     nil,
			scheduleOpts: []types.SchedulingOption{
				types.ScheduleAt(at),
			},
			expectError: true,
			expectTask:  nil,
		},
		{
			name:     "wrong args type (nil)",
			id:       "task",
			typename: "withargs",
			args:     nil,
			scheduleOpts: []types.SchedulingOption{
				types.ScheduleAt(at),
			},
			expectError: true,
			expectTask:  nil,
		},
		{
			name:     "wrong args type (non-nil)",
			id:       "task",
			typename: "withargs",
			args:     args2,
			scheduleOpts: []types.SchedulingOption{
				types.ScheduleAt(at),
			},
			expectError: true,
			expectTask:  nil,
		},
		{
			name:     "duplicate task id",
			id:       "existing",
			typename: "noargs",
			args:     nil,
			scheduleOpts: []types.SchedulingOption{
				types.ScheduleAt(at),
			},
			expectError: true,
			expectTask:  nil,
		},
		{
			name:     "invalid scheduling",
			id:       "task",
			typename: "noargs",
			args:     nil,
			scheduleOpts: []types.SchedulingOption{
				types.ScheduleAt(now.Add(-time.Hour)),
			},
			expectError: true,
			expectTask:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storeKey := storetypes.NewKVStoreKey("scheduler")
			storeService := runtime.NewKVStoreService(storeKey)
			encCfg := moduletestutil.MakeTestEncodingConfig()
			k := NewKeeper(storeService, encCfg.Codec)
			err := k.RegisterTaskHandlers(types.TaskHandlers{
				types.NewNoArgsTaskHandler("noargs", nil, nil, nil),
				types.NewTaskHandler[*cosmostypes.Coin]("withargs", nil, nil, nil),
			})
			require.NoError(t, err)

			ctx := testutil.DefaultContextWithKeys(
				map[string]*storetypes.KVStoreKey{"scheduler": storeKey},
				map[string]*storetypes.TransientStoreKey{"transient_test": storetypes.NewTransientStoreKey("transient_test")},
				nil,
			).WithHeaderInfo(header.Info{Height: 1, Time: now}).
				WithBlockHeight(1).
				WithBlockTime(now)

			// pre-insert a task to test duplicate ids
			err = k.ScheduleTask(ctx, "noargs", "existing", nil, types.ScheduleAt(at))
			require.NoError(t, err)

			err = k.ScheduleTask(ctx, tc.typename, tc.id, tc.args, tc.scheduleOpts...)
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			task, err := k.tasks.Get(ctx, tc.id)
			require.NoError(t, err)
			require.Equal(t, tc.expectTask.Id, task.Id)
			require.Equal(t, tc.expectTask.Typename, task.Typename)
			require.Equal(t, tc.expectTask.ScheduledFor, task.ScheduledFor)
			require.Equal(t, tc.expectTask.Interval, task.Interval)
			require.Equal(t, tc.expectTask.LastRunAt, task.LastRunAt)
			require.Equal(t, tc.expectTask.RunCount, task.RunCount)
			require.Equal(t, tc.expectTask.SchedulingStrategy, task.SchedulingStrategy)
			if tc.expectTask.Args == nil {
				require.Nil(t, task.Args)
			} else {
				require.NotNil(t, task.Args)
				require.Equal(t, packedArgs.TypeUrl, task.Args.TypeUrl)
				require.Equal(t, packedArgs.Value, task.Args.Value)
			}

			exists, err := k.tasksSchedule.Has(ctx, collections.Join3(tc.expectTask.Typename, *tc.expectTask.ScheduledFor, tc.expectTask.Id))
			require.NoError(t, err)
			require.True(t, exists)
		})
	}
}

func TestCancelTask(t *testing.T) {
	testCases := []struct {
		name                    string
		id                      types.TaskID
		expectError             bool
		expectTaskCount         int
		expectTaskScheduleCount int
	}{
		{
			name:                    "scheduled task",
			id:                      "task1",
			expectError:             false,
			expectTaskCount:         2,
			expectTaskScheduleCount: 1,
		},
		{
			name:                    "unscheduled task",
			id:                      "task2",
			expectError:             false,
			expectTaskCount:         2,
			expectTaskScheduleCount: 2,
		},
		{
			name:                    "unexisting task",
			id:                      "taskX",
			expectError:             true,
			expectTaskCount:         3,
			expectTaskScheduleCount: 2,
		},
	}

	now := time.Now()
	at := now.Add(10 * time.Minute)
	d1Hour := 1 * time.Hour

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storeKey := storetypes.NewKVStoreKey("scheduler")
			storeService := runtime.NewKVStoreService(storeKey)
			encCfg := moduletestutil.MakeTestEncodingConfig()
			k := NewKeeper(storeService, encCfg.Codec)
			err := k.RegisterTaskHandlers(types.TaskHandlers{
				types.NewNoArgsTaskHandler("noargs", nil, nil, nil),
			})
			require.NoError(t, err)

			ctx := testutil.DefaultContextWithKeys(
				map[string]*storetypes.KVStoreKey{"scheduler": storeKey},
				map[string]*storetypes.TransientStoreKey{"transient_test": storetypes.NewTransientStoreKey("transient_test")},
				nil,
			).WithHeaderInfo(header.Info{Height: 1, Time: now}).
				WithBlockHeight(1).
				WithBlockTime(now)

			// pre-insert tasks
			require.NoError(t, k.ScheduleTask(ctx, "noargs", "task1", nil, types.ScheduleAt(at)))
			require.NoError(t, k.ScheduleTask(ctx, "noargs", "task2", nil, types.ScheduleAt(at), types.ScheduleEvery(&d1Hour)))
			require.NoError(t, k.ScheduleTask(ctx, "noargs", "task3", nil, types.ScheduleAt(at)))
			// pause the task2 to test canceling a paused task
			require.NoError(t, k.tasksSchedule.Remove(ctx, collections.Join3[string, time.Time, types.TaskID]("noargs", at, "task2")))

			err = k.CancelTask(ctx, tc.id)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				exists, err := k.tasks.Has(ctx, tc.id)
				require.NoError(t, err)
				require.False(t, exists)
			}

			it, err := k.tasks.IterateRaw(ctx, nil, nil, collections.OrderAscending)
			require.NoError(t, err)
			keys, err := it.Keys()
			require.NoError(t, err)
			require.Len(t, keys, tc.expectTaskCount)

			it2, err := k.tasksSchedule.IterateRaw(ctx, nil, nil, collections.OrderAscending)
			require.NoError(t, err)
			keys2, err := it2.Keys()
			require.NoError(t, err)
			require.Len(t, keys2, tc.expectTaskScheduleCount)
		})
	}
}

func TestRescheduleTask(t *testing.T) {
	now := time.Now()
	d1Hour := 1 * time.Hour
	at := now.Add(d1Hour).UTC()

	testCases := []struct {
		name        string
		id          types.TaskID
		newSchedule []types.SchedulingOption
		expectError bool
		expectTask  *types.Task
	}{
		{
			name:        "existing task",
			id:          "task1",
			newSchedule: []types.SchedulingOption{types.ScheduleAt(at)},
			expectError: false,
			expectTask: &types.Task{
				Id:                 "task1",
				Typename:           "noargs",
				Args:               nil,
				ScheduledFor:       &at,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: types.SchedulingStrategy_RELATIVE,
			},
		},
		{
			name:        "resume task",
			id:          "task2",
			newSchedule: []types.SchedulingOption{types.ScheduleAt(at)},
			expectError: false,
			expectTask: &types.Task{
				Id:                 "task2",
				Typename:           "noargs",
				Args:               nil,
				ScheduledFor:       &at,
				Interval:           &d1Hour,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: types.SchedulingStrategy_RELATIVE,
			},
		},
		{
			name:        "pause task",
			id:          "task1",
			newSchedule: []types.SchedulingOption{types.Unschedule()},
			expectError: false,
			expectTask: &types.Task{
				Id:                 "task1",
				Typename:           "noargs",
				Args:               nil,
				ScheduledFor:       nil,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: types.SchedulingStrategy_RELATIVE,
			},
		},
		{
			name:        "unexisting task",
			id:          "taskX",
			newSchedule: []types.SchedulingOption{types.ScheduleAt(at)},
			expectError: true,
			expectTask:  nil,
		},
		{
			name:        "new time in the past",
			id:          "task1",
			newSchedule: []types.SchedulingOption{types.ScheduleAt(now.Add(-d1Hour))},
			expectError: true,
			expectTask:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storeKey := storetypes.NewKVStoreKey("scheduler")
			storeService := runtime.NewKVStoreService(storeKey)
			encCfg := moduletestutil.MakeTestEncodingConfig()
			k := NewKeeper(storeService, encCfg.Codec)
			err := k.RegisterTaskHandlers(types.TaskHandlers{
				types.NewNoArgsTaskHandler("noargs", nil, nil, nil),
			})
			require.NoError(t, err)

			ctx := testutil.DefaultContextWithKeys(
				map[string]*storetypes.KVStoreKey{"scheduler": storeKey},
				map[string]*storetypes.TransientStoreKey{"transient_test": storetypes.NewTransientStoreKey("transient_test")},
				nil,
			).WithHeaderInfo(header.Info{Height: 1, Time: now}).
				WithBlockHeight(1).
				WithBlockTime(now)

			// pre-insert tasks
			require.NoError(t, k.ScheduleTask(ctx, "noargs", "task1", nil, types.ScheduleAt(now.Add(10*time.Minute))))
			require.NoError(t, k.ScheduleTask(ctx, "noargs", "task2", nil, types.ScheduleAt(now.Add(10*time.Minute)), types.ScheduleEvery(&d1Hour)))
			// pause the task2 to test rescheduling a paused task
			require.NoError(t, k.tasksSchedule.Remove(ctx, collections.Join3[string, time.Time, types.TaskID]("noargs", at, "task2")))

			err = k.RescheduleTask(ctx, tc.id, tc.newSchedule...)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				task, err := k.tasks.Get(ctx, tc.id)
				require.NoError(t, err)
				require.Equal(t, tc.expectTask, &task)

				if tc.expectTask.ScheduledFor != nil {
					exists, err := k.tasksSchedule.Has(ctx, collections.Join3("noargs", *tc.expectTask.ScheduledFor, tc.expectTask.Id))
					require.NoError(t, err)
					require.True(t, exists)
				}
			}
		})
	}
}
