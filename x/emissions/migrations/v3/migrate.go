package v3

import (
	"cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
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
		TopicFeeRevenueDecayRate:            oldParams.TopicFeeRevenueDecayRate,
		EpsilonReputer:                      oldParams.EpsilonReputer,
		HalfMaxProcessStakeRemovalsEndBlock: oldParams.HalfMaxProcessStakeRemovalsEndBlock,
		EpsilonSafeDiv:                      oldParams.EpsilonSafeDiv,
		DataSendingFee:                      oldParams.DataSendingFee,
		MaxElementsPerForecast:              defaultParams.MaxElementsPerForecast,
	}

	store.Delete(types.ParamsKey)
	store.Set(types.ParamsKey, cdc.MustMarshal(&newParams))
	return nil
}
