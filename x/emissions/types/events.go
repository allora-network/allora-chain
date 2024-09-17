package types

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

/// Emitters

func EmitNewInfererScoresSetEvent(ctx sdk.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	err := ctx.EventManager().EmitTypedEvent(NewScoresSetEventBase(ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, scores))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewInfererScoresSetEvent: ", err.Error())
	}
}

func EmitNewForecasterScoresSetEvent(ctx sdk.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	err := ctx.EventManager().EmitTypedEvent(NewScoresSetEventBase(ActorType_ACTOR_TYPE_FORECASTER, scores))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewForecasterScoresSetEvent: ", err.Error())
	}
}

func EmitNewReputerScoresSetEvent(ctx sdk.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	err := ctx.EventManager().EmitTypedEvent(NewScoresSetEventBase(ActorType_ACTOR_TYPE_REPUTER, scores))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewReputerScoresSetEvent: ", err.Error())
	}
}

func EmitNewNetworkLossSetEvent(ctx sdk.Context, topicId TopicId, blockHeight BlockHeight, lossBundle ValueBundle) {
	err := ctx.EventManager().EmitTypedEvent(NewNetworkLossSetEventBase(topicId, blockHeight, lossBundle))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewNetworkLossSetEvent: ", err.Error())
	}
}

func EmitNewInfererRewardsSettledEvent(ctx sdk.Context, blockHeight BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	err := ctx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_ACTOR_TYPE_INFERER_UNSPECIFIED, blockHeight, rewards))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewInfererRewardsSettledEvent: ", err.Error())
	}
}

func EmitNewForecasterRewardsSettledEvent(ctx sdk.Context, blockHeight BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	err := ctx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_ACTOR_TYPE_FORECASTER, blockHeight, rewards))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewForecasterRewardsSettledEvent: ", err.Error())
	}
}

func EmitNewReputerAndDelegatorRewardsSettledEvent(ctx sdk.Context, blockHeight BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	err := ctx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_ACTOR_TYPE_REPUTER, blockHeight, rewards))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewReputerAndDelegatorRewardsSettledEvent: ", err.Error())
	}
}

func EmitNewWorkerLastCommitSetEvent(ctx sdk.Context, topicId TopicId, height BlockHeight, nonce *Nonce) {
	err := ctx.EventManager().EmitTypedEvent(NewWorkerLastCommitSetEventBase(topicId, height, nonce))
	if err != nil {
		ctx.Logger().Warn("Error emitting NewWorkerLastCommitSetEvent: ", err.Error())
	}
}

func EmitNewReputerLastCommitSetEvent(ctx sdk.Context, topicId TopicId, height BlockHeight, nonce *Nonce) {
	err := ctx.EventManager().EmitTypedEvent(NewReputerLastCommitSetEventBase(topicId, height, nonce))
	if err != nil {
		ctx.Logger().Warn("Error emitting EmitNewReputerLastCommitSetEvent: ", err.Error())
	}
}

func EmitNewForecastTaskScoreSetEvent(ctx sdk.Context, topicId TopicId, score alloraMath.Dec) {
	err := ctx.EventManager().EmitTypedEvent(NewForecastTaskScoreSetEventBase(topicId, score))
	if err != nil {
		ctx.Logger().Warn("Error emitting EmitNewReputerLastCommitSetEvent: ", err.Error())
	}
}

func EmitNewTopicRewardSetEvent(ctx sdk.Context, topicRewards map[uint64]*alloraMath.Dec) {
	err := ctx.EventManager().EmitTypedEvent(NewTopicRewardSetEventBase(topicRewards))
	if err != nil {
		ctx.Logger().Warn("Error emitting EmitNewTopicRewardSetEvent: ", err.Error())
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
