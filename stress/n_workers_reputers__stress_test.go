package stress_test

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
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

// Inserts worker bulk, given a topic, blockHeight, and leader worker address (which should exist in the keyring)
func InsertLeaderWorkerBulk(
	m TestMetadata,
	topic *types.Topic,
	blockHeight int64,
	leaderWorkerAccountName, leaderWorkerAddress string,
	WorkerDataBundles []*types.WorkerDataBundle) {

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
	require.NoError(m.t, err)
	txResp, err := m.n.Client.BroadcastTx(m.ctx, LeaderAcc, workerMsg)
	require.NoError(m.t, err)
	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	require.NoError(m.t, err)

}

// Worker Bob inserts bulk inference and forecast
func InsertWorkerBulk(m TestMetadata, topic *types.Topic, leaderWorkerAccountName string, workerAddresses map[string]string) (int64, int64) {
	// Get fresh topic to use its EpochLastRan
	topicResponse, err := m.n.QueryEmissions.GetTopic(m.ctx, &emissionstypes.QueryTopicRequest{TopicId: topic.Id})
	require.NoError(m.t, err)
	freshTopic := topicResponse.Topic
	blockHeight := freshTopic.EpochLastEnded + freshTopic.EpochLength
	// Get Bundles
	workerDataBundles := make([]*types.WorkerDataBundle, 0)
	for key := range workerAddresses {
		workerDataBundles = append(workerDataBundles, generateSingleWorkerBundle(m, topic, blockHeight, key, workerAddresses))
	}
	leaderWorkerAddress := workerAddresses[leaderWorkerAccountName]
	fmt.Println("Inserting worker bulk for blockHeight: ", blockHeight, "leader name: ", leaderWorkerAccountName, ", addr: ", leaderWorkerAddress, " len: ", len(workerDataBundles))
	InsertLeaderWorkerBulk(m, freshTopic, blockHeight, leaderWorkerAccountName, leaderWorkerAddress, workerDataBundles)
	return blockHeight, blockHeight - freshTopic.EpochLength
}

