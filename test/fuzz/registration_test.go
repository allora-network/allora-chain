package fuzz_test

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
) (success bool) {
	iterLog(m.T, iteration, "registering ", actor, "as worker in topic id", topicId)
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, &emissionstypes.RegisterRequest{
		Sender:    actor.addr,
		Owner:     actor.addr, // todo pick random other actor
		IsReputer: false,
		TopicId:   topicId,
	})
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, "failed to register ", actor, "as worker in topic id ", topicId, "tx broadcast error", err)
		return false
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, "failed to register ", actor, "as worker in topic id ", topicId, "tx wait error", err)
		return false
	}

	registerWorkerResponse := &emissionstypes.RegisterResponse{} // nolint:exhaustruct // the fields are populated by decode
	err = txResp.Decode(registerWorkerResponse)
	failIfOnErr(m.T, data.failOnErr, err)
	if data.failOnErr {
		require.True(m.T, registerWorkerResponse.Success)
	}
	if err != nil {
		iterFailLog(m.T, iteration, "failed to register ", actor, "as worker in topic id ", topicId, "tx decode error", err)
		return false
	}

	data.addWorkerRegistration(topicId, actor)
	data.counts.incrementRegisterWorkerCount()
	iterSuccessLog(m.T, iteration, "registered ", actor, "as worker in topic id ", topicId)
	return true
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
) (success bool) {
	iterLog(m.T, iteration, "unregistering ", actor, "as worker in topic id", topicId)
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, &emissionstypes.RemoveRegistrationRequest{
		Sender:    actor.addr,
		TopicId:   topicId,
		IsReputer: false,
	})
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"failed to unregister ", actor, "as worker in topic id ", topicId, "tx broadcast error", err)
		return false
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"failed to unregister ", actor, "as worker in topic id ", topicId, "tx wait error", err)
		return false
	}

	removeRegistrationResponse := &emissionstypes.RemoveRegistrationResponse{} // nolint:exhaustruct // the fields are populated by decode
	err = txResp.Decode(removeRegistrationResponse)
	failIfOnErr(m.T, data.failOnErr, err)
	if data.failOnErr {
		require.True(m.T, removeRegistrationResponse.Success)
	}
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"failed to unregister ", actor, "as worker in topic id ", topicId, "tx decode error", err)
		return false
	}

	data.removeWorkerRegistration(topicId, actor)
	data.counts.incrementUnregisterWorkerCount()
	iterSuccessLog(m.T, iteration, "unregistered ", actor, "as worker in topic id ", topicId)
	return true
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
) (success bool) {
	iterLog(m.T, iteration, "registering ", actor, "as reputer in topic id", topicId)
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, &emissionstypes.RegisterRequest{
		Sender:    actor.addr,
		Owner:     actor.addr, // todo pick random other actor
		IsReputer: true,
		TopicId:   topicId,
	})
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"failed to register ", actor, "as reputer in topic id ", topicId, "tx broadcast error", err)
		return false
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"failed to register ", actor, "as reputer in topic id ", topicId, "tx wait error", err)
		return false
	}

	registerWorkerResponse := &emissionstypes.RegisterResponse{} // nolint:exhaustruct // the fields are populated by decode
	err = txResp.Decode(registerWorkerResponse)
	failIfOnErr(m.T, data.failOnErr, err)
	if data.failOnErr {
		require.True(m.T, registerWorkerResponse.Success)
	}
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"failed to register ", actor, "as reputer in topic id ", topicId, "tx decode error", err)
		return false
	}

	data.addReputerRegistration(topicId, actor)
	data.counts.incrementRegisterReputerCount()
	iterSuccessLog(m.T, iteration, "registered ", actor, "as reputer in topic id ", topicId)
	return true
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
) (success bool) {
	iterLog(m.T, iteration, "unregistering ", actor, "as reputer in topic id", topicId)
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, &emissionstypes.RemoveRegistrationRequest{
		Sender:    actor.addr,
		TopicId:   topicId,
		IsReputer: true,
	})
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"failed to unregister ", actor, "as reputer in topic id ", topicId, "tx broadcast error", err)
		return false
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"failed to unregister ", actor, "as reputer in topic id ", topicId, "tx wait error", err)
		return false
	}

	removeRegistrationResponseMsg := &emissionstypes.RemoveRegistrationResponse{} // nolint:exhaustruct // the fields are populated by decode
	err = txResp.Decode(removeRegistrationResponseMsg)
	failIfOnErr(m.T, data.failOnErr, err)
	if data.failOnErr {
		require.True(m.T, removeRegistrationResponseMsg.Success)
	}
	if err != nil {
		iterFailLog(
			m.T,
			iteration,
			"failed to unregister ", actor, "as reputer in topic id ", topicId, "tx decode error", err)
		return false
	}

	data.removeReputerRegistration(topicId, actor)
	data.counts.incrementUnregisterReputerCount()
	iterSuccessLog(m.T, iteration, "unregistered ", actor, "as reputer in topic id ", topicId)
	return true
}
