package types

import (
	"cosmossdk.io/math"
	"github.com/cosmos/gogoproto/proto"

	alloraMath "github.com/allora-network/allora-chain/math"
)

// Scores

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

func NewNetworkInferencesEventBase(topicId TopicId, blockHeight BlockHeight, networkInferences ValueBundle) proto.Message {
	return &EventNetworkInferences{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		ValueBundle: &networkInferences,
	}
}

func NewInsertWorkerPayloadEventBase(topicId TopicId, bundle *WorkerDataBundle) proto.Message {
	return &EventInsertWorkerPayload{
		TopicId:          topicId,
		WorkerDataBundle: bundle,
	}
}

func NewCreateNewTopicEventBase(topic *Topic) proto.Message {
	return &EventCreateNewTopic{
		Topic: topic,
	}
}

func NewAddTopicFeeRevenueEventBase(topicId TopicId, amount, topicFeeRevenue math.Int) proto.Message {
	return &EventAddTopicFeeRevenue{
		TopicId:         topicId,
		Amount:          amount,
		TopicFeeRevenue: topicFeeRevenue,
	}
}

func NewAddReputerStakeEventBase(topicId TopicId, reputer string, amount, topicStake math.Int) proto.Message {
	return &EventAddReputerStake{
		TopicId:    topicId,
		Reputer:    reputer,
		Amount:     amount,
		TopicStake: topicStake,
	}
}

func NewRemoveReputerStakeEventBase(removal StakeRemovalInfo) proto.Message {
	return &EventRemoveReputerStake{
		TopicId:               removal.TopicId,
		Reputer:               removal.Reputer,
		Amount:                removal.Amount,
		BlockRemovalStarted:   removal.BlockRemovalStarted,
		BlockRemovalCompleted: removal.BlockRemovalCompleted,
	}
}

func NewCancelRemoveReputerStakeEventBase(removal StakeRemovalInfo) proto.Message {
	return &EventCancelRemoveReputerStake{
		TopicId:               removal.TopicId,
		Reputer:               removal.Reputer,
		Amount:                removal.Amount,
		BlockRemovalStarted:   removal.BlockRemovalStarted,
		BlockRemovalCompleted: removal.BlockRemovalCompleted,
	}
}

func NewAddDelegateStakeEventBase(topicId TopicId, reputer, delegator string, amount, topicStake math.Int) proto.Message {
	return &EventAddDelegateStake{
		TopicId:    topicId,
		Reputer:    reputer,
		Delegator:  delegator,
		Amount:     amount,
		TopicStake: topicStake,
	}
}

func NewRemoveDelegateStakeEventBase(removal DelegateStakeRemovalInfo) proto.Message {
	return &EventRemoveDelegateStake{
		TopicId:               removal.TopicId,
		Reputer:               removal.Reputer,
		Delegator:             removal.Delegator,
		Amount:                removal.Amount,
		BlockRemovalStarted:   removal.BlockRemovalStarted,
		BlockRemovalCompleted: removal.BlockRemovalCompleted,
	}
}

func NewCancelRemoveDelegateStakeEventBase(removal DelegateStakeRemovalInfo) proto.Message {
	return &EventCancelRemoveDelegateStake{
		TopicId:               removal.TopicId,
		Reputer:               removal.Reputer,
		Delegator:             removal.Delegator,
		Amount:                removal.Amount,
		BlockRemovalStarted:   removal.BlockRemovalStarted,
		BlockRemovalCompleted: removal.BlockRemovalCompleted,
	}
}

func NewRewardDelegateStakeEventBase(topicId TopicId, reputer, delegator string, amount alloraMath.Dec) proto.Message {
	return &EventRewardDelegateStake{
		TopicId:   topicId,
		Reputer:   reputer,
		Delegator: delegator,
		Amount:    amount,
	}
}

func NewInsertReputerPayloadEventBase(topicId TopicId, bundle *ReputerValueBundle) proto.Message {
	return &EventInsertReputerPayload{
		TopicId:            topicId,
		ReputerValueBundle: bundle,
	}
}

