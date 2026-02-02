package keeper

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"
	"github.com/allora-network/allora-chain/x/mint/types"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TaskHandlers returns the task handlers for the mint module
func (k *Keeper) TaskHandlers() schedulertypes.TaskHandlers {
	return schedulertypes.TaskHandlers{
		schedulertypes.NewNoArgsTaskHandler(
			types.TaskEmissionRecalculation,
			nil,
			nil,
			func(ctx context.Context, _ schedulertypes.Task) error {
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

// EnsureEmissionRecalculationTaskScheduled schedules the emission recalculation task if needed.
func (k *Keeper) EnsureEmissionRecalculationTaskScheduled(ctx context.Context, initialDelay time.Duration) error {
	if err := k.ScheduleEmissionRecalculationTask(ctx, initialDelay); err != nil {
		if errorsmod.IsOf(err, schedulertypes.ErrTaskAlreadyExists) {
			k.Logger(ctx).Info("emission recalculation task already scheduled")
			return nil
		}
		return err
	}

	k.Logger(ctx).Info("scheduled emission recalculation task", "initialDelay", initialDelay)
	return nil
}

// ScheduleEmissionRecalculationTask schedules the emission recalculation task to run roughly every month.
func (k *Keeper) ScheduleEmissionRecalculationTask(
	ctx context.Context,
	initialDelay time.Duration,
) error {
	// Approximate one month as 30 days. The exact cadence is recalculated by the
	// mint module based on on-chain parameters when the task runs.
	oneMonthDuration := time.Hour * 24 * 30

	k.Logger(ctx).Info("scheduling emission recalculation task",
		"interval", oneMonthDuration,
		"initialDelay", initialDelay,
	)

	return k.schedulerKeeper.ScheduleTask(
		ctx,
		types.TaskEmissionRecalculation,
		schedulertypes.TaskID(types.TaskEmissionRecalculation),
		nil,
		schedulertypes.ScheduleIn(initialDelay),
		schedulertypes.ScheduleEvery(&oneMonthDuration),
		schedulertypes.WithRelativeScheduling(),
	)
}
