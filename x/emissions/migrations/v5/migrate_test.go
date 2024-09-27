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
func (s *EmissionsV4MigrationTestSuite) TestMigratedTopicWithNaNInitialRegret() {
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

// test for deletes on maps that have score as the value of the map
func testScoreMapDeletion(
	s *EmissionsV4MigrationTestSuite,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	key collections.Prefix,
) {
	score := emissionstypes.Score{
		TopicId:     uint64(1),
		BlockHeight: int64(1),
		Address:     "address",
		Score:       alloraMath.NewDecFromInt64(10),
	}

	bz, err := proto.Marshal(&score)
	s.Require().NoError(err)

	mapStore := prefix.NewStore(store, key)
	mapStore.Set([]byte("testKey"), bz)

	// Sanity check
	iterator := mapStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	err = proto.Unmarshal(iterator.Value(), &score)
	s.Require().NoError(err)
	defer iterator.Close()

	v5.ResetMapsWithNonNumericValues(s.ctx, store, cdc)

	// Verify the store has been updated correctly
	iterator = mapStore.Iterator(nil, nil)
	s.Require().False(iterator.Valid(), "iterator should be invalid because the store should be empty")
}

// test for deletes on maps that have scores as the value of the map
func testScoresMapDeletion(
	s *EmissionsV4MigrationTestSuite,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	key collections.Prefix,
) {
	score := emissionstypes.Score{
		TopicId:     uint64(1),
		BlockHeight: int64(1),
		Address:     "address",
		Score:       alloraMath.NewDecFromInt64(10),
	}
	scores := emissionstypes.Scores{Scores: []*emissionstypes.Score{&score}}

	bz, err := proto.Marshal(&scores)
	s.Require().NoError(err)

	mapStore := prefix.NewStore(store, key)
	mapStore.Set([]byte("testKey"), bz)

	// Sanity check
	iterator := mapStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	err = proto.Unmarshal(iterator.Value(), &scores)
	s.Require().NoError(err)
	defer iterator.Close()
	s.Require().Len(scores.Scores, 1)

	v5.ResetMapsWithNonNumericValues(s.ctx, store, cdc)

	// Verify the store has been updated correctly
	iterator = mapStore.Iterator(nil, nil)
	s.Require().False(iterator.Valid(), "iterator should be invalid because the store should be empty")
}

// check that the specified maps are reset correctly
func (s *EmissionsV4MigrationTestSuite) TestResetMapsWithNonNumericValues() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()
	testScoresMapDeletion(s, store, cdc, emissionstypes.InferenceScoresKey)
	testScoresMapDeletion(s, store, cdc, emissionstypes.ForecastScoresKey)
	testScoresMapDeletion(s, store, cdc, emissionstypes.ReputerScoresKey)
	testScoreMapDeletion(s, store, cdc, emissionstypes.InfererScoreEmasKey)
	testScoreMapDeletion(s, store, cdc, emissionstypes.ForecasterScoreEmasKey)
	testScoreMapDeletion(s, store, cdc, emissionstypes.ReputerScoreEmasKey)
	testReputerValueBundleMapDeletion(s, store, cdc, emissionstypes.AllLossBundlesKey)
	testValueBundleMapDeletion(s, store, cdc, emissionstypes.NetworkLossBundlesKey)
	testTimeStampedValueMapDeletion(s, store, cdc, emissionstypes.InfererNetworkRegretsKey)
	testTimeStampedValueMapDeletion(s, store, cdc, emissionstypes.ForecasterNetworkRegretsKey)
	testTimeStampedValueMapDeletion(s, store, cdc, emissionstypes.OneInForecasterNetworkRegretsKey)
	testTimeStampedValueMapDeletion(s, store, cdc, emissionstypes.LatestNaiveInfererNetworkRegretsKey)
	testTimeStampedValueMapDeletion(s, store, cdc, emissionstypes.LatestOneOutInfererInfererNetworkRegretsKey)
	testTimeStampedValueMapDeletion(s, store, cdc, emissionstypes.LatestOneOutInfererForecasterNetworkRegretsKey)
	testTimeStampedValueMapDeletion(s, store, cdc, emissionstypes.LatestOneOutForecasterInfererNetworkRegretsKey)
	testTimeStampedValueMapDeletion(s, store, cdc, emissionstypes.LatestOneOutForecasterForecasterNetworkRegretsKey)
}

/// HELPER FUNCTIONS

