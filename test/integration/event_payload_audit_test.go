package integration_test

import (
	"bytes"
	"fmt"
	"slices"

	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// Largest difference accepted when the combined value is recomputed from the
// emitted weights, which only carry the chain's finite decimal precision.
var combinedValueTolerance = alloraMath.MustNewDecFromString("1e-15")

// networkWeights holds the weights a network inference used for one nonce.
type networkWeights struct {
	infererAddresses    []string
	infererWeights      []alloraMath.Dec
	forecasterAddresses []string
	forecasterWeights   []alloraMath.Dec
}

// takeTransfer removes one transfer matching the arguments and reports whether
// one existed, so each settled amount accounts for exactly one coin movement.
func takeTransfer(transfers []bankTransfer, sender, recipient string, amount cosmosMath.Int) ([]bankTransfer, bool) {
	for i, transfer := range transfers {
		if transfer.sender == sender && transfer.recipient == recipient && transfer.amount.Equal(amount) {
			return slices.Delete(transfers, i, i+1), true
		}
	}
	return transfers, false
}

// auditRewardPayouts ties every settled reward to the coins that moved: one bank
// transfer of its integer part per actor, a restake event for reputers, and a
// rewards account whose balance change is fully explained by the emission,
// forfeits, delegate rewards and settled amounts the block's events report.
func (a *eventAudit) auditRewardPayouts(block chainBlock) {
	height := block.height
	stakingAddr, err := sdk.Bech32ifyAddressBytes(params.HumanCoinUnit, authtypes.NewModuleAddress(emissionstypes.AlloraStakingAccountName))
	if err != nil {
		a.failf(height, "staking module address: %v", err)
		return
	}
	transfers := a.blockTransfers(block)
	received := cosmosMath.ZeroInt()
	for _, transfer := range transfers {
		if transfer.recipient == a.rewardsAddr {
			received = received.Add(transfer.amount)
		}
	}
	restakes := make([]*emissionstypes.EventAddStake, 0)
	for _, event := range block.events {
		if restake, ok := event.typed.(*emissionstypes.EventAddStake); ok && event.phase == phaseEndBlock {
			restakes = append(restakes, restake)
		}
	}

	settledTotal := cosmosMath.ZeroInt()
	for _, event := range block.events {
		settled, ok := event.typed.(*emissionstypes.EventRewardsSettled)
		if !ok {
			continue
		}
		if len(settled.Addresses) != len(settled.Rewards) {
			a.failf(height, "EventRewardsSettled topic %d has %d addresses and %d rewards", settled.TopicId, len(settled.Addresses), len(settled.Rewards))
			continue
		}
		for i, address := range settled.Addresses {
			amount, err := settled.Rewards[i].SdkIntTrim()
			if err != nil {
				a.failf(height, "EventRewardsSettled topic %d reward %s: %v", settled.TopicId, settled.Rewards[i], err)
				continue
			}
			settledTotal = settledTotal.Add(amount)
			recipient := stakingAddr
			if settled.ActorType != emissionstypes.ActorType_ACTOR_TYPE_REPUTER {
				recipient = a.workerOwner(height, address)
			}
			var moved bool
			transfers, moved = takeTransfer(transfers, a.rewardsAddr, recipient, amount)
			if !moved {
				a.failf(height, "EventRewardsSettled %s %s topic %d settled %s, no transfer of that amount from rewards to %s", settled.ActorType, address, settled.TopicId, amount, recipient)
			}
			if settled.ActorType == emissionstypes.ActorType_ACTOR_TYPE_REPUTER {
				match := slices.IndexFunc(restakes, func(restake *emissionstypes.EventAddStake) bool {
					return restake.TopicId == settled.TopicId && restake.Reputer == address && restake.Delegator == address && restake.Amount.Equal(amount)
				})
				if match < 0 {
					a.failf(height, "EventRewardsSettled reputer %s topic %d settled %s, no EventAddStake restakes that amount", address, settled.TopicId, amount)
				} else {
					restakes = slices.Delete(restakes, match, match+1)
				}
			}
		}
	}

	forfeited, delegateDistributed, _, _ := expectedLedgerEvents(block)
	delta := a.balance(height, a.rewardsAddr).Sub(a.balance(height-1, a.rewardsAddr))
	expected := received.Sub(forfeited).Sub(delegateDistributed).Sub(settledTotal)
	if !delta.Equal(expected) {
		a.failf(height, "rewards account changed by %s, events explain %s: received %s, forfeited %s, delegate rewards %s, settled %s", delta, expected, received, forfeited, delegateDistributed, settledTotal)
	}
}

func (a *eventAudit) workerOwner(height int64, address string) string {
	response, err := a.m.Client.QueryEmissions().GetWorkerNodeInfo(atHeight(a.ctx, height), &emissionstypes.GetWorkerNodeInfoRequest{Address: address})
	if err != nil {
		a.failf(height, "GetWorkerNodeInfo %s: %v", address, err)
		return ""
	}
	return response.NodeInfo.Owner
}

// auditRegretsUsed checks the regrets a network inference used. Regrets only change
// when a reputer nonce closes, which runs before worker closes in EndBlock, so the
// state committed by this block holds the values the inference read.
func (a *eventAudit) auditRegretsUsed(height int64, name string, topicId uint64, addresses []string, regrets []alloraMath.Dec, query func(string) (*emissionstypes.TimestampedValue, error)) {
	if len(addresses) != len(regrets) {
		a.failf(height, "%s topic %d has %d addresses and %d regrets", name, topicId, len(addresses), len(regrets))
		return
	}
	for i, actor := range addresses {
		stored, err := query(actor)
		if err != nil {
			a.failf(height, "%s stored regret topic %d actor %s: %v", name, topicId, actor, err)
			continue
		}
		a.requireDecEqual(height, fmt.Sprintf("%s topic %d actor %s", name, topicId, actor), regrets[i], stored.Value)
	}
}

// auditCombinedValuesFromWeights recomputes each network inference from its emitted
// weights: I_i = Σ w_il I_il / Σ w_il over inferers and forecast-implied inferences.
// No query returns these weights, so the bundle they produced is what checks them.
func (a *eventAudit) auditCombinedValuesFromWeights(height int64, facts blockContext) {
	for key, weights := range facts.networkWeights { //nolint:maprange // every entry is checked and the order does not matter
		bundle, found := facts.networkBundles[key]
		if !found {
			a.failf(height, "network inference weights for %s have no EventNetworkInferenceBundle", key)
			continue
		}
		if !slices.Equal(weights.infererAddresses, bundle.InfererAddresses) {
			a.failf(height, "network inference weights for %s name inferers %v, bundle has %v", key, weights.infererAddresses, bundle.InfererAddresses)
			continue
		}
		if !slices.Equal(weights.forecasterAddresses, bundle.ForecasterAddresses) {
			a.failf(height, "network inference weights for %s name forecasters %v, bundle has %v", key, weights.forecasterAddresses, bundle.ForecasterAddresses)
			continue
		}
		if !hasRowsForWeights(bundle.InfererValues, weights.infererWeights, len(bundle.CombinedValue)) ||
			!hasRowsForWeights(bundle.ForecasterValues, weights.forecasterWeights, len(bundle.CombinedValue)) {
			a.failf(height, "network inference weights for %s do not fit the bundle's value matrices", key)
			continue
		}
		for label := range bundle.CombinedValue {
			sum, sumWeights := alloraMath.ZeroDec(), alloraMath.ZeroDec()
			accumulate := func(weight, value alloraMath.Dec) error {
				product, err := weight.Mul(value)
				if err != nil {
					return err
				}
				if sum, err = sum.Add(product); err != nil {
					return err
				}
				sumWeights, err = sumWeights.Add(weight)
				return err
			}
			var err error
			for i, weight := range weights.infererWeights {
				if err == nil {
					err = accumulate(weight, bundle.InfererValues[i][label])
				}
			}
			for i, weight := range weights.forecasterWeights {
				if err == nil {
					err = accumulate(weight, bundle.ForecasterValues[i][label])
				}
			}
			if err != nil {
				a.failf(height, "recomputing combined value of %s: %v", key, err)
				continue
			}
			recomputed, err := sum.Quo(sumWeights)
			if err != nil {
				a.failf(height, "recomputing combined value of %s: %v", key, err)
				continue
			}
			difference, err := recomputed.Sub(bundle.CombinedValue[label])
			if err == nil {
				difference, err = difference.Abs()
			}
			if err != nil || difference.Gt(combinedValueTolerance) {
				a.failf(height, "%s combined_value[%d] is %s, its emitted weights produce %s", key, label, bundle.CombinedValue[label], recomputed)
			}
		}
	}
}

// hasRowsForWeights reports whether a matrix has one row of numLabels per weight.
func hasRowsForWeights(rows alloraMath.DecMatrix, weights []alloraMath.Dec, numLabels int) bool {
	if len(rows) != len(weights) {
		return false
	}
	for _, row := range rows {
		if len(row) != numLabels {
			return false
		}
	}
	return true
}

// requireLabeledRow compares one event row with a stored labeled vector.
func (a *eventAudit) requireLabeledRow(height int64, what string, labels []string, row alloraMath.DecArray, stored []*emissionstypes.LabeledValue) {
	if len(row) != len(stored) || len(row) != len(labels) {
		a.failf(height, "%s has %d values for %d labels, stored row has %d", what, len(row), len(labels), len(stored))
		return
	}
	for i, value := range stored {
		if value.LabelName != labels[i] {
			a.failf(height, "%s position %d is label %q in the event, %q in the stored row", what, i, labels[i], value.LabelName)
		}
		a.requireDecEqual(height, fmt.Sprintf("%s[%s]", what, labels[i]), row[i], value.Value)
	}
}

// requireRowCount reports whether an event matrix and its stored rows have one size.
func (a *eventAudit) requireRowCount(height int64, what string, event, stored int) bool {
	if event != stored {
		a.failf(height, "%s has %d rows in the event, %d stored", what, event, stored)
		return false
	}
	return true
}

// auditNetworkInferenceBundle compares every field of the event with the bundle
// stored under its nonce. Rows without an address of their own are only usable by
// position, so their stored actor must sit at the same index of the address list.
func (a *eventAudit) auditNetworkInferenceBundle(height int64, facts blockContext, event *emissionstypes.EventNetworkInferenceBundle) {
	key := topicNonceKey(event.TopicId, event.Nonce)
	a.requireWorkerNonceOpen(height, "EventNetworkInferenceBundle", event.TopicId, event.Nonce)
	if labels, frozen := a.frozenLabels[key]; !frozen {
		a.failf(height, "EventNetworkInferenceBundle topic %d nonce %d has no EventEpochLabelRegistryFrozen", event.TopicId, event.Nonce)
	} else if !slices.Equal(labels, event.LabelNames) {
		a.failf(height, "EventNetworkInferenceBundle topic %d nonce %d label_names %v, frozen registry %v", event.TopicId, event.Nonce, event.LabelNames, labels)
	}
	if facts.droppedNonces[key] {
		a.skipped["network inference bundle vs stored: close failed"]++
		return
	}
	stored := a.storedNetworkInferences(height, event)
	if stored == nil {
		return
	}
	prefix := fmt.Sprintf("EventNetworkInferenceBundle topic %d nonce %d outlier=%t ", event.TopicId, event.Nonce, event.OutlierResistant)
	labels := event.LabelNames
	a.requireLabeledRow(height, prefix+"combined_value", labels, event.CombinedValue, stored.CombinedValue)
	a.requireLabeledRow(height, prefix+"naive_value", labels, event.NaiveValue, stored.NaiveValue)

	workerRows := func(name string, addresses []string, rows alloraMath.DecMatrix, workers []*emissionstypes.WorkerInference) {
		if !a.requireRowCount(height, prefix+name, len(rows), len(workers)) || !a.requireRowCount(height, prefix+name+" addresses", len(addresses), len(workers)) {
			return
		}
		for i, worker := range workers {
			if worker.Worker != addresses[i] {
				a.failf(height, "%s%s address %d is %s, stored %s", prefix, name, i, addresses[i], worker.Worker)
			}
			a.requireLabeledRow(height, fmt.Sprintf("%s%s[%s]", prefix, name, worker.Worker), labels, rows[i], worker.Values)
		}
	}
	workerRows("inferer_values", event.InfererAddresses, event.InfererValues, stored.InfererValues)
	workerRows("forecaster_values", event.ForecasterAddresses, event.ForecasterValues, stored.ForecasterValues)

	if a.requireRowCount(height, prefix+"one_out_inferer_values", len(event.OneOutInfererValues), len(stored.OneOutInfererValues)) {
		for i, value := range stored.OneOutInfererValues {
			a.requireAlignedActor(height, prefix+"one_out_inferer_values", i, value.WithheldInferer, event.InfererAddresses)
			a.requireLabeledRow(height, fmt.Sprintf("%sone_out_inferer_values[%s]", prefix, value.WithheldInferer), labels, event.OneOutInfererValues[i], value.CombinedInference)
		}
	}
	if a.requireRowCount(height, prefix+"one_out_forecaster_values", len(event.OneOutForecasterValues), len(stored.OneOutForecasterValues)) {
		for i, value := range stored.OneOutForecasterValues {
			a.requireAlignedActor(height, prefix+"one_out_forecaster_values", i, value.WithheldForecaster, event.ForecasterAddresses)
			a.requireLabeledRow(height, fmt.Sprintf("%sone_out_forecaster_values[%s]", prefix, value.WithheldForecaster), labels, event.OneOutForecasterValues[i], value.CombinedInference)
		}
	}
	if a.requireRowCount(height, prefix+"one_in_forecaster_values", len(event.OneInForecasterValues), len(stored.OneInForecasterValues)) {
		for i, value := range stored.OneInForecasterValues {
			a.requireAlignedActor(height, prefix+"one_in_forecaster_values", i, value.Forecaster, event.ForecasterAddresses)
			a.requireLabeledRow(height, fmt.Sprintf("%sone_in_forecaster_values[%s]", prefix, value.Forecaster), labels, event.OneInForecasterValues[i], value.CombinedInference)
		}
	}
	a.auditOneOutInfererForecasterMatrix(height, prefix, event, stored)
}

// auditOneOutInfererForecasterMatrix checks the proto's documented shape,
// #forecasters × #inferers × L, against the stored rows.
func (a *eventAudit) auditOneOutInfererForecasterMatrix(height int64, prefix string, event *emissionstypes.EventNetworkInferenceBundle, stored *emissionstypes.NetworkInferenceBundle) {
	const name = "one_out_inferer_forecaster_values"
	order := make([]string, 0)
	byForecaster := make(map[string][]*emissionstypes.OneOutInfererForecasterValue)
	for _, value := range stored.OneOutInfererForecasterValues {
		if _, seen := byForecaster[value.Forecaster]; !seen {
			order = append(order, value.Forecaster)
		}
		byForecaster[value.Forecaster] = append(byForecaster[value.Forecaster], value)
	}
	if !a.requireRowCount(height, prefix+name, len(event.OneOutInfererForecasterValues), len(order)) {
		return
	}
	for f, forecaster := range order {
		a.requireAlignedActor(height, prefix+name, f, forecaster, event.ForecasterAddresses)
		rows := byForecaster[forecaster]
		if !a.requireRowCount(height, fmt.Sprintf("%s%s[%s]", prefix, name, forecaster), len(event.OneOutInfererForecasterValues[f]), len(rows)) {
			continue
		}
		if len(rows) != len(event.InfererAddresses) {
			a.failf(height, "%s%s[%s] has %d rows, the proto documents one per inferer and the event lists %d", prefix, name, forecaster, len(rows), len(event.InfererAddresses))
		}
		for i, row := range rows {
			a.requireAlignedActor(height, fmt.Sprintf("%s%s[%s]", prefix, name, forecaster), i, row.WithheldInferer, event.InfererAddresses)
			a.requireLabeledRow(height, fmt.Sprintf("%s%s[%s][%s]", prefix, name, forecaster, row.WithheldInferer), event.LabelNames, event.OneOutInfererForecasterValues[f][i], row.CombinedInference)
		}
	}
}

// requireAlignedActor checks that the actor of a positional row is the address at
// the same index of the list a consumer must read it against.
func (a *eventAudit) requireAlignedActor(height int64, what string, index int, actor string, addresses []string) {
	if index >= len(addresses) || addresses[index] != actor {
		a.failf(height, "%s row %d belongs to %s, the address list has %v", what, index, actor, addresses)
	}
}

func (a *eventAudit) storedNetworkInferences(height int64, event *emissionstypes.EventNetworkInferenceBundle) *emissionstypes.NetworkInferenceBundle {
	queryCtx := atHeight(a.ctx, height)
	if event.OutlierResistant {
		response, err := a.m.Client.QueryEmissions().GetNetworkInferencesAtBlockOutlierResistant(queryCtx, &emissionstypes.GetNetworkInferencesAtBlockOutlierResistantRequest{TopicId: event.TopicId, BlockHeightLastInference: event.Nonce})
		if err != nil {
			a.failf(height, "GetNetworkInferencesAtBlockOutlierResistant topic %d nonce %d: %v", event.TopicId, event.Nonce, err)
			return nil
		}
		return response.NetworkInferences
	}
	response, err := a.m.Client.QueryEmissions().GetNetworkInferencesAtBlock(queryCtx, &emissionstypes.GetNetworkInferencesAtBlockRequest{TopicId: event.TopicId, BlockHeightLastInference: event.Nonce})
	if err != nil {
		a.failf(height, "GetNetworkInferencesAtBlock topic %d nonce %d: %v", event.TopicId, event.Nonce, err)
		return nil
	}
	return response.NetworkInferences
}

// requireValues compares a positional event list with stored attributed values.
func (a *eventAudit) requireValues(height int64, what string, event []alloraMath.Dec, stored []alloraMath.Dec) {
	if !a.requireRowCount(height, what, len(event), len(stored)) {
		return
	}
	for i := range stored {
		a.requireDecEqual(height, fmt.Sprintf("%s[%d]", what, i), event[i], stored[i])
	}
}

// auditNetworkLoss compares every field of the event's bundle with the loss bundle
// stored under its nonce.
func (a *eventAudit) auditNetworkLoss(height int64, event *emissionstypes.EventNetworkLossSet) {
	a.requireReputerNonceOpen(height, "EventNetworkLossSet", event.TopicId, event.Nonce)
	response, err := a.m.Client.QueryEmissions().GetNetworkLossBundleAtBlock(atHeight(a.ctx, height), &emissionstypes.GetNetworkLossBundleAtBlockRequest{TopicId: event.TopicId, BlockHeight: event.Nonce})
	if err != nil {
		a.failf(height, "GetNetworkLossBundleAtBlock topic %d nonce %d: %v", event.TopicId, event.Nonce, err)
		return
	}
	stored := response.LossBundle
	bundle := event.Bundle
	prefix := fmt.Sprintf("EventNetworkLossSet topic %d nonce %d ", event.TopicId, event.Nonce)
	if !bytes.Equal(bundle.ExtraData, stored.ExtraData) {
		a.failf(height, "%sextra_data %x, stored %x", prefix, bundle.ExtraData, stored.ExtraData)
	}
	a.requireDecEqual(height, prefix+"combined_value", bundle.CombinedValue, stored.CombinedValue)
	a.requireDecEqual(height, prefix+"naive_value", bundle.NaiveValue, stored.NaiveValue)

	attributed := func(name string, addresses []string, values []alloraMath.Dec, stored []*emissionstypes.WorkerAttributedValue) {
		storedValues := make([]alloraMath.Dec, len(stored))
		for i, value := range stored {
			storedValues[i] = value.Value
			a.requireAlignedActor(height, prefix+name, i, value.Worker, addresses)
		}
		a.requireValues(height, prefix+name, values, storedValues)
	}
	attributed("inferer_values", bundle.InfererAddresses, bundle.InfererValues, stored.InfererValues)
	attributed("forecaster_values", bundle.ForecasterAddresses, bundle.ForecasterValues, stored.ForecasterValues)
	attributed("one_in_forecaster_values", bundle.ForecasterAddresses, bundle.OneInForecasterValues, stored.OneInForecasterValues)

	withheld := func(name string, addresses []string, values []alloraMath.Dec, stored []*emissionstypes.WithheldWorkerAttributedValue) {
		storedValues := make([]alloraMath.Dec, len(stored))
		for i, value := range stored {
			storedValues[i] = value.Value
			a.requireAlignedActor(height, prefix+name, i, value.Worker, addresses)
		}
		a.requireValues(height, prefix+name, values, storedValues)
	}
	withheld("one_out_inferer_values", bundle.InfererAddresses, bundle.OneOutInfererValues, stored.OneOutInfererValues)
	withheld("one_out_forecaster_values", bundle.ForecasterAddresses, bundle.OneOutForecasterValues, stored.OneOutForecasterValues)

	const matrix = "one_out_inferer_forecaster_values"
	if len(bundle.InfererAddresses) == 0 || len(stored.OneOutInfererForecasterValues) == 0 {
		if len(bundle.OneOutInfererForecasterValues) != 0 {
			a.failf(height, "%s%s has %d rows with nothing stored", prefix, matrix, len(bundle.OneOutInfererForecasterValues))
		}
		return
	}
	if !a.requireRowCount(height, prefix+matrix, len(bundle.OneOutInfererForecasterValues), len(stored.OneOutInfererForecasterValues)) {
		return
	}
	for f, row := range stored.OneOutInfererForecasterValues {
		a.requireAlignedActor(height, prefix+matrix, f, row.Forecaster, bundle.ForecasterAddresses)
		cells := bundle.OneOutInfererForecasterValues[f]
		if !a.requireRowCount(height, fmt.Sprintf("%s%s[%s]", prefix, matrix, row.Forecaster), len(cells), len(bundle.InfererAddresses)) {
			continue
		}
		for j, inferer := range bundle.InfererAddresses {
			expected := alloraMath.NewNaN()
			for _, cell := range row.OneOutInfererValues {
				if cell.Worker == inferer {
					expected = cell.Value
				}
			}
			a.requireDecEqual(height, fmt.Sprintf("%s%s[%s][%s]", prefix, matrix, row.Forecaster, inferer), cells[j], expected)
		}
	}
}
