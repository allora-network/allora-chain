package app_test

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	dbm "github.com/cosmos/cosmos-db"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	"github.com/stretchr/testify/require"

	"github.com/allora-network/allora-chain/app"
	v0_18_0 "github.com/allora-network/allora-chain/app/upgrades/v0_18_0"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
)

func init() {
	app.UseFeeMarketDecorator = false
}

func initTestApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	a, err := app.NewAlloraApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.EmptyAppOptions{})
	if err != nil {
		panic(err)
	}
	return a, a.DefaultGenesis()
}

// TestUpgradeHandlerSetsInvestorsTeamVestingDelay builds the real app,
// seeds mint params as consensus-version-6 state, and runs the v0.18.0 upgrade handler.
func TestUpgradeHandlerSetsInvestorsTeamVestingDelay(t *testing.T) {
	ibctesting.DefaultTestingAppInit = initTestApp
	coordinator := ibctesting.NewCoordinator(t, 1)
	chain, ok := coordinator.Chains[ibctesting.GetChainID(1)]
	require.True(t, ok)

	a, ok := chain.App.(*app.AlloraApp)
	require.True(t, ok)

	ctx := chain.GetContext()

	// Seed mint params as v6: delay is 0.
	paramsBefore, err := a.MintKeeper.GetParams(ctx)
	require.NoError(t, err)
	paramsBefore.InvestorsTeamVestingDelayMonths = 0
	require.NoError(t, a.MintKeeper.Params.Set(ctx, paramsBefore))

	// Set mint version to 6.
	vm := a.ModuleManager.GetVersionMap()
	vm[minttypes.ModuleName] = 6
	require.NoError(t, a.UpgradeKeeper.SetModuleVersionMap(ctx, vm))

	// Run the upgrade handler.
	handler := v0_18_0.CreateUpgradeHandler(a.ModuleManager, a.Configurator(), &a.AppKeepers)
	upgradeVM, err := handler(ctx, upgradetypes.Plan{Name: "v0.18.0"}, vm) //nolint:exhaustruct
	require.NoError(t, err)

	// Mint version is now 7.
	require.Equal(t, uint64(7), upgradeVM[minttypes.ModuleName])

	// Delay is 6.
	paramsAfter, err := a.MintKeeper.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, v0_18_0.InvestorsTeamVestingDelayMonths, paramsAfter.InvestorsTeamVestingDelayMonths)

	// Every other param is unchanged.
	paramsAfter.InvestorsTeamVestingDelayMonths = paramsBefore.InvestorsTeamVestingDelayMonths
	require.Equal(t, paramsBefore, paramsAfter)

	// EventParamsSet was emitted.
	var found bool
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == "mint.v3.EventParamsSet" {
			found = true
			break
		}
	}
	require.True(t, found, "expected EventParamsSet event")
}
