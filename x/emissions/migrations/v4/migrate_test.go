package v4_test

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
	oldV2Types "github.com/allora-network/allora-chain/x/emissions/migrations/v3/oldtypes"
	v4 "github.com/allora-network/allora-chain/x/emissions/migrations/v4"
	oldV3Types "github.com/allora-network/allora-chain/x/emissions/migrations/v4/oldtypes"
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

// in this test we check that the emissions module params have been migrated
// and the expected new field is set.
func (s *EmissionsV4MigrationTestSuite) TestMigrateParams() {
	storageService := s.emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	defaultParams := emissionstypes.DefaultParams()
	paramsOld := oldV3Types.Params{
		Version:                             defaultParams.Version,
		MaxSerializedMsgLength:              defaultParams.MaxSerializedMsgLength,
		MinTopicWeight:                      defaultParams.MinTopicWeight,
		RequiredMinimumStake:                defaultParams.RequiredMinimumStake,
		RemoveStakeDelayWindow:              defaultParams.RemoveStakeDelayWindow,
		MinEpochLength:                      defaultParams.MinEpochLength,
		BetaEntropy:                         defaultParams.BetaEntropy,
		LearningRate:                        defaultParams.LearningRate,
		MaxGradientThreshold:                defaultParams.MaxGradientThreshold,
		MinStakeFraction:                    defaultParams.MinStakeFraction,
		MaxUnfulfilledWorkerRequests:        defaultParams.MaxUnfulfilledWorkerRequests,
		MaxUnfulfilledReputerRequests:       defaultParams.MaxUnfulfilledReputerRequests,
		TopicRewardStakeImportance:          defaultParams.TopicRewardStakeImportance,
		TopicRewardFeeRevenueImportance:     defaultParams.TopicRewardFeeRevenueImportance,
		TopicRewardAlpha:                    defaultParams.TopicRewardAlpha,
		TaskRewardAlpha:                     defaultParams.TaskRewardAlpha,
		ValidatorsVsAlloraPercentReward:     defaultParams.ValidatorsVsAlloraPercentReward,
		MaxSamplesToScaleScores:             defaultParams.MaxSamplesToScaleScores,
		MaxTopInferersToReward:              defaultParams.MaxTopInferersToReward,
		MaxTopForecastersToReward:           defaultParams.MaxTopForecastersToReward,
		MaxTopReputersToReward:              defaultParams.MaxTopReputersToReward,
		CreateTopicFee:                      defaultParams.CreateTopicFee,
		GradientDescentMaxIters:             defaultParams.GradientDescentMaxIters,
		RegistrationFee:                     defaultParams.RegistrationFee,
		DefaultPageLimit:                    defaultParams.DefaultPageLimit,
		MaxPageLimit:                        defaultParams.MaxPageLimit,
		MinEpochLengthRecordLimit:           defaultParams.MinEpochLengthRecordLimit,
		BlocksPerMonth:                      defaultParams.BlocksPerMonth,
		PRewardInference:                    defaultParams.PRewardInference,
		PRewardForecast:                     defaultParams.PRewardForecast,
		PRewardReputer:                      defaultParams.PRewardReputer,
		CRewardInference:                    defaultParams.CRewardInference,
		CRewardForecast:                     defaultParams.CRewardForecast,
		CNorm:                               defaultParams.CNorm,
		EpsilonReputer:                      defaultParams.EpsilonReputer,
		HalfMaxProcessStakeRemovalsEndBlock: defaultParams.HalfMaxProcessStakeRemovalsEndBlock,
		EpsilonSafeDiv:                      defaultParams.EpsilonSafeDiv,
		DataSendingFee:                      defaultParams.DataSendingFee,
		MaxElementsPerForecast:              defaultParams.MaxElementsPerForecast,
		MaxActiveTopicsPerBlock:             defaultParams.MaxActiveTopicsPerBlock,
	}

	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&paramsOld))

	// Run migration
	err := v4.MigrateParams(store, cdc)
	s.Require().NoError(err)

	// TO BE ADDED VIA DEFAULT PARAMS
	// MaxStringLength - defaultParams.MaxStringLength

	paramsExpected := defaultParams

	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(paramsExpected.Version, params.Version)
	s.Require().Equal(paramsExpected.MaxSerializedMsgLength, params.MaxSerializedMsgLength)
	s.Require().True(paramsExpected.MinTopicWeight.Equal(params.MinTopicWeight), "%s!=%s", paramsExpected.MinTopicWeight.String(), params.MinTopicWeight.String())
	s.Require().True(paramsExpected.RequiredMinimumStake.Equal(params.RequiredMinimumStake), "%s!=%s", paramsExpected.RequiredMinimumStake, params.RequiredMinimumStake)
	s.Require().Equal(paramsExpected.RemoveStakeDelayWindow, params.RemoveStakeDelayWindow)
	s.Require().Equal(paramsExpected.MinEpochLength, params.MinEpochLength)
	s.Require().True(paramsExpected.BetaEntropy.Equal(params.BetaEntropy), "%s!=%s", paramsExpected.BetaEntropy, params.BetaEntropy)
	s.Require().True(paramsExpected.LearningRate.Equal(params.LearningRate), "%s!=%s", paramsExpected.LearningRate, params.LearningRate)
	s.Require().True(paramsExpected.MaxGradientThreshold.Equal(params.MaxGradientThreshold), "%s!=%s", paramsExpected.MaxGradientThreshold, params.MaxGradientThreshold)
	s.Require().True(paramsExpected.MinStakeFraction.Equal(params.MinStakeFraction), "%s!=%s", paramsExpected.MinStakeFraction, params.MinStakeFraction)
	s.Require().Equal(paramsExpected.MaxUnfulfilledWorkerRequests, params.MaxUnfulfilledWorkerRequests)
	s.Require().Equal(paramsExpected.MaxUnfulfilledReputerRequests, params.MaxUnfulfilledReputerRequests)
	s.Require().True(paramsExpected.TopicRewardStakeImportance.Equal(params.TopicRewardStakeImportance), "%s!=%s", paramsExpected.TopicRewardStakeImportance, params.TopicRewardStakeImportance)
	s.Require().True(paramsExpected.TopicRewardFeeRevenueImportance.Equal(params.TopicRewardFeeRevenueImportance), "%s!=%s", paramsExpected.TopicRewardFeeRevenueImportance, params.TopicRewardFeeRevenueImportance)
	s.Require().True(paramsExpected.TopicRewardAlpha.Equal(params.TopicRewardAlpha), "%s!=%s", paramsExpected.TopicRewardAlpha, params.TopicRewardAlpha)
	s.Require().True(paramsExpected.TaskRewardAlpha.Equal(params.TaskRewardAlpha), "%s!=%s", paramsExpected.TaskRewardAlpha, params.TaskRewardAlpha)
	s.Require().True(paramsExpected.ValidatorsVsAlloraPercentReward.Equal(params.ValidatorsVsAlloraPercentReward), "%s!=%s", paramsExpected.ValidatorsVsAlloraPercentReward, params.ValidatorsVsAlloraPercentReward)
	s.Require().Equal(paramsExpected.MaxSamplesToScaleScores, params.MaxSamplesToScaleScores)
	s.Require().Equal(paramsExpected.MaxTopInferersToReward, params.MaxTopInferersToReward)
	s.Require().Equal(paramsExpected.MaxTopForecastersToReward, params.MaxTopForecastersToReward)
	s.Require().Equal(paramsExpected.MaxTopReputersToReward, params.MaxTopReputersToReward)
	s.Require().True(paramsExpected.CreateTopicFee.Equal(params.CreateTopicFee), "%s!=%s", paramsExpected.CreateTopicFee, params.CreateTopicFee)
	s.Require().Equal(paramsExpected.GradientDescentMaxIters, params.GradientDescentMaxIters)
	s.Require().True(paramsExpected.RegistrationFee.Equal(params.RegistrationFee), "%s!=%s", paramsExpected.RegistrationFee, params.RegistrationFee)
	s.Require().Equal(paramsExpected.DefaultPageLimit, params.DefaultPageLimit)
	s.Require().Equal(paramsExpected.MaxPageLimit, params.MaxPageLimit)
	s.Require().Equal(paramsExpected.MinEpochLengthRecordLimit, params.MinEpochLengthRecordLimit)
	s.Require().Equal(paramsExpected.BlocksPerMonth, params.BlocksPerMonth)
	s.Require().True(paramsExpected.PRewardInference.Equal(params.PRewardInference), "%s!=%s", paramsExpected.PRewardInference, params.PRewardInference)
	s.Require().True(paramsExpected.PRewardForecast.Equal(params.PRewardForecast), "%s!=%s", paramsExpected.PRewardForecast, params.PRewardForecast)
	s.Require().True(paramsExpected.PRewardReputer.Equal(params.PRewardReputer), "%s!=%s", paramsExpected.PRewardReputer, params.PRewardReputer)
	s.Require().True(paramsExpected.CRewardInference.Equal(params.CRewardInference), "%s!=%s", paramsExpected.CRewardInference, params.CRewardInference)
	s.Require().True(paramsExpected.CRewardForecast.Equal(params.CRewardForecast), "%s!=%s", paramsExpected.CRewardForecast, params.CRewardForecast)
	s.Require().True(paramsExpected.CNorm.Equal(params.CNorm), "%s!=%s", paramsExpected.CNorm, params.CNorm)
	s.Require().True(paramsExpected.EpsilonReputer.Equal(params.EpsilonReputer), "%s!=%s", paramsExpected.EpsilonReputer, params.EpsilonReputer)
	s.Require().Equal(paramsExpected.HalfMaxProcessStakeRemovalsEndBlock, params.HalfMaxProcessStakeRemovalsEndBlock)
	s.Require().True(paramsExpected.EpsilonSafeDiv.Equal(params.EpsilonSafeDiv), "%s!=%s", paramsExpected.EpsilonSafeDiv, params.EpsilonSafeDiv)
	s.Require().True(paramsExpected.DataSendingFee.Equal(params.DataSendingFee), "%s!=%s", paramsExpected.DataSendingFee, params.DataSendingFee)
	s.Require().Equal(paramsExpected.MaxElementsPerForecast, params.MaxElementsPerForecast)
	s.Require().Equal(paramsExpected.MaxActiveTopicsPerBlock, params.MaxActiveTopicsPerBlock)
	s.Require().Equal(paramsExpected.MaxStringLength, params.MaxStringLength)
	s.Require().Equal(paramsExpected, params)
}

