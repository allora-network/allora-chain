package fuzz_test

import (
	"context"

	cosmossdk_io_math "cosmossdk.io/math"

	testcommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// helper function that queries the chain for the amount of stake that a reputer has in a topic
func getReputerStakeFromChain(
	m *testcommon.TestConfig,
	actor Actor,
	topicId uint64,
	failOnErr bool,
	iteration int,
) cosmossdk_io_math.Int {
	ctx := context.Background()
	qe := m.Client.QueryEmissions()
	stakeResp, err := qe.GetStakeFromReputerInTopicInSelf(ctx, &emissionstypes.GetStakeFromReputerInTopicInSelfRequest{
		ReputerAddress: actor.addr,
		TopicId:        topicId,
	})
	failIfOnErr(m.T, failOnErr, err)
	iterLog(m.T, iteration, "Query chain stake from reputer in topic in self", actor, "in topic id", topicId, "is", stakeResp.Amount)
	return stakeResp.Amount
}

// pickPercentOfStakeByReputer picks a random percent (1/10, 1/3, 1/2, 6/7, or full amount) of the stake by a reputer
func pickPercentOfStakeByReputer(
	m *testcommon.TestConfig,
	topicId uint64,
	actor Actor,
	data *SimulationData,
	iteration int,
) cosmossdk_io_math.Int {
	reg := Registration{
		TopicId: topicId,
		Actor:   actor,
	}
	_, exists := data.reputerStakes.Get(reg)
	if !exists {
		return cosmossdk_io_math.ZeroInt()
	}
	amount := getReputerStakeFromChain(m, actor, topicId, data.failOnErr, iteration)
	return pickPercentOf(m.Client.Rand, amount)
}

// stake actor as a reputer, pick a random amount to stake that is less than half their current balance
func stakeAsReputer(
	m *testcommon.TestConfig,
	actor Actor,
	_ Actor,
	amount *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) (success bool) {
	iterLog(
		m.T,
		iteration,
		"staking as a reputer",
		actor,
		"in topic id",
		topicId,
		" in amount",
		amount.String(),
	)
	msg := emissionstypes.AddStakeRequest{
		Sender:  actor.addr,
		TopicId: topicId,
		Amount:  *amount,
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, &msg)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"stake failed",
			actor,
			"as a reputer in topic id ",
			topicId,
			" in amount ",
			amount.String(),
			"tx broadcast error",
			err,
		)
		return false
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"stake failed",
			actor,
			"as a reputer in topic id ",
			topicId,
			" in amount ",
			amount.String(),
			"tx wait error",
			err,
		)
		return false
	}

	response := &emissionstypes.AddStakeResponse{}
	err = txResp.Decode(response)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"stake failed",
			actor,
			"as a reputer in topic id ",
			topicId,
			" in amount ",
			amount.String(),
			"tx decode error",
			err,
		)
		return false
	}
	data.addReputerStaked(topicId, actor)
	data.counts.incrementStakeAsReputerCount()
	iterSuccessLog(
		m.T,
		iteration,
		"staked ",
		actor,
		"as a reputer in topic id ",
		topicId,
		" in amount ",
		amount.String(),
	)
	return true
}

// tell if any reputers are currently staked
func anyReputersStaked(data *SimulationData) bool {
	return data.reputerStakes.Len() > 0
}

// tell if any delegators are currently staked
func anyDelegatorsStaked(data *SimulationData) bool {
	return data.delegatorStakes.Len() > 0
}

// mark stake for removal as a reputer
// the amount will either be 1/10, 1/3, 1/2, 6/7, or the full amount of their
// current stake to be removed
func unstakeAsReputer(
	m *testcommon.TestConfig,
	actor Actor,
	_ Actor,
	amount *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) (success bool) {
	iterLog(
		m.T,
		iteration,
		"unstaking as a reputer",
		actor,
		"in topic id",
		topicId,
		" in amount",
		amount.String(),
	)

	msg := emissionstypes.RemoveStakeRequest{
		Sender:  actor.addr,
		TopicId: topicId,
		Amount:  *amount,
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, &msg)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"unstake failed",
			actor,
			"as a reputer in topic id ",
			topicId,
			" in amount ",
			amount.String(),
			"tx broadcast error",
			err,
		)
		return false
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"unstake failed",
			actor,
			"as a reputer in topic id ",
			topicId,
			" in amount ",
			amount.String(),
			"tx wait error",
			err,
		)
		return false
	}
	response := &emissionstypes.RemoveStakeResponse{}
	err = txResp.Decode(response)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"unstake failed",
			actor,
			"as a reputer in topic id ",
			topicId,
			" in amount ",
			amount.String(),
			"tx decode error",
			err,
		)
		return false
	}
	data.counts.incrementUnstakeAsReputerCount()
	iterSuccessLog(
		m.T,
		iteration,
		"unstaked from ",
		actor,
		"as a reputer in topic id ",
		topicId,
		" in amount ",
		amount.String(),
	)
	// if the reputer will have no stake left after this unstake, remove them from the list of staked reputers
	chainAmount := getReputerStakeFromChain(m, actor, topicId, data.failOnErr, iteration)
	if chainAmount.Equal(*amount) {
		data.removeReputerStaked(topicId, actor)
	}
	return true
}

