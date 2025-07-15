package keeper

import (
	"context"
	"fmt"
	"sort"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/core/store"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
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
	cdc          codec.BinaryCodec
	schema       collections.Schema

	taskTypesByName map[string]types.TaskType
	taskTypesOrder  []string

	tasks         *collections.IndexedMap[types.TaskID, types.Task, TasksIndexes]
	tasksSchedule collections.KeySet[collections.Triple[string, time.Time, types.TaskID]]
}

// NewKeeper returns a new keeper by codec and storeKey inputs.
func NewKeeper(storeService store.KVStoreService, cdc codec.BinaryCodec) Keeper {
	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,

		taskTypesByName: make(map[string]types.TaskType),
		taskTypesOrder:  nil,

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

// RegisterTaskTypes registers the provided task types, this must be called once at startup to configure the handlers.
func (k *Keeper) RegisterTaskTypes(taskTypes types.TaskTypes) error {
	taskNames := make([]string, 0, len(taskTypes))
	for _, taskType := range taskTypes {
		if taskType.Name == "" {
			return fmt.Errorf("task spec name cannot be empty")
		}

		if _, exists := k.taskTypesByName[taskType.Name]; exists {
			return fmt.Errorf("duplicated task spec: '%s'", taskType.Name)
		}

		if taskType.TaskHandler == nil {
			return fmt.Errorf("task spec '%s' has no task handler defined", taskType.Name)
		}

		k.taskTypesByName[taskType.Name] = taskType
		taskNames = append(taskNames, taskType.Name)
	}

	added := make(map[string]struct{}, len(taskNames))
	visited := make(map[string]struct{}, len(taskNames))
	var addRec func(string) error
	addRec = func(name string) error {
		spec := k.taskTypesByName[name]
		if _, ok := visited[name]; ok {
			return fmt.Errorf("task spec circular dependency over %s", spec.Name)
		}
		if _, ok := added[name]; ok {
			return nil
		}
		visited[name] = struct{}{}

		for _, dep := range spec.DependsOn {
			if _, ok := k.taskTypesByName[dep]; !ok {
				return fmt.Errorf("unexisting dependency '%s' on task spec '%s'", dep, spec.Name)
			}

			if err := addRec(dep); err != nil {
				return err
			}
		}

		delete(visited, name)
		added[name] = struct{}{}
		k.taskTypesOrder = append(k.taskTypesOrder, name)
		return nil
	}

	sort.Strings(taskNames)
	for _, name := range taskNames {
		if err := addRec(name); err != nil {
			return err
		}
	}
	return nil
}

// ScheduleTask schedules a task of the provided type to run at the specified time.
func (k *Keeper) ScheduleTask(ctx context.Context, typename string, id types.TaskID, args proto.Message, at time.Time) error {
	return k.scheduleTask(ctx, typename, id, args, at, nil)
}

// SchedulePeriodicTask schedules a task of the provided type to run every given interval, starting at the specified time.
func (k *Keeper) SchedulePeriodicTask(ctx context.Context, typename string, id types.TaskID, args proto.Message, startAt time.Time, every time.Duration) error {
	return k.scheduleTask(ctx, typename, id, args, startAt, &every)
}

// PausePeriodicTask pauses a periodic task, preventing it from running until resumed.
func (k *Keeper) PausePeriodicTask(ctx context.Context, taskID types.TaskID) error {
	// TODO: Implement
	return nil
}

// ResumePeriodicTask resumes a paused periodic task, allowing it to run again.
func (k *Keeper) ResumePeriodicTask(ctx context.Context, taskID types.TaskID) error {
	// TODO: Implement
	return nil
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
	taskType, ok := k.taskTypesByName[typename]
	if !ok {
		return fmt.Errorf("task type not registered: %s", typename)
	}
	if err := taskType.ValidateArgs(args); err != nil {
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

	var argsAny *codectypes.Any
	if args != nil {
		argsAny, err = codectypes.NewAnyWithValue(args)
		if err != nil {
			return err
		}
	}

	if err := k.tasks.Set(ctx, id, types.Task{
		Id:        id,
		Typename:  typename,
		Args:      argsAny,
		NextRunAt: startAt,
		Interval:  every,
		RunCount:  0,
	}); err != nil {
		return err
	}

	return k.tasksSchedule.Set(ctx, collections.Join3(typename, startAt, id))
}
