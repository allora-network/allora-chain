package stress_test

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	cosmossdk_io_math "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/stretchr/testify/require"
)

const secondsInAMonth = 2592000

const defaultEpochLength = 10      // Default epoch length in blocks if none is found yet from chain
const minWaitingNumberofEpochs = 3 // To control the number of epochs to wait before inserting the first batch
const iterationsInABatch = 1       // To control the number of epochs in each iteration of the loop (eg to manage insertions)

// This function gets the topic checking activity. After that it will sleep for a number of epoch to ensure nonces are available.
func getNonZeroTopicEpochLastRan(ctx context.Context, query types.QueryClient, topicID uint64, maxRetries int, approximateSecondsPerBlock time.Duration) (*types.Topic, error) {
	sleepingTimeBlocks := defaultEpochLength
	// Retry loop for a maximum of 5 times
	for retries := 0; retries < maxRetries; retries++ {
		topicResponse, err := query.GetTopic(ctx, &types.QueryTopicRequest{TopicId: topicID})
		if err == nil {
			storedTopic := topicResponse.Topic
			if storedTopic.EpochLastEnded != 0 {
				nBlocks := storedTopic.EpochLength * minWaitingNumberofEpochs
				sleepingTime := time.Duration(nBlocks) * approximateSecondsPerBlock
				fmt.Println(time.Now(), " Topic found, sleeping...", sleepingTime)
				time.Sleep(sleepingTime)
				fmt.Println(time.Now(), " Looking for topic: Slept.")
				return topicResponse.Topic, nil
			}
			sleepingTimeBlocks = int(storedTopic.EpochLength)
		} else {
			fmt.Println("Error getting topic, retry...", err)
		}
		// Sleep for a while before retrying
		fmt.Println("Retrying sleeping for a default epoch, retry ", retries, " for sleeping time ", sleepingTimeBlocks)
		time.Sleep(time.Duration(sleepingTimeBlocks) * approximateSecondsPerBlock * time.Second)
	}

	return nil, errors.New("topicEpochLastRan is still 0 after retrying")
}

func generateWorkerAttributedValueLosses(workerAddresses map[string]string, lowLimit, sum int) []*types.WorkerAttributedValue {
	values := make([]*types.WorkerAttributedValue, 0)
	for key := range workerAddresses {
		values = append(values, &types.WorkerAttributedValue{
			Worker: workerAddresses[key],
			Value:  alloraMath.NewDecFromInt64(int64(rand.Intn(lowLimit) + sum)),
		})
	}
	return values
}

func generateWithheldWorkerAttributedValueLosses(workerAddresses map[string]string, lowLimit, sum int) []*types.WithheldWorkerAttributedValue {
	values := make([]*types.WithheldWorkerAttributedValue, 0)
	for key := range workerAddresses {
		values = append(values, &types.WithheldWorkerAttributedValue{
			Worker: workerAddresses[key],
			Value:  alloraMath.NewDecFromInt64(int64(rand.Intn(lowLimit) + sum)),
		})
	}
	return values
}