func cancelStakeRemoval(
	m *testcommon.TestConfig,
	actor Actor,
	_ Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) (success bool) {
	iterLog(
		m.T,
		iteration,
		"cancelling stake removal as a reputer",
		actor,
		"in topic id",
		topicId,
	)
	msg := emissionstypes.CancelRemoveStakeRequest{
		Sender:  actor.addr,
		TopicId: topicId,
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, &msg)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"cancelling stake removal as a reputer failed",
			actor,
			"in topic id",
			topicId,
			"tx broadcast error",
			err,
		)
		return false
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"cancelling stake removal as a reputer failed",
			actor,
			"in topic id",
			topicId,
			"tx wait error",
			err,
		)
		return false
	}
	response := &emissionstypes.CancelRemoveStakeResponse{}
	err = txResp.Decode(response)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"cancelling stake removal as a reputer failed",
			actor,
			"in topic id",
			topicId,
			"tx decode error",
			err,
		)
		return false
	}

	data.counts.incrementCancelStakeRemovalCount()
	iterSuccessLog(
		m.T,
		iteration,
		"cancelled stake removal as a reputer",
		actor,
		"in topic id",
		topicId,
	)
	// make sure this reputer is still in the list of staked reputers
	data.addReputerStaked(topicId, actor)
	return true
}

// helper function that queries the chain for the amount of stake that a delegator has in a reputer
func getDelegatorStakeFromChain(
	m *testcommon.TestConfig,
	delegator Actor,
	reputer Actor,
	topicId uint64,
	failOnErr bool,
	iteration int,
) cosmossdk_io_math.Int {
	ctx := context.Background()
	qe := m.Client.QueryEmissions()
	stakeResp, err := qe.GetStakeFromDelegatorInTopicInReputer(
		ctx,
		&emissionstypes.GetStakeFromDelegatorInTopicInReputerRequest{
			DelegatorAddress: delegator.addr,
			TopicId:          topicId,
			ReputerAddress:   reputer.addr,
		},
	)
	failIfOnErr(m.T, failOnErr, err)
	iterLog(m.T, iteration, "Query chain stake from delegator", delegator, "upon reputer", reputer, "in topic id", topicId, "is", stakeResp.Amount)
	return stakeResp.Amount
}

// pick a random percent (1/10, 1/3, 1/2, 6/7, or full amount) of the stake that a delegator has in a reputer
func pickPercentOfStakeByDelegator(
	m *testcommon.TestConfig,
	topicId uint64,
	delegator Actor,
	reputer Actor,
	data *SimulationData,
	iteration int,
) cosmossdk_io_math.Int {
	del := Delegation{
		TopicId:   topicId,
		Delegator: delegator,
		Reputer:   reputer,
	}
	_, exists := data.delegatorStakes.Get(del)
	if !exists {
		return cosmossdk_io_math.ZeroInt()
	}
	amount := getDelegatorStakeFromChain(m, delegator, reputer, topicId, data.failOnErr, iteration)

	return pickPercentOf(m.Client.Rand, amount)
}

// stake as a delegator upon a reputer
// NOTE: in this case, the param actor is the reputer being staked upon,
// rather than the actor doing the staking.
func delegateStake(
	m *testcommon.TestConfig,
	delegator Actor,
	reputer Actor,
	amount *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) (success bool) {
	iterLog(
		m.T,
		iteration,
		"delegating stake",
		delegator,
		"upon reputer",
		reputer,
		"in topic id",
		topicId,
		" in amount",
		amount.String(),
	)
	msg := emissionstypes.DelegateStakeRequest{
		Sender:  delegator.addr,
		Reputer: reputer.addr,
		TopicId: topicId,
		Amount:  *amount,
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, delegator.acc, &msg)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"delegation failed",
			delegator,
			"upon reputer",
			reputer,
			"in topic id",
			topicId,
			" in amount",
			amount.String(),
			"tx broadcast error",
			err,
		)
		return false
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"delegation failed",
			delegator,
			"upon reputer",
			reputer,
			"in topic id",
			topicId,
			" in amount",
			amount.String(),
			"tx wait error",
			err,
		)
		return false
	}

	registerWorkerResponse := &emissionstypes.DelegateStakeResponse{}
	err = txResp.Decode(registerWorkerResponse)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"delegation failed",
			delegator,
			"upon reputer",
			reputer,
			"in topic id",
			topicId,
			" in amount",
			amount.String(),
			"tx decode error",
			err,
		)
		return false
	}

	data.addDelegatorDelegated(topicId, delegator, reputer)
	data.counts.incrementDelegateStakeCount()
	iterSuccessLog(
		m.T,
		iteration,
		"delegating stake",
		delegator,
		"upon reputer",
		"in topic id",
		topicId,
		" in amount",
		amount.String(),
	)
	return true
}

