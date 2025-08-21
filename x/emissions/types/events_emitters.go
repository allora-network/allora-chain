package types

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
)

// Scores

func EmitNewInfererScoresSetEvent(ctx context.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.INFERER_SCORE_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewScoresSetEventBase(ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, scores))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewInfererScoresSetEvent", "error", err)
	}
}

func EmitNewForecasterScoresSetEvent(ctx context.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.FORECASTER_SCORE_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewScoresSetEventBase(ActorType_ACTOR_TYPE_FORECASTER, scores))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewForecasterScoresSetEvent", "error", err)
	}
}

func EmitNewReputerScoresSetEvent(ctx context.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.REPUTER_SOCRE_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewScoresSetEventBase(ActorType_ACTOR_TYPE_REPUTER, scores))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewReputerScoresSetEvent", "error", err)
	}
}

func EmitNewNetworkLossSetEvent(ctx context.Context, lossBundle ValueBundle) {
	metrics.IncrProducerEventCount(metrics.NETWORK_LOSS_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewNetworkLossSetEventBase(&lossBundle))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewNetworkLossSetEvent", "error", err)
	}
}

func EmitNewNetworkInferencesEvent(ctx context.Context, networkInferences ValueBundle) {
	metrics.IncrProducerEventCount(metrics.NETWORK_INFERENCES_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewNetworkInferencesEventBase(&networkInferences))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewNetworkLossSetEvent", "error", err)
	}
}

func EmitNewInsertInfererPayloadEvent(ctx context.Context, bundle *WorkerDataBundle) {
	metrics.IncrProducerEventCount(metrics.INSERT_INFERER_PAYLOAD_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewInsertInfererPayloadEventBase(bundle))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewForecasterPayloadEvent", "error", err)
	}
}

func EmitNewInsertForecasterPayloadEvent(ctx context.Context, bundle *WorkerDataBundle) {
	metrics.IncrProducerEventCount(metrics.INSERT_FORECASTER_PAYLOAD_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewInsertForecasterPayloadEventBase(bundle))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewInfererPayloadEvent", "error", err)
	}
}

func EmitNewCreateNewTopicEvent(ctx context.Context, topic *Topic) {
	metrics.IncrProducerEventCount(metrics.CREATE_NEW_TOPIC_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewCreateNewTopicEventBase(topic))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewCreateNewTopicEvent", "error", err)
	}
}

func EmitNewAddStakeEvent(ctx context.Context, topicId TopicId, reputer, delegator string, amount math.Int) {
	metrics.IncrProducerEventCount(metrics.ADD_STAKE_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewAddStakeEventBase(topicId, reputer, delegator, amount))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewAddReputerStakeEvent", "error", err)
	}
}

func EmitNewRewardDelegateStakeEvent(ctx context.Context, topicId TopicId, reputer, delegator string, amount alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.REWARD_DELEGATE_STAKE_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewRewardDelegateStakeEventBase(topicId, reputer, delegator, amount))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewRewardDelegateStakeEvent", "error", err)
	}
}

func EmitNewInsertReputerPayloadEvent(ctx context.Context, bundle *ReputerValueBundle) {
	metrics.IncrProducerEventCount(metrics.INSERT_REPUTER_PAYLOAD_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewInsertReputerPayloadEventBase(bundle))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewInsertReputerPayloadEvent", "error", err)
	}
}

func EmitNewReputerRegisteredEvent(ctx context.Context, topicId TopicId, reputer, owner string) {
	metrics.IncrProducerEventCount(metrics.REPUTER_REGISTERED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewReputerRegisteredEventBase(topicId, reputer, owner))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewReputerRegisteredEvent", "error", err)
	}
}

func EmitNewWorkerRegisteredEvent(ctx context.Context, topicId TopicId, worker, owner string) {
	metrics.IncrProducerEventCount(metrics.WORKER_REGISTERED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewWorkerRegisteredEventBase(topicId, worker, owner))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewWorkerRegisteredEvent", "error", err)
	}
}