func NewReputerRegisteredEventBase(topicId TopicId, reputer, owner string) proto.Message {
	return &EventReputerRegistered{
		TopicId: topicId,
		Reputer: reputer,
		Owner:   owner,
	}
}

func NewWorkerRegisteredEventBase(topicId TopicId, worker, owner string) proto.Message {
	return &EventWorkerRegistered{
		TopicId: topicId,
		Worker:  worker,
		Owner:   owner,
	}
}

func NewReputerUnregisteredEventBase(topicId TopicId, reputer string) proto.Message {
	return &EventReputerUnregistered{
		TopicId: topicId,
		Reputer: reputer,
	}
}

func NewWorkerUnregisteredEventBase(topicId TopicId, worker string) proto.Message {
	return &EventWorkerUnregistered{
		TopicId: topicId,
		Worker:  worker,
	}
}

func NewFundTopicEventBase(topicId TopicId, funder string, amount math.Int) proto.Message {
	return &EventFundTopic{
		TopicId: topicId,
		Funder:  funder,
		Amount:  amount,
	}
}

func NewParamsSetEventBase(params Params) proto.Message {
	return &EventParamsSet{
		Params: &params,
	}
}

func NewWhitelistAdminAddedEventBase(admin string) proto.Message {
	return &EventWhitelistAdminAdded{
		Admin: admin,
	}
}

func NewWhitelistAdminRemovedEventBase(admin string) proto.Message {
	return &EventWhitelistAdminRemoved{
		Admin: admin,
	}
}

func NewGlobalWhitelistAddedEventBase(address string) proto.Message {
	return &EventGlobalWhitelistAdded{
		Address: address,
	}
}

func NewGlobalWhitelistRemovedEventBase(address string) proto.Message {
	return &EventGlobalWhitelistRemoved{
		Address: address,
	}
}

func NewGlobalWorkerWhitelistAddedEventBase(worker string) proto.Message {
	return &EventGlobalWorkerWhitelistAdded{
		Address: worker,
	}
}

func NewGlobalWorkerWhitelistRemovedEventBase(worker string) proto.Message {
	return &EventGlobalWorkerWhitelistRemoved{
		Address: worker,
	}
}

func NewGlobalReputerWhitelistAddedEventBase(reputer string) proto.Message {
	return &EventGlobalReputerWhitelistAdded{
		Address: reputer,
	}
}

func NewGlobalReputerWhitelistRemovedEventBase(reputer string) proto.Message {
	return &EventGlobalReputerWhitelistRemoved{
		Address: reputer,
	}
}

func NewGlobalAdminWhitelistAddedEventBase(admin string) proto.Message {
	return &EventGlobalAdminWhitelistAdded{
		Address: admin,
	}
}

func NewGlobalAdminWhitelistRemovedEventBase(admin string) proto.Message {
	return &EventGlobalAdminWhitelistRemoved{
		Address: admin,
	}
}

func NewGlobalWorkerWhitelistBulkAddedEventBase(addresses []string) proto.Message {
	return &EventGlobalWorkerWhitelistBulkAdded{
		Addresses: addresses,
	}
}

func NewGlobalWorkerWhitelistBulkRemovedEventBase(addresses []string) proto.Message {
	return &EventGlobalWorkerWhitelistBulkRemoved{
		Addresses: addresses,
	}
}

func NewGlobalReputerWhitelistBulkAddedEventBase(addresses []string) proto.Message {
	return &EventGlobalReputerWhitelistBulkAdded{
		Addresses: addresses,
	}
}

func NewGlobalReputerWhitelistBulkRemovedEventBase(addresses []string) proto.Message {
	return &EventGlobalReputerWhitelistBulkRemoved{
		Addresses: addresses,
	}
}

func NewTopicWorkerWhitelistBulkAddedEventBase(topicId TopicId, addresses []string) proto.Message {
	return &EventTopicWorkerWhitelistBulkAdded{
		TopicId:   topicId,
		Addresses: addresses,
	}
}

