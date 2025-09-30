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

func NewNetworkLossSetEventBase(lossBundle ValueBundle) proto.Message {
	return &EventNetworkLossSet{
		TopicId:     lossBundle.TopicId,
		BlockHeight: lossBundle.ReputerRequestNonce.ReputerNonce.BlockHeight,
		ValueBundle: nil,
		Nonce:       lossBundle.ReputerRequestNonce.ReputerNonce.BlockHeight,
		Bundle:      valueBundleToEventValueBundleBase(&lossBundle),
	}
}

func NewNetworkInferencesEventBase(networkInferences ValueBundle) proto.Message {
	return &EventNetworkInferences{
		TopicId:     networkInferences.TopicId,
		BlockHeight: networkInferences.ReputerRequestNonce.ReputerNonce.BlockHeight,
		ValueBundle: nil,
		Nonce:       networkInferences.ReputerRequestNonce.ReputerNonce.BlockHeight,
		Bundle:      valueBundleToEventValueBundleBase(&networkInferences),
	}
}

func NewOutlierResistantNetworkInferencesEventBase(networkInferences ValueBundle) proto.Message {
	return &EventOutlierResistantNetworkInferences{
		TopicId: networkInferences.TopicId,
		Nonce:   networkInferences.ReputerRequestNonce.ReputerNonce.BlockHeight,
		Bundle:  valueBundleToEventValueBundleBase(&networkInferences),
	}
}

func NewInsertInfererPayloadEventBase(bundle *WorkerDataBundle) proto.Message {
	return &EventInsertInfererPayload{
		Inferer:   bundle.Worker,
		Nonce:     bundle.Nonce.BlockHeight,
		TopicId:   bundle.TopicId,
		Value:     bundle.InferenceForecastsBundle.Inference.Value,
		ExtraData: bundle.InferenceForecastsBundle.Inference.ExtraData,
	}
}

func NewInsertForecasterPayloadEventBase(bundle *WorkerDataBundle) proto.Message {
	infererAddresses := make([]string, 0, len(bundle.InferenceForecastsBundle.Forecast.ForecastElements))
	infererValues := make([]alloraMath.Dec, 0, len(bundle.InferenceForecastsBundle.Forecast.ForecastElements))
	for _, infVal := range bundle.InferenceForecastsBundle.Forecast.ForecastElements {
		infererAddresses = append(infererAddresses, infVal.Inferer)
		infererValues = append(infererValues, infVal.Value)
	}
	return &EventInsertForecasterPayload{
		Forecaster:       bundle.Worker,
		Nonce:            bundle.Nonce.BlockHeight,
		TopicId:          bundle.TopicId,
		InfererAddresses: infererAddresses,
		InfererValues:    infererValues,
		ExtraData:        bundle.InferenceForecastsBundle.Forecast.ExtraData,
	}
}

func NewCreateNewTopicEventBase(topic *Topic) proto.Message {
	return &EventCreateNewTopic{
		Topic: topic,
	}
}

func NewAddStakeEventBase(topicId TopicId, reputer, delegator string, amount math.Int) proto.Message {
	return &EventAddStake{
		TopicId:   topicId,
		Reputer:   reputer,
		Delegator: delegator,
		Amount:    amount,
	}
}

func NewRemoveStakeEventBase(topicId TopicId, reputer string, delegator string, amount math.Int) proto.Message {
	return &EventRemoveStake{
		TopicId:   topicId,
		Reputer:   reputer,
		Delegator: delegator,
		Amount:    amount,
	}
}

func NewRequestStakeRemovalEventBase(topicId TopicId, reputer string, delegator string, amount math.Int, completionHeight int64) proto.Message {
	return &EventRequestStakeRemoval{
		TopicId:          topicId,
		Reputer:          reputer,
		Delegator:        delegator,
		Amount:           amount,
		CompletionHeight: completionHeight,
	}
}

func NewCancelStakeRemovalEventBase(topicId TopicId, reputer string, delegator string) proto.Message {
	return &EventCancelStakeRemoval{
		TopicId:   topicId,
		Reputer:   reputer,
		Delegator: delegator,
	}
}