// undelegate a percentage of the stake that the delegator has upon the reputer, either 1/10, 1/3, 1/2, 6/7, or the full amount
func undelegateStake(
	m *testcommon.TestConfig,
	delegator Actor,
	reputer Actor,
	amount *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) (success bool) {
	iterLog(
		m.T,
		iteration,
		"delegator ",
		delegator,
		" unstaking from reputer ",
		reputer,
		" in topic id ",
		topicId,
		" in amount ",
		amount.String(),
	)
	msg := emissionstypes.RemoveDelegateStakeRequest{
		Sender:  delegator.addr,
		Reputer: reputer.addr,
		TopicId: topicId,
		Amount:  *amount,
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, delegator.acc, &msg)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"undelegation failed",
			delegator,
			"from reputer",
			reputer,
			"in topic id",
			topicId,
			" in amount",
			amount.String(),
			"tx broadcast error",
			err,
		)
		return false
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"undelegation failed",
			delegator,
			"from reputer",
			reputer,
			"in topic id",
			topicId,
			" in amount",
			amount.String(),
			"tx wait error",
			err,
		)
		return false
	}

	response := &emissionstypes.RemoveDelegateStakeResponse{}
	err = txResp.Decode(response)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"undelegation failed",
			delegator,
			"from reputer",
			reputer,
			"in topic id",
			topicId,
			" in amount",
			amount.String(),
			"tx decode error",
			err,
		)
		return false
	}

	data.counts.incrementUndelegateStakeCount()
	iterSuccessLog(
		m.T,
		iteration,
		"delegator ",
		delegator,
		" unstaked from reputer ",
		reputer,
		" in topic id ",
		topicId,
		" in amount ",
		amount.String(),
	)
	// if the delegator will have no stake left after this undelegation, remove them from the list of delegators
	chainAmount := getDelegatorStakeFromChain(m, delegator, reputer, topicId, data.failOnErr, iteration)
	if chainAmount.Equal(*amount) {
		data.removeDelegatorDelegated(topicId, delegator, reputer)
	}
	return true
}

func cancelDelegateStakeRemoval(
	m *testcommon.TestConfig,
	delegator Actor,
	reputer Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) (success bool) {
	iterLog(
		m.T,
		iteration,
		"cancelling stake removal as a delegator",
		delegator,
		"on reputer",
		reputer,
		"in topic id",
		topicId,
	)
	msg := emissionstypes.CancelRemoveDelegateStakeRequest{
		Sender:    delegator.addr,
		TopicId:   topicId,
		Delegator: delegator.addr,
		Reputer:   reputer.addr,
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, delegator.acc, &msg)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"cancelling stake removal as a delegator failed",
			delegator,
			"reputer",
			reputer,
			"in topic id",
			topicId,
			"tx broadcast error",
			err,
		)
		return false
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"cancelling stake removal as a delegator failed",
			delegator,
			"reputer",
			reputer,
			"in topic id",
			topicId,
			"tx wait error",
			err,
		)
		return false
	}

	response := &emissionstypes.CancelRemoveDelegateStakeResponse{}
	err = txResp.Decode(response)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"cancelling stake removal as a delegator failed",
			delegator,
			"reputer",
			reputer,
			"in topic id",
			topicId,
			"tx decode error",
			err,
		)
		return false
	}

	data.counts.incrementCancelDelegateStakeRemovalCount()
	iterSuccessLog(
		m.T,
		iteration,
		"cancelled stake removal as a delegator",
		delegator,
		"in topic id",
		topicId,
	)
	// make sure this delegator is still in the list of delegators
	data.addDelegatorDelegated(topicId, delegator, reputer)
	return true
}

func collectDelegatorRewards(
	m *testcommon.TestConfig,
	delegator Actor,
	reputer Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) (success bool) {
	iterLog(
		m.T,
		iteration,
		"delegator ",
		delegator,
		" collecting rewards for delegating on ",
		reputer,
		" in topic id ",
		topicId,
	)
	msg := emissionstypes.RewardDelegateStakeRequest{
		Sender:  delegator.addr,
		TopicId: topicId,
		Reputer: reputer.addr,
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, delegator.acc, &msg)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"delegator ",
			delegator,
			" failed to collect rewards in topic id ",
			topicId,
			"tx broadcast error",
			err,
		)
		return false
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"delegator ",
			delegator,
			" failed to collect rewards in topic id ",
			topicId,
			"tx wait error",
			err,
		)
		return false
	}

	response := &emissionstypes.RewardDelegateStakeResponse{}
	err = txResp.Decode(response)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"delegator ",
			delegator,
			" failed to collect rewards in topic id ",
			topicId,
			"tx decode error",
			err,
		)
		return false
	}

	data.counts.incrementCollectDelegatorRewardsCount()
	iterSuccessLog(
		m.T,
		iteration,
		"delegator ",
		delegator,
		" collected rewards in topic id ",
		topicId,
	)
	return true
}
