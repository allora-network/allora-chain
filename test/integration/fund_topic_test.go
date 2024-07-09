package integration_test

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

func FundTopic1(m testCommon.TestConfig) {
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(
		ctx,
		m.BobAcc,
		&emissionstypes.MsgFundTopic{
			Sender:  m.BobAddr,
			TopicId: uint64(1),
			Amount:  cosmosMath.NewInt(10000),
		},
	)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
	resp := &emissionstypes.MsgFundTopicResponse{}
	err = txResp.Decode(resp)
	require.NoError(m.T, err)
}

func CheckTopic1Activated(m testCommon.TestConfig) {
	ctx := context.Background()
	// Fetch only active topics
	pagi := &emissionstypes.QueryActiveTopicsRequest{
		Pagination: &emissionstypes.SimpleCursorPaginationRequest{
			Limit: 10,
		},
	}
	activeTopics, err := m.Client.QueryEmissions().GetActiveTopics(
		ctx,
		pagi)
	require.NoError(m.T, err, "Fetching active topics should not produce an error")

	// Verify the correct number of active topics is retrieved
	require.Equal(m.T, len(activeTopics.Topics), 1, "Should retrieve exactly one active topics")
}

// Must come after a reputer is registered and staked in topic 1
func TopicFundingChecks(m testCommon.TestConfig) {
	m.T.Log("--- Check funding Topic 1 ---")
	FundTopic1(m)
	m.T.Log("--- Check reactivating Topic 1 ---")
	CheckTopic1Activated(m) // Should have stake (from earlier test) AND funds by now
}
