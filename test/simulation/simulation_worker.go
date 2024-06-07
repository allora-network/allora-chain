package simulation

import (
	"encoding/hex"
	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"
	"math/rand"
	"strings"
)

const RetryTime = 3

// Inserts bulk inference and forecast data for a worker
func insertWorkerBulk(
	m testCommon.TestConfig,
	topic *emissionstypes.Topic,
	inferers []testCommon.AccountAndAddress,
	forecasters []testCommon.AccountAndAddress,
) (int64, error) {
	workers := append(inferers, forecasters...)
	leaderIndex := rand.Intn(len(workers))
	leaderWorker := workers[leaderIndex]
	var blockHeightCurrent int64 = 0
	for index := 0; index < RetryTime; index++ {
		blockHeightCurrent = topic.EpochLastEnded + topic.EpochLength
		// Get Bundles
		workerDataBundles := make([]*emissionstypes.WorkerDataBundle, 0)
		for _, inferer := range inferers {
			forecasterIndex := rand.Intn(len(forecasters))
			workerDataBundles = append(workerDataBundles,
				generateSingleWorkerBundle(m, topic.Id, blockHeightCurrent, inferer.Addr, forecasters[forecasterIndex].Addr, inferers, leaderWorker.Acc.Name))
		}

		nonce := emissionstypes.Nonce{BlockHeight: blockHeightCurrent}
		// Create a MsgInsertBulkReputerPayload message
		workerMsg := &emissionstypes.MsgInsertBulkWorkerPayload{
			Sender:            leaderWorker.Addr,
			Nonce:             &nonce,
			TopicId:           topic.Id,
			WorkerDataBundles: workerDataBundles,
		}
		txResp, err := m.Client.BroadcastTx(m.Ctx, leaderWorker.Acc, workerMsg)
		if err != nil {
			if strings.Contains(err.Error(), "nonce already fulfilled") ||
				strings.Contains(err.Error(), "nonce still unfulfilled") {
				topic, err = getTopic(m, topic.Id)
				if err == nil {
					continue
				}
			}
			m.T.Log("Error broadcasting worker bulk: ", err)
			return 0, err
		}
		_, err = m.Client.WaitForTx(m.Ctx, txResp.TxHash)
		if err != nil {
			m.T.Log("Error waiting for worker bulk: ", err)
			return 0, err
		}
		m.T.Log("Inserted Worker Bulk", blockHeightCurrent)
		break
	}
	return blockHeightCurrent, nil
}

func generateSingleWorkerBundle(
	m testCommon.TestConfig,
	topicId uint64,
	blockHeight int64,
	workerAddress string,
	forecasterAddress string,
	inferers []testCommon.AccountAndAddress,
	signerName string,
) *emissionstypes.WorkerDataBundle {
	// Iterate workerAddresses to get the worker address, and generate as many forecasts as there are workers
	forecastElements := make([]*emissionstypes.ForecastElement, 0)
	for _, inferer := range inferers {
		forecastElements = append(forecastElements, &emissionstypes.ForecastElement{
			Inferer: inferer.Addr,
			Value:   alloraMath.NewDecFromInt64(int64(rand.Intn(51) + 50)),
		})
	}
	infererValue := alloraMath.NewDecFromInt64(int64(rand.Intn(300) + 3000))

	// Create a MsgInsertBulkReputerPayload message
	workerDataBundle := &emissionstypes.WorkerDataBundle{
		Worker: workerAddress,
		InferenceForecastsBundle: &emissionstypes.InferenceForecastBundle{
			Inference: &emissionstypes.Inference{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     workerAddress,
				Value:       infererValue,
			},
			Forecast: &emissionstypes.Forecast{
				TopicId:          topicId,
				BlockHeight:      blockHeight,
				Forecaster:       forecasterAddress,
				ForecastElements: forecastElements,
			},
		},
	}

	// Sign
	src := make([]byte, 0)
	src, err := workerDataBundle.InferenceForecastsBundle.XXX_Marshal(src, true)
	require.NoError(m.T, err, "Marshall reputer value bundle should not return an error")

	sig, pubKey, err := m.Client.Context().Keyring.Sign(signerName, src, signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(m.T, err, "Sign should not return an error")
	workerPublicKeyBytes := pubKey.Bytes()
	workerDataBundle.InferencesForecastsBundleSignature = sig
	workerDataBundle.Pubkey = hex.EncodeToString(workerPublicKeyBytes)

	return workerDataBundle
}
