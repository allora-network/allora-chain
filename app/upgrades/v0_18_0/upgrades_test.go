package v0_18_0_test //nolint:revive // var-naming: don't use an underscore in package name

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	v0_18_0 "github.com/allora-network/allora-chain/app/upgrades/v0_18_0"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	migrationsV6 "github.com/allora-network/allora-chain/x/mint/migrations/v6"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	minttestutil "github.com/allora-network/allora-chain/x/mint/testutil"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
)

func TestSetInvestorsTeamVestingDelay(t *testing.T) {
	encCfg := testutil.MakeTestEncodingConfig(mint.AppModule{}) //nolint:exhaustruct
	key := storetypes.NewKVStoreKey(minttypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := cosmostestutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

	ctrl := gomock.NewController(t)
	accountKeeper := minttestutil.NewMockAccountKeeper(ctrl)
	accountKeeper.EXPECT().GetModuleAddress(minttypes.ModuleName).Return(authtypes.NewModuleAddress(minttypes.ModuleName))
	mintKeeper := keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		minttestutil.NewMockStakingKeeper(ctrl),
		accountKeeper,
		minttestutil.NewMockBankKeeper(ctrl),
		minttestutil.NewMockEmissionsKeeper(ctrl),
		authtypes.FeeCollectorName,
	)

	before := minttypes.DefaultParams()
	before.FEmission = before.FEmission.Add(before.FEmission)
	require.NoError(t, mintKeeper.Params.Set(testCtx.Ctx, before))

	require.NoError(t, v0_18_0.SetInvestorsTeamVestingDelay(testCtx.Ctx, mintKeeper, v0_18_0.InvestorsTeamVestingDelayMonths))

	after, err := mintKeeper.Params.Get(testCtx.Ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(6), after.InvestorsTeamVestingDelayMonths)

	// every other param is untouched
	after.InvestorsTeamVestingDelayMonths = before.InvestorsTeamVestingDelayMonths
	require.Equal(t, before, after)
}

// TestSetInvestorsTeamVestingDelayAfterMigration proves that the delay set
// after RunMigrations survives: the v6→v7 migration writes the delay as 0,
// and the upgrade handler then overwrites it to 6.
func TestSetInvestorsTeamVestingDelayAfterMigration(t *testing.T) {
	encCfg := testutil.MakeTestEncodingConfig(mint.AppModule{}) //nolint:exhaustruct
	key := storetypes.NewKVStoreKey(minttypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := cosmostestutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

	ctrl := gomock.NewController(t)
	accountKeeper := minttestutil.NewMockAccountKeeper(ctrl)
	accountKeeper.EXPECT().GetModuleAddress(minttypes.ModuleName).Return(authtypes.NewModuleAddress(minttypes.ModuleName))
	mintKeeper := keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		minttestutil.NewMockStakingKeeper(ctrl),
		accountKeeper,
		minttestutil.NewMockBankKeeper(ctrl),
		minttestutil.NewMockEmissionsKeeper(ctrl),
		authtypes.FeeCollectorName,
	)
	require.NoError(t, mintKeeper.Params.Set(testCtx.Ctx, minttypes.DefaultParams()))

	// Run the v6→v7 migration, which writes the delay as 0.
	oldParams := minttypes.DefaultParams()
	oldBytes, err := encCfg.Codec.Marshal(&oldParams)
	require.NoError(t, err)
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(testCtx.Ctx))
	store.Set(minttypes.ParamsKey, oldBytes)
	require.NoError(t, migrationsV6.MigrateStore(testCtx.Ctx, mintKeeper))

	afterMigration, err := mintKeeper.Params.Get(testCtx.Ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(0), afterMigration.InvestorsTeamVestingDelayMonths)

	// The upgrade handler then sets the delay.
	require.NoError(t, v0_18_0.SetInvestorsTeamVestingDelay(testCtx.Ctx, mintKeeper, v0_18_0.InvestorsTeamVestingDelayMonths))

	afterUpgrade, err := mintKeeper.Params.Get(testCtx.Ctx)
	require.NoError(t, err)
	require.Equal(t, v0_18_0.InvestorsTeamVestingDelayMonths, afterUpgrade.InvestorsTeamVestingDelayMonths)
}
