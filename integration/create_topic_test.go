package integration_test

import (
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

// test that we can create topics and that the resultant topics are what we asked for
func CreateTopic(m TestMetadata) (topicId uint64) {
	topicIdStart, err := m.n.QueryEmissions.GetNextTopicId(
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
	topicId = createTopicResponse.TopicId
	require.Equal(m.t, topicIdStart.NextTopicId, topicId)
	topicIdEnd, err := m.n.QueryEmissions.GetNextTopicId(
		m.ctx,
		&emissionstypes.QueryNextTopicIdRequest{},
	)
	require.NoError(m.t, err)
	require.Equal(m.t, topicIdEnd.NextTopicId, topicId+1)

	storedTopicResponse, err := m.n.QueryEmissions.GetTopic(
		m.ctx,
		&emissionstypes.QueryTopicRequest{
			TopicId: topicId,
		},
	)
	require.NoError(m.t, err)
	storedTopic := storedTopicResponse.Topic
	require.Equal(m.t, createTopicRequest.Metadata, storedTopic.Metadata)
	require.Equal(m.t, createTopicRequest.LossLogic, storedTopic.LossLogic)
	require.Equal(m.t, createTopicRequest.LossMethod, storedTopic.LossMethod)
	require.Equal(m.t, createTopicRequest.InferenceLogic, storedTopic.InferenceLogic)
	require.Equal(m.t, createTopicRequest.InferenceMethod, storedTopic.InferenceMethod)
	require.Equal(m.t, createTopicRequest.EpochLength, storedTopic.EpochLength)
	require.Equal(m.t, createTopicRequest.GroundTruthLag, storedTopic.GroundTruthLag)
	require.Equal(m.t, createTopicRequest.DefaultArg, storedTopic.DefaultArg)
	require.Equal(m.t, createTopicRequest.Pnorm, storedTopic.Pnorm)
	require.True(m.t, createTopicRequest.AlphaRegret.Equal(storedTopic.AlphaRegret), "Alpha Regret not equal %s != %s", createTopicRequest.AlphaRegret, storedTopic.AlphaRegret)
	require.True(m.t, createTopicRequest.PrewardReputer.Equal(storedTopic.PrewardReputer), "Preward Reputer not equal %s != %s", createTopicRequest.PrewardReputer, storedTopic.PrewardReputer)
	require.True(m.t, createTopicRequest.PrewardInference.Equal(storedTopic.PrewardInference), "Preward Inference not equal %s != %s", createTopicRequest.PrewardInference, storedTopic.PrewardInference)
	require.True(m.t, createTopicRequest.PrewardForecast.Equal(storedTopic.PrewardForecast), "Preward Forecast not equal %s != %s", createTopicRequest.PrewardForecast, storedTopic.PrewardForecast)
	require.True(m.t, createTopicRequest.FTolerance.Equal(storedTopic.FTolerance), "FTolerance not equal %s != %s", createTopicRequest.FTolerance, storedTopic.FTolerance)

	return topicId
}
