package integration_test

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

func FundTopic(m testCommon.TestConfig) {
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(
		ctx,
		m.BobAcc,
		&emissionstypes.FundTopicRequest{
			Sender:  m.BobAddr,
			TopicId: m.TopicID,
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

func CheckTopicActivated(m testCommon.TestConfig) {
	ctx := context.Background()
	// Fetch only active topics
	topicIsActive, err := m.Client.QueryEmissions().IsTopicActive(
		ctx,
		&emissionstypes.IsTopicActiveRequest{TopicId: m.TopicID},
	)
	require.NoError(m.T, err, "Fetching active topics should not produce an error")

	// Verify the correct number of active topics is retrieved
	require.True(m.T, topicIsActive.IsActive, "Should retrieve exactly one active topics")
}

// Must come after a reputer is registered and staked in topic
func TopicFundingChecks(m testCommon.TestConfig) {
	m.T.Logf("--- Check funding Topic %d ---", m.TopicID)
	FundTopic(m)
	m.T.Logf("--- Check reactivating Topic %d ---", m.TopicID)
	CheckTopicActivated(m) // Should have stake (from earlier test) AND funds by now
}
