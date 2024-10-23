package v3

import (
	"encoding/binary"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/utils/migutils"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	oldtypes "github.com/allora-network/allora-chain/x/emissions/migrations/v3/oldtypes"
	types "github.com/allora-network/allora-chain/x/emissions/types"
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
	err := ResetMapsWithNonNumericValues(ctx, store, cdc)
	if err != nil {
		ctx.Logger().Error("ERROR RESETTING MAPS WITH NON NUMERIC VALUES: %v", err)
		return err
	}

	return nil
}

func MigrateParams(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	oldParams := oldtypes.Params{} //nolint: exhaustruct // populated in unmarshal below
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
	newParams := types.Params{ //nolint: exhaustruct // not sure if safe to fix, also this upgrade has already happened.
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
	churningBlockStore := prefix.NewStore(store, types.TopicToNextPossibleChurningBlockKey)
	blockToActiveStore := prefix.NewStore(store, types.BlockToActiveTopicsKey)
	blockLowestWeightStore := prefix.NewStore(store, types.BlockToLowestActiveTopicWeightKey)
	params, err := emissionsKeeper.GetParams(ctx)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to get params for active topic migration")
	}

	iterator := topicStore.Iterator(nil, nil)
	defer iterator.Close()

	blockToActiveTopics := make(map[types.BlockHeight]types.TopicIds, 0)
	lowestWeight := make(map[types.BlockHeight]types.TopicIdWeightPair, 0)
	churningBlock := make(map[types.TopicId]types.BlockHeight, 0)
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
		stake := cosmosMath.NewInt(0)
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

		activeTopicIds := blockToActiveTopics[blockHeight]
		activeTopicIds.TopicIds = append(activeTopicIds.TopicIds, oldMsg.Id)
		// If number of active topic is over global param then remove lowest topic
		if uint64(len(blockToActiveTopics[blockHeight].TopicIds)) >= params.MaxActiveTopicsPerBlock {
			// If current weight is lower than lowest then skip
			// Otherwise upgrade lowest weight
			if weight.Lt(lowestWeight[blockHeight].Weight) {
				continue
			} else {
				newActiveTopicIds := []types.TopicId{}
				for i, id := range activeTopicIds.TopicIds {
					if id == lowestWeight[blockHeight].TopicId {
						delete(churningBlock, id)
						newActiveTopicIds = append(activeTopicIds.TopicIds[:i],
							activeTopicIds.TopicIds[i+1:]...)
						break
					}
				}
				activeTopicIds.TopicIds = newActiveTopicIds
				lowestWeight[blockHeight] = getLowestTopicIdWeightPair(topicWeightData, activeTopicIds)
			}
		}
		churningBlock[oldMsg.Id] = blockHeight
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

		blockToActiveTopics[blockHeight] = activeTopicIds
		blockHeightBytes, err := collections.Int64Value.Encode(blockHeight)
		if err != nil {
			return err
		}
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
	err = iterator.Close()
	if err != nil {
		return err
	}

	for key, value := range churningBlock {
		blockHeightBytes, err := collections.Int64Value.Encode(value)
		if err != nil {
			return err
		}
		idArray := make([]byte, 8)
		binary.BigEndian.PutUint64(idArray, key)
		churningBlockStore.Set(idArray, blockHeightBytes)
	}
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

func ResetMapsWithNonNumericValues(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	prefixes := []collections.Prefix{
		types.InferenceScoresKey,
		types.ForecastScoresKey,
		types.ReputerScoresKey,
		types.InfererScoreEmasKey,
		types.ForecasterScoreEmasKey,
		types.ReputerScoreEmasKey,
		types.AllLossBundlesKey,
		types.NetworkLossBundlesKey,
		types.InfererNetworkRegretsKey,
		types.ForecasterNetworkRegretsKey,
		types.OneInForecasterNetworkRegretsKey,
		types.LatestNaiveInfererNetworkRegretsKey,
		types.LatestOneOutInfererInfererNetworkRegretsKey,
		types.LatestOneOutInfererForecasterNetworkRegretsKey,
		types.LatestOneOutForecasterInfererNetworkRegretsKey,
		types.LatestOneOutForecasterForecasterNetworkRegretsKey,
	}
	for _, prefix := range prefixes {
		err := migutils.SafelyClearWholeMap(ctx, store, prefix, maxPageSize)
		if err != nil {
			return err
		}
	}
	return nil
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
