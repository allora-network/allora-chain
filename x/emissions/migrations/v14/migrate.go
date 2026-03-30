package v14

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
)

var targetActiveTopicQuantile = alloraMath.MustNewDecFromString("0.05")

// MigrateStore migrates the emissions module from version 13 to version 14.
// It updates every existing topic to use the v0.16.0 active quantile defaults.
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 13 TO VERSION 14")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	if err := MigrateTopics(ctx, store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateTopics() FROM VERSION 13 TO VERSION 14")
		return err
	}

	ctx.Logger().Info("MIGRATION EMISSIONS MODULE FROM VERSION 13 TO VERSION 14 COMPLETE")
	return nil
}

// MigrateTopics rewrites the stored per-topic active quantiles
// used by the EMA-based sortition flow introduced in v0.16.0.
func MigrateTopics(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	iterator := topicStore.Iterator(nil, nil)
	defer iterator.Close()

	ctx.Logger().Info("MIGRATION V14: starting topic active quantiles migration")

	topicsToChange := make(map[string]emissionstypes.Topic)
	for ; iterator.Valid(); iterator.Next() {
		var topic emissionstypes.Topic
		err := proto.Unmarshal(iterator.Value(), &topic)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal topic")
		}

		topic.ActiveInfererQuantile = targetActiveTopicQuantile
		topic.ActiveForecasterQuantile = targetActiveTopicQuantile
		topic.ActiveReputerQuantile = targetActiveTopicQuantile

		topicsToChange[string(iterator.Key())] = topic
		ctx.Logger().Info("MIGRATION V14: updated topic quantiles in memory", "topicID", topic.Id)
	}

	for key, topic := range topicsToChange {
		topicStore.Set([]byte(key), cdc.MustMarshal(&topic))
		ctx.Logger().Info("MIGRATION V14: stored updated topic quantiles", "topicID", topic.Id)
	}

	ctx.Logger().Info("MIGRATION V14: topic active quantiles migration completed", "topicsUpdated", len(topicsToChange))
	return nil
}
