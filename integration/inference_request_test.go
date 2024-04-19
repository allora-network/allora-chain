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
			Requests: []*emissionstypes.RequestInferenceListItem{
				{
					Nonce:                1,
					TopicId:              1,
					Cadence:              10800,
					MaxPricePerInference: cosmosMath.NewUint(10000),
					BidAmount:            cosmosMath.NewUint(10000),
					BlockValidUntil:      currBlock + 10805,
				},
			},
		},
	)
	require.NoError(m.t, err)
	resp := &emissionstypes.MsgRequestInferenceResponse{}
	err = txResp.Decode(resp)
	require.NoError(m.t, err)

	// todo make msgRequestInferenceResponse return the id of the resultant request
	// and then query for it that way, esp given wanting to delete that endpoint

	// query for the request
	allInferenceRequestsResponse, err := m.n.QueryEmissions.GetAllExistingInferenceRequests(
		m.ctx,
		&emissionstypes.QueryAllExistingInferenceRequest{},
	)
	require.NoError(m.t, err)
	require.Greater(m.t, len(allInferenceRequestsResponse.InferenceRequests), 0)
}

func ReactivateTopic1(m TestMetadata) {
	txResp, err := m.n.Client.BroadcastTx(
		m.ctx,
		m.n.AliceAcc,
		&emissionstypes.MsgReactivateTopic{
			Sender:  m.n.AliceAddr,
			TopicId: 1,
		},
	)
	require.NoError(m.t, err)
	reactivateTopicResponse := &emissionstypes.MsgReactivateTopicResponse{}
	err = txResp.Decode(reactivateTopicResponse)
	require.NoError(m.t, err)
	require.True(m.t, reactivateTopicResponse.Success)
}

func InferenceRequestsChecks(m TestMetadata) {
	m.t.Log("--- Check creating an Inference Request on Topic 1 ---")
	CreateInferenceRequestOnTopic1(m)
	m.t.Log("--- Check reactivating Topic 1 ---")
	ReactivateTopic1(m)
}
