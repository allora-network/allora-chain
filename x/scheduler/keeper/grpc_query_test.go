package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/header"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/x/scheduler/keeper"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
)

func TestGRPCQueryTasks(t *testing.T) {
	now := time.Now().UTC()
	in10min := now.Add(10 * time.Minute)

	testCases := []struct {
		name       string
		req        *types.QueryTasksRequest
		expectResp *types.QueryTasksResponse
	}{
		{
			name: "unfiltered tasks",
			req: &types.QueryTasksRequest{
				Typename:   "",
				Pagination: &query.PageRequest{Limit: 2},
			},
			expectResp: &types.QueryTasksResponse{
				Tasks: []types.Task{
					{
						Id:           "task11",
						Typename:     "type1",
						ScheduledFor: &in10min,
					},
					{
						Id:       "task12",
						Typename: "type1",
					},
				},
				Pagination: &query.PageResponse{NextKey: []byte{0x74, 0x61, 0x73, 0x6b, 0x31, 0x33}}, // "task13"
			},
		},
		{
			name: "unfiltered tasks with pagination by offset",
			req: &types.QueryTasksRequest{
				Typename:   "",
				Pagination: &query.PageRequest{Limit: 2, Offset: 2},
			},
			expectResp: &types.QueryTasksResponse{
				Tasks: []types.Task{
					{
						Id:       "task13",
						Typename: "type1",
					},
					{
						Id:           "task21",
						Typename:     "type2",
						ScheduledFor: &in10min,
					},
				},
				Pagination: &query.PageResponse{
					NextKey: []byte{0x74, 0x61, 0x73, 0x6b, 0x32, 0x32}, // "task22"
				},
			},
		},
		{
			name: "unfiltered with pagination by key",
			req: &types.QueryTasksRequest{
				Typename:   "",
				Pagination: &query.PageRequest{Limit: 2, Key: []byte{0x74, 0x61, 0x73, 0x6b, 0x31, 0x33}},
			},
			expectResp: &types.QueryTasksResponse{
				Tasks: []types.Task{
					{
						Id:       "task13",
						Typename: "type1",
					},
					{
						Id:           "task21",
						Typename:     "type2",
						ScheduledFor: &in10min,
					},
				},
				Pagination: &query.PageResponse{
					NextKey: []byte{0x74, 0x61, 0x73, 0x6b, 0x32, 0x32}, // "task22"
				},
			},
		},
		{
			name: "filtered tasks unexisting type",
			req: &types.QueryTasksRequest{
				Typename:   "unknown",
				Pagination: nil,
			},
			expectResp: &types.QueryTasksResponse{
				Tasks:      nil,
				Pagination: &query.PageResponse{},
			},
		},
		{
			name: "filtered tasks",
			req: &types.QueryTasksRequest{
				Typename:   "type2",
				Pagination: &query.PageRequest{Limit: 2},
			},
			expectResp: &types.QueryTasksResponse{
				Tasks: []types.Task{
					{
						Id:           "task21",
						Typename:     "type2",
						ScheduledFor: &in10min,
					},
					{
						Id:       "task22",
						Typename: "type2",
					},
				},
				Pagination: &query.PageResponse{
					NextKey: []byte{0x74, 0x61, 0x73, 0x6b, 0x32, 0x33}, // "task23"
				},
			},
		},
		{
			name: "filtered tasks with pagination by offset",
			req: &types.QueryTasksRequest{
				Typename:   "type2",
				Pagination: &query.PageRequest{Limit: 2, Offset: 2},
			},
			expectResp: &types.QueryTasksResponse{
				Tasks: []types.Task{
					{
						Id:       "task23",
						Typename: "type2",
					},
				},
				Pagination: &query.PageResponse{},
			},
		},
		{
			name: "filtered tasks with pagination by key",
			req: &types.QueryTasksRequest{
				Typename:   "type2",
				Pagination: &query.PageRequest{Limit: 2, Key: []byte{0x74, 0x61, 0x73, 0x6b, 0x32, 0x33}},
			},
			expectResp: &types.QueryTasksResponse{
				Tasks: []types.Task{
					{
						Id:       "task23",
						Typename: "type2",
					},
				},
				Pagination: &query.PageResponse{},
			},
		},
	}

	storeKey := storetypes.NewKVStoreKey("scheduler")
	storeService := runtime.NewKVStoreService(storeKey)
	encCfg := moduletestutil.MakeTestEncodingConfig()
	k := keeper.NewKeeper(storeService, encCfg.Codec)

	require.NoError(t, k.RegisterTaskHandlers(types.TaskHandlers{
		types.NewNoArgsTaskHandler("type1", nil, nil, nil),
		types.NewNoArgsTaskHandler("type2", nil, nil, nil),
	}))

	ctx := testutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{"scheduler": storeKey},
		map[string]*storetypes.TransientStoreKey{"transient_test": storetypes.NewTransientStoreKey("transient_test")},
		nil,
	).WithHeaderInfo(header.Info{Height: 1, Time: now}).
		WithBlockHeight(1).
		WithBlockTime(now)

	require.NoError(t, k.ScheduleTask(ctx, "type1", "task11", nil, types.ScheduleAt(in10min)))
	require.NoError(t, k.ScheduleTask(ctx, "type1", "task12", nil))
	require.NoError(t, k.ScheduleTask(ctx, "type1", "task13", nil))
	require.NoError(t, k.ScheduleTask(ctx, "type2", "task21", nil, types.ScheduleAt(in10min)))
	require.NoError(t, k.ScheduleTask(ctx, "type2", "task22", nil))
	require.NoError(t, k.ScheduleTask(ctx, "type2", "task23", nil))

	querier := keeper.NewQuerier(k)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := querier.Tasks(ctx, tc.req)
			require.NoError(t, err)
			require.Equal(t, tc.expectResp, resp)
		})
	}
}

