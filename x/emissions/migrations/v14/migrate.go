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
	"github.com/allora-network/allora-chain/x/emissions/migrations/v14/oldtypes"
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

	ctx.Logger().Info("MIGRATING NETWORK INFERENCES FROM VERSION 13 TO VERSION 14")
	if err := MigrateNetworkInferences(ctx, store, cdc); err != nil {
		ctx.Logger().Error("ERROR INVOKING MIGRATION HANDLER MigrateNetworkInferences() FROM VERSION 13 TO VERSION 14")
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

	type kv struct {
		key   []byte
		value []byte
	}
	updates := make([]kv, 0)

	for ; iterator.Valid(); iterator.Next() {
		var oldTopic oldtypes.Topic
		if err := cdc.Unmarshal(iterator.Value(), &oldTopic); err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal topic")
		}

		ctx.Logger().Debug("MIGRATION V14: Updating topic", "topicId", oldTopic.Id)

		newTopic := emissionstypes.Topic{
			Id:                       oldTopic.Id,
			Creator:                  oldTopic.Creator,
			Metadata:                 oldTopic.Metadata,
			LossMethod:               oldTopic.LossMethod,
			EpochLastEnded:           oldTopic.EpochLastEnded,
			EpochLength:              oldTopic.EpochLength,
			GroundTruthLag:           oldTopic.GroundTruthLag,
			PNorm:                    oldTopic.PNorm,
			AlphaRegret:              oldTopic.AlphaRegret,
			AllowNegative:            oldTopic.AllowNegative,
			Epsilon:                  oldTopic.Epsilon,
			InitialRegret:            oldTopic.InitialRegret,
			WorkerSubmissionWindow:   oldTopic.WorkerSubmissionWindow,
			MeritSortitionAlpha:      oldTopic.MeritSortitionAlpha,
			ActiveInfererQuantile:    oldTopic.ActiveInfererQuantile,
			ActiveForecasterQuantile: oldTopic.ActiveForecasterQuantile,
			ActiveReputerQuantile:    oldTopic.ActiveReputerQuantile,
			CNorm:                    oldTopic.CNorm,
			TopicType:                emissionstypes.TopicType_TOPIC_TYPE_REGRESSION,
			OutputArity:              emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
			RequireUnity:             false,
			UnityTolerance:           alloraMath.ZeroDec(),
		}

		updates = append(updates, kv{
			key:   append([]byte(nil), iterator.Key()...),
			value: cdc.MustMarshal(&newTopic),
		})
	}
	for _, u := range updates {
		topicStore.Set(u.key, u.value)
	}

	ctx.Logger().Info("MIGRATION V14: Topics migration complete", "topicsUpdated", len(updates))
	return nil
}

func MigrateNetworkInferences(
	ctx sdk.Context,
	store storetypes.KVStore,
	cdc codec.BinaryCodec,
) error {
	networkInferencesStore := prefix.NewStore(store, emissionstypes.NetworkInferencesKey)
	iterator := networkInferencesStore.Iterator(nil, nil)
	defer iterator.Close()

	ctx.Logger().Info("MIGRATION V14: Migrating networkInferences to networkInferenceBundles")

	type kv struct {
		key   []byte
		value []byte
	}
	updates := make([]kv, 0)

	for ; iterator.Valid(); iterator.Next() {
		var oldNetworkInference emissionstypes.ValueBundle
		if err := cdc.Unmarshal(iterator.Value(), &oldNetworkInference); err != nil {
			return errorsmod.Wrapf(err, "failed to unmarshal networkInference")
		}

		networkInferenceBundle := emissionstypes.ValueBundleToNetworkInferenceBundle(&oldNetworkInference)
		if networkInferenceBundle == nil {
			return errorsmod.Wrapf(emissionstypes.ErrInvalidValue, "converted network inference bundle is nil; topicId: %d, nonce: %d",
				oldNetworkInference.TopicId, oldNetworkInference.ReputerRequestNonce.ReputerNonce.BlockHeight)
		}

		if err := networkInferenceBundle.Validate(); err != nil {
			return errorsmod.Wrapf(err, "failed to validate networkInference")
		}

		ctx.Logger().Debug("MIGRATION V14: Creating network inference bundle from network inference",
			"topicId", networkInferenceBundle.TopicId,
			"nonce", networkInferenceBundle.Nonce)
		updates = append(updates, kv{
			key:   append([]byte(nil), iterator.Key()...),
			value: cdc.MustMarshal(networkInferenceBundle),
		})
	}

	networkInferenceBundlesStore := prefix.NewStore(store, emissionstypes.NetworkInferenceBundleKey)

	for _, u := range updates {
		networkInferenceBundlesStore.Set(u.key, u.value)
		networkInferencesStore.Delete(u.key)
	}

	ctx.Logger().Info("MIGRATION V14: Network Inference Bundles migration complete", "networkInferencesUpdated", len(updates))
	return nil
}
