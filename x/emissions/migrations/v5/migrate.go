package v5

import (
	"encoding/binary"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/utils/migutils"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	oldV4Types "github.com/allora-network/allora-chain/x/emissions/migrations/v5/oldtypes"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
)

const maxPageSize = uint64(10000)

// MigrateStore migrates the store from version 4 to version 5
// it does the following:
// - migrates topics to set initial regret
// - Deletes the contents of previous quantile score maps
// - Sets the sumTotalPreviousTopicWeights to the sum of previousTopicWeight
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 4 TO VERSION 5")
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 4 TO VERSION 5")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	ctx.Logger().Info("MIGRATING PARAMS FROM VERSION 4 TO VERSION 5")
	if err := MigrateParams(store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateParams() FROM VERSION 4 TO VERSION 5")
		return err
	}

	ctx.Logger().Info("MIGRATING TOPICS FROM VERSION 4 TO VERSION 5")
	if err := MigrateTopics(ctx, store, cdc, emissionsKeeper); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateTopics() FROM VERSION 4 TO VERSION 5")
		return err
	}

	ctx.Logger().Info("INVOKING ResetMapsWithNonNumericValues() IN MIGRATION FROM VERSION 4 TO VERSION 5")
	if err := ResetMapsWithNonNumericValues(ctx, store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER ResetMapsWithNonNumericValues() FROM VERSION 4 TO VERSION 5")
		return err
	}

	ctx.Logger().Info("MIGRATING EMISSIONS MODULE FROM VERSION 4 TO VERSION 5 COMPLETE")
	return nil
}

// migrate params for this new version
// the changes are the addition of InitialRegretQuantile,PNormSafeDiv
func MigrateParams(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	oldParams := oldV4Types.Params{} //nolint: exhaustruct // empty struct used by cosmos-sdk Unmarshal below
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
	//      InitialRegretQuantile, PNormSafeDiv
	newParams := emissionstypes.Params{
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
		// NEW PARAMS
		InitialRegretQuantile: defaultParams.InitialRegretQuantile,
		PNormSafeDiv:          defaultParams.PNormSafeDiv,
	}

	store.Delete(emissionstypes.ParamsKey)
	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&newParams))
	return nil
}

// migrate topics for this new version
// iterate through all topics, keep all the old values of these topics
// need to initialize InitialRegret to zero as default
// and set allownegative as false.
func MigrateTopics(
	ctx sdk.Context,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	emissionsKeeper keeper.Keeper,
) error {
	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)

	nextTopicId, err := emissionsKeeper.GetNextTopicId(ctx)
	if err != nil {
		return err
	}
	// iterate all topics to migrate using collections.go api
	topicsToChange := make(map[string]emissionstypes.Topic, 0)
	sumTotalPreviousTopicWeights := alloraMath.ZeroDec()
	for id := uint64(1); id < nextTopicId; id++ {
		idByte := make([]byte, 8)
		binary.BigEndian.PutUint64(idByte, id)
		ctx.Logger().Info(fmt.Sprintf("MIGRATION V5: Updating topic:%d", id))
		topic, err := emissionsKeeper.GetTopic(ctx, id)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to get topic")
		}
		newTopic := copyTopic(topic)
		// fix egregious values
		newTopic.InitialRegret = alloraMath.MustNewDecFromString("0")
		// was wrongly set to true for existing topics
		newTopic.AllowNegative = false
		topicsToChange[string(idByte)] = newTopic

		// Initialization of sumTotalPreviousTopicWeights, summing up all active topic weights
		isActive, err := emissionsKeeper.IsTopicActive(ctx, topic.Id)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to get topic")
		}
		if isActive {
			topicWeight, _, err := emissionsKeeper.GetPreviousTopicWeight(ctx, topic.Id)
			if err != nil {
				return errorsmod.Wrapf(err, "failed to get topic weight")
			}
			sumTotalPreviousTopicWeights, err = sumTotalPreviousTopicWeights.Add(topicWeight)
			if err != nil {
				return errorsmod.Wrapf(err, "failed to add topic weight")
			}
		} else {
			ctx.Logger().Debug("MIGRATION V5: Topic is not active, skipping weight - topic", topic.Id)
		}
	}
	ctx.Logger().Debug("MIGRATION V5: Setting modified topics")
	for key, value := range topicsToChange {
		topicStore.Set([]byte(key), cdc.MustMarshal(&value))
	}
	ctx.Logger().Debug("MIGRATION V5: Updating total sum previous topic weights, sum: ", sumTotalPreviousTopicWeights)
	err = emissionsKeeper.SetTotalSumPreviousTopicWeights(ctx, sumTotalPreviousTopicWeights)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to set total sum previous topic weights")
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
		ctx.Logger().Info(fmt.Sprintf("MIGRATION V5: RESETTING %v MAP", prefix.name))
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
