package v4

import (
	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/utils/migutils"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	oldV2Types "github.com/allora-network/allora-chain/x/emissions/migrations/v3/oldtypes"
	oldV3Types "github.com/allora-network/allora-chain/x/emissions/migrations/v4/oldtypes"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

const maxPageSize = uint64(10000)

// MigrateStore migrates the store from version 3 to version 4
// it does the following:
// - migrates params
// - migrates topics
// - Deletes the contents of several maps that had NaN values in them
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 3 TO VERSION 4")
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 3 TO VERSION 4")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	ctx.Logger().Info("MIGRATING PARAMS FROM VERSION 3 TO VERSION 4")
	if err := MigrateParams(store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateParams() FROM VERSION 3 TO VERSION 4")
		return err
	}

	ctx.Logger().Info("MIGRATING TOPICS FROM VERSION 3 TO VERSION 4")
	if err := MigrateTopics(ctx, store, cdc, emissionsKeeper); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateTopics() FROM VERSION 3 TO VERSION 4")
		return err
	}

	ctx.Logger().Info("INVOKING MIGRATION HANDLER ResetMapsWithNonNumericValues() FROM VERSION 3 TO VERSION 4")
	err := ResetMapsWithNonNumericValues(ctx, store, cdc)
	if err != nil {
		ctx.Logger().Error("ERROR RESETTING MAPS WITH NON NUMERIC VALUES: %v", err)
		return err
	}

	ctx.Logger().Info("MIGRATING EMISSIONS MODULE FROM VERSION 3 TO VERSION 4 COMPLETE")
	return nil
}

// migrate params for this new version
// the only change is the addition of MaxStringLength
func MigrateParams(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	oldParams := oldV3Types.Params{} //nolint: exhaustruct // populated in unmarshal below
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
	//      MaxStringLength
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
		CNorm:                               oldParams.CNorm,
		EpsilonReputer:                      oldParams.EpsilonReputer,
		HalfMaxProcessStakeRemovalsEndBlock: oldParams.HalfMaxProcessStakeRemovalsEndBlock,
		EpsilonSafeDiv:                      oldParams.EpsilonSafeDiv,
		DataSendingFee:                      oldParams.DataSendingFee,
		MaxElementsPerForecast:              oldParams.MaxElementsPerForecast,
		MaxActiveTopicsPerBlock:             oldParams.MaxActiveTopicsPerBlock,
		// NEW PARAMS
		MaxStringLength: defaultParams.MaxStringLength,
	}

	store.Delete(emissionstypes.ParamsKey)
	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&newParams))
	return nil
}

// migrate topics for this new version
// iterate through all topics, keep all the old values of these topics
// if a topic has a NaN value for InitialRegret, set the value to zero
// if the topic doesn't have a value for MeritSortitionAlpha,
// ActiveInfererQuantile, ActiveForecasterQuantile, or ActiveReputerQuantile,
// set those values to the default.
func MigrateTopics(
	ctx sdk.Context,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	emissionsKeeper keeper.Keeper,
) error {
	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	iterator := topicStore.Iterator(nil, nil)
	defer iterator.Close()

	// iterate over the topic store by manually checking prefixes
	// to make sure we absolutely do not miss any topics
	// (as opposed to using collections.go API)
	topicsToChange := make(map[string]emissionstypes.Topic, 0)
	for ; iterator.Valid(); iterator.Next() {
		iterator.Key()
		var oldMsg oldV2Types.Topic
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal old topic")
		}
		// now get the topic from the collections.go
		// API, to check if the fields in the topic exist
		topic, err := emissionsKeeper.GetTopic(ctx, oldMsg.Id)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to get topic")
		}
		newTopic := copyTopic(topic)
		// fix NaN "poison" values
		if topic.InitialRegret.IsNaN() || !topic.InitialRegret.IsFinite() {
			newTopic.InitialRegret = alloraMath.MustNewDecFromString("0")
		}
		// new fields - uninitialized will appear as zero values from collections.go
		if topic.MeritSortitionAlpha.IsZero() {
			newTopic.MeritSortitionAlpha = alloraMath.MustNewDecFromString("0.1")
		}
		if topic.ActiveInfererQuantile.IsZero() {
			newTopic.ActiveInfererQuantile = alloraMath.MustNewDecFromString("0.25")
		}
		if topic.ActiveForecasterQuantile.IsZero() {
			newTopic.ActiveForecasterQuantile = alloraMath.MustNewDecFromString("0.25")
		}
		if topic.ActiveReputerQuantile.IsZero() {
			newTopic.ActiveReputerQuantile = alloraMath.MustNewDecFromString("0.25")
		}
		topicsToChange[string(iterator.Key())] = newTopic
	}
	_ = iterator.Close()
	for key, value := range topicsToChange {
		topicStore.Set([]byte(key), cdc.MustMarshal(&value))
	}

	return nil
}

