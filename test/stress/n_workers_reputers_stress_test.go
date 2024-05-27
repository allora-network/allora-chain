package stress_test

import (
	"sync"
	"time"

	testCommon "github.com/allora-network/allora-chain/test/common"
	"github.com/stretchr/testify/require"
)

const iterationsInABatch = 1 // To control the number of epochs in each iteration of the loop (eg to manage insertions)
const stakeToAdd uint64 = 9e4
const topicFunds int64 = 1e6
const initialWorkerReputerFundAmount int64 = 1e5
const retryBundleUploadTimes int = 2

// create a topic, fund the topic,
// register the topic funder as a worker, reputer,
// and then stake the topic funder as a reputer
func setupTopic(
	m testCommon.TestConfig,
	funder NameAccountAndAddress,
	epochLength int64,
) uint64 {
	m.T.Log("Creating new Topic")

	topicId := createTopic(m, epochLength, funder)

	err := fundTopic(m, topicId, funder, topicFunds)
	if err != nil {
		m.T.Fatal(err)
	}

	err = RegisterWorkerForTopic(m, funder, topicId)
	if err != nil {
		m.T.Fatal(err)
	}

	err = RegisterReputerForTopic(m, funder, topicId)
	if err != nil {
		m.T.Fatal(err)
	}

	err = stakeReputer(m, topicId, funder, stakeToAdd)
	if err != nil {
		m.T.Fatal(err)
	}

	m.T.Log("Created new Topic with topicId", topicId)

	return topicId
}

// Worker reputer coordination loop.
// Creates topics, workers, and reputers in a loop to run the test.
func workerReputerCoordinationLoop(
	m testCommon.TestConfig,
	reputersPerIteration,
	maxReputersPerTopic,
	workersPerIteration,
	maxWorkersPerTopic,
	topicsPerTopicIteration,
	topicsMax,
	maxWorkerReputerIterations,
	epochLength int,
	makeReport bool,
) {
	approximateSecondsPerBlock := getApproximateBlockTimeSeconds(m)
	m.T.Log("Approximate block time seconds: ", approximateSecondsPerBlock)
	iterationTime := time.Duration(epochLength) * approximateSecondsPerBlock * iterationsInABatch

	// make sure topicFunder is rich enough to send funds to all workers and reputers
	// once for staking from the funder
	// every iteration for funding the topic
	// every iteration for each worker funding
	initialFunderAccountAmount := int64(stakeToAdd) +
		int64(topicsPerTopicIteration)*int64(maxWorkerReputerIterations+1)*topicFunds +
		int64(maxWorkerReputerIterations+1)*initialWorkerReputerFundAmount*int64(maxReputersPerTopic+maxWorkersPerTopic)

	// 1. For every single topic that will be created over the duration of the test
	//    create a topic funder that will create and fund the topic
	topicFunders := createTopicFunderAddresses(m, topicsMax)
	err := fundAccounts(
		m,
		0,
		NameAccountAndAddress{
			name: "faucet",
			aa: AccountAndAddress{
				acc:  m.FaucetAcc,
				addr: m.FaucetAddr,
			},
		},
		topicFunders,
		initialFunderAccountAmount,
	)
	if err != nil {
		m.T.Log("Error funding funder accounts: ", err)
	} else {
		m.T.Log("Funded", len(topicFunders), "funder accounts.")
	}

	// 2. Outer "Topic Iteration."
	//    Every iteration of this loop, topicsPerTopicIteration topics are created
	//    up until the topicsMax is hit.
	numTopicsThisIteration := topicsPerTopicIteration
	var wg sync.WaitGroup
	for topicCount := 0; topicCount < topicsMax; {
		startIteration := time.Now()

		// 3. the last time through the loop, we may not have enough
		//    topics left before the max to reach topicsPerTopicIteration
		if topicCount+topicsPerTopicIteration > topicsMax {
			numTopicsThisIteration = topicsMax - topicCount
		}
		for j := 0; j < numTopicsThisIteration; j++ {
			// 4. Get ahold of the funder for this topic
			topicFunderAccountName := getTopicFunderAccountName(m.Seed, topicCount)
			funder := topicFunders[topicFunderAccountName]

			wg.Add(1)
			// 5. call the inner worker reputer loop that will create
			// reputers and workers for this topic and push data
			// to the chain for this topic
			go workerReputerLoop(
				&wg,
				m,
				NameAccountAndAddress{
					name: topicFunderAccountName,
					aa:   funder,
				},
				reputersPerIteration,
				maxReputersPerTopic,
				workersPerIteration,
				maxWorkersPerTopic,
				maxWorkerReputerIterations,
				epochLength,
				makeReport,
			)
			topicCount++
		}

		// 5. Sleep for enough time to let an epoch to complete before making the next topic
		elapsedIteration := time.Since(startIteration)
		sleepingTime := iterationTime - elapsedIteration
		m.T.Log("Topics created ", topicCount, " ", time.Now(), "Main loop sleeping", sleepingTime)
		time.Sleep(sleepingTime)
	}
	m.T.Log("All routines launched: waiting for running routines to end.")
	wg.Wait()

	// 6. If applicable, generate a summary report of the test
	if makeReport {
		reportSummaryStatistics()
	}
}

