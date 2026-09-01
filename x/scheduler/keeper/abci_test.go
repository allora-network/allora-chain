//nolint:exhaustruct
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
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/gogoproto/proto"
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
		func(ctx context.Context, task types.Task, a *cosmostypes.Coin) error {
			handlerARunCallCount += 1

			// handler A must be called first
			require.Equal(t, 0, handlerBArbitrateCallCount)
			require.Equal(t, 0, handlerBRunCallCount)

			require.Equal(t, types.TaskID("taskA2"), task.Id)
			require.Equal(t, args, a)
			require.Equal(t, uint64(0), task.RunCount)

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

			return nil, nil //nolint:nilnil
		},
		func(ctx context.Context, task types.Task) error {
			handlerBRunCallCount += 1
			bTaskRun = append(bTaskRun, task.Id)

			// handler A must be called first
			require.Equal(t, 1, handlerAArbitrateCallCount)
			require.Equal(t, 1, handlerARunCallCount)

			require.Equal(t, uint64(0), task.RunCount)

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
		require.ErrorIs(t, err, collections.ErrNotFound)
	}

	// check taskA3 still exists
	_, err := k.tasks.Get(ctx, "taskA3")
	require.NoError(t, err)

	// only one task should remain scheduled
	it, err := k.tasks.IterateRaw(ctx, nil, nil, collections.OrderAscending)
	require.NoError(t, err)
	keys, err := it.Keys()
	require.NoError(t, err)
	require.Len(t, keys, 1)
	it2, err := k.tasksSchedule.IterateRaw(ctx, nil, nil, collections.OrderAscending)
	require.NoError(t, err)
	keys2, err := it2.Keys()
	require.NoError(t, err)
	require.Len(t, keys2, 1)
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

					return nil, nil //nolint:nilnil
				},
				func(ctx context.Context, task types.Task) error {
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
		name         string
		taskID       types.TaskID
		decision     types.ArbitrageDecision
		expectError  bool
		expectEvents []proto.Message
		expectTask   *types.Task
	}{
		{
			name:   "cancel task",
			taskID: "task1",
			decision: types.ArbitrageDecision{
				Action: types.ArbitrageActionCancel,
			},
			expectError: false,
			expectEvents: []proto.Message{
				&types.EventTaskUnscheduled{
					Id:       "task1",
					Typename: "noargs",
					At:       &in5min,
				},
			},
			expectTask: nil,
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
			expectEvents: []proto.Message{
				&types.EventTaskScheduled{
					Id:       "task1",
					Typename: "noargs",
					At:       &in10min,
				},
			},
			expectTask: &types.Task{
				Id:                 "task1",
				Typename:           "noargs",
				Args:               nil,
				ScheduledFor:       &in10min,
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
			expectError:  false,
			expectEvents: nil,
			expectTask: &types.Task{
				Id:                 "task1",
				Typename:           "noargs",
				Args:               nil,
				ScheduledFor:       &in5min,
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
			expectEvents: nil,
			expectError:  true,
			expectTask:   nil,
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

			ctx = ctx.WithEventManager(cosmostypes.NewEventManager())

			err = k.applyArbitrageDecision(ctx, tc.taskID, tc.decision)
			if tc.expectError {
				require.Error(t, err)
			} else {
				task, err := k.tasks.Get(ctx, tc.taskID)
				if tc.expectTask == nil {
					require.ErrorIs(t, err, collections.ErrNotFound)
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.expectTask, &task)
				}
			}

			evts := parseEvents(t, ctx.EventManager().Events().ToABCIEvents())
			require.Equal(t, tc.expectEvents, evts)
		})
	}
}

func TestExecuteTask(t *testing.T) {
	now := time.Now().UTC()
	interval := 10 * time.Minute
	in10Min := now.Add(interval)

	testCases := []struct {
		name                string
		taskID              types.TaskID
		runFail             bool
		expectError         bool
		expectEvents        []proto.Message
		expectTaskRemoval   bool
		expectNewScheduleAt *time.Time
	}{
		{
			name:        "successful run non-periodic task",
			taskID:      "task1",
			runFail:     false,
			expectError: false,
			expectEvents: []proto.Message{
				&types.EventTaskExecuted{
					Id:       "task1",
					Typename: "withargs",
					At:       &now,
				},
			},
			expectTaskRemoval:   true,
			expectNewScheduleAt: nil,
		},
		{
			name:        "successful run periodic task",
			taskID:      "task2",
			runFail:     false,
			expectError: false,
			expectEvents: []proto.Message{
				&types.EventTaskExecuted{
					Id:       "task2",
					Typename: "withargs",
					At:       &now,
				},
				&types.EventTaskScheduled{
					Id:       "task2",
					Typename: "withargs",
					At:       &in10Min,
				},
			},
			expectTaskRemoval:   false,
			expectNewScheduleAt: &in10Min,
		},
		{
			name:                "failed run task",
			taskID:              "task1",
			runFail:             true,
			expectError:         true,
			expectEvents:        nil,
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
				func(ctx context.Context, task types.Task, a *cosmostypes.Coin) error {
					require.Equal(t, tc.taskID, task.Id)
					require.Equal(t, args, a)
					require.Equal(t, uint64(0), task.RunCount)

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

			ctx = ctx.WithBlockTime(now).WithEventManager(cosmostypes.NewEventManager())
			task, err := k.tasks.Get(ctx, tc.taskID)
			require.NoError(t, err)
			err = k.executeTask(ctx, task, handler)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				taskAfterRun, err := k.tasks.Get(ctx, tc.taskID)
				if tc.expectTaskRemoval {
					require.ErrorIs(t, err, collections.ErrNotFound)
				} else {
					require.NoError(t, err)
					require.NoError(t, err)
					require.Equal(t, uint64(1), taskAfterRun.RunCount)
					require.NotNil(t, taskAfterRun.LastRunAt)
					require.Equal(t, now, *taskAfterRun.LastRunAt)
					if tc.expectNewScheduleAt != nil {
						require.Equal(t, *tc.expectNewScheduleAt, *taskAfterRun.ScheduledFor)
						exists, err := k.tasksSchedule.Has(ctx, collections.Join3("withargs", *tc.expectNewScheduleAt, tc.taskID))
						require.NoError(t, err)
						require.True(t, exists)
					}
				}
			}

			evts := parseEvents(t, ctx.EventManager().Events().ToABCIEvents())
			require.Equal(t, tc.expectEvents, evts)
		})
	}
}

func parseEvents(t *testing.T, events []abci.Event) []proto.Message {
	t.Helper()

	var parsed []proto.Message
	for _, e := range events {
		evt, err := cosmostypes.ParseTypedEvent(e)
		require.NoError(t, err)
		parsed = append(parsed, evt)
	}
	return parsed
}
