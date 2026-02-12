package v14

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"

	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// MigrateStore migrates the store from version 13 to version 14.
// It does the following:
// - Migrate all topics to add defaults for TopicType and OutputArity fields
func MigrateStore(ctx sdk.Context, emissionsKeeper keeper.Keeper) error {
	ctx.Logger().Info("STARTING EMISSIONS MODULE MIGRATION FROM VERSION 13 TO VERSION 14")
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 13 TO VERSION 14")
	storageService := emissionsKeeper.GetStorageService()
	store := runtime.KVStoreAdapter(storageService.OpenKVStore(ctx))
	cdc := emissionsKeeper.GetBinaryCodec()

	ctx.Logger().Info("MIGRATING TOPICS FROM VERSION 13 TO VERSION 14")
	if err := MigrateTopics(ctx, store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateTopics() FROM VERSION 13 TO VERSION 14")
		return err
	}

	ctx.Logger().Info("MIGRATING EMISSIONS MODULE FROM VERSION 13 TO VERSION 14 COMPLETE")
	return nil
}

// MigrateTopics iterates through all topics and adds TopicType and OutputArity field defaults to REGRESSION and SINGLE, respectively
func MigrateTopics(
	ctx sdk.Context,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
) error {
	topicStore := prefix.NewStore(store, emissionstypes.TopicsKey)
	iterator := topicStore.Iterator(nil, nil)
	defer iterator.Close()

	ctx.Logger().Info("MIGRATION V14: Migrating topics to add TopicType and OutputArity")

	topicsToChange := make(map[string]emissionstypes.Topic)
	for ; iterator.Valid(); iterator.Next() {
		var topic emissionstypes.Topic
		err := proto.Unmarshal(iterator.Value(), &topic)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal topic")
		}

		ctx.Logger().Debug("MIGRATION V14: Updating topic", "topicId", topic.Id)

		topic.TopicType = emissionstypes.TopicType_TOPIC_TYPE_REGRESSION
		topic.OutputArity = emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE

		topicsToChange[string(iterator.Key())] = topic
	}

	for key, topic := range topicsToChange {
		topicStore.Set([]byte(key), cdc.MustMarshal(&topic))
		ctx.Logger().Debug("MIGRATION V14: Updated topic with TopicType and OutputArity", "topicId", topic.Id)
	}

	ctx.Logger().Info("MIGRATION V14: Topics migration complete", "topicsUpdated", len(topicsToChange))
	return nil
}
