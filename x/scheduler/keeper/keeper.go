package keeper

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/core/store"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

type Keeper struct {
	storeService store.KVStoreService
	cdc          codec.BinaryCodec

	taskSpecsByName map[string]types.TaskSpec
	taskSpecsOrder  []string
}

// NewKeeper returns a new keeper by codec and storeKey inputs.
func NewKeeper(storeService store.KVStoreService, cdc codec.BinaryCodec) Keeper {
	k := Keeper{
		storeService:    storeService,
		cdc:             cdc,
		taskSpecsByName: make(map[string]types.TaskSpec),
		taskSpecsOrder:  nil,
	}

	return k
}

func (k *Keeper) RegisterTaskSpec(spec types.TaskSpec) error {
	if _, ok := k.taskSpecsByName[spec.Name]; ok {
		return fmt.Errorf("task type already registered: %s", spec.Name)
	}

	k.taskSpecsByName[spec.Name] = spec

	for i, name := range k.taskSpecsOrder {
		if spec.Priority < k.taskSpecsByName[name].Priority {
			k.taskSpecsOrder = append(k.taskSpecsOrder[:i], append([]string{spec.Name}, k.taskSpecsOrder[i:]...)...)
			return nil
		}
	}

	k.taskSpecsOrder = append(k.taskSpecsOrder, spec.Name)
	return nil
}

func (k *Keeper) ScheduleTask(ctx context.Context, typename string, id types.TaskID, args proto.Message, at time.Time) error {
	spec, ok := k.taskSpecsByName[typename]
	if !ok {
		return fmt.Errorf("task type not registered: %s", typename)
	}

	if err := spec.ValidateArgs(args); err != nil {
		return fmt.Errorf("invalid args for task type %s: %w", typename, err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockTime().After(at) {
		return fmt.Errorf("cannot schedule task %s for a time in the past: %s", typename, at)
	}

	// TODO: Implement

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
