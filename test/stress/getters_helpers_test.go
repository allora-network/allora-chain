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

type NameAccountAndAddress struct {
	name string
	aa   AccountAndAddress
}

// holder for account and address
type AccountAndAddress struct {
	acc  cosmosaccount.Account
	addr string
}

// maps of worker and reputer names to their account and address information
type NameToAccountMap map[NAME]AccountAndAddress

// simple wrapper around topicLog
func topicLog(topicId uint64, a ...any) string {
	return fmt.Sprint("[TOPIC", topicId, "] ", a)
}

// return standardized account name for funders
func getTopicFunderAccountName(seed int, topicFunderIndex int) string {
	return "stress" + strconv.Itoa(seed) + "_topic_funder" + strconv.Itoa(int(topicFunderIndex))
}

// return standardized account name for workers
func getWorkerAccountName(seed int, workerIndex int, topicId uint64) string {
	return "stress" + strconv.Itoa(seed) + "_topic" + strconv.Itoa(int(topicId)) + "_worker" + strconv.Itoa(workerIndex)
}

// return standardized account name for reputers
func getReputerAccountName(seed int, reputerIndex int, topicId uint64) string {
	return "stress" + strconv.Itoa(seed) + "_topic" + strconv.Itoa(int(topicId)) + "_reputer" + strconv.Itoa(reputerIndex)
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
func getMaxTopWorkersReputersToReward(m testCommon.TestConfig) (uint64, uint64, uint64, error) {
	emissionsParams := GetEmissionsParams(m)
	topInferersCount := emissionsParams.GetMaxTopInferersToReward()
	topForecastersCount := emissionsParams.GetMaxTopForecastersToReward()
	topReputersCount := emissionsParams.GetMaxTopReputersToReward()
	return topInferersCount, topForecastersCount, topReputersCount, nil
}

// This function gets the topic checking activity.
// After that it will sleep for a number of epoch
// to ensure nonces are available.
func getNonZeroTopicEpochLastRan(
	m testCommon.TestConfig,
	topicId uint64,
	maxRetries int,
	approximateSecondsPerBlock time.Duration,
) (*types.Topic, error) {
	sleepingTimeBlocks := defaultEpochLength
	// Retry loop for a maximum of 5 times
	ctx := context.Background()
	for retries := 0; retries < maxRetries; retries++ {
		topicResponse, err := m.Client.QueryEmissions().GetTopic(
			ctx,
			&types.QueryTopicRequest{TopicId: topicId},
		)
		if err == nil {
			storedTopic := topicResponse.Topic
			if storedTopic.EpochLastEnded != 0 {
				nBlocks := storedTopic.EpochLength * minWaitingNumberofEpochs
				sleepingTime := time.Duration(nBlocks) * approximateSecondsPerBlock
				m.T.Log(topicLog(topicId, time.Now(), " Topic found, sleeping...", sleepingTime))
				time.Sleep(sleepingTime)
				m.T.Log(topicLog(topicId, time.Now(), " Looking for topic: Slept."))
				return topicResponse.Topic, nil
			}
			sleepingTimeBlocks = int(storedTopic.EpochLength)
		} else {
			m.T.Log(topicLog(topicId, "Error getting topic, retry...", err))
		}
		// Sleep for a while before retrying
		m.T.Log(topicLog(topicId, "Retrying sleeping for a default epoch, retry ", retries, " for sleeping time ", sleepingTimeBlocks, " blocks"))
		time.Sleep(time.Duration(sleepingTimeBlocks) * approximateSecondsPerBlock * time.Second)
	}

	return nil, errors.New("topicEpochLastRan is still 0 after retrying")
}

// This function picks a key from a map randomly and returns it.
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

// copies two maps into a new combined map
// O(n) over each map separately
// if the maps contain the same keys, the first map
// takes precedence for value
func mapUnion(a NameToAccountMap, b NameToAccountMap) NameToAccountMap {
	combined := make(NameToAccountMap)
	for k, v := range b {
		combined[k] = v
	}
	for k, v := range a {
		combined[k] = v
	}
	return combined
}
