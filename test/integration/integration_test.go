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

	seed := testCommon.LookupEnvInt(t, "SEED", 0)
	rpcMode := testCommon.LookupRpcMode(t, "RPC_MODE", testCommon.SingleRpc)
	rpcEndpoints := testCommon.LookupEnvStringArray("RPC_URLS", []string{"http://localhost:26657"})

	testConfig := testCommon.NewTestConfig(
		t,
		rpcMode,
		rpcEndpoints,
		"../devnet/genesis",
		seed,
	)

	t.Log(">>> Test Getting Chain Params <<<")
	GetParams(testConfig)
	t.Log(">>> Test Update Params <<<")
	UpdateParamsChecks(testConfig)
	t.Log(">>> Test Topic Creation <<<")
	CreateTopic(testConfig)
	t.Log(">>> Test Distribution Checks <<<")
	DistributionChecks(testConfig)
	t.Log(">>> Test Actor Registration <<<")
	RegistrationChecks(testConfig)
	t.Log(">>> Test Reputer Staking <<<")
	StakingChecks(testConfig)
	t.Log(">>> Test Topic Funding and Activation <<<")
	TopicFundingChecks(testConfig)
	t.Log(">>> Test Making Inference <<<")
	WorkerInferenceAndForecastChecks(testConfig)
	t.Log(">>> Test Reputer Un-Staking <<<")
	UnstakingChecks(testConfig)
}

func TestUpgradeTestSuite(t *testing.T) {
	if _, runUpgradeChecks := os.LookupEnv("UPGRADE"); runUpgradeChecks == false {
		t.Skip("Skipping Upgrade Test Outside CI")
	}
	t.Log(">>> Setting up connection to local node <<<")

	seed := testCommon.LookupEnvInt(t, "SEED", 0)
	rpcMode := testCommon.LookupRpcMode(t, "RPC_MODE", testCommon.SingleRpc)
	rpcEndpoints := testCommon.LookupEnvStringArray("RPC_URLS", []string{"http://localhost:26657"})

	testConfig := testCommon.NewTestConfig(
		t,
		rpcMode,
		rpcEndpoints,
		"../devnet/genesis",
		seed,
	)

	t.Log(">>> Test Upgrading Emissions Module Version")
	UpgradeChecks(testConfig)
}
