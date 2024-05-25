package stress_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
)

const secondsInAMonth = 2592000
const defaultEpochLength = 10      // Default epoch length in blocks if none is found yet from chain
const minWaitingNumberofEpochs = 3 // To control the number of epochs to wait before inserting the first batch

// holder for account and address
type AccountAndAddress struct {
	acc  cosmosaccount.Account
	addr string
}

// maps of worker and reputer names to their account and address information
type NameToAccountMap map[NAME]AccountAndAddress

// simple wrapper around fmt.Println
func topicLog(topicId uint64, a ...any) {
	fmt.Println("[ TOPIC", topicId, "] ", a)
}

// return standardized account name for funders
func getTopicFunderAccountName(topicFunderIndex int) string {
	return "topicFunder" + strconv.Itoa(int(topicFunderIndex))
}

// return standardized account name for workers
func getWorkerAccountName(workerIndex int, topicId uint64) string {
	return "stressWorker" + strconv.Itoa(workerIndex) + "_topic" + strconv.Itoa(int(topicId))
}

// return standardized account name for reputers
func getReputerAccountName(reputerIndex int, topicId uint64) string {
	return "stressReputer" + strconv.Itoa(reputerIndex) + "_topic" + strconv.Itoa(int(topicId))
}

// return the approximate block time in seconds
func getApproximateBlockTimeSeconds(m testCommon.TestConfig) time.Duration {
	emissionsParams := GetEmissionsParams(m)
	blocksPerMonth := emissionsParams.GetBlocksPerMonth()
	return time.Duration(secondsInAMonth/blocksPerMonth) * time.Second
}

// Get the most recent topic
func getLastTopic(
	ctx context.Context,
	query types.QueryClient,
	topicID uint64,
) (*types.Topic, error) {
	topicResponse, err := query.GetTopic(ctx, &types.QueryTopicRequest{TopicId: topicID})
	if err == nil {
		return topicResponse.Topic, nil
	}
	return nil, err
}

// get the token holdings of an address from the bank module
func getAccountBalance(
	ctx context.Context,
	queryClient banktypes.QueryClient,
	address string,
) (*sdktypes.Coin, error) {
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

// get from the emissions module what the reputer's stake is
func getReputerStake(
	ctx context.Context,
	queryClient types.QueryClient,
	topicId uint64,
	reputerAddress string,
) (alloraMath.Dec, error) {
	req := &types.QueryReputerStakeInTopicRequest{
		Address: reputerAddress,
		TopicId: topicId,
	}
	res, err := queryClient.GetReputerStakeInTopic(ctx, req)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	return alloraMath.MustNewDecFromString(res.Amount.String()), nil
}

// return from the emissions module what the maximum amount of rewarded workers and reporters should be
func getMaxTopWorkersReputersToReward(m testCommon.TestConfig) (uint64, uint64, error) {
	emissionsParams := GetEmissionsParams(m)
	topWorkersCount := emissionsParams.GetMaxTopWorkersToReward()
	topReputersCount := emissionsParams.GetMaxTopReputersToReward()
	return topWorkersCount, topReputersCount, nil
}

// This function gets the topic checking activity.
// After that it will sleep for a number of epoch
// to ensure nonces are available.
func getNonZeroTopicEpochLastRan(
	ctx context.Context,
	query types.QueryClient,
	topicID uint64,
	maxRetries int,
	approximateSecondsPerBlock time.Duration,
) (*types.Topic, error) {
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
		fmt.Println(
			"Retrying sleeping for a default epoch, retry ",
			retries,
			" for sleeping time ",
			sleepingTimeBlocks,
		)
		time.Sleep(time.Duration(sleepingTimeBlocks) * approximateSecondsPerBlock * time.Second)
	}

	return nil, errors.New("topicEpochLastRan is still 0 after retrying")
}

// This function picks a key from a map randomly and returns it.
// TYLER NOTE TODO: ideally this should be seeded with the seed from the env
// and made to be concurrency aware
func pickRandomKeyFromMap(x map[string]AccountAndAddress) (string, error) {
	// Get the number of entries in the map
	numEntries := len(x)
	if numEntries == 0 {
		return "", fmt.Errorf("map is empty")
	}

	// Generate a random index
	randomIndex := rand.Intn(numEntries)

	// Iterate over the map to find the entry at the random index
	var randomKey string
	var i int
	for key := range x {
		if i == randomIndex {
			randomKey = key
			break
		}
		i++
	}

	// Return the value corresponding to the randomly selected key
	return randomKey, nil
}
