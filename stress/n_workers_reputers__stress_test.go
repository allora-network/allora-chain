package stress_test

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	cosmossdk_io_math "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/stretchr/testify/require"
)

const MAX_ITERATIONS = 10000

const defaultEpochLength = 10
const approximateBlockLengthSeconds = 5
const minWaitingNumberofEpochs = 3

func getNonZeroTopicEpochLastRan(ctx context.Context, query emissionstypes.QueryClient, topicID uint64, maxRetries int) (*emissionstypes.Topic, error) {
	sleepingTimeBlocks := defaultEpochLength
	// Retry loop for a maximum of 5 times
	for retries := 0; retries < maxRetries; retries++ {
		topicResponse, err := query.GetTopic(ctx, &emissionstypes.QueryTopicRequest{TopicId: topicID})
		if err == nil {
			storedTopic := topicResponse.Topic
			if storedTopic.EpochLastEnded != 0 {
				sleepingTimeSeconds := time.Duration(minWaitingNumberofEpochs*storedTopic.EpochLength*approximateBlockLengthSeconds) * time.Second
				fmt.Println(time.Now(), " Topic found, sleeping...", sleepingTimeSeconds)
				time.Sleep(sleepingTimeSeconds)
				fmt.Println(time.Now(), " Slept.")
				return topicResponse.Topic, nil
			}
			sleepingTimeBlocks = int(storedTopic.EpochLength)
		} else {
			fmt.Println("Error getting topic, retry...", err)
		}
		// Sleep for a while before retrying
		fmt.Println("Retrying sleeping for a default epoch, retry ", retries, " for sleeping time ", sleepingTimeBlocks)
		time.Sleep(time.Duration(sleepingTimeBlocks*approximateBlockLengthSeconds) * time.Second)
	}

	return nil, errors.New("topicEpochLastRan is still 0 after retrying")
}

func generateWorkerAttributedValueLosses(m TestMetadata, workerAddresses map[string]string, lowLimit, sum int) []*types.WorkerAttributedValue {
	values := make([]*types.WorkerAttributedValue, 0)
	for key := range workerAddresses {
		values = append(values, &types.WorkerAttributedValue{
			Worker: workerAddresses[key],
			Value:  alloraMath.NewDecFromInt64(int64(rand.Intn(lowLimit) + sum)),
		})
	}
	return values
}

func generateWithheldWorkerAttributedValueLosses(m TestMetadata, workerAddresses map[string]string, lowLimit, sum int) []*types.WithheldWorkerAttributedValue {
	values := make([]*types.WithheldWorkerAttributedValue, 0)
	for key := range workerAddresses {
		values = append(values, &types.WithheldWorkerAttributedValue{
			Worker: workerAddresses[key],
			Value:  alloraMath.NewDecFromInt64(int64(rand.Intn(lowLimit) + sum)),
		})
	}
	return values
}

