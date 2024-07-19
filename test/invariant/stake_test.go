package invariant_test

import (
	"context"

	cosmossdk_io_math "cosmossdk.io/math"

	testcommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// stake actor as a reputer, pick a random amount to stake that is less than half their current balance
func stakeAsReputer(
	m *testcommon.TestConfig,
	actor Actor,
	_ Actor,
	amount *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) {
	wasErr := false
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
	msg := emissionstypes.MsgAddStake{
		Sender:  actor.addr,
		TopicId: topicId,
		Amount:  *amount,
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, &msg)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		iterFailLog(m.T, iteration, "stake failed", actor, "as a reputer in topic id ", topicId, " in amount ", amount.String())
		return
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	response := &emissionstypes.MsgAddStakeResponse{}
	err = txResp.Decode(response)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	if !wasErr {
		data.addReputerStake(topicId, actor, *amount)
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
	} else {
		iterFailLog(m.T, iteration, "stake failed", actor, "as a reputer in topic id ", topicId, " in amount ", amount.String())
	}
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
) {
	wasErr := false
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
	msg := emissionstypes.MsgRemoveStake{
		Sender:  actor.addr,
		TopicId: topicId,
		Amount:  *amount,
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, &msg)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		iterFailLog(m.T, iteration, "unstake failed", actor, "as a reputer in topic id ", topicId, " in amount ", amount.String())
		return
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	response := &emissionstypes.MsgRemoveStakeResponse{}
	err = txResp.Decode(response)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	if !wasErr {
		data.markStakeRemovalReputerStake(topicId, actor, amount)
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
	} else {
		iterFailLog(m.T, iteration, "unstake failed", actor, "as a reputer in topic id ", topicId, " in amount ", amount.String())
	}
}

// ask the chain if any stake removals exist
func findFirstValidStakeRemovalFromChain(m *testcommon.TestConfig) (emissionstypes.StakeRemovalInfo, bool, error) {
	ctx := context.Background()
	blockHeightNow, err := m.Client.BlockHeight(ctx)
	if err != nil {
		return emissionstypes.StakeRemovalInfo{}, false, err
	}
	moduleParams, err := m.Client.QueryEmissions().Params(ctx, &emissionstypes.QueryParamsRequest{})
	if err != nil {
		return emissionstypes.StakeRemovalInfo{}, false, err
	}
	blockHeightEnd := blockHeightNow + moduleParams.Params.RemoveStakeDelayWindow
	for i := blockHeightNow; i < blockHeightEnd; i++ {
		query := &emissionstypes.QueryStakeRemovalsForBlockRequest{
			BlockHeight: i,
		}
		resp, err := m.Client.QueryEmissions().GetStakeRemovalsForBlock(ctx, query)
		if err != nil || resp == nil {
			continue
		}
		if len(resp.Removals) == 0 {
			continue
		}
		if resp.Removals[0] == nil {
			continue
		}
		return *resp.Removals[0], true, nil
	}
	return emissionstypes.StakeRemovalInfo{}, false, nil
}

func cancelStakeRemoval(
	m *testcommon.TestConfig,
	actor Actor,
	_ Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) {
	wasErr := true
	iterLog(
		m.T,
		iteration,
		"cancelling stake removal as a reputer",
		actor,
		"in topic id",
		topicId,
	)
	msg := emissionstypes.MsgCancelRemoveStake{
		Sender:  actor.addr,
		TopicId: topicId,
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, &msg)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		iterFailLog(m.T, iteration, "cancelling stake removal as a reputer failed", actor, "in topic id", topicId)
		return
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	response := &emissionstypes.MsgCancelRemoveStakeResponse{}
	err = txResp.Decode(response)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	if !wasErr {
		data.counts.incrementCancelStakeRemovalCount()
	} else {
		iterFailLog(m.T, iteration, "cancelling stake removal as a reputer failed", actor, "in topic id", topicId)
	}
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
) {
	wasErr := false
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
	msg := emissionstypes.MsgDelegateStake{
		Sender:  delegator.addr,
		Reputer: reputer.addr,
		TopicId: topicId,
		Amount:  *amount,
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, delegator.acc, &msg)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		iterFailLog(m.T, iteration, "delegation failed", delegator, "upon reputer", reputer, "in topic id", topicId, " in amount", amount.String())
		return
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	registerWorkerResponse := &emissionstypes.MsgDelegateStakeResponse{}
	err = txResp.Decode(registerWorkerResponse)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	if !wasErr {
		data.addDelegatorStake(topicId, delegator, reputer, *amount)
		data.counts.incrementDelegateStakeCount()
		iterSuccessLog(
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
	} else {
		iterFailLog(m.T, iteration, "delegation failed", delegator, "upon reputer", reputer, "in topic id", topicId, " in amount", amount.String())
	}
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
) {
	wasErr := false
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
	msg := emissionstypes.MsgRemoveDelegateStake{
		Sender:  delegator.addr,
		Reputer: reputer.addr,
		TopicId: topicId,
		Amount:  *amount,
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, delegator.acc, &msg)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		iterFailLog(m.T, iteration, "undelegation failed", delegator, "from reputer", reputer, "in topic id", topicId, " in amount", amount.String())
		return
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	response := &emissionstypes.MsgRemoveDelegateStakeResponse{}
	err = txResp.Decode(response)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	if !wasErr {
		data.markStakeRemovalDelegatorStake(topicId, delegator, reputer, amount)
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
	} else {
		iterFailLog(m.T, iteration, "undelegation failed", delegator, "from reputer", reputer, "in topic id", topicId, " in amount", amount.String())
	}
}

// ask the chain if any stake removals exist
func findFirstValidDelegateStakeRemovalFromChain(m *testcommon.TestConfig) (emissionstypes.DelegateStakeRemovalInfo, bool, error) {
	ctx := context.Background()
	blockHeightNow, err := m.Client.BlockHeight(ctx)
	if err != nil {
		return emissionstypes.DelegateStakeRemovalInfo{}, false, err
	}
	moduleParams, err := m.Client.QueryEmissions().Params(ctx, &emissionstypes.QueryParamsRequest{})
	if err != nil {
		return emissionstypes.DelegateStakeRemovalInfo{}, false, err
	}
	blockHeightEnd := blockHeightNow + moduleParams.Params.RemoveStakeDelayWindow
	for i := blockHeightNow; i < blockHeightEnd; i++ {
		query := &emissionstypes.QueryDelegateStakeRemovalsForBlockRequest{
			BlockHeight: i,
		}
		resp, err := m.Client.QueryEmissions().GetDelegateStakeRemovalsForBlock(ctx, query)
		if err != nil || resp == nil {
			continue
		}
		if len(resp.Removals) == 0 {
			continue
		}
		if resp.Removals[0] == nil {
			continue
		}
		return *resp.Removals[0], true, nil
	}
	return emissionstypes.DelegateStakeRemovalInfo{}, false, nil
}

func cancelDelegateStakeRemoval(
	m *testcommon.TestConfig,
	delegator Actor,
	reputer Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) {
	wasErr := true
	iterLog(
		m.T,
		iteration,
		"cancelling stake removal as a delegator",
		delegator,
		"in topic id",
		topicId,
	)
	msg := emissionstypes.MsgCancelRemoveDelegateStake{
		Sender:    delegator.addr,
		TopicId:   topicId,
		Delegator: delegator.addr,
		Reputer:   reputer.addr,
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, delegator.acc, &msg)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		iterFailLog(m.T, iteration, "cancelling stake removal as a delegator failed delegator ", delegator, " reputer ", reputer, "in topic id", topicId)
		return
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	response := &emissionstypes.MsgCancelRemoveDelegateStakeResponse{}
	err = txResp.Decode(response)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if !wasErr {
		data.counts.incrementCancelDelegateStakeRemovalCount()
	} else {
		iterFailLog(m.T, iteration, "cancelling stake removal as a delegator failed delegator ", delegator, " reputer", reputer, "in topic id", topicId)
	}
}

func collectDelegatorRewards(
	m *testcommon.TestConfig,
	delegator Actor,
	reputer Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) {
	wasErr := false
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
	msg := emissionstypes.MsgRewardDelegateStake{
		Sender:  delegator.addr,
		TopicId: topicId,
		Reputer: reputer.addr,
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, delegator.acc, &msg)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		iterFailLog(m.T, iteration, "delegator ", delegator, " failed to collect rewards in topic id ", topicId)
		return
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	response := &emissionstypes.MsgRewardDelegateStakeResponse{}
	err = txResp.Decode(response)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	if !wasErr {
		data.counts.incrementCollectDelegatorRewardsCount()
		iterSuccessLog(
			m.T,
			iteration,
			"delegator ",
			delegator,
			" collected rewards in topic id ",
			topicId,
		)
	} else {
		iterFailLog(m.T, iteration, "delegator ", delegator, " failed to collect rewards in topic id ", topicId)
	}
}
