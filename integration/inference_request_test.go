package integration_test

import (
	cosmosMath "cosmossdk.io/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

func CreateInferenceRequestOnTopic1(m TestMetadata) {
	currBlock, err := m.n.Client.LatestBlockHeight(m.ctx)
	require.NoError(m.t, err)
	txResp, err := m.n.Client.BroadcastTx(
		m.ctx,
		m.n.BobAcc,
		&emissionstypes.MsgRequestInference{
			Sender: m.n.BobAddr,
			Request: &emissionstypes.InferenceRequestInbound{
				Nonce:                1,
				TopicId:              1,
				Cadence:              10800,
				MaxPricePerInference: cosmosMath.NewUint(10000),
				BidAmount:            cosmosMath.NewUint(10000),
				BlockValidUntil:      currBlock + 10805,
			},
		},
	)
	require.NoError(m.t, err)
	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	require.NoError(m.t, err)
	resp := &emissionstypes.MsgRequestInferenceResponse{}
	err = txResp.Decode(resp)
	require.NoError(m.t, err)

	// todo make msgRequestInferenceResponse return the id of the resultant request
	// and then query for it that way, esp given wanting to delete that endpoint

	// query for the request
	_, err = m.n.QueryEmissions.GetMempoolInferenceRequest(
		m.ctx,
		&emissionstypes.QueryMempoolInferenceRequest{RequestId: resp.RequestId},
	)
	require.NoError(m.t, err)
}

func CheckTopic1Activated(m TestMetadata) {

	// Fetch only active topics
	pagi := &emissionstypes.QueryActiveTopicsRequest{
		Pagination: &emissionstypes.SimpleCursorPaginationRequest{
			Limit: 10,
		},
	}
	activeTopics, err := m.n.QueryEmissions.GetActiveTopics(
		m.ctx,
		pagi)
	require.NoError(m.t, err, "Fetching active topics should not produce an error")

	// Verify the correct number of active topics is retrieved
	// s.Require().Len(activeTopics, 2, "Should retrieve exactly two active topics")
	require.Equal(m.t, len(activeTopics.Topics), 1, "Should retrieve exactly one active topics")
}

func InferenceRequestsChecks(m TestMetadata) {
	m.t.Log("--- Check creating an Inference Request on Topic 1 ---")
	CreateInferenceRequestOnTopic1(m)
	m.t.Log("--- Check reactivating Topic 1 ---")
	CheckTopic1Activated(m)
}
