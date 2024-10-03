package types

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

/// Emitters

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

func EmitNewForecastTaskUtilityScoreSetEvent(ctx sdk.Context, topicId TopicId, score alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.FORECAST_TASK_SCORE_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewForecastTaskScoreSetEventBase(topicId, score))
	if err != nil {
		ctx.Logger().Warn("Error emitting EmitNewReputerLastCommitSetEvent: ", err.Error())
	}
}

func EmitNewTopicRewardSetEvent(ctx sdk.Context, topicRewards map[uint64]*alloraMath.Dec) {
	metrics.IncrProducerEventCount(metrics.TOPIC_REWARD_EVENT)
	err := ctx.EventManager().EmitTypedEvent(NewTopicRewardSetEventBase(topicRewards))
	if err != nil {
		ctx.Logger().Warn("Error emitting EmitNewTopicRewardSetEvent: ", err.Error())
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

/// Utils

// Assumes length of `scores` is at least 1
func NewScoresSetEventBase(actorType ActorType, scores []Score) proto.Message {
	topicId := scores[0].TopicId
	blockHeight := scores[0].BlockHeight
	addresses := make([]string, len(scores))
	scoreValues := make([]alloraMath.Dec, len(scores))
	for i, score := range scores {
		addresses[i] = score.Address
		scoreValues[i] = score.Score
	}
	return &EventScoresSet{
		ActorType:   actorType,
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Addresses:   addresses,
		Scores:      scoreValues,
	}
}

// Assumes length of `rewards` is at least 1
func NewRewardsSetEventBase(actorType ActorType, blockHeight BlockHeight, rewards []TaskReward) proto.Message {
	topicId := rewards[0].TopicId
	addresses := make([]string, len(rewards))
	rewardValues := make([]alloraMath.Dec, len(rewards))
	for i, reward := range rewards {
		addresses[i] = reward.Address
		rewardValues[i] = reward.Reward
	}
	return &EventRewardsSettled{
		ActorType:   actorType,
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Addresses:   addresses,
		Rewards:     rewardValues,
	}
}

func NewNetworkLossSetEventBase(topicId TopicId, blockHeight BlockHeight, lossValueBundle ValueBundle) proto.Message {
	return &EventNetworkLossSet{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		ValueBundle: &lossValueBundle,
	}
}

func NewWorkerLastCommitSetEventBase(topicId TopicId, blockHeight BlockHeight, nonce *Nonce) proto.Message {
	return &EventWorkerLastCommitSet{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Nonce:       nonce,
	}
}

func NewReputerLastCommitSetEventBase(topicId TopicId, blockHeight BlockHeight, nonce *Nonce) proto.Message {
	return &EventReputerLastCommitSet{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Nonce:       nonce,
	}
}

func NewForecastTaskScoreSetEventBase(topicId TopicId, score alloraMath.Dec) proto.Message {
	return &EventForecastTaskScoreSet{
		TopicId: topicId,
		Score:   score,
	}
}

func NewTopicRewardSetEventBase(topicRewards map[uint64]*alloraMath.Dec) proto.Message {
	ids := alloraMath.GetSortedKeys(topicRewards)
	rewardValues := make([]alloraMath.Dec, 0)
	for _, id := range ids {
		rewardValues = append(rewardValues, *topicRewards[id])
	}
	return &EventTopicRewardsSet{
		TopicIds: ids,
		Rewards:  rewardValues,
	}
}

// Assumes length of `scores` is at least 1
func NewEMAScoresSetEventBase(actorType ActorType, scores []Score, activations map[string]bool) proto.Message {
	topicId := scores[0].TopicId
	blockHeight := scores[0].BlockHeight
	activeArr := make([]bool, len(scores))
	addresses := make([]string, len(scores))
	scoreValues := make([]alloraMath.Dec, len(scores))
	for i, score := range scores {
		addresses[i] = score.Address
		scoreValues[i] = score.Score
		activeArr[i] = activations[addresses[i]]
	}
	return &EventEMAScoresSet{
		ActorType: actorType,
		TopicId:   topicId,
		Nonce:     blockHeight,
		Addresses: addresses,
		Scores:    scoreValues,
		IsActive:  activeArr,
	}
}
