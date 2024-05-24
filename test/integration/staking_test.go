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

// func CheckTopic1Activated(m testCommon.TestConfig) {
// 	// Fetch only active topics
// 	pagi := &emissionstypes.QueryActiveTopicsRequest{
// 		Pagination: &emissionstypes.SimpleCursorPaginationRequest{
// 			Limit: 10,
// 		},
// 	}
// 	activeTopics, err := m.Client.QueryEmissions().GetActiveTopics(
// 		m.Ctx,
// 		pagi)
// 	require.NoError(m.t, err, "Fetching active topics should not produce an error")

// 	// Verify the correct number of active topics is retrieved
// 	require.Equal(m.t, len(activeTopics.Topics), 1, "Should retrieve exactly one active topic")
// }

// Register two actors and check their registrations went through
func StakingChecks(m testCommon.TestConfig) {
	m.T.Log("--- Staking Alice as Reputer ---")
	StakeAliceAsReputerTopic1(m)
	m.T.Log("--- Check reactivating Topic 1 ---")
	CheckTopic1Activated(m)
}
