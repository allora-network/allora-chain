package v14

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/keeper"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
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

	type kv struct {
		key   []byte
		value []byte
	}
	updates := make([]kv, 0)

	for ; iterator.Valid(); iterator.Next() {
		var topic emissionstypes.Topic
		if err := cdc.Unmarshal(iterator.Value(), &topic); err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal topic")
		}

		topic.ActiveInfererQuantile = targetActiveTopicQuantile
		topic.ActiveForecasterQuantile = targetActiveTopicQuantile
		topic.ActiveReputerQuantile = targetActiveTopicQuantile

		updates = append(updates, kv{
			key:   append([]byte(nil), iterator.Key()...),
			value: cdc.MustMarshal(&topic),
		})
	}

	for _, u := range updates {
		topicStore.Set(u.key, u.value)
	}

	ctx.Logger().Info("MIGRATION V14: topic active quantiles migration completed", "topicsUpdated", len(updates))
	return nil
}
