package keeper

import (
	"context"
	"fmt"
	"sort"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

type Keeper struct {
	storeService store.KVStoreService
	cdc          codec.BinaryCodec
	schema       collections.Schema

	taskTypesByName map[string]types.TaskType
	taskTypesOrder  []string

	tasks         collections.Map[types.TaskID, types.Task]
	tasksByType   collections.KeySet[collections.Pair[string, types.TaskID]]
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

		tasks:         collections.NewMap(sb, types.TasksKeyPrefix, "tasks", types.TaskIDKey, codec.CollValue[types.Task](cdc)),
		tasksByType:   collections.NewKeySet(sb, types.TasksByTypeKeyPrefix, "tasks_by_type", collections.PairKeyCodec(collections.StringKey, types.TaskIDKey)),
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

func (k *Keeper) ScheduleTask(ctx context.Context, typename string, id types.TaskID, args proto.Message, at time.Time) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockTime().After(at) {
		return fmt.Errorf("cannot schedule task %s for a time in the past: %s", typename, at)
	}

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

	var argsAny *codectypes.Any
	if args != nil {
		argsAny, err = codectypes.NewAnyWithValue(args)
		if err != nil {
			return fmt.Errorf("failed to pack args for task %s: %w", typename, err)
		}
	}

	if err := k.tasks.Set(ctx, id, types.Task{
		Id:        id,
		Typename:  typename,
		Args:      argsAny,
		NextRunAt: at,
		RunCount:  0,
	}); err != nil {
		return fmt.Errorf("failed to set task %s: %w", id, err)
	}

	if err := k.tasksByType.Set(ctx, collections.Join(typename, id)); err != nil {
		return fmt.Errorf("failed to add task %s to tasks by type: %w", id, err)
	}

	if err := k.tasksSchedule.Set(ctx, collections.Join3(typename, at, id)); err != nil {
		return fmt.Errorf("failed to add task %s to tasks by type: %w", id, err)
	}

	return nil
}

func (k *Keeper) SchedulePeriodicTask(ctx context.Context, typename string, id types.TaskID, args proto.Message, startAt time.Time, every time.Duration) error {
	// TODO: Implement
	return nil
}

func (k *Keeper) PausePeriodicTask(ctx context.Context, taskID types.TaskID) error {
	// TODO: Implement
	return nil
}

func (k *Keeper) ResumePeriodicTask(ctx context.Context, taskID types.TaskID) error {
	// TODO: Implement
	return nil
}
