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

	moduleParams, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	// if emissions are not enabled, do nothing
	if !moduleParams.EmissionEnabled {
		return nil
	}

	// Get the balance of the "ecosystem" module account
	ecosystemBalance, err := k.GetEcosystemBalance(ctx, moduleParams.MintDenom)
	if err != nil {
		return err
	}

	blockHeight := uint64(sdkCtx.BlockHeight())

	ecosystemMintSupplyRemaining, err := k.GetEcosystemMintSupplyRemaining(ctx, moduleParams)
	if err != nil {
		return err
	}

	blocksPerMonth, err := k.GetParamsBlocksPerMonth(ctx)
	if err != nil {
		return err
	}

	vPercentADec, err := k.GetValidatorsVsAlloraPercentReward(ctx)
	if err != nil {
		return err
	}

	vPercent, err := vPercentADec.SdkLegacyDec()
	if err != nil {
		return err
	}

	// Always recalculate the target emission when the task runs (monthly via scheduler)
	// WARNING: After Calling RecalculateTargetEmission,
	// PreviousRewardEmissionPerUnitStakedToken and PreviousBlockEmission
	// are set to new values. If later in begin blocker you need to use these
	// you should get them first before this function is called!!
	currentBlockEmission, _, err := RecalculateTargetEmission(
		sdkCtx,
		*k,
		blockHeight,
		blocksPerMonth,
		moduleParams,
		ecosystemBalance,
		ecosystemMintSupplyRemaining,
		vPercent,
	)
	if err != nil {
		return err
	}

	k.Logger(ctx).Info("Emission recalculation task executed",
		"blockHeight", blockHeight,
		"blockEmission", currentBlockEmission.String(),
		"recalculated", true,
	)

	return nil
}

// ScheduleEmissionRecalculationTask schedules the emission recalculation task to run every 6 months
func (k *Keeper) ScheduleEmissionRecalculationTask(ctx context.Context, schedulerKeeper types.SchedulerKeeper) error {
	// Calculate 6 months duration (approximately 182.5 days * 24 hours * 60 minutes * 60 seconds)
	sixMonthDuration := time.Hour * 24 * 182 // 182 days ≈ 6 months

	k.Logger(ctx).Info("scheduling emission recalculation task", "interval", sixMonthDuration)

	// Use empty struct since this task doesn't need any arguments
	args := &types.EmissionRecalculationTaskArgs{}

	return schedulerKeeper.ScheduleTask(
		ctx,
		types.TaskEmissionRecalculation,
		schedulertypes.TaskID("emission_recalculation"),
		args,
		schedulertypes.ScheduleIn(sixMonthDuration),
		schedulertypes.ScheduleEvery(&sixMonthDuration),
		schedulertypes.WithRelativeScheduling(),
	)
}
