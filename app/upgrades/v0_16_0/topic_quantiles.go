package v0_16_0 //nolint:revive // Upgrade package naming follows version directory convention.

import (
	"context"
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var targetActiveTopicQuantile = alloraMath.MustNewDecFromString("0.05")

type topicQuantileMigrationKeeper interface {
	GetNextTopicId(ctx context.Context) (uint64, error)
	GetTopic(ctx context.Context, topicId uint64) (emissionstypes.Topic, error)
	SetTopic(ctx context.Context, topicId uint64, topic emissionstypes.Topic) error
}

func migrateTopicActiveQuantiles(ctx sdk.Context, emissionsKeeper *emissionskeeper.Keeper) error {
	return migrateTopicActiveQuantilesWithTopicKeeper(ctx, emissionsKeeper.GetTopicKeeper())
}

func migrateTopicActiveQuantilesWithTopicKeeper(ctx sdk.Context, topicKeeper topicQuantileMigrationKeeper) error {
	ctx.Logger().Info("MIGRATION v0.16.0: starting topic active quantiles migration")

	nextTopicID, err := topicKeeper.GetNextTopicId(ctx)
	if err != nil {
		ctx.Logger().Error("MIGRATION v0.16.0: failed to get next topic id", "error", err)
		return errorsmod.Wrap(err, "failed to get next topic id")
	}
	ctx.Logger().Info("MIGRATION v0.16.0: loaded next topic id", "nextTopicID", nextTopicID)

	updatedTopics := 0
	for topicID := uint64(1); topicID < nextTopicID; topicID++ {
		topic, err := topicKeeper.GetTopic(ctx, topicID)
		if errors.Is(err, emissionstypes.ErrTopicDoesNotExist) {
			continue
		}
		if err != nil {
			ctx.Logger().Error("MIGRATION v0.16.0: failed to fetch topic", "topicID", topicID, "error", err)
			return errorsmod.Wrapf(err, "failed to fetch topic %d", topicID)
		}

		topic.ActiveInfererQuantile = targetActiveTopicQuantile
		topic.ActiveForecasterQuantile = targetActiveTopicQuantile
		topic.ActiveReputerQuantile = targetActiveTopicQuantile

		if err := topicKeeper.SetTopic(ctx, topicID, topic); err != nil {
			ctx.Logger().Error("MIGRATION v0.16.0: failed to update topic quantiles", "topicID", topicID, "error", err)
			return errorsmod.Wrap(err, fmt.Sprintf("failed to update topic %d quantiles", topicID))
		}
		ctx.Logger().Info("MIGRATION v0.16.0: updated topic quantiles", "topicID", topicID)
		updatedTopics++
	}

	ctx.Logger().Info(
		"MIGRATION v0.16.0: topic active quantiles migration completed",
		"topicsUpdated",
		updatedTopics,
	)
	return nil
}