// Main worker-reputer per-topic loop
// inside each topic, take the following actions
// Each call of the workerReputerLoop corresponds to ONE topic
func workerReputerLoop(
	wg *sync.WaitGroup,
	m testCommon.TestConfig,
	funder NameAccountAndAddress,
	reputersPerIteration,
	maxReputersPerTopic,
	workersPerIteration,
	maxWorkersPerTopic,
	maxIterations,
	epochLength int,
	makeReport bool,
) {
	defer wg.Done()
	approximateSecondsBlockTime := getApproximateBlockTimeSeconds(m)

	// Create and fund a new topic
	// Additionally register the topic funder as a worker and reputer for that topic
	topicId := setupTopic(m, funder, int64(epochLength))

	// Generate the worker and reputer bech32 accounts we will need for this new topic
	numWorkersInThisTopic := min(maxIterations*workersPerIteration, maxWorkersPerTopic)
	numReputersInThisTopic := min(maxIterations*reputersPerIteration, maxReputersPerTopic)
	workers := createWorkerAddresses(m, topicId, numWorkersInThisTopic)
	reputers := createReputerAddresses(m, topicId, numReputersInThisTopic)

	actors := mapUnion(workers, reputers)
	// Fund the accounts
	err := fundAccounts(
		m,
		topicId,
		funder,
		actors,
		initialWorkerReputerFundAmount*int64(maxIterations),
	)
	if err != nil {
		m.T.Log(topicLog(topicId, "Error funding reputer and worker accounts: ", err))
	}

	// Wait until the topic has had minWaitingNumberofEpochs before starting to provide inferences for it
	topic, err := getNonZeroTopicEpochLastRan(
		m,
		topicId,
		5,
		approximateSecondsBlockTime,
	)
	if err != nil {
		m.T.Log(topicLog(topicId, "--- Failed getting a topic that was ran ---"))
		require.NoError(m.T, err)
	}

	// Translate the epoch length into time
	iterationTime := time.Duration(topic.EpochLength) * approximateSecondsBlockTime * iterationsInABatch

	countWorkers := 0
	countReputers := 0
	// Begin "iterations" inside the topic
	for i := 0; i < maxIterations; i++ {
		// Fund the topic to give it money to make inferences
		err := fundTopic(m, topicId, funder, topicFunds)
		if err != nil {
			m.T.Log(topicLog(topicId, "Funding topic failed: ", err))
			if makeReport {
				saveTopicError(topicId, err)
			}
		}
		startIteration := time.Now()

		m.T.Log(topicLog(topicId, "iteration: ", i, " / ", maxIterations-1))

		// Register the newly created accounts for this iteration
		countWorkers = registerWorkersForIteration(
			m,
			topicId,
			i,
			workersPerIteration,
			countWorkers,
			maxWorkersPerTopic,
			workers,
			makeReport,
		)
		// Register the reputers, and additionally stake some tokens
		countReputers = registerReputersForIteration(
			m,
			topicId,
			i,
			reputersPerIteration,
			countReputers,
			maxReputersPerTopic,
			reputers,
			makeReport,
		)

		// Generate and insert a worker bundle (adjust nonces if failure)
		err = generateInsertWorkerBundle(
			m,
			topic,
			workers,
			retryBundleUploadTimes,
			makeReport,
		)
		if err != nil {
			m.T.Log(topicLog(topicId, "Error generate/inserting worker bundle: ", err))
			if makeReport {
				saveTopicError(topicId, err)
			}
		}

		// Generate and insert reputer bundle scoring workers
		err = generateInsertReputerBulk(
			m,
			topic,
			reputers,
			workers,
			retryBundleUploadTimes,
			makeReport,
		)
		if err != nil {
			m.T.Log(topicLog(topicId, "Error generate/inserting reputer bundle: ", err))
			if makeReport {
				saveTopicError(topicId, err)
			}
		}

		// Sleep for an epoch
		elapsedIteration := time.Since(startIteration)
		sleepingTime := iterationTime - elapsedIteration
		m.T.Log(topicLog(topicId, time.Now(), " Sleeping...", sleepingTime, ", elapsed: ", elapsedIteration, " epoch length seconds: ", iterationTime))
		time.Sleep(sleepingTime)
	}

	// Check that the workers have been paid rewards
	rewardedWorkersCount, err := checkWorkersReceivedRewards(
		m,
		topicId,
		workers,
		countWorkers,
		maxIterations,
		makeReport,
	)
	if err != nil {
		m.T.Log(topicLog(topicId, "Error checking worker rewards: ", err))
		if makeReport {
			saveTopicError(topicId, err)
		}
	}

	// Check that the reputer have been paid rewards via a stake greater than the initial amount
	rewardedReputersCount, err := checkReputersReceivedRewards(
		m,
		topicId,
		reputers,
		countReputers,
		maxIterations,
		makeReport,
	)
	if err != nil {
		m.T.Log(topicLog(topicId, "Error checking reputer rewards: ", err))
		if makeReport {
			saveTopicError(topicId, err)
		}
	}

	// Check that only the top workers and reputers are rewarded
	maxTopWorkersCount, maxTopReputersCount, _ := getMaxTopWorkersReputersToReward(m)
	require.Less(m.T, rewardedWorkersCount, maxTopWorkersCount, "Only top workers can get reward")
	require.Less(m.T, rewardedReputersCount, maxTopReputersCount, "Only top reputers can get reward")
}
