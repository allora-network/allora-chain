package simulation

import (
	testCommon "github.com/allora-network/allora-chain/test/common"
	"os"
	"testing"
)

func TestSimulationSuite(t *testing.T) {
	if _, isSimulation := os.LookupEnv("SIMULATION_TEST"); isSimulation == false {
		t.Skip("Skipping Simulation Test unless explicitly enabled")
	}

	t.Log(">>> Setting up connection to local node <<<")

	seed := testCommon.LookupEnvInt(t, "SEED", 0)
	rpcMode := testCommon.LookupRpcMode(t, "RPC_MODE", testCommon.SingleRpc)
	rpcEndpoints := testCommon.LookupEnvStringArray("RPC_URLS", []string{"http://localhost:26657"})

	inferersCount := testCommon.LookupEnvInt(t, "INFERERS_COUNT", 5)
	forecastersCount := testCommon.LookupEnvInt(t, "FORECASTER_COUNT", 3)
	reputersCount := testCommon.LookupEnvInt(t, "REPUTERS_COUNT", 5)
	iterationCount := testCommon.LookupEnvInt(t, "ITERATION_COUNT", 10)
	testConfig := testCommon.NewTestConfig(
		t,
		rpcMode,
		rpcEndpoints,
		"../devnet/genesis",
		seed,
	)

	t.Log(">>> Setup Topic <<<")
	topicId := SetupTopic(testConfig)
	t.Log(">>> Generate Actors <<<")
	GenerateActors(testConfig, inferersCount, forecastersCount, reputersCount)
	t.Log(">>> Register and Stake Topic <<<")
	RegisterAndStakeTopic(testConfig, inferersCount, forecastersCount, reputersCount, topicId)
	t.Log(">>> Repute Simulation <<<")
	ReputeSimulation(testConfig, seed, iterationCount, inferersCount, forecastersCount, reputersCount, topicId)
}