// Clear out poison NaN values on different inferences, scores etc
func ResetMapsWithNonNumericValues(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	prefixes := []struct {
		prefix collections.Prefix
		name   string
	}{
		{emissionstypes.InferenceScoresKey, "InferenceScores"},
		{emissionstypes.ForecastScoresKey, "ForecastScores"},
		{emissionstypes.ReputerScoresKey, "ReputerScores"},
		{emissionstypes.InfererScoreEmasKey, "InfererScoreEmas"},
		{emissionstypes.ForecasterScoreEmasKey, "ForecasterScoreEmas"},
		{emissionstypes.ReputerScoreEmasKey, "ReputerScoreEmas"},
		{emissionstypes.AllLossBundlesKey, "AllLossBundles"},
		{emissionstypes.NetworkLossBundlesKey, "NetworkLossBundles"},
		{emissionstypes.InfererNetworkRegretsKey, "InfererNetworkRegrets"},
		{emissionstypes.ForecasterNetworkRegretsKey, "ForecasterNetworkRegrets"},
		{emissionstypes.OneInForecasterNetworkRegretsKey, "OneInForecasterNetworkRegrets"},
		{emissionstypes.LatestNaiveInfererNetworkRegretsKey, "LatestNaiveInfererNetworkRegrets"},
		{emissionstypes.LatestOneOutInfererInfererNetworkRegretsKey, "LatestOneOutInfererInfererNetworkRegrets"},
		{emissionstypes.LatestOneOutInfererForecasterNetworkRegretsKey, "LatestOneOutInfererForecasterNetworkRegrets"},
		{emissionstypes.LatestOneOutForecasterInfererNetworkRegretsKey, "LatestOneOutForecasterInfererNetworkRegrets"},
		{emissionstypes.LatestOneOutForecasterForecasterNetworkRegretsKey, "LatestOneOutForecasterForecasterNetworkRegrets"},
		{emissionstypes.ReputerListeningCoefficientKey, "ReputerListeningCoefficient"},
		{emissionstypes.PreviousTopicQuantileInfererScoreEmaKey, "PreviousTopicQuantileInfererScoreEma"},
		{emissionstypes.PreviousTopicQuantileForecasterScoreEmaKey, "PreviousTopicQuantileForecasterScoreEma"},
		{emissionstypes.PreviousTopicQuantileReputerScoreEmaKey, "PreviousTopicQuantileReputerScoreEma"},
		{emissionstypes.PreviousInferenceRewardFractionKey, "PreviousInferenceRewardFraction"},
		{emissionstypes.PreviousForecastRewardFractionKey, "PreviousForecastRewardFraction"},
		{emissionstypes.PreviousReputerRewardFractionKey, "PreviousReputerRewardFraction"},
	}

	for _, prefix := range prefixes {
		ctx.Logger().Info("MIGRATION V4: RESETTING %v MAP", prefix.name)
		err := migutils.SafelyClearWholeMap(ctx, store, prefix.prefix, maxPageSize)
		if err != nil {
			return err
		}
	}
	return nil
}

// copyTopic duplicates a topic into a new struct
func copyTopic(original emissionstypes.Topic) emissionstypes.Topic {
	return emissionstypes.Topic{
		Id:                       original.Id,
		Creator:                  original.Creator,
		Metadata:                 original.Metadata,
		LossMethod:               original.LossMethod,
		EpochLastEnded:           original.EpochLastEnded,
		EpochLength:              original.EpochLength,
		GroundTruthLag:           original.GroundTruthLag,
		PNorm:                    original.PNorm,
		AlphaRegret:              original.AlphaRegret,
		AllowNegative:            original.AllowNegative,
		Epsilon:                  original.Epsilon,
		InitialRegret:            original.InitialRegret,
		WorkerSubmissionWindow:   original.WorkerSubmissionWindow,
		MeritSortitionAlpha:      original.MeritSortitionAlpha,
		ActiveInfererQuantile:    original.ActiveInfererQuantile,
		ActiveForecasterQuantile: original.ActiveForecasterQuantile,
		ActiveReputerQuantile:    original.ActiveReputerQuantile,
	}
}
