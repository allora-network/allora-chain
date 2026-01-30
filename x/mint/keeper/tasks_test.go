package keeper_test

import (
	"time"

	"cosmossdk.io/core/header"
	storetypes "cosmossdk.io/store/types"
	schedulerkeeper "github.com/allora-network/allora-chain/x/scheduler/keeper"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
	"github.com/allora-network/allora-chain/x/mint/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
)

// setupSchedulerKeeper creates a scheduler keeper for testing task scheduling.
// Returns the keeper and a context configured for the scheduler's store.
func setupSchedulerKeeper() (schedulerkeeper.Keeper, sdk.Context) {
	now := time.Now().UTC()
	storeKey := storetypes.NewKVStoreKey("scheduler")
	storeService := runtime.NewKVStoreService(storeKey)
	encCfg := moduletestutil.MakeTestEncodingConfig()
	keeper := schedulerkeeper.NewKeeper(storeService, encCfg.Codec)

	ctx := testutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{"scheduler": storeKey},
		map[string]*storetypes.TransientStoreKey{"transient_test": storetypes.NewTransientStoreKey("transient_test")},
		nil,
	).WithHeaderInfo(header.Info{Height: 1, Time: now}).
		WithBlockHeight(1).
		WithBlockTime(now)

	return keeper, ctx
}

func (s *MintKeeperTestSuite) TestScheduleEmissionRecalculationTask_Success() {
	schedulerKeeper, ctx := setupSchedulerKeeper()
	now := ctx.BlockTime()

	// Register mint task handlers with scheduler
	mintHandlers := s.mintKeeper.TaskHandlers()
	err := schedulerKeeper.RegisterTaskHandlers(mintHandlers)
	s.Require().NoError(err)

	// Schedule the emission recalculation task
	initialDelay := 10 * time.Minute
	err = s.mintKeeper.ScheduleEmissionRecalculationTask(ctx, &schedulerKeeper, initialDelay)
	s.Require().NoError(err)

	// Verify the task was scheduled
	task, err := schedulerKeeper.GetTask(ctx, schedulertypes.TaskID("emission_recalculation"))
	s.Require().NoError(err)
	s.Require().Equal(types.TaskEmissionRecalculation, task.Typename)
	s.Require().NotNil(task.ScheduledFor)

	// Verify the scheduled time is approximately now + initialDelay
	expectedTime := now.Add(initialDelay)
	s.Require().WithinDuration(expectedTime, *task.ScheduledFor, time.Second)
}

func (s *MintKeeperTestSuite) TestScheduleEmissionRecalculationTask_DuplicateTaskFails() {
	schedulerKeeper, ctx := setupSchedulerKeeper()

	// Register mint task handlers with scheduler
	mintHandlers := s.mintKeeper.TaskHandlers()
	err := schedulerKeeper.RegisterTaskHandlers(mintHandlers)
	s.Require().NoError(err)

	// Schedule the emission recalculation task - first time should succeed
	err = s.mintKeeper.ScheduleEmissionRecalculationTask(ctx, &schedulerKeeper, 0)
	s.Require().NoError(err)

	// Schedule again - should fail with ErrTaskAlreadyExists
	err = s.mintKeeper.ScheduleEmissionRecalculationTask(ctx, &schedulerKeeper, 0)
	s.Require().Error(err)
	s.Require().ErrorIs(err, schedulertypes.ErrTaskAlreadyExists)
}

func (s *MintKeeperTestSuite) TestScheduleEmissionRecalculationTask_PeriodicSchedule() {
	schedulerKeeper, ctx := setupSchedulerKeeper()

	// Register mint task handlers with scheduler
	mintHandlers := s.mintKeeper.TaskHandlers()
	err := schedulerKeeper.RegisterTaskHandlers(mintHandlers)
	s.Require().NoError(err)

	// Schedule the emission recalculation task
	err = s.mintKeeper.ScheduleEmissionRecalculationTask(ctx, &schedulerKeeper, 0)
	s.Require().NoError(err)

	// Verify the task has periodic scheduling (Interval field should be set)
	task, err := schedulerKeeper.GetTask(ctx, schedulertypes.TaskID("emission_recalculation"))
	s.Require().NoError(err)
	s.Require().NotNil(task.Interval, "Task should have periodic scheduling")

	// Verify the interval is approximately 30 days
	expectedInterval := 30 * 24 * time.Hour
	s.Require().Equal(expectedInterval, *task.Interval)
}