// register alice as a reputer in topic 1, then check success
func InsertReputerBulk(m TestMetadata, topic *types.Topic, BlockHeightCurrent, BlockHeightEval int64) {
	// Nonce: calculate from EpochLastRan + EpochLength
	topicId := topic.Id
	// Define inferer address as Bob's address, reputer as Alice's
	workerAddr := m.n.UpshotAddr
	reputerAddr := m.n.FaucetAddr
	// Nonces are last two blockHeights
	reputerNonce := &types.Nonce{
		BlockHeight: BlockHeightCurrent,
	}
	workerNonce := &types.Nonce{
		BlockHeight: BlockHeightEval,
	}

	reputerValueBundle := &types.ValueBundle{
		TopicId:       topicId,
		Reputer:       reputerAddr,
		CombinedValue: alloraMath.NewDecFromInt64(100),
		InfererValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr,
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		ForecasterValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr,
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		NaiveValue: alloraMath.NewDecFromInt64(100),
		OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
			// There cannot be a 1-out inferer value if there is just 1 inferer => this will be ignored by msgserver
			{
				Worker: workerAddr,
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{
			{
				Worker: workerAddr,
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		// Just as valid:
		// OneOutInfererValues:    []*types.WithheldWorkerAttributedValue{},
		// OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{},
		OneInForecasterValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr,
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
	}

	// Sign
	src := make([]byte, 0)
	src, err := reputerValueBundle.XXX_Marshal(src, true)
	require.NoError(m.t, err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature, pubKey, err := m.n.Client.Context().Keyring.Sign(m.n.FaucetAcc.Name, src, signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(m.t, err, "Sign should not return an error")
	reputerPublicKeyBytes := pubKey.Bytes()

	// Create a MsgInsertBulkReputerPayload message
	lossesMsg := &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddr,
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				ValueBundle: reputerValueBundle,
				Signature:   valueBundleSignature,
				Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
			},
		},
	}

	txResp, err := m.n.Client.BroadcastTx(m.ctx, m.n.FaucetAcc, lossesMsg)
	require.NoError(m.t, err)
	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	require.NoError(m.t, err)

	result, err := m.n.QueryEmissions.GetNetworkLossBundleAtBlock(m.ctx,
		&emissionstypes.QueryNetworkLossBundleAtBlockRequest{
			TopicId:     topicId,
			BlockHeight: BlockHeightCurrent,
		},
	)
	require.NoError(m.t, err)
	require.NotNil(m.t, result)
	require.NotNil(m.t, result.LossBundle, "Retrieved data should match inserted data")

}

// Register two actors and check their registrations went through
func WorkerReputerLoop(m TestMetadata, topicId uint64) {
	const stakeToAdd uint64 = 10000
	workerAddresses := make(map[string]string)
	reputerAddresses := make(map[string]string)
	// Make a loop, in each iteration
	// 1. generate a new bech32 reputer account and a bech32 worker account. Store them in a slice
	// 2. Pass worker slice in the call to insertWorkerBulk
	// 3. Pass reputer slice in the call to insertReputerBulk
	// 4. sleep one epoch, then repeat.

	// Get fresh topic
	topic, err := getNonZeroTopicEpochLastRan(m.ctx, m.n.QueryEmissions, 1, 5)
	if err != nil {
		m.t.Log("--- Failed getting a topic that was ran ---")
		require.NoError(m.t, err)
	}

	for i := 0; i < MAX_ITERATIONS; i++ {
		fmt.Println("iteration: ", i, " / ", MAX_ITERATIONS)
		// Generate new worker accounts
		workerAccountName := "stressWorker" + strconv.Itoa(i)
		workerAccount, _, err := m.n.Client.AccountRegistry.Create(workerAccountName)
		if err != nil {
			fmt.Println("Error creating worker address: ", workerAccountName, " - ", err)
			continue
		}
		workerAddress, err := workerAccount.Address(params.HumanCoinUnit)
		if err != nil {
			fmt.Println("Error getting worker address: ", workerAccountName, " - ", err)
			continue
		}
		fmt.Println("Worker address: ", workerAddress)
		err = fundAccount(m, m.n.FaucetAcc, m.n.FaucetAddr, workerAddress, 100000)
		if err != nil {
			fmt.Println("Error funding worker address: ", workerAddress, " - ", err)
			continue
		}
		err = RegisterWorkerForTopic(m, workerAddress, workerAccount, topicId)
		if err != nil {
			fmt.Println("Error registering worker address: ", workerAddress, " - ", err)
			continue
		}
		workerAddresses[workerAccountName] = workerAddress

		// Generate new reputer account
		reputerAccountName := "stressReputer" + strconv.Itoa(i)
		reputerAccount, _, err := m.n.Client.AccountRegistry.Create(reputerAccountName)
		if err != nil {
			fmt.Println("Error creating reputer address: ", reputerAccountName, " - ", err)
			continue
		}
		reputerAddress, err := reputerAccount.Address(params.HumanCoinUnit)
		if err != nil {
			fmt.Println("Error getting reputer address: ", reputerAccountName, " - ", err)
			continue
		}
		fmt.Println("Reputer address: ", reputerAddress)
		err = fundAccount(m, m.n.FaucetAcc, m.n.FaucetAddr, reputerAddress, 100000)
		if err != nil {
			fmt.Println("Error funding reputer address: ", reputerAddress, " - ", err)
			continue
		}
		err = RegisterReputerForTopic(m, reputerAddress, reputerAccount, topicId)
		if err != nil {
			fmt.Println("Error registering reputer address: ", reputerAddress, " - ", err)
			continue
		}
		err = StakeReputer(m, topicId, reputerAddress, reputerAccount, stakeToAdd)
		if err != nil {
			fmt.Println("Error staking reputer address: ", reputerAddress, " - ", err)
			continue
		}
		reputerAddresses[reputerAccountName] = reputerAddress

		// Choose one random leader from the worker accounts
		leaderWorkerAccountName, leaderWorkerAddress, err := GetRandomMapEntryValue(workerAddresses)
		if err != nil {
			fmt.Println("Error getting random worker address: ", err)
			continue
		}

		// Insert worker
		m.t.Log("--- Insert Worker Bulk --- with leader: ", leaderWorkerAccountName, " and worker address: ", leaderWorkerAddress)
		start := time.Now()
		blockHeightCurrent, blockHeightEval := InsertWorkerBulk(m, topic, leaderWorkerAccountName, workerAddresses)
		elapsed := time.Since(start)
		fmt.Println("Insert Worker Elapsed time:", elapsed)
		fmt.Println("BlockHeightCurrent: ", blockHeightCurrent, "BlockHeightEval: ", blockHeightEval)

		// Insert reputer bulk
		// start = time.Now()
		// InsertReputerBulk(m, topic, reputerAccounts, blockHeightCurrent, blockHeightEval)
		// elapsed = time.Since(start)
		// fmt.Println("Insert Reputer Elapsed time:", elapsed)

		// Sleep for one epoch
		sleepingTimeSeconds := time.Duration(topic.EpochLength*approximateBlockLengthSeconds) * time.Second
		fmt.Println(time.Now(), " Sleeping...", sleepingTimeSeconds)
		time.Sleep(sleepingTimeSeconds)
		fmt.Println(time.Now(), " Slept.")
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
