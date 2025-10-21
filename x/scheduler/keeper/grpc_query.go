package keeper

import (
	"context"
	"errors"
	"time"

	"cosmossdk.io/collections"
	collutils "github.com/allora-network/allora-chain/utils/collections"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Querier implements the gRPC query service for the scheduler module.
type Querier struct {
	*Keeper
}

var _ types.QueryServer = Querier{} //nolint:exhaustruct

func NewQuerier(keeper *Keeper) Querier {
	return Querier{Keeper: keeper}
}

func (q Querier) Tasks(ctx context.Context, req *types.QueryTasksRequest) (*types.QueryTasksResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	var tasks []types.Task
	var page *query.PageResponse
	var err error
	if req.Typename == "" {
		tasks, page, err = query.CollectionPaginate(
			ctx,
			q.tasks,
			req.Pagination,
			func(key types.TaskID, value types.Task) (types.Task, error) {
				return value, nil
			},
		)
	} else {
		tasks, page, err = query.CollectionPaginate(
			ctx,
			collutils.WrapMultiIndexToCollection(q.tasks.Indexes.ByType),
			req.Pagination,
			func(key collections.Pair[string, types.TaskID], _ collections.NoValue) (types.Task, error) {
				return q.tasks.Get(ctx, key.K2())
			},
			query.WithCollectionPaginationPairPrefix[string, types.TaskID](req.Typename),
		)
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTasksResponse{
		Tasks:      tasks,
		Pagination: page,
	}, nil
}

func (q Querier) Task(ctx context.Context, req *types.QueryTaskRequest) (*types.QueryTaskResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	task, err := q.GetTask(ctx, types.TaskID(req.TaskId))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTaskResponse{Task: task}, nil
}

func (q Querier) ScheduledTasks(ctx context.Context, req *types.QueryScheduledTasksRequest) (*types.QueryScheduledTasksResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.Typename == "" {
		return nil, status.Error(codes.InvalidArgument, "empty typename")
	}

	if req.From != nil {
		// offset pagination is not supported with from filter
		if req.Pagination.Offset != 0 {
			return nil, status.Error(codes.InvalidArgument, "cannot use offset pagination with from filter")
		}

		// If a from filter is provided without a pagination key, we need to set the key
		if req.Pagination.Key == nil {
			prefix := make([]byte, collections.StringKey.Size(req.Typename))
			if _, err := collections.StringKey.Encode(prefix, req.Typename); err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			key, err := collections.EncodeKeyWithPrefix(nil, sdk.TimeKey, *req.From)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			req.Pagination.Key = key
		}
	}

	tasks, page, err := query.CollectionPaginate(
		ctx,
		q.tasksSchedule,
		req.Pagination,
		func(key collections.Triple[string, time.Time, types.TaskID], _ collections.NoValue) (types.Task, error) {
			return q.tasks.Get(ctx, key.K3())
		},
		collutils.WithCollectionPaginationTriplePrefix[string, time.Time, types.TaskID](req.Typename),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryScheduledTasksResponse{
		Tasks:      tasks,
		Pagination: page,
	}, nil
}

func (q Querier) Handlers(_ context.Context, _ *types.QueryHandlersRequest) (*types.QueryHandlersResponse, error) {
	return &types.QueryHandlersResponse{Handlers: q.handlersOrder}, nil
}