func EmitNewReputerUnregisteredEvent(ctx context.Context, topicId TopicId, reputer string) {
	metrics.IncrProducerEventCount(metrics.REPUTER_UNREGISTERED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewReputerUnregisteredEventBase(topicId, reputer))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewReputerUnregisteredEvent", "error", err)
	}
}

func EmitNewWorkerUnregisteredEvent(ctx context.Context, topicId TopicId, worker string) {
	metrics.IncrProducerEventCount(metrics.WORKER_UNREGISTERED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewWorkerUnregisteredEventBase(topicId, worker))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewWorkerUnregisteredEvent", "error", err)
	}
}

func EmitNewFundTopicEvent(ctx context.Context, topicId TopicId, funder string, amount math.Int) {
	metrics.IncrProducerEventCount(metrics.FUND_TOPIC_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewFundTopicEventBase(topicId, funder, amount))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewFundTopicEvent", "error", err)
	}
}

func EmitNewParamsSetEvent(ctx context.Context, params Params) {
	metrics.IncrProducerEventCount(metrics.PARAMS_SET_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewParamsSetEventBase(params))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewParamsSetEvent", "error", err)
	}
}

func EmitNewWhitelistAdminAddedEvent(ctx context.Context, address string) {
	metrics.IncrProducerEventCount(metrics.WHITELIST_ADMIN_ADDED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewWhitelistAdminAddedEventBase(address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewWhitelistAdminAddedEvent", "error", err)
	}
}

func EmitNewWhitelistAdminRemovedEvent(ctx context.Context, address string) {
	metrics.IncrProducerEventCount(metrics.WHITELIST_ADMIN_REMOVED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewWhitelistAdminRemovedEventBase(address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewWhitelistAdminRemovedEvent", "error", err)
	}
}

func EmitNewGlobalWhitelistAddedEvent(ctx context.Context, address string) {
	metrics.IncrProducerEventCount(metrics.GLOBAL_WHITELIST_ADDED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewGlobalWhitelistAddedEventBase(address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewGlobalWhitelistSetEvent", "error", err)
	}
}

func EmitNewGlobalWhitelistRemovedEvent(ctx context.Context, address string) {
	metrics.IncrProducerEventCount(metrics.GLOBAL_WHITELIST_REMOVED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewGlobalWhitelistRemovedEventBase(address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewGlobalWhitelistRemovedEvent", "error", err)
	}
}

func EmitNewGlobalWorkerWhitelistAddedEvent(ctx context.Context, address string) {
	metrics.IncrProducerEventCount(metrics.GLOBAL_WORKER_WHITELIST_ADDED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewGlobalWorkerWhitelistAddedEventBase(address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewGlobalWorkerWhitelistAddedEvent", "error", err)
	}
}

func EmitNewGlobalWorkerWhitelistRemovedEvent(ctx context.Context, address string) {
	metrics.IncrProducerEventCount(metrics.GLOBAL_WORKER_WHITELIST_REMOVED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewGlobalWorkerWhitelistRemovedEventBase(address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewGlobalWorkerWhitelistRemovedEvent", "error", err)
	}
}

func EmitNewGlobalReputerWhitelistAddedEvent(ctx context.Context, address string) {
	metrics.IncrProducerEventCount(metrics.GLOBAL_REPUTER_WHITELIST_ADDED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewGlobalReputerWhitelistAddedEventBase(address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewGlobalReputerWhitelistAddedEvent", "error", err)
	}
}


func EmitNewGlobalReputerWhitelistRemovedEvent(ctx context.Context, address string) {
	metrics.IncrProducerEventCount(metrics.GLOBAL_REPUTER_WHITELIST_REMOVED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewGlobalReputerWhitelistRemovedEventBase(address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewGlobalReputerWhitelistRemovedEvent", "error", err)
	}
}

func EmitNewGlobalAdminWhitelistAddedEvent(ctx context.Context, address string) {
	metrics.IncrProducerEventCount(metrics.GLOBAL_ADMIN_WHITELIST_ADDED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewGlobalAdminWhitelistAddedEventBase(address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewGlobalAdminWhitelistAddedEvent", "error", err)
	}
}

func EmitNewGlobalAdminWhitelistRemovedEvent(ctx context.Context, address string) {
	metrics.IncrProducerEventCount(metrics.GLOBAL_ADMIN_WHITELIST_REMOVED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewGlobalAdminWhitelistRemovedEventBase(address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewGlobalAdminWhitelistRemovedEvent", "error", err)
	}
}

func EmitNewOutlierResistantNetworkInferencesEvent(ctx sdk.Context, topicId TopicId, blockHeight BlockHeight, networkInferences ValueBundle) {
	err := ctx.EventManager().EmitTypedEvent(NewOutlierResistantNetworkInferencesEventBase(topicId, blockHeight, networkInferences))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewNetworkLossSetEvent", "error", err)
	}
}

func EmitNewForecastTaskUtilityScoreSetEvent(ctx sdk.Context, topicId TopicId, score alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.FORECAST_TASK_SCORE_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewForecastTaskScoreSetEventBase(topicId, score))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewForecastTaskUtilityScoreSetEvent", "error", err)
	}
}

func EmitNewTopicWorkerWhitelistEnabledEvent(ctx context.Context, topicId TopicId) {
	metrics.IncrProducerEventCount(metrics.TOPIC_WORKER_WHITELIST_ENABLED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewTopicWorkerWhitelistEnabledEventBase(topicId))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewTopicWorkerWhitelistEnabledEvent", "error", err)
	}
}

func EmitNewTopicWorkerWhitelistDisabledEvent(ctx context.Context, topicId TopicId) {
	metrics.IncrProducerEventCount(metrics.TOPIC_WORKER_WHITELIST_DISABLED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewTopicWorkerWhitelistDisabledEventBase(topicId))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewTopicWorkerWhitelistDisabledEvent", "error", err)
	}
}

func EmitNewTopicReputerWhitelistEnabledEvent(ctx context.Context, topicId TopicId) {
	metrics.IncrProducerEventCount(metrics.TOPIC_REPUTER_WHITELIST_ENABLED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewTopicReputerWhitelistEnabledEventBase(topicId))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewTopicReputerWhitelistEnabledEvent", "error", err)
	}
}

func EmitNewTopicReputerWhitelistDisabledEvent(ctx context.Context, topicId TopicId) {
	metrics.IncrProducerEventCount(metrics.TOPIC_REPUTER_WHITELIST_DISABLED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewTopicReputerWhitelistDisabledEventBase(topicId))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewTopicReputerWhitelistDisabledEvent", "error", err)
	}
}

func EmitNewTopicCreatorWhitelistAddedEvent(ctx context.Context, address string) {
	metrics.IncrProducerEventCount(metrics.TOPIC_CREATOR_WHITELIST_ADDED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewTopicCreatorWhitelistAddedEventBase(address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewTopicCreatorWhitelistAddedEvent", "error", err)
	}
}

func EmitNewTopicCreatorWhitelistRemovedEvent(ctx context.Context, address string) {
	metrics.IncrProducerEventCount(metrics.TOPIC_CREATOR_WHITELIST_REMOVED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewTopicCreatorWhitelistRemovedEventBase(address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewTopicCreatorWhitelistRemovedEvent", "error", err)
	}
}

func EmitNewTopicWorkerWhitelistAddedEvent(ctx context.Context, topicId TopicId, address string) {
	metrics.IncrProducerEventCount(metrics.TOPIC_WORKER_WHITELIST_ADDED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewTopicWorkerWhitelistAddedEventBase(topicId, address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewTopicWorkerWhitelistAddedEvent", "error", err)
	}
}

func EmitNewTopicWorkerWhitelistRemovedEvent(ctx context.Context, topicId TopicId, address string) {
	metrics.IncrProducerEventCount(metrics.TOPIC_WORKER_WHITELIST_REMOVED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewTopicWorkerWhitelistRemovedEventBase(topicId, address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewTopicWorkerWhitelistRemovedEvent", "error", err)
	}
}

func EmitNewTopicReputerWhitelistAddedEvent(ctx context.Context, topicId TopicId, address string) {
	metrics.IncrProducerEventCount(metrics.TOPIC_REPUTER_WHITELIST_ADDED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewTopicReputerWhitelistAddedEventBase(topicId, address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewTopicReputerWhitelistAddedEvent", "error", err)
	}
}

func EmitNewTopicReputerWhitelistRemovedEvent(ctx context.Context, topicId TopicId, address string) {
	metrics.IncrProducerEventCount(metrics.TOPIC_REPUTER_WHITELIST_REMOVED_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewTopicReputerWhitelistRemovedEventBase(topicId, address))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewTopicReputerWhitelistRemovedEvent", "error", err)
	}
}

func EmitNewActorEMAScoresSetEvent(ctx context.Context, actorType ActorType, scores []Score, activations map[string]bool) {
	if len(scores) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.WORKER_EMA_SCORE_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewEMAScoresSetEventBase(actorType, scores, activations))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewActorEMAScoresSetEvent", "error", err)
	}
}

// Rewards

func EmitNewInfererRewardsSettledEvent(ctx context.Context, blockHeight, blockHeightTx BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.INFERER_REWARD_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, blockHeight, blockHeightTx, rewards))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewInfererRewardsSettledEvent", "error", err)
	}
}

func EmitNewForecasterRewardsSettledEvent(ctx context.Context, blockHeight, blockHeightTx BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.FORECASTER_REWARD_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_ACTOR_TYPE_FORECASTER, blockHeight, blockHeightTx, rewards))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewForecasterRewardsSettledEvent", "error", err)
	}
}

