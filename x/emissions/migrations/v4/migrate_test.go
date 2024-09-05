package v4_test

import (
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"

	collections "cosmossdk.io/collections"
	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"

	"cosmossdk.io/core/store"
	"github.com/allora-network/allora-chain/app/params"

	"cosmossdk.io/store/prefix"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	oldtypes "github.com/allora-network/allora-chain/x/emissions/migrations/v3/types"
	v4 "github.com/allora-network/allora-chain/x/emissions/migrations/v4"
	emissions "github.com/allora-network/allora-chain/x/emissions/module"
	emissionstestutil "github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
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

type EmissionsV4MigrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	ctx             sdk.Context
	storeService    store.KVStoreService
	emissionsKeeper *keeper.Keeper
}

func TestEmissionsV4MigrationTestSuite(t *testing.T) {
	suite.Run(t, new(EmissionsV4MigrationTestSuite))
}

func (s *EmissionsV4MigrationTestSuite) SetupTest() {
	encCfg := moduletestutil.MakeTestEncodingConfig(emissions.AppModule{})
	key := storetypes.NewKVStoreKey(types.StoreKey)
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
		authtypes.FeeCollectorName,
		keeper.DefaultConfig(),
	)

	s.emissionsKeeper = &emissionsKeeper
}

// in this test, we check that an already migrated topic, that has all the new fields
// for merit sortition, and everything looks correct, will not change at all
func (s *EmissionsV4MigrationTestSuite) TestMigratedTopicWithNoProblems() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	migratedOldTopic := types.Topic{
		Id:                       1,
		Creator:                  "creator",
		Metadata:                 "metadata",
		LossMethod:               "lossMethod",
		EpochLastEnded:           0,
		EpochLength:              100,
		GroundTruthLag:           10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            false,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		InitialRegret:            alloraMath.MustNewDecFromString("11"),
		WorkerSubmissionWindow:   120,
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.1337"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.1337"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.1337"),
	}

	bz, err := proto.Marshal(&migratedOldTopic)
	s.Require().NoError(err)

	topicStore := prefix.NewStore(store, types.TopicsKey)
	bytesKey := make([]byte, collections.Uint64Key.Size(migratedOldTopic.Id))
	countWritten, err := collections.Uint64Key.Encode(bytesKey, migratedOldTopic.Id)
	s.Require().NoError(err)
	s.Require().NotEqual(countWritten, 0)
	topicStore.Set(bytesKey, bz)

	err = v4.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator := topicStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	defer iterator.Close()

	var newMsg types.Topic
	err = proto.Unmarshal(iterator.Value(), &newMsg)
	s.Require().NoError(err)

	// correct props are the same
	s.Require().Equal(migratedOldTopic.Id, newMsg.Id)
	s.Require().Equal(migratedOldTopic.Creator, newMsg.Creator)
	s.Require().Equal(migratedOldTopic.Metadata, newMsg.Metadata)
	s.Require().Equal(migratedOldTopic.LossMethod, newMsg.LossMethod)
	s.Require().Equal(migratedOldTopic.EpochLastEnded, newMsg.EpochLastEnded)
	s.Require().Equal(migratedOldTopic.EpochLength, newMsg.EpochLength)
	s.Require().Equal(migratedOldTopic.GroundTruthLag, newMsg.GroundTruthLag)
	s.Require().Equal(migratedOldTopic.PNorm.String(), newMsg.PNorm.String())
	s.Require().Equal(migratedOldTopic.AlphaRegret.String(), newMsg.AlphaRegret.String())
	s.Require().Equal(migratedOldTopic.AllowNegative, newMsg.AllowNegative)
	s.Require().Equal(migratedOldTopic.Epsilon.String(), newMsg.Epsilon.String())
	s.Require().Equal(migratedOldTopic.InitialRegret.String(), newMsg.InitialRegret.String())
	s.Require().Equal(migratedOldTopic.WorkerSubmissionWindow, newMsg.WorkerSubmissionWindow)
	s.Require().Equal(migratedOldTopic.MeritSortitionAlpha.String(), newMsg.MeritSortitionAlpha.String())
	s.Require().Equal(migratedOldTopic.ActiveInfererQuantile.String(), newMsg.ActiveInfererQuantile.String())
	s.Require().Equal(migratedOldTopic.ActiveForecasterQuantile.String(), newMsg.ActiveForecasterQuantile.String())
	s.Require().Equal(migratedOldTopic.ActiveReputerQuantile.String(), newMsg.ActiveReputerQuantile.String())

	// sanity check that the emissions keeper collections.go API also gets the same data
	topic, err := s.emissionsKeeper.GetTopic(s.ctx, 1)
	s.Require().NoError(err)
	s.Require().Equal(newMsg, topic)
}

