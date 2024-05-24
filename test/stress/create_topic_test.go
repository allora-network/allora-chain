package stress_test

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/stretchr/testify/require"
)

func CreateTopic(
	m TestMetadata,
	epochLength int64,
	creatorAddress string,
	creatorAccount cosmosaccount.Account,
) (topicId uint64) {
	createTopicRequest := &emissionstypes.MsgCreateNewTopic{
		Creator:          creatorAddress,
		Metadata:         "ETH 24h Prediction",
		LossLogic:        "bafybeid7mmrv5qr4w5un6c64a6kt2y4vce2vylsmfvnjt7z2wodngknway",
		LossMethod:       "loss-calculation-eth.wasm",
		InferenceLogic:   "bafybeigx43n7kho3gslauwtsenaxehki6ndjo3s63ahif3yc5pltno3pyq",
		InferenceMethod:  "allora-inference-function.wasm",
		EpochLength:      epochLength,
		GroundTruthLag:   0,
		DefaultArg:       "ETH",
		Pnorm:            2,
		AlphaRegret:      alloraMath.MustNewDecFromString("3.14"),
		PrewardReputer:   alloraMath.MustNewDecFromString("6.2"),
		PrewardInference: alloraMath.MustNewDecFromString("7.3"),
		PrewardForecast:  alloraMath.MustNewDecFromString("8.4"),
		FTolerance:       alloraMath.MustNewDecFromString("5.5"),
		AllowNegative:    true,
	}

	txResp, err := m.n.Client.BroadcastTx(m.ctx, creatorAccount, createTopicRequest)
	require.NoError(m.t, err)

	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	require.NoError(m.t, err)

	createTopicResponse := &emissionstypes.MsgCreateNewTopicResponse{}
	err = txResp.Decode(createTopicResponse)
	require.NoError(m.t, err)

	incrementCountTopics()

	return createTopicResponse.TopicId
}
