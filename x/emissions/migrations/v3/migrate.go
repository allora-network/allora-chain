package v3

import (
	"encoding/binary"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
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

const maxPageSize = uint64(10000)

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
	if err := MigrateTopics(ctx, store, cdc, emissionsKeeper); err != nil {
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
		return errorsmod.Wrapf(types.ErrNotFound, "old parameters not found")
	}
	err := proto.Unmarshal(oldParamsBytes, &oldParams)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to unmarshal old parameters")
	}

	defaultParams := types.DefaultParams()

	// DIFFERENCE BETWEEN OLD PARAMS AND NEW PARAMS:
	// ADDED:
	//      MaxElementsPerForecast
	//      MaxActiveTopicsPerBlock
	// REMOVED:
	//      MinEffectiveTopicRevenue
	//      TopicFeeRevenueDecayRate
	//      MaxRetriesToFulfilNoncesWorker
	//      MaxRetriesToFulfilNoncesReputer
	//      MaxTopicsPerBlock
	newParams := types.Params{
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
		// NEW PARAMS
		MaxElementsPerForecast:  defaultParams.MaxElementsPerForecast,
		MaxActiveTopicsPerBlock: defaultParams.MaxActiveTopicsPerBlock,
	}

	store.Delete(types.ParamsKey)
	store.Set(types.ParamsKey, cdc.MustMarshal(&newParams))
	return nil
}

func MigrateTopics(
	ctx sdk.Context,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	emissionsKeeper keeper.Keeper,
) error {
	topicStore := prefix.NewStore(store, types.TopicsKey)
	topicFeeRevStore := prefix.NewStore(store, types.TopicFeeRevenueKey)
	topicStakeStore := prefix.NewStore(store, types.TopicStakeKey)
	topicPreviousWeightStore := prefix.NewStore(store, types.PreviousTopicWeightKey)
	iterator := topicStore.Iterator(nil, nil)
	churningBlockStore := prefix.NewStore(store, types.TopicToNextPossibleChurningBlockKey)
	blockToActiveStore := prefix.NewStore(store, types.BlockToActiveTopicsKey)
	blockLowestWeightStore := prefix.NewStore(store, types.BlockToLowestActiveTopicWeightKey)
	params, err := emissionsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to get params for active topic migration")
	}
	churningBlock := make(map[types.TopicId]types.BlockHeight, 0)
	blockToActiveTopics := make(map[types.BlockHeight]types.TopicIds, 0)
	lowestWeight := make(map[types.BlockHeight]types.TopicIdWeightPair, 0)

	topicWeightData := make(map[types.TopicId]alloraMath.Dec, 0)

	topicsToChange := make(map[string]types.Topic, 0)
	for ; iterator.Valid(); iterator.Next() {
		var oldMsg oldtypes.Topic
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal old topic")
		}
		var feeRevenue = cosmosMath.NewInt(0)
		idArray := make([]byte, 8)
		binary.BigEndian.PutUint64(idArray, oldMsg.Id)
		err = feeRevenue.Unmarshal(topicFeeRevStore.Get(idArray))
		if err != nil {
			topicsToChange[string(iterator.Key())] = getNewTopic(oldMsg)
			continue
		}
		var stake = cosmosMath.NewInt(0)
		err = stake.Unmarshal(topicStakeStore.Get(idArray))
		if err != nil {
			topicsToChange[string(iterator.Key())] = getNewTopic(oldMsg)
			continue
		}
		var previousWeight = alloraMath.NewDecFromInt64(0)
		err = previousWeight.Unmarshal(topicPreviousWeightStore.Get(idArray))
		if err != nil {
			topicsToChange[string(iterator.Key())] = getNewTopic(oldMsg)
			continue
		}
		// Get topic's latest weight
		weight, err := getTopicWeight(
			feeRevenue,
			stake,
			previousWeight,
			oldMsg.EpochLength,
			params.TopicRewardAlpha,
			params.TopicRewardStakeImportance,
			params.TopicRewardFeeRevenueImportance,
			emissionsKeeper,
		)
		if err != nil {
			topicsToChange[string(iterator.Key())] = getNewTopic(oldMsg)
			continue
		}
		topicWeightData[oldMsg.Id] = weight
		blockHeight := oldMsg.EpochLastEnded + oldMsg.EpochLength
		ctx.Logger().Warn(fmt.Sprintf("update blockHeight %d", blockHeight))
		// If the weight is less than minimum weight then skip this topic
		if weight.Lt(params.MinTopicWeight) {
			topicsToChange[string(iterator.Key())] = getNewTopic(oldMsg)
			continue
		}

		cuLowestWeight := lowestWeight[blockHeight]
		// Update lowest weight of topic per block
		if cuLowestWeight.Weight.Equal(alloraMath.ZeroDec()) ||
			weight.Lt(lowestWeight[blockHeight].Weight) {
			cuLowestWeight = types.TopicIdWeightPair{
				Weight:  weight,
				TopicId: oldMsg.Id,
			}
			lowestWeight[blockHeight] = cuLowestWeight
		}

		churningBlock[oldMsg.Id] = blockHeight

		activeTopicIds := blockToActiveTopics[blockHeight]
		activeTopicIds.TopicIds = append(activeTopicIds.TopicIds, oldMsg.Id)

		// If number of active topic is over global param then remove lowest topic
		if uint64(len(blockToActiveTopics[blockHeight].TopicIds)) >= params.MaxActiveTopicsPerBlock {
			// Remove from topicToNextPossibleChurningBlock
			delete(churningBlock, lowestWeight[blockHeight].TopicId)
			newActiveTopicIds := []types.TopicId{}
			for i, id := range blockToActiveTopics[blockHeight].TopicIds {
				if id == lowestWeight[blockHeight].TopicId {
					newActiveTopicIds = append(blockToActiveTopics[blockHeight].TopicIds[:i],
						blockToActiveTopics[blockHeight].TopicIds[i+1:]...)
					break
				}
			}
			// Reset active topics per block
			activeTopicIds.TopicIds = newActiveTopicIds
			//blockToActiveTopics[blockHeight] = types.TopicIds{TopicIds: newActiveTopicIds}
			// Reset lowest weight per block
			cuLowestWeight = getLowestTopicIdWeightPair(topicWeightData, blockToActiveTopics[blockHeight])
		}
		blockToActiveTopics[blockHeight] = activeTopicIds
		blockHeightBytes, err := collections.Int64Value.Encode(blockHeight)
		if err != nil {
			return err
		}
		churningBlockStore.Set(idArray, blockHeightBytes)
		activeTopicsBytes, err := activeTopicIds.Marshal()
		if err != nil {
			return err
		}
		lowestWeightBytes, err := cuLowestWeight.Marshal()
		if err != nil {
			return err
		}
		blockToActiveStore.Set(blockHeightBytes, activeTopicsBytes)
		blockLowestWeightStore.Set(blockHeightBytes, lowestWeightBytes)

		topicsToChange[string(iterator.Key())] = getNewTopic(oldMsg)
	}
	_ = iterator.Close()
	for key, value := range topicsToChange {
		topicStore.Set([]byte(key), cdc.MustMarshal(&value))
	}

	return nil
}

