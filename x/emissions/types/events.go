package types

import (
	"encoding/json"
	fmt "fmt"

	alloraMath "github.com/allora-network/allora-chain/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	EventTypeInfererScoresSet                  = "inferer_scores_set"
	EventTypeForecasterScoresSet               = "forecaster_scores_set"
	EventTypeReputerScoresSet                  = "reputer_scores_set"
	EventTypeInfererRewardsSettled             = "inferer_rewards_settled"
	EventTypeForecasterRewardsSettled          = "forecaster_rewards_settled"
	EventTypeReputerAndDelegatorRewardsSettled = "reputer_and_delegator_rewards_settled"

	AttributeKeyTopicId     = "topic_id"
	AttributeKeyBlockHeight = "block_height"
	AttributeKeyAddresses   = "addresses"
	AttributeKeyScores      = "scores"
	AttributeKeyRewards     = "rewards"
)

/// Emitters

func EmitNewInfererScoresSetEvent(ctx sdk.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	ctx.EventManager().EmitEvent(NewScoresSetEventBase(EventTypeInfererScoresSet, scores))
}

func EmitNewForecasterScoresSetEvent(ctx sdk.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	ctx.EventManager().EmitEvent(NewScoresSetEventBase(EventTypeForecasterScoresSet, scores))
}

func EmitNewReputerScoresSetEvent(ctx sdk.Context, scores []Score) {
	if len(scores) < 1 {
		return
	}
	ctx.EventManager().EmitEvent(NewScoresSetEventBase(EventTypeReputerScoresSet, scores))
}

func EmitNewInfererRewardsSettledEvent(ctx sdk.Context, blockHeight BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	ctx.EventManager().EmitEvent(NewRewardsSetEventBase(EventTypeInfererRewardsSettled, blockHeight, rewards))
}

func EmitNewForecasterRewardsSettledEvent(ctx sdk.Context, blockHeight BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	ctx.EventManager().EmitEvent(NewRewardsSetEventBase(EventTypeForecasterRewardsSettled, blockHeight, rewards))
}

func EmitNewReputererAndDelegaterRewardsSettledEvent(ctx sdk.Context, blockHeight BlockHeight, rewards []TaskReward) {
	if len(rewards) < 1 {
		return
	}
	ctx.EventManager().EmitEvent(NewRewardsSetEventBase(EventTypeReputerAndDelegatorRewardsSettled, blockHeight, rewards))
}

/// Utils

// Assumes length of `scores` is at least 1
func NewScoresSetEventBase(eventType string, scores []Score) sdk.Event {
	topicId := scores[0].TopicId
	blockHeight := scores[0].BlockHeight
	addresses := make([]string, len(scores))
	scoreValues := make([]alloraMath.Dec, len(scores))
	for i, score := range scores {
		addresses[i] = score.Address
		scoreValues[i] = score.Score
	}
	return sdk.NewEvent(
		eventType,
		sdk.NewAttribute(AttributeKeyTopicId, FormatNumberForEventAttribute(topicId)),
		sdk.NewAttribute(AttributeKeyBlockHeight, FormatNumberForEventAttribute(blockHeight)),
		sdk.NewAttribute(AttributeKeyAddresses, FormatStringsForEventAttribute(addresses)),
		sdk.NewAttribute(AttributeKeyScores, FormatDecsForEventAttribute(scoreValues)),
	)
}

// Assumes length of `rewards` is at least 1
func NewRewardsSetEventBase(eventType string, blockHeight BlockHeight, rewards []TaskReward) sdk.Event {
	topicId := rewards[0].TopicId
	addresses := make([]string, len(rewards))
	rewardValues := make([]alloraMath.Dec, len(rewards))
	for i, reward := range rewards {
		addresses[i] = reward.Address
		rewardValues[i] = reward.Reward
	}
	return sdk.NewEvent(
		eventType,
		sdk.NewAttribute(AttributeKeyTopicId, FormatNumberForEventAttribute(topicId)),
		sdk.NewAttribute(AttributeKeyBlockHeight, FormatNumberForEventAttribute(blockHeight)),
		sdk.NewAttribute(AttributeKeyAddresses, FormatStringsForEventAttribute(addresses)),
		sdk.NewAttribute(AttributeKeyRewards, FormatDecsForEventAttribute(rewardValues)),
	)
}

func FormatNumberForEventAttribute[T uint64 | int64](n T) string {
	return fmt.Sprintf("%d", n)
}

func FormatStringsForEventAttribute(s []string) string {
	jsonBytes, err := json.Marshal(s)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

func FormatDecsForEventAttribute(d []alloraMath.Dec) string {
	decsAsStrings := make([]string, len(d))
	for i, dec := range d {
		decsAsStrings[i] = dec.String()
	}
	jsonBytes, err := json.Marshal(decsAsStrings)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}
