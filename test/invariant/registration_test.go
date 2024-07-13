package invariant_test

import (
	"context"

	cosmossdk_io_math "cosmossdk.io/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

// determine if this state transition is worth trying based on our knowledge of the state
func anyWorkersRegistered(data *SimulationData) bool {
	return data.registeredWorkers.Len() > 0
}

// determine if this state transition is worth trying based on our knowledge of the state
func anyReputersRegistered(data *SimulationData) bool {
	return data.registeredReputers.Len() > 0
}

// register actor as a new worker in topicId
func registerWorker(
	m *testcommon.TestConfig,
	actor Actor,
	_ Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) {
	wasErr := false
	iterLog(m.T, iteration, "registering ", actor, "as worker in topic id", topicId)
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, &emissionstypes.MsgRegister{
		Sender:       actor.addr,
		Owner:        actor.addr, // todo pick random other actor
		LibP2PKey:    getLibP2pKeyName(actor),
		MultiAddress: getMultiAddressName(actor),
		IsReputer:    false,
		TopicId:      topicId,
	})
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		iterFailLog(m.T, iteration, "failed to register ", actor, "as worker in topic id ", topicId)
		return
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	registerWorkerResponse := &emissionstypes.MsgRegisterResponse{}
	err = txResp.Decode(registerWorkerResponse)
	requireNoError(m.T, data.failOnErr, err)
	if data.failOnErr {
		require.True(m.T, registerWorkerResponse.Success)
	}
	wasErr = orErr(wasErr, err)

	if !wasErr {
		data.addWorkerRegistration(topicId, actor)
		data.counts.incrementRegisterWorkerCount()
		iterSuccessLog(m.T, iteration, "registered ", actor, "as worker in topic id ", topicId)
	} else {
		iterFailLog(m.T, iteration, "failed to register ", actor, "as worker in topic id ", topicId)
	}
}

// unregister actor from being a worker in topic topicId
func unregisterWorker(
	m *testcommon.TestConfig,
	actor Actor,
	_ Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) {
	wasErr := false
	iterLog(m.T, iteration, "unregistering ", actor, "as worker in topic id", topicId)
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, &emissionstypes.MsgRemoveRegistration{
		Sender:    actor.addr,
		TopicId:   topicId,
		IsReputer: false,
	})
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		iterFailLog(m.T, iteration, "failed to unregister ", actor, "as worker in topic id ", topicId)
		return
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	removeRegistrationResponse := &emissionstypes.MsgRemoveRegistrationResponse{}
	err = txResp.Decode(removeRegistrationResponse)
	requireNoError(m.T, data.failOnErr, err)
	if data.failOnErr {
		require.True(m.T, removeRegistrationResponse.Success)
	}
	wasErr = orErr(wasErr, err)

	if !wasErr {
		data.removeWorkerRegistration(topicId, actor)
		data.counts.incrementUnregisterWorkerCount()
		iterSuccessLog(m.T, iteration, "unregistered ", actor, "as worker in topic id ", topicId)
	} else {
		iterFailLog(m.T, iteration, "failed to unregister ", actor, "as worker in topic id ", topicId)
	}
}

// register actor as a new reputer in topicId
func registerReputer(
	m *testcommon.TestConfig,
	actor Actor,
	_ Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) {
	wasErr := false
	iterLog(m.T, iteration, "registering ", actor, "as reputer in topic id", topicId)
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, &emissionstypes.MsgRegister{
		Sender:       actor.addr,
		Owner:        actor.addr, // todo pick random other actor
		LibP2PKey:    getLibP2pKeyName(actor),
		MultiAddress: getMultiAddressName(actor),
		IsReputer:    true,
		TopicId:      topicId,
	})
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		iterFailLog(m.T, iteration, "failed to register ", actor, "as reputer in topic id ", topicId)
		return
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	registerWorkerResponse := &emissionstypes.MsgRegisterResponse{}
	err = txResp.Decode(registerWorkerResponse)
	requireNoError(m.T, data.failOnErr, err)
	if data.failOnErr {
		require.True(m.T, registerWorkerResponse.Success)
	}
	wasErr = orErr(wasErr, err)

	if !wasErr {
		data.addReputerRegistration(topicId, actor)
		data.counts.incrementRegisterReputerCount()
		iterSuccessLog(m.T, iteration, "registered ", actor, "as reputer in topic id ", topicId)
	} else {
		iterFailLog(m.T, iteration, "failed to register ", actor, "as reputer in topic id ", topicId)
	}
}

// unregister actor as a reputer in topicId
func unregisterReputer(
	m *testcommon.TestConfig,
	actor Actor,
	_ Actor,
	_ *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) {
	wasErr := false
	iterLog(m.T, iteration, "unregistering ", actor, "as reputer in topic id", topicId)
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, &emissionstypes.MsgRemoveRegistration{
		Sender:    actor.addr,
		TopicId:   topicId,
		IsReputer: true,
	})
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		iterFailLog(m.T, iteration, "failed to unregister ", actor, "as reputer in topic id ", topicId)
		return
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	removeRegistrationResponseMsg := &emissionstypes.MsgRemoveRegistrationResponse{}
	err = txResp.Decode(removeRegistrationResponseMsg)
	requireNoError(m.T, data.failOnErr, err)
	if data.failOnErr {
		require.True(m.T, removeRegistrationResponseMsg.Success)
	}
	wasErr = orErr(wasErr, err)

	if !wasErr {
		data.removeReputerRegistration(topicId, actor)
		data.counts.incrementUnregisterReputerCount()
		iterSuccessLog(m.T, iteration, "unregistered ", actor, "as reputer in topic id ", topicId)
	} else {
		iterFailLog(m.T, iteration, "failed to unregister ", actor, "as reputer in topic id ", topicId)
	}
}
