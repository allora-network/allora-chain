package types

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/// Scores

func EmitNewInfererScoresSetEvent(ctx sdk.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.INFERER_SCORE_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewScoresSetEventBase(ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, scores))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewInfererScoresSetEvent: ", err.Error())
	}
}

func EmitNewForecasterScoresSetEvent(ctx sdk.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.FORECASTER_SCORE_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewScoresSetEventBase(ActorType_ACTOR_TYPE_FORECASTER, scores))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewForecasterScoresSetEvent: ", err.Error())
	}
}

func EmitNewReputerScoresSetEvent(ctx sdk.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.REPUTER_SOCRE_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewScoresSetEventBase(ActorType_ACTOR_TYPE_REPUTER, scores))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewReputerScoresSetEvent: ", err.Error())
	}
}

func EmitNewNetworkLossSetEvent(ctx sdk.Context, topicId TopicId, blockHeight BlockHeight, lossBundle ValueBundle) {
	metrics.IncrProducerEventCount(metrics.NETWORK_LOSS_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewNetworkLossSetEventBase(topicId, blockHeight, lossBundle))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewNetworkLossSetEvent: ", err.Error())
	}
}

func EmitNewForecastTaskUtilityScoreSetEvent(ctx sdk.Context, topicId TopicId, score alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.FORECAST_TASK_SCORE_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewForecastTaskScoreSetEventBase(topicId, score))
	if err != nil {
		ctx.Logger().Warn("Error emitting EmitNewReputerLastCommitSetEvent: ", err.Error())
	}
}

func EmitNewActorEMAScoresSetEvent(ctx sdk.Context, actorType ActorType, scores []Score, activations map[string]bool) {
	if len(scores) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.WORKER_EMA_SCORE_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewEMAScoresSetEventBase(actorType, scores, activations))
	if err != nil {
		ctx.Logger().Warn("Error emitting EmitNewActorEMAScoresSetEvent: ", err.Error())
	}
}

/// Rewards

func EmitNewInfererRewardsSettledEvent(ctx sdk.Context, blockHeight BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.INFERER_REWARD_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, blockHeight, rewards))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewInfererRewardsSettledEvent: ", err.Error())
	}
}

func EmitNewForecasterRewardsSettledEvent(ctx sdk.Context, blockHeight BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.FORECASTER_REWARD_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_ACTOR_TYPE_FORECASTER, blockHeight, rewards))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewForecasterRewardsSettledEvent: ", err.Error())
	}
}

func EmitNewReputerAndDelegatorRewardsSettledEvent(ctx sdk.Context, blockHeight BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.REPUTER_REWARD_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_ACTOR_TYPE_REPUTER, blockHeight, rewards))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewReputerAndDelegatorRewardsSettledEvent: ", err.Error())
	}
}

func EmitNewTopicRewardSetEvent(ctx sdk.Context, topicRewards map[uint64]*alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.TOPIC_REWARD_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewTopicRewardSetEventBase(topicRewards))
	if err != nil {
		ctx.Logger().Warn("Error emitting EmitNewTopicRewardSetEvent: ", err.Error())
	}
}

/// Commits

func EmitNewWorkerLastCommitSetEvent(ctx sdk.Context, topicId TopicId, height BlockHeight, nonce *Nonce) {
	metrics.IncrProducerEventCount(metrics.WORKER_LAST_COMMIT_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewWorkerLastCommitSetEventBase(topicId, height, nonce))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewWorkerLastCommitSetEvent: ", err.Error())
	}
}

func EmitNewReputerLastCommitSetEvent(ctx sdk.Context, topicId TopicId, height BlockHeight, nonce *Nonce) {
	metrics.IncrProducerEventCount(metrics.REPUTER_LAST_COMMIT_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewReputerLastCommitSetEventBase(topicId, height, nonce))
	if err != nil {
		ctx.Logger().Warn("Error emitting EmitNewReputerLastCommitSetEvent: ", err.Error())
	}
}

/// Listening Coefficients

func EmitNewListeningCoefficientsSetEvent(ctx sdk.Context, actorType ActorType, topicId uint64, blockHeight int64, addresses []string, coefficients []alloraMath.Dec) {
	if len(addresses) == 0 || len(coefficients) == 0 {
		return
	}
	metrics.IncrProducerEventCount(metrics.LISTENING_COEFFICIENTS_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewListeningCoefficientsSetEventBase(topicId, blockHeight, addresses, actorType, coefficients))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewListeningCoefficientsSetEvent: ", err.Error())
	}
}

/// Regrets

func EmitNewInfererNetworkRegretSetEvent(ctx sdk.Context, topicId uint64, blockHeight int64, addresses []string, regrets []alloraMath.Dec) {
	if len(addresses) == 0 || len(regrets) == 0 {
		return
	}
	metrics.IncrProducerEventCount(metrics.INFERER_NETWORK_REGRET_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewInfererNetworkRegretSetEventBase(topicId, blockHeight, addresses, regrets))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewInfererNetworkRegretSetEvent: ", err.Error())
	}
}

func EmitNewForecasterNetworkRegretSetEvent(ctx sdk.Context, topicId uint64, blockHeight int64, addresses []string, regrets []alloraMath.Dec) {
	if len(addresses) == 0 || len(regrets) == 0 {
		return
	}
	metrics.IncrProducerEventCount(metrics.FORECASTER_NETWORK_REGRET_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewForecasterNetworkRegretSetEventBase(topicId, blockHeight, addresses, regrets))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewForecasterNetworkRegretSetEvent: ", err.Error())
	}
}

func EmitNewNaiveInfererNetworkRegretSetEvent(ctx sdk.Context, topicId uint64, blockHeight int64, addresses []string, regrets []alloraMath.Dec) {
	if len(addresses) == 0 || len(regrets) == 0 {
		return
	}
	metrics.IncrProducerEventCount(metrics.NAIVE_INFERER_NETWORK_REGRET_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewNaiveInfererNetworkRegretSetEventBase(topicId, blockHeight, addresses, regrets))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewNaiveInfererNetworkRegretSetEvent: ", err.Error())
	}
}

func EmitNewTopicInitialRegretSetEvent(ctx sdk.Context, topicId uint64, blockHeight int64, regret alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.TOPIC_INITIAL_REGRET_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewTopicInitialRegretSetEventBase(topicId, blockHeight, regret))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewTopicInitialRegretSetEvent: ", err.Error())
	}
}
