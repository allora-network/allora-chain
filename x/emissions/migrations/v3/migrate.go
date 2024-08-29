package v3

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	oldtypes "github.com/allora-network/allora-chain/x/emissions/migrations/v3/types"
	types "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
)

func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 2 TO VERSION 3")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	ctx.Logger().Info("INVOKING MIGRATION HANDLER MigrateParams() FROM VERSION 2 TO VERSION 3")
	if err := MigrateParams(store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateParams() FROM VERSION 2 TO VERSION 3")
		return err
	}

	ctx.Logger().Info("INVOKING MIGRATION HANDLER MigrateTopics() FROM VERSION 2 TO VERSION 3")
	if err := MigrateTopics(store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateTopics() FROM VERSION 2 TO VERSION 3")
		return err
	}

	ctx.Logger().Info("INVOKING MIGRATION HANDLER ResetMapsWithNonNumericValues() FROM VERSION 2 TO VERSION 3")
	ResetMapsWithNonNumericValues(store, cdc)

	return nil
}

func MigrateParams(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	oldParams := oldtypes.Params{}
	oldParamsBytes := store.Get(types.ParamsKey)
	if oldParamsBytes == nil {
		return errors.Wrapf(types.ErrNotFound, "old parameters not found")
	}
	err := proto.Unmarshal(oldParamsBytes, &oldParams)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal old parameters")
	}

	defaultParams := types.DefaultParams()

	// DIFFERENCE BETWEEN OLD PARAMS AND NEW PARAMS:
	// ADDED:
	//      MaxElementsPerForecast
	// REMOVED:
	// 		MinEffectiveTopicRevenue
	//      TopicFeeRevenueDecayRate
	//      MaxRetriesToFulfilNoncesWorker
	// 		MaxRetriesToFulfilNoncesReputer
	newParams := types.Params{
		Version:                             oldParams.Version,
		MaxSerializedMsgLength:              oldParams.MaxSerializedMsgLength,
		MinTopicWeight:                      oldParams.MinTopicWeight,
		MaxTopicsPerBlock:                   oldParams.MaxTopicsPerBlock,
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
		// NEW PARAMS
		MaxElementsPerForecast: defaultParams.MaxElementsPerForecast,
	}

	store.Delete(types.ParamsKey)
	store.Set(types.ParamsKey, cdc.MustMarshal(&newParams))
	return nil
}

func MigrateTopics(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	topicStore := prefix.NewStore(store, types.TopicsKey)
	iterator := topicStore.Iterator(nil, nil)

	valueToAdd := make(map[string]types.Topic, 0)
	for ; iterator.Valid(); iterator.Next() {
		var oldMsg oldtypes.Topic
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		newMsg := types.Topic{
			Id:             oldMsg.Id,
			Creator:        oldMsg.Creator,
			Metadata:       oldMsg.Metadata,
			LossMethod:     oldMsg.LossMethod,
			EpochLastEnded: oldMsg.EpochLastEnded, // Add default value
			EpochLength:    oldMsg.EpochLength,
			GroundTruthLag: oldMsg.GroundTruthLag,
			PNorm:          oldMsg.PNorm,
			AlphaRegret:    oldMsg.AlphaRegret,
			AllowNegative:  oldMsg.AllowNegative,
			Epsilon:        alloraMath.MustNewDecFromString("0.01"),
			// InitialRegret is being reset to account for NaNs that were previously stored due to insufficient validation
			InitialRegret:          alloraMath.MustNewDecFromString("0"),
			WorkerSubmissionWindow: oldMsg.WorkerSubmissionWindow,
			// These are new fields
			MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
			ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.25"),
			ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.25"),
			ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.25"),
		}

		valueToAdd[string(iterator.Key())] = newMsg
	}
	iterator.Close()

	for key, value := range valueToAdd {
		topicStore.Set([]byte(key), cdc.MustMarshal(&value))
	}

	return nil
}

var maxPageSize = uint64(10000)

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

func ResetMapsWithNonNumericValues(store storetypes.KVStore, cdc codec.BinaryCodec) {
	safelyClearWholeMap(store, types.InferenceScoresKey)
	safelyClearWholeMap(store, types.ForecastScoresKey)
	safelyClearWholeMap(store, types.ReputerScoresKey)
	safelyClearWholeMap(store, types.InfererScoreEmasKey)
	safelyClearWholeMap(store, types.ForecasterScoreEmasKey)
	safelyClearWholeMap(store, types.ReputerScoreEmasKey)
	safelyClearWholeMap(store, types.AllLossBundlesKey)
	safelyClearWholeMap(store, types.NetworkLossBundlesKey)
	safelyClearWholeMap(store, types.InfererNetworkRegretsKey)
	safelyClearWholeMap(store, types.ForecasterNetworkRegretsKey)
	safelyClearWholeMap(store, types.OneInForecasterNetworkRegretsKey)
	safelyClearWholeMap(store, types.LatestNaiveInfererNetworkRegretsKey)
	safelyClearWholeMap(store, types.LatestOneOutInfererInfererNetworkRegretsKey)
	safelyClearWholeMap(store, types.LatestOneOutInfererForecasterNetworkRegretsKey)
	safelyClearWholeMap(store, types.LatestOneOutForecasterInfererNetworkRegretsKey)
	safelyClearWholeMap(store, types.LatestOneOutForecasterForecasterNetworkRegretsKey)
}
