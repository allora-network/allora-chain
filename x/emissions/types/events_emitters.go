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
		ctx.Logger().Warn("Error emitting NewInfererScoresSetEvent", "error", err)
	}
}

func EmitNewForecasterScoresSetEvent(ctx sdk.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.FORECASTER_SCORE_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewScoresSetEventBase(ActorType_ACTOR_TYPE_FORECASTER, scores))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewForecasterScoresSetEvent", "error", err)
	}
}

func EmitNewReputerScoresSetEvent(ctx sdk.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.REPUTER_SOCRE_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewScoresSetEventBase(ActorType_ACTOR_TYPE_REPUTER, scores))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewReputerScoresSetEvent", "error", err)
	}
}

func EmitNewNetworkLossSetEvent(ctx sdk.Context, topicId TopicId, blockHeight BlockHeight, lossBundle ValueBundle) {
	metrics.IncrProducerEventCount(metrics.NETWORK_LOSS_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewNetworkLossSetEventBase(topicId, blockHeight, lossBundle))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewNetworkLossSetEvent", "error", err)
	}
}

func EmitNewNetworkInferencesEvent(ctx sdk.Context, topicId TopicId, blockHeight BlockHeight, networkInferences ValueBundle) {
	metrics.IncrProducerEventCount(metrics.NETWORK_INFERENCES_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewNetworkInferencesEventBase(topicId, blockHeight, networkInferences))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewNetworkLossSetEvent", "error", err)
	}
}

func EmitNewForecastTaskUtilityScoreSetEvent(ctx sdk.Context, topicId TopicId, score alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.FORECAST_TASK_SCORE_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewForecastTaskScoreSetEventBase(topicId, score))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewForecastTaskUtilityScoreSetEvent", "error", err)
	}
}

func EmitNewActorEMAScoresSetEvent(ctx sdk.Context, actorType ActorType, scores []Score, activations map[string]bool) {
	if len(scores) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.WORKER_EMA_SCORE_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewEMAScoresSetEventBase(actorType, scores, activations))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewActorEMAScoresSetEvent", "error", err)
	}
}

/// Rewards

func EmitNewInfererRewardsSettledEvent(ctx sdk.Context, blockHeight, blockHeightTx BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.INFERER_REWARD_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, blockHeight, blockHeightTx, rewards))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewInfererRewardsSettledEvent", "error", err)
	}
}

func EmitNewForecasterRewardsSettledEvent(ctx sdk.Context, blockHeight, blockHeightTx BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.FORECASTER_REWARD_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_ACTOR_TYPE_FORECASTER, blockHeight, blockHeightTx, rewards))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewForecasterRewardsSettledEvent", "error", err)
	}
}

func EmitNewReputerAndDelegatorRewardsSettledEvent(ctx sdk.Context, blockHeight, blockHeightTx BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.REPUTER_REWARD_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_ACTOR_TYPE_REPUTER, blockHeight, blockHeightTx, rewards))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewReputerAndDelegatorRewardsSettledEvent", "error", err)
	}
}

func EmitNewTopicRewardSetEvent(ctx sdk.Context, topicRewards map[uint64]*alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.TOPIC_REWARD_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewTopicRewardSetEventBase(topicRewards))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewTopicRewardSetEvent", "error", err)
	}
}

/// Commits

func EmitNewWorkerLastCommitSetEvent(ctx sdk.Context, topicId TopicId, height BlockHeight, nonce *Nonce) {
	metrics.IncrProducerEventCount(metrics.WORKER_LAST_COMMIT_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewWorkerLastCommitSetEventBase(topicId, height, nonce))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewWorkerLastCommitSetEvent", "error", err)
	}
}

