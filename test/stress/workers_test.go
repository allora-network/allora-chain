package stress_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"
)

// creates the worker addresses in the account registry
func createWorkerAddresses(
	m testCommon.TestConfig,
	topicId uint64,
	workersMax int,
) (workers NameToAccountMap) {
	workers = make(map[string]AccountAndAddress)

	for workerIndex := 0; workerIndex < workersMax; workerIndex++ {
		workerAccountName := getWorkerAccountName(m.Seed, workerIndex, topicId)
		workerAccount, _, err := m.Client.AccountRegistryCreate(workerAccountName)
		if err != nil {
			m.T.Log(topicLog(topicId, "Error creating funder address: ", workerAccountName, " - ", err))
			continue
		}
		workerAddressToFund, err := workerAccount.Address(params.HumanCoinUnit)
		if err != nil {
			m.T.Log(topicLog(topicId, "Error creating funder address: ", workerAccountName, " - ", err))
			continue
		}
		workers[workerAccountName] = AccountAndAddress{
			acc:  workerAccount,
			addr: workerAddressToFund,
		}
	}
	return workers
}

// register all the created workers for this iteration
func registerWorkersForIteration(
	m testCommon.TestConfig,
	topicId uint64,
	iteration int,
	workersPerIteration int,
	countWorkers int,
	maxWorkersPerTopic int,
	workers NameToAccountMap,
	makeReport bool,
) int {
	for j := 0; j < workersPerIteration && countWorkers < maxWorkersPerTopic; j++ {
		workerIndex := iteration*workersPerIteration + j
		workerName := getWorkerAccountName(m.Seed, workerIndex, topicId)
		worker := workers[workerName]
		err := RegisterWorkerForTopic(
			m,
			NameAccountAndAddress{
				name: workerName,
				aa:   worker,
			},
			topicId,
		)
		if err != nil {
			m.T.Log(topicLog(topicId, "Error registering worker address: ", worker.addr, " - ", err))
			if makeReport {
				saveWorkerError(topicId, workerName, err)
				saveTopicError(topicId, err)
			}
			return countWorkers
		}
		countWorkers++
	}
	return countWorkers
}

// pick a worker to upload a bundle, then try to insert the bundle
// if the bundle nonce is already fulfilled, realign the blockHeights and retry
// up to retry times
func generateInsertWorkerBundle(
	m testCommon.TestConfig,
	topic *emissionstypes.Topic,
	workers NameToAccountMap,
	retryTimes int,
	makeReport bool,
) (insertedBlockHeight int64, err error) {
	leaderWorkerAccountName, err := pickRandomKeyFromMap(workers)
	if err != nil {
		m.T.Log(topicLog(topic.Id, "Error getting random worker address: ", err))
		if makeReport {
			saveReputerError(topic.Id, leaderWorkerAccountName, err)
			saveTopicError(topic.Id, err)
		}
		return 0, err
	}

	blockHeightCurrent := topic.EpochLastEnded + topic.EpochLength

	startWorker := time.Now()
	for i := 0; i < retryTimes; i++ {
		err = insertWorkerBulk(m, topic, leaderWorkerAccountName, workers, blockHeightCurrent)
		if err != nil {
			if strings.Contains(err.Error(), "nonce already fulfilled") ||
				strings.Contains(err.Error(), "nonce still unfulfilled") {
				// realign blockHeights before retrying
				ctx := context.Background()
				topic, err = getLastTopic(ctx, m.Client.QueryEmissions(), topic.Id)
				if err == nil {
					blockHeightCurrent = topic.EpochLastEnded + topic.EpochLength
					m.T.Log(topicLog(topic.Id, "Reset ", leaderWorkerAccountName, "blockHeight to (", blockHeightCurrent, ")"))
				} else {
					m.T.Log(topicLog(topic.Id, "Error getting topic!"))
					if makeReport {
						saveWorkerError(topic.Id, leaderWorkerAccountName, err)
						saveTopicError(topic.Id, err)
					}
					return blockHeightCurrent, err
				}
			} else {
				m.T.Log(topicLog(topic.Id, "Error inserting worker bulk: ", err))
				if makeReport {
					saveWorkerError(topic.Id, leaderWorkerAccountName, err)
					saveTopicError(topic.Id, err)
				}
				return blockHeightCurrent, err
			}
		} else {
			m.T.Log(topicLog(topic.Id, "Inserted worker bulk, blockHeight: ", blockHeightCurrent, " with ", len(workers), " workers"))
			elapsedBulk := time.Since(startWorker)
			m.T.Log(topicLog(topic.Id, "Insert Worker ", leaderWorkerAccountName, " ", blockHeightCurrent, " Elapsed time:", elapsedBulk))
			return blockHeightCurrent, nil
		}
	}
	return blockHeightCurrent, err
}

// Inserts bulk inference and forecast data for a worker
func insertWorkerBulk(
	m testCommon.TestConfig,
	topic *emissionstypes.Topic,
	leaderWorkerAccountName string,
	workers map[string]AccountAndAddress,
	blockHeight int64,
) error {
	// Get Bundles
	workerDataBundles := make([]*emissionstypes.WorkerDataBundle, 0)
	for key := range workers {
		workerDataBundles = append(workerDataBundles,
			generateSingleWorkerBundle(m, topic.Id, blockHeight, key, workers))
	}
	leaderWorker := workers[leaderWorkerAccountName]
	return insertLeaderWorkerBulk(m, topic.Id, blockHeight, leaderWorkerAccountName, leaderWorker.addr, workerDataBundles)
}

