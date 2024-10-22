package v3_test

import (
	"strconv"
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"

	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"

	"cosmossdk.io/core/store"
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"

	"cosmossdk.io/store/prefix"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	v3 "github.com/allora-network/allora-chain/x/emissions/migrations/v3"
	oldtypes "github.com/allora-network/allora-chain/x/emissions/migrations/v3/oldtypes"
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

type EmissionsV3MigrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	ctx             sdk.Context
	storeService    store.KVStoreService
	emissionsKeeper *keeper.Keeper
}

func TestEmissionsV3MigrationTestSuite(t *testing.T) {
	suite.Run(t, new(EmissionsV3MigrationTestSuite))
}

func (s *EmissionsV3MigrationTestSuite) SetupTest() {
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
		authtypes.FeeCollectorName)

	s.emissionsKeeper = &emissionsKeeper
}

func (s *EmissionsV3MigrationTestSuite) TestMigrate() {
	storageService := s.emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	defaultParams := types.DefaultParams()
	paramsOld := oldtypes.Params{
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

		// TO BE DELETED
		MinEffectiveTopicRevenue:        alloraMath.NewDecFromInt64(1337),
		TopicFeeRevenueDecayRate:        alloraMath.NewDecFromInt64(1338),
		MaxRetriesToFulfilNoncesWorker:  4242,
		MaxRetriesToFulfilNoncesReputer: 4243,
		MaxTopicsPerBlock:               4244,
	}

	store.Set(types.ParamsKey, cdc.MustMarshal(&paramsOld))

	// Run migration
	err := v3.MigrateStore(s.ctx, *s.emissionsKeeper)
	s.Require().NoError(err)

	// TO BE ADDED VIA DEFAULT PARAMS
	// MaxElementsPerForecast: defaultParams.MaxElementsPerForecast
	// MaxActiveTopicsPerBlock: defaultParams.MaxActiveTopicsPerBlock

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
	// commenting this out as this migration has already happened, so this test is no longer relevant
	//s.Require().Equal(paramsExpected, params)
}

func (s *EmissionsV3MigrationTestSuite) TestMigrateTopics() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	oldTopic := oldtypes.Topic{
		Id:             1,
		Creator:        "creator",
		Metadata:       "metadata",
		LossMethod:     "lossmethod",
		EpochLastEnded: 0,
		EpochLength:    100,
		GroundTruthLag: 10,
		PNorm:          alloraMath.NewDecFromInt64(3),
		AlphaRegret:    alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:  false,
		Epsilon:        alloraMath.MustNewDecFromString("0.01"),
		// InitialRegret is being reset to account for NaNs that were previously stored due to insufficient validation
		InitialRegret:          alloraMath.MustNewDecFromString("11"),
		WorkerSubmissionWindow: 120,
	}

	bz, err := proto.Marshal(&oldTopic)
	s.Require().NoError(err)

	topicStore := prefix.NewStore(store, types.TopicsKey)
	topicStore.Set([]byte("testKey"), bz)

	err = v3.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator := topicStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	defer iterator.Close()

	var newMsg types.Topic
	err = proto.Unmarshal(iterator.Value(), &newMsg)
	s.Require().NoError(err)

	// Old props are the same
	s.Require().Equal(oldTopic.Id, newMsg.Id)
	s.Require().Equal(oldTopic.Creator, newMsg.Creator)
	s.Require().Equal(oldTopic.Metadata, newMsg.Metadata)
	s.Require().Equal(oldTopic.LossMethod, newMsg.LossMethod)
	s.Require().Equal(oldTopic.EpochLength, newMsg.EpochLength)
	s.Require().Equal(oldTopic.GroundTruthLag, newMsg.GroundTruthLag)
	s.Require().Equal(oldTopic.PNorm.String(), newMsg.PNorm.String())
	s.Require().Equal(oldTopic.AlphaRegret.String(), newMsg.AlphaRegret.String())
	s.Require().Equal(oldTopic.AllowNegative, newMsg.AllowNegative)
	s.Require().Equal(oldTopic.EpochLastEnded, newMsg.EpochLastEnded)
	// New props are imputed with defaults
	s.Require().Equal("0.1", newMsg.MeritSortitionAlpha.String())
	s.Require().Equal("0.25", newMsg.ActiveInfererQuantile.String())
	s.Require().Equal("0.25", newMsg.ActiveForecasterQuantile.String())
	s.Require().Equal("0.25", newMsg.ActiveReputerQuantile.String())
	// InitialRegret is reset to 0
	s.Require().Equal("0", newMsg.InitialRegret.String())
}

