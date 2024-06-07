package simulation

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

const topicFunds int64 = 1e6
const EpochLength = 5

// test that we can create topics and that the resultant topics are what we asked for
func createTopic(m testCommon.TestConfig) (topicId uint64) {
	topicIdStart, err := m.Client.QueryEmissions().GetNextTopicId(
		m.Ctx,
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
		EpochLength:     EpochLength,
		GroundTruthLag:  20,
		DefaultArg:      "ETH",
		PNorm:           alloraMath.NewDecFromInt64(3),
		AlphaRegret:     alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:   true,
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

	require.NoError(m.T, err)
	return topicId
}

// broadcast a tx to fund a topic
func fundTopic(
	m testCommon.TestConfig,
	topicId uint64,
	amount int64,
) error {
	txResp, err := m.Client.BroadcastTx(
		m.Ctx,
		m.AliceAcc,
		&emissionstypes.MsgFundTopic{
			Sender:  m.AliceAddr,
			TopicId: topicId,
			Amount:  cosmosMath.NewInt(amount),
		},
	)
	if err != nil {
		return err
	}
	_, err = m.Client.WaitForTx(m.Ctx, txResp.TxHash)
	if err != nil {
		return err
	}
	resp := &emissionstypes.MsgFundTopicResponse{}
	err = txResp.Decode(resp)
	if err != nil {
		return err
	}
	return nil
}

func SetupTopic(m testCommon.TestConfig) uint64 {
	topicId := createTopic(m)
	m.T.Log("created Topic", topicId, "from", m.AliceAddr)
	_ = fundTopic(m, topicId, topicFunds)
	m.T.Log("funded topic with ", topicFunds, "from", m.AliceAddr)
	return topicId
}
