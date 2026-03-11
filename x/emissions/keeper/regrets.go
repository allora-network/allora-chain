package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

func NewRegretsKeeper(cdc codec.BinaryCodec, sb *collections.SchemaBuilder, topicKeeper *TopicKeeper) *RegretsKeeper {
	return &RegretsKeeper{
		latestInfererNetworkRegrets:                    collections.NewMap(sb, types.InfererNetworkRegretsKey, "inferer_network_regrets", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestForecasterNetworkRegrets:                 collections.NewMap(sb, types.ForecasterNetworkRegretsKey, "forecaster_network_regrets", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestOneInForecasterNetworkRegrets:            collections.NewMap(sb, types.OneInForecasterNetworkRegretsKey, "one_in_forecaster_network_regrets", collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestNaiveInfererNetworkRegrets:               collections.NewMap(sb, types.LatestNaiveInfererNetworkRegretsKey, "latest_naive_inferer_network_regrets", collections.PairKeyCodec(collections.Uint64Key, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestOneOutInfererInfererNetworkRegrets:       collections.NewMap(sb, types.LatestOneOutInfererInfererNetworkRegretsKey, "latest_one_out_inferer_inferer_network_regrets", collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestOneOutInfererForecasterNetworkRegrets:    collections.NewMap(sb, types.LatestOneOutInfererForecasterNetworkRegretsKey, "latest_one_out_inferer_forecaster_network_regrets", collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestOneOutForecasterInfererNetworkRegrets:    collections.NewMap(sb, types.LatestOneOutForecasterInfererNetworkRegretsKey, "latest_one_out_forecaster_inferer_network_regrets", collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		latestOneOutForecasterForecasterNetworkRegrets: collections.NewMap(sb, types.LatestOneOutForecasterForecasterNetworkRegretsKey, "latest_one_out_forecaster_forecaster_network_regrets", collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey, collections.StringKey), codec.CollValue[types.TimestampedValue](cdc)),
		topicKeeper: topicKeeper,
	}
}

type RegretsKeeper struct {
	// map of (topic, worker) -> regret of worker from comparing loss of worker relative to loss of other inferers
	latestInfererNetworkRegrets collections.Map[collections.Pair[TopicId, ActorId], types.TimestampedValue]
	// map of (topic, worker) -> regret of worker from comparing loss of worker relative to loss of other forecasters
	latestForecasterNetworkRegrets collections.Map[collections.Pair[TopicId, ActorId], types.TimestampedValue]
	// map of (topic, forecaster, inferer) -> R^+_{ij_kk} regret of forecaster loss from comparing one-in loss with
	// all network inferer (3rd index) regrets L_ij made under the regime of the one-in forecaster (2nd index)
	latestOneInForecasterNetworkRegrets collections.Map[collections.Triple[TopicId, ActorId, ActorId], types.TimestampedValue]
	// map of (topic id, inferer) -> regret
	latestNaiveInfererNetworkRegrets collections.Map[collections.Pair[TopicId, ActorId], types.TimestampedValue]
	// map of (topic id , one out inferer, inferer)-> regret
	latestOneOutInfererInfererNetworkRegrets collections.Map[collections.Triple[TopicId, ActorId, ActorId], types.TimestampedValue]
	// map of (topicId, oneOutInferer, forecaster) -> regret
	latestOneOutInfererForecasterNetworkRegrets collections.Map[collections.Triple[TopicId, ActorId, ActorId], types.TimestampedValue]
	// map of (topicId, oneOutInferer, inferer) -> regret
	latestOneOutForecasterInfererNetworkRegrets collections.Map[collections.Triple[TopicId, ActorId, ActorId], types.TimestampedValue]
	// map of (topicId, oneOutForecaster, forecaster) -> regret
	latestOneOutForecasterForecasterNetworkRegrets collections.Map[collections.Triple[TopicId, ActorId, ActorId], types.TimestampedValue]
	// topic keeper
	topicKeeper *TopicKeeper
}

func (k *RegretsKeeper) SetInfererNetworkRegret(ctx context.Context, topicId TopicId, worker ActorId, regret types.TimestampedValue) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrap(err, "error validating worker id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join(topicId, worker)
	return k.latestInfererNetworkRegrets.Set(ctx, key, regret)
}

// Returns the regret of a inferer from comparing loss of inferer relative to loss of other inferers
// Returns (0, true) if no regret is found
func (k *RegretsKeeper) GetInfererNetworkRegret(
	ctx context.Context, topicId TopicId, worker ActorId) (
	regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join(topicId, worker)
	regret, err = k.latestInfererNetworkRegrets.Get(ctx, key)

	if errors.Is(err, collections.ErrNotFound) {
		topic, err := k.topicKeeper.GetTopic(ctx, topicId)
		if err != nil {
			return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
		}
		return types.TimestampedValue{
			BlockHeight: 0,
			Value:       topic.InitialRegret,
		}, true, nil
	} else if err != nil {
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting inferer network regret")
	}
	return regret, false, nil
}

func (k *RegretsKeeper) SetForecasterNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	worker ActorId,
	regret types.TimestampedValue,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(worker); err != nil {
		return errorsmod.Wrap(err, "error validating worker id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join(topicId, worker)
	return k.latestForecasterNetworkRegrets.Set(ctx, key, regret)
}

// Returns the regret of a forecaster from comparing loss of forecaster relative to loss of other forecasters
// Returns (0, true) if no regret is found
func (k *RegretsKeeper) GetForecasterNetworkRegret(
	ctx context.Context, topicId TopicId, worker ActorId) (
	regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join(topicId, worker)
	regret, err = k.latestForecasterNetworkRegrets.Get(ctx, key)

	if errors.Is(err, collections.ErrNotFound) {
		topic, err := k.topicKeeper.GetTopic(ctx, topicId)
		if err != nil {
			return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
		}
		return types.TimestampedValue{
			BlockHeight: 0,
			Value:       topic.InitialRegret,
		}, true, nil
	} else if err != nil {
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting forecaster network regret")
	}

	return regret, false, nil
}

func (k *RegretsKeeper) SetOneInForecasterNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	oneInForecaster ActorId,
	inferer ActorId,
	regret types.TimestampedValue,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(oneInForecaster); err != nil {
		return errorsmod.Wrap(err, "error validating one in forecaster id")
	}
	if err := types.ValidateBech32(inferer); err != nil {
		return errorsmod.Wrap(err, "error validating inferer id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join3(topicId, oneInForecaster, inferer)
	return k.latestOneInForecasterNetworkRegrets.Set(ctx, key, regret)
}

// Returns the regret of a forecaster from comparing loss of forecaster relative to loss of other forecasters
// Returns (0, true) if no regret is found
func (k *RegretsKeeper) GetOneInForecasterNetworkRegret(
	ctx context.Context, topicId TopicId, oneInForecaster ActorId, inferer ActorId) (
	regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join3(topicId, oneInForecaster, inferer)
	regret, err = k.latestOneInForecasterNetworkRegrets.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		topic, err := k.topicKeeper.GetTopic(ctx, topicId)
		if err != nil {
			return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
		}
		return types.TimestampedValue{
			BlockHeight: 0,
			Value:       topic.InitialRegret,
		}, true, nil
	} else if err != nil {
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting one in forecaster network regret")
	}
	return regret, false, nil
}

func (k *RegretsKeeper) SetNaiveInfererNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	inferer ActorId,
	regret types.TimestampedValue,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(inferer); err != nil {
		return errorsmod.Wrap(err, "error validating inferer id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join(topicId, inferer)
	return k.latestNaiveInfererNetworkRegrets.Set(ctx, key, regret)
}

func (k *RegretsKeeper) GetNaiveInfererNetworkRegret(ctx context.Context, topicId TopicId, inferer ActorId) (
	regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join(topicId, inferer)
	regret, err = k.latestNaiveInfererNetworkRegrets.Get(ctx, key)

	if errors.Is(err, collections.ErrNotFound) {
		topic, err := k.topicKeeper.GetTopic(ctx, topicId)
		if err != nil {
			return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
		}
		return types.TimestampedValue{
			BlockHeight: 0,
			Value:       topic.InitialRegret,
		}, true, nil
	} else if err != nil {
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting naive inferer network regret")
	}
	return regret, false, nil
}

func (k *RegretsKeeper) SetOneOutInfererInfererNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	oneOutInferer ActorId,
	inferer ActorId,
	regret types.TimestampedValue,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(oneOutInferer); err != nil {
		return errorsmod.Wrap(err, "error validating one out inferer id")
	}
	if err := types.ValidateBech32(inferer); err != nil {
		return errorsmod.Wrap(err, "error validating inferer id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join3(topicId, oneOutInferer, inferer)
	return k.latestOneOutInfererInfererNetworkRegrets.Set(ctx, key, regret)
}

// return the one out inferer regret, how much the one out inferer affects the network loss
// if no prior is found, return the initial regret of the topic
func (k *RegretsKeeper) GetOneOutInfererInfererNetworkRegret(
	ctx context.Context, topicId TopicId, oneOutInferer ActorId, inferer ActorId) (
	regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join3(topicId, oneOutInferer, inferer)
	regret, err = k.latestOneOutInfererInfererNetworkRegrets.Get(ctx, key)

	if errors.Is(err, collections.ErrNotFound) {
		topic, err := k.topicKeeper.GetTopic(ctx, topicId)
		if err != nil {
			return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
		}
		return types.TimestampedValue{
			BlockHeight: 0,
			Value:       topic.InitialRegret,
		}, true, nil
	} else if err != nil {
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting one out inferer inferer network regret")
	}
	return regret, false, nil
}

func (k *RegretsKeeper) SetOneOutInfererForecasterNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	oneOutInferer ActorId,
	forecaster ActorId,
	regret types.TimestampedValue,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(oneOutInferer); err != nil {
		return errorsmod.Wrap(err, "error validating one out inferer id")
	}
	if err := types.ValidateBech32(forecaster); err != nil {
		return errorsmod.Wrap(err, "error validating forecaster id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join3(topicId, oneOutInferer, forecaster)
	return k.latestOneOutInfererForecasterNetworkRegrets.Set(ctx, key, regret)
}

// return the one out inferer forecaster regret, how much that inferer affects the forecast loss
// if no prior is found, return the initial regret of the topic
func (k *RegretsKeeper) GetOneOutInfererForecasterNetworkRegret(
	ctx context.Context, topicId TopicId, oneOutInferer ActorId, forecaster ActorId) (
	regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join3(topicId, oneOutInferer, forecaster)
	regret, err = k.latestOneOutInfererForecasterNetworkRegrets.Get(ctx, key)

	if errors.Is(err, collections.ErrNotFound) {
		topic, err := k.topicKeeper.GetTopic(ctx, topicId)
		if err != nil {
			return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
		}
		return types.TimestampedValue{
			BlockHeight: 0,
			Value:       topic.InitialRegret,
		}, true, nil
	} else if err != nil {
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting one out inferer forecaster network regret")
	}
	return regret, false, nil
}

func (k *RegretsKeeper) SetOneOutForecasterInfererNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	oneOutForecaster ActorId,
	inferer ActorId,
	regret types.TimestampedValue,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(oneOutForecaster); err != nil {
		return errorsmod.Wrap(err, "error validating one out forecaster id")
	}
	if err := types.ValidateBech32(inferer); err != nil {
		return errorsmod.Wrap(err, "error validating inferer id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join3(topicId, oneOutForecaster, inferer)
	return k.latestOneOutForecasterInfererNetworkRegrets.Set(ctx, key, regret)
}

// return the one out forecaster inferer regret, how much that forecaster affects the inferer network loss
// if no prior is found, return the initial regret of the topic
func (k *RegretsKeeper) GetOneOutForecasterInfererNetworkRegret(
	ctx context.Context, topicId TopicId, oneOutForecaster ActorId, inferer ActorId) (
	regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join3(topicId, oneOutForecaster, inferer)
	regret, err = k.latestOneOutForecasterInfererNetworkRegrets.Get(ctx, key)

	if errors.Is(err, collections.ErrNotFound) {
		topic, err := k.topicKeeper.GetTopic(ctx, topicId)
		if err != nil {
			return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
		}
		return types.TimestampedValue{
			BlockHeight: 0,
			Value:       topic.InitialRegret,
		}, true, nil
	} else if err != nil {
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting one out forecaster inferer network regret")
	}
	return regret, false, nil
}

func (k *RegretsKeeper) SetOneOutForecasterForecasterNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	oneOutForecaster ActorId,
	forecaster ActorId,
	regret types.TimestampedValue,
) error {
	if err := types.ValidateTopicId(topicId); err != nil {
		return errorsmod.Wrap(err, "error validating topic id")
	}
	if err := types.ValidateBech32(oneOutForecaster); err != nil {
		return errorsmod.Wrap(err, "error validating one out forecaster id")
	}
	if err := types.ValidateBech32(forecaster); err != nil {
		return errorsmod.Wrap(err, "error validating forecaster id")
	}
	if err := regret.Validate(); err != nil {
		return errorsmod.Wrap(err, "error validating regret")
	}
	key := collections.Join3(topicId, oneOutForecaster, forecaster)
	return k.latestOneOutForecasterForecasterNetworkRegrets.Set(ctx, key, regret)
}

func (k *RegretsKeeper) GetOneOutForecasterForecasterNetworkRegret(
	ctx context.Context,
	topicId TopicId,
	oneOutForecaster ActorId,
	forecaster ActorId,
) (regret types.TimestampedValue, noPrior bool, err error) {
	key := collections.Join3(topicId, oneOutForecaster, forecaster)
	regret, err = k.latestOneOutForecasterForecasterNetworkRegrets.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		topic, err := k.topicKeeper.GetTopic(ctx, topicId)
		if err != nil {
			return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting topic")
		}
		return types.TimestampedValue{
			BlockHeight: 0,
			Value:       topic.InitialRegret,
		}, true, nil
	} else if err != nil {
		return types.TimestampedValue{}, false, errorsmod.Wrap(err, "error getting one out forecaster forecaster network regret")
	}
	return regret, false, nil
}
