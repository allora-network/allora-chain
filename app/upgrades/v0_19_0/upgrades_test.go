package v0_19_0_test //nolint:revive // var-naming: don't use an underscore in package name

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	v0_19_0 "github.com/allora-network/allora-chain/app/upgrades/v0_19_0"
	"github.com/allora-network/allora-chain/x/mint/keeper"
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

	require.NoError(t, v0_19_0.SetInvestorsTeamVestingDelay(testCtx.Ctx, mintKeeper, v0_19_0.InvestorsTeamVestingDelayMonths))

	after, err := mintKeeper.Params.Get(testCtx.Ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(6), after.InvestorsTeamVestingDelayMonths)

	// every other param is untouched
	after.InvestorsTeamVestingDelayMonths = before.InvestorsTeamVestingDelayMonths
	require.Equal(t, before, after)
}
