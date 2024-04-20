package integration_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

func getAlloraStakingModuleAccountBalance(m TestMetadata) cosmosMath.Int {
	alloraStakingModuleAccResponse, err := m.n.QueryAuth.ModuleAccountByName(
		m.ctx,
		&authtypes.QueryModuleAccountByNameRequest{
			Name: emissionstypes.AlloraStakingAccountName,
		},
	)
	require.NoError(m.t, err)
	var alloraStakingModuleAcc authtypes.ModuleAccount
	err = m.n.Cdc.Unmarshal(
		alloraStakingModuleAccResponse.Account.Value,
		&alloraStakingModuleAcc,
	)
	require.NoError(m.t, err)

	alloraStakingModuleAccBalance, err := m.n.QueryBank.Balance(
		m.ctx,
		&banktypes.QueryBalanceRequest{
			Address: alloraStakingModuleAcc.Address,
			Denom:   params.BaseCoinUnit,
		},
	)
	require.NoError(m.t, err)
	return alloraStakingModuleAccBalance.Balance.Amount
}

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
	registerAliceResponse := &emissionstypes.MsgRegisterResponse{}
	err = txResp.Decode(registerAliceResponse)
	require.NoError(m.t, err)
	require.True(m.t, registerAliceResponse.Success)
	require.Equal(m.t, "Node successfully registered", registerAliceResponse.Message)
	aliceRegistered, err := m.n.QueryEmissions.GetRegisteredTopicIds(
		m.ctx,
		&emissionstypes.QueryRegisteredTopicIdsRequest{
			Address:   m.n.AliceAddr,
			IsReputer: true,
		},
	)
	require.NoError(m.t, err)
	require.Contains(m.t, aliceRegistered.TopicIds, uint64(1))
}

// register bob as worker in topic 1, then check sucess
func RegisterBobAsWorkerTopic1(m TestMetadata) {
	registerBobRequest := &emissionstypes.MsgRegister{
		Sender:       m.n.BobAddr,
		Owner:        m.n.BobAddr,
		LibP2PKey:    "workerkey",
		MultiAddress: "workermultiaddress",
		TopicId:      uint64(1),
		IsReputer:    false,
	}
	txResp, err := m.n.Client.BroadcastTx(m.ctx, m.n.BobAcc, registerBobRequest)
	require.NoError(m.t, err)
	registerBobResponse := &emissionstypes.MsgRegisterResponse{}
	err = txResp.Decode(registerBobResponse)
	require.NoError(m.t, err)
	require.True(m.t, registerBobResponse.Success)
	require.Equal(m.t, "Node successfully registered", registerBobResponse.Message)
	bobRegistered, err := m.n.QueryEmissions.GetRegisteredTopicIds(
		m.ctx,
		&emissionstypes.QueryRegisteredTopicIdsRequest{
			Address:   m.n.BobAddr,
			IsReputer: false,
		},
	)
	require.NoError(m.t, err)
	require.Contains(m.t, bobRegistered.TopicIds, uint64(1))
}

// Register two actors and check their registrations went through
func RegistrationChecks(m TestMetadata) {
	m.t.Log("--- Registering Alice as Reputer ---")
	RegisterAliceAsReputerTopic1(m)
	m.t.Log("--- Registering Bob as Worker ---")
	RegisterBobAsWorkerTopic1(m)
}
