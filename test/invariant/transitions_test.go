package invariant_test

import (
	"fmt"

	cosmossdk_io_math "cosmossdk.io/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
)

// Every function responsible for doing a state transition
// should adhere to this function signature
type StateTransitionFunc func(
	m *testcommon.TestConfig,
	actor1 Actor,
	actor2 Actor,
	amount *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
)

// keep track of the name of the state transition as well as the function
type StateTransition struct {
	name string
	f    StateTransitionFunc
}

// The list of possible state transitions we can take are:
//
// create a new topic,
// fund a topic some more,
// register as a reputer,
// register as a worker,
// unregister as a reputer,
// unregister as a worker,
// stake as a reputer,
// stake in a reputer (delegate),
// unstake as a reputer,
// unstake from a reputer (undelegate),
// cancel the removal of stake (as a reputer),
// cancel the removal of delegated stake (delegator),
// collect delegator rewards,
// produce an inference (insert a bulk worker payload),
// produce reputation scores (insert a bulk reputer payload)
func allTransitions() ([]StateTransition, []StateTransition) {
	// return additive state transitions first
	// and subtractive state transitions second
	return []StateTransition{
			{"createTopic", createTopic},
			{"fundTopic", fundTopic},
			{"registerWorker", registerWorker},
			{"registerReputer", registerReputer},
			{"stakeAsReputer", stakeAsReputer},
			{"delegateStake", delegateStake},
			{"collectDelegatorRewards", collectDelegatorRewards},
			{"doInferenceAndReputation", doInferenceAndReputation},
		}, []StateTransition{
			{"unregisterWorker", unregisterWorker},
			{"unregisterReputer", unregisterReputer},
			{"unstakeAsReputer", unstakeAsReputer},
			{"undelegateStake", undelegateStake},
			{"cancelStakeRemoval", cancelStakeRemoval},
			{"cancelDelegateStakeRemoval", cancelDelegateStakeRemoval},
		}
}

// state machine dependencies for valid transitions
//
// fundTopic: CreateTopic
// RegisterWorkerForTopic: CreateTopic
// RegisterReputerForTopic: CreateTopic
// unRegisterReputer: RegisterReputerForTopic
// unRegisterWorker: RegisterWorkerForTopic
// stakeReputer: RegisterReputerForTopic, CreateTopic
// delegateStake: CreateTopic, RegisterReputerForTopic
// unstakeReputer: stakeReputer
// unstakeDelegator: delegateStake
// cancelStakeRemoval: unstakeReputer
// cancelDelegateStakeRemoval: unstakeDelegator
// collectDelegatorRewards: delegateStake, fundTopic, InsertBulkWorkerPayload, InsertBulkReputerPayload
// InsertBulkWorkerPayload: RegisterWorkerForTopic, FundTopic
// InsertBulkReputerPayload: RegisterReputerForTopic, InsertBulkWorkerPayload
func canTransitionOccur(m *testcommon.TestConfig, data *SimulationData, transition StateTransition) bool {
	switch transition.name {
	case "unregisterWorker":
		return anyWorkersRegistered(data)
	case "unregisterReputer":
		return anyReputersRegistered(data)
	case "stakeAsReputer":
		return anyReputersRegistered(data)
	case "delegateStake":
		return anyReputersRegistered(data)
	case "unstakeAsReputer":
		return anyReputersStaked(data)
	case "undelegateStake":
		return anyDelegatorsStaked(data)
	case "collectDelegatorRewards":
		return anyDelegatorsStaked(data) && anyReputersRegistered(data)
	case "cancelStakeRemoval":
		// too expensive to do this twice
		// figure this out in picking step
		return true
	case "cancelDelegateStakeRemoval":
		// too expensive to do this twice
		// figure this out in picking step
		return true
	case "doInferenceAndReputation":
		activeTopics := findActiveTopics(m, data)
		for i := 0; i < len(activeTopics); i++ {
			workerExists := data.isAnyWorkerRegisteredInTopic(activeTopics[i].Id)
			reputerExists := data.isAnyReputerRegisteredInTopic(activeTopics[i].Id)
			if workerExists && reputerExists {
				return true
			}
		}
		return false

	default:
		return true
	}
}

// is this specific combination of actors, amount, and topicId valid for the transition?
func isValidTransition(m *testcommon.TestConfig, transition StateTransition, actor1 Actor, actor2 Actor, amount *cosmossdk_io_math.Int, topicId uint64, data *SimulationData, iteration int) bool {
	switch transition.name {
	case "collectDelegatorRewards":
		// if the reputer unregisters before the delegator withdraws stake, it can be invalid for a
		// validator to collecte rewards
		if !data.isReputerRegistered(topicId, actor2) {
			iterLog(m.T, iteration, "Transition not valid: ", transition.name, actor1, actor2, amount, topicId)
			return false
		}
		return true
	default:
		return true
	}
}

// pickRandomActor picks a random actor from the list of actors in the simulation data
func pickRandomActor(m *testcommon.TestConfig, data *SimulationData) Actor {
	return data.actors[m.Client.Rand.Intn(len(data.actors))]
}

// pickRandomActorExcept picks a random actor from the list of actors in the simulation data
// and panics if it can't find one after 5 tries that is not the same as the given actors
func pickRandomActorExcept(m *testcommon.TestConfig, data *SimulationData, actors []Actor) Actor {
	count := 0
	for ; count < 5; count++ {
		randomActor := pickRandomActor(m, data)
		match := false
		for _, actor := range actors {
			if randomActor == actor {
				match = true
			}
		}
		if !match {
			return randomActor
		}
	}
	panic(
		fmt.Sprintf(
			"could not find a random actor that is not the same as the given actor after %d tries",
			count,
		),
	)
}