func EmitNewReputerLastCommitSetEvent(ctx sdk.Context, topicId TopicId, height BlockHeight, nonce *Nonce) {
	metrics.IncrProducerEventCount(metrics.REPUTER_LAST_COMMIT_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewReputerLastCommitSetEventBase(topicId, height, nonce))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewReputerLastCommitSetEvent", "error", err)
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
		ctx.Logger().Warn("Error emitting NewListeningCoefficientsSetEvent", "error", err)
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
		ctx.Logger().Warn("Error emitting NewInfererNetworkRegretSetEvent", "error", err)
	}
}

func EmitNewForecasterNetworkRegretSetEvent(ctx sdk.Context, topicId uint64, blockHeight int64, addresses []string, regrets []alloraMath.Dec) {
	if len(addresses) == 0 || len(regrets) == 0 {
		return
	}
	metrics.IncrProducerEventCount(metrics.FORECASTER_NETWORK_REGRET_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewForecasterNetworkRegretSetEventBase(topicId, blockHeight, addresses, regrets))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewForecasterNetworkRegretSetEvent", "error", err)
	}
}

func EmitNewNaiveInfererNetworkRegretSetEvent(ctx sdk.Context, topicId uint64, blockHeight int64, addresses []string, regrets []alloraMath.Dec) {
	if len(addresses) == 0 || len(regrets) == 0 {
		return
	}
	metrics.IncrProducerEventCount(metrics.NAIVE_INFERER_NETWORK_REGRET_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewNaiveInfererNetworkRegretSetEventBase(topicId, blockHeight, addresses, regrets))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewNaiveInfererNetworkRegretSetEvent", "error", err)
	}
}

func EmitNewTopicInitialRegretSetEvent(ctx sdk.Context, topicId uint64, blockHeight int64, regret alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.TOPIC_INITIAL_REGRET_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewTopicInitialRegretSetEventBase(topicId, blockHeight, regret))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewTopicInitialRegretSetEvent", "error", err)
	}
}

func EmitNewTopicInitialEmaScoreSetEvent(ctx sdk.Context, actorType ActorType, topicId uint64, blockHeight int64, score alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.TOPIC_INITIAL_EMA_SCORE_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewTopicInitialEmaScoreSetEventBase(actorType, topicId, blockHeight, score))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewTopicInitialEmaScoreSetEvent", "error", err)
	}
}

// Individual events
func EmitNewRegretStdNormSetEvent(ctx sdk.Context, topicId uint64, blockHeight int64, stdNorm alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.REGRET_STDNORM_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewRegretStdNormSetEventBase(topicId, blockHeight, stdNorm))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewRegretStdNormSetEvent", "error", err)
	}
}

func EmitNewInfererWeightSetEvent(ctx sdk.Context, topicId uint64, blockHeight int64, address string, weight alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.INFERER_WEIGHTS_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewInfererWeightSetEventBase(topicId, blockHeight, address, weight))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewInfererWeightSetEvent", "error", err)
	}
}

func EmitNewForecasterWeightSetEvent(ctx sdk.Context, topicId uint64, blockHeight int64, address string, weight alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.FORECASTER_WEIGHTS_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewForecasterWeightSetEventBase(topicId, blockHeight, address, weight))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewForecasterWeightSetEvent", "error", err)
	}
}

/// Previous Percentage Reward

func EmitPreviousPercentageRewardToStakedReputersSetEvent(ctx sdk.Context, blockHeight int64, percentage alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.PREVIOUS_PERCENTAGE_REWARD_TO_STAKED_REPUTERS_SET_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewPreviousPercentageRewardToStakedReputersSetEventBase(blockHeight, percentage))
	if err != nil {
		ctx.Logger().Warn("Error emitting PreviousPercentageRewardToStakedReputersSetEvent", "error", err)
	}
}

// EmitPruneRecordsEvent emits a metric for pruning records after rewards.
func EmitPruneRecordsEvent(ctx sdk.Context, blockHeight int64, topicId uint64) {
	metrics.IncrProducerEventCount(metrics.PRUNE_RECORDS_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewPruneRecordsSetEventBase(blockHeight, topicId))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewPruneRecordsSetEventBase", "error", err)
	}
}