// in this test, we check that an already migrated topic, that has all the new fields
// for merit sortition, but has a NaN for initial regret, will have its initial regret
// set to 0 but everything else will remain the same
func (s *EmissionsV4MigrationTestSuite) TestMigratedTopicWithNaNInitialRegret() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	migratedOldTopicWithNaNInitialRegret := types.Topic{
		Id:                       1,
		Creator:                  "creator",
		Metadata:                 "metadata",
		LossMethod:               "lossMethod",
		EpochLastEnded:           0,
		EpochLength:              100,
		GroundTruthLag:           10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            false,
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

	topicStore := prefix.NewStore(store, types.TopicsKey)
	bytesKey := make([]byte, collections.Uint64Key.Size(migratedOldTopicWithNaNInitialRegret.Id))
	countWritten, err := collections.Uint64Key.Encode(bytesKey, migratedOldTopicWithNaNInitialRegret.Id)
	s.Require().NoError(err)
	s.Require().NotEqual(countWritten, 0)
	topicStore.Set(bytesKey, bz)
	err = v4.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator := topicStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	defer iterator.Close()

	var newMsg types.Topic
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
	s.Require().Equal(migratedOldTopicWithNaNInitialRegret.AllowNegative, newMsg.AllowNegative)
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

// in this test, we check that a not migrated topic, that does not have any of the
// new fields for merit sortition, will have all the new fields set to the default values
// and everything else will remain the same
func (s *EmissionsV4MigrationTestSuite) TestNotMigratedTopic() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	notMigratedTopic := oldtypes.Topic{
		Id:                     1,
		Creator:                "creator",
		Metadata:               "metadata",
		LossMethod:             "lossMethod",
		EpochLastEnded:         80,
		EpochLength:            100,
		GroundTruthLag:         10,
		PNorm:                  alloraMath.NewDecFromInt64(3),
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:          false,
		Epsilon:                alloraMath.MustNewDecFromString("0.01"),
		InitialRegret:          alloraMath.MustNewDecFromString("11"),
		WorkerSubmissionWindow: 120,
	}

	bz, err := proto.Marshal(&notMigratedTopic)
	s.Require().NoError(err)

	topicStore := prefix.NewStore(store, types.TopicsKey)
	bytesKey := make([]byte, collections.Uint64Key.Size(notMigratedTopic.Id))
	countWritten, err := collections.Uint64Key.Encode(bytesKey, notMigratedTopic.Id)
	s.Require().NoError(err)
	s.Require().NotEqual(countWritten, 0)
	topicStore.Set(bytesKey, bz)
	err = v4.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	err = v4.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator := topicStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	defer iterator.Close()

	var newMsg types.Topic
	err = proto.Unmarshal(iterator.Value(), &newMsg)
	s.Require().NoError(err)

	// Old props are the same
	s.Require().Equal(notMigratedTopic.Id, newMsg.Id)
	s.Require().Equal(notMigratedTopic.Creator, newMsg.Creator)
	s.Require().Equal(notMigratedTopic.Metadata, newMsg.Metadata)
	s.Require().Equal(notMigratedTopic.LossMethod, newMsg.LossMethod)
	s.Require().Equal(notMigratedTopic.EpochLastEnded, newMsg.EpochLastEnded)
	s.Require().Equal(notMigratedTopic.EpochLength, newMsg.EpochLength)
	s.Require().Equal(notMigratedTopic.GroundTruthLag, newMsg.GroundTruthLag)
	s.Require().Equal(notMigratedTopic.PNorm.String(), newMsg.PNorm.String())
	s.Require().Equal(notMigratedTopic.AlphaRegret.String(), newMsg.AlphaRegret.String())
	s.Require().Equal(notMigratedTopic.AllowNegative, newMsg.AllowNegative)
	s.Require().Equal(notMigratedTopic.Epsilon.String(), newMsg.Epsilon.String())
	s.Require().Equal(notMigratedTopic.InitialRegret.String(), newMsg.InitialRegret.String())
	s.Require().Equal(notMigratedTopic.WorkerSubmissionWindow, newMsg.WorkerSubmissionWindow)
	// New props are imputed with defaults
	s.Require().Equal("0.1", newMsg.MeritSortitionAlpha.String())
	s.Require().Equal("0.25", newMsg.ActiveInfererQuantile.String())
	s.Require().Equal("0.25", newMsg.ActiveForecasterQuantile.String())
	s.Require().Equal("0.25", newMsg.ActiveReputerQuantile.String())

	// sanity check that the emissions keeper collections.go API also gets the same data
	topic, err := s.emissionsKeeper.GetTopic(s.ctx, 1)
	s.Require().NoError(err)
	s.Require().Equal(newMsg, topic)
}

