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
		&emissionstypes.FundTopicRequest{
			Sender:  m.BobAddr,
			TopicId: uint64(1),
			Amount:  cosmosMath.NewInt(10000),
		},
	)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
	resp := &emissionstypes.FundTopicResponse{}
	err = txResp.Decode(resp)
	require.NoError(m.T, err)
}

func CheckTopic1Activated(m testCommon.TestConfig) {
	ctx := context.Background()
	// Fetch only active topics
	topicIsActive, err := m.Client.QueryEmissions().IsTopicActive(
		ctx,
		&emissionstypes.IsTopicActiveRequest{TopicId: 1},
	)
	require.NoError(m.T, err, "Fetching active topics should not produce an error")

	// Verify the correct number of active topics is retrieved
	require.True(m.T, topicIsActive.IsActive, "Should retrieve exactly one active topics")
}

// Must come after a reputer is registered and staked in topic 1
func TopicFundingChecks(m testCommon.TestConfig) {
	m.T.Log("--- Check funding Topic 1 ---")
	FundTopic1(m)
	m.T.Log("--- Check reactivating Topic 1 ---")
	CheckTopic1Activated(m) // Should have stake (from earlier test) AND funds by now
}
