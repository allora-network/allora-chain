package v12_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/stretchr/testify/suite"

	alloraMath "github.com/allora-network/allora-chain/math"
	v12 "github.com/allora-network/allora-chain/x/emissions/migrations/v12"
	oldV11Types "github.com/allora-network/allora-chain/x/emissions/migrations/v12/oldtypes"
	"github.com/allora-network/allora-chain/x/emissions/testutil"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

type EmissionsV12MigrationTestSuite struct {
	testutil.TestSuite
}

func TestEmissionsV12MigrationTestSuite(t *testing.T) {
	suite.Run(t, &EmissionsV12MigrationTestSuite{
		testutil.NewTestSuite("emissions_V12Migrations"),
	})
}

// TestMigrateParams tests that CNorm is removed from params
func (s *EmissionsV12MigrationTestSuite) TestMigrateParams() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()

	defaultParams := emissionstypes.DefaultParams()
	oldCNorm := alloraMath.MustNewDecFromString("0.75")

	paramsOld := oldV11Types.Params{ //nolint: exhaustruct // this is an old version of the params
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
		CNorm:                               oldCNorm,
		EpsilonReputer:                      defaultParams.EpsilonReputer,
		HalfMaxProcessStakeRemovalsEndBlock: defaultParams.HalfMaxProcessStakeRemovalsEndBlock,
		EpsilonSafeDiv:                      defaultParams.EpsilonSafeDiv,
		DataSendingFee:                      defaultParams.DataSendingFee,
		MaxElementsPerForecast:              defaultParams.MaxElementsPerForecast,
		MaxActiveTopicsPerBlock:             defaultParams.MaxActiveTopicsPerBlock,
		MaxStringLength:                     defaultParams.MaxStringLength,
		InitialRegretQuantile:               defaultParams.InitialRegretQuantile,
		PNormSafeDiv:                        defaultParams.PNormSafeDiv,
		GlobalWhitelistEnabled:              defaultParams.GlobalWhitelistEnabled,
		TopicCreatorWhitelistEnabled:        defaultParams.TopicCreatorWhitelistEnabled,
		MinExperiencedWorkerRegrets:         defaultParams.MinExperiencedWorkerRegrets,
		InferenceOutlierDetectionThreshold:  defaultParams.InferenceOutlierDetectionThreshold,
		InferenceOutlierDetectionAlpha:      defaultParams.InferenceOutlierDetectionAlpha,
		LambdaInitialScore:                  defaultParams.LambdaInitialScore,
		GlobalWorkerWhitelistEnabled:        defaultParams.GlobalWorkerWhitelistEnabled,
		GlobalReputerWhitelistEnabled:       defaultParams.GlobalReputerWhitelistEnabled,
		GlobalAdminWhitelistAppended:        defaultParams.GlobalAdminWhitelistAppended,
		MaxWhitelistInputArrayLength:        defaultParams.MaxWhitelistInputArrayLength,
		MinWeightThresholdForStdnorm:        defaultParams.MinWeightThresholdForStdnorm,
	}

	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&paramsOld))

	// Run migration
	err := v12.MigrateParams(store, cdc)
	s.Require().NoError(err)

	// Verify params after migration
	params, err := s.EmissionsKeeper().GetParams(s.Ctx())
	s.Require().NoError(err)

	// Check that all other params are preserved
	s.Require().Equal(defaultParams.Version, params.Version)
	s.Require().Equal(defaultParams.MaxSerializedMsgLength, params.MaxSerializedMsgLength)
	s.Require().True(defaultParams.MinTopicWeight.Equal(params.MinTopicWeight))
	s.Require().True(defaultParams.RequiredMinimumStake.Equal(params.RequiredMinimumStake))
	s.Require().Equal(defaultParams.RemoveStakeDelayWindow, params.RemoveStakeDelayWindow)
	s.Require().Equal(defaultParams.MinEpochLength, params.MinEpochLength)
	s.Require().True(defaultParams.BetaEntropy.Equal(params.BetaEntropy))
	s.Require().True(defaultParams.LearningRate.Equal(params.LearningRate))
	s.Require().True(defaultParams.MaxGradientThreshold.Equal(params.MaxGradientThreshold))
	s.Require().True(defaultParams.MinStakeFraction.Equal(params.MinStakeFraction))
	s.Require().Equal(defaultParams.MaxUnfulfilledWorkerRequests, params.MaxUnfulfilledWorkerRequests)
	s.Require().Equal(defaultParams.MaxUnfulfilledReputerRequests, params.MaxUnfulfilledReputerRequests)
	s.Require().True(defaultParams.TopicRewardStakeImportance.Equal(params.TopicRewardStakeImportance))
	s.Require().True(defaultParams.TopicRewardFeeRevenueImportance.Equal(params.TopicRewardFeeRevenueImportance))
	s.Require().True(defaultParams.TopicRewardAlpha.Equal(params.TopicRewardAlpha))
	s.Require().True(defaultParams.TaskRewardAlpha.Equal(params.TaskRewardAlpha))
	s.Require().True(defaultParams.ValidatorsVsAlloraPercentReward.Equal(params.ValidatorsVsAlloraPercentReward))
	s.Require().Equal(defaultParams.MaxSamplesToScaleScores, params.MaxSamplesToScaleScores)
	s.Require().Equal(defaultParams.MaxTopInferersToReward, params.MaxTopInferersToReward)
	s.Require().Equal(defaultParams.MaxTopForecastersToReward, params.MaxTopForecastersToReward)
	s.Require().Equal(defaultParams.MaxTopReputersToReward, params.MaxTopReputersToReward)
	s.Require().True(defaultParams.CreateTopicFee.Equal(params.CreateTopicFee))
	s.Require().Equal(defaultParams.GradientDescentMaxIters, params.GradientDescentMaxIters)
	s.Require().True(defaultParams.RegistrationFee.Equal(params.RegistrationFee))
	s.Require().Equal(defaultParams.DefaultPageLimit, params.DefaultPageLimit)
	s.Require().Equal(defaultParams.MaxPageLimit, params.MaxPageLimit)
	s.Require().Equal(defaultParams.MinEpochLengthRecordLimit, params.MinEpochLengthRecordLimit)
	s.Require().Equal(defaultParams.BlocksPerMonth, params.BlocksPerMonth)
	s.Require().True(defaultParams.PRewardInference.Equal(params.PRewardInference))
	s.Require().True(defaultParams.PRewardForecast.Equal(params.PRewardForecast))
	s.Require().True(defaultParams.PRewardReputer.Equal(params.PRewardReputer))
	s.Require().True(defaultParams.CRewardInference.Equal(params.CRewardInference))
	s.Require().True(defaultParams.CRewardForecast.Equal(params.CRewardForecast))
	s.Require().True(defaultParams.EpsilonReputer.Equal(params.EpsilonReputer))
	s.Require().Equal(defaultParams.HalfMaxProcessStakeRemovalsEndBlock, params.HalfMaxProcessStakeRemovalsEndBlock)
	s.Require().True(defaultParams.EpsilonSafeDiv.Equal(params.EpsilonSafeDiv))
	s.Require().True(defaultParams.DataSendingFee.Equal(params.DataSendingFee))
	s.Require().Equal(defaultParams.MaxElementsPerForecast, params.MaxElementsPerForecast)
	s.Require().Equal(defaultParams.MaxActiveTopicsPerBlock, params.MaxActiveTopicsPerBlock)
	s.Require().Equal(defaultParams.MaxStringLength, params.MaxStringLength)
	s.Require().True(defaultParams.InitialRegretQuantile.Equal(params.InitialRegretQuantile))
	s.Require().True(defaultParams.PNormSafeDiv.Equal(params.PNormSafeDiv))
	s.Require().Equal(defaultParams.GlobalWhitelistEnabled, params.GlobalWhitelistEnabled)
	s.Require().Equal(defaultParams.TopicCreatorWhitelistEnabled, params.TopicCreatorWhitelistEnabled)
	s.Require().Equal(defaultParams.MinExperiencedWorkerRegrets, params.MinExperiencedWorkerRegrets)
	s.Require().True(defaultParams.InferenceOutlierDetectionThreshold.Equal(params.InferenceOutlierDetectionThreshold))
	s.Require().True(defaultParams.InferenceOutlierDetectionAlpha.Equal(params.InferenceOutlierDetectionAlpha))
	s.Require().True(defaultParams.LambdaInitialScore.Equal(params.LambdaInitialScore))
	s.Require().Equal(defaultParams.GlobalWorkerWhitelistEnabled, params.GlobalWorkerWhitelistEnabled)
	s.Require().Equal(defaultParams.GlobalReputerWhitelistEnabled, params.GlobalReputerWhitelistEnabled)
	s.Require().Equal(defaultParams.GlobalAdminWhitelistAppended, params.GlobalAdminWhitelistAppended)
	s.Require().Equal(defaultParams.MaxWhitelistInputArrayLength, params.MaxWhitelistInputArrayLength)
	s.Require().True(defaultParams.MinWeightThresholdForStdnorm.Equal(params.MinWeightThresholdForStdnorm))
}

