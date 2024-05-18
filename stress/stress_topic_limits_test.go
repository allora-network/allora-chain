package stress_test

import (
	"context"
	"os"
	"testing"

	chain_test "github.com/allora-network/allora-chain/stress/chain"
	"github.com/stretchr/testify/require"
)

func SetupTopicLimitsTest(t *testing.T) TestMetadata {
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

func TestStressTestTopicLimitsSuite(t *testing.T) {
	if _, isIntegration := os.LookupEnv("STRESS_TEST_TOPIC_LIMITS"); isIntegration == false {
		t.Skip("Skipping Stress Test topic limits unless explicitly enabled")
	}

	const stakeToAdd uint64 = 10000
	const topicFunds int64 = 10000000000000000

	t.Log(">>> Setting up connection to local node <<<")
	m := Setup(t)
	t.Log(">>> Test Topic Creation <<<")
	topicId := CreateTopic(m)

	t.Log(">>> Test Topic Funding and Activation <<<")
	err := FundTopic(m, topicId, m.n.FaucetAddr, m.n.FaucetAcc, topicFunds)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(">>> Starting initial registration, to start topic churn cycle <<<")
	err = RegisterWorkerForTopic(m, m.n.UpshotAddr, m.n.UpshotAcc, topicId)
	if err != nil {
		t.Fatal(err)
	}
	err = RegisterReputerForTopic(m, m.n.FaucetAddr, m.n.FaucetAcc, topicId)
	if err != nil {
		t.Fatal(err)
	}

	err = StakeReputer(m, topicId, m.n.FaucetAddr, m.n.FaucetAcc, stakeToAdd)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(">>> Test Making Inference <<<")
	WorkerReputerLoop(m, topicId)

}
