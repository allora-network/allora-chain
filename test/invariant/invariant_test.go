package invariant_test

import (
	"os"
	"strings"
	"testing"

	"context"

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

func TestInvariantTestSuite(t *testing.T) {
	if _, isInvariant := os.LookupEnv("INVARIANT_TEST"); isInvariant == false {
		t.Skip("Skipping Invariant Test unless explicitly enabled")
	}

	t.Log(">>> Environment <<<")
	seed := testcommon.LookupEnvInt(t, "SEED", 1)
	rpcMode := testcommon.LookupRpcMode(t, "RPC_MODE", testcommon.SingleRpc)
	rpcEndpoints := testcommon.LookupEnvStringArray("RPC_URLS", []string{"http://localhost:26657"})

	testConfig := testcommon.NewTestConfig(
		t,
		rpcMode,
		rpcEndpoints,
		"../devnet/genesis",
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
	simulate(
		&testConfig,
		maxIterations,
		numActors,
		epochLength,
		mode,
	)
}

// set up the common state for the simulator
// prior to either doing random simulation
// or manual simulation
func simulateSetUp(
	m *testcommon.TestConfig,
	numActors int,
	epochLength int,
	mode SimulationMode,
) (
	faucet Actor,
	simulationData *SimulationData,
) {
	// fund all actors from the faucet with some amount
	// give everybody the same amount of money to start with
	actorsList := createActors(m, numActors)
	faucet = Actor{
		name: getFaucetName(m.Seed),
		addr: m.FaucetAddr,
		acc:  m.FaucetAcc,
	}
	preFundAmount, err := getPreFundAmount(m, faucet, numActors)
	if err != nil {
		m.T.Fatal(err)
	}
	err = fundActors(
		m,
		faucet,
		actorsList,
		preFundAmount,
	)
	if err != nil {
		m.T.Fatal(err)
	}
	data := SimulationData{
		epochLength:        int64(epochLength),
		actors:             actorsList,
		counts:             StateTransitionCounts{},
		registeredWorkers:  testcommon.NewRandomKeyMap[Registration, struct{}](m.Client.Rand),
		registeredReputers: testcommon.NewRandomKeyMap[Registration, struct{}](m.Client.Rand),
		reputerStakes: testcommon.NewRandomKeyMap[Registration, cosmossdk_io_math.Int](
			m.Client.Rand,
		),
		delegatorStakes: testcommon.NewRandomKeyMap[Delegation, cosmossdk_io_math.Int](
			m.Client.Rand,
		),
		mode:      mode,
		failOnErr: false,
	}
	// if we're in manual mode or behaving mode we want to fail on errors
	if mode == Manual || mode == Behave {
		data.failOnErr = true
	}
	return faucet, &data
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

// this is the body of the "manual" simulation mode
// put your code here if you wish to manually send transactions
// in some specific order to test something
func simulateManual(
	m *testcommon.TestConfig,
	faucet Actor,
	simulationData *SimulationData,
) {
	iterLog(m.T, 0, "manual simulation mode")
	reputer := pickRandomActor(m, simulationData)
	delegator := pickRandomActorExcept(m, simulationData, []Actor{reputer})
	worker := pickRandomActorExcept(m, simulationData, []Actor{reputer, delegator})
	amount := cosmossdk_io_math.NewInt(1e10)

	// create topic
	createTopic(m, faucet, Actor{}, nil, 0, simulationData, 0)
	// register reputer
	registerReputer(m, reputer, Actor{}, nil, 1, simulationData, 1)
	// delegate from delegator on reputer
	delegateStake(m, delegator, reputer, &amount, 1, simulationData, 2)
	// fund the topic from delegator
	fundTopic(m, delegator, Actor{}, &amount, 1, simulationData, 5)
	// register worker
	registerWorker(m, worker, Actor{}, nil, 1, simulationData, 6)
	// now nobody has stake, is the topic active?
	// make sure an ABCI endblock has passed
	ctx := context.Background()
	m.Client.WaitForNextBlock(ctx)
	isActive := len(findActiveTopics(m, simulationData)) > 0
	m.T.Log("Is topic active?", isActive)
	doInferenceAndReputation(m, worker, reputer, &amount, 1, simulationData, 7)
	m.T.Log("Done.")
}

// this is the body of the "normal" simulation mode
func simulateAutomatic(
	m *testcommon.TestConfig,
	faucet Actor,
	data *SimulationData,
	maxIterations int,
) {
	// iteration 0, always create a topic to start
	createTopic(m, faucet, Actor{}, nil, 0, data, 0)

	// for every iteration
	// pick a state transition, then run it. every 5 print a summary
	// if the test mode is alternating, flip whether to behave nicely or not
	for iteration := 1; maxIterations == 0 || iteration < maxIterations; iteration++ {
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

// weight transitions that add registrations or stake, more heavily than those that take it away
// 70% of the time do additive stuff
// 30% of the time do subtractive stuff
func pickTransitionWithWeight(m *testcommon.TestConfig) StateTransition {
	transitionsAdditive, transitionsSubtractive := allTransitions()
	coinFlip := m.Client.Rand.Intn(10)
	if coinFlip < 7 {
		randIndex := m.Client.Rand.Intn(len(transitionsAdditive))
		stateTransition := transitionsAdditive[randIndex]
		return stateTransition
	} else {
		randIndex := m.Client.Rand.Intn(len(transitionsSubtractive))
		stateTransition := transitionsSubtractive[randIndex]
		return stateTransition
	}
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
