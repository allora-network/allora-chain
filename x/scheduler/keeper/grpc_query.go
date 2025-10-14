package keeper

import (
	"context"
	"errors"
	"time"

	"cosmossdk.io/collections"
	collutils "github.com/allora-network/allora-chain/utils/collections"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Querier implements the gRPC query service for the scheduler module.
type Querier struct {
	*Keeper
}

var _ types.QueryServer = Querier{}

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
		tasks, page, err = q.getAllTasks(ctx, req.Pagination)
	} else {
		tasks, page, err = q.getTasksByType(ctx, req.Typename, req.Pagination)
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTasksResponse{
		Tasks:      tasks,
		Pagination: page,
	}, nil
}

func (q Querier) getAllTasks(ctx context.Context, pagination *query.PageRequest) ([]types.Task, *query.PageResponse, error) {
	return query.CollectionPaginate(
		ctx,
		q.tasks,
		pagination,
		func(key types.TaskID, value types.Task) (types.Task, error) {
			return value, nil
		},
	)
}

func (q Querier) getTasksByType(ctx context.Context, typename string, pagination *query.PageRequest) ([]types.Task, *query.PageResponse, error) {
	return query.CollectionPaginate(
		ctx,
		collutils.WrapMultiIndexToCollection(q.tasks.Indexes.ByType),
		pagination,
		func(key collections.Pair[string, types.TaskID], _ collections.NoValue) (types.Task, error) {
			return q.tasks.Get(ctx, key.K2())
		},
		query.WithCollectionPaginationPairPrefix[string, types.TaskID](typename),
	)
}

func (q Querier) Task(ctx context.Context, req *types.QueryTaskRequest) (*types.QueryTaskResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	task, err := q.GetTask(ctx, req.TaskId)
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

	var prefixOpt func(opt *query.CollectionsPaginateOptions[collections.Triple[string, time.Time, types.TaskID]])
	if req.From != nil {
		prefixOpt = collutils.WithCollectionPaginationTripleSuperPrefix[string, time.Time, types.TaskID](req.Typename, *req.From)
	} else {
		prefixOpt = collutils.WithCollectionPaginationTriplePrefix[string, time.Time, types.TaskID](req.Typename)
	}

	tasks, page, err := query.CollectionPaginate(
		ctx,
		q.tasksSchedule,
		req.Pagination,
		func(key collections.Triple[string, time.Time, types.TaskID], _ collections.NoValue) (types.Task, error) {
			return q.tasks.Get(ctx, key.K3())
		},
		prefixOpt,
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