func NewTopicWorkerWhitelistBulkRemovedEventBase(topicId TopicId, addresses []string) proto.Message {
	return &EventTopicWorkerWhitelistBulkRemoved{
		TopicId:   topicId,
		Addresses: addresses,
	}
}

func NewTopicReputerWhitelistBulkAddedEventBase(topicId TopicId, addresses []string) proto.Message {
	return &EventTopicReputerWhitelistBulkAdded{
		TopicId:   topicId,
		Addresses: addresses,
	}
}

func NewTopicReputerWhitelistBulkRemovedEventBase(topicId TopicId, addresses []string) proto.Message {
	return &EventTopicReputerWhitelistBulkRemoved{
		TopicId:   topicId,
		Addresses: addresses,
	}
}

func NewTopicWorkerWhitelistEnabledEventBase(topicId TopicId) proto.Message {
	return &EventTopicWorkerWhitelistEnabled{
		TopicId: topicId,
	}
}

func NewTopicWorkerWhitelistDisabledEventBase(topicId TopicId) proto.Message {
	return &EventTopicWorkerWhitelistDisabled{
		TopicId: topicId,
	}
}

func NewTopicReputerWhitelistEnabledEventBase(topicId TopicId) proto.Message {
	return &EventTopicReputerWhitelistEnabled{
		TopicId: topicId,
	}
}

func NewTopicReputerWhitelistDisabledEventBase(topicId TopicId) proto.Message {
	return &EventTopicReputerWhitelistDisabled{
		TopicId: topicId,
	}
}

func NewTopicCreatorWhitelistAddedEventBase(address string) proto.Message {
	return &EventTopicCreatorWhitelistAdded{
		Address: address,
	}
}

func NewTopicCreatorWhitelistRemovedEventBase(address string) proto.Message {
	return &EventTopicCreatorWhitelistRemoved{
		Address: address,
	}
}

func NewTopicWorkerWhitelistAddedEventBase(topicId TopicId, address string) proto.Message {
	return &EventTopicWorkerWhitelistAdded{
		TopicId: topicId,
		Address: address,
	}
}

func NewTopicWorkerWhitelistRemovedEventBase(topicId TopicId, address string) proto.Message {
	return &EventTopicWorkerWhitelistRemoved{
		TopicId: topicId,
		Address: address,
	}
}

func NewTopicReputerWhitelistAddedEventBase(topicId TopicId, address string) proto.Message {
	return &EventTopicReputerWhitelistAdded{
		TopicId: topicId,
		Address: address,
	}
}

