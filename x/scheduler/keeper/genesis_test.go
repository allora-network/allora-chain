package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/header"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/x/scheduler/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/stretchr/testify/require"
)

func TestExportImportGenesis(t *testing.T) {
	now := time.Now().UTC()
	storeKey := storetypes.NewKVStoreKey("scheduler")
	storeService := runtime.NewKVStoreService(storeKey)
	encCfg := moduletestutil.MakeTestEncodingConfig()
	k := NewKeeper(storeService, encCfg.Codec)
	err := k.RegisterTaskHandlers(types.TaskHandlers{
		types.NewNoArgsTaskHandler("noargs", nil, nil, nil),
		types.NewTaskHandler[*cosmostypes.Coin]("withargs", nil, nil, nil),
	})
	require.NoError(t, err)

	ctx := testutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{"scheduler": storeKey},
		map[string]*storetypes.TransientStoreKey{"transient_test": storetypes.NewTransientStoreKey("transient_test")},
		nil,
	).WithHeaderInfo(header.Info{Height: 1, Time: now}).
		WithBlockHeight(1).
		WithBlockTime(now)

	err = k.ScheduleTask(ctx, "noargs", "1", nil, types.ScheduleAt(now.Add(10*time.Second)))
	require.NoError(t, err)
	coin := cosmostypes.NewCoin("uallo", math.NewInt(10))
	err = k.ScheduleTask(ctx, "withargs", "2", &coin, types.ScheduleAt(now.Add(10*time.Second)))
	require.NoError(t, err)

	genesis := k.ExportGenesis(ctx)
	require.Len(t, genesis.Tasks, 2)
	require.NoError(t, genesis.Validate())

	// re-init from exported genesis
	storeKey2 := storetypes.NewKVStoreKey("scheduler2")
	storeService2 := runtime.NewKVStoreService(storeKey2)
	k2 := NewKeeper(storeService2, encCfg.Codec)
	err = k2.RegisterTaskHandlers(types.TaskHandlers{
		types.NewNoArgsTaskHandler("noargs", nil, nil, nil),
		types.NewTaskHandler[*cosmostypes.Coin]("withargs", nil, nil, nil),
	})
	require.NoError(t, err)

	ctx = testutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{"scheduler": storeKey2},
		map[string]*storetypes.TransientStoreKey{"transient_test": storetypes.NewTransientStoreKey("transient_test")},
		nil,
	).WithHeaderInfo(header.Info{Height: 1, Time: now}).
		WithBlockHeight(1).
		WithBlockTime(now)

	beforeReimportGenesis := k2.ExportGenesis(ctx)
	require.Len(t, beforeReimportGenesis.Tasks, 0)

	k2.InitGenesis(ctx, genesis)

	it, err := k2.tasks.IterateRaw(ctx, nil, nil, collections.OrderAscending)
	require.NoError(t, err)
	keys, err := it.Keys()
	require.NoError(t, err)
	require.Equal(t, 2, len(keys))

	it2, err := k2.tasksSchedule.IterateRaw(ctx, nil, nil, collections.OrderAscending)
	require.NoError(t, err)
	keys2, err := it2.Keys()
	require.NoError(t, err)
	require.Equal(t, 2, len(keys2))

	genesis2 := k2.ExportGenesis(ctx)
	require.Equal(t, genesis, genesis2)
}