func EmitNewReputerAndDelegatorRewardsSettledEvent(ctx context.Context, blockHeight, blockHeightTx BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	metrics.IncrProducerEventCount(metrics.REPUTER_REWARD_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_ACTOR_TYPE_REPUTER, blockHeight, blockHeightTx, rewards))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewReputerAndDelegatorRewardsSettledEvent", "error", err)
	}
}

func EmitNewTopicRewardSetEvent(ctx context.Context, topicRewards map[uint64]*alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.TOPIC_REWARD_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewTopicRewardSetEventBase(topicRewards))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewTopicRewardSetEvent", "error", err)
	}
}

// Commits

func EmitNewWorkerLastCommitSetEvent(ctx context.Context, topicId TopicId, height BlockHeight, nonce *Nonce) {
	metrics.IncrProducerEventCount(metrics.WORKER_LAST_COMMIT_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewWorkerLastCommitSetEventBase(topicId, height, nonce))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewWorkerLastCommitSetEvent", "error", err)
	}
}

func EmitNewReputerLastCommitSetEvent(ctx context.Context, topicId TopicId, height BlockHeight, nonce *Nonce) {
	metrics.IncrProducerEventCount(metrics.REPUTER_LAST_COMMIT_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewReputerLastCommitSetEventBase(topicId, height, nonce))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewReputerLastCommitSetEvent", "error", err)
	}
}

