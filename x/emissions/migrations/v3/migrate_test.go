package v3_test

import (
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"

	codecAddress "github.com/cosmos/cosmos-sdk/codec/address"

	"github.com/allora-network/allora-chain/app/params"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	migrations "github.com/allora-network/allora-chain/x/emissions/migrations/v3"
	emissions "github.com/allora-network/allora-chain/x/emissions/module"
	emissionstestutil "github.com/allora-network/allora-chain/x/emissions/testutil"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	storetypes "cosmossdk.io/store/types"
	oldparams "github.com/allora-network/allora-chain/x/emissions/migrations/v3/types"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
)

type MigrationTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	ctx             sdk.Context
	emissionsKeeper *keeper.Keeper
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(MigrationTestSuite))
}

func (s *MigrationTestSuite) SetupTest() {
	encCfg := moduletestutil.MakeTestEncodingConfig(emissions.AppModule{})
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
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

func (s *MigrationTestSuite) TestMigrate() {

	storageService := s.emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.ctx))
	cdc := s.emissionsKeeper.GetBinaryCodec()

	defaultParams := types.DefaultParams()
	paramsOld := oldparams.Params{
		Version:                             defaultParams.Version,
		MaxSerializedMsgLength:              defaultParams.MaxSerializedMsgLength,
		MinTopicWeight:                      defaultParams.MinTopicWeight,
		MaxTopicsPerBlock:                   defaultParams.MaxTopicsPerBlock,
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
	}

	store.Set(types.ParamsKey, cdc.MustMarshal(&paramsOld))

	// Run migration
	err := migrations.MigrateStore(s.ctx, *s.emissionsKeeper)
	s.Require().NoError(err)

	// TO BE ADDED VIA DEFAULT PARAMS
	// MaxElementsPerForecast: defaultParams.MaxElementsPerForecast
	paramsExpected := defaultParams

	params, err := s.emissionsKeeper.GetParams(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(paramsExpected.Version, params.Version)
	s.Require().Equal(paramsExpected.MaxSerializedMsgLength, params.MaxSerializedMsgLength)
	s.Require().True(paramsExpected.MinTopicWeight.Equal(params.MinTopicWeight), "%s!=%s", paramsExpected.MinTopicWeight.String(), params.MinTopicWeight.String())
	s.Require().Equal(paramsExpected.MaxTopicsPerBlock, params.MaxTopicsPerBlock)
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
	s.Require().Equal(paramsExpected, params)
}
