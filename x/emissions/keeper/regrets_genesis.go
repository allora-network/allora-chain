package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func (k *RegretsKeeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	// LatestInfererNetworkRegrets
	for _, topicIdActorIdTimeStampedValue := range data.LatestInfererNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.SetInfererNetworkRegret(ctx,
				topicIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdTimeStampedValue.ActorId,
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestInfererNetworkRegrets")
			}
		}
	}
	// LatestNaiveInfererNetworkRegrets
	for _, topicIdActorIdTimeStampedValue := range data.LatestNaiveInfererNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.SetNaiveInfererNetworkRegret(ctx,
				topicIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdTimeStampedValue.ActorId,
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestNaiveInfererNetworkRegrets")
			}
		}
	}
	// LatestForecasterNetworkRegrets
	for _, topicIdActorIdTimeStampedValue := range data.LatestForecasterNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.SetForecasterNetworkRegret(ctx,
				topicIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdTimeStampedValue.ActorId,
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestForecasterNetworkRegrets")
			}
		}
	}
	// LatestOneOutInfererInfererNetworkRegrets
	for _, topicIdActorIdTimeStampedValue := range data.LatestOneOutInfererInfererNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.SetOneOutInfererInfererNetworkRegret(ctx,
				topicIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdTimeStampedValue.ActorId1,
				topicIdActorIdTimeStampedValue.ActorId2,
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestOneOutInfererInfererNetworkRegrets")
			}
		}
	}
	// LatestOneOutInfererForecasterNetworkRegrets — direct field access to match original genesis behavior
	for _, topicIdActorIdTimeStampedValue := range data.LatestOneOutInfererForecasterNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.latestOneOutInfererForecasterNetworkRegrets.Set(ctx,
				collections.Join3(
					topicIdActorIdTimeStampedValue.TopicId,
					topicIdActorIdTimeStampedValue.ActorId1,
					topicIdActorIdTimeStampedValue.ActorId2,
				),
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestOneOutInfererForecasterNetworkRegrets")
			}
		}
	}
	// LatestOneOutForecasterInfererNetworkRegrets
	for _, topicIdActorIdTimeStampedValue := range data.LatestOneOutForecasterInfererNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.SetOneOutForecasterInfererNetworkRegret(ctx,
				topicIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdTimeStampedValue.ActorId1,
				topicIdActorIdTimeStampedValue.ActorId2,
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestOneOutForecasterInfererNetworkRegrets")
			}
		}
	}
	// LatestOneOutForecasterForecasterNetworkRegrets
	for _, topicIdActorIdTimeStampedValue := range data.LatestOneOutForecasterForecasterNetworkRegrets {
		if topicIdActorIdTimeStampedValue != nil {
			if err := k.SetOneOutForecasterForecasterNetworkRegret(ctx,
				topicIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdTimeStampedValue.ActorId1,
				topicIdActorIdTimeStampedValue.ActorId2,
				*topicIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestOneOutForecasterForecasterNetworkRegrets")
			}
		}
	}
	// LatestOneInForecasterNetworkRegrets
	for _, topicIdActorIdActorIdTimeStampedValue := range data.LatestOneInForecasterNetworkRegrets {
		if topicIdActorIdActorIdTimeStampedValue != nil {
			if err := k.SetOneInForecasterNetworkRegret(ctx,
				topicIdActorIdActorIdTimeStampedValue.TopicId,
				topicIdActorIdActorIdTimeStampedValue.ActorId1,
				topicIdActorIdActorIdTimeStampedValue.ActorId2,
				*topicIdActorIdActorIdTimeStampedValue.TimestampedValue); err != nil {
				return errors.Wrap(err, "error setting latestOneInForecasterNetworkRegrets")
			}
		}
	}

	return nil
}

