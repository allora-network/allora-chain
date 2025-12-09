package v12

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	oldV11Types "github.com/allora-network/allora-chain/x/emissions/migrations/v12/oldtypes"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
)

// MigrateStore migrates the store from version 11 to version 12
// It does the following:
// - Migrate params to remove CNorm (it's now per-topic)
// - Migrate all topics to add CNorm field with the old global value
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 11 TO VERSION 12")
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 11 TO VERSION 12")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	ctx.Logger().Info("GETTING OLD CNORM VALUE FROM PARAMS")
	oldCNorm, err := getOldCNormFromParams(store)
	if err != nil {
		ctx.Logger().Error("ERROR GETTING OLD CNORM VALUE FROM PARAMS")
		return err
	}
	ctx.Logger().Info("OLD CNORM VALUE", "cNorm", oldCNorm)

	ctx.Logger().Info("MIGRATING PARAMS FROM VERSION 11 TO VERSION 12")
	if err := MigrateParams(store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateParams() FROM VERSION 11 TO VERSION 12")
		return err
	}

	ctx.Logger().Info("MIGRATING TOPICS FROM VERSION 11 TO VERSION 12")
	if err := MigrateTopics(ctx, store, cdc, oldCNorm); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateTopics() FROM VERSION 11 TO VERSION 12")
		return err
	}

	ctx.Logger().Info("MIGRATING EMISSIONS MODULE FROM VERSION 11 TO VERSION 12 COMPLETE")
	return nil
}

// getOldCNormFromParams extracts the CNorm value from the old params before migration
func getOldCNormFromParams(store storetypes.KVStore) (alloraMath.Dec, error) {
	oldParams := oldV11Types.Params{} //nolint: exhaustruct
	oldParamsBytes := store.Get(emissionstypes.ParamsKey)
	if oldParamsBytes == nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(emissionstypes.ErrNotFound, "old parameters not found")
	}
	err := proto.Unmarshal(oldParamsBytes, &oldParams)
	if err != nil {
		return alloraMath.Dec{}, errorsmod.Wrapf(err, "failed to unmarshal old parameters")
	}
	return oldParams.CNorm, nil
}

// MigrateParams migrates params by removing the CNorm field
// CNorm is now stored per-topic instead of globally
func MigrateParams(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	oldParams := oldV11Types.Params{} //nolint: exhaustruct
	oldParamsBytes := store.Get(emissionstypes.ParamsKey)
	if oldParamsBytes == nil {
		return errorsmod.Wrapf(emissionstypes.ErrNotFound, "old parameters not found")
	}
	err := proto.Unmarshal(oldParamsBytes, &oldParams)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to unmarshal old parameters")
	}

	// DIFFERENCE BETWEEN OLD PARAMS AND NEW PARAMS:
	// REMOVED: CNorm
	newParams := emissionstypes.Params{ //nolint: exhaustruct
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
		MinWeightThresholdForStdnorm:        oldParams.MinWeightThresholdForStdnorm,
	}

	store.Delete(emissionstypes.ParamsKey)
	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&newParams))
	return nil
}

// MigrateTopics iterates through all topics and adds CNorm field with the old global value
func MigrateTopics(
	ctx sdk.Context,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	oldCNorm alloraMath.Dec,
) error {
	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	iterator := topicStore.Iterator(nil, nil)
	defer iterator.Close()

	ctx.Logger().Info("MIGRATION V12: Migrating topics to add CNorm", "cNorm", oldCNorm)

	topicsToChange := make(map[string]emissionstypes.Topic, 0)
	for ; iterator.Valid(); iterator.Next() {
		var topic emissionstypes.Topic
		err := proto.Unmarshal(iterator.Value(), &topic)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal topic")
		}

		ctx.Logger().Debug("MIGRATION V12: Updating topic", "topicId", topic.Id)

		topic.CNorm = oldCNorm

		topicsToChange[string(iterator.Key())] = topic
	}
	_ = iterator.Close()

	for key, topic := range topicsToChange {
		topicStore.Set([]byte(key), cdc.MustMarshal(&topic))
		ctx.Logger().Debug("MIGRATION V12: Updated topic with CNorm", "topicId", topic.Id, "cNorm", oldCNorm)
	}

	ctx.Logger().Info("MIGRATION V12: Topics migration complete", "topicsUpdated", len(topicsToChange))
	return nil
}
