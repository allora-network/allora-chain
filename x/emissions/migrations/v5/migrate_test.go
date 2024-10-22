package v5_test

import (
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	oldV4Types "github.com/allora-network/allora-chain/x/emissions/migrations/v5/oldtypes"

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

// in this test we check that the emissions module params have been migrated
// and the expected new field is set.
func (s *EmissionsV5MigrationTestSuite) TestMigrateParams() {
	storageService := s.emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	defaultParams := emissionstypes.DefaultParams()
	paramsOld := oldV4Types.Params{
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
		MaxStringLength:                     defaultParams.MaxStringLength,
	}

	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&paramsOld))

	// Run migration
	err := v5.MigrateParams(store, cdc)
	s.Require().NoError(err)

	// TO BE ADDED VIA DEFAULT PARAMS
	// InitialRegretQuantile - defaultParams.InitialRegretQuantile
	// PNormSafeDiv - defaultParams.PNormSafeDiv

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
	s.Require().Equal(paramsExpected.InitialRegretQuantile, params.InitialRegretQuantile)
	s.Require().Equal(paramsExpected.PNormSafeDiv, params.PNormSafeDiv)
	s.Require().Equal(paramsExpected, params)
}

// in this test, we check that an already migrated topic, that only initialRegret
// set to 0 but everything else will remain the same
func (s *EmissionsV5MigrationTestSuite) TestMigratedTopicWithNaNInitialRegret() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	_, err := s.emissionsKeeper.IncrementTopicId(s.ctx)
	s.Require().NoError(err)
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

	_, err = s.emissionsKeeper.IncrementTopicId(s.ctx)
	s.Require().NoError(err)
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

