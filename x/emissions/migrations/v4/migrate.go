package v4

import (
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

func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
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

	return nil
}

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
