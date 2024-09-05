package v4

import (
	errorsmod "cosmossdk.io/errors"
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
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 3 TO VERSION 4")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	ctx.Logger().Info("MIGRATING STORE FROM VERSION 3 TO VERSION 4")
	if err := MigrateTopics(ctx, store, cdc, emissionsKeeper); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateTopics() FROM VERSION 3 TO VERSION 4")
		return err
	}

	ctx.Logger().Info("MIGRATING STORE FROM VERSION 3 TO VERSION 4")
	ResetMapsWithNonNumericValues(store, cdc)

	return nil
}

func MigrateTopics(
	ctx sdk.Context,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
	emissionsKeeper keeper.Keeper,
) error {
	topicStore := prefix.NewStore(store, types.TopicsKey)
	iterator := topicStore.Iterator(nil, nil)

	topicsToChange := make(map[string]types.Topic, 0)
	for ; iterator.Valid(); iterator.Next() {
		var oldMsg oldtypes.Topic
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal old topic")
		}

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
