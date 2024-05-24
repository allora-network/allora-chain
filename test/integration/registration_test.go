package integration_test

import (
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

// register alice as a reputer in topic 1, then check success
func RegisterAliceAsReputerTopic1(m TestMetadata) {
	registerAliceRequest := &emissionstypes.MsgRegister{
		Sender:       m.n.AliceAddr,
		Owner:        m.n.AliceAddr,
		LibP2PKey:    "reputerkey",
		MultiAddress: "reputermultiaddress",
		TopicId:      1,
		IsReputer:    true,
	}
	txResp, err := m.n.Client.BroadcastTx(m.ctx, m.n.AliceAcc, registerAliceRequest)
	require.NoError(m.t, err)
	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	require.NoError(m.t, err)
	registerAliceResponse := &emissionstypes.MsgRegisterResponse{}
	err = txResp.Decode(registerAliceResponse)
	require.NoError(m.t, err)
	require.True(m.t, registerAliceResponse.Success)
	require.Equal(m.t, "Node successfully registered", registerAliceResponse.Message)

	// Check Alice registered as reputer
	aliceRegistered, err := m.n.Client.QueryEmissions().IsReputerRegisteredInTopicId(
		m.ctx,
		&emissionstypes.QueryIsReputerRegisteredInTopicIdRequest{
			TopicId: 1,
			Address: m.n.AliceAddr,
		},
	)
	require.NoError(m.t, err)
	require.True(m.t, aliceRegistered.IsRegistered)

	// Check Alice not registered as worker
	aliceNotRegisteredAsWorker, err := m.n.Client.QueryEmissions().IsWorkerRegisteredInTopicId(
		m.ctx,
		&emissionstypes.QueryIsWorkerRegisteredInTopicIdRequest{
			Address: m.n.AliceAddr,
		},
	)
	require.NoError(m.t, err)
	require.False(m.t, aliceNotRegisteredAsWorker.IsRegistered)
}

// register bob as worker in topic 1, then check sucess
func RegisterBobAsWorkerTopic1(m TestMetadata) {
	registerBobRequest := &emissionstypes.MsgRegister{
		Sender:       m.n.BobAddr,
		Owner:        m.n.BobAddr,
		LibP2PKey:    "workerkey",
		MultiAddress: "workermultiaddress",
		TopicId:      1,
		IsReputer:    false,
	}
	txResp, err := m.n.Client.BroadcastTx(m.ctx, m.n.BobAcc, registerBobRequest)
	require.NoError(m.t, err)
	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	require.NoError(m.t, err)
	registerBobResponse := &emissionstypes.MsgRegisterResponse{}
	err = txResp.Decode(registerBobResponse)
	require.NoError(m.t, err)
	require.True(m.t, registerBobResponse.Success)
	require.Equal(m.t, "Node successfully registered", registerBobResponse.Message)
	// Check Bob registered as worker
	bobRegistered, err := m.n.Client.QueryEmissions().IsWorkerRegisteredInTopicId(
		m.ctx,
		&emissionstypes.QueryIsWorkerRegisteredInTopicIdRequest{
			TopicId: 1,
			Address: m.n.BobAddr,
		},
	)
	require.NoError(m.t, err)
	require.True(m.t, bobRegistered.IsRegistered)

	// Check Bob not registered as reputer
	bobNotRegisteredAsWorker, err := m.n.Client.QueryEmissions().IsReputerRegisteredInTopicId(
		m.ctx,
		&emissionstypes.QueryIsReputerRegisteredInTopicIdRequest{
			Address: m.n.BobAddr,
		},
	)
	require.NoError(m.t, err)
	require.False(m.t, bobNotRegisteredAsWorker.IsRegistered)
}

// Register two actors and check their registrations went through
func RegistrationChecks(m TestMetadata) {
	m.t.Log("--- Registering Alice as Reputer ---")
	RegisterAliceAsReputerTopic1(m)
	m.t.Log("--- Registering Bob as Worker ---")
	RegisterBobAsWorkerTopic1(m)
}
