package v5

import (
	"encoding/binary"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	alloraMath "github.com/allora-network/allora-chain/math"
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
	ResetMapsWithNonNumericValues(ctx, store, cdc)

	ctx.Logger().Info("MIGRATING EMISSIONS MODULE FROM VERSION 4 TO VERSION 5 COMPLETE")
	return nil
}

// migrate params for this new version
// the changes are the addition of InitialRegretQuantile,PnormSafeDiv
func MigrateParams(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	oldParams := oldV4Types.Params{} //nolint: exhaustruct
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
	//      InitialRegretQuantile, PnormSafeDiv
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
		PnormSafeDiv:          defaultParams.PnormSafeDiv,
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
	}
	for key, value := range topicsToChange {
		topicStore.Set([]byte(key), cdc.MustMarshal(&value))
	}

	return nil
}

// Deletes all keys in the store with the given keyPrefix `maxPageSize` keys at a time
func safelyClearWholeMap(ctx sdk.Context, store storetypes.KVStore, keyPrefix []byte) {
	s := prefix.NewStore(store, keyPrefix)

	// Loop until all keys are deleted.
	// Unbounded not best practice but we are sure that the number of keys will be limited
	// and not deleting all keys means "poison" will remain in the store.
	for {
		// Gather keys to eventually delete
		iterator := s.Iterator(nil, nil)
		defer iterator.Close()
		keysToDelete := make([][]byte, 0)
		count := uint64(0)
		for ; iterator.Valid(); iterator.Next() {
			ctx.Logger().Info("MIGRATION V5: DELETING keys in store with prefix", "prefix", keyPrefix, "page", count)
			if count >= maxPageSize {
				break
			}

			keysToDelete = append(keysToDelete, iterator.Key())
			count++
		}
		err := iterator.Close()
		if err != nil {
			break
		}

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
	ctx.Logger().Info("MIGRATION V5: RESETTING PreviousTopicQuantileInfererScoreEmaKey MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.PreviousTopicQuantileInfererScoreEmaKey)
	ctx.Logger().Info("MIGRATION V5: RESETTING PreviousTopicQuantileForecasterScoreEmaKey MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.PreviousTopicQuantileForecasterScoreEmaKey)
	ctx.Logger().Info("MIGRATION V5: RESETTING PreviousTopicQuantileReputerScoreEmaKey MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.PreviousTopicQuantileReputerScoreEmaKey)
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
