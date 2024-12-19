package v6

import (
	"encoding/binary"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	oldV5Types "github.com/allora-network/allora-chain/x/emissions/migrations/v6/oldtypes"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
)

// MigrateStore migrates the store from version 5 to version 6
// It does the following:
// - Migrate params to add and set GlobalWhitelistEnabled, TopicCreatorWhitelistEnabled
// - Iterates through all topics to turn on their respective worker and reputer whitelists
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 5 TO VERSION 6")
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 5 TO VERSION 6")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	ctx.Logger().Info("MIGRATING PARAMS FROM VERSION 5 TO VERSION 6")
	// This also flips on global and topic creator whitelists
	if err := MigrateParams(store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateParams() FROM VERSION 5 TO VERSION 6")
		return err
	}

	ctx.Logger().Info("FLIPPING ON ALL TOPIC WORKER AND REPUTER WHITELISTS - VERSION 5 TO VERSION 6")
	if err := FlipOnTopicWhitelists(ctx, store, cdc, emissionsKeeper); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER FlipOnTopicWhitelists() - VERSION 5 TO VERSION 6")
		return err
	}

	ctx.Logger().Info("MIGRATING EMISSIONS MODULE FROM VERSION 5 TO VERSION 6 COMPLETE")
	return nil
}

// Migrate params for this new version
// The changes are the addition of GlobalWhitelistEnabled, TopicCreatorWhitelistEnabled
func MigrateParams(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	oldParams := oldV5Types.Params{} //nolint: exhaustruct // empty struct used by cosmos-sdk Unmarshal below
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
	//      GlobalWhitelistEnabled, TopicCreatorWhitelistEnabled
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
		MaxStringLength:                     oldParams.MaxStringLength,
		InitialRegretQuantile:               oldParams.InitialRegretQuantile,
		PNormSafeDiv:                        oldParams.PNormSafeDiv,
		// NEW PARAMS
		GlobalWhitelistEnabled:       defaultParams.GlobalWhitelistEnabled,
		TopicCreatorWhitelistEnabled: defaultParams.TopicCreatorWhitelistEnabled,
		MinExperiencedWorkerRegrets:  defaultParams.MinExperiencedWorkerRegrets,
	}

	store.Delete(emissionstypes.ParamsKey)
	store.Set(emissionstypes.ParamsKey, cdc.MustMarshal(&newParams))
	return nil
}

// Iterate through all topics and turn on their respective worker and reputer whitelists
func FlipOnTopicWhitelists(
	ctx sdk.Context,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	emissionsKeeper keeper.Keeper,
) error {
	nextTopicId, err := emissionsKeeper.GetNextTopicId(ctx)
	if err != nil {
		return err
	}

	// Iterate all topics to migrate using collections.go api
	// Turn on worker and reputer whitelists for each topic
	for id := uint64(1); id < nextTopicId; id++ {
		idByte := make([]byte, 8)
		binary.BigEndian.PutUint64(idByte, id)
		ctx.Logger().Info(fmt.Sprintf("MIGRATION V6: Updating topic:%d", id))

		err = emissionsKeeper.EnableTopicWorkerWhitelist(ctx, id)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to enable topic %d worker whitelist", id)
		}

		err = emissionsKeeper.EnableTopicReputerWhitelist(ctx, id)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to enable topic %d reputer whitelist", id)
		}
	}

	return nil
}
