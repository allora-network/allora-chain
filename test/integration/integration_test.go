package integration_test

import (
	"os"
	"testing"

	testCommon "github.com/allora-network/allora-chain/test/common"
)

func TestExternalTestSuite(t *testing.T) {
	if _, isIntegration := os.LookupEnv("INTEGRATION"); isIntegration == false {
		t.Skip("Skipping Integration Test Outside CI")
	}
	t.Log(">>> Setting up connection to local node <<<")

	nodeConfig := testCommon.NewTestConfig(
		t,
		testCommon.SingleRpc,
		[]string{"http://localhost:26657"},
		"../devnet/genesis",
		0,
	)

	t.Log(">>> Test Getting Chain Params <<<")
	GetParams(nodeConfig)
	t.Log(">>> Test Topic Creation <<<")
	CreateTopic(nodeConfig)
	t.Log(">>> Test Distribution Checks <<<")
	DistributionChecks(nodeConfig)
	t.Log(">>> Test Actor Registration <<<")
	RegistrationChecks(nodeConfig)
	t.Log(">>> Test Update Params <<<")
	UpdateParamsChecks(nodeConfig)
	t.Log(">>> Test Reputer Staking <<<")
	StakingChecks(nodeConfig)
	t.Log(">>> Test Topic Funding and Activation <<<")
	TopicFundingChecks(nodeConfig)
	t.Log(">>> Test Making Inference <<<")
	WorkerInferenceAndForecastChecks(nodeConfig)
}
