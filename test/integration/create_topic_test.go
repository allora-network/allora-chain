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
	require.Positive(m.T, topicIdStart.NextTopicId)
	require.NoError(m.T, err)
	createTopicRequest := &emissionstypes.MsgCreateNewTopic{
		Creator:                  m.AliceAddr,
		Metadata:                 "ETH 24h Prediction",
		LossMethod:               "mse",
		EpochLength:              5,
		GroundTruthLag:           10,
		WorkerSubmissionWindow:   4,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
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
	require.Equal(m.T, createTopicRequest.LossMethod, storedTopic.LossMethod)
	require.Equal(m.T, createTopicRequest.EpochLength, storedTopic.EpochLength)
	require.Equal(m.T, createTopicRequest.GroundTruthLag, storedTopic.GroundTruthLag)
	require.Equal(m.T, createTopicRequest.WorkerSubmissionWindow, storedTopic.WorkerSubmissionWindow)
	require.Equal(m.T, createTopicRequest.PNorm, storedTopic.PNorm)
	require.True(m.T, createTopicRequest.AlphaRegret.Equal(storedTopic.AlphaRegret), "Alpha Regret not equal %s != %s", createTopicRequest.AlphaRegret, storedTopic.AlphaRegret)

	return topicId
}
