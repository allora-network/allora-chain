package integration_test

import (
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

// register alice as a reputer in topic 1, then check success
func RegisterAliceAsReputerTopic1(m testCommon.TestConfig) {
	registerAliceRequest := &emissionstypes.MsgRegister{
		Sender:       m.AliceAddr,
		Owner:        m.AliceAddr,
		LibP2PKey:    "reputerkey",
		MultiAddress: "reputermultiaddress",
		TopicId:      1,
		IsReputer:    true,
	}
	txResp, err := m.Client.BroadcastTx(m.Ctx, m.AliceAcc, registerAliceRequest)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(m.Ctx, txResp.TxHash)
	require.NoError(m.T, err)
	registerAliceResponse := &emissionstypes.MsgRegisterResponse{}
	err = txResp.Decode(registerAliceResponse)
	require.NoError(m.T, err)
	require.True(m.T, registerAliceResponse.Success)
	require.Equal(m.T, "Node successfully registered", registerAliceResponse.Message)

	// Check Alice registered as reputer
	aliceRegistered, err := m.Client.QueryEmissions().IsReputerRegisteredInTopicId(
		m.Ctx,
		&emissionstypes.QueryIsReputerRegisteredInTopicIdRequest{
			TopicId: 1,
			Address: m.AliceAddr,
		},
	)
	require.NoError(m.T, err)
	require.True(m.T, aliceRegistered.IsRegistered)

	// Check Alice not registered as worker
	aliceNotRegisteredAsWorker, err := m.Client.QueryEmissions().IsWorkerRegisteredInTopicId(
		m.Ctx,
		&emissionstypes.QueryIsWorkerRegisteredInTopicIdRequest{
			TopicId: 1,
			Address: m.AliceAddr,
		},
	)
	require.NoError(m.T, err)
	require.False(m.T, aliceNotRegisteredAsWorker.IsRegistered)
}

// register bob as worker in topic 1, then check sucess
func RegisterBobAsWorkerTopic1(m testCommon.TestConfig) {
	registerBobRequest := &emissionstypes.MsgRegister{
		Sender:       m.BobAddr,
		Owner:        m.BobAddr,
		LibP2PKey:    "workerkey",
		MultiAddress: "workermultiaddress",
		TopicId:      1,
		IsReputer:    false,
	}
	txResp, err := m.Client.BroadcastTx(m.Ctx, m.BobAcc, registerBobRequest)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(m.Ctx, txResp.TxHash)
	require.NoError(m.T, err)
	registerBobResponse := &emissionstypes.MsgRegisterResponse{}
	err = txResp.Decode(registerBobResponse)
	require.NoError(m.T, err)
	require.True(m.T, registerBobResponse.Success)
	require.Equal(m.T, "Node successfully registered", registerBobResponse.Message)
	// Check Bob registered as worker
	bobRegistered, err := m.Client.QueryEmissions().IsWorkerRegisteredInTopicId(
		m.Ctx,
		&emissionstypes.QueryIsWorkerRegisteredInTopicIdRequest{
			TopicId: 1,
			Address: m.BobAddr,
		},
	)
	require.NoError(m.T, err)
	require.True(m.T, bobRegistered.IsRegistered)

	// Check Bob not registered as reputer
	bobNotRegisteredAsWorker, err := m.Client.QueryEmissions().IsReputerRegisteredInTopicId(
		m.Ctx,
		&emissionstypes.QueryIsReputerRegisteredInTopicIdRequest{
			TopicId: 1,
			Address: m.BobAddr,
		},
	)
	require.NoError(m.T, err)
	require.False(m.T, bobNotRegisteredAsWorker.IsRegistered)
}

// Register two actors and check their registrations went through
func RegistrationChecks(m testCommon.TestConfig) {
	m.T.Log("--- Registering Alice as Reputer ---")
	RegisterAliceAsReputerTopic1(m)
	m.T.Log("--- Registering Bob as Worker ---")
	RegisterBobAsWorkerTopic1(m)
}