func (s *EmissionsV3MigrationTestSuite) TestMigrateTopicsWithWeightSameEpoch() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	oldTopics := []oldtypes.Topic{
		{
			Id:             1,
			Creator:        "creator",
			Metadata:       "metadata",
			LossMethod:     "lossmethod",
			EpochLastEnded: 0,
			EpochLength:    100,
			GroundTruthLag: 10,
			PNorm:          alloraMath.NewDecFromInt64(3),
			AlphaRegret:    alloraMath.MustNewDecFromString("0.1"),
			AllowNegative:  false,
			Epsilon:        alloraMath.MustNewDecFromString("0.01"),
			// InitialRegret is being reset to account for NaNs that were previously stored due to insufficient validation
			InitialRegret:          alloraMath.MustNewDecFromString("11"),
			WorkerSubmissionWindow: 120,
		},
		{
			Id:                     2,
			Creator:                "creator2",
			Metadata:               "metadata2",
			LossMethod:             "lossmethod2",
			EpochLastEnded:         0,
			EpochLength:            100,
			GroundTruthLag:         20,
			PNorm:                  alloraMath.NewDecFromInt64(3),
			AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
			AllowNegative:          false,
			Epsilon:                alloraMath.MustNewDecFromString("0.01"),
			InitialRegret:          alloraMath.MustNewDecFromString("11"),
			WorkerSubmissionWindow: 120,
		},
		{
			Id:                     3,
			Creator:                "creator3",
			Metadata:               "metadata3",
			LossMethod:             "lossmethod3",
			EpochLastEnded:         0,
			EpochLength:            100,
			GroundTruthLag:         30,
			PNorm:                  alloraMath.NewDecFromInt64(3),
			AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
			AllowNegative:          false,
			Epsilon:                alloraMath.MustNewDecFromString("0.01"),
			InitialRegret:          alloraMath.MustNewDecFromString("11"),
			WorkerSubmissionWindow: 130,
		},
	}
	err := s.emissionsKeeper.AddTopicFeeRevenue(s.ctx, 1, cosmosMath.NewInt(40000))
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddTopicFeeRevenue(s.ctx, 2, cosmosMath.NewInt(70000))
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddTopicFeeRevenue(s.ctx, 3, cosmosMath.NewInt(60000))
	s.Require().NoError(err)

	err = s.emissionsKeeper.SetTopicStake(s.ctx, 1, cosmosMath.NewInt(40000))
	s.Require().NoError(err)
	err = s.emissionsKeeper.SetTopicStake(s.ctx, 2, cosmosMath.NewInt(70000))
	s.Require().NoError(err)
	err = s.emissionsKeeper.SetTopicStake(s.ctx, 3, cosmosMath.NewInt(60000))
	s.Require().NoError(err)

	topicStore := prefix.NewStore(store, types.TopicsKey)
	for i, oldTopic := range oldTopics {
		bz, err := proto.Marshal(&oldTopic)
		s.Require().NoError(err)

		topicStore.Set([]byte("testKey"+strconv.Itoa(i+1)), bz)
	}

	err = v3.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator := topicStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	defer iterator.Close()

	// this is from topic.BlockHeightEnded + topic.EpochLength
	blockHeightEnded := int64(100)

	churningBlock, inFuture, err := s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, 1)
	s.Require().NoError(err)
	s.Require().Equal(int64(0), churningBlock)
	s.Require().False(inFuture)

	churningBlock, inFuture, err = s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, 2)
	s.Require().NoError(err)
	s.Require().Equal(churningBlock, blockHeightEnded)
	s.Require().True(inFuture)

	churningBlock, inFuture, err = s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, 3)
	s.Require().NoError(err)
	s.Require().Equal(int64(0), churningBlock)
	s.Require().False(inFuture)

	// not the same as feeRev * stake because weight is EMAd with 0
	lowestWeight, noPrior, err := s.emissionsKeeper.GetLowestActiveTopicWeightAtBlock(s.ctx, blockHeightEnded)
	s.Require().False(noPrior)
	s.Require().NoError(err)
	s.Require().True(lowestWeight.Weight.Gt(alloraMath.ZeroDec()))

	activeTopicIds, err := s.emissionsKeeper.GetActiveTopicIdsAtBlock(s.ctx, blockHeightEnded)
	s.Require().NoError(err)
	s.Require().Len(activeTopicIds.TopicIds, 1)
	s.Require().NotContains(activeTopicIds.TopicIds, uint64(1))
	s.Require().Contains(activeTopicIds.TopicIds, uint64(2))
	s.Require().NotContains(activeTopicIds.TopicIds, uint64(3))
}

