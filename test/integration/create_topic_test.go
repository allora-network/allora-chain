package integration_test

import (
	"context"

	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

// test that we can create topics and that the resultant topics are what we asked for
func CreateTopic(m testCommon.TestConfig) (topicId uint64) {
	ctx := context.Background()
	topicIdStart, err := m.Client.QueryEmissions().GetNextTopicId(
		ctx,
		&emissionstypes.QueryNextTopicIdRequest{},
	)
	require.NoError(m.T, err)
	require.Greater(m.T, topicIdStart.NextTopicId, uint64(0))
	require.NoError(m.T, err)
	createTopicRequest := &emissionstypes.MsgCreateNewTopic{
		Creator:         m.AliceAddr,
		Metadata:        "ETH 24h Prediction",
		LossLogic:       "bafybeid7mmrv5qr4w5un6c64a6kt2y4vce2vylsmfvnjt7z2wodngknway",
		LossMethod:      "loss-calculation-eth.wasm",
		InferenceLogic:  "bafybeigx43n7kho3gslauwtsenaxehki6ndjo3s63ahif3yc5pltno3pyq",
		InferenceMethod: "allora-inference-function.wasm",
		EpochLength:     5,
		GroundTruthLag:  20,
		DefaultArg:      "ETH",
		PNorm:           alloraMath.NewDecFromInt64(3),
		AlphaRegret:     alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:   true,
	}
	txResp, err := m.Client.BroadcastTx(ctx, m.AliceAcc, createTopicRequest)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
	createTopicResponse := &emissionstypes.MsgCreateNewTopicResponse{}
	err = txResp.Decode(createTopicResponse)
	require.NoError(m.T, err)
	topicId = createTopicResponse.TopicId
	require.Equal(m.T, topicIdStart.NextTopicId, topicId)
	topicIdEnd, err := m.Client.QueryEmissions().GetNextTopicId(
		ctx,
		&emissionstypes.QueryNextTopicIdRequest{},
	)
	require.NoError(m.T, err)
	require.Equal(m.T, topicIdEnd.NextTopicId, topicId+1)

	storedTopicResponse, err := m.Client.QueryEmissions().GetTopic(
		ctx,
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
	require.Equal(m.T, createTopicRequest.PNorm, storedTopic.PNorm)
	require.True(m.T, createTopicRequest.AlphaRegret.Equal(storedTopic.AlphaRegret), "Alpha Regret not equal %s != %s", createTopicRequest.AlphaRegret, storedTopic.AlphaRegret)

	return topicId
}