func generateSingleWorkerBundle(m TestMetadata, topicId uint64, blockHeight int64,
	workerAddressName string, workerAddresses map[string]string) *types.WorkerDataBundle {
	// Iterate workerAddresses to get the worker address, and generate as many forecasts as there are workers
	forecastElements := make([]*types.ForecastElement, 0)
	for key := range workerAddresses {
		forecastElements = append(forecastElements, &types.ForecastElement{
			Inferer: workerAddresses[key],
			Value:   alloraMath.NewDecFromInt64(int64(rand.Intn(51) + 50)),
		})
	}
	infererAddress := workerAddresses[workerAddressName]
	infererValue := alloraMath.NewDecFromInt64(int64(rand.Intn(300) + 3000))

	// Create a MsgInsertBulkReputerPayload message
	workerDataBundle := &types.WorkerDataBundle{
		Worker: infererAddress,
		InferenceForecastsBundle: &types.InferenceForecastBundle{
			Inference: &types.Inference{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     infererAddress,
				Value:       infererValue,
			},
			Forecast: &types.Forecast{
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
	require.NoError(m.t, err, "Marshall reputer value bundle should not return an error")

	sig, pubKey, err := m.n.Client.Context().Keyring.Sign(workerAddressName, src, signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(m.t, err, "Sign should not return an error")
	workerPublicKeyBytes := pubKey.Bytes()
	workerDataBundle.InferencesForecastsBundleSignature = sig
	workerDataBundle.Pubkey = hex.EncodeToString(workerPublicKeyBytes)

	return workerDataBundle
}

// Generate the same valueBundle for a reputer
func generateValueBundle(topicId uint64, workerAddresses map[string]string, reputerNonce, workerNonce *types.Nonce) types.ValueBundle {
	return types.ValueBundle{
		TopicId:                topicId,
		CombinedValue:          alloraMath.NewDecFromInt64(100),
		InfererValues:          generateWorkerAttributedValueLosses(workerAddresses, 3000, 3500),
		ForecasterValues:       generateWorkerAttributedValueLosses(workerAddresses, 50, 50),
		NaiveValue:             alloraMath.NewDecFromInt64(100),
		OneOutInfererValues:    generateWithheldWorkerAttributedValueLosses(workerAddresses, 50, 50),
		OneOutForecasterValues: generateWithheldWorkerAttributedValueLosses(workerAddresses, 50, 50),
		OneInForecasterValues:  generateWorkerAttributedValueLosses(workerAddresses, 50, 50),
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
	}
}

// Inserts worker bulk, given a topic, blockHeight, and leader worker address (which should exist in the keyring)
func InsertLeaderWorkerBulk(
	m TestMetadata,
	topicId uint64,
	blockHeight int64,
	leaderWorkerAccountName, leaderWorkerAddress string,
	WorkerDataBundles []*types.WorkerDataBundle) error {

	nonce := types.Nonce{BlockHeight: blockHeight}

	// Create a MsgInsertBulkReputerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:            leaderWorkerAddress,
		Nonce:             &nonce,
		TopicId:           topicId,
		WorkerDataBundles: WorkerDataBundles,
	}
	// serialize workerMsg to json and print
	LeaderAcc, err := m.n.Client.AccountRegistry.GetByName(leaderWorkerAccountName)
	if err != nil {
		fmt.Println("Error getting leader worker account: ", leaderWorkerAccountName, " - ", err)
		return err
	}
	txResp, err := m.n.Client.BroadcastTx(m.ctx, LeaderAcc, workerMsg)
	if err != nil {
		fmt.Println("Error broadcasting worker bulk: ", err)
		return err
	}
	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	if err != nil {
		fmt.Println("Error waiting for worker bulk: ", err)
		return err
	}
	return nil
}

// Worker Bob inserts bulk inference and forecast
func InsertWorkerBulk(m TestMetadata, topic *types.Topic, leaderWorkerAccountName string, workerAddresses map[string]string, blockHeight int64) error {
	// Get Bundles
	workerDataBundles := make([]*types.WorkerDataBundle, 0)
	for key := range workerAddresses {
		workerDataBundles = append(workerDataBundles, generateSingleWorkerBundle(m, topic.Id, blockHeight, key, workerAddresses))
	}
	leaderWorkerAddress := workerAddresses[leaderWorkerAccountName]
	return InsertLeaderWorkerBulk(m, topic.Id, blockHeight, leaderWorkerAccountName, leaderWorkerAddress, workerDataBundles)
}

// Generate a ReputerValueBundle
func generateSingleReputerValueBundle(
	m TestMetadata,
	reputerAddressName, reputerAddress string,
	valueBundle types.ValueBundle) *types.ReputerValueBundle {

	valueBundle.Reputer = reputerAddress
	// Sign
	src := make([]byte, 0)
	src, err := valueBundle.XXX_Marshal(src, true)
	require.NoError(m.t, err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature, pubKey, err := m.n.Client.Context().Keyring.Sign(reputerAddressName, src, signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(m.t, err, "Sign should not return an error")
	reputerPublicKeyBytes := pubKey.Bytes()

	// Create a MsgInsertBulkReputerPayload message
	reputerValueBundle := &types.ReputerValueBundle{
		ValueBundle: &valueBundle,
		Signature:   valueBundleSignature,
		Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
	}

	return reputerValueBundle
}

func generateReputerValueBundleMsg(
	topicId uint64,
	reputerValueBundles []*types.ReputerValueBundle,
	leaderReputerAddress string,
	reputerNonce, workerNonce *types.Nonce) *types.MsgInsertBulkReputerPayload {

	return &types.MsgInsertBulkReputerPayload{
		Sender:  leaderReputerAddress,
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
		ReputerValueBundles: reputerValueBundles,
	}
}

func InsertReputerBulk(m TestMetadata,
	topic *types.Topic,
	leaderReputerAccountName string,
	reputerAddresses, workerAddresses map[string]string,
	BlockHeightCurrent, BlockHeightEval int64) error {
	// Nonce: calculate from EpochLastRan + EpochLength
	topicId := topic.Id
	// Nonces are last two blockHeights
	reputerNonce := &types.Nonce{
		BlockHeight: BlockHeightCurrent,
	}
	workerNonce := &types.Nonce{
		BlockHeight: BlockHeightEval,
	}
	leaderReputerAddress := reputerAddresses[leaderReputerAccountName]
	valueBundle := generateValueBundle(topicId, workerAddresses, reputerNonce, workerNonce)
	reputerValueBundles := make([]*types.ReputerValueBundle, 0)
	for reputerAddressName := range reputerAddresses {
		reputerAddress := reputerAddresses[reputerAddressName]
		reputerValueBundle := generateSingleReputerValueBundle(m, reputerAddressName, reputerAddress, valueBundle)
		reputerValueBundles = append(reputerValueBundles, reputerValueBundle)
	}

	reputerValueBundleMsg := generateReputerValueBundleMsg(topicId, reputerValueBundles, leaderReputerAddress, reputerNonce, workerNonce)
	LeaderAcc, err := m.n.Client.AccountRegistry.GetByName(leaderReputerAccountName)
	if err != nil {
		fmt.Println("Error getting leader worker account: ", leaderReputerAccountName, " - ", err)
		return err
	}
	txResp, err := m.n.Client.BroadcastTx(m.ctx, LeaderAcc, reputerValueBundleMsg)
	if err != nil {
		fmt.Println("Error broadcasting reputer value bundle: ", err)
		return err
	}
	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	require.NoError(m.t, err)
	return nil
}

func lookupEnvInt(m TestMetadata, key string, defaultValue int) int {
	value, found := os.LookupEnv(key)
	if !found {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		m.t.Fatal("Error converting string to int: ", err)
	}
	return intValue
}

const stakeToAdd uint64 = 90000
const topicFunds int64 = 1000000
const epochLength int64 = 5

func SetupTopic(m TestMetadata, topicFunderAddress string, topicFunderAccount cosmosaccount.Account) uint64 {

	m.t.Log("Creating new Topic")

	topicId := CreateTopic(m, epochLength, topicFunderAddress, topicFunderAccount)

	err := FundTopic(m, topicId, topicFunderAddress, topicFunderAccount, topicFunds)
	if err != nil {
		m.t.Fatal(err)
	}

	err = RegisterWorkerForTopic(m, topicFunderAddress, topicFunderAccount, topicId)
	if err != nil {
		m.t.Fatal(err)
	}

	err = RegisterReputerForTopic(m, topicFunderAddress, topicFunderAccount, topicId)
	if err != nil {
		m.t.Fatal(err)
	}

	err = StakeReputer(m, topicId, topicFunderAddress, topicFunderAccount, stakeToAdd)
	if err != nil {
		m.t.Fatal(err)
	}

	m.t.Log("Created new Topic with topicId", topicId)

	return topicId
}

func WorkerReputerCoordinationLoop(m TestMetadata, reputersPerEpoch, reputersMax, workersPerEpoch, workersMax, topicsPerEpoch, topicsMax, maxIterations int) {

	approximateSecondsPerBlock := getApproximateBlockTimeSeconds(m)
	fmt.Println("Approximate block time seconds: ", approximateSecondsPerBlock)

	iterationTime := time.Duration(epochLength) * approximateSecondsPerBlock * iterationsInABatch
	topicCount := 0
	topicFunderCount := 0

	topicFunderAddresses := make([]string, 0)

	for topicFunderIndex := 0; topicFunderIndex < topicsMax; topicFunderIndex++ {
		topicFunderAccountName := getTopicFunderAccountName(topicFunderIndex)
		topicFunderAccount, _, err := m.n.Client.AccountRegistry.Create(topicFunderAccountName)
		if err != nil {
			fmt.Println("Error creating funder address: ", topicFunderAccountName, " - ", err)
			continue
		}
		topicFunderAddress, err := topicFunderAccount.Address(params.HumanCoinUnit)
		if err != nil {
			fmt.Println("Error creating funder address: ", topicFunderAccountName, " - ", err)
			continue
		}
		topicFunderAddresses = append(topicFunderAddresses, topicFunderAddress)
	}

	err := fundAccounts(m, m.n.FaucetAcc, m.n.FaucetAddr, topicFunderAddresses, 1e9)
	if err != nil {
		fmt.Println("Error funding funder accounts: ", err)
	} else {
		fmt.Println("Funded", len(topicFunderAddresses), "funder accounts.")
	}

	getTopicFunder := func() (string, cosmosaccount.Account) {
		topicFunderAccountName := getTopicFunderAccountName(topicFunderCount)
		topicFunderCount++
		topicFunderAccount, err := m.n.Client.AccountRegistry.GetByName(topicFunderAccountName)
		if err != nil {
			fmt.Println("Error getting funder account: ", topicFunderAccountName, " - ", err)
			return "", topicFunderAccount
		}
		topicFunderAddress, err := topicFunderAccount.Address(params.HumanCoinUnit)
		if err != nil {
			fmt.Println("Error getting funder address: ", topicFunderAccountName, " - ", err)
			return topicFunderAddress, topicFunderAccount
		}
		return topicFunderAddress, topicFunderAccount
	}

	workerCount := 1
	reputerCount := 1

	var wg sync.WaitGroup
	if topicsPerEpoch == 0 {
		topicFunderAddress, topicFunderAccount := getTopicFunder()
		wg.Add(1)
		WorkerReputerLoop(&wg, m, topicFunderAddress, topicFunderAccount, workerCount, reputerCount,
			reputersPerEpoch, reputersMax, workersPerEpoch, workersMax, maxIterations)
		topicCount++
	} else {
		for {
			startIteration := time.Now()

			for j := 0; j < topicsPerEpoch && topicCount < topicsMax; j++ {
				topicFunderAddress, topicFunderAccount := getTopicFunder()
				wg.Add(1)
				go WorkerReputerLoop(&wg, m, topicFunderAddress, topicFunderAccount, workerCount, reputerCount,
					reputersPerEpoch, reputersMax, workersPerEpoch, workersMax, maxIterations)
				topicCount++
			}
			if topicCount >= topicsMax {
				fmt.Println("Exiting main loop: reached maximum number of topics.")
				break
			}
			workerCount += workersPerEpoch
			reputerCount += reputersPerEpoch

			elapsedIteration := time.Since(startIteration)
			sleepingTime := iterationTime - elapsedIteration
			fmt.Println(time.Now(), "Main loop sleeping", sleepingTime)
			time.Sleep(sleepingTime)
		}
	}

	wg.Wait()
}

func getTopicFunderAccountName(topicFunderIndex int) string {
	return "topicFunder" + strconv.Itoa(int(topicFunderIndex))
}

func getWorkerAccountName(workerIndex int, topicId uint64) string {
	return "stressWorker" + strconv.Itoa(workerIndex) + "_topic" + strconv.Itoa(int(topicId))
}

func getReputerAccountName(reputerIndex int, topicId uint64) string {
	return "stressReputer" + strconv.Itoa(reputerIndex) + "_topic" + strconv.Itoa(int(topicId))
}

// Main worker-reputer per-topic loop
func WorkerReputerLoop(
	wg *sync.WaitGroup,
	m TestMetadata,
	topicFunderAddress string,
	topicFunderAccount cosmosaccount.Account,
	initialWorkerCount, initialReputerCount,
	reputersPerEpoch, reputersMax, workersPerEpoch, workersMax, maxIterations int,
) {
	defer wg.Done()

	const initialWorkerReputerFundAmount = 1e5

	topicId := SetupTopic(m, topicFunderAddress, topicFunderAccount)

	report := func(a ...any) {
		fmt.Println("[ TOPIC", topicId, "] ", a)
	}

	workerAddresses := make(map[string]string)
	reputerAddresses := make(map[string]string)

	approximateSecondsBlockTime := getApproximateBlockTimeSeconds(m)

	// Make a loop, in each iteration
	// 1. generate a new bech32 reputer account and a bech32 worker account. Store them in a slice
	// 2. Fund the accounts
	// 3. Register the accounts
	// 4. Generate a worker bundle
	// 5. Generate a reputer bundle
	// 6. Insert the worker bundle (adjust nonces if failure)
	// 7. Insert the reputer bundle (adjust nonces if failure)
	// 8. Sleep one epoch
	// 9. Repeat

	workerAddressesToFund := make([]string, 0)

	for workerIndex := 0; workerIndex < workersMax; workerIndex++ {
		workerAccountName := getWorkerAccountName(workerIndex, topicId)
		workerAccount, _, err := m.n.Client.AccountRegistry.Create(workerAccountName)
		if err != nil {
			fmt.Println("Error creating funder address: ", workerAccountName, " - ", err)
			continue
		}
		workerAddressToFund, err := workerAccount.Address(params.HumanCoinUnit)
		if err != nil {
			fmt.Println("Error creating funder address: ", workerAccountName, " - ", err)
			continue
		}
		workerAddressesToFund = append(workerAddressesToFund, workerAddressToFund)
	}

	err := fundAccounts(m, topicFunderAccount, topicFunderAddress, workerAddressesToFund, initialWorkerReputerFundAmount)
	if err != nil {
		fmt.Println("Error funding worker accounts: ", err)
	} else {
		fmt.Println("Funded", len(workerAddressesToFund), "worker accounts.")
	}

	reputerAddressesToFund := make([]string, 0)

	for reputerIndex := 0; reputerIndex < reputersMax; reputerIndex++ {
		reputerAccountName := getReputerAccountName(reputerIndex, topicId)
		workerAccount, _, err := m.n.Client.AccountRegistry.Create(reputerAccountName)
		if err != nil {
			fmt.Println("Error creating funder address: ", reputerAccountName, " - ", err)
			continue
		}
		reputerAddressToFund, err := workerAccount.Address(params.HumanCoinUnit)
		if err != nil {
			fmt.Println("Error creating funder address: ", reputerAccountName, " - ", err)
			continue
		}
		reputerAddressesToFund = append(reputerAddressesToFund, reputerAddressToFund)
	}

	err = fundAccounts(m, topicFunderAccount, topicFunderAddress, reputerAddressesToFund, initialWorkerReputerFundAmount)
	if err != nil {
		fmt.Println("Error funding reputer accounts: ", err)
	} else {
		fmt.Println("Funded", len(reputerAddressesToFund), "reputer accounts.")
	}

	// Get fresh topic
	topic, err := getNonZeroTopicEpochLastRan(m.ctx, m.n.QueryEmissions, topicId, 5, approximateSecondsBlockTime)
	if err != nil {
		report("--- Failed getting a topic that was ran ---")
		require.NoError(m.t, err)
	}

	blockHeightCurrent := topic.EpochLastEnded - topic.EpochLength
	blockHeightEval := blockHeightCurrent + topic.EpochLength
	// Translate the epoch length into time
	iterationTime := time.Duration(topic.EpochLength) * approximateSecondsBlockTime * iterationsInABatch

	for i := 0; i < maxIterations; i++ {

		// Funding topic
		err := FundTopic(m, topicId, topicFunderAddress, topicFunderAccount, topicFunds)
		if err != nil {
			report("Funding topic failed: ", err)
		}

		blockHeightCurrent += topic.EpochLength * iterationsInABatch
		blockHeightEval += topic.EpochLength * iterationsInABatch

		startIteration := time.Now()

		report("iteration: ", i, " / ", maxIterations)

		initializeNewWorkerAccount := func() {
			// Generate new worker accounts
			workerAccountName := getWorkerAccountName(len(workerAddresses), topicId)
			report("Initializing worker address: ", workerAccountName)
			workerAccount, err := m.n.Client.AccountRegistry.GetByName(workerAccountName)
			if err != nil {
				report("Error getting worker address: ", workerAccountName, " - ", err)
				return
			}
			workerAddress, err := workerAccount.Address(params.HumanCoinUnit)
			if err != nil {
				report("Error getting worker address: ", workerAccountName, " - ", err)
				return
			}
			err = RegisterWorkerForTopic(m, workerAddress, workerAccount, topicId)
			if err != nil {
				report("Error registering worker address: ", workerAddress, " - ", err)
				return
			}
			workerAddresses[workerAccountName] = workerAddress
		}

		initializeNewReputerAccount := func() {
			// Generate new reputer account
			reputerAccountName := getReputerAccountName(len(reputerAddresses), topicId)
			report("Initializing reputer address: ", reputerAccountName)
			reputerAccount, err := m.n.Client.AccountRegistry.GetByName(reputerAccountName)
			if err != nil {
				report("Error getting reputer address: ", reputerAccountName, " - ", err)
				return
			}
			reputerAddress, err := reputerAccount.Address(params.HumanCoinUnit)
			if err != nil {
				report("Error getting reputer address: ", reputerAccountName, " - ", err)
				return
			}
			err = RegisterReputerForTopic(m, reputerAddress, reputerAccount, topicId)
			if err != nil {
				report("Error registering reputer address: ", reputerAddress, " - ", err)
				return
			}
			err = StakeReputer(m, topicId, reputerAddress, reputerAccount, stakeToAdd)
			if err != nil {
				report("Error staking reputer address: ", reputerAddress, " - ", err)
				return
			}
			reputerAddresses[reputerAccountName] = reputerAddress
		}

		if i == 0 {
			for j := 0; j < initialWorkerCount; j++ {
				initializeNewWorkerAccount()
			}
			for j := 0; j < initialReputerCount; j++ {
				initializeNewReputerAccount()
			}
		} else {
			for j := 0; j < workersPerEpoch; j++ {
				if len(workerAddresses) >= workersMax {
					break
				}
				initializeNewWorkerAccount()
			}

			for j := 0; j < reputersPerEpoch; j++ {
				if len(reputerAddresses) >= reputersMax {
					break
				}
				initializeNewReputerAccount()
			}
		}

		// Insert worker bulk, choosing one random leader from the worker accounts
		leaderWorkerAccountName, _, err := GetRandomMapEntryValue(workerAddresses)
		if err != nil {
			report("Error getting random worker address: ", err)
			continue
		}
		startWorker := time.Now()
		err = InsertWorkerBulk(m, topic, leaderWorkerAccountName, workerAddresses, blockHeightCurrent)
		if err != nil {
			if strings.Contains(err.Error(), "nonce already fulfilled") {
				// realign blockHeights before retrying
				topic, err = getLastTopic(m.ctx, m.n.QueryEmissions, topicId)
				if err == nil {
					blockHeightCurrent = topic.EpochLastEnded + topic.EpochLength
					blockHeightEval = blockHeightCurrent - topic.EpochLength
					report("Reset blockHeights to (", blockHeightCurrent, ",", blockHeightEval, ")")
				} else {
					report("Error getting topic!")
				}
			}
			continue
		} else {
			report("Inserted worker bulk, blockHeight: ", blockHeightCurrent, " with ", len(workerAddresses), " workers")
			elapsedBulk := time.Since(startWorker)
			report("Insert Worker ", blockHeightCurrent, " Elapsed time:", elapsedBulk)
		}

		// Insert reputer bulk, choosing one random leader from reputer accounts
		leaderReputerAccountName, _, err := GetRandomMapEntryValue(reputerAddresses)
		if err != nil {
			report("Error getting random worker address: ", err)
			continue
		}
		startReputer := time.Now()
		err = InsertReputerBulk(m, topic, leaderReputerAccountName, reputerAddresses, workerAddresses, blockHeightCurrent, blockHeightEval)
		if err != nil {
			if strings.Contains(err.Error(), "nonce already fulfilled") || strings.Contains(err.Error(), "nonce still unfulfilled") {
				// realign blockHeights before retrying
				topic, err = getLastTopic(m.ctx, m.n.QueryEmissions, topicId)
				if err == nil {
					blockHeightCurrent = topic.EpochLastEnded + topic.EpochLength
					blockHeightEval = blockHeightCurrent - topic.EpochLength
					report("Reset blockHeights to (", blockHeightCurrent, ",", blockHeightEval, ")")
				} else {
					report("Error getting topic!")
				}
			}
			continue
		} else {
			report("Inserted reputer bulk, blockHeight: ", blockHeightCurrent, " with ", len(reputerAddresses), " reputers")
			elapsedBulk := time.Since(startReputer)
			report("Insert Reputer Elapsed time:", elapsedBulk)
		}

		// Sleep for 2 epoch
		elapsedIteration := time.Since(startIteration)
		sleepingTime := iterationTime - elapsedIteration
		report(time.Now(), " Sleeping...", sleepingTime, ", elapsed: ", elapsedIteration, " epoch length seconds: ", iterationTime)
		time.Sleep(sleepingTime)
	}

	countWorkers := len(workerAddresses)
	countReputers := len(reputerAddresses)

	for workerIndex := 0; workerIndex < countWorkers; workerIndex++ {
		balance, err := getAccountBalance(m.ctx, m.n.QueryBank, workerAddresses[getWorkerAccountName(workerIndex, topicId)])
		if err != nil {
			report("Error getting worker balance for worker: ", workerIndex, err)
		} else {
			if balance.Amount.Int64() <= initialWorkerReputerFundAmount {
				report("Worker ", workerIndex, " balance is less than initial amount: ", balance.Amount.Int64())
			} else {
				report("Worker ", workerIndex, " balance: ", balance.Amount.Int64())
			}
		}
	}

	for reputerIndex := 0; reputerIndex < countReputers; reputerIndex++ {
		reputerStake, err := getReputerStake(m.ctx, m.n.QueryEmissions, topicId, reputerAddresses[getReputerAccountName(reputerIndex, topicId)])
		if err != nil {
			report("Error getting reputer stake for reputer: ", reputerIndex, err)
		} else {
			if reputerStake <= stakeToAdd {
				report("Reputer ", reputerIndex, " stake is less than initial amount: ", reputerStake)
			} else {
				report("Reputer ", reputerIndex, " stake: ", reputerStake)
			}
		}
	}
}

func GetRandomMapEntryValue(workerAddresses map[string]string) (string, string, error) {
	// Get the number of entries in the map
	numEntries := len(workerAddresses)
	if numEntries == 0 {
		return "", "", fmt.Errorf("map is empty")
	}

	// Generate a random index
	randomIndex := rand.Intn(numEntries)

	// Iterate over the map to find the entry at the random index
	var randomKey string
	var i int
	for key := range workerAddresses {
		if i == randomIndex {
			randomKey = key
			break
		}
		i++
	}

	// Return the value corresponding to the randomly selected key
	return randomKey, workerAddresses[randomKey], nil
}

/* This code is unused
func fundAccount(m TestMetadata, senderAccount cosmosaccount.Account, senderAddress, address string, amount int64) error {
	initialCoins := sdktypes.NewCoins(sdktypes.NewCoin(params.BaseCoinUnit, cosmossdk_io_math.NewInt(amount)))
	// Fund the account from faucet account
	sendMsg := &banktypes.MsgSend{
		FromAddress: senderAddress,
		ToAddress:   address,
		Amount:      initialCoins,
	}
	_, err := m.n.Client.BroadcastTx(m.ctx, senderAccount, sendMsg)
	if err != nil {
		fmt.Println("Error funding worker address: ", err)
		return err
	}
	return nil
}
*/

func fundAccounts(
	m TestMetadata,
	senderAccount cosmosaccount.Account,
	senderAddress string,
	addresses []string,
	amount int64,
) error {
	inputCoins := sdktypes.NewCoins(sdktypes.NewCoin(params.BaseCoinUnit, cosmossdk_io_math.NewInt(amount*int64(len(addresses)))))
	outputCoins := sdktypes.NewCoins(sdktypes.NewCoin(params.BaseCoinUnit, cosmossdk_io_math.NewInt(amount)))

	outputs := []banktypes.Output{}
	for _, address := range addresses {
		outputs = append(outputs, banktypes.Output{
			Address: address,
			Coins:   outputCoins,
		})
	}

	// Fund the accounts from faucet account in a single transaction
	sendMsg := &banktypes.MsgMultiSend{
		Inputs: []banktypes.Input{
			{
				Address: senderAddress,
				Coins:   inputCoins,
			},
		},
		Outputs: outputs,
	}
	_, err := m.n.Client.BroadcastTx(m.ctx, senderAccount, sendMsg)
	if err != nil {
		fmt.Println("Error worker address: ", err)
		return err
	}
	return nil
}

func getApproximateBlockTimeSeconds(m TestMetadata) time.Duration {
	emissionsParams := GetEmissionsParams(m)
	blocksPerMonth := emissionsParams.GetBlocksPerMonth()
	return time.Duration(secondsInAMonth/blocksPerMonth) * time.Second
}

func getLastTopic(ctx context.Context, query types.QueryClient, topicID uint64) (*types.Topic, error) {
	topicResponse, err := query.GetTopic(ctx, &types.QueryTopicRequest{TopicId: topicID})
	if err == nil {
		return topicResponse.Topic, nil
	}
	return nil, err
}

func getAccountBalance(ctx context.Context, queryClient banktypes.QueryClient, address string) (*sdktypes.Coin, error) {
	req := &banktypes.QueryAllBalancesRequest{
		Address:    address,
		Pagination: &query.PageRequest{Limit: 1},
	}

	res, err := queryClient.AllBalances(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(res.Balances) > 0 {
		return &res.Balances[0], nil
	}
	return nil, fmt.Errorf("no balance found for address: %s", address)
}

func getReputerStake(ctx context.Context, queryClient types.QueryClient, topicId uint64, reputerAddress string) (uint64, error) {
	req := &types.QueryReputerStakeInTopicRequest{
		Address: reputerAddress,
		TopicId: topicId,
	}
	res, err := queryClient.GetReputerStakeInTopic(ctx, req)
	if err != nil {
		return 0, err
	}
	return res.Amount.Uint64(), nil
}
