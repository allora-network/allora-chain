package fuzz_test

import (
	"os"
	"strings"
	"testing"

	"fmt"
	"time"

	cosmossdk_io_math "cosmossdk.io/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
)

type SimulationMode string

const (
	Behave    SimulationMode = "behave"
	Fuzz      SimulationMode = "fuzz"
	Alternate SimulationMode = "alternate"
	Manual    SimulationMode = "manual"
)

func lookupEnvSimulationMode() SimulationMode {
	simulationModeStr, found := os.LookupEnv("MODE")
	if !found {
		return Behave
	}
	simulationModeStr = strings.ToLower(simulationModeStr)
	switch simulationModeStr {
	case "behave":
		return Behave
	case "fuzz":
		return Fuzz
	case "alternate":
		return Alternate
	case "manual":
		return Manual
	default:
		return Behave
	}
}

func TestFuzzTestSuite(t *testing.T) {
	if _, isFuzz := os.LookupEnv("FUZZ_TEST"); isFuzz == false {
		t.Skip("Skipping Fuzz Test unless explicitly enabled")
	}

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
	numActors := testcommon.LookupEnvInt(t, "NUM_ACTORS", 100)
	epochLength := testcommon.LookupEnvInt(t, "EPOCH_LENGTH", 12) // in blocks
	mode := lookupEnvSimulationMode()

	t.Log("Max Actors: ", numActors)
	t.Log("Max Iterations: ", maxIterations)
	t.Log("Epoch Length: ", epochLength)
	t.Log("Simulation mode: ", mode)

	t.Log(">>> Starting Test <<<")
	timestr := fmt.Sprintf(">>> Starting %s <<<", time.Now().Format(time.RFC850))
	t.Log(timestr)

	simulate(
		&testConfig,
		maxIterations,
		numActors,
		epochLength,
		mode,
	)

	timestr = fmt.Sprintf(">>> Complete %s <<<", time.Now().Format(time.RFC850))
	t.Log(timestr)
}

// run the outer loop of the simulator
func simulate(
	m *testcommon.TestConfig,
	maxIterations int,
	numActors int,
	epochLength int,
	mode SimulationMode,
) {
	faucet, simulationData := simulateSetUp(m, numActors, epochLength, mode)
	if mode == Manual {
		simulateManual(m, faucet, simulationData)
	} else {
		simulateAutomatic(m, faucet, simulationData, maxIterations)
	}
}

// Note: this code never runs unless you're in manual mode
// body of the "manual" simulation mode
// put your code here if you wish to manually send transactions
// in some specific order to test something
func simulateManual(
	m *testcommon.TestConfig,
	faucet Actor,
	data *SimulationData,
) {
	iterLog(m.T, 0, "manual simulation mode")
	reputer := pickRandomActor(m, data)
	delegator := pickRandomActorExcept(m, data, []Actor{reputer})
	worker := pickRandomActorExcept(m, data, []Actor{reputer, delegator})
	amount := cosmossdk_io_math.NewInt(1e10)

	// create topic
	createTopic(m, faucet, UnusedActor, nil, 0, data, 0)
	// register reputer
	registerReputer(m, reputer, UnusedActor, nil, 1, data, 1)
	// delegate from delegator on reputer
	delegateStake(m, delegator, reputer, &amount, 1, data, 2)
	// fund the topic from delegator
	fundTopic(m, delegator, UnusedActor, &amount, 1, data, 5)
	// register worker
	registerWorker(m, worker, UnusedActor, nil, 1, data, 6)
	// now nobody has stake, is the topic active?
	// make sure an ABCI endblock has passed
	collectDelegatorRewards(m, delegator, reputer, nil, 1, data, 7)
	doInferenceAndReputation(m, UnusedActor, UnusedActor, nil, 1, data, 8)
	collectDelegatorRewards(m, delegator, reputer, nil, 1, data, 9)
	collectDelegatorRewards(m, delegator, reputer, nil, 1, data, 10)
	collectDelegatorRewards(m, delegator, reputer, nil, 1, data, 11)
	collectDelegatorRewards(m, delegator, reputer, nil, 1, data, 12)
	collectDelegatorRewards(m, delegator, reputer, nil, 1, data, 13)
	collectDelegatorRewards(m, delegator, reputer, nil, 1, data, 14)
	doInferenceAndReputation(m, UnusedActor, UnusedActor, nil, 1, data, 15)
	amount2 := amount.QuoRaw(2)
	undelegateStake(m, delegator, reputer, &amount2, 1, data, 16)
	m.T.Log("Done.")
}

// this is the body of the "normal" simulation mode
func simulateAutomatic(
	m *testcommon.TestConfig,
	faucet Actor,
	data *SimulationData,
	maxIterations int,
) {
	// start with some initial state so we have something to work with in the test
	iterationCountInitialState := simulateAutomaticInitialState(m, faucet, data)

	m.T.Log("Initial State Summary:", data.counts)
	m.T.Log("Starting post-setup iterations, first non-setup fuzz iteration is ", iterationCountInitialState)

	// for every iteration
	// pick a state transition, then run it. every 5 print a summary
	// if the test mode is alternating, flip whether to behave nicely or not
	maxIterations = maxIterations + iterationCountInitialState
	for iteration := iterationCountInitialState; maxIterations == 0 || iteration < maxIterations; iteration++ {
		if data.mode == Alternate {
			data.randomlyFlipFailOnErr(m, iteration)
		}
		stateTransition, actor1, actor2, amount, topicId := pickTransition(m, data, iteration)
		stateTransition.f(m, actor1, actor2, amount, topicId, data, iteration)
		if iteration%5 == 0 {
			m.T.Log("State Transitions Summary:", data.counts)
		}
	}
	m.T.Log("Final Summary:", data.counts)
}

// for every iteration
// pick a state transition to try
// check that that state transition even makes sense based on what we know
// try to pick some actors and a topic id that will work for this transition
// if errors at any point, pick a new state transition to try
func pickTransition(
	m *testcommon.TestConfig,
	data *SimulationData,
	iteration int,
) (stateTransition StateTransition, actor1, actor2 Actor, amount *cosmossdk_io_math.Int, topicId uint64) {
	for {
		stateTransition := pickTransitionWithWeight(m)
		canOccur := canTransitionOccur(m, data, stateTransition)
		if data.failOnErr && !canOccur {
			iterLog(m.T, iteration, "Transition not possible: ", stateTransition.name)
			continue
		}
		couldPickActors, actor1, actor2, amount, topicId := pickActorAndTopicIdForStateTransition(
			m,
			stateTransition,
			data,
		)
		if data.failOnErr && !couldPickActors {
			iterLog(m.T, iteration, "Could not pick actors for transition: ", stateTransition.name)
			continue
		}
		if data.failOnErr && !isValidTransition(m, stateTransition, actor1, actor2, amount, topicId, data, iteration) {
			iterLog(m.T, iteration, "Invalid state transition: ", stateTransition.name)
			continue
		}
		// if we're straight up fuzzing, then pick some randos and yolo it
		if !data.failOnErr {
			_, actor1, actor2, amount, topicId = pickFullRandomValues(m, data)
		}
		return stateTransition, actor1, actor2, amount, topicId
	}
}
