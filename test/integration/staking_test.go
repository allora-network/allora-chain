package integration_test

import (
	"fmt"

	cosmosMath "cosmossdk.io/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

// register alice as a reputer in topic 1, then check success
func StakeAliceAsReputerTopic1(m testCommon.TestConfig) {
	// Record Alice stake before adding more
	aliceStakedBefore, err := m.Client.QueryEmissions().GetReputerStakeInTopic(
		m.Ctx,
		&emissionstypes.QueryReputerStakeInTopicRequest{
			TopicId: 1,
			Address: m.AliceAddr,
		},
	)
	require.NoError(m.T, err)

	const stakeToAdd = 1000000

	// Have Alice stake more
	addStake := &emissionstypes.MsgAddStake{
		Sender:  m.AliceAddr,
		TopicId: 1,
		Amount:  cosmosMath.NewInt(stakeToAdd),
	}
	txResp, err := m.Client.BroadcastTx(m.Ctx, m.AliceAcc, addStake)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(m.Ctx, txResp.TxHash)
	require.NoError(m.T, err)

	// Check Alice has stake on the topic
	aliceStakedAfter, err := m.Client.QueryEmissions().GetReputerStakeInTopic(
		m.Ctx,
		&emissionstypes.QueryReputerStakeInTopicRequest{
			TopicId: 1,
			Address: m.AliceAddr,
		},
	)
	require.NoError(m.T, err)
	require.Equal(m.T, fmt.Sprint(stakeToAdd), aliceStakedAfter.Amount.Sub(aliceStakedBefore.Amount).String())
}

// Register two actors and check their registrations went through
func StakingChecks(m testCommon.TestConfig) {
	m.T.Log("--- Staking Alice as Reputer ---")
	StakeAliceAsReputerTopic1(m)

	res, _ := m.Client.QueryEmissions().GetTopic(m.Ctx, &emissionstypes.QueryTopicRequest{
		TopicId: uint64(1),
	})
	// Topic is not expected to be funded yet => expect 0 weight => topic not active!
	// But we still have this conditional just in case there are > 0 funds
	if res.EffectiveRevenue != "0" {
		m.T.Log("--- Check reactivating Topic 1 ---")
		CheckTopic1Activated(m)
	}
}
