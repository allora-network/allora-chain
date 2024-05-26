package stress_test

import (
	"sync"
	"time"

	testCommon "github.com/allora-network/allora-chain/test/common"
	"github.com/stretchr/testify/require"
)

const iterationsInABatch = 1 // To control the number of epochs in each iteration of the loop (eg to manage insertions)
const stakeToAdd = 90000
const topicFunds = 1000000
const initialWorkerReputerFundAmount = 1e5
const retryBundleUploadTimes = 2

// create a topic, fund the topic,
// register the topic funder as a worker, reputer,
// and then stake the topic funder as a reputer
func setupTopic(
	m testCommon.TestConfig,
	funder AccountAndAddress,
	epochLength int64,
) uint64 {
	m.T.Log("Creating new Topic")

	topicId := createTopic(m, epochLength, funder.addr, funder.acc)

	err := fundTopic(m, topicId, funder, topicFunds)
	if err != nil {
		m.T.Fatal(err)
	}

	err = RegisterWorkerForTopic(m, funder.addr, funder.acc, topicId)
	if err != nil {
		m.T.Fatal(err)
	}

	err = RegisterReputerForTopic(m, funder.addr, funder.acc, topicId)
	if err != nil {
		m.T.Fatal(err)
	}

	err = StakeReputer(m, topicId, funder.addr, funder.acc, stakeToAdd)
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

	// 1. For every single topic that will be created over the duration of the test
	//    create a topic funder that will create and fund the topic
	topicFunders := createTopicFunderAddresses(m, topicsMax)
	faucet := AccountAndAddress{acc: m.FaucetAcc, addr: m.FaucetAddr}
	err := fundAccounts(m, faucet, topicFunders, 1e9)
	if err != nil {
		m.T.Log("Error funding funder accounts: ", err)
	} else {
		m.T.Log("Funded", len(topicFunders), "funder accounts.")
	}

	// 2. Outer "Topic Iteration."
	//    Every iteration of this loop, topicsPerTopicIteration topics are created
	//    up until the topicsMax is hit.
	topicsThisEpoch := topicsPerTopicIteration
	var wg sync.WaitGroup
	for topicCount := 0; topicCount < topicsMax; {
		startIteration := time.Now()

		// 3. the last time through the loop, we may not have enough
		//    topics left before the max to reach topicsPerTopicIteration
		if topicCount+topicsPerTopicIteration > topicsMax {
			topicsThisEpoch = topicsMax - topicCount
		}
		for j := 0; j < topicsThisEpoch; j++ {
			// 4. Get ahold of the funder for this topic
			topicFunderAccountName := getTopicFunderAccountName(topicCount)
			funder := topicFunders[topicFunderAccountName]

			wg.Add(1)
			// 5. call the inner worker reputer loop that will create
			// reputers and workers for this topic and push data
			// to the chain for this topic
			go workerReputerLoop(
				&wg,
				m,
				funder,
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
		m.T.Log(topicLog(uint64(topicCount), time.Now(), "Main loop sleeping", sleepingTime))
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
	funder AccountAndAddress,
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
		funder,
		actors,
		initialWorkerReputerFundAmount,
	)
	if err != nil {
		m.T.Log(topicLog(topicId, "Error funding reputer and worker accounts: ", err))
	} else {
		m.T.Log(topicLog(topicId, "Funded", len(actors), "total actor accounts for this topic."))
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

		m.T.Log(topicLog(topicId, "iteration: ", i, " / ", maxIterations))

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
