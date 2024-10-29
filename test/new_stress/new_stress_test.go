package newstress_test

import (
	"fmt"
	"testing"
	"time"

	testcommon "github.com/allora-network/allora-chain/test/common"
)

const topicFunds int64 = 1e6

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
	maxIterations := testcommon.LookupEnvInt(t, "MAX_ITERATIONS", 1000)
	epochLength := testcommon.LookupEnvInt(t, "EPOCH_LENGTH", 12) // in blocks

	
	numTopics := testcommon.LookupEnvInt(t, "NUM_TOPICS", 10)
	workersPerTopic := testcommon.LookupEnvInt(t, "WORKERS_PER_TOPIC", 5)
	reputersPerTopic := testcommon.LookupEnvInt(t, "REPUTERS_PER_TOPIC", 4)
	createTopicsSameBlock := testcommon.LookupEnvBool(t, "CREATE_TOPICS_SAME_BLOCK", false)

	t.Log("Max Iterations: ", maxIterations)
	t.Log("Epoch Length: ", epochLength)
	t.Log("Number of Topics: ", numTopics)
	t.Log("Workers per Topic: ", workersPerTopic)
	t.Log("Reputers per Topic: ", reputersPerTopic)
	t.Log("Create Topics in Same Block: ", createTopicsSameBlock)
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

	fmt.Println("Topic IDs: ", topicIds)

	// simulateAutomatic(
	// 	&testConfig,
	// 	faucet,
	// 	simulationData,
	// 	maxIterations,
	// )
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

	// Fund Topics
	err = fundTopics(m, topicIds, actor, topicFunds)
	if err != nil {
		return nil, err
	}

	// Register Workers and Reputers
	for _, topicId := range topicIds {
		fmt.Println("Registering workers in topic: ", workersPerTopic)
		err = registerWorkers(m, data.actors, topicId, data, workersPerTopic)
		if err != nil {
			fmt.Println("Error registering workers: ", err)
			return nil, err
		}
		fmt.Println("Registering reputers in topic: ", reputersPerTopic)
		err = registerReputers(m, data.actors, topicId, data, reputersPerTopic)
		if err != nil {
			return nil, err
		}
	}

	return topicIds, nil
}
