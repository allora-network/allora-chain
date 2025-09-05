package keeper

import (
	"context"
	"errors"
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/header"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/stretchr/testify/require"
)

func TestBeginBlocker(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey("scheduler")
	storeService := runtime.NewKVStoreService(storeKey)
	encCfg := moduletestutil.MakeTestEncodingConfig()
	k := NewKeeper(storeService, encCfg.Codec)

	handlerAArbitrateCallCount := 0
	handlerBArbitrateCallCount := 0
	handlerARunCallCount := 0
	handlerBRunCallCount := 0
	var bTaskRun []types.TaskID

	args := &cosmostypes.Coin{Denom: "udenom", Amount: math.NewInt(12000)}

	handlerA := types.NewTaskHandler[*cosmostypes.Coin](
		"handlerA",
		nil,
		func(ctx context.Context, tasks []types.Invocation[*cosmostypes.Coin]) (map[types.TaskID]types.ArbitrageDecision, error) {
			handlerAArbitrateCallCount += 1

			// handler A must be called first
			require.Equal(t, 0, handlerBArbitrateCallCount)
			require.Equal(t, 0, handlerBRunCallCount)

			require.Equal(t, []types.Invocation[*cosmostypes.Coin]{
				{TaskID: "taskA2", Args: args},
				{TaskID: "taskA1", Args: args},
			}, tasks)

			return map[types.TaskID]types.ArbitrageDecision{
				"taskA1": {Action: types.ArbitrageActionCancel},
			}, nil
		},
		func(ctx context.Context, id types.TaskID, a *cosmostypes.Coin, runCount uint64) error {
			handlerARunCallCount += 1

			// handler A must be called first
			require.Equal(t, 0, handlerBArbitrateCallCount)
			require.Equal(t, 0, handlerBRunCallCount)

			require.Equal(t, types.TaskID("taskA2"), id)
			require.Equal(t, args, a)
			require.Equal(t, uint64(1), runCount)

			return nil
		},
	)
	handlerB := types.NewNoArgsTaskHandler(
		"handlerB",
		[]string{"handlerA"},
		func(ctx context.Context, tasks []types.TaskID) (map[types.TaskID]types.ArbitrageDecision, error) {
			handlerBArbitrateCallCount += 1

			// handler A must be called first
			require.Equal(t, 1, handlerAArbitrateCallCount)
			require.Equal(t, 1, handlerARunCallCount)

			require.Equal(t, []types.TaskID{"taskB1", "taskB2"}, tasks)

			return nil, nil
		},
		func(ctx context.Context, id types.TaskID, runCount uint64) error {
			handlerBRunCallCount += 1
			bTaskRun = append(bTaskRun, id)

			// handler A must be called first
			require.Equal(t, 1, handlerAArbitrateCallCount)
			require.Equal(t, 1, handlerARunCallCount)

			require.Equal(t, uint64(1), runCount)

			return nil
		},
	)
	require.NoError(t, k.RegisterTaskHandlers(types.TaskHandlers{handlerB, handlerA}))

	now := time.Now().UTC()
	ctx := testutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{"scheduler": storeKey},
		map[string]*storetypes.TransientStoreKey{"transient_test": storetypes.NewTransientStoreKey("transient_test")},
		nil,
	).WithBlockHeight(1).
		WithBlockTime(now)

	require.NoError(t, k.ScheduleTask(ctx, "handlerA", "taskA1", args, types.ScheduleAt(now.Add(3*time.Minute)))) // will be due
	require.NoError(t, k.ScheduleTask(ctx, "handlerA", "taskA2", args, types.ScheduleAt(now.Add(2*time.Minute)))) // will be due
	require.NoError(t, k.ScheduleTask(ctx, "handlerA", "taskA3", args, types.ScheduleAt(now.Add(20*time.Minute))))
	require.NoError(t, k.ScheduleTask(ctx, "handlerB", "taskB1", nil, types.ScheduleAt(now.Add(time.Minute))))   // will be due
	require.NoError(t, k.ScheduleTask(ctx, "handlerB", "taskB2", nil, types.ScheduleAt(now.Add(2*time.Minute)))) // will be due

	// advance in time to make some tasks due
	ctx = ctx.WithBlockTime(now.Add(10 * time.Minute))

	require.NoError(t, k.BeginBlock(ctx))

	// Check all expected handler calls were made
	require.Equal(t, 1, handlerAArbitrateCallCount)
	require.Equal(t, 1, handlerBArbitrateCallCount)
	require.Equal(t, 1, handlerARunCallCount)
	require.Equal(t, 2, handlerBRunCallCount)
	require.Equal(t, []types.TaskID{"taskB1", "taskB2"}, bTaskRun)

	// check taskA1, taskA2, taskB1 and taskB2 were deleted
	for _, id := range []types.TaskID{"taskA1", "taskA2", "taskB1", "taskB2"} {
		_, err := k.tasks.Get(ctx, id)
		require.True(t, errors.Is(err, collections.ErrNotFound))
	}

	// check taskA3 still exists
	_, err := k.tasks.Get(ctx, "taskA3")
	require.NoError(t, err)

	// only one task should remain scheduled
	it, err := k.tasks.IterateRaw(ctx, nil, nil, collections.OrderAscending)
	require.NoError(t, err)
	keys, err := it.Keys()
	require.NoError(t, err)
	require.Equal(t, 1, len(keys))
	it2, err := k.tasksSchedule.IterateRaw(ctx, nil, nil, collections.OrderAscending)
	require.NoError(t, err)
	keys2, err := it2.Keys()
	require.NoError(t, err)
	require.Equal(t, 1, len(keys2))
}

