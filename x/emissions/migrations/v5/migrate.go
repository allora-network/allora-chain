package v4

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	oldV2Types "github.com/allora-network/allora-chain/x/emissions/migrations/v3/oldtypes"
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
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 4 TO VERSION 5")
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 4 TO VERSION 5")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

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
		ctx.Logger().Info("MIGRATION V5: Updating topic", iterator.Key())
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
		// fix egregious values
		newTopic.InitialRegret = alloraMath.MustNewDecFromString("0")
		// was wrongly set to true for existing topics
		newTopic.AllowNegative = false
		topicsToChange[string(iterator.Key())] = newTopic
	}
	_ = iterator.Close()
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
	ctx.Logger().Info("MIGRATION V4: RESETTING infererScoresByBlock MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.InferenceScoresKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING forecasterScoresByBlock MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.ForecastScoresKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING reputerScoresByBlock MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.ReputerScoresKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING infererScoreEmas MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.InfererScoreEmasKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING forecasterScoreEmas MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.ForecasterScoreEmasKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING reputerScoreEmas MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.ReputerScoreEmasKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING allLossBundles MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.AllLossBundlesKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING networkLossBundles MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.NetworkLossBundlesKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING latestInfererNetworkRegrets MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.InfererNetworkRegretsKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING latestForecasterNetworkRegrets MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.ForecasterNetworkRegretsKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING latestOneInForecasterNetworkRegrets MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.OneInForecasterNetworkRegretsKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING latestNaiveInfererNetworkRegrets MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.LatestNaiveInfererNetworkRegretsKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING latestOneOutInfererInfererNetworkRegrets MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.LatestOneOutInfererInfererNetworkRegretsKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING latestOneOutInfererForecasterNetworkRegrets MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.LatestOneOutInfererForecasterNetworkRegretsKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING latestOneOutForecasterInfererNetworkRegrets MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.LatestOneOutForecasterInfererNetworkRegretsKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING latestOneOutForecasterForecasterNetworkRegrets MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.LatestOneOutForecasterForecasterNetworkRegretsKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING ReputerListeningCoefficientKey MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.ReputerListeningCoefficientKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING PreviousTopicQuantileInfererScoreEmaKey MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.PreviousTopicQuantileInfererScoreEmaKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING PreviousTopicQuantileForecasterScoreEmaKey MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.PreviousTopicQuantileForecasterScoreEmaKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING PreviousTopicQuantileReputerScoreEmaKey MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.PreviousTopicQuantileReputerScoreEmaKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING PreviousInferenceRewardFractionKey MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.PreviousInferenceRewardFractionKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING PreviousForecastRewardFractionKey MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.PreviousForecastRewardFractionKey)
	ctx.Logger().Info("MIGRATION V4: RESETTING PreviousReputerRewardFractionKey MAP")
	safelyClearWholeMap(ctx, store, emissionstypes.PreviousReputerRewardFractionKey)
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
