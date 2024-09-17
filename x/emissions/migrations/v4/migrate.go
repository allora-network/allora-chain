package v4

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	oldV2Types "github.com/allora-network/allora-chain/x/emissions/migrations/v3/oldtypes"
	oldV3Types "github.com/allora-network/allora-chain/x/emissions/migrations/v4/oldtypes"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
)

const maxPageSize = uint64(10000)

// MigrateStore migrates the store from version 3 to version 4
// it does the following:
// - migrates params
// - migrates topics
// - Deletes the contents of several maps that had NaN values in them
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	fmt.Println("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 3 TO VERSION 4")
	fmt.Println("MIGRATING STORE FROM VERSION 3 TO VERSION 4")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	fmt.Println("MIGRATING PARAMS FROM VERSION 3 TO VERSION 4")
	if err := MigrateParams(store, cdc); err != nil {
		fmt.Println("ERROR INVOKING MIGRATION HANDLER MigrateParams() FROM VERSION 3 TO VERSION 4")
		return err
	}

	fmt.Println("MIGRATING TOPICS FROM VERSION 3 TO VERSION 4")
	if err := MigrateTopics(ctx, store, cdc, emissionsKeeper); err != nil {
		fmt.Println("ERROR INVOKING MIGRATION HANDLER MigrateTopics() FROM VERSION 3 TO VERSION 4")
		return err
	}

	fmt.Println("INVOKING MIGRATION HANDLER ResetMapsWithNonNumericValues() FROM VERSION 3 TO VERSION 4")
	ResetMapsWithNonNumericValues(ctx, store, cdc)

	fmt.Println("MIGRATING EMISSIONS MODULE FROM VERSION 3 TO VERSION 4 COMPLETE")
	return nil
}

// migrate params for this new version
// the only change is the addition of MaxStringLength
func MigrateParams(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	oldParams := oldV3Types.Params{}
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

// Deletes all keys in the store with the given keyPrefix `maxPageSize` keys at a time
func safelyClearWholeMap(store storetypes.KVStore, keyPrefix []byte) {
	s := prefix.NewStore(store, keyPrefix)

	// Loop until all keys are deleted.
	// Unbounded not best practice but we are sure that the number of keys will be limited
	// and not deleting all keys means "poison" will remain in the store.
	for {
		// Gather keys to eventually delete
		iterator := s.Iterator(nil, nil)
		keysToDelete := make([][]byte, 0)
		count := uint64(0)
		for ; iterator.Valid(); iterator.Next() {
			if count >= maxPageSize {
				break
			}

			keysToDelete = append(keysToDelete, iterator.Key())
			count++
		}
		iterator.Close()

		// If no keys to delete, break => Exit whole function
		if len(keysToDelete) == 0 {
			break
		}

		// Delete the keys
		for _, key := range keysToDelete {
			s.Delete(key)
		}
	}
}

// Clear out poison NaN values on different inferences, scores etc
func ResetMapsWithNonNumericValues(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) {
	fmt.Println("MIGRATION V4: RESETTING infererScoresByBlock MAP")
	safelyClearWholeMap(store, emissionstypes.InferenceScoresKey)
	fmt.Println("MIGRATION V4: RESETTING forecasterScoresByBlock MAP")
	safelyClearWholeMap(store, emissionstypes.ForecastScoresKey)
	fmt.Println("MIGRATION V4: RESETTING reputerScoresByBlock MAP")
	safelyClearWholeMap(store, emissionstypes.ReputerScoresKey)
	fmt.Println("MIGRATION V4: RESETTING infererScoreEmas MAP")
	safelyClearWholeMap(store, emissionstypes.InfererScoreEmasKey)
	fmt.Println("MIGRATION V4: RESETTING forecasterScoreEmas MAP")
	safelyClearWholeMap(store, emissionstypes.ForecasterScoreEmasKey)
	fmt.Println("MIGRATION V4: RESETTING reputerScoreEmas MAP")
	safelyClearWholeMap(store, emissionstypes.ReputerScoreEmasKey)
	fmt.Println("MIGRATION V4: RESETTING allLossBundles MAP")
	safelyClearWholeMap(store, emissionstypes.AllLossBundlesKey)
	fmt.Println("MIGRATION V4: RESETTING networkLossBundles MAP")
	safelyClearWholeMap(store, emissionstypes.NetworkLossBundlesKey)
	fmt.Println("MIGRATION V4: RESETTING latestInfererNetworkRegrets MAP")
	safelyClearWholeMap(store, emissionstypes.InfererNetworkRegretsKey)
	fmt.Println("MIGRATION V4: RESETTING latestForecasterNetworkRegrets MAP")
	safelyClearWholeMap(store, emissionstypes.ForecasterNetworkRegretsKey)
	fmt.Println("MIGRATION V4: RESETTING latestOneInForecasterNetworkRegrets MAP")
	safelyClearWholeMap(store, emissionstypes.OneInForecasterNetworkRegretsKey)
	fmt.Println("MIGRATION V4: RESETTING latestNaiveInfererNetworkRegrets MAP")
	safelyClearWholeMap(store, emissionstypes.LatestNaiveInfererNetworkRegretsKey)
	fmt.Println("MIGRATION V4: RESETTING latestOneOutInfererInfererNetworkRegrets MAP")
	safelyClearWholeMap(store, emissionstypes.LatestOneOutInfererInfererNetworkRegretsKey)
	fmt.Println("MIGRATION V4: RESETTING latestOneOutInfererForecasterNetworkRegrets MAP")
	safelyClearWholeMap(store, emissionstypes.LatestOneOutInfererForecasterNetworkRegretsKey)
	fmt.Println("MIGRATION V4: RESETTING latestOneOutForecasterInfererNetworkRegrets MAP")
	safelyClearWholeMap(store, emissionstypes.LatestOneOutForecasterInfererNetworkRegretsKey)
	fmt.Println("MIGRATION V4: RESETTING latestOneOutForecasterForecasterNetworkRegrets MAP")
	safelyClearWholeMap(store, emissionstypes.LatestOneOutForecasterForecasterNetworkRegretsKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING ReputerListeningCoefficientKey MAP")
	safelyClearWholeMap(store, emissionstypes.ReputerListeningCoefficientKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING PreviousTopicQuantileInfererScoreEmaKey MAP")
	safelyClearWholeMap(store, emissionstypes.PreviousTopicQuantileInfererScoreEmaKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING PreviousTopicQuantileForecasterScoreEmaKey MAP")
	safelyClearWholeMap(store, emissionstypes.PreviousTopicQuantileForecasterScoreEmaKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING PreviousTopicQuantileReputerScoreEmaKey MAP")
	safelyClearWholeMap(store, emissionstypes.PreviousTopicQuantileReputerScoreEmaKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING PreviousInferenceRewardFractionKey MAP")
	safelyClearWholeMap(store, emissionstypes.PreviousInferenceRewardFractionKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING PreviousForecastRewardFractionKey MAP")
	safelyClearWholeMap(store, emissionstypes.PreviousForecastRewardFractionKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING PreviousReputerRewardFractionKey MAP")
	safelyClearWholeMap(store, emissionstypes.PreviousReputerRewardFractionKey)
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
