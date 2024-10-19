package types

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	proto "github.com/cosmos/gogoproto/proto"
)

/// Scores

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

func NewNetworkLossSetEventBase(topicId TopicId, blockHeight BlockHeight, lossValueBundle ValueBundle) proto.Message {
	return &EventNetworkLossSet{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		ValueBundle: &lossValueBundle,
	}
}

func NewForecastTaskScoreSetEventBase(topicId TopicId, score alloraMath.Dec) proto.Message {
	return &EventForecastTaskScoreSet{
		TopicId: topicId,
		Score:   score,
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

/// Rewards

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

/// Commits

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

/// Listening Coefficients

func NewListeningCoefficientsSetEventBase(topicID uint64, blockHeight int64, addresses []string, actorType ActorType, coefficients []alloraMath.Dec) proto.Message {
	return &EventListeningCoefficientsSet{
		ActorType:    actorType,
		TopicId:      topicID,
		BlockHeight:  blockHeight,
		Addresses:    addresses,
		Coefficients: coefficients,
	}
}

/// Regrets

func NewInfererNetworkRegretSetEventBase(topicID uint64, blockHeight int64, addresses []string, regrets []alloraMath.Dec) proto.Message {
	return &EventInfererNetworkRegretSet{
		TopicId:     topicID,
		BlockHeight: blockHeight,
		Addresses:   addresses,
		Regrets:     regrets,
	}
}

func NewForecasterNetworkRegretSetEventBase(topicID uint64, blockHeight int64, addresses []string, regrets []alloraMath.Dec) proto.Message {
	return &EventForecasterNetworkRegretSet{
		TopicId:     topicID,
		BlockHeight: blockHeight,
		Addresses:   addresses,
		Regrets:     regrets,
	}
}

func NewNaiveInfererNetworkRegretSetEventBase(topicID uint64, blockHeight int64, addresses []string, regrets []alloraMath.Dec) proto.Message {
	return &EventNaiveInfererNetworkRegretSet{
		TopicId:     topicID,
		BlockHeight: blockHeight,
		Addresses:   addresses,
		Regrets:     regrets,
	}
}

func NewTopicInitialRegretSetEventBase(topicID uint64, blockHeight int64, regret alloraMath.Dec) proto.Message {
	return &EventTopicInitialRegretSet{
		TopicId:     topicID,
		BlockHeight: blockHeight,
		Regret:      regret,
	}
}
