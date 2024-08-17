package types

import (
	"fmt"

	alloraMath "github.com/allora-network/allora-chain/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
)

/// Emitters

func EmitNewInfererScoresSetEvent(ctx sdk.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	err := ctx.EventManager().EmitTypedEvent(NewScoresSetEventBase(ActorType_INFERER, scores))
	if err != nil {
		fmt.Println("Error emitting NewInfererScoresSetEvent: ", err)
	}
}

func EmitNewForecasterScoresSetEvent(ctx sdk.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	err := ctx.EventManager().EmitTypedEvent(NewScoresSetEventBase(ActorType_FORECASTER, scores))
	if err != nil {
		fmt.Println("Error emitting NewForecasterScoresSetEvent: ", err)
	}
}

func EmitNewReputerScoresSetEvent(ctx sdk.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	err := ctx.EventManager().EmitTypedEvent(NewScoresSetEventBase(ActorType_REPUTER, scores))
	if err != nil {
		fmt.Println("Error emitting NewReputerScoresSetEvent: ", err)
	}
}

func EmitNewNetworkLossSetEvent(ctx sdk.Context, topicId TopicId, blockHeight BlockHeight, lossBundle ValueBundle) {
	err := ctx.EventManager().EmitTypedEvent(NewNetworkLossSetEventBase(topicId, blockHeight, lossBundle))
	if err != nil {
		fmt.Println("Error emitting NewNetworkLossSetEvent: ", err)
	}
}

func EmitNewInfererRewardsSettledEvent(ctx sdk.Context, blockHeight BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	err := ctx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_INFERER, blockHeight, rewards))
	if err != nil {
		fmt.Println("Error emitting NewInfererRewardsSettledEvent: ", err)
	}
}

func EmitNewForecasterRewardsSettledEvent(ctx sdk.Context, blockHeight BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	err := ctx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_FORECASTER, blockHeight, rewards))
	if err != nil {
		fmt.Println("Error emitting NewForecasterRewardsSettledEvent: ", err)
	}
}

func EmitNewReputerAndDelegatorRewardsSettledEvent(ctx sdk.Context, blockHeight BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	err := ctx.EventManager().EmitTypedEvent(NewRewardsSetEventBase(ActorType_REPUTER, blockHeight, rewards))
	if err != nil {
		fmt.Println("Error emitting NewReputerAndDelegatorRewardsSettledEvent: ", err)
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
