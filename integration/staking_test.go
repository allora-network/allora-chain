package integration_test

import (
	"fmt"

	cosmosMath "cosmossdk.io/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

// register alice as a reputer in topic 1, then check success
func StakeAliceAsReputerTopic1(m TestMetadata) {
	// Record Alice stake before adding more
	aliceStakedBefore, err := m.n.QueryEmissions.GetReputerStakeInTopic(
		m.ctx,
		&emissionstypes.QueryReputerStakeInTopicRequest{
			TopicId: 1,
			Address: m.n.AliceAddr,
		},
	)
	require.NoError(m.t, err)

	const stakeToAdd = 1000000

	// Have Alice stake more
	addStake := &emissionstypes.MsgAddStake{
		Sender:  m.n.AliceAddr,
		TopicId: 1,
		Amount:  cosmosMath.NewUint(stakeToAdd),
	}
	txResp, err := m.n.Client.BroadcastTx(m.ctx, m.n.AliceAcc, addStake)
	require.NoError(m.t, err)
	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	require.NoError(m.t, err)

	// Check Alice has stake on the topic
	aliceStakedAfter, err := m.n.QueryEmissions.GetReputerStakeInTopic(
		m.ctx,
		&emissionstypes.QueryReputerStakeInTopicRequest{
			TopicId: 1,
			Address: m.n.AliceAddr,
		},
	)
	require.NoError(m.t, err)
	require.Equal(m.t, fmt.Sprint(stakeToAdd), aliceStakedAfter.Amount.Sub(aliceStakedBefore.Amount).String())
}

// Assumes Alice already registered as reputer in topic 1 and has `>=stakeToAdd` funds
func StakingChecks(m TestMetadata) {
	m.t.Log("--- Staking Alice as Reputer ---")
	StakeAliceAsReputerTopic1(m)
}