// in this test, we check that an already migrated topic, that has all the new fields
// for merit sortition, and everything looks correct, will not change at all
func (s *EmissionsV4MigrationTestSuite) TestMigratedTopicWithNoProblems() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	migratedOldTopic := emissionstypes.Topic{
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

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	bytesKey := make([]byte, collections.Uint64Key.Size(migratedOldTopic.Id))
	countWritten, err := collections.Uint64Key.Encode(bytesKey, migratedOldTopic.Id)
	s.Require().NoError(err)
	s.Require().NotEqual(0, countWritten)
	topicStore.Set(bytesKey, bz)

	err = v4.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator := topicStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	defer iterator.Close()

	var newMsg emissionstypes.Topic
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

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	bytesKey := make([]byte, collections.Uint64Key.Size(migratedOldTopicWithNaNInitialRegret.Id))
	countWritten, err := collections.Uint64Key.Encode(bytesKey, migratedOldTopicWithNaNInitialRegret.Id)
	s.Require().NoError(err)
	s.Require().NotEqual(0, countWritten)
	topicStore.Set(bytesKey, bz)
	err = v4.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
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

	notMigratedTopic := oldV2Types.Topic{
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

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	bytesKey := make([]byte, collections.Uint64Key.Size(notMigratedTopic.Id))
	countWritten, err := collections.Uint64Key.Encode(bytesKey, notMigratedTopic.Id)
	s.Require().NoError(err)
	s.Require().NotEqual(0, countWritten)
	topicStore.Set(bytesKey, bz)
	err = v4.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	err = v4.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator := topicStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	defer iterator.Close()

	var newMsg emissionstypes.Topic
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

	notMigratedTopicWithNaNInitialRegret := oldV2Types.Topic{
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

	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	bytesKey := make([]byte, collections.Uint64Key.Size(notMigratedTopicWithNaNInitialRegret.Id))
	countWritten, err := collections.Uint64Key.Encode(bytesKey, notMigratedTopicWithNaNInitialRegret.Id)
	s.Require().NoError(err)
	s.Require().NotEqual(0, countWritten)
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

	var newMsg emissionstypes.Topic
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
	defer iterator.Close()
	s.Require().True(iterator.Valid())
	err = proto.Unmarshal(iterator.Value(), &score)
	s.Require().NoError(err)
	iterator.Close()

	err = v4.ResetMapsWithNonNumericValues(s.ctx, store, cdc)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator = mapStore.Iterator(nil, nil)
	defer iterator.Close()
	s.Require().False(iterator.Valid(), "iterator should be invalid because the store should be empty")
	iterator.Close()
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
	defer iterator.Close()
	s.Require().True(iterator.Valid())
	err = proto.Unmarshal(iterator.Value(), &scores)
	s.Require().NoError(err)
	iterator.Close()
	s.Require().Len(scores.Scores, 1)

	err = v4.ResetMapsWithNonNumericValues(s.ctx, store, cdc)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator = mapStore.Iterator(nil, nil)
	defer iterator.Close()
	s.Require().False(iterator.Valid(), "iterator should be invalid because the store should be empty")
	iterator.Close()
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
	defer iterator.Close()
	s.Require().True(iterator.Valid())
	err = proto.Unmarshal(iterator.Value(), &bundle)
	s.Require().NoError(err)
	iterator.Close()
	s.Require().Equal(bundle, getBundle())

	err = v4.ResetMapsWithNonNumericValues(s.ctx, store, cdc)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator = mapStore.Iterator(nil, nil)
	defer iterator.Close()
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
	defer iterator.Close()
	s.Require().True(iterator.Valid())
	err = proto.Unmarshal(iterator.Value(), &reputerValueBundles)
	s.Require().NoError(err)
	iterator.Close()
	s.Require().Len(reputerValueBundles.ReputerValueBundles, 1)

	err = v4.ResetMapsWithNonNumericValues(s.ctx, store, cdc)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator = mapStore.Iterator(nil, nil)
	defer iterator.Close()
	s.Require().False(iterator.Valid(), "iterator should be invalid because the store should be empty")
	iterator.Close()
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
	defer iterator.Close()
	s.Require().True(iterator.Valid())
	err = proto.Unmarshal(iterator.Value(), &timeStampedValue)
	s.Require().NoError(err)
	iterator.Close()

	err = v4.ResetMapsWithNonNumericValues(s.ctx, store, cdc)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator = mapStore.Iterator(nil, nil)
	defer iterator.Close()
	s.Require().False(iterator.Valid(), "iterator should be invalid because the store should be empty")
	iterator.Close()
}
