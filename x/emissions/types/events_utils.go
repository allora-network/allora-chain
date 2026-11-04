package types

import (
	"fmt"

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
		Scores:      clampDecs(scoreValues),
	}
}

func NewNetworkLossSetEventBase(lossBundle ValueBundle) proto.Message {
	return &EventNetworkLossSet{
		TopicId:     lossBundle.TopicId,
		BlockHeight: lossBundle.ReputerRequestNonce.ReputerNonce.BlockHeight,
		ValueBundle: nil,
		Nonce:       lossBundle.ReputerRequestNonce.ReputerNonce.BlockHeight,
		Bundle:      ValueBundleToEventValueBundleBase(&lossBundle),
	}
}

func NewNetworkInferencesEventBase(networkInferences NetworkInferenceBundle) proto.Message {
	return convertNetworkInferenceBundleToEvent(networkInferences, false)
}

func NewOutlierResistantNetworkInferencesEventBase(networkInferences NetworkInferenceBundle) proto.Message {
	return convertNetworkInferenceBundleToEvent(networkInferences, true)
}

func convertNetworkInferenceBundleToEvent(b NetworkInferenceBundle, outlierResistant bool) *EventNetworkInferenceBundle {
	// clamp during extraction: single pass, covers every bundle array/matrix
	labeledValuesToArray := func(vals []*LabeledValue) alloraMath.DecArray {
		return convertArray(vals, func(v *LabeledValue) alloraMath.Dec { return clampDec(v.Value) })
	}

	labelNames := convertArray(b.GetCombinedValue(), func(v *LabeledValue) string { return v.GetLabelName() })

	infererAddrs := convertArray(b.GetInfererValues(), func(w *WorkerInference) string { return w.GetWorker() })
	infererMatrix := convertArray(b.GetInfererValues(), func(w *WorkerInference) alloraMath.DecArray {
		return labeledValuesToArray(w.GetValues())
	})

	forecasterAddrs := convertArray(b.GetForecasterValues(), func(w *WorkerInference) string { return w.GetWorker() })
	forecasterMatrix := convertArray(b.GetForecasterValues(), func(w *WorkerInference) alloraMath.DecArray {
		return labeledValuesToArray(w.GetValues())
	})

	oneOutInferer := convertArray(b.GetOneOutInfererValues(), func(v *OneOutInfererValue) alloraMath.DecArray {
		return labeledValuesToArray(v.GetCombinedInference())
	})
	oneOutForecaster := convertArray(b.GetOneOutForecasterValues(), func(v *OneOutForecasterValue) alloraMath.DecArray {
		return labeledValuesToArray(v.GetCombinedInference())
	})
	oneInForecaster := convertArray(b.GetOneInForecasterValues(), func(v *OneInForecasterValue) alloraMath.DecArray {
		return labeledValuesToArray(v.GetCombinedInference())
	})

	ooif := groupOneOutInfererForecaster(b.GetOneOutInfererForecasterValues(), labeledValuesToArray)

	return &EventNetworkInferenceBundle{
		TopicId:                       b.GetTopicId(),
		Nonce:                         b.GetNonce(),
		LabelNames:                    labelNames,
		InfererAddresses:              infererAddrs,
		ForecasterAddresses:           forecasterAddrs,
		CombinedValue:                 labeledValuesToArray(b.GetCombinedValue()),
		NaiveValue:                    labeledValuesToArray(b.GetNaiveValue()),
		InfererValues:                 infererMatrix,
		ForecasterValues:              forecasterMatrix,
		OneOutInfererValues:           oneOutInferer,
		OneOutForecasterValues:        oneOutForecaster,
		OneInForecasterValues:         oneInForecaster,
		OneOutInfererForecasterValues: ooif,
		OutlierResistant:              outlierResistant,
	}
}