// example value bundle for testing
func getBundle() emissionstypes.ValueBundle {
	return emissionstypes.ValueBundle{
		TopicId:             1,
		ReputerRequestNonce: &emissionstypes.ReputerRequestNonce{ReputerNonce: &emissionstypes.Nonce{BlockHeight: 1}},
		Reputer:             "reputer",
		ExtraData:           []byte("extraData"),
		CombinedValue:       alloraMath.NewDecFromInt64(10),
		InfererValues: []*emissionstypes.WorkerAttributedValue{
			{
				Worker: "inferer",
				Value:  alloraMath.NewDecFromInt64(10),
			},
		},
		ForecasterValues: []*emissionstypes.WorkerAttributedValue{
			{
				Worker: "forecaster",
				Value:  alloraMath.NewDecFromInt64(10),
			},
		},
		NaiveValue: alloraMath.NewDecFromInt64(10),
		OneOutInfererValues: []*emissionstypes.WithheldWorkerAttributedValue{
			{
				Worker: "oneOutInferer",
				Value:  alloraMath.NewDecFromInt64(10),
			},
		},
		OneOutForecasterValues: []*emissionstypes.WithheldWorkerAttributedValue{
			{
				Worker: "oneOutForecaster",
				Value:  alloraMath.NewDecFromInt64(10),
			},
		},
		OneOutInfererForecasterValues: []*emissionstypes.OneOutInfererForecasterValues{
			{
				Forecaster: "oneOutInfererForecaster",
				OneOutInfererValues: []*emissionstypes.WithheldWorkerAttributedValue{
					{
						Worker: "oneOutInferer",
						Value:  alloraMath.NewDecFromInt64(10),
					},
				},
			},
		},
	}
}

// test for deletes on maps that have ValueBundles as the value of the map
func testValueBundleMapDeletion(
	s *EmissionsV4MigrationTestSuite,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	key collections.Prefix,
) {
	bundle := getBundle()
	bz, err := proto.Marshal(&bundle)
	s.Require().NoError(err)

	mapStore := prefix.NewStore(store, key)
	mapStore.Set([]byte("testKey"), bz)

	// Sanity check
	iterator := mapStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	err = proto.Unmarshal(iterator.Value(), &bundle)
	s.Require().NoError(err)
	iterator.Close()
	s.Require().Equal(bundle, getBundle())

	v5.ResetMapsWithNonNumericValues(s.ctx, store, cdc)

	// Verify the store has been updated correctly
	iterator = mapStore.Iterator(nil, nil)
	s.Require().False(iterator.Valid(), "iterator should be invalid because the store should be empty")
	iterator.Close()
}

// test for deletes on maps that have ReputerValueBundles as the value of the map
func testReputerValueBundleMapDeletion(
	s *EmissionsV4MigrationTestSuite,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	key collections.Prefix,
) {
	bundle := getBundle()
	reputerValueBundles := emissionstypes.ReputerValueBundles{
		ReputerValueBundles: []*emissionstypes.ReputerValueBundle{
			{
				ValueBundle: &bundle,
				Pubkey:      "something",
				Signature:   []byte("signature"),
			},
		},
	}
	bz, err := proto.Marshal(&reputerValueBundles)
	s.Require().NoError(err)

	mapStore := prefix.NewStore(store, key)
	mapStore.Set([]byte("testKey"), bz)

	// Sanity check
	iterator := mapStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	err = proto.Unmarshal(iterator.Value(), &reputerValueBundles)
	s.Require().NoError(err)
	defer iterator.Close()
	s.Require().Len(reputerValueBundles.ReputerValueBundles, 1)

	v5.ResetMapsWithNonNumericValues(s.ctx, store, cdc)

	// Verify the store has been updated correctly
	iterator = mapStore.Iterator(nil, nil)
	s.Require().False(iterator.Valid(), "iterator should be invalid because the store should be empty")
}

// test for deletes on maps that have TimeStampedValues as the value of the map
func testTimeStampedValueMapDeletion(
	s *EmissionsV4MigrationTestSuite,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	key collections.Prefix,
) {
	timeStampedValue := emissionstypes.TimestampedValue{
		Value:       alloraMath.NewDecFromInt64(10),
		BlockHeight: 1,
	}

	bz, err := proto.Marshal(&timeStampedValue)
	s.Require().NoError(err)

	mapStore := prefix.NewStore(store, key)
	mapStore.Set([]byte("testKey"), bz)

	// Sanity check
	iterator := mapStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	err = proto.Unmarshal(iterator.Value(), &timeStampedValue)
	s.Require().NoError(err)
	defer iterator.Close()

	v5.ResetMapsWithNonNumericValues(s.ctx, store, cdc)

	// Verify the store has been updated correctly
	iterator = mapStore.Iterator(nil, nil)
	s.Require().False(iterator.Valid(), "iterator should be invalid because the store should be empty")
}