func (s *EmissionsV3MigrationTestSuite) TestMigrateTopicsWithWeightDifferentEpoch() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	blockHeightEnded1 := int64(100)
	blockHeightEnded2 := int64(200)
	blockHeightEnded3 := int64(300)

	oldTopics := []oldtypes.Topic{
		{
			Id:             1,
			Creator:        "creator",
			Metadata:       "metadata",
			LossMethod:     "lossmethod",
			EpochLastEnded: 0,
			EpochLength:    blockHeightEnded1,
			GroundTruthLag: 10,
			PNorm:          alloraMath.NewDecFromInt64(3),
			AlphaRegret:    alloraMath.MustNewDecFromString("0.1"),
			AllowNegative:  false,
			Epsilon:        alloraMath.MustNewDecFromString("0.01"),
			// InitialRegret is being reset to account for NaNs that were previously stored due to insufficient validation
			InitialRegret:          alloraMath.MustNewDecFromString("11"),
			WorkerSubmissionWindow: 120,
		},
		{
			Id:                     2,
			Creator:                "creator2",
			Metadata:               "metadata2",
			LossMethod:             "lossmethod2",
			EpochLastEnded:         0,
			EpochLength:            blockHeightEnded2,
			GroundTruthLag:         20,
			PNorm:                  alloraMath.NewDecFromInt64(3),
			AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
			AllowNegative:          false,
			Epsilon:                alloraMath.MustNewDecFromString("0.01"),
			InitialRegret:          alloraMath.MustNewDecFromString("11"),
			WorkerSubmissionWindow: 120,
		},
		{
			Id:                     3,
			Creator:                "creator3",
			Metadata:               "metadata3",
			LossMethod:             "lossmethod3",
			EpochLastEnded:         0,
			EpochLength:            blockHeightEnded3,
			GroundTruthLag:         30,
			PNorm:                  alloraMath.NewDecFromInt64(3),
			AlphaRegret:            alloraMath.MustNewDecFromString("0.1"),
			AllowNegative:          false,
			Epsilon:                alloraMath.MustNewDecFromString("0.01"),
			InitialRegret:          alloraMath.MustNewDecFromString("11"),
			WorkerSubmissionWindow: 130,
		},
	}
	err := s.emissionsKeeper.AddTopicFeeRevenue(s.ctx, 1, cosmosMath.NewInt(20000))
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddTopicFeeRevenue(s.ctx, 2, cosmosMath.NewInt(40000))
	s.Require().NoError(err)
	err = s.emissionsKeeper.AddTopicFeeRevenue(s.ctx, 3, cosmosMath.NewInt(60000))
	s.Require().NoError(err)

	err = s.emissionsKeeper.SetTopicStake(s.ctx, 1, cosmosMath.NewInt(20000))
	s.Require().NoError(err)
	err = s.emissionsKeeper.SetTopicStake(s.ctx, 2, cosmosMath.NewInt(40000))
	s.Require().NoError(err)
	err = s.emissionsKeeper.SetTopicStake(s.ctx, 3, cosmosMath.NewInt(60000))
	s.Require().NoError(err)

	topicStore := prefix.NewStore(store, types.TopicsKey)
	for i, oldTopic := range oldTopics {
		bz, err := proto.Marshal(&oldTopic)
		s.Require().NoError(err)

		topicStore.Set([]byte("testKey"+strconv.Itoa(i+1)), bz)
	}

	err = v3.MigrateTopics(s.ctx, store, cdc, *s.emissionsKeeper)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator := topicStore.Iterator(nil, nil)
	s.Require().True(iterator.Valid())
	defer iterator.Close()

	// this is from topic.BlockHeightEnded + topic.EpochLength

	churningBlock, inFuture, err := s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, 1)
	s.Require().NoError(err)
	s.Require().Equal(churningBlock, blockHeightEnded1)
	s.Require().True(inFuture)

	churningBlock, inFuture, err = s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, 2)
	s.Require().NoError(err)
	s.Require().Equal(churningBlock, blockHeightEnded2)
	s.Require().True(inFuture)

	churningBlock, inFuture, err = s.emissionsKeeper.GetNextPossibleChurningBlockByTopicId(s.ctx, 3)
	s.Require().NoError(err)
	s.Require().Equal(churningBlock, blockHeightEnded3)
	s.Require().True(inFuture)

	// not the same as feeRev * stake because weight is EMAd with 0
	lowestWeight, noPrior, err := s.emissionsKeeper.GetLowestActiveTopicWeightAtBlock(s.ctx, blockHeightEnded1)
	s.Require().False(noPrior)
	s.Require().NoError(err)
	s.Require().True(lowestWeight.Weight.Gt(alloraMath.ZeroDec()))

	lowestWeight, noPrior, err = s.emissionsKeeper.GetLowestActiveTopicWeightAtBlock(s.ctx, blockHeightEnded2)
	s.Require().False(noPrior)
	s.Require().NoError(err)
	s.Require().True(lowestWeight.Weight.Gt(alloraMath.ZeroDec()))

	lowestWeight, noPrior, err = s.emissionsKeeper.GetLowestActiveTopicWeightAtBlock(s.ctx, blockHeightEnded3)
	s.Require().False(noPrior)
	s.Require().NoError(err)
	s.Require().True(lowestWeight.Weight.Gt(alloraMath.ZeroDec()))

	activeTopicIds, err := s.emissionsKeeper.GetActiveTopicIdsAtBlock(s.ctx, blockHeightEnded1)
	s.Require().NoError(err)
	s.Require().Len(activeTopicIds.TopicIds, 1)
	s.Require().Contains(activeTopicIds.TopicIds, uint64(1))

	activeTopicIds, err = s.emissionsKeeper.GetActiveTopicIdsAtBlock(s.ctx, blockHeightEnded2)
	s.Require().NoError(err)
	s.Require().Len(activeTopicIds.TopicIds, 1)
	s.Require().Contains(activeTopicIds.TopicIds, uint64(2))

	activeTopicIds, err = s.emissionsKeeper.GetActiveTopicIdsAtBlock(s.ctx, blockHeightEnded3)
	s.Require().NoError(err)
	s.Require().Len(activeTopicIds.TopicIds, 1)
	s.Require().Contains(activeTopicIds.TopicIds, uint64(3))
}

