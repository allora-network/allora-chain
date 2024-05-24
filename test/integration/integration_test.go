package integration_test

import (
	"context"
	"os"
	"testing"

	testCommon "github.com/allora-network/allora-chain/test/common"
)

type TestMetadata struct {
	t   *testing.T
	ctx context.Context
	n   testCommon.NodeConfig
}

func Setup(t *testing.T) TestMetadata {
	ret := TestMetadata{}
	ret.t = t
	ret.ctx = context.Background()
	node := testCommon.NewNodeConfig(
		t,
		testCommon.SingleRpc,
		[]string{"http://localhost:26657"},
		"../devnet/genesis",
	)
	ret.n = node
	return ret
}

func TestExternalTestSuite(t *testing.T) {
	if _, isIntegration := os.LookupEnv("INTEGRATION"); isIntegration == false {
		t.Skip("Skipping Integration Test Outside CI")
	}
	t.Log(">>> Setting up connection to local node <<<")
	m := Setup(t)
	t.Log(">>> Test Getting Chain Params <<<")
	GetParams(m)
	t.Log(">>> Test Topic Creation <<<")
	CreateTopic(m)
	t.Log(">>> Test Distribution Checks <<<")
	DistributionChecks(m)
	t.Log(">>> Test Actor Registration <<<")
	RegistrationChecks(m)
	t.Log(">>> Test Update Params <<<")
	UpdateParamsChecks(m)
	t.Log(">>> Test Reputer Staking <<<")
	StakingChecks(m)
	t.Log(">>> Test Topic Funding and Activation <<<")
	TopicFundingChecks(m)
	t.Log(">>> Test Making Inference <<<")
	WorkerInferenceAndForecastChecks(m)
}