func TestGRPCQueryTask(t *testing.T) {
	now := time.Now().UTC()
	in10min := now.Add(10 * time.Minute)

	testCases := []struct {
		name       string
		reqID      string
		expectErr  bool
		expectTask *types.Task
	}{
		{
			name:       "task not found",
			reqID:      "nonexistent-task-id",
			expectErr:  true,
			expectTask: nil,
		},
		{
			name:      "task found",
			reqID:     "task-123",
			expectErr: false,
			expectTask: &types.Task{
				Id:                 "task-123",
				Typename:           "noargs",
				Args:               nil,
				ScheduledFor:       &in10min,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: 0,
			},
		},
		{
			name:      "task found with no schedule",
			reqID:     "task-paused",
			expectErr: false,
			expectTask: &types.Task{
				Id:                 "task-paused",
				Typename:           "noargs",
				Args:               nil,
				ScheduledFor:       nil,
				Interval:           nil,
				LastRunAt:          nil,
				RunCount:           0,
				SchedulingStrategy: 0,
			},
		},
	}

	storeKey := storetypes.NewKVStoreKey("scheduler")
	storeService := runtime.NewKVStoreService(storeKey)
	encCfg := moduletestutil.MakeTestEncodingConfig()
	k := keeper.NewKeeper(storeService, encCfg.Codec)

	require.NoError(t, k.RegisterTaskHandlers(types.TaskHandlers{types.NewNoArgsTaskHandler("noargs", nil, nil, nil)}))

	ctx := testutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{"scheduler": storeKey},
		map[string]*storetypes.TransientStoreKey{"transient_test": storetypes.NewTransientStoreKey("transient_test")},
		nil,
	).WithHeaderInfo(header.Info{Height: 1, Time: now}).
		WithBlockHeight(1).
		WithBlockTime(now)

	require.NoError(t, k.ScheduleTask(ctx, "noargs", "task-123", nil, types.ScheduleAt(in10min)))
	require.NoError(t, k.ScheduleTask(ctx, "noargs", "task-paused", nil))

	querier := keeper.NewQuerier(k)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := querier.Task(ctx, &types.QueryTaskRequest{TaskId: tc.reqID})
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectTask, &resp.Task)
			}
		})
	}
}