func generateSingleWorkerBundle(m TestMetadata, topic *types.Topic, blockHeight int64,
	workerAddressName string, workerAddresses map[string]string) *types.WorkerDataBundle {
	// Nonce: calculate from EpochLastRan + EpochLength
	topicId := topic.Id
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
func generateValueBundle(m TestMetadata, topicId uint64, workerAddresses map[string]string, reputerNonce, workerNonce *types.Nonce) types.ValueBundle {
	return types.ValueBundle{
		TopicId:                topicId,
		CombinedValue:          alloraMath.NewDecFromInt64(100),
		InfererValues:          generateWorkerAttributedValueLosses(m, workerAddresses, 3000, 3500),
		ForecasterValues:       generateWorkerAttributedValueLosses(m, workerAddresses, 50, 50),
		NaiveValue:             alloraMath.NewDecFromInt64(100),
		OneOutInfererValues:    generateWithheldWorkerAttributedValueLosses(m, workerAddresses, 50, 50),
		OneOutForecasterValues: generateWithheldWorkerAttributedValueLosses(m, workerAddresses, 50, 50),
		OneInForecasterValues:  generateWorkerAttributedValueLosses(m, workerAddresses, 50, 50),
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
	}
}

// Inserts worker bulk, given a topic, blockHeight, and leader worker address (which should exist in the keyring)
func InsertLeaderWorkerBulk(
	m TestMetadata,
	topic *types.Topic,
	blockHeight int64,
	leaderWorkerAccountName, leaderWorkerAddress string,
	WorkerDataBundles []*types.WorkerDataBundle) error {

	topicId := topic.Id
	nonce := emissionstypes.Nonce{BlockHeight: blockHeight}

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
func InsertWorkerBulk(m TestMetadata, topic *types.Topic, leaderWorkerAccountName string, workerAddresses map[string]string, blockHeight int64) {
	// Get fresh topic to use its EpochLastRan
	topicResponse, err := m.n.QueryEmissions.GetTopic(m.ctx, &emissionstypes.QueryTopicRequest{TopicId: topic.Id})
	require.NoError(m.t, err)
	freshTopic := topicResponse.Topic
	// Get Bundles
	workerDataBundles := make([]*types.WorkerDataBundle, 0)
	for key := range workerAddresses {
		workerDataBundles = append(workerDataBundles, generateSingleWorkerBundle(m, topic, blockHeight, key, workerAddresses))
	}
	leaderWorkerAddress := workerAddresses[leaderWorkerAccountName]
	fmt.Println("Inserting worker bulk for blockHeight: ", blockHeight, "leader name: ", leaderWorkerAccountName, ", addr: ", leaderWorkerAddress, " len: ", len(workerDataBundles))
	InsertLeaderWorkerBulk(m, freshTopic, blockHeight, leaderWorkerAccountName, leaderWorkerAddress, workerDataBundles)
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
	reputerNonce, workerNonce *emissionstypes.Nonce) *types.MsgInsertBulkReputerPayload {

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
	valueBundle := generateValueBundle(m, topicId, workerAddresses, reputerNonce, workerNonce)
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

func SetupTopic(m TestMetadata) (uint64, *types.Topic) {

	m.t.Log("Creating new Topic")

	topicId, topic := CreateTopic(m)

	err := FundTopic(m, topicId, m.n.FaucetAddr, m.n.FaucetAcc, topicFunds)
	if err != nil {
		m.t.Fatal(err)
	}

	err = RegisterWorkerForTopic(m, m.n.UpshotAddr, m.n.UpshotAcc, topicId)
	if err != nil {
		m.t.Fatal(err)
	}

	err = RegisterReputerForTopic(m, m.n.FaucetAddr, m.n.FaucetAcc, topicId)
	if err != nil {
		m.t.Fatal(err)
	}

	err = StakeReputer(m, topicId, m.n.FaucetAddr, m.n.FaucetAcc, stakeToAdd)
	if err != nil {
		m.t.Fatal(err)
	}

	m.t.Log("Created new Topic with topicId", topicId)

	return topicId, topic
}

const stakeToAdd uint64 = 10000
const topicFunds int64 = 10000000000000000

// Register two actors and check their registrations went through
func WorkerReputerLoop(m TestMetadata) {

	reputersPerEpoch := lookupEnvInt(m, "REPUTERS_PER_EPOCH", 0)
	reputersMax := lookupEnvInt(m, "REPUTERS_MAX", 10000)
	workersPerEpoch := lookupEnvInt(m, "WORKERS_PER_EPOCH", 0)
	workersMax := lookupEnvInt(m, "WORKERS_MAX", 10000)
	topicsPerEpoch := lookupEnvInt(m, "TOPICS_PER_EPOCH", 0)
	topicsMax := lookupEnvInt(m, "TOPICS_MAX", 100)

	workerCount := 0
	reputerCount := 0

	createWorkerAccountName := func() string {
		cacheWorkerAcount := workerCount
		workerCount++
		return "stressWorker" + strconv.Itoa(cacheWorkerAcount)
	}

	createReputerAccountName := func() string {
		cacheReputerAcount := reputerCount
		reputerCount++
		return "stressReputer" + strconv.Itoa(cacheReputerAcount)
	}

	workerAccountNames := []string{createWorkerAccountName()}
	reputerAccountNames := []string{createReputerAccountName()}

	initialTopicId, _ := SetupTopic(m)

	topic, err := getNonZeroTopicEpochLastRan(m.ctx, m.n.QueryEmissions, initialTopicId, 5)
	if err != nil {
		m.t.Log("--- Failed getting a topic that was ran ---")
		require.NoError(m.t, err)
	}

	topics := []*types.Topic{}
	topics = append(topics, topic)

	createAndFundNewAccounts := func(m TestMetadata) {
		allCombinedAccountNames := []string{}

		for _, topic := range topics {
			for _, workerAccountName := range combineAccountNamesAndTopicId(workerAccountNames, topic.Id) {
				allCombinedAccountNames = append(allCombinedAccountNames, workerAccountName)
			}

			for _, reputerAccountName := range combineAccountNamesAndTopicId(reputerAccountNames, topic.Id) {
				allCombinedAccountNames = append(allCombinedAccountNames, reputerAccountName)
			}
		}

		allNewAccountAddresses := createAccountsIfNotExists(m, allCombinedAccountNames)

		err := fundAccounts(m, m.n.FaucetAcc, m.n.FaucetAddr, allNewAccountAddresses, 100000)
		if err != nil {
			fmt.Println("Error funding accounts: ", err)
		} else {
			fmt.Println("Funded", len(allNewAccountAddresses), "accounts.")
		}
	}

	createAndFundNewAccounts(m)

	blockHeightCurrent := topic.EpochLastEnded
	blockHeightEval := blockHeightCurrent + topic.EpochLength

	ProcessTopicLoopInner(
		m,
		workerAccountNames,
		reputerAccountNames,
		topic,
		blockHeightCurrent,
		blockHeightEval,
	)

	// Translate the epoch length into time
	epochTimeSeconds := time.Duration(topic.EpochLength*approximateBlockLengthSeconds) * time.Second

	for i := 0; i < MAX_ITERATIONS; i++ {
		start := time.Now()

		fmt.Println("iteration: ", i, " / ", MAX_ITERATIONS)

		for j := 0; j < workersPerEpoch; j++ {
			if workerCount >= workersMax {
				break
			}
			workerAccountNames = append(workerAccountNames, createWorkerAccountName())
		}

		for j := 0; j < reputersPerEpoch; j++ {
			if reputerCount >= reputersMax {
				break
			}
			reputerAccountNames = append(reputerAccountNames, createReputerAccountName())
		}

		var wg sync.WaitGroup
		wg.Add(len(topics))

		createAndFundNewAccounts(m)

		for _, topic := range topics {
			go ProcessTopicLoop(
				m,
				workerAccountNames,
				reputerAccountNames,
				topic,
				blockHeightCurrent,
				blockHeightEval,
				&wg,
			)
		}

		wg.Wait()

		for j := 0; j < topicsPerEpoch; j++ {
			if len(topics) >= topicsMax {
				break
			}
			// Generate new topic
			_, topic := SetupTopic(m)
			require.NoError(m.t, err)
			topics = append(topics, topic)
		}

		// Sleep for one epoch
		elapsed := time.Since(start)
		fmt.Println("Elapsed time:", elapsed, " BlockHeightCurrent: ", blockHeightCurrent, " BlockHeightEval: ", blockHeightEval)
		sleepingTimeSeconds := epochTimeSeconds - elapsed
		for sleepingTimeSeconds < 0 {
			fmt.Println("Passed over an epoch, moving to the next one")
			sleepingTimeSeconds += epochTimeSeconds
		}
		fmt.Println(time.Now(), " Sleeping...", sleepingTimeSeconds, ", epoch length: ", epochTimeSeconds)
		time.Sleep(sleepingTimeSeconds)

		// Update blockHeights
		blockHeightCurrent += topic.EpochLength * 2
		blockHeightEval += topic.EpochLength * 2
	}
}

func combineAccountNameAndTopicId(accountName string, topicId uint64) string {
	return accountName + "_" + strconv.Itoa(int(topicId))
}

func combineAccountNamesAndTopicId(accountNames []string, topicId uint64) []string {
	combinedAccountNames := make([]string, 0)
	for _, accountName := range accountNames {
		combinedAccountNames = append(combinedAccountNames, combineAccountNameAndTopicId(accountName, topicId))
	}
	return combinedAccountNames
}

func getAccountNameToAddressMap(m TestMetadata, accountNames []string, topicId uint64) map[string]string {
	accountNameToAddress := make(map[string]string)
	for _, accountName := range accountNames {
		address, _ := fetchAccountAndAddress(m, accountName)
		accountNameToAddress[accountName] = address
	}
	return accountNameToAddress
}

func ProcessTopicLoop(
	m TestMetadata,
	workerAccountNames []string,
	reputerAccountNames []string,
	topic *types.Topic,
	blockHeightCurrent,
	blockHeightEval int64,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	ProcessTopicLoopInner(
		m,
		workerAccountNames,
		reputerAccountNames,
		topic,
		blockHeightCurrent,
		blockHeightEval,
	)
}

func createAccount(m TestMetadata, accountName string) (string, cosmosaccount.Account) {
	account, _, err := m.n.Client.AccountRegistry.Create(accountName)
	if err != nil {
		m.t.Log("Error creating account: ", accountName, " - ", err)
		return "", account
	}

	address, err := account.Address(params.HumanCoinUnit)
	if err != nil {
		m.t.Log("Error getting address: ", accountName, " - ", err)
		return address, account
	}

	m.t.Log("Created new account", accountName, "address", address)

	return address, account
}

func fetchAccountAndAddress(m TestMetadata, accountName string) (string, cosmosaccount.Account) {
	account, err := m.n.Client.AccountRegistry.GetByName(accountName)
	if err != nil {
		m.t.Log("Error fetching account: ", accountName, " - ", err)
		return "", account
	}

	address, err := account.Address(params.HumanCoinUnit)
	if err != nil {
		m.t.Log("Error fetching address: ", accountName, " - ", err)
		return address, account
	}

	return address, account
}

func createAccountsIfNotExists(m TestMetadata, accountNames []string) []string {
	addresses := make([]string, 0)
	for _, accountName := range accountNames {
		address, _, justCreated := fetchAccountAndAddressAndCreate(m, accountName)
		if justCreated {
			addresses = append(addresses, address)
		}
	}
	return addresses
}

func fetchAccountAndAddressAndCreate(m TestMetadata, accountName string) (string, cosmosaccount.Account, bool) {
	account, err := m.n.Client.AccountRegistry.GetByName(accountName)
	if err != nil {
		address, account := createAccount(m, accountName)
		return address, account, true
	}

	address, err := account.Address(params.HumanCoinUnit)
	if err != nil {
		m.t.Log("Error fetching address: ", accountName, " - ", err)
		return address, account, false
	}

	return address, account, false
}

func registerWorker(
	m TestMetadata,
	workerAccountName string,
	topicId uint64,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	workerAddress, workerAccount := fetchAccountAndAddress(m, workerAccountName)

	err := RegisterWorkerForTopic(m, workerAddress, workerAccount, topicId)
	if err != nil {
		fmt.Println("Error registering worker address: ", workerAddress, " - ", err)
		return
	}

	fmt.Println("Registered worker address:", workerAddress, "on topicId:", topicId)
}

func registerAndStakeReputer(
	m TestMetadata,
	reputerAccountName string,
	topicId uint64,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	reputerAddress, reputerAccount := fetchAccountAndAddress(m, reputerAccountName)

	err := RegisterReputerForTopic(m, reputerAddress, reputerAccount, topicId)
	if err != nil {
		fmt.Println("Error registering reputer address: ", reputerAddress, " - ", err)
	}
	err = StakeReputer(m, topicId, reputerAddress, reputerAccount, stakeToAdd)
	if err != nil {
		fmt.Println("Error staking reputer address: ", reputerAddress, " - ", err)
		return
	}

	fmt.Println("Registered and staked reputer address:", reputerAddress, "on topicId:", topicId)
}

func ProcessTopicLoopInner(
	m TestMetadata,
	rawWorkerAccountNames []string,
	rawReputerAccountNames []string,
	topic *types.Topic,
	blockHeightCurrent,
	blockHeightEval int64,
) {

	topicId := topic.Id

	workerAccountNames := combineAccountNamesAndTopicId(rawWorkerAccountNames, topicId)
	reputerAccountNames := combineAccountNamesAndTopicId(rawReputerAccountNames, topicId)

	// Register workers and reputers
	var wg2 sync.WaitGroup
	wg2.Add(len(workerAccountNames) + len(reputerAccountNames))

	for _, workerAccountName := range workerAccountNames {
		go registerWorker(
			m,
			workerAccountName,
			topicId,
			&wg2,
		)
	}

	for _, reputerAccountName := range reputerAccountNames {
		go registerAndStakeReputer(
			m,
			reputerAccountName,
			topicId,
			&wg2,
		)
	}

	wg2.Wait()

	workerAccountMap := getAccountNameToAddressMap(m, workerAccountNames, topicId)

	// Choose one random leader from the worker accounts
	leaderWorkerAccountName, leaderWorkerAddress, err := GetRandomMapEntryValue(workerAccountMap)
	if err != nil {
		fmt.Println("Error getting random worker address: ", err)
		return
	}

	// Insert worker
	m.t.Log("--- Insert Worker Bulk --- with leader: ", leaderWorkerAccountName, " and worker address: ", leaderWorkerAddress)
	InsertWorkerBulk(m, topic, leaderWorkerAccountName, workerAccountMap, blockHeightCurrent)
	InsertWorkerBulk(m, topic, leaderWorkerAccountName, workerAccountMap, blockHeightEval)

	reputerAccountMap := getAccountNameToAddressMap(m, reputerAccountNames, topicId)

	// Insert reputer bulk
	// Choose one random leader from reputer accounts
	leaderReputerAccountName, leaderReputerAddress, err := GetRandomMapEntryValue(reputerAccountMap)
	if err != nil {
		fmt.Println("Error getting random reputer address: ", err)
		return
	}

	m.t.Log("--- Insert Reputer Bulk --- with leader: ", leaderReputerAccountName, " and reputer address: ", leaderReputerAddress)
	InsertReputerBulk(m, topic, leaderReputerAccountName, reputerAccountMap, reputerAccountMap, blockHeightCurrent, blockHeightEval)
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

func fundAccount(
	m TestMetadata,
	senderAccount cosmosaccount.Account,
	senderAddress, string,
	address string,
	amount int64,
) error {
	initialCoins := sdktypes.NewCoins(sdktypes.NewCoin(params.BaseCoinUnit, cosmossdk_io_math.NewInt(amount)))
	// Fund the account from faucet account
	sendMsg := &banktypes.MsgSend{
		FromAddress: senderAddress,
		ToAddress:   address,
		Amount:      initialCoins,
	}
	_, err := m.n.Client.BroadcastTx(m.ctx, senderAccount, sendMsg)
	if err != nil {
		fmt.Println("Error worker address: ", err)
		return err
	}
	return nil
}

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

	// Fund the account from faucet account
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