func (s *EmissionsV3MigrationTestSuite) TestResetMapsWithNonNumericValues() {
	store := runtime.KVStoreAdapter(s.storeService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	score := []*types.Score{
		{
			TopicId:     uint64(1),
			BlockHeight: int64(1),
			Address:     "address",
			Score:       alloraMath.NewDecFromInt64(10),
		},
	}
	scores := types.Scores{Scores: score}

	bz, err := proto.Marshal(&scores)
	s.Require().NoError(err)

	infererScoresByBlock := prefix.NewStore(store, types.InferenceScoresKey)
	infererScoresByBlock.Set([]byte("testKey"), bz)

	// Sanity check
	iterator := infererScoresByBlock.Iterator(nil, nil)
	defer iterator.Close()
	s.Require().True(iterator.Valid())
	err = proto.Unmarshal(iterator.Value(), &scores)
	s.Require().NoError(err)
	iterator.Close()
	s.Require().Len(scores.Scores, 1)

	err = v3.ResetMapsWithNonNumericValues(s.ctx, store, cdc)
	s.Require().NoError(err)

	// Verify the store has been updated correctly
	iterator = infererScoresByBlock.Iterator(nil, nil)
	defer iterator.Close()
	s.Require().False(iterator.Valid(), "iterator should be invalid because the store should be empty")
	iterator.Close()
}
