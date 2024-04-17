package integration_test

import (
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

func GetParams(m TestMetadata) {
	paramsReq := &emissionstypes.QueryParamsRequest{}
	p, err := m.n.QueryClient.Params(
		m.ctx,
		paramsReq,
	)
	require.NoError(m.t, err)
	require.NotNil(m.t, p)
}

func CreateTopic(m TestMetadata) (topicId uint64) {
	topicIdStart, err := m.n.QueryClient.GetNextTopicId(
		m.ctx,
		&emissionstypes.QueryNextTopicIdRequest{},
	)
	require.NoError(m.t, err)
	require.Greater(m.t, topicIdStart.NextTopicId, uint64(0))
	aliceAddr, err := m.n.AliceAcc.Address(params.HumanCoinUnit)
	require.NoError(m.t, err)
	createTopicRequest := &emissionstypes.MsgCreateNewTopic{
		Creator:          aliceAddr,
		Metadata:         "ETH 24h Prediction",
		LossLogic:        "bafybeiazhgps7ywkhouwj6m6a7bkq36w3g734kx4b5iqql4n52zf3jjdxa",
		LossMethod:       "loss-calculation-eth.wasm",
		InferenceLogic:   "bafybeigpiwl3o73zvvl6dxdqu7zqcub5mhg65jiky2xqb4rdhfmikswzqm",
		InferenceMethod:  "allora-inference-function.wasm",
		EpochLength:      10800,
		GroundTruthLag:   60,
		DefaultArg:       "ETH",
		Pnorm:            2,
		AlphaRegret:      alloraMath.MustNewDecFromString("3.14"),
		PrewardReputer:   alloraMath.MustNewDecFromString("6.2"),
		PrewardInference: alloraMath.MustNewDecFromString("7.3"),
		PrewardForecast:  alloraMath.MustNewDecFromString("8.4"),
		FTolerance:       alloraMath.MustNewDecFromString("5.5"),
	}
	txResp, err := m.n.Client.BroadcastTx(m.ctx, m.n.AliceAcc, createTopicRequest)
	require.NoError(m.t, err)
	createTopicResponse := &emissionstypes.MsgCreateNewTopicResponse{}
	err = txResp.Decode(createTopicResponse)
	require.NoError(m.t, err)
	require.Equal(m.t, topicIdStart.NextTopicId, createTopicResponse.TopicId)
	topicIdEnd, err := m.n.QueryClient.GetNextTopicId(
		m.ctx,
		&emissionstypes.QueryNextTopicIdRequest{},
	)
	require.NoError(m.t, err)
	require.Equal(m.t, topicIdEnd.NextTopicId, createTopicResponse.TopicId+1)
	return createTopicResponse.TopicId
}
