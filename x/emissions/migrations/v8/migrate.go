package v8

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	oldV7Types "github.com/allora-network/allora-chain/x/emissions/migrations/v8/oldtypes"
	v8Types "github.com/allora-network/allora-chain/x/emissions/migrations/v9/oldtypes"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
)

// MigrateStore migrates the store from version 6 to version 7
// It does the following:
// - Migrate params to add and set MinWeightThresholdForStdnorm
// - Adds new empty stores
// - Adds queries for this empty stores
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 7 TO VERSION 8")
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 7 TO VERSION 8")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	ctx.Logger().Info("MIGRATING PARAMS FROM VERSION 7 TO VERSION 8")
	// This also flips on global and topic creator whitelists
	if err := MigrateParams(ctx, store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateParams() FROM VERSION 7 TO VERSION 8")
		return err
	}

	ctx.Logger().Info("MIGRATING EMISSIONS MODULE FROM VERSION 7 TO VERSION 8 COMPLETE")
	return nil
}

// Migrate params for this new version
// The changes are the addition of GlobalWhitelistEnabled, TopicCreatorWhitelistEnabled
func MigrateParams(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	oldParams := oldV7Types.Params{} //nolint:exhaustruct // empty struct used by cosmos-sdk Unmarshal below
	oldParamsBytes := store.Get(emissionstypes.ParamsKey)
	if oldParamsBytes == nil {
		return errorsmod.Wrapf(emissionstypes.ErrNotFound, "old parameters not found")
	}
	err := proto.Unmarshal(oldParamsBytes, &oldParams)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to unmarshal old parameters")
	}

	defaultParams := emissionstypes.DefaultParams()

	// DIFFERENCE BETWEEN OLD PARAMS AND NEW PARAMS:
	// ADDED:
	//       MinWeightThresholdForStdnorm
	newParams := v8Types.Params{ //nolint:exhaustruct
		Version:                             oldParams.Version,
		MaxSerializedMsgLength:              oldParams.MaxSerializedMsgLength,
		MinTopicWeight:                      oldParams.MinTopicWeight,
		RequiredMinimumStake:                oldParams.RequiredMinimumStake,
		RemoveStakeDelayWindow:              oldParams.RemoveStakeDelayWindow,
		MinEpochLength:                      oldParams.MinEpochLength,
		BetaEntropy:                         oldParams.BetaEntropy,
		LearningRate:                        oldParams.LearningRate,
		MaxGradientThreshold:                oldParams.MaxGradientThreshold,
		MinStakeFraction:                    oldParams.MinStakeFraction,
		MaxUnfulfilledWorkerRequests:        oldParams.MaxUnfulfilledWorkerRequests,
		MaxUnfulfilledReputerRequests:       oldParams.MaxUnfulfilledReputerRequests,
		TopicRewardStakeImportance:          oldParams.TopicRewardStakeImportance,
		TopicRewardFeeRevenueImportance:     oldParams.TopicRewardFeeRevenueImportance,
		TopicRewardAlpha:                    oldParams.TopicRewardAlpha,
		TaskRewardAlpha:                     oldParams.TaskRewardAlpha,
		ValidatorsVsAlloraPercentReward:     oldParams.ValidatorsVsAlloraPercentReward,
		MaxSamplesToScaleScores:             oldParams.MaxSamplesToScaleScores,
		MaxTopInferersToReward:              oldParams.MaxTopInferersToReward,
		MaxTopForecastersToReward:           oldParams.MaxTopForecastersToReward,
		MaxTopReputersToReward:              oldParams.MaxTopReputersToReward,
		CreateTopicFee:                      oldParams.CreateTopicFee,
		GradientDescentMaxIters:             oldParams.GradientDescentMaxIters,
		RegistrationFee:                     oldParams.RegistrationFee,
		DefaultPageLimit:                    oldParams.DefaultPageLimit,
		MaxPageLimit:                        oldParams.MaxPageLimit,
		MinEpochLengthRecordLimit:           oldParams.MinEpochLengthRecordLimit,
		BlocksPerMonth:                      oldParams.BlocksPerMonth,
		PRewardInference:                    oldParams.PRewardInference,
		PRewardForecast:                     oldParams.PRewardForecast,
		PRewardReputer:                      oldParams.PRewardReputer,
		CRewardInference:                    oldParams.CRewardInference,
		CRewardForecast:                     oldParams.CRewardForecast,
		CNorm:                               oldParams.CNorm,
		EpsilonReputer:                      oldParams.EpsilonReputer,
		HalfMaxProcessStakeRemovalsEndBlock: oldParams.HalfMaxProcessStakeRemovalsEndBlock,
		EpsilonSafeDiv:                      oldParams.EpsilonSafeDiv,
		DataSendingFee:                      oldParams.DataSendingFee,
		MaxElementsPerForecast:              oldParams.MaxElementsPerForecast,
		MaxActiveTopicsPerBlock:             oldParams.MaxActiveTopicsPerBlock,
		MaxStringLength:                     oldParams.MaxStringLength,
		InitialRegretQuantile:               oldParams.InitialRegretQuantile,
		PNormSafeDiv:                        oldParams.PNormSafeDiv,
		GlobalWhitelistEnabled:              oldParams.GlobalWhitelistEnabled,
		TopicCreatorWhitelistEnabled:        oldParams.TopicCreatorWhitelistEnabled,
		MinExperiencedWorkerRegrets:         oldParams.MinExperiencedWorkerRegrets,
		InferenceOutlierDetectionThreshold:  oldParams.InferenceOutlierDetectionThreshold,
		InferenceOutlierDetectionAlpha:      oldParams.InferenceOutlierDetectionAlpha,
		LambdaInitialScore:                  oldParams.LambdaInitialScore,
		GlobalWorkerWhitelistEnabled:        oldParams.GlobalWorkerWhitelistEnabled,
		GlobalReputerWhitelistEnabled:       oldParams.GlobalReputerWhitelistEnabled,
		GlobalAdminWhitelistAppended:        oldParams.GlobalAdminWhitelistAppended,
		MaxWhitelistInputArrayLength:        oldParams.MaxWhitelistInputArrayLength,
		// NEW PARAMS
		MinWeightThresholdForStdnorm: defaultParams.MinWeightThresholdForStdnorm,
	}

	ctx.Logger().Info("MIGRATED PARAMS", "params", newParams)
	store.Delete(emissionstypes.ParamsKey)
	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&newParams))
	return nil
}
