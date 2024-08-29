package v3

import (
	"encoding/binary"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
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

func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 2 TO VERSION 3")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	if err := MigrateParams(store, cdc); err != nil {
		return err
	}
	if err := MigrateActiveTopics(store, ctx, emissionsKeeper); err != nil {
		return err
	}

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
		MaxElementsPerForecast:              defaultParams.MaxElementsPerForecast,
		MaxActiveTopicsPerBlock:             defaultParams.MaxActiveTopicsPerBlock,
	}

	store.Delete(types.ParamsKey)
	store.Set(types.ParamsKey, cdc.MustMarshal(&newParams))
	return nil
}

func MigrateActiveTopics(store storetypes.KVStore, ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Warn("MigrateActiveTopics")
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
		return errors.Wrapf(err, "failed to get params for active topic migration")
	}
	churningBlock := make(map[types.TopicId]types.BlockHeight, 0)
	blockToActiveTopics := make(map[types.BlockHeight]types.TopicIds, 0)
	lowestWeight := make(map[types.BlockHeight]types.TopicIdWeightPair, 0)

	topicWeightData := make(map[types.TopicId]alloraMath.Dec, 0)

	for ; iterator.Valid(); iterator.Next() {
		var oldMsg oldtypes.Topic
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			continue
		}
		var feeRevenue = cosmosMath.NewInt(0)
		idArray := make([]byte, 8)
		binary.BigEndian.PutUint64(idArray, oldMsg.Id)
		err = feeRevenue.Unmarshal(topicFeeRevStore.Get(idArray))
		if err != nil {
			continue
		}
		var stake = cosmosMath.NewInt(0)
		err = stake.Unmarshal(topicStakeStore.Get(idArray))
		if err != nil {
			continue
		}
		var previousWeight = alloraMath.NewDecFromInt64(0)
		err = previousWeight.Unmarshal(topicPreviousWeightStore.Get(idArray))
		if err != nil {
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
			continue
		}
		topicWeightData[oldMsg.Id] = weight
		blockHeight := oldMsg.EpochLastEnded + oldMsg.EpochLength
		sdkCtx.Logger().Warn(fmt.Sprintf("update blockHeight %d", blockHeight))
		// If the weight less than minimum weight then skip this topic
		if weight.Lt(params.MinTopicWeight) {
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
		}

		churningBlock[oldMsg.Id] = blockHeight

		activeTopicIds := blockToActiveTopics[blockHeight]
		activeTopicIds.TopicIds = append(activeTopicIds.TopicIds, oldMsg.Id)

		// If number of active topic is over global param then remove lowest topic
		if uint64(len(blockToActiveTopics[blockHeight].TopicIds)) > params.MaxActiveTopicsPerBlock {
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
	}
	_ = iterator.Close()
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
