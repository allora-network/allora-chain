package v2

import (
	"context"

	"cosmossdk.io/core/store"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	emissionsv1 "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
)

func MigrateStore(ctx sdk.Context, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	ctx.Logger().Info("MIGRATING STORE FROM VERSION 1 TO VERSION 2")

	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	err := MigrateMsgCreateNewTopic(ctx, store, cdc)
	if err != nil {
		return err
	}

	return nil
}

func MigrateMsgCreateNewTopic(ctx context.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	topicStore := prefix.NewStore(store, emissionsv1.TopicsKey)
	iterator := topicStore.Iterator(nil, nil)

	for ; iterator.Valid(); iterator.Next() {
		var oldMsg emissionsv1.MsgCreateNewTopic
		err := proto.Unmarshal(iterator.Value(), &oldMsg)
		if err != nil {
			return err
		}

		newMsg := emissionsv1.MsgCreateNewTopic{
			Creator:                oldMsg.Creator,
			Metadata:               oldMsg.Metadata,
			LossMethod:             oldMsg.LossMethod,
			EpochLength:            oldMsg.EpochLength,
			GroundTruthLag:         oldMsg.GroundTruthLag,
			PNorm:                  oldMsg.PNorm,
			AlphaRegret:            oldMsg.AlphaRegret,
			AllowNegative:          oldMsg.AllowNegative,
			Epsilon:                oldMsg.Epsilon,
			WorkerSubmissionWindow: oldMsg.WorkerSubmissionWindow,
		}

		store.Delete(iterator.Key())
		store.Set(iterator.Key(), cdc.MustMarshal(&newMsg))
	}

	return nil
}