func NewReputerStakeUpdatedEventBase(topicId TopicId, reputer string, amount math.Int) proto.Message {
	return &EventReputerStakeUpdated{
		TopicId: topicId,
		Reputer: reputer,
		Amount:  amount,
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

func NewInsertReputerPayloadEventBase(bundle *ReputerValueBundle) proto.Message {
	return &EventInsertReputerPayload{
		TopicId: bundle.ValueBundle.TopicId,
		Nonce:   bundle.ValueBundle.ReputerRequestNonce.ReputerNonce.BlockHeight,
		Reputer: bundle.ValueBundle.Reputer,
		Bundle:  valueBundleToEventValueBundleBase(bundle.ValueBundle),
	}
}

func valueBundleToEventValueBundleBase(bundle *ValueBundle) *EventValueBundle {
	infererAddresses := make([]string, 0, len(bundle.InfererValues))
	infererValues := make([]alloraMath.Dec, 0, len(bundle.InfererValues))
	forecasterAddresses := make([]string, 0, len(bundle.ForecasterValues))
	forecasterValues := make([]alloraMath.Dec, 0, len(bundle.ForecasterValues))
	oneOutInfererValues := make([]alloraMath.Dec, 0, len(bundle.OneOutInfererValues))
	oneOutForecasterValues := make([]alloraMath.Dec, 0, len(bundle.OneOutForecasterValues))
	oneInForecasterValues := make([]alloraMath.Dec, 0, len(bundle.OneInForecasterValues))
	oneOutInfererForecasterValues := make([]alloraMath.DecArray, 0, len(bundle.OneOutInfererForecasterValues))

	for _, infVal := range bundle.InfererValues {
		infererAddresses = append(infererAddresses, infVal.Worker)
		infererValues = append(infererValues, infVal.Value)
	}
	for _, forcVal := range bundle.ForecasterValues {
		forecasterAddresses = append(forecasterAddresses, forcVal.Worker)
		forecasterValues = append(forecasterValues, forcVal.Value)
	}
	for _, ooIval := range bundle.OneOutInfererValues {
		oneOutInfererValues = append(oneOutInfererValues, ooIval.Value)
	}
	for _, ooFVal := range bundle.OneOutForecasterValues {
		oneOutForecasterValues = append(oneOutForecasterValues, ooFVal.Value)
	}
	for _, oiFVal := range bundle.OneInForecasterValues {
		oneInForecasterValues = append(oneInForecasterValues, oiFVal.Value)
	}
	for _, ooifVal := range bundle.OneOutInfererForecasterValues {
		ooInfererValues := make([]alloraMath.Dec, 0, len(ooifVal.OneOutInfererValues))
		for _, ooiVal := range ooifVal.OneOutInfererValues {
			ooInfererValues = append(ooInfererValues, ooiVal.Value)
		}
		oneOutInfererForecasterValues = append(oneOutInfererForecasterValues, ooInfererValues)
	}
	return &EventValueBundle{
		ExtraData:                     bundle.ExtraData,
		CombinedValue:                 bundle.CombinedValue,
		NaiveValue:                    bundle.NaiveValue,
		InfererAddresses:              infererAddresses,
		ForecasterAddresses:           forecasterAddresses,
		InfererValues:                 infererValues,
		ForecasterValues:              forecasterValues,
		OneOutInfererValues:           oneOutInfererValues,
		OneOutForecasterValues:        oneOutForecasterValues,
		OneInForecasterValues:         oneInForecasterValues,
		OneOutInfererForecasterValues: oneOutInfererForecasterValues,
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

func NewForecastTaskScoreSetEventBase(topicId TopicId, score alloraMath.Dec, nonce int64) proto.Message {
	return &EventForecastTaskScoreSet{
		TopicId:          topicId,
		Score:            score,
		NonceBlockHeight: nonce,
	}
}

// Assumes length of `scores` is at least 1
func NewEMAScoresSetEventBase(actorType ActorType, nonceBlockHeight BlockHeight, scores []Score, activations map[string]bool) proto.Message {
	topicId := scores[0].TopicId
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
		Nonce:     nonceBlockHeight,
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

func NewInfererWeightsSetEventBase(topicId uint64, blockHeight int64, addresses []string, weights []alloraMath.Dec) proto.Message {
	return &EventInfererWeightsSet{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Addresses:   addresses,
		Weights:     weights,
	}
}

func NewForecasterWeightsSetEventBase(topicId uint64, blockHeight int64, addresses []string, weights []alloraMath.Dec) proto.Message {
	return &EventForecasterWeightsSet{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Addresses:   addresses,
		Weights:     weights,
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

func NewDelegateRewardShareUpdatedEventBase(topicId TopicId, reputer string, rewardPerShare alloraMath.Dec) proto.Message {
	return &EventDelegateRewardShareUpdated{
		TopicId:        topicId,
		Reputer:        reputer,
		RewardPerShare: rewardPerShare,
	}
}

func NewDelegateRewardDistributedEventBase(topicId TopicId, reputer string, amount math.Int) proto.Message {
	return &EventDelegateRewardDistributed{
		TopicId: topicId,
		Reputer: reputer,
		Amount:  amount,
	}
}

func NewActiveReputersSetEventBase(topicId TopicId, nonceBlockHeight int64, addresses []string) proto.Message {
	return &EventActiveReputersSet{
		TopicId:          topicId,
		Addresses:        addresses,
		NonceBlockHeight: nonceBlockHeight,
	}
}

func NewActiveInferersSetEventBase(topicId TopicId, nonceBlockHeight int64, addresses []string) proto.Message {
	return &EventActiveInferersSet{
		TopicId:          topicId,
		Addresses:        addresses,
		NonceBlockHeight: nonceBlockHeight,
	}
}

func NewActiveForecastersSetEventBase(topicId TopicId, nonceBlockHeight int64, addresses []string) proto.Message {
	return &EventActiveForecastersSet{
		TopicId:          topicId,
		Addresses:        addresses,
		NonceBlockHeight: nonceBlockHeight,
	}
}

func NewTopicStatusChangedEventBase(topicId TopicId, isActive bool) proto.Message {
	return &EventTopicStatusChanged{
		TopicId:  topicId,
		IsActive: isActive,
	}
}

func NewNetworkInferenceInfererWeightsSetEventBase(topicId TopicId, blockHeight int64, addresses []string, weights []alloraMath.Dec) proto.Message {
	return &EventNetworkInferenceInfererWeightsSet{
		TopicId:          topicId,
		NonceBlockHeight: blockHeight,
		Addresses:        addresses,
		Weights:          weights,
	}
}

func NewNetworkInferenceForecasterWeightsSetEventBase(topicId TopicId, blockHeight int64, addresses []string, weights []alloraMath.Dec) proto.Message {
	return &EventNetworkInferenceForecasterWeightsSet{
		TopicId:          topicId,
		NonceBlockHeight: blockHeight,
		Addresses:        addresses,
		Weights:          weights,
	}
}

func NewNetworkInferenceInfererRegretsUsedSetEventBase(topicId TopicId, blockHeight int64, addresses []string, regrets []alloraMath.Dec) proto.Message {
	return &EventNetworkInferenceInfererRegretsUsedSet{
		TopicId:          topicId,
		NonceBlockHeight: blockHeight,
		Addresses:        addresses,
		Regrets:          regrets,
	}
}

func NewNetworkInferenceForecasterRegretsUsedSetEventBase(topicId TopicId, blockHeight int64, addresses []string, regrets []alloraMath.Dec) proto.Message {
	return &EventNetworkInferenceForecasterRegretsUsedSet{
		TopicId:          topicId,
		NonceBlockHeight: blockHeight,
		Addresses:        addresses,
		Regrets:          regrets,
	}
}

func NewTopicWeightUpdatedEventBase(topicId TopicId, newWeight alloraMath.Dec, topicStake math.Int, topicFeeRevenue math.Int) proto.Message {
	return &EventTopicWeightUpdated{
		TopicId:         topicId,
		NewWeight:       newWeight,
		TopicStake:      topicStake,
		TopicFeeRevenue: topicFeeRevenue,
	}
}

func NewTopicFeeRevenueDrippedEventBase(topicId TopicId, oldRevenue math.Int, newRevenue math.Int, dripAmount math.Int) proto.Message {
	return &EventTopicFeeRevenueDripped{
		TopicId:    topicId,
		OldRevenue: oldRevenue,
		NewRevenue: newRevenue,
		DripAmount: dripAmount,
	}
}

// Submission window events
func NewWorkerSubmissionWindowOpenedEventBase(topicId TopicId, nonceBlockHeight int64, windowEndBlock int64) proto.Message {
	return &EventWorkerSubmissionWindowOpened{
		TopicId:          topicId,
		NonceBlockHeight: nonceBlockHeight,
		WindowEndBlock:   windowEndBlock,
	}
}

func NewWorkerSubmissionWindowClosedEventBase(topicId TopicId, nonceBlockHeight int64) proto.Message {
	return &EventWorkerSubmissionWindowClosed{
		TopicId:          topicId,
		NonceBlockHeight: nonceBlockHeight,
	}
}

func NewReputerSubmissionWindowOpenedEventBase(topicId TopicId, nonceBlockHeight int64, windowEndBlock int64) proto.Message {
	return &EventReputerSubmissionWindowOpened{
		TopicId:          topicId,
		NonceBlockHeight: nonceBlockHeight,
		WindowEndBlock:   windowEndBlock,
	}
}

func NewReputerSubmissionWindowClosedEventBase(topicId TopicId, nonceBlockHeight int64) proto.Message {
	return &EventReputerSubmissionWindowClosed{
		TopicId:          topicId,
		NonceBlockHeight: nonceBlockHeight,
	}
}