func groupOneOutInfererForecaster(
	in []*OneOutInfererForecasterValue,
	labeledValuesToArray func([]*LabeledValue) alloraMath.DecArray,
) []alloraMath.DecMatrix {
	if len(in) == 0 {
		return nil
	}

	order := make([]string, 0)
	rowsByForecaster := make(map[string][]alloraMath.DecArray)

	for _, x := range in {
		if x == nil {
			continue
		}
		fc := x.GetForecaster()
		if _, ok := rowsByForecaster[fc]; !ok {
			order = append(order, fc)
		}
		rowsByForecaster[fc] = append(rowsByForecaster[fc], labeledValuesToArray(x.GetCombinedInference()))
	}

	out := make([]alloraMath.DecMatrix, 0, len(order))
	for _, fc := range order {
		rows := rowsByForecaster[fc]
		m := make(alloraMath.DecMatrix, len(rows))
		copy(m, rows)
		out = append(out, m)
	}
	return out
}

func NewInsertInfererPayloadEventBase(bundle *InputWorkerDataBundle) proto.Message {
	return &EventInsertInfererPayload{
		Inferer:   bundle.Worker,
		Nonce:     bundle.Nonce.BlockHeight,
		TopicId:   bundle.TopicId,
		Value:     bundle.InferenceForecastsBundle.Inference.Value,
		Values:    bundle.InferenceForecastsBundle.Inference.Values,
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

func NewTopicUpdatedEventBase(topic *Topic) proto.Message {
	return &EventTopicUpdated{
		TopicId: topic.Id,
		Topic:   topic,
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
		Amount:    clampDec(amount),
	}
}

func NewInsertReputerPayloadEventBase(bundle *LossBundle) proto.Message {
	return &EventInsertReputerPayload{
		TopicId: bundle.TopicId,
		Nonce:   bundle.ReputerRequestNonce.ReputerNonce.BlockHeight,
		Reputer: bundle.Reputer,
		Bundle:  ValueBundleToEventValueBundleBase(bundle),
	}
}

func ValueBundleToEventValueBundleBase(bundle *ValueBundle) *EventValueBundle {
	//nolint:exhaustruct
	evb := &EventValueBundle{
		ExtraData:     bundle.ExtraData,
		CombinedValue: clampDec(bundle.CombinedValue),
		NaiveValue:    clampDec(bundle.NaiveValue),
	}

	// value fields clamp during extraction; address fields are untouched
	evb.InfererValues = convertArray(bundle.InfererValues, func(i *WorkerAttributedValue) alloraMath.Dec { return clampDec(i.GetValue()) })
	evb.InfererAddresses = convertArray(bundle.InfererValues, func(i *WorkerAttributedValue) string { return i.GetWorker() })
	evb.ForecasterValues = convertArray(bundle.ForecasterValues, func(i *WorkerAttributedValue) alloraMath.Dec { return clampDec(i.GetValue()) })
	evb.ForecasterAddresses = convertArray(bundle.ForecasterValues, func(i *WorkerAttributedValue) string { return i.GetWorker() })
	evb.OneOutInfererValues = convertArray(bundle.OneOutInfererValues, func(i *WithheldWorkerAttributedValue) alloraMath.Dec { return clampDec(i.GetValue()) })
	evb.OneInForecasterValues = convertArray(bundle.OneInForecasterValues, func(i *WorkerAttributedValue) alloraMath.Dec { return clampDec(i.GetValue()) })
	evb.OneOutForecasterValues = convertArray(bundle.OneOutForecasterValues, func(i *WithheldWorkerAttributedValue) alloraMath.Dec { return clampDec(i.GetValue()) })

	lenInf, looif := len(evb.InfererAddresses), len(bundle.OneOutInfererForecasterValues)
	if lenInf == 0 || looif == 0 {
		return evb
	}

	evb.OneOutInfererForecasterValues = make([]alloraMath.DecArray, looif)

	// ensure the number of inferers for each forecaster is the same as the full set of inferers,
	// where any gaps in values would be filled with NaNs.
	infererIndex := make(map[string]int, lenInf)
	for i, addr := range evb.InfererAddresses {
		infererIndex[addr] = i
	}

	for idx, row := range bundle.OneOutInfererForecasterValues {
		oo := make([]alloraMath.Dec, lenInf)
		for i := range lenInf {
			oo[i] = alloraMath.NewNaN()
		}
		for _, cell := range row.OneOutInfererValues {
			if j, ok := infererIndex[cell.Worker]; ok {
				oo[j] = clampDec(cell.Value)
			}
		}
		evb.OneOutInfererForecasterValues[idx] = oo
	}
	return evb
}

func convertArray[I, O any](in []I, get func(I) O) (out []O) {
	lin := len(in)
	if lin == 0 {
		return
	}
	out = make([]O, lin)
	for i, v := range in {
		out[i] = get(v)
	}
	return
}

// Magnitude window for Dec values emitted in events, so tiny/huge computed
// values don't serialize into oversized strings. Independent of the ingress
// BoundedExp40Dec guard; shapes event output only, never consensus state.
const eventDecClampExponent = 40

var (
	eventDecMinMagnitude = alloraMath.MustNewDecFromString(fmt.Sprintf("1e-%d", eventDecClampExponent))
	eventDecMaxMagnitude = alloraMath.MustNewDecFromString(fmt.Sprintf("1e%d", eventDecClampExponent))
)

// clampDec bounds one Dec for event output.
func clampDec(d alloraMath.Dec) alloraMath.Dec {
	return alloraMath.ClampMagnitude(d, eventDecMinMagnitude, eventDecMaxMagnitude)
}

// clampDecs clamps a pre-built slice (params not routed through convertArray).
func clampDecs(in []alloraMath.Dec) []alloraMath.Dec {
	if in == nil {
		return nil
	}
	out := make([]alloraMath.Dec, len(in))
	for i := range in {
		out[i] = clampDec(in[i])
	}
	return out
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

func NewNodeOwnerUpdatedEventBase(nodeAddress, oldOwner, newOwner string, isReputer bool) proto.Message {
	return &EventNodeOwnerUpdated{
		NodeAddress: nodeAddress,
		OldOwner:    oldOwner,
		NewOwner:    newOwner,
		IsReputer:   isReputer,
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
		Score:            clampDec(score),
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
		Scores:    clampDecs(scoreValues),
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
		Rewards:       clampDecs(rewardValues),
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
		Rewards:  clampDecs(rewardValues),
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
		Coefficients: clampDecs(coefficients),
	}
}

// Regrets

func NewInfererNetworkRegretSetEventBase(topicID uint64, blockHeight int64, addresses []string, regrets []alloraMath.Dec) proto.Message {
	return &EventInfererNetworkRegretSet{
		TopicId:     topicID,
		BlockHeight: blockHeight,
		Addresses:   addresses,
		Regrets:     clampDecs(regrets),
	}
}

func NewForecasterNetworkRegretSetEventBase(topicID uint64, blockHeight int64, addresses []string, regrets []alloraMath.Dec) proto.Message {
	return &EventForecasterNetworkRegretSet{
		TopicId:     topicID,
		BlockHeight: blockHeight,
		Addresses:   addresses,
		Regrets:     clampDecs(regrets),
	}
}

func NewNaiveInfererNetworkRegretSetEventBase(topicID uint64, blockHeight int64, addresses []string, regrets []alloraMath.Dec) proto.Message {
	return &EventNaiveInfererNetworkRegretSet{
		TopicId:     topicID,
		BlockHeight: blockHeight,
		Addresses:   addresses,
		Regrets:     clampDecs(regrets),
	}
}

func NewTopicInitialRegretSetEventBase(topicID uint64, blockHeight int64, regret alloraMath.Dec) proto.Message {
	return &EventTopicInitialRegretSet{
		TopicId:     topicID,
		BlockHeight: blockHeight,
		Regret:      clampDec(regret),
	}
}

func NewTopicInitialEmaScoreSetEventBase(actorType ActorType, topicId uint64, blockHeight int64, score alloraMath.Dec) proto.Message {
	return &EventTopicInitialEmaScoreSet{
		ActorType:   actorType,
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Score:       clampDec(score),
	}
}

func NewRegretStdNormSetEventBase(topicId uint64, blockHeight int64, stdNorm alloraMath.Dec) proto.Message {
	return &EventRegretStdNormSet{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Stdnorm:     clampDec(stdNorm),
	}
}

func NewInfererWeightsSetEventBase(topicId uint64, blockHeight int64, addresses []string, weights []alloraMath.Dec) proto.Message {
	return &EventInfererWeightsSet{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Addresses:   addresses,
		Weights:     clampDecs(weights),
	}
}

func NewForecasterWeightsSetEventBase(topicId uint64, blockHeight int64, addresses []string, weights []alloraMath.Dec) proto.Message {
	return &EventForecasterWeightsSet{
		TopicId:     topicId,
		BlockHeight: blockHeight,
		Addresses:   addresses,
		Weights:     clampDecs(weights),
	}
}

// Previous Percentage Reward

func NewPreviousPercentageRewardToStakedReputersSetEventBase(blockHeight int64, percentage alloraMath.Dec) proto.Message {
	return &EventPreviousPercentageRewardToStakedReputersSet{
		BlockHeight: blockHeight,
		Percentage:  clampDec(percentage),
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
		RewardPerShare: clampDec(rewardPerShare),
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
		Weights:          clampDecs(weights),
	}
}

func NewNetworkInferenceForecasterWeightsSetEventBase(topicId TopicId, blockHeight int64, addresses []string, weights []alloraMath.Dec) proto.Message {
	return &EventNetworkInferenceForecasterWeightsSet{
		TopicId:          topicId,
		NonceBlockHeight: blockHeight,
		Addresses:        addresses,
		Weights:          clampDecs(weights),
	}
}

func NewNetworkInferenceInfererRegretsUsedSetEventBase(topicId TopicId, blockHeight int64, addresses []string, regrets []alloraMath.Dec) proto.Message {
	return &EventNetworkInferenceInfererRegretsUsedSet{
		TopicId:          topicId,
		NonceBlockHeight: blockHeight,
		Addresses:        addresses,
		Regrets:          clampDecs(regrets),
	}
}

func NewNetworkInferenceForecasterRegretsUsedSetEventBase(topicId TopicId, blockHeight int64, addresses []string, regrets []alloraMath.Dec) proto.Message {
	return &EventNetworkInferenceForecasterRegretsUsedSet{
		TopicId:          topicId,
		NonceBlockHeight: blockHeight,
		Addresses:        addresses,
		Regrets:          clampDecs(regrets),
	}
}

func NewTopicWeightUpdatedEventBase(topicId TopicId, newWeight alloraMath.Dec, topicStake math.Int, topicFeeRevenue math.Int) proto.Message {
	return &EventTopicWeightUpdated{
		TopicId:         topicId,
		NewWeight:       clampDec(newWeight),
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

// NewEpochLabelRegistryFrozenEventBase is emitted once per (topicId, nonce)
// after the final active inputs have been finalized into a registry at
// CloseWorkerNonce time. Offchain indexers can reconstruct the full
// registry by looking up topicLabelRegistry at the same key, but we
// advertise the size here so explorers don't have to read state.
func NewEpochLabelRegistryFrozenEventBase(topicId TopicId, nonceBlockHeight int64, registrySize uint64) proto.Message {
	return &EventEpochLabelRegistryFrozen{
		TopicId:          topicId,
		NonceBlockHeight: nonceBlockHeight,
		RegistrySize:     registrySize,
	}
}
