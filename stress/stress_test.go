package stress_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"

	chain_test "github.com/allora-network/allora-chain/stress/chain"
	"github.com/stretchr/testify/require"
)

type TestMetadata struct {
	t   *testing.T
	ctx context.Context
	n   chain_test.Node
}

func Setup(t *testing.T) TestMetadata {
	ret := TestMetadata{}
	ret.t = t
	var err error
	ret.ctx = context.Background()
	// userHomeDir, _ := os.UserHomeDir()
	// home := filepath.Join(userHomeDir, ".allorad")
	node, err := chain_test.NewNode(
		t,
		chain_test.NodeConfig{
			NodeRPCAddress: "http://localhost:26657",
			AlloraHomeDir:  "./devnet/genesis",
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

	t.Log(">>> Test Making Inference <<<")
	WorkerReputerLoop(m, true)
}
