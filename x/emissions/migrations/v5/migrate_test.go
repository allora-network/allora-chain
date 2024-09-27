package v5_test

import (
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"

	collections "cosmossdk.io/collections"
	"github.com/cosmos/cosmos-sdk/codec"
	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"

	"cosmossdk.io/core/store"
	"github.com/allora-network/allora-chain/app/params"

	"cosmossdk.io/store/prefix"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	v5 "github.com/allora-network/allora-chain/x/emissions/migrations/v5"
	emissions "github.com/allora-network/allora-chain/x/emissions/module"
	emissionstestutil "github.com/allora-network/allora-chain/x/emissions/testutil"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	storetypes "cosmossdk.io/store/types"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
)

type EmissionsV5MigrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	ctx             sdk.Context
	storeService    store.KVStoreService
	emissionsKeeper *keeper.Keeper
}

func TestEmissionsV5MigrationTestSuite(t *testing.T) {
	suite.Run(t, new(EmissionsV5MigrationTestSuite))
}

func (s *EmissionsV5MigrationTestSuite) SetupTest() {
	encCfg := moduletestutil.MakeTestEncodingConfig(emissions.AppModule{})
	key := storetypes.NewKVStoreKey(emissionstypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	s.storeService = storeService
	testCtx := cosmostestutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	s.ctx = testCtx.Ctx

	// gomock initializations
	s.ctrl = gomock.NewController(s.T())
	accountKeeper := emissionstestutil.NewMockAccountKeeper(s.ctrl)
	bankKeeper := emissionstestutil.NewMockBankKeeper(s.ctrl)
	emissionsKeeper := keeper.NewKeeper(
		encCfg.Codec,
		codecAddress.NewBech32Codec(params.Bech32PrefixAccAddr),
		storeService,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName)

	s.emissionsKeeper = &emissionsKeeper
}

// in this test, we check that an already migrated topic, that has all the new fields
// for merit sortition, but has a NaN for initial regret, will have its initial regret
// set to 0 but everything else will remain the same
func (s *EmissionsV5MigrationTestSuite) TestMigratedTopicWithNaNInitialRegret() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	migratedOldTopicWithNaNInitialRegret := emissionstypes.Topic{
		Id:                       1,
		Creator:                  "creator",
		Metadata:                 "metadata",
		LossMethod:               "lossMethod",
		EpochLastEnded:           0,
		EpochLength:              100,
		GroundTruthLag:           10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		InitialRegret:            alloraMath.NewNaN(), // OBSERVE: NAN FOR INITIAL REGRET
		WorkerSubmissionWindow:   120,
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.1337"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.1337"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.1337"),
	}

	bz, err := proto.Marshal(&migratedOldTopicWithNaNInitialRegret)
	s.Require().NoError(err)

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	bytesKey := make([]byte, collections.Uint64Key.Size(migratedOldTopicWithNaNInitialRegret.Id))
	countWritten, err := collections.Uint64Key.Encode(bytesKey, migratedOldTopicWithNaNInitialRegret.Id)
	s.Require().NoError(err)
	s.Require().NotEqual(0, countWritten)
	topicStore.Set(bytesKey, bz)
	err = v5.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator := topicStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	defer iterator.Close()

	var newMsg emissionstypes.Topic
	err = proto.Unmarshal(iterator.Value(), &newMsg)
	s.Require().NoError(err)

	// correct props are the same
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.Id, newMsg.Id)
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.Creator, newMsg.Creator)
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.Metadata, newMsg.Metadata)
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.LossMethod, newMsg.LossMethod)
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.EpochLastEnded, newMsg.EpochLastEnded)
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.EpochLength, newMsg.EpochLength)
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.GroundTruthLag, newMsg.GroundTruthLag)
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.PNorm.String(), newMsg.PNorm.String())
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.AlphaRegret.String(), newMsg.AlphaRegret.String())
	s.Require().False(newMsg.AllowNegative)
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.Epsilon.String(), newMsg.Epsilon.String())
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.WorkerSubmissionWindow, newMsg.WorkerSubmissionWindow)
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.MeritSortitionAlpha.String(), newMsg.MeritSortitionAlpha.String())
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.ActiveInfererQuantile.String(), newMsg.ActiveInfererQuantile.String())
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.ActiveForecasterQuantile.String(), newMsg.ActiveForecasterQuantile.String())
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.ActiveReputerQuantile.String(), newMsg.ActiveReputerQuantile.String())
	// InitialRegret is reset to 0
	s.Require().Equal("0", newMsg.InitialRegret.String())

	// sanity check that the emissions keeper collections.go API also gets the same data
	topic, err := s.emissionsKeeper.GetTopic(s.ctx, 1)
	s.Require().NoError(err)
	s.Require().Equal(newMsg, topic)
}

// check that the specified maps are reset correctly
func (s *EmissionsV5MigrationTestSuite) TestResetMapsWithNonNumericValues() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()
	testPreviousQuantileMapDeletion(s, store, cdc, emissionstypes.PreviousTopicQuantileInfererScoreEmaKey)
}

/// HELPER FUNCTIONS

// test for deletes on maps that have previous quantile as the value of the map
func testPreviousQuantileMapDeletion(
	s *EmissionsV5MigrationTestSuite,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	key collections.Prefix,
) {
	infererQuantile, err := alloraMath.NewDecFromString("0.1")
	forecasterQuantile, err := alloraMath.NewDecFromString("0.2")
	reputerQuantile, err := alloraMath.NewDecFromString("0.3")

	bz, err := infererQuantile.Marshal()
	s.Require().NoError(err)
	mapInfererStore := prefix.NewStore(store, key)
	mapInfererStore.Set([]byte("testKey1"), bz)

	bz, err = forecasterQuantile.Marshal()
	s.Require().NoError(err)
	mapForecasterStore := prefix.NewStore(store, key)
	mapForecasterStore.Set([]byte("testKey2"), bz)

	bz, err = reputerQuantile.Marshal()
	s.Require().NoError(err)
	mapReputerStore := prefix.NewStore(store, key)
	mapReputerStore.Set([]byte("testKey3"), bz)
	// Sanity check
	iterator := mapInfererStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	iterator = mapForecasterStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	iterator = mapReputerStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	defer iterator.Close()

	v5.ResetMapsWithNonNumericValues(s.ctx, store, cdc)

	// Verify the inferer store has been updated correctly
	iterator = mapInfererStore.Iterator(nil, nil)
	s.Require().False(iterator.Valid(), "iterator should be invalid because the store should be empty")

	// Verify the forecaster store has been updated correctly
	iterator = mapForecasterStore.Iterator(nil, nil)
	s.Require().False(iterator.Valid(), "iterator should be invalid because the store should be empty")

	// Verify the reputer store has been updated correctly
	iterator = mapReputerStore.Iterator(nil, nil)
	s.Require().False(iterator.Valid(), "iterator should be invalid because the store should be empty")
}