// TestMigrateTopics tests that CNorm is added to all existing topics
func (s *EmissionsV12MigrationTestSuite) TestMigrateTopics() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()
	keeper := s.EmissionsKeeper()

	// Create some topics first (without CNorm, as they would be before migration)
	topicMsg1 := s.MockTopicMsg()
	topicMsg2 := s.MockTopicMsg()
	topicMsg3 := s.MockTopicMsg()

	// Create topics using keeper
	s.MintTokensToAddress(s.Addrs(0), emissionstypes.DefaultParams().CreateTopicFee.MulRaw(3))

	resp1, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), topicMsg1)
	s.Require().NoError(err)
	topicId1 := resp1.TopicId

	resp2, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), topicMsg2)
	s.Require().NoError(err)
	topicId2 := resp2.TopicId

	resp3, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), topicMsg3)
	s.Require().NoError(err)
	topicId3 := resp3.TopicId

	// Old CNorm value from params
	oldCNorm := alloraMath.MustNewDecFromString("0.85")

	// Run migration
	err = v12.MigrateTopics(s.Ctx(), store, cdc, oldCNorm)
	s.Require().NoError(err)

	// Verify that all topics now have the CNorm value
	topic1, err := keeper.GetTopic(s.Ctx(), topicId1)
	s.Require().NoError(err)
	s.Require().True(topic1.CNorm.Equal(oldCNorm), "Topic 1 CNorm should be %s, got %s", oldCNorm, topic1.CNorm)

	topic2, err := keeper.GetTopic(s.Ctx(), topicId2)
	s.Require().NoError(err)
	s.Require().True(topic2.CNorm.Equal(oldCNorm), "Topic 2 CNorm should be %s, got %s", oldCNorm, topic2.CNorm)

	topic3, err := keeper.GetTopic(s.Ctx(), topicId3)
	s.Require().NoError(err)
	s.Require().True(topic3.CNorm.Equal(oldCNorm), "Topic 3 CNorm should be %s, got %s", oldCNorm, topic3.CNorm)
}

