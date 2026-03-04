package v0_16_0

import (
	"context"
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

var targetActiveTopicQuantile = alloraMath.MustNewDecFromString("0.05")

type topicQuantileMigrationKeeper interface {
	GetNextTopicId(ctx context.Context) (uint64, error)
	GetTopic(ctx context.Context, topicId uint64) (emissionstypes.Topic, error)
	SetTopic(ctx context.Context, topicId uint64, topic emissionstypes.Topic) error
}

func migrateTopicActiveQuantiles(ctx context.Context, emissionsKeeper *emissionskeeper.Keeper) error {
	return migrateTopicActiveQuantilesWithTopicKeeper(ctx, emissionsKeeper.GetTopicKeeper())
}

func migrateTopicActiveQuantilesWithTopicKeeper(ctx context.Context, topicKeeper topicQuantileMigrationKeeper) error {
	nextTopicID, err := topicKeeper.GetNextTopicId(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get next topic id")
	}

	for topicID := uint64(1); topicID < nextTopicID; topicID++ {
		topic, err := topicKeeper.GetTopic(ctx, topicID)
		if errors.Is(err, emissionstypes.ErrTopicDoesNotExist) {
			continue
		}
		if err != nil {
			return errorsmod.Wrapf(err, "failed to fetch topic %d", topicID)
		}

		topic.ActiveInfererQuantile = targetActiveTopicQuantile
		topic.ActiveForecasterQuantile = targetActiveTopicQuantile
		topic.ActiveReputerQuantile = targetActiveTopicQuantile

		if err := topicKeeper.SetTopic(ctx, topicID, topic); err != nil {
			return errorsmod.Wrap(err, fmt.Sprintf("failed to update topic %d quantiles", topicID))
		}
	}

	return nil
}