func TestBeginBlockerShouldErrIfHandlerErr(t *testing.T) {
	testCases := []struct {
		name          string
		arbitrateErr  bool
		runErr        bool
		expectRunCall bool
	}{
		{
			name:          "arbitrate error",
			arbitrateErr:  true,
			runErr:        false,
			expectRunCall: false,
		},
		{
			name:          "run error",
			arbitrateErr:  false,
			runErr:        true,
			expectRunCall: true,
		},
	}

	for _, tc := range testCases {
		storeKey := storetypes.NewKVStoreKey("scheduler")
		storeService := runtime.NewKVStoreService(storeKey)
		encCfg := moduletestutil.MakeTestEncodingConfig()
		k := NewKeeper(storeService, encCfg.Codec)

		runCalled := false

		require.NoError(t, k.RegisterTaskHandlers(types.TaskHandlers{
			types.NewNoArgsTaskHandler(
				"handler",
				nil,
				func(ctx context.Context, tasks []types.TaskID) (map[types.TaskID]types.ArbitrageDecision, error) {
					if tc.arbitrateErr {
						return nil, errors.New("arbitrate error")
					}

					return nil, nil
				},
				func(ctx context.Context, id types.TaskID, runCount uint64) error {
					runCalled = true
					if tc.runErr {
						return errors.New("run error")
					}

					return nil
				},
			),
		}))

		now := time.Now().UTC()
		ctx := testutil.DefaultContextWithKeys(
			map[string]*storetypes.KVStoreKey{"scheduler": storeKey},
			map[string]*storetypes.TransientStoreKey{"transient_test": storetypes.NewTransientStoreKey("transient_test")},
			nil,
		).WithBlockHeight(1).
			WithBlockTime(now)

		require.NoError(t, k.ScheduleTask(ctx, "handler", "task", nil, types.ScheduleAt(now.Add(3*time.Minute))))

		// advance in time to make some tasks due
		ctx = ctx.WithBlockTime(now.Add(10 * time.Minute))

		require.Error(t, k.BeginBlock(ctx))
		require.Equal(t, tc.expectRunCall, runCalled)
	}
}

func TestApplyArbitrageDecision(t *testing.T) {
	now := time.Now()
	in5min := now.Add(5 * time.Minute).UTC()
	in10min := now.Add(10 * time.Minute).UTC()

	testCases := []struct {
		name        string
		taskID      types.TaskID
		decision    types.ArbitrageDecision
		expectError bool
		expectTask  *types.Task
	}{
		{
			name:   "cancel task",
			taskID: "task1",
			decision: types.ArbitrageDecision{
				Action: types.ArbitrageActionCancel,
			},
			expectError: false,
			expectTask:  nil,
		},
		{
			name:   "reschedule task",
			taskID: "task1",
			decision: types.ArbitrageDecision{
				Action: types.ArbitrageActionReschedule,
				RescheduleOpts: []types.SchedulingOption{
					types.ScheduleAt(in10min),
				},
			},
			expectError: false,
			expectTask: &types.Task{
				Id:                 "task1",
				Typename:           "noargs",
				Args:               nil,
				NextRunAt:          &in10min,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: types.SchedulingStrategy_RELATIVE,
			},
		},
		{
			name:   "reschedule task next block",
			taskID: "task1",
			decision: types.ArbitrageDecision{
				Action: types.ArbitrageActionReschedule,
			},
			expectError: false,
			expectTask: &types.Task{
				Id:                 "task1",
				Typename:           "noargs",
				Args:               nil,
				NextRunAt:          &in5min,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: types.SchedulingStrategy_RELATIVE,
			},
		},
		{
			name:   "wrong schedule",
			taskID: "task1",
			decision: types.ArbitrageDecision{
				Action: types.ArbitrageActionReschedule,
				RescheduleOpts: []types.SchedulingOption{
					types.ScheduleAt(now.Add(-10 * time.Minute)),
				},
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
			})
			require.NoError(t, err)

			ctx := testutil.DefaultContextWithKeys(
				map[string]*storetypes.KVStoreKey{"scheduler": storeKey},
				map[string]*storetypes.TransientStoreKey{"transient_test": storetypes.NewTransientStoreKey("transient_test")},
				nil,
			).WithHeaderInfo(header.Info{Height: 1, Time: now}).
				WithBlockHeight(1).
				WithBlockTime(now)

			require.NoError(t, k.ScheduleTask(ctx, "noargs", "task1", nil, types.ScheduleAt(in5min)))

			err = k.applyArbitrageDecision(ctx, tc.taskID, tc.decision)
			if tc.expectError {
				require.Error(t, err)
			} else {
				task, err := k.tasks.Get(ctx, tc.taskID)
				if tc.expectTask == nil {
					require.True(t, errors.Is(err, collections.ErrNotFound))
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.expectTask, &task)
				}
			}
		})
	}
}