func TestGRPCQueryScheduledTasks(t *testing.T) {
	now := time.Now().UTC()
	in10min := now.Add(10 * time.Minute)
	in15min := now.Add(15 * time.Minute)
	in20min := now.Add(20 * time.Minute)
	in30min := now.Add(30 * time.Minute)

	encodeKey := func(at time.Time, taskID string) []byte {
		buf := make([]byte, sdk.TimeKey.Size(at)+types.TaskIDKey.Size(types.TaskID(taskID)))
		l, err := sdk.TimeKey.Encode(buf, at)
		require.NoError(t, err)
		_, err = types.TaskIDKey.Encode(buf[l:], types.TaskID(taskID))
		require.NoError(t, err)
		return buf
	}

	testCases := []struct {
		name       string
		req        *types.QueryScheduledTasksRequest
		expectErr  bool
		expectResp *types.QueryScheduledTasksResponse
	}{
		{
			name: "not filtered by schedule",
			req: &types.QueryScheduledTasksRequest{
				Typename:   "type1",
				Pagination: &query.PageRequest{Limit: 2},
			},
			expectErr: false,
			expectResp: &types.QueryScheduledTasksResponse{
				Tasks: []types.Task{
					{
						Id:           "task11",
						Typename:     "type1",
						ScheduledFor: &in10min,
					},
					{
						Id:           "task12",
						Typename:     "type1",
						ScheduledFor: &in20min,
					},
				},
				Pagination: &query.PageResponse{NextKey: encodeKey(in30min, "task13")},
			},
		},
		{
			name: "not filtered by schedule with pagination by offset",
			req: &types.QueryScheduledTasksRequest{
				Typename:   "type1",
				Pagination: &query.PageRequest{Limit: 2, Offset: 2},
			},
			expectErr: false,
			expectResp: &types.QueryScheduledTasksResponse{
				Tasks: []types.Task{
					{
						Id:           "task13",
						Typename:     "type1",
						ScheduledFor: &in30min,
					},
				},
				Pagination: &query.PageResponse{},
			},
		},
		{
			name: "not filtered by schedule with pagination by key",
			req: &types.QueryScheduledTasksRequest{
				Typename:   "type2",
				Pagination: &query.PageRequest{Limit: 2, Key: encodeKey(in20min, "task23")},
			},
			expectErr: false,
			expectResp: &types.QueryScheduledTasksResponse{
				Tasks: []types.Task{
					{
						Id:           "task23",
						Typename:     "type2",
						ScheduledFor: &in30min,
					},
				},
				Pagination: &query.PageResponse{},
			},
		},
		{
			name: "unexisting type",
			req: &types.QueryScheduledTasksRequest{
				Typename:   "unknown",
				Pagination: nil,
			},
			expectErr: false,
			expectResp: &types.QueryScheduledTasksResponse{
				Tasks:      nil,
				Pagination: &query.PageResponse{},
			},
		},
		{
			name: "filtered by schedule",
			req: &types.QueryScheduledTasksRequest{
				Typename:   "type2",
				From:       &in15min,
				Pagination: &query.PageRequest{Limit: 2},
			},
			expectErr: false,
			expectResp: &types.QueryScheduledTasksResponse{
				Tasks: []types.Task{
					{
						Id:           "task22",
						Typename:     "type2",
						ScheduledFor: &in20min,
					},
					{
						Id:           "task23",
						Typename:     "type2",
						ScheduledFor: &in30min,
					},
				},
				Pagination: &query.PageResponse{},
			},
		},
		{
			name: "filtered by schedule is inclusive",
			req: &types.QueryScheduledTasksRequest{
				Typename:   "type2",
				From:       &in10min,
				Pagination: &query.PageRequest{Limit: 2},
			},
			expectErr: false,
			expectResp: &types.QueryScheduledTasksResponse{
				Tasks: []types.Task{
					{
						Id:           "task21",
						Typename:     "type2",
						ScheduledFor: &in10min,
					},
					{
						Id:           "task22",
						Typename:     "type2",
						ScheduledFor: &in20min,
					},
				},
				Pagination: &query.PageResponse{
					NextKey: encodeKey(in30min, "task23"),
				},
			},
		},
		{
			name: "filtered by schedule with pagination by offset is forbidden",
			req: &types.QueryScheduledTasksRequest{
				Typename:   "type2",
				From:       &in15min,
				Pagination: &query.PageRequest{Limit: 2, Offset: 1},
			},
			expectErr:  true,
			expectResp: nil,
		},
		{
			name: "filtered by schedule with pagination by key",
			req: &types.QueryScheduledTasksRequest{
				Typename:   "type2",
				From:       &in15min,
				Pagination: &query.PageRequest{Limit: 2, Key: encodeKey(in30min, "task23")},
			},
			expectResp: &types.QueryScheduledTasksResponse{
				Tasks: []types.Task{
					{
						Id:           "task23",
						Typename:     "type2",
						ScheduledFor: &in30min,
					},
				},
				Pagination: &query.PageResponse{},
			},
		},
		{
			name: "filtered by schedule with reverse order",
			req: &types.QueryScheduledTasksRequest{
				Typename:   "type2",
				From:       &in15min,
				Pagination: &query.PageRequest{Limit: 2, Reverse: true},
			},
			expectResp: &types.QueryScheduledTasksResponse{
				Tasks: []types.Task{
					{
						Id:           "task21",
						Typename:     "type2",
						ScheduledFor: &in10min,
					},
				},
				Pagination: &query.PageResponse{},
			},
		},
	}

	storeKey := storetypes.NewKVStoreKey("scheduler")
	storeService := runtime.NewKVStoreService(storeKey)
	encCfg := moduletestutil.MakeTestEncodingConfig()
	k := keeper.NewKeeper(storeService, encCfg.Codec)

	require.NoError(t, k.RegisterTaskHandlers(types.TaskHandlers{
		types.NewNoArgsTaskHandler("type1", nil, nil, nil),
		types.NewNoArgsTaskHandler("type2", nil, nil, nil),
	}))

	ctx := testutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{"scheduler": storeKey},
		map[string]*storetypes.TransientStoreKey{"transient_test": storetypes.NewTransientStoreKey("transient_test")},
		nil,
	).WithHeaderInfo(header.Info{Height: 1, Time: now}).
		WithBlockHeight(1).
		WithBlockTime(now)

	require.NoError(t, k.ScheduleTask(ctx, "type1", "task13", nil, types.ScheduleAt(in30min)))
	require.NoError(t, k.ScheduleTask(ctx, "type1", "task12", nil, types.ScheduleAt(in20min)))
	require.NoError(t, k.ScheduleTask(ctx, "type1", "task11", nil, types.ScheduleAt(in10min)))
	require.NoError(t, k.ScheduleTask(ctx, "type2", "task23", nil, types.ScheduleAt(in30min)))
	require.NoError(t, k.ScheduleTask(ctx, "type2", "task22", nil, types.ScheduleAt(in20min)))
	require.NoError(t, k.ScheduleTask(ctx, "type2", "task21", nil, types.ScheduleAt(in10min)))
	require.NoError(t, k.ScheduleTask(ctx, "type2", "task24", nil))

	querier := keeper.NewQuerier(k)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := querier.ScheduledTasks(ctx, tc.req)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectResp, resp)
			}
		})
	}
}

