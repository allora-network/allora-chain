package stress_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"

	testCommon "github.com/allora-network/allora-chain/test/common"
	"github.com/stretchr/testify/require"
)

type TestMetadata struct {
	t   *testing.T
	ctx context.Context
	n   testCommon.Node
}

func Setup(t *testing.T) TestMetadata {
	ret := TestMetadata{}
	ret.t = t
	var err error
	ret.ctx = context.Background()
	node, err := testCommon.NewNode(
		t,
		testCommon.NodeConfig{
			NodeRPCAddress: "http://localhost:26657",
			AlloraHomeDir:  "../devnet/genesis",
		},
	)
	require.NoError(t, err)
	ret.n = node
	return ret
}

func TestStressTestSuite(t *testing.T) {
	if _, isIntegration := os.LookupEnv("STRESS_TEST"); isIntegration == false {
		t.Skip("Skipping Stress Test unless explicitly enabled")
	}

	numCPUs := runtime.NumCPU()
	gomaxprocs := runtime.GOMAXPROCS(0)
	fmt.Printf("Number of logical CPUs: %d, GOMAXPROCS %d \n", numCPUs, gomaxprocs)

	t.Log(">>> Setting up connection to local node <<<")
	m := Setup(t)

	// Read env vars with defaults
	reputersPerEpoch := lookupEnvInt(m, "REPUTERS_PER_EPOCH", 0)
	reputersMax := lookupEnvInt(m, "REPUTERS_MAX", 100)
	workersPerEpoch := lookupEnvInt(m, "WORKERS_PER_EPOCH", 0)
	workersMax := lookupEnvInt(m, "WORKERS_MAX", 100)
	topicsPerEpoch := lookupEnvInt(m, "TOPICS_PER_EPOCH", 0)
	topicsMax := lookupEnvInt(m, "TOPICS_MAX", 100)
	maxIterations := lookupEnvInt(m, "MAX_ITERATIONS", 1000)
	epochLength := lookupEnvInt(m, "EPOCH_LENGTH", 5)
	doFinalReport := lookupEnvBool(m, "FINAL_REPORT", false)

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
		m,
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
