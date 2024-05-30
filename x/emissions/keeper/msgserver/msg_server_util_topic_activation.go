package msgserver

import (
	"context"

	"cosmossdk.io/errors"
	cosmosMath "cosmossdk.io/math"
)

type TopicId = uint64
type Allo = cosmosMath.Int

func (ms *msgServer) ActivateTopicIfWeightAtLeastGlobalMin(
	ctx context.Context,
	topicId TopicId,
	amount Allo,
) error {
	isActivated, err := ms.k.IsTopicActive(ctx, topicId)
	if err != nil {
		return errors.Wrapf(err, "error getting topic activation status")
	}
	if !isActivated {
		params, err := ms.k.GetParams(ctx)
		if err != nil {
			return errors.Wrapf(err, "error getting params")
		}
		topic, err := ms.k.GetTopic(ctx, topicId)
		if err != nil {
			return errors.Wrapf(err, "error getting topic")
		}

		newTopicWeight, _, err := ms.k.GetCurrentTopicWeight(
			ctx,
			topicId,
			topic.EpochLength,
			params.TopicRewardAlpha,
			params.TopicRewardStakeImportance,
			params.TopicRewardFeeRevenueImportance,
			amount,
		)
		if err != nil {
			return errors.Wrapf(err, "error getting current topic weight")
		}

		if newTopicWeight.Gte(params.MinTopicWeight) {
			err = ms.k.ActivateTopic(ctx, topicId)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
