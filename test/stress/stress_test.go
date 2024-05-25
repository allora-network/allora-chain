package stress_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	testCommon "github.com/allora-network/allora-chain/test/common"
)

func TestStressTestSuite(t *testing.T) {
	if _, isIntegration := os.LookupEnv("STRESS_TEST"); isIntegration == false {
		t.Skip("Skipping Stress Test unless explicitly enabled")
	}

	numCPUs := runtime.NumCPU()
	gomaxprocs := runtime.GOMAXPROCS(0)
	fmt.Printf("Number of logical CPUs: %d, GOMAXPROCS %d \n", numCPUs, gomaxprocs)

	t.Log(">>> Setting up connection to local node <<<")

	seed := int64(testCommon.LookupEnvInt(t, "SEED", 0))
	rpcMode := testCommon.LookupRpcMode(t, "RPC_MODE", testCommon.SingleRpc)
	rpcEndpoints := testCommon.LookupEnvStringArray("RPC_URLS", []string{"http://localhost:26657"})

	testConfig := testCommon.NewTestConfig(
		t,
		rpcMode,
		rpcEndpoints,
		"../devnet/genesis",
		seed,
	)

	// Read env vars with defaults
	reputersPerIteration := testCommon.LookupEnvInt(t, "REPUTERS_PER_ITERATION", 1)
	maxReputersPerTopic := testCommon.LookupEnvInt(t, "MAX_REPUTERS_PER_TOPIC", 20)
	workersPerIteration := testCommon.LookupEnvInt(t, "WORKERS_PER_ITERATION", 1)
	maxWorkersPerTopic := testCommon.LookupEnvInt(t, "MAX_WORKERS_PER_TOPIC", 20)
	topicsPerIteration := testCommon.LookupEnvInt(t, "TOPICS_PER_ITERATION", 1)
	topicsMax := testCommon.LookupEnvInt(t, "TOPICS_MAX", 100)
	maxIterations := testCommon.LookupEnvInt(t, "MAX_ITERATIONS", 1000)
	epochLength := testCommon.LookupEnvInt(t, "EPOCH_LENGTH", 5)
	doFinalReport := testCommon.LookupEnvBool(t, "FINAL_REPORT", false)

	fmt.Println("Reputers per iteration: ", reputersPerIteration)
	fmt.Println("Max Reputers per topic: ", maxReputersPerTopic)
	fmt.Println("Workers per iteration: ", workersPerIteration)
	fmt.Println("Max Workers per topic: ", maxWorkersPerTopic)
	fmt.Println("Topics per iteration of topics: ", topicsPerIteration)
	fmt.Println("Topics global max: ", topicsMax)
	fmt.Println("Max worker+reputer iterations: ", maxIterations)
	fmt.Println("Epoch Length: ", epochLength)
	fmt.Println("Use mutex to prepare final report: ", doFinalReport)

	t.Log(">>> Test Making Inference <<<")
	workerReputerCoordinationLoop(
		testConfig,
		reputersPerIteration,
		maxReputersPerTopic,
		workersPerIteration,
		maxWorkersPerTopic,
		topicsPerIteration,
		topicsMax,
		maxIterations,
		epochLength,
		doFinalReport,
	)
}