// in this test, we check that a not migrated topic, that does not have any of the
// new fields for merit sortition, and also  has a NaN for initial regret,
// will have its initial regret set to 0 and all the new fields set to the default values
func (s *EmissionsV4MigrationTestSuite) TestNotMigratedTopicWithNaNInitialRegret() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	notMigratedTopicWithNaNInitialRegret := oldtypes.Topic{
		Id:                     1,
		Creator:                "creator",
		Metadata:               "metadata",
		LossMethod:             "lossMethod",
		EpochLastEnded:         80,
		EpochLength:            100,
		GroundTruthLag:         10,
		PNorm:                  alloraMath.NewDecFromInt64(3),
		AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:          false,
		Epsilon:                alloraMath.MustNewDecFromString("0.01"),
		InitialRegret:          alloraMath.NewNaN(),
		WorkerSubmissionWindow: 120,
	}

	bz, err := proto.Marshal(&notMigratedTopicWithNaNInitialRegret)
	s.Require().NoError(err)

	topicStore := prefix.NewStore(store, types.TopicsKey)
	bytesKey := make([]byte, collections.Uint64Key.Size(notMigratedTopicWithNaNInitialRegret.Id))
	countWritten, err := collections.Uint64Key.Encode(bytesKey, notMigratedTopicWithNaNInitialRegret.Id)
	s.Require().NoError(err)
	s.Require().NotEqual(countWritten, 0)
	topicStore.Set(bytesKey, bz)
	err = v4.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	err = v4.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	err = v4.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator := topicStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	defer iterator.Close()

	var newMsg types.Topic
	err = proto.Unmarshal(iterator.Value(), &newMsg)
	s.Require().NoError(err)

	// Old props are the same
	s.Require().Equal(notMigratedTopicWithNaNInitialRegret.Id, newMsg.Id)
	s.Require().Equal(notMigratedTopicWithNaNInitialRegret.Creator, newMsg.Creator)
	s.Require().Equal(notMigratedTopicWithNaNInitialRegret.Metadata, newMsg.Metadata)
	s.Require().Equal(notMigratedTopicWithNaNInitialRegret.LossMethod, newMsg.LossMethod)
	s.Require().Equal(notMigratedTopicWithNaNInitialRegret.EpochLastEnded, newMsg.EpochLastEnded)
	s.Require().Equal(notMigratedTopicWithNaNInitialRegret.EpochLength, newMsg.EpochLength)
	s.Require().Equal(notMigratedTopicWithNaNInitialRegret.GroundTruthLag, newMsg.GroundTruthLag)
	s.Require().Equal(notMigratedTopicWithNaNInitialRegret.PNorm.String(), newMsg.PNorm.String())
	s.Require().Equal(notMigratedTopicWithNaNInitialRegret.AlphaRegret.String(), newMsg.AlphaRegret.String())
	s.Require().Equal(notMigratedTopicWithNaNInitialRegret.AllowNegative, newMsg.AllowNegative)
	s.Require().Equal(notMigratedTopicWithNaNInitialRegret.Epsilon.String(), newMsg.Epsilon.String())
	s.Require().Equal(notMigratedTopicWithNaNInitialRegret.WorkerSubmissionWindow, newMsg.WorkerSubmissionWindow)
	// New props are imputed with defaults
	s.Require().Equal("0.1", newMsg.MeritSortitionAlpha.String())
	s.Require().Equal("0.25", newMsg.ActiveInfererQuantile.String())
	s.Require().Equal("0.25", newMsg.ActiveForecasterQuantile.String())
	s.Require().Equal("0.25", newMsg.ActiveReputerQuantile.String())
	// InitialRegret is reset to 0
	s.Require().Equal("0", newMsg.InitialRegret.String())

	// sanity check that the emissions keeper collections.go API also gets the same data
	topic, err := s.emissionsKeeper.GetTopic(s.ctx, 1)
	s.Require().NoError(err)
	s.Require().Equal(newMsg, topic)
}
