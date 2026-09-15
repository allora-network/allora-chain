package fuzz_test

import (
	"os"
	"testing"

	"fmt"
	"time"

	cosmossdk_io_math "cosmossdk.io/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
	fuzzcommon "github.com/allora-network/allora-chain/test/fuzz/common"
)

func TestFuzzTestSuite(t *testing.T) {
	if _, isFuzz := os.LookupEnv("FUZZ_TEST"); isFuzz == false {
		t.Skip("Skipping Fuzz Test unless explicitly enabled")
	}

	fuzzConfig := fuzzcommon.GetFuzzConfig(t)

	t.Log(">>> Environment <<<")

	t.Log("Max Actors: ", fuzzConfig.NumActors)
	if fuzzConfig.MaxIterations == 0 {
		t.Log("Max Iterations:0, will continue forever until interrupted")
	} else {
		t.Log("Max Iterations: ", fuzzConfig.MaxIterations)
	}
	t.Log("Epoch Length: ", fuzzConfig.EpochLength)
	t.Log("Simulation mode: ", fuzzConfig.Mode)
	if fuzzConfig.Mode == fuzzcommon.Alternate {
		t.Log("Alternate Weight Percentage: ", fuzzConfig.AlternateWeight)
	}
	t.Log("Seed: ", fuzzConfig.Seed)

	t.Log(">>> Starting Test <<<")
	timestr := fmt.Sprintf(">>> Starting %s <<<", time.Now().Format(time.RFC850))
	t.Log(timestr)

	simulate(&fuzzConfig)

	timestr = fmt.Sprintf(">>> Complete %s <<<", time.Now().Format(time.RFC850))
	t.Log(timestr)
}

// run the outer loop of the simulator
func simulate(f *fuzzcommon.FuzzConfig) {
	faucet, simulationData := simulateSetUp(
		f.TestConfig,
		f.NumActors,
		f.EpochLength,
		f.Mode,
		f.Seed,
	)
	if f.Mode == fuzzcommon.Manual {
		simulateManual(f.TestConfig, faucet, simulationData)
	} else {
		simulateAutomatic(f, faucet, simulationData)
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
func simulateAutomatic(f *fuzzcommon.FuzzConfig, faucet Actor, data *SimulationData) {
	// start with some initial state so we have something to work with in the test
	iterationCountInitialState := simulateAutomaticInitialState(f, faucet, data)

	f.TestConfig.T.Log("Initial State Summary:", data.counts)
	f.TestConfig.T.Log("Starting post-setup iterations, first non-setup fuzz iteration is ", iterationCountInitialState)

	// for every iteration
	// pick a state transition, then run it. every 5 print a summary
	// if the test mode is alternating, flip whether to behave nicely or not
	infiniteMode := f.MaxIterations == 0
	maxIterations := f.MaxIterations + iterationCountInitialState
	var followTransition *StateTransition = nil
	stateTransition := StateTransition{
		name:         "",
		f:            nil,
		weight:       0,
		follow:       nil,
		followWeight: 0,
	}
	actor1, actor2 := UnusedActor, UnusedActor
	var amount *cosmossdk_io_math.Int = nil
	var topicId uint64 = 0
	for iteration := iterationCountInitialState; infiniteMode || iteration < maxIterations; iteration++ {
		// This is a follow-on transition, do it with the same actors values and topic id as the previous iteration
		if followTransition != nil {
			followTransition.f(f.TestConfig, actor1, actor2, amount, topicId, data, iteration)
			followTransition = nil
		} else { // This is not a follow-on transition, pick new actors and topic id
			if data.mode == fuzzcommon.Alternate {
				data.randomlyFlipFailOnErr(f, iteration)
			}
			stateTransition, actor1, actor2, amount, topicId = pickTransition(f, data, iteration)
			stateTransition.f(f.TestConfig, actor1, actor2, amount, topicId, data, iteration)

			// if this state transition has a follow-on transition, decide whether to do it or not
			followTransition = pickFollowOnTransitionWithWeight(f.TestConfig, stateTransition)
		}
		if iteration%5 == 0 {
			f.TestConfig.T.Log("State Transitions Summary:", data.counts)
		}
	}
	f.TestConfig.T.Log("Final Summary:", data.counts)
}

// for every iteration
// pick a state transition to try
// check that that state transition even makes sense based on what we know
// try to pick some actors and a topic id that will work for this transition
// if errors at any point, pick a new state transition to try
func pickTransition(
	f *fuzzcommon.FuzzConfig,
	data *SimulationData,
	iteration int,
) (stateTransition StateTransition, actor1, actor2 Actor, amount *cosmossdk_io_math.Int, topicId uint64) {
	for {
		stateTransition := pickTransitionWithWeight(f)
		canOccur := canTransitionOccur(f.TestConfig, data, stateTransition)
		if data.failOnErr && !canOccur {
			iterLog(f.TestConfig.T, iteration, "Transition not possible: ", stateTransition.name)
			continue
		}
		couldPickActors, actor1, actor2, amount, topicId := pickActorAndTopicIdForStateTransition(
			f.TestConfig,
			stateTransition,
			data,
			iteration,
		)
		if data.failOnErr && !couldPickActors {
			iterLog(f.TestConfig.T, iteration, "Could not pick actors for transition: ", stateTransition.name)
			continue
		}
		if data.failOnErr && !isValidTransition(f.TestConfig, stateTransition, actor1, actor2, amount, topicId, data, iteration) {
			iterLog(f.TestConfig.T, iteration, "Invalid state transition: ", stateTransition.name)
			continue
		}
		// if we're straight up fuzzing, then pick some randos and yolo it
		if !data.failOnErr {
			_, actor1, actor2, amount, topicId = pickFullRandomValues(f.TestConfig, data)
		}
		return stateTransition, actor1, actor2, amount, topicId
	}
}