func TestRunTask(t *testing.T) {
	now := time.Now().UTC()
	interval := 10 * time.Minute
	in10Min := now.Add(interval)

	testCases := []struct {
		name                string
		taskID              types.TaskID
		runFail             bool
		expectError         bool
		expectTaskRemoval   bool
		expectNewScheduleAt *time.Time
	}{
		{
			name:                "successful run non-periodic task",
			taskID:              "task1",
			runFail:             false,
			expectError:         false,
			expectTaskRemoval:   true,
			expectNewScheduleAt: nil,
		},
		{
			name:                "successful run periodic task",
			taskID:              "task2",
			runFail:             false,
			expectError:         false,
			expectTaskRemoval:   false,
			expectNewScheduleAt: &in10Min,
		},
		{
			name:                "failed run task",
			taskID:              "task1",
			runFail:             true,
			expectError:         true,
			expectTaskRemoval:   false,
			expectNewScheduleAt: nil,
		},
	}

	args := &cosmostypes.Coin{Denom: "udenom", Amount: math.NewInt(12000)}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storeKey := storetypes.NewKVStoreKey("scheduler")
			storeService := runtime.NewKVStoreService(storeKey)
			encCfg := moduletestutil.MakeTestEncodingConfig()
			k := NewKeeper(storeService, encCfg.Codec)

			handler := types.NewTaskHandler[*cosmostypes.Coin](
				"withargs",
				nil,
				nil,
				func(ctx context.Context, id types.TaskID, a *cosmostypes.Coin, runCount uint64) error {
					require.Equal(t, tc.taskID, id)
					require.Equal(t, args, a)
					require.Equal(t, uint64(1), runCount)

					if tc.runFail {
						return errors.New("run failure")
					}
					return nil
				},
			)
			require.NoError(t, k.RegisterTaskHandlers(types.TaskHandlers{handler}))

			ctx := testutil.DefaultContextWithKeys(
				map[string]*storetypes.KVStoreKey{"scheduler": storeKey},
				map[string]*storetypes.TransientStoreKey{"transient_test": storetypes.NewTransientStoreKey("transient_test")},
				nil,
			).WithBlockHeight(1).
				WithBlockTime(now.Add(-time.Minute))

			require.NoError(t, k.ScheduleTask(ctx, "withargs", "task1", args, types.ScheduleAt(now)))
			require.NoError(t, k.ScheduleTask(ctx, "withargs", "task2", args, types.ScheduleAt(now), types.ScheduleEvery(&interval)))

			ctx = ctx.WithBlockTime(now)
			task, err := k.tasks.Get(ctx, tc.taskID)
			require.NoError(t, err)
			err = k.runTask(ctx, task, handler)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				task, err = k.tasks.Get(ctx, tc.taskID)
				if tc.expectTaskRemoval {
					require.True(t, errors.Is(err, collections.ErrNotFound))
				} else {
					require.NoError(t, err)
					task, err := k.tasks.Get(ctx, tc.taskID)
					require.NoError(t, err)
					require.Equal(t, uint64(1), task.RunCount)
					require.NotNil(t, task.LastRunAt)
					require.Equal(t, now, *task.LastRunAt)
					if tc.expectNewScheduleAt != nil {
						require.Equal(t, *tc.expectNewScheduleAt, *task.NextRunAt)
						exists, err := k.tasksSchedule.Has(ctx, collections.Join3("withargs", *tc.expectNewScheduleAt, tc.taskID))
						require.NoError(t, err)
						require.True(t, exists)
					}
				}
			}
		})
	}
}
