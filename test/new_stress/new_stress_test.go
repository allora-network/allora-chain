package newstress_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	testcommon "github.com/allora-network/allora-chain/test/common"
)

const topicFunds int64 = 1e6
const stakeToAdd uint64 = 9e4

func TestNewStressTestSuite(t *testing.T) {
	t.Log(">>> Environment <<<")
	seed := testcommon.LookupEnvInt(t, "SEED", 1)
	rpcMode := testcommon.LookupRpcMode(t, "RPC_MODE", testcommon.SingleRpc)
	rpcEndpoints := testcommon.LookupEnvStringArray("RPC_URLS", []string{"http://localhost:26657"})

	testConfig := testcommon.NewTestConfig(
		t,
		rpcMode,
		rpcEndpoints,
		"../localnet/genesis",
		seed,
	)

	// Read env vars with defaults
	epochLength := testcommon.LookupEnvInt(t, "EPOCH_LENGTH", 12) // in blocks
	numTopics := testcommon.LookupEnvInt(t, "NUM_TOPICS", 10)
	workersPerTopic := testcommon.LookupEnvInt(t, "WORKERS_PER_TOPIC", 5)
	reputersPerTopic := testcommon.LookupEnvInt(t, "REPUTERS_PER_TOPIC", 4)
	createTopicsSameBlock := testcommon.LookupEnvBool(t, "CREATE_TOPICS_SAME_BLOCK", false)
	timeoutMinutes := testcommon.LookupEnvInt(t, "STRESS_TIMEOUT_MINUTES", 5)

	t.Log("Epoch Length: ", epochLength)
	t.Log("Number of Topics: ", numTopics)
	t.Log("Workers per Topic: ", workersPerTopic)
	t.Log("Reputers per Topic: ", reputersPerTopic)
	t.Log("Create Topics in Same Block: ", createTopicsSameBlock)
	t.Log("Stress Test Timeout: ", timeoutMinutes)
	t.Log(">>> Starting Test <<<")
	timestr := fmt.Sprintf(">>> Starting %s <<<", time.Now().Format(time.RFC850))
	t.Log(timestr)

	numActors := workersPerTopic + reputersPerTopic
	_, simulationData := simulateSetUp(&testConfig, numActors, epochLength)

	topicIds, err := startCreateTopicsAndRegister(
		&testConfig,
		simulationData,
		numTopics,
		workersPerTopic,
		reputersPerTopic,
		createTopicsSameBlock,
	)
	requireNoError(t, simulationData.failOnErr, err)

	err = simulateAutomatic(
		&testConfig,
		simulationData,
		topicIds,
		timeoutMinutes,
	)
	requireNoError(t, simulationData.failOnErr, err)
}

// startCreateTopicsAndRegister creates topics and registers workers and reputers according to env variables
func startCreateTopicsAndRegister(
	m *testcommon.TestConfig,
	data *SimulationData,
	numTopics int,
	workersPerTopic int,
	reputersPerTopic int,
	createTopicsSameBlock bool,
) ([]uint64, error) {
	actor := data.actors[0]
	topicIds, err := createTopics(m, actor, numTopics, data.epochLength, createTopicsSameBlock)
	if err != nil {
		return nil, err
	}

	/* TODO:
	We will need to create individual workers and reputers accounts for each topic to be able to do concurrent registration
	that's why I added 4 seconds sleep between registering workers and reputers for each topic, to avoid race conditions
	*/
	workers := data.actors[:workersPerTopic]
	reputers := data.actors[workersPerTopic:]
	for _, topicId := range topicIds {
		m.T.Logf("Registering workers in topic: %d", topicId)
		err = registerWorkers(m, workers, topicId, data, workersPerTopic)
		if err != nil {
			m.T.Logf("Error registering workers: %v", err)
			return nil, err
		}
		m.T.Logf("Registering reputers and adding stake in topic: %d", topicId)
		err = registerReputersAndStake(m, reputers, topicId, data, reputersPerTopic)
		if err != nil {
			return nil, err
		}
		time.Sleep(4 * time.Second)
	}

	err = fundTopics(m, topicIds, actor, topicFunds)
	if err != nil {
		return nil, err
	}

	return topicIds, nil
}

func simulateAutomatic(
	m *testcommon.TestConfig,
	data *SimulationData,
	topicIds []uint64,
	timeoutMinutes int,
) error {
	m.T.Logf("Starting submission loop for %d topics", len(topicIds))
	// Create wait group to track all goroutines
	var wg sync.WaitGroup
	// Create error channel to catch any errors from goroutines
	errChan := make(chan error, len(topicIds)*2)

	// For each topic, start a worker routine and a reputer routine
	for _, topicId := range topicIds {
		// Add 2 to waitgroup (1 for worker, 1 for reputer)
		wg.Add(2)

		// Start worker routine
		go func(tid uint64) {
			defer wg.Done()
			err := runTopicWorkersLoop(m, data, tid)
			if err != nil {
				errChan <- fmt.Errorf("worker routine failed for topic %d: %w", tid, err)
			}
		}(topicId)

		// Start reputer routine
		go func(tid uint64) {
			defer wg.Done()
			err := runReputersProcess(m, data, tid)
			if err != nil {
				errChan <- fmt.Errorf("reputer routine failed for topic %d: %w", tid, err)
			}
		}(topicId)
	}

	// Create a channel that closes when all goroutines are done
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait for either completion or an error
	select {
	case err := <-errChan:
		return err
	case <-done:
		return nil
	case <-time.After(time.Duration(timeoutMinutes) * time.Minute):
		return fmt.Errorf("simulation timed out after %d minutes", timeoutMinutes)
	}
}