// create inferences and forecasts for a worker
func generateSingleWorkerBundle(
	m testCommon.TestConfig,
	topicId uint64,
	blockHeight int64,
	workerAddressName string,
	workers map[string]AccountAndAddress,
) *emissionstypes.WorkerDataBundle {
	// Iterate workerAddresses to get the worker address, and generate as many forecasts as there are workers
	forecastElements := make([]*emissionstypes.ForecastElement, 0)
	for key := range workers {
		forecastElements = append(forecastElements, &emissionstypes.ForecastElement{
			Inferer: workers[key].addr,
			Value:   alloraMath.NewDecFromInt64(int64(rand.Intn(51) + 50)),
		})
	}
	infererAddress := workers[workerAddressName].addr
	infererValue := alloraMath.NewDecFromInt64(int64(rand.Intn(300) + 3000))

	// Create a MsgInsertBulkReputerPayload message
	workerDataBundle := &emissionstypes.WorkerDataBundle{
		Worker: infererAddress,
		InferenceForecastsBundle: &emissionstypes.InferenceForecastBundle{
			Inference: &emissionstypes.Inference{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     infererAddress,
				Value:       infererValue,
			},
			Forecast: &emissionstypes.Forecast{
				TopicId:          topicId,
				BlockHeight:      blockHeight,
				Forecaster:       infererAddress,
				ForecastElements: forecastElements,
			},
		},
	}

	// Sign
	src := make([]byte, 0)
	src, err := workerDataBundle.InferenceForecastsBundle.XXX_Marshal(src, true)
	require.NoError(m.T, err, "Marshall reputer value bundle should not return an error")

	sig, pubKey, err := m.Client.Context().Keyring.Sign(workerAddressName, src, signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(m.T, err, "Sign should not return an error")
	workerPublicKeyBytes := pubKey.Bytes()
	workerDataBundle.InferencesForecastsBundleSignature = sig
	workerDataBundle.Pubkey = hex.EncodeToString(workerPublicKeyBytes)

	return workerDataBundle
}

// Inserts worker bulk, given a topic, blockHeight, and leader worker address (which should exist in the keyring)
func insertLeaderWorkerBulk(
	m testCommon.TestConfig,
	topicId uint64,
	blockHeight int64,
	leaderWorkerAccountName, leaderWorkerAddress string,
	WorkerDataBundles []*emissionstypes.WorkerDataBundle) error {

	nonce := emissionstypes.Nonce{BlockHeight: blockHeight}

	// Create a MsgInsertBulkReputerPayload message
	workerMsg := &emissionstypes.MsgInsertBulkWorkerPayload{
		Sender:            leaderWorkerAddress,
		Nonce:             &nonce,
		TopicId:           topicId,
		WorkerDataBundles: WorkerDataBundles,
	}
	// serialize workerMsg to json and print
	LeaderAcc, err := m.Client.AccountRegistryGetByName(leaderWorkerAccountName)
	if err != nil {
		m.T.Log(topicLog(topicId, "Error getting leader worker account: ", leaderWorkerAccountName, " - ", err))
		return err
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, LeaderAcc, workerMsg)
	if err != nil {
		m.T.Log(topicLog(topicId, "Error broadcasting worker bulk: ", err))
		return err
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	if err != nil {
		m.T.Log(topicLog(topicId, "Error waiting for worker bulk: ", err))
		return err
	}
	return nil
}

// check that workers balances have risen due to rewards being paid out
func checkWorkersReceivedRewards(
	m testCommon.TestConfig,
	topicId uint64,
	workers NameToAccountMap,
	countWorkers int,
	maxIterations int,
	makeReport bool,
) (rewardedWorkersCount uint64, err error) {
	rewardedWorkersCount = 0
	err = nil
	for workerIndex := 0; workerIndex < countWorkers; workerIndex++ {
		ctx := context.Background()
		workerName := getWorkerAccountName(m.Seed, workerIndex, topicId)
		balance, err := getAccountBalance(
			ctx,
			m.Client.QueryBank(),
			workers[workerName].addr,
		)
		if err != nil {
			m.T.Log(topicLog(topicId, "Error getting worker balance for worker: ", workerName, err))
			if maxIterations > 20 && workerIndex < 10 {
				m.T.Log(topicLog(topicId, "ERROR: Worker", workerName, "has insufficient stake:", balance))
			}
			if makeReport {
				saveWorkerError(topicId, workerName, err)
				saveTopicError(topicId, err)
			}
		} else {
			if balance.Amount.LTE(cosmosMath.NewInt(initialWorkerReputerFundAmount)) {
				m.T.Log(topicLog(topicId, "Worker ", workerName, " balance is not greater than initial amount: ", balance.Amount.String()))
				if makeReport {
					saveWorkerError(topicId, workerName, fmt.Errorf("Balance Not Greater"))
					saveTopicError(topicId, fmt.Errorf("Balance Not Greater"))
				}
			} else {
				m.T.Log(topicLog(topicId, "Worker ", workerName, " balance: ", balance.Amount.String()))
				rewardedWorkersCount += 1
			}
		}
	}
	return rewardedWorkersCount, err
}