// TestFullMigration tests the complete migration from v11 to v12
func (s *EmissionsV12MigrationTestSuite) TestFullMigration() {
	storageService := s.EmissionsKeeper().GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(s.Ctx()))
	cdc := s.EmissionsKeeper().GetBinaryCodec()
	keeper := s.EmissionsKeeper()

	// Create some topics first
	topicMsg1 := s.MockTopicMsg()
	topicMsg2 := s.MockTopicMsg()

	s.MintTokensToAddress(s.Addrs(0), emissionstypes.DefaultParams().CreateTopicFee.MulRaw(2))

	resp1, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), topicMsg1)
	s.Require().NoError(err)
	topicId1 := resp1.TopicId

	resp2, err := s.EmissionsMsgServer().CreateNewTopic(s.Ctx(), topicMsg2)
	s.Require().NoError(err)
	topicId2 := resp2.TopicId

	// Set old params with CNorm
	defaultParams := emissionstypes.DefaultParams()
	oldCNorm := alloraMath.MustNewDecFromString("0.65")

	paramsOld := oldV11Types.Params{ //nolint: exhaustruct // this is an old version of the params
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
		CNorm:                               oldCNorm,
		EpsilonReputer:                      defaultParams.EpsilonReputer,
		HalfMaxProcessStakeRemovalsEndBlock: defaultParams.HalfMaxProcessStakeRemovalsEndBlock,
		EpsilonSafeDiv:                      defaultParams.EpsilonSafeDiv,
		DataSendingFee:                      defaultParams.DataSendingFee,
		MaxElementsPerForecast:              defaultParams.MaxElementsPerForecast,
		MaxActiveTopicsPerBlock:             defaultParams.MaxActiveTopicsPerBlock,
		MaxStringLength:                     defaultParams.MaxStringLength,
		InitialRegretQuantile:               defaultParams.InitialRegretQuantile,
		PNormSafeDiv:                        defaultParams.PNormSafeDiv,
		GlobalWhitelistEnabled:              defaultParams.GlobalWhitelistEnabled,
		TopicCreatorWhitelistEnabled:        defaultParams.TopicCreatorWhitelistEnabled,
		MinExperiencedWorkerRegrets:         defaultParams.MinExperiencedWorkerRegrets,
		InferenceOutlierDetectionThreshold:  defaultParams.InferenceOutlierDetectionThreshold,
		InferenceOutlierDetectionAlpha:      defaultParams.InferenceOutlierDetectionAlpha,
		LambdaInitialScore:                  defaultParams.LambdaInitialScore,
		GlobalWorkerWhitelistEnabled:        defaultParams.GlobalWorkerWhitelistEnabled,
		GlobalReputerWhitelistEnabled:       defaultParams.GlobalReputerWhitelistEnabled,
		GlobalAdminWhitelistAppended:        defaultParams.GlobalAdminWhitelistAppended,
		MaxWhitelistInputArrayLength:        defaultParams.MaxWhitelistInputArrayLength,
		MinWeightThresholdForStdnorm:        defaultParams.MinWeightThresholdForStdnorm,
	}

	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&paramsOld))

	// Run full migration
	err = v12.MigrateStore(s.Ctx(), *keeper)
	s.Require().NoError(err)

	// Verify params after migration (CNorm should be removed)
	params, err := keeper.GetParams(s.Ctx())
	s.Require().NoError(err)
	s.Require().Equal(defaultParams.Version, params.Version)

	// Verify topics have CNorm set to the old global value
	topic1, err := keeper.GetTopic(s.Ctx(), topicId1)
	s.Require().NoError(err)
	s.Require().True(topic1.CNorm.Equal(oldCNorm), "Topic 1 CNorm should be %s, got %s", oldCNorm, topic1.CNorm)

	topic2, err := keeper.GetTopic(s.Ctx(), topicId2)
	s.Require().NoError(err)
	s.Require().True(topic2.CNorm.Equal(oldCNorm), "Topic 2 CNorm should be %s, got %s", oldCNorm, topic2.CNorm)
}