// helper for when the transition values can be fully fully random
func pickFullRandomValues(
	m *testcommon.TestConfig,
	data *SimulationData,
) (bool, Actor, Actor, *cosmossdk_io_math.Int, uint64) {
	randomTopicId, err := pickRandomTopicId(m)
	requireNoError(m.T, data.failOnErr, err)
	randomActor1 := pickRandomActor(m, data)
	randomActor2 := pickRandomActor(m, data)
	amount, err := pickRandomBalanceLessThanHalf(m, randomActor1)
	requireNoError(m.T, data.failOnErr, err)
	return true, randomActor1, randomActor2, &amount, randomTopicId
}

// pickActorAndTopicIdForStateTransition picks random actors
// able to take the state transition and returns which one it picked.
// if the transition requires only one actor (the majority) the second is empty
func pickActorAndTopicIdForStateTransition(
	m *testcommon.TestConfig,
	transition StateTransition,
	data *SimulationData,
) (success bool, actor1 Actor, actor2 Actor, amount *cosmossdk_io_math.Int, topicId uint64) {
	switch transition.name {
	case "unregisterWorker":
		worker, topicId, err := data.pickRandomRegisteredWorker()
		if err != nil {
			return false, Actor{}, Actor{}, nil, 0
		}
		return true, worker, Actor{}, nil, topicId
	case "unregisterReputer":
		reputer, topicId, err := data.pickRandomRegisteredReputer()
		if err != nil {
			return false, Actor{}, Actor{}, nil, 0
		}
		return true, reputer, Actor{}, nil, topicId
	case "stakeAsReputer":
		reputer, topicId, err := data.pickRandomRegisteredReputer()
		if err != nil {
			return false, Actor{}, Actor{}, nil, 0
		}
		amount, err := pickRandomBalanceLessThanHalf(m, reputer) // if err amount=zero which is a valid transition
		requireNoError(m.T, data.failOnErr, err)
		return true, reputer, Actor{}, &amount, topicId
	case "delegateStake":
		reputer, topicId, err := data.pickRandomRegisteredReputer()
		if err != nil {
			return false, Actor{}, Actor{}, nil, 0
		}
		delegator := pickRandomActorExcept(m, data, []Actor{reputer})
		amount, err := pickRandomBalanceLessThanHalf(m, delegator)
		requireNoError(m.T, data.failOnErr, err)
		return true, delegator, reputer, &amount, topicId
	case "unstakeAsReputer":
		reputer, topicId, err := data.pickRandomStakedReputer()
		if err != nil {
			return false, Actor{}, Actor{}, nil, 0
		}
		amount := data.pickPercentOfStakeByReputer(m.Client.Rand, topicId, reputer)
		return true, reputer, Actor{}, &amount, topicId
	case "undelegateStake":
		delegator, reputer, topicId, err := data.pickRandomStakedDelegator()
		if err != nil {
			return false, Actor{}, Actor{}, nil, 0
		}
		amount := data.pickPercentOfStakeByDelegator(m.Client.Rand, topicId, delegator, reputer)
		return true, delegator, reputer, &amount, topicId
	case "collectDelegatorRewards":
		delegator, reputer, topicId, err := data.pickRandomStakedDelegator()
		if err != nil {
			return false, Actor{}, Actor{}, nil, 0
		}
		return true, delegator, reputer, nil, topicId
	case "cancelStakeRemoval":
		stakeRemoval, found, err := findFirstValidStakeRemovalFromChain(m)
		if err != nil || !found {
			return false, Actor{}, Actor{}, nil, 0
		}
		reputer, found := data.getActorFromAddr(stakeRemoval.Reputer)
		if !found {
			return false, Actor{}, Actor{}, nil, 0
		}
		return true, reputer, Actor{}, &stakeRemoval.Amount, stakeRemoval.TopicId
	case "cancelDelegateStakeRemoval":
		stakeRemoval, found, err := findFirstValidDelegateStakeRemovalFromChain(m)
		if err != nil || !found {
			return false, Actor{}, Actor{}, nil, 0
		}
		delegator, found := data.getActorFromAddr(stakeRemoval.Delegator)
		if !found {
			return false, Actor{}, Actor{}, nil, 0
		}
		reputer, found := data.getActorFromAddr(stakeRemoval.Reputer)
		if !found {
			return false, Actor{}, Actor{}, nil, 0
		}
		return true, delegator, reputer, &stakeRemoval.Amount, stakeRemoval.TopicId
	case "doInferenceAndReputation":
		topics := findActiveTopics(m, data)
		if len(topics) > 0 {
			for i := 0; i < 10; i++ {
				randIndex := m.Client.Rand.Intn(len(topics))
				topicId := topics[randIndex].Id
				worker, err := data.pickRandomWorkerRegisteredInTopic(m.Client.Rand, topicId)
				if err != nil {
					continue
				}
				reputer, err := data.pickRandomReputerRegisteredInTopic(m.Client.Rand, topicId)
				if err != nil {
					continue
				}
				return true, worker, reputer, nil, topicId
			}
		}
		return false, Actor{}, Actor{}, nil, 0
	default:
		return pickFullRandomValues(m, data)
	}
}