func (s *EmissionsV5MigrationTestSuite) TestMigratedSumTotalPreviousTopicWeights() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	_, err := s.emissionsKeeper.IncrementTopicId(s.ctx)
	s.Require().NoError(err)
	migratedOldTopic1 := emissionstypes.Topic{
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
		InitialRegret:            alloraMath.MustNewDecFromString("0"),
		WorkerSubmissionWindow:   120,
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.1337"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.1337"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.1337"),
	}
	migratedOldTopic2 := migratedOldTopic1
	migratedOldTopic2.Id = 2

	_, err = s.emissionsKeeper.IncrementTopicId(s.ctx)
	s.Require().NoError(err)

	// Create 2 topics
	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	// Topic 1
	bz, err := proto.Marshal(&migratedOldTopic1)
	s.Require().NoError(err)
	bytesKey := make([]byte, collections.Uint64Key.Size(migratedOldTopic1.Id))
	countWritten, err := collections.Uint64Key.Encode(bytesKey, migratedOldTopic1.Id)
	s.Require().NoError(err)
	s.Require().NotEqual(0, countWritten)
	topicStore.Set(bytesKey, bz)

	// Topic 2
	_, err = s.emissionsKeeper.IncrementTopicId(s.ctx)
	s.Require().NoError(err)

	bz, err = proto.Marshal(&migratedOldTopic2)
	s.Require().NoError(err)
	bytesKey = make([]byte, collections.Uint64Key.Size(migratedOldTopic2.Id))
	countWritten, err = collections.Uint64Key.Encode(bytesKey, migratedOldTopic2.Id)
	s.Require().NoError(err)
	s.Require().NotEqual(0, countWritten)
	topicStore.Set(bytesKey, bz)

	// Activate 1 topic
	activeTopicsStore := prefix.NewStore(store, emissionstypes.TopicToNextPossibleChurningBlockKey)
	bytesKey = make([]byte, collections.Uint64Key.Size(migratedOldTopic1.Id))
	countWritten, err = collections.Uint64Key.Encode(bytesKey, migratedOldTopic1.Id)
	s.Require().NoError(err)
	s.Require().NotEqual(0, countWritten)
	blockHeightBytes, err := collections.Int64Value.Encode(100)
	s.Require().NoError(err)
	activeTopicsStore.Set(bytesKey, blockHeightBytes)

	// Set 1 topic weight
	topicWeightStore := prefix.NewStore(store, emissionstypes.PreviousTopicWeightKey)
	bytesKey = make([]byte, collections.Uint64Key.Size(migratedOldTopic1.Id))
	countWritten, err = collections.Uint64Key.Encode(bytesKey, migratedOldTopic1.Id)
	s.Require().NoError(err)
	s.Require().NotEqual(0, countWritten)
	topic1WeightDec := alloraMath.MustNewDecFromString("1000.1")
	marshaledWeight, err := topic1WeightDec.Marshal()
	s.Require().NoError(err)
	topicWeightStore.Set(bytesKey, marshaledWeight)

	err = v5.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	// Verify the sumPreviousTopicWeights store has been updated correctly
	sumPreviousTopicWeightsStore := prefix.NewStore(store, emissionstypes.TotalSumPreviousTopicWeightsKey)
	weightsIterator := sumPreviousTopicWeightsStore.Iterator(nil, nil)
	s.Require().True(weightsIterator.Valid())
	defer weightsIterator.Close()

	var weightsMsg alloraMath.Dec // collcodec.ValueCodec[alloraMath.Dec]
	err = weightsMsg.Unmarshal(weightsIterator.Value())
	s.Require().NoError(err)
	s.Require().True(topic1WeightDec.Equal(weightsMsg))

	// Verify the rest of the topic store has been updated correctly
	topicIterator := topicStore.Iterator(nil, nil)
	s.Require().True(topicIterator.Valid())
	defer topicIterator.Close()

	// Check topic 1
	var topic1 emissionstypes.Topic
	err = proto.Unmarshal(topicIterator.Value(), &topic1)
	s.Require().NoError(err)

	// correct props are the same topic 1
	s.Require().Equal(migratedOldTopic1.Id, topic1.Id)
	s.Require().Equal(migratedOldTopic1.Creator, topic1.Creator)
	s.Require().Equal(migratedOldTopic1.Metadata, topic1.Metadata)
	s.Require().Equal(migratedOldTopic1.LossMethod, topic1.LossMethod)
	s.Require().Equal(migratedOldTopic1.EpochLastEnded, topic1.EpochLastEnded)
	s.Require().Equal(migratedOldTopic1.EpochLength, topic1.EpochLength)
	s.Require().Equal(migratedOldTopic1.GroundTruthLag, topic1.GroundTruthLag)
	s.Require().Equal(migratedOldTopic1.PNorm.String(), topic1.PNorm.String())
	s.Require().Equal(migratedOldTopic1.AlphaRegret.String(), topic1.AlphaRegret.String())
	s.Require().False(topic1.AllowNegative)
	s.Require().Equal(migratedOldTopic1.Epsilon.String(), topic1.Epsilon.String())
	s.Require().Equal(migratedOldTopic1.WorkerSubmissionWindow, topic1.WorkerSubmissionWindow)
	s.Require().Equal(migratedOldTopic1.MeritSortitionAlpha.String(), topic1.MeritSortitionAlpha.String())
	s.Require().Equal(migratedOldTopic1.ActiveInfererQuantile.String(), topic1.ActiveInfererQuantile.String())
	s.Require().Equal(migratedOldTopic1.ActiveForecasterQuantile.String(), topic1.ActiveForecasterQuantile.String())
	s.Require().Equal(migratedOldTopic1.ActiveReputerQuantile.String(), topic1.ActiveReputerQuantile.String())
	// InitialRegret is reset to 0
	s.Require().Equal("0", topic1.InitialRegret.String())

	var topic2 emissionstypes.Topic
	err = proto.Unmarshal(topicIterator.Value(), &topic2)
	s.Require().NoError(err)

	// correct props are the same topic 1
	s.Require().Equal(migratedOldTopic1.Id, topic2.Id)
	s.Require().Equal(migratedOldTopic1.Creator, topic2.Creator)
	s.Require().Equal(migratedOldTopic1.Metadata, topic2.Metadata)
	s.Require().Equal(migratedOldTopic1.LossMethod, topic2.LossMethod)
	s.Require().Equal(migratedOldTopic1.EpochLastEnded, topic2.EpochLastEnded)
	s.Require().Equal(migratedOldTopic1.EpochLength, topic2.EpochLength)
	s.Require().Equal(migratedOldTopic1.GroundTruthLag, topic2.GroundTruthLag)
	s.Require().Equal(migratedOldTopic1.PNorm.String(), topic2.PNorm.String())
	s.Require().Equal(migratedOldTopic1.AlphaRegret.String(), topic2.AlphaRegret.String())
	s.Require().False(topic1.AllowNegative)
	s.Require().Equal(migratedOldTopic1.Epsilon.String(), topic2.Epsilon.String())
	s.Require().Equal(migratedOldTopic1.WorkerSubmissionWindow, topic2.WorkerSubmissionWindow)
	s.Require().Equal(migratedOldTopic1.MeritSortitionAlpha.String(), topic2.MeritSortitionAlpha.String())
	s.Require().Equal(migratedOldTopic1.ActiveInfererQuantile.String(), topic2.ActiveInfererQuantile.String())
	s.Require().Equal(migratedOldTopic1.ActiveForecasterQuantile.String(), topic2.ActiveForecasterQuantile.String())
	s.Require().Equal(migratedOldTopic1.ActiveReputerQuantile.String(), topic2.ActiveReputerQuantile.String())
	// InitialRegret is reset to 0
	s.Require().Equal("0", topic2.InitialRegret.String())

	// sanity check that the emissions keeper collections.go API also gets the same data
	topic, err := s.emissionsKeeper.GetTopic(s.ctx, 1)
	s.Require().NoError(err)
	s.Require().Equal(topic1, topic)
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
	s.Require().NoError(err)
	forecasterQuantile, err := alloraMath.NewDecFromString("0.2")
	s.Require().NoError(err)
	reputerQuantile, err := alloraMath.NewDecFromString("0.3")
	s.Require().NoError(err)

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

	err = v5.ResetMapsWithNonNumericValues(s.ctx, store, cdc)
	s.Require().NoError(err)

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