func (k *RegretsKeeper) ExportGenesis(ctx context.Context, data *types.GenesisState) error {
	// latestInfererNetworkRegrets
	latestInfererNetworkRegrets := make([]*types.TopicIdActorIdTimeStampedValue, 0)
	latestInfererNetworkRegretsIter, err := k.latestInfererNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate latest inferer network regrets")
	}
	for ; latestInfererNetworkRegretsIter.Valid(); latestInfererNetworkRegretsIter.Next() {
		keyValue, err := latestInfererNetworkRegretsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: latestInfererNetworkRegretsIter")
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId:          keyValue.Key.K2(),
			TimestampedValue: &keyValue.Value,
		}
		latestInfererNetworkRegrets = append(latestInfererNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}
	data.LatestInfererNetworkRegrets = latestInfererNetworkRegrets

	// latestNaiveInfererNetworkRegrets
	latestNaiveInfererNetworkRegrets := make([]*types.TopicIdActorIdTimeStampedValue, 0)
	latestNaiveInfererNetworkRegretsIter, err := k.latestNaiveInfererNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate latest naive inferer network regrets")
	}
	for ; latestNaiveInfererNetworkRegretsIter.Valid(); latestNaiveInfererNetworkRegretsIter.Next() {
		keyValue, err := latestNaiveInfererNetworkRegretsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: latestNaiveInfererNetworkRegretsIter")
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId:          keyValue.Key.K2(),
			TimestampedValue: &keyValue.Value,
		}
		latestNaiveInfererNetworkRegrets = append(latestNaiveInfererNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}
	data.LatestNaiveInfererNetworkRegrets = latestNaiveInfererNetworkRegrets

	// latestForecasterNetworkRegrets
	latestForecasterNetworkRegrets := make([]*types.TopicIdActorIdTimeStampedValue, 0)
	latestForecasterNetworkRegretsIter, err := k.latestForecasterNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate latest forecaster network regrets")
	}
	for ; latestForecasterNetworkRegretsIter.Valid(); latestForecasterNetworkRegretsIter.Next() {
		keyValue, err := latestForecasterNetworkRegretsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: latestForecasterNetworkRegretsIter")
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId:          keyValue.Key.K2(),
			TimestampedValue: &keyValue.Value,
		}
		latestForecasterNetworkRegrets = append(latestForecasterNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}
	data.LatestForecasterNetworkRegrets = latestForecasterNetworkRegrets

	// latestOneOutInfererInfererNetworkRegrets
	latestOneOutInfererInfererNetworkRegrets := make([]*types.TopicIdActorIdActorIdTimeStampedValue, 0)
	latestOneOutInfererInfererNetworkRegretsIter, err := k.latestOneOutInfererInfererNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate latest one out inferer inferer network regrets")
	}
	for ; latestOneOutInfererInfererNetworkRegretsIter.Valid(); latestOneOutInfererInfererNetworkRegretsIter.Next() {
		keyValue, err := latestOneOutInfererInfererNetworkRegretsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: latestOneOutInfererInfererNetworkRegretsIter")
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId1:         keyValue.Key.K2(),
			ActorId2:         keyValue.Key.K3(),
			TimestampedValue: &keyValue.Value,
		}
		latestOneOutInfererInfererNetworkRegrets = append(latestOneOutInfererInfererNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}
	data.LatestOneOutInfererInfererNetworkRegrets = latestOneOutInfererInfererNetworkRegrets

	// latestOneOutInfererForecasterNetworkRegrets
	latestOneOutInfererForecasterNetworkRegrets := make([]*types.TopicIdActorIdActorIdTimeStampedValue, 0)
	latestOneOutInfererForecasterNetworkRegretsIter, err := k.latestOneOutInfererForecasterNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate latest one out inferer forecaster network regrets")
	}
	for ; latestOneOutInfererForecasterNetworkRegretsIter.Valid(); latestOneOutInfererForecasterNetworkRegretsIter.Next() {
		keyValue, err := latestOneOutInfererForecasterNetworkRegretsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: latestOneOutInfererForecasterNetworkRegretsIter")
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId1:         keyValue.Key.K2(),
			ActorId2:         keyValue.Key.K3(),
			TimestampedValue: &keyValue.Value,
		}
		latestOneOutInfererForecasterNetworkRegrets = append(latestOneOutInfererForecasterNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}
	data.LatestOneOutInfererForecasterNetworkRegrets = latestOneOutInfererForecasterNetworkRegrets

	// latestOneOutForecasterInfererNetworkRegrets
	latestOneOutForecasterInfererNetworkRegrets := make([]*types.TopicIdActorIdActorIdTimeStampedValue, 0)
	latestOneOutForecasterInfererNetworkRegretsIter, err := k.latestOneOutForecasterInfererNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate latest one out forecaster inferer network regrets")
	}
	for ; latestOneOutForecasterInfererNetworkRegretsIter.Valid(); latestOneOutForecasterInfererNetworkRegretsIter.Next() {
		keyValue, err := latestOneOutForecasterInfererNetworkRegretsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: latestOneOutForecasterInfererNetworkRegretsIter")
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId1:         keyValue.Key.K2(),
			ActorId2:         keyValue.Key.K3(),
			TimestampedValue: &keyValue.Value,
		}
		latestOneOutForecasterInfererNetworkRegrets = append(latestOneOutForecasterInfererNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}
	data.LatestOneOutForecasterInfererNetworkRegrets = latestOneOutForecasterInfererNetworkRegrets

	// latestOneOutForecasterForecasterNetworkRegrets
	latestOneOutForecasterForecasterNetworkRegrets := make([]*types.TopicIdActorIdActorIdTimeStampedValue, 0)
	latestOneOutForecasterForecasterNetworkRegretsIter, err := k.latestOneOutForecasterForecasterNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate latest one out forecaster forecaster network regrets")
	}
	for ; latestOneOutForecasterForecasterNetworkRegretsIter.Valid(); latestOneOutForecasterForecasterNetworkRegretsIter.Next() {
		keyValue, err := latestOneOutForecasterForecasterNetworkRegretsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: latestOneOutForecasterForecasterNetworkRegretsIter")
		}
		topicIdActorIdTimeStampedValue := types.TopicIdActorIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId1:         keyValue.Key.K2(),
			ActorId2:         keyValue.Key.K3(),
			TimestampedValue: &keyValue.Value,
		}
		latestOneOutForecasterForecasterNetworkRegrets = append(latestOneOutForecasterForecasterNetworkRegrets, &topicIdActorIdTimeStampedValue)
	}
	data.LatestOneOutForecasterForecasterNetworkRegrets = latestOneOutForecasterForecasterNetworkRegrets

	// latestOneInForecasterNetworkRegrets
	latestOneInForecasterNetworkRegrets := make([]*types.TopicIdActorIdActorIdTimeStampedValue, 0)
	latestOneInForecasterNetworkRegretsIter, err := k.latestOneInForecasterNetworkRegrets.Iterate(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to iterate latest one in forecaster network regrets")
	}
	for ; latestOneInForecasterNetworkRegretsIter.Valid(); latestOneInForecasterNetworkRegretsIter.Next() {
		keyValue, err := latestOneInForecasterNetworkRegretsIter.KeyValue()
		if err != nil {
			return errors.Wrap(err, "failed to get key value: latestOneInForecasterNetworkRegretsIter")
		}
		topicIdActorIdActorIdTimeStampedValue := types.TopicIdActorIdActorIdTimeStampedValue{
			TopicId:          keyValue.Key.K1(),
			ActorId1:         keyValue.Key.K2(),
			ActorId2:         keyValue.Key.K3(),
			TimestampedValue: &keyValue.Value,
		}
		latestOneInForecasterNetworkRegrets = append(latestOneInForecasterNetworkRegrets, &topicIdActorIdActorIdTimeStampedValue)
	}
	data.LatestOneInForecasterNetworkRegrets = latestOneInForecasterNetworkRegrets

	return nil
}