func NewTopicReputerWhitelistRemovedEventBase(topicId TopicId, address string) proto.Message {
	return &EventTopicReputerWhitelistRemoved{
		TopicId: topicId,
		Address: address,
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

// Rewards

// Assumes length of `rewards` is at least 1
func NewRewardsSetEventBase(actorType ActorType, blockHeight, blockHeightTx BlockHeight, rewards []TaskReward) proto.Message {
	topicId := rewards[0].TopicId
	addresses := make([]string, len(rewards))
	rewardValues := make([]alloraMath.Dec, len(rewards))
	for i, reward := range rewards {
		addresses[i] = reward.Address
		rewardValues[i] = reward.Reward
	}
	return &EventRewardsSettled{
		ActorType:     actorType,
		TopicId:       topicId,
		BlockHeight:   blockHeight,
		Addresses:     addresses,
		Rewards:       rewardValues,
		BlockHeightTx: blockHeightTx,
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

// Commits

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

// Listening Coefficients

func NewListeningCoefficientsSetEventBase(topicID uint64, blockHeight int64, addresses []string, actorType ActorType, coefficients []alloraMath.Dec) proto.Message {
	return &EventListeningCoefficientsSet{
		ActorType:    actorType,
		TopicId:      topicID,
		BlockHeight:  blockHeight,
		Addresses:    addresses,
		Coefficients: coefficients,
	}
}

// Regrets

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

func NewTopicInitialEmaScoreSetEventBase(actorType ActorType, topicId uint64, blockHeight int64, score alloraMath.Dec) proto.Message {
	return &EventTopicInitialEmaScoreSet{
		ActorType:   actorType,
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Score:       score,
	}
}

func NewRegretStdNormSetEventBase(topicId uint64, blockHeight int64, stdNorm alloraMath.Dec) proto.Message {
	return &EventRegretStdNormSet{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Stdnorm:     stdNorm,
	}
}

func NewInfererWeightSetEventBase(topicId uint64, blockHeight int64, address string, weight alloraMath.Dec) proto.Message {
	return &EventInfererWeightSet{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Address:     address,
		Weight:      weight,
	}
}

func NewForecasterWeightSetEventBase(topicId uint64, blockHeight int64, address string, weight alloraMath.Dec) proto.Message {
	return &EventForecasterWeightSet{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Address:     address,
		Weight:      weight,
	}
}

// Previous Percentage Reward

func NewPreviousPercentageRewardToStakedReputersSetEventBase(blockHeight int64, percentage alloraMath.Dec) proto.Message {
	return &EventPreviousPercentageRewardToStakedReputersSet{
		BlockHeight: blockHeight,
		Percentage:  percentage,
	}
}

// Pruning
func NewPruneRecordsSetEventBase(blockHeight int64, topicId TopicId) proto.Message {
	return &EventPruneRecords{
		BlockHeight: blockHeight,
		TopicId:     topicId,
	}
}

// Stake removal
func NewReputerStakeRemovalCompletedEventBase(topicId TopicId, blockHeight BlockHeight, reputer string, amount math.Int) proto.Message {
	return &EventReputerStakeRemovalCompleted{
		TopicId:     topicId,
		Reputer:     reputer,
		Amount:      amount,
		BlockHeight: blockHeight,
	}
}

func NewDelegateStakeRemovalCompletedEventBase(topicId TopicId, blockHeight BlockHeight, delegator string, reputer string, amount math.Int) proto.Message {
	return &EventDelegateStakeRemovalCompleted{
		TopicId:     topicId,
		Reputer:     reputer,
		Delegator:   delegator,
		Amount:      amount,
		BlockHeight: blockHeight,
	}
}

func NewDelegateRewardShareUpdatedEventBase(topicId TopicId, reputer string, rewardPerShare alloraMath.Dec, blockHeight int64) proto.Message {
	return &EventDelegateRewardShareUpdated{
		TopicId:        topicId,
		Reputer:        reputer,
		RewardPerShare: rewardPerShare,
		BlockHeight:    blockHeight,
	}
}

func NewDelegateRewardDistributedEventBase(topicId TopicId, reputer string, amount math.Int, blockHeight int64) proto.Message {
	return &EventDelegateRewardDistributed{
		TopicId:     topicId,
		Reputer:     reputer,
		Amount:      amount,
		BlockHeight: blockHeight,
	}
}

func NewActiveReputersSetEventBase(topicId TopicId, nonceBlockHeight int64, addresses []string, blockHeight int64) proto.Message {
	return &EventActiveReputersSet{
		TopicId:          topicId,
		Addresses:        addresses,
		NonceBlockHeight: nonceBlockHeight,
		BlockHeight:      blockHeight,
	}
}

func NewActiveInferersSetEventBase(topicId TopicId, nonceBlockHeight int64, addresses []string, blockHeight int64) proto.Message {
	return &EventActiveInferersSet{
		TopicId:          topicId,
		Addresses:        addresses,
		NonceBlockHeight: nonceBlockHeight,
		BlockHeight:      blockHeight,
	}
}

func NewActiveForecastersSetEventBase(topicId TopicId, nonceBlockHeight int64, addresses []string, blockHeight int64) proto.Message {
	return &EventActiveForecastersSet{
		TopicId:          topicId,
		Addresses:        addresses,
		NonceBlockHeight: nonceBlockHeight,
		BlockHeight:      blockHeight,
	}
}

func NewActiveTopicsAtBlockSetEventBase(targetBlockHeight int64, topicIds []uint64, blockHeight int64) proto.Message {
	return &EventActiveTopicsAtBlockSet{
		TargetBlockHeight: targetBlockHeight,
		TopicIds:          topicIds,
		BlockHeight:       blockHeight,
	}
}
