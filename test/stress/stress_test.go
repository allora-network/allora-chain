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
	reputersPerEpoch := testCommon.LookupEnvInt(t, "REPUTERS_PER_EPOCH", 1)
	reputersMax := testCommon.LookupEnvInt(t, "REPUTERS_MAX", 100)
	workersPerEpoch := testCommon.LookupEnvInt(t, "WORKERS_PER_EPOCH", 1)
	workersMax := testCommon.LookupEnvInt(t, "WORKERS_MAX", 100)
	topicsPerEpoch := testCommon.LookupEnvInt(t, "TOPICS_PER_EPOCH", 1)
	topicsMax := testCommon.LookupEnvInt(t, "TOPICS_MAX", 100)
	maxIterations := testCommon.LookupEnvInt(t, "MAX_ITERATIONS", 1000)
	epochLength := testCommon.LookupEnvInt(t, "EPOCH_LENGTH", 5)
	doFinalReport := testCommon.LookupEnvBool(t, "FINAL_REPORT", false)

	fmt.Println("Reputers per epoch: ", reputersPerEpoch)
	fmt.Println("Reputers max: ", reputersMax)
	fmt.Println("Workers per epoch: ", workersPerEpoch)
	fmt.Println("Workers max: ", workersMax)
	fmt.Println("Topics per epoch: ", topicsPerEpoch)
	fmt.Println("Topics max: ", topicsMax)
	fmt.Println("Max iterations: ", maxIterations)
	fmt.Println("Epoch Length: ", epochLength)
	fmt.Println("Using mutex to prepare final report: ", doFinalReport)

	t.Log(">>> Test Making Inference <<<")
	WorkerReputerCoordinationLoop(
		testConfig,
		reputersPerEpoch,
		reputersMax,
		workersPerEpoch,
		workersMax,
		topicsPerEpoch,
		topicsMax,
		maxIterations,
		epochLength,
		doFinalReport,
	)
}