func getNewTopic(oldMsg oldtypes.Topic) types.Topic {
	return types.Topic{
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
		Epsilon:        oldMsg.Epsilon,
		// InitialRegret is being reset to account for NaNs that were previously stored due to insufficient validation
		InitialRegret:          alloraMath.MustNewDecFromString("0"),
		WorkerSubmissionWindow: oldMsg.WorkerSubmissionWindow,
		// These are new fields
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.25"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.25"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.25"),
	}
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

func getTopicWeight(
	feeRevenue, stake cosmosMath.Int,
	previousWeight alloraMath.Dec,
	topicEpochLength int64,
	topicRewardAlpha alloraMath.Dec,
	stakeImportance alloraMath.Dec,
	feeImportance alloraMath.Dec,
	emissionsKeeper keeper.Keeper,
) (alloraMath.Dec, error) {
	feeRevenueDec, err := alloraMath.NewDecFromSdkInt(feeRevenue)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	topicStakeDec, err := alloraMath.NewDecFromSdkInt(stake)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	if !feeRevenueDec.Equal(alloraMath.ZeroDec()) {
		targetWeight, err := emissionsKeeper.GetTargetWeight(
			topicStakeDec,
			topicEpochLength,
			feeRevenueDec,
			stakeImportance,
			feeImportance,
		)
		if err != nil {
			return alloraMath.ZeroDec(), err
		}
		weight, err := alloraMath.CalcEma(topicRewardAlpha, targetWeight, previousWeight, false)
		if err != nil {
			return alloraMath.ZeroDec(), err
		}
		return weight, nil
	}
	return alloraMath.ZeroDec(), nil
}

func getLowestTopicIdWeightPair(weightData map[types.TopicId]alloraMath.Dec, ids types.TopicIds) types.TopicIdWeightPair {
	lowestWeight := types.TopicIdWeightPair{
		Weight:  alloraMath.ZeroDec(),
		TopicId: uint64(0),
	}
	firstIter := true
	for _, id := range ids.TopicIds {
		if weightData[id].Lt(lowestWeight.Weight) || firstIter {
			lowestWeight = types.TopicIdWeightPair{
				Weight:  weightData[id],
				TopicId: id,
			}
			firstIter = false
		}
	}
	return lowestWeight
}
