package keeper

import (
	"context"
	"fmt"
	"sort"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/core/store"
	"cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

func NewTasksIndexes(sb *collections.SchemaBuilder) TasksIndexes {
	return TasksIndexes{
		ByType: indexes.NewMulti(
			sb, types.TasksByTypeKeyPrefix, "tasks_by_type", collections.StringKey, types.TaskIDKey,
			func(_ types.TaskID, t types.Task) (string, error) {
				return t.Typename, nil
			},
		),
	}
}

type TasksIndexes struct {
	// ByType indexes tasks by their type.
	ByType *indexes.Multi[string, types.TaskID, types.Task]
}

func (i TasksIndexes) IndexesList() []collections.Index[types.TaskID, types.Task] {
	return []collections.Index[types.TaskID, types.Task]{
		i.ByType,
	}
}

type Keeper struct {
	storeService store.KVStoreService
	cdc          codec.Codec
	schema       collections.Schema

	handlersByTypename map[string]types.TaskHandler
	handlersOrder      []string

	tasks         *collections.IndexedMap[types.TaskID, types.Task, TasksIndexes]
	tasksSchedule collections.KeySet[collections.Triple[string, time.Time, types.TaskID]]
}

// NewKeeper returns a new keeper by codec and storeKey inputs.
func NewKeeper(storeService store.KVStoreService, cdc codec.Codec) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	//nolint:exhaustruct
	k := Keeper{
		storeService: storeService,
		cdc:          cdc,

		handlersByTypename: make(map[string]types.TaskHandler),
		handlersOrder:      nil,

		tasks:         collections.NewIndexedMap(sb, types.TasksKeyPrefix, "tasks", types.TaskIDKey, codec.CollValue[types.Task](cdc), NewTasksIndexes(sb)),
		tasksSchedule: collections.NewKeySet(sb, types.TasksSchedulePrefix, "tasks_schedule", collections.TripleKeyCodec(collections.StringKey, sdk.TimeKey, types.TaskIDKey)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.schema = schema
	return k
}

// RegisterTaskHandlers registers the provided task handlers, this must be called once at startup to configure the handlers.
func (k *Keeper) RegisterTaskHandlers(taskHandlers types.TaskHandlers) error {
	typenames := make([]string, 0, len(taskHandlers))
	for _, taskHandler := range taskHandlers {
		typename := taskHandler.Typename()
		if typename == "" {
			return errors.Wrap(types.ErrInvalidTaskHandler, "task handler typename cannot be empty")
		}

		if _, exists := k.handlersByTypename[typename]; exists {
			return errors.Wrapf(types.ErrInvalidTaskHandler, "duplicate task handler: '%s'", typename)
		}

		k.handlersByTypename[typename] = taskHandler
		typenames = append(typenames, typename)
	}

	added := make(map[string]struct{}, len(typenames))
	visited := make(map[string]struct{}, len(typenames))
	var addRec func(string) error
	addRec = func(typename string) error {
		handler := k.handlersByTypename[typename]
		if _, ok := visited[typename]; ok {
			return errors.Wrapf(types.ErrInvalidTaskHandler, "task handler circular dependency over '%s'", handler.Typename())
		}
		if _, ok := added[typename]; ok {
			return nil
		}
		visited[typename] = struct{}{}

		for _, dep := range handler.DependsOn() {
			if _, ok := k.handlersByTypename[dep]; !ok {
				return errors.Wrapf(types.ErrInvalidTaskHandler, "unexisting dependency '%s' on task handler '%s'", dep, handler.Typename())
			}

			if err := addRec(dep); err != nil {
				return err
			}
		}

		delete(visited, typename)
		added[typename] = struct{}{}
		k.handlersOrder = append(k.handlersOrder, typename)
		return nil
	}

	sort.Strings(typenames)
	for _, name := range typenames {
		if err := addRec(name); err != nil {
			return err
		}
	}
	return nil
}

// ScheduleTask schedules a task of the provided type to run at the specified scheduling options.
//
// NOTE: Depending on the provided scheduling options, the task may be created but not scheduled (i.e. paused).
func (k *Keeper) ScheduleTask(ctx context.Context, typename string, id types.TaskID, args proto.Message, scheduleOpts ...types.SchedulingOption) error {
	handler, ok := k.handlersByTypename[typename]
	if !ok {
		return errors.Wrapf(types.ErrInvalidTask, "task '%s' handler not registered: '%s'", id, typename)
	}

	packedArgs, err := handler.PackArgs(args)
	if err != nil {
		return err
	}

	exists, err := k.tasks.Has(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return errors.Wrapf(types.ErrInvalidTask, "task '%s' already exists", id)
	}

	task, err := types.NewTask(ctx, id, typename, packedArgs, scheduleOpts...)
	if err != nil {
		return err
	}

	if err := k.tasks.Set(ctx, id, *task); err != nil {
		return err
	}

	if key := getTaskScheduleKey(*task); key != nil {
		return k.tasksSchedule.Set(ctx, *key)
	}
	return nil
}

// CancelTask removes a task, removing it from the schedule.
func (k *Keeper) CancelTask(ctx context.Context, taskID types.TaskID) error {
	task, err := k.tasks.Get(ctx, taskID)
	if err != nil {
		return err
	}

	if err := k.tasks.Remove(ctx, taskID); err != nil {
		return err
	}

	if key := getTaskScheduleKey(task); key != nil {
		return k.tasksSchedule.Remove(ctx, *key)
	}
	return nil
}

// RescheduleTask reschedules a task applying the provided scheduling options, if none is provided it is a noop.
//
// NOTE: The options are applied on the task with its current scheduling configuration, there's no reset before applying
// the new scheduling.
func (k *Keeper) RescheduleTask(ctx context.Context, id types.TaskID, scheduleOpts ...types.SchedulingOption) error {
	if len(scheduleOpts) == 0 {
		return nil
	}

	task, err := k.tasks.Get(ctx, id)
	if err != nil {
		return err
	}

	if key := getTaskScheduleKey(task); key != nil {
		if err := k.tasksSchedule.Remove(ctx, *key); err != nil {
			return err
		}
	}

	if err := task.ApplySchedulingOpts(ctx, scheduleOpts...); err != nil {
		return err
	}

	if err := k.tasks.Set(ctx, id, task); err != nil {
		return err
	}

	if key := getTaskScheduleKey(task); key != nil {
		return k.tasksSchedule.Set(ctx, *key)
	}
	return nil
}

// GetDueTasksAt retrieves the tasks of the specified type that are due at the provided time.
func (k *Keeper) GetDueTasksAt(
	ctx context.Context,
	typename string,
	at time.Time,
) ([]types.TaskID, error) {
	lb := collections.TriplePrefix[string, time.Time, types.TaskID](typename)
	ub := collections.TripleSuperPrefix[string, time.Time, types.TaskID](typename, at)

	ranger := (&collections.Range[collections.Triple[string, time.Time, types.TaskID]]{}).
		StartInclusive(lb).
		EndInclusive(ub)

	tasks := make([]types.TaskID, 0)
	return tasks, k.tasksSchedule.Walk(ctx, ranger, func(key collections.Triple[string, time.Time, types.TaskID]) (bool, error) {
		tasks = append(tasks, key.K3())
		return false, nil
	})
}

func (k *Keeper) GetTask(ctx context.Context, id types.TaskID) (types.Task, error) {
	return k.tasks.Get(ctx, id)
}

func getTaskScheduleKey(t types.Task) *collections.Triple[string, time.Time, types.TaskID] {
	if t.NextRunAt == nil {
		return nil
	}
	key := collections.Join3(t.Typename, *t.NextRunAt, t.Id)
	return &key
}

// GetDueTasksAtIter retrieves an iterator over the task of the specified type that are due at the provided time.
// TODO: Test that!
func (k *Keeper) GetDueTasksAtIter(
	ctx context.Context,
	typename string,
	at time.Time,
) (collections.KeySetIterator[collections.Triple[string, time.Time, types.TaskID]], error) {
	lb := collections.TriplePrefix[string, time.Time, types.TaskID](typename)
	ub := collections.TripleSuperPrefix[string, time.Time, types.TaskID](typename, at)

	ranger := (&collections.Range[collections.Triple[string, time.Time, types.TaskID]]{}).
		StartInclusive(lb).
		EndInclusive(ub)

	return k.tasksSchedule.Iterate(ctx, ranger)
}

func (k *Keeper) scheduleTask(
	ctx context.Context,
	typename string,
	id types.TaskID,
	args proto.Message,
	startAt time.Time,
	every *time.Duration,
) error {
	handler, ok := k.handlersByTypename[typename]
	if !ok {
		return fmt.Errorf("task type not registered: %s", typename)
	}
	packedArgs, err := handler.PackArgs(args)
	if err != nil {
		return fmt.Errorf("invalid args for task type %s: %w", typename, err)
	}

	exists, err := k.tasks.Has(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("task with ID %s already exists", id)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockTime().After(startAt) {
		return fmt.Errorf("cannot schedule task %s for a time in the past: %s", typename, startAt)
	}

	if err := k.tasks.Set(ctx, id, types.Task{
		Id:        id,
		Typename:  typename,
		Args:      packedArgs,
		NextRunAt: startAt,
		Interval:  every,
		LastRunAt: nil,
		RunCount:  0,
	}); err != nil {
		return err
	}

	return k.tasksSchedule.Set(ctx, collections.Join3(typename, startAt, id))
}