// Listening Coefficients

func EmitNewListeningCoefficientsSetEvent(ctx context.Context, actorType ActorType, topicId uint64, blockHeight int64, addresses []string, coefficients []alloraMath.Dec) {
	if len(addresses) == 0 || len(coefficients) == 0 {
		return
	}
	metrics.IncrProducerEventCount(metrics.LISTENING_COEFFICIENTS_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewListeningCoefficientsSetEventBase(topicId, blockHeight, addresses, actorType, coefficients))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewListeningCoefficientsSetEvent", "error", err)
	}
}

// Regrets

func EmitNewInfererNetworkRegretSetEvent(ctx context.Context, topicId uint64, blockHeight int64, addresses []string, regrets []alloraMath.Dec) {
	if len(addresses) == 0 || len(regrets) == 0 {
		return
	}
	metrics.IncrProducerEventCount(metrics.INFERER_NETWORK_REGRET_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewInfererNetworkRegretSetEventBase(topicId, blockHeight, addresses, regrets))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewInfererNetworkRegretSetEvent", "error", err)
	}
}

func EmitNewForecasterNetworkRegretSetEvent(ctx context.Context, topicId uint64, blockHeight int64, addresses []string, regrets []alloraMath.Dec) {
	if len(addresses) == 0 || len(regrets) == 0 {
		return
	}
	metrics.IncrProducerEventCount(metrics.FORECASTER_NETWORK_REGRET_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewForecasterNetworkRegretSetEventBase(topicId, blockHeight, addresses, regrets))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewForecasterNetworkRegretSetEvent", "error", err)
	}
}

