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

	// iterate over the topic store by manually checking prefixes
	// to make sure we absolutely do not miss any topics
	// (as opposed to using collections.go API)
	topicsToChange := make(map[string]types.Topic, 0)
	for ; iterator.Valid(); iterator.Next() {
		iterator.Key()
		var oldMsg oldtypes.Topic
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

func copyTopic(original types.Topic) types.Topic {
	return types.Topic{
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
