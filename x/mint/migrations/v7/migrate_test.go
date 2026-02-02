package v7_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"cosmossdk.io/core/header"
	storetypes "cosmossdk.io/store/types"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	v7 "github.com/allora-network/allora-chain/x/mint/migrations/v7"
	minttestutil "github.com/allora-network/allora-chain/x/mint/testutil"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	schedulerkeeper "github.com/allora-network/allora-chain/x/scheduler/keeper"
	schedulertypes "github.com/allora-network/allora-chain/x/scheduler/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"
)

type MintV7MigrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller
}

func TestMintV7MigrationTestSuite(t *testing.T) {
	suite.Run(t, new(MintV7MigrationTestSuite))
}

func (s *MintV7MigrationTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.T().Cleanup(s.ctrl.Finish)
}

func (s *MintV7MigrationTestSuite) setupMigrationEnv(
	blockHeight int64,
	blockTime time.Time,
	blocksPerMonth uint64,
) (sdk.Context, keeper.Keeper, *schedulerkeeper.Keeper) {
	encCfg := moduletestutil.MakeTestEncodingConfig(mint.AppModule{}) // nolint: exhaustruct
	mintKey := storetypes.NewKVStoreKey(minttypes.StoreKey)
	schedulerKey := storetypes.NewKVStoreKey("scheduler")
	mintStoreService := runtime.NewKVStoreService(mintKey)
	schedulerStoreService := runtime.NewKVStoreService(schedulerKey)

	ctx := testutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{
			minttypes.StoreKey: mintKey,
			"scheduler":        schedulerKey,
		},
		map[string]*storetypes.TransientStoreKey{
			"transient_test": storetypes.NewTransientStoreKey("transient_test"),
		},
		nil,
	).WithHeaderInfo(header.Info{Height: blockHeight, Time: blockTime}).
		WithBlockHeight(blockHeight).
		WithBlockTime(blockTime)

	accountKeeper := minttestutil.NewMockAccountKeeper(s.ctrl)
	bankKeeper := minttestutil.NewMockBankKeeper(s.ctrl)
	emissionsKeeper := minttestutil.NewMockEmissionsKeeper(s.ctrl)
	stakingKeeper := minttestutil.NewMockStakingKeeper(s.ctrl)

	accountKeeper.EXPECT().
		GetModuleAddress(minttypes.ModuleName).
		Return(authtypes.NewModuleAddress(minttypes.ModuleName)).
		AnyTimes()
	emissionsKeeper.EXPECT().
		GetParams(gomock.Any()).
		Return(emissionstypes.Params{BlocksPerMonth: blocksPerMonth}, nil).
		AnyTimes()

	schedulerKeeper := schedulerkeeper.NewKeeper(schedulerStoreService, encCfg.Codec)
	mintKeeper := keeper.NewKeeper(
		encCfg.Codec,
		mintStoreService,
		stakingKeeper,
		accountKeeper,
		bankKeeper,
		emissionsKeeper,
		&schedulerKeeper,
		authtypes.FeeCollectorName,
	)

	s.Require().NoError(schedulerKeeper.RegisterTaskHandlers(mintKeeper.TaskHandlers()))

	return ctx, mintKeeper, &schedulerKeeper
}

// expectedAlignedDelay mirrors the migration calculation for deterministic assertions.
func expectedAlignedDelay(blockHeight uint64, blocksPerMonth uint64, oneMonthDuration time.Duration) time.Duration {
	initialDelay := oneMonthDuration
	if blocksPerMonth == 0 {
		return initialDelay
	}

	var blocksElapsed uint64
	if blockHeight > 0 {
		blocksElapsed = (blockHeight - 1) % blocksPerMonth
	}

	blocksRemaining := blocksPerMonth - blocksElapsed
	if blocksRemaining == 0 {
		blocksRemaining = blocksPerMonth
	}

	monthNanoseconds := oneMonthDuration.Nanoseconds()
	perBlockNanoseconds := monthNanoseconds / int64(blocksPerMonth)
	remainingNanoseconds := perBlockNanoseconds * int64(blocksRemaining)
	if remainder := monthNanoseconds % int64(blocksPerMonth); remainder > 0 {
		remainingNanoseconds += remainder * int64(blocksRemaining) / int64(blocksPerMonth)
	}

	initialDelay = time.Duration(remainingNanoseconds)
	if initialDelay <= 0 {
		initialDelay = time.Second
	}

	return initialDelay
}

// Test that the migration schedules a task aligned to the current monthly cadence.
// The task schedule should reflect the remaining blocks in the current month.
func (s *MintV7MigrationTestSuite) TestMigrateStoreSchedulesAlignedTask() {
	now := time.Now().UTC()
	blockHeight := int64(4)
	blocksPerMonth := uint64(10)

	ctx, mintKeeper, schedulerKeeper := s.setupMigrationEnv(blockHeight, now, blocksPerMonth)

	err := v7.MigrateStore(ctx, mintKeeper)
	s.Require().NoError(err)

	task, err := schedulerKeeper.GetTask(ctx, schedulertypes.TaskID(minttypes.TaskEmissionRecalculation))
	s.Require().NoError(err)
	s.Require().NotNil(task.ScheduledFor)
	s.Require().NotNil(task.Interval)

	expectedDelay := expectedAlignedDelay(uint64(blockHeight), blocksPerMonth, 30*24*time.Hour)
	s.Require().WithinDuration(now.Add(expectedDelay), *task.ScheduledFor, time.Second)
	s.Require().Equal(30*24*time.Hour, *task.Interval)
}

// Test that a zero blocks-per-month config falls back to a full month delay.
// This guards the migration against invalid or unset parameter values.
func (s *MintV7MigrationTestSuite) TestMigrateStoreDefaultsToMonthlyDelayWhenBlocksPerMonthZero() {
	now := time.Now().UTC()
	blockHeight := int64(4)
	blocksPerMonth := uint64(0)

	ctx, mintKeeper, schedulerKeeper := s.setupMigrationEnv(blockHeight, now, blocksPerMonth)

	err := v7.MigrateStore(ctx, mintKeeper)
	s.Require().NoError(err)

	task, err := schedulerKeeper.GetTask(ctx, schedulertypes.TaskID(minttypes.TaskEmissionRecalculation))
	s.Require().NoError(err)
	s.Require().NotNil(task.ScheduledFor)
	s.Require().NotNil(task.Interval)

	expectedDelay := 30 * 24 * time.Hour
	s.Require().WithinDuration(now.Add(expectedDelay), *task.ScheduledFor, time.Second)
	s.Require().Equal(expectedDelay, *task.Interval)
}