func EmitNewNaiveInfererNetworkRegretSetEvent(ctx context.Context, topicId uint64, blockHeight int64, addresses []string, regrets []alloraMath.Dec) {
	if len(addresses) == 0 || len(regrets) == 0 {
		return
	}
	metrics.IncrProducerEventCount(metrics.NAIVE_INFERER_NETWORK_REGRET_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewNaiveInfererNetworkRegretSetEventBase(topicId, blockHeight, addresses, regrets))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewNaiveInfererNetworkRegretSetEvent", "error", err)
	}
}

func EmitNewTopicInitialRegretSetEvent(ctx context.Context, topicId uint64, blockHeight int64, regret alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.TOPIC_INITIAL_REGRET_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewTopicInitialRegretSetEventBase(topicId, blockHeight, regret))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewTopicInitialRegretSetEvent", "error", err)
	}
}

func EmitNewTopicInitialEmaScoreSetEvent(ctx context.Context, actorType ActorType, topicId uint64, blockHeight int64, score alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.TOPIC_INITIAL_EMA_SCORE_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewTopicInitialEmaScoreSetEventBase(actorType, topicId, blockHeight, score))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewTopicInitialEmaScoreSetEvent", "error", err)
	}
}

// Individual events
func EmitNewRegretStdNormSetEvent(ctx context.Context, topicId uint64, blockHeight int64, stdNorm alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.REGRET_STDNORM_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewRegretStdNormSetEventBase(topicId, blockHeight, stdNorm))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewRegretStdNormSetEvent", "error", err)
	}
}

func EmitNewInfererWeightSetEvent(ctx context.Context, topicId uint64, blockHeight int64, address string, weight alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.INFERER_WEIGHTS_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewInfererWeightSetEventBase(topicId, blockHeight, address, weight))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewInfererWeightSetEvent", "error", err)
	}
}

func EmitNewForecasterWeightSetEvent(ctx context.Context, topicId uint64, blockHeight int64, address string, weight alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.FORECASTER_WEIGHTS_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewForecasterWeightSetEventBase(topicId, blockHeight, address, weight))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewForecasterWeightSetEvent", "error", err)
	}
}

// Previous Percentage Reward

func EmitPreviousPercentageRewardToStakedReputersSetEvent(ctx context.Context, blockHeight int64, percentage alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.PREVIOUS_PERCENTAGE_REWARD_TO_STAKED_REPUTERS_SET_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewPreviousPercentageRewardToStakedReputersSetEventBase(blockHeight, percentage))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting PreviousPercentageRewardToStakedReputersSetEvent", "error", err)
	}
}

// EmitPruneRecordsEvent emits a metric for pruning records after rewards.
func EmitPruneRecordsEvent(ctx context.Context, blockHeight int64, topicId uint64) {
	metrics.IncrProducerEventCount(metrics.PRUNE_RECORDS_EVENT)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := sdkCtx.EventManager().EmitTypedEvent(NewPruneRecordsSetEventBase(blockHeight, topicId))
	if err != nil {
		sdkCtx.Logger().Warn("Error emitting NewPruneRecordsSetEventBase", "error", err)
	}
}
