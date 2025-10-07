package keeper

import (
	"context"
	"time"

	"github.com/allora-network/allora-chain/x/mint/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TaskHandlers returns the task handlers for the mint module
func (k *Keeper) TaskHandlers() schedulertypes.TaskHandlers {
	return schedulertypes.TaskHandlers{
		schedulertypes.NewTaskHandler(
			types.TaskEmissionRecalculation, // Task type name
			nil,                             // Dependencies, if any
			func(ctx context.Context, tasks []schedulertypes.Invocation[*types.EmissionRecalculationTaskArgs]) (map[schedulertypes.TaskID]schedulertypes.ArbitrageDecision, error) {
				return nil, nil // No arbitrage logic needed for now
			},
			func(ctx context.Context, id schedulertypes.TaskID, args *types.EmissionRecalculationTaskArgs, runCount uint64) error {
				return k.executeEmissionRecalculationTask(ctx)
			},
		),
	}
}

// executeEmissionRecalculationTask executes the emission recalculation task
func (k *Keeper) executeEmissionRecalculationTask(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockEmission, err := k.RecalculateEmission(ctx)
	if err != nil {
		return err
	}

	k.Logger(ctx).Info("Emission recalculation task executed",
		"blockHeight", sdkCtx.BlockHeight(),
		"blockEmission", blockEmission.String(),
	)

	return nil
}

// ScheduleEmissionRecalculationTask schedules the emission recalculation task to run roughly every month
func (k *Keeper) ScheduleEmissionRecalculationTask(ctx context.Context, schedulerKeeper types.SchedulerKeeper) error {
	// Approximate one month as 30 days. The exact cadence is recalculated by the
	// mint module based on on-chain parameters when the task runs.
	oneMonthDuration := time.Hour * 24 * 30

	k.Logger(ctx).Info("scheduling emission recalculation task", "interval", oneMonthDuration)

	// Use empty struct since this task doesn't need any arguments
	args := &types.EmissionRecalculationTaskArgs{}

	return schedulerKeeper.ScheduleTask(
		ctx,
		types.TaskEmissionRecalculation,
		schedulertypes.TaskID("emission_recalculation"),
		args,
		schedulertypes.ScheduleIn(oneMonthDuration),
		schedulertypes.ScheduleEvery(&oneMonthDuration),
		schedulertypes.WithRelativeScheduling(),
	)
}