func TestGRPCQueryHandlers(t *testing.T) {
	testCases := []struct {
		name        string
		handlers    types.TaskHandlers
		expectOrder []string
	}{
		{
			name:        "no handlers",
			handlers:    nil,
			expectOrder: nil,
		},
		{
			name: "a single handler",
			handlers: types.TaskHandlers{
				types.NewNoArgsTaskHandler("handler1", nil, nil, nil),
			},
			expectOrder: []string{"handler1"},
		},
		{
			name: "handler without deps should be ordered alphanumerically",
			handlers: types.TaskHandlers{
				types.NewNoArgsTaskHandler("handlerB", nil, nil, nil),
				types.NewNoArgsTaskHandler("handlerA", nil, nil, nil),
				types.NewNoArgsTaskHandler("handlerC2", nil, nil, nil),
				types.NewNoArgsTaskHandler("handlerC1", nil, nil, nil),
			},
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
			expectOrder: []string{"handlerA", "handlerB", "handlerD", "handlerC"},
		},
	}

	storeService := runtime.NewKVStoreService(storetypes.NewKVStoreKey("emissions"))
	encCfg := moduletestutil.MakeTestEncodingConfig()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			k := keeper.NewKeeper(storeService, encCfg.Codec)
			require.NoError(t, k.RegisterTaskHandlers(tc.handlers))

			ctx := testutil.DefaultContextWithKeys(
				map[string]*storetypes.KVStoreKey{},
				map[string]*storetypes.TransientStoreKey{},
				nil,
			)

			req, err := keeper.NewQuerier(k).Handlers(ctx, &types.QueryHandlersRequest{})
			require.NoError(t, err)
			require.Equal(t, tc.expectOrder, req.Handlers)
		})
	}
}
