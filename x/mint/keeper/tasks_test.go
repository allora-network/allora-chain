package keeper_test

import (
	"time"

	"cosmossdk.io/core/header"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	"github.com/allora-network/allora-chain/x/mint/types"
	schedulerkeeper "github.com/allora-network/allora-chain/x/scheduler/keeper"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
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

func (s *MintKeeperTestSuite) mintKeeperWithScheduler(schedulerKeeper *schedulerkeeper.Keeper) keeper.Keeper {
	return keeper.NewKeeper(
		s.mintKeeper.GetBinaryCodec(),
		s.mintKeeper.GetStorageService(),
		s.stakingKeeper,
		s.accountKeeper,
		s.bankKeeper,
		s.emissionsKeeper,
		schedulerKeeper,
		authtypes.FeeCollectorName,
	)
}

func (s *MintKeeperTestSuite) TestScheduleEmissionRecalculationTask_Success() {
	schedulerKeeper, ctx := setupSchedulerKeeper()
	now := ctx.BlockTime()
	mintKeeper := s.mintKeeperWithScheduler(&schedulerKeeper)

	// Register mint task handlers with scheduler
	mintHandlers := mintKeeper.TaskHandlers()
	err := schedulerKeeper.RegisterTaskHandlers(mintHandlers)
	s.Require().NoError(err)

	// Schedule the emission recalculation task
	initialDelay := 10 * time.Minute
	err = mintKeeper.ScheduleEmissionRecalculationTask(ctx, initialDelay)
	s.Require().NoError(err)

	// Verify the task was scheduled
	task, err := schedulerKeeper.GetTask(ctx, schedulertypes.TaskID(types.TaskEmissionRecalculation))
	s.Require().NoError(err)
	s.Require().Equal(types.TaskEmissionRecalculation, task.Typename)
	s.Require().NotNil(task.ScheduledFor)

	// Verify the scheduled time is approximately now + initialDelay
	expectedTime := now.Add(initialDelay)
	s.Require().WithinDuration(expectedTime, *task.ScheduledFor, time.Second)
}

func (s *MintKeeperTestSuite) TestScheduleEmissionRecalculationTask_DuplicateTaskFails() {
	schedulerKeeper, ctx := setupSchedulerKeeper()
	mintKeeper := s.mintKeeperWithScheduler(&schedulerKeeper)

	// Register mint task handlers with scheduler
	mintHandlers := mintKeeper.TaskHandlers()
	err := schedulerKeeper.RegisterTaskHandlers(mintHandlers)
	s.Require().NoError(err)

	// Schedule the emission recalculation task - first time should succeed
	err = mintKeeper.ScheduleEmissionRecalculationTask(ctx, 0)
	s.Require().NoError(err)

	// Schedule again - should fail with ErrTaskAlreadyExists
	err = mintKeeper.ScheduleEmissionRecalculationTask(ctx, 0)
	s.Require().Error(err)
	s.Require().ErrorIs(err, schedulertypes.ErrTaskAlreadyExists)
}

func (s *MintKeeperTestSuite) TestScheduleEmissionRecalculationTask_PeriodicSchedule() {
	schedulerKeeper, ctx := setupSchedulerKeeper()
	mintKeeper := s.mintKeeperWithScheduler(&schedulerKeeper)

	// Register mint task handlers with scheduler
	mintHandlers := mintKeeper.TaskHandlers()
	err := schedulerKeeper.RegisterTaskHandlers(mintHandlers)
	s.Require().NoError(err)

	// Schedule the emission recalculation task
	err = mintKeeper.ScheduleEmissionRecalculationTask(ctx, 0)
	s.Require().NoError(err)

	// Verify the task has periodic scheduling (Interval field should be set)
	task, err := schedulerKeeper.GetTask(ctx, schedulertypes.TaskID(types.TaskEmissionRecalculation))
	s.Require().NoError(err)
	s.Require().NotNil(task.Interval, "Task should have periodic scheduling")

	// Verify the interval is approximately 30 days
	expectedInterval := 30 * 24 * time.Hour
	s.Require().Equal(expectedInterval, *task.Interval)
}
