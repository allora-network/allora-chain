package testutil

import (
	rand "math/rand/v2"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
)

type TestDB struct {
	StoreKeyStr  string
	StoreKey     *storetypes.KVStoreKey
	KeyStr       string
	Key          *storetypes.TransientStoreKey
	StoreService store.KVStoreService
	TestCtx      cosmostestutil.TestContext
	Store        storetypes.KVStore
	SB           *collections.SchemaBuilder
}

func NewTestDB(t *testing.T) TestDB {
	t.Helper()

	var testDB TestDB
	testDB.StoreKeyStr = MustRandomString(rand.IntN(30) + 2) //nolint:gosec
	testDB.StoreKey = storetypes.NewKVStoreKey(testDB.StoreKeyStr)
	testDB.KeyStr = MustRandomString(rand.IntN(30) + 2) //nolint:gosec
	testDB.Key = storetypes.NewTransientStoreKey(testDB.KeyStr)
	testDB.StoreService = runtime.NewKVStoreService(testDB.StoreKey)
	testDB.TestCtx = cosmostestutil.DefaultContextWithDB(t, testDB.StoreKey, testDB.Key)
	testDB.Store = runtime.KVStoreAdapter(testDB.StoreService.OpenKVStore(testDB.TestCtx.Ctx))
	testDB.SB = collections.NewSchemaBuilder(testDB.StoreService)
	return testDB
}
