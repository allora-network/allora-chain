package integration_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

// test that we can create topics and that the resultant topics are what we asked for
func CreateTopic(m testCommon.TestConfig) (topicId uint64) {
	topicIdStart, err := m.Client.QueryEmissions().GetNextTopicId(
		m.Ctx,
		&emissionstypes.QueryNextTopicIdRequest{},
	)
	require.NoError(m.T, err)
	require.Greater(m.T, topicIdStart.NextTopicId, uint64(0))
	require.NoError(m.T, err)
	createTopicRequest := &emissionstypes.MsgCreateNewTopic{
		Creator:          m.AliceAddr,
		Metadata:         "ETH 24h Prediction",
		LossLogic:        "bafybeid7mmrv5qr4w5un6c64a6kt2y4vce2vylsmfvnjt7z2wodngknway",
		LossMethod:       "loss-calculation-eth.wasm",
		InferenceLogic:   "bafybeigx43n7kho3gslauwtsenaxehki6ndjo3s63ahif3yc5pltno3pyq",
		InferenceMethod:  "allora-inference-function.wasm",
		EpochLength:      5,
		GroundTruthLag:   20,
		DefaultArg:       "ETH",
		Pnorm:            2,
		AlphaRegret:      alloraMath.MustNewDecFromString("3.14"),
		PrewardReputer:   alloraMath.MustNewDecFromString("6.2"),
		PrewardInference: alloraMath.MustNewDecFromString("7.3"),
		PrewardForecast:  alloraMath.MustNewDecFromString("8.4"),
		FTolerance:       alloraMath.MustNewDecFromString("5.5"),
		AllowNegative:    true,
	}
	txResp, err := m.Client.BroadcastTx(m.Ctx, m.AliceAcc, createTopicRequest)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(m.Ctx, txResp.TxHash)
	require.NoError(m.T, err)
	createTopicResponse := &emissionstypes.MsgCreateNewTopicResponse{}
	err = txResp.Decode(createTopicResponse)
	require.NoError(m.T, err)
	topicId = createTopicResponse.TopicId
	require.Equal(m.T, topicIdStart.NextTopicId, topicId)
	topicIdEnd, err := m.Client.QueryEmissions().GetNextTopicId(
		m.Ctx,
		&emissionstypes.QueryNextTopicIdRequest{},
	)
	require.NoError(m.T, err)
	require.Equal(m.T, topicIdEnd.NextTopicId, topicId+1)

	storedTopicResponse, err := m.Client.QueryEmissions().GetTopic(
		m.Ctx,
		&emissionstypes.QueryTopicRequest{
			TopicId: topicId,
		},
	)
	require.NoError(m.T, err)
	storedTopic := storedTopicResponse.Topic
	require.Equal(m.T, createTopicRequest.Metadata, storedTopic.Metadata)
	require.Equal(m.T, createTopicRequest.LossLogic, storedTopic.LossLogic)
	require.Equal(m.T, createTopicRequest.LossMethod, storedTopic.LossMethod)
	require.Equal(m.T, createTopicRequest.InferenceLogic, storedTopic.InferenceLogic)
	require.Equal(m.T, createTopicRequest.InferenceMethod, storedTopic.InferenceMethod)
	require.Equal(m.T, createTopicRequest.EpochLength, storedTopic.EpochLength)
	require.Equal(m.T, createTopicRequest.GroundTruthLag, storedTopic.GroundTruthLag)
	require.Equal(m.T, createTopicRequest.DefaultArg, storedTopic.DefaultArg)
	require.Equal(m.T, createTopicRequest.Pnorm, storedTopic.Pnorm)
	require.True(m.T, createTopicRequest.AlphaRegret.Equal(storedTopic.AlphaRegret), "Alpha Regret not equal %s != %s", createTopicRequest.AlphaRegret, storedTopic.AlphaRegret)
	require.True(m.T, createTopicRequest.PrewardReputer.Equal(storedTopic.PrewardReputer), "Preward Reputer not equal %s != %s", createTopicRequest.PrewardReputer, storedTopic.PrewardReputer)
	require.True(m.T, createTopicRequest.PrewardInference.Equal(storedTopic.PrewardInference), "Preward Inference not equal %s != %s", createTopicRequest.PrewardInference, storedTopic.PrewardInference)
	require.True(m.T, createTopicRequest.PrewardForecast.Equal(storedTopic.PrewardForecast), "Preward Forecast not equal %s != %s", createTopicRequest.PrewardForecast, storedTopic.PrewardForecast)
	require.True(m.T, createTopicRequest.FTolerance.Equal(storedTopic.FTolerance), "FTolerance not equal %s != %s", createTopicRequest.FTolerance, storedTopic.FTolerance)

	return topicId
}
