package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	collutils "github.com/allora-network/allora-chain/utils/collections"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Querier is used as Keeper will have duplicate methods if used directly, and gRPC names take precedence over keeper
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
	var pageRes *query.PageResponse
	var err error
	if req.Typename == "" {
		tasks, pageRes, err = q.getAllTasks(ctx, req.Pagination)
	} else {
		tasks, pageRes, err = q.getTasksByType(ctx, req.Typename, req.Pagination)
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryTasksResponse{
		Tasks:      tasks,
		Pagination: pageRes,
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
