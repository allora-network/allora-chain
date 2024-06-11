package simulation

import (
	"encoding/hex"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"
)

const RetryTime = 3

func generateForeacstInitValue(inferenceVal alloraMath.Dec) alloraMath.Dec {
	rand.Seed(time.Now().UnixNano())
	forecast_context := 1 / (1 + math.Exp(-1*rand.Float64()))
	forecast_bias := rand.Float64() * forecast_context * 0.3
	forecast_error := (-0.222 + rand.Float64()*0.398) * forecast_context * 0.3
	start := math.Min(forecast_bias, forecast_error)
	end := math.Max(forecast_bias, forecast_error)
	randVal := start + rand.Float64()*(end-start)
	mul := alloraMath.MustNewDecFromString(strconv.FormatFloat(randVal, 'f', -1, 64))
	initVal, _ := inferenceVal.Quo(mul)
	return initVal
}

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
		for index, inferer := range inferers {
			forecasterIndex := index % len(forecasters)
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
	infererValue := alloraMath.NewDecFromInt64(int64(rand.Intn(300) + 3000))
	forecastElements := make([]*emissionstypes.ForecastElement, 0)
	for _, inferer := range inferers {
		forecastElements = append(forecastElements, &emissionstypes.ForecastElement{
			Inferer: inferer.Addr,
			Value:   generateForeacstInitValue(infererValue),
		})
	}

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
