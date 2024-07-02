package invariant_test

import (
	"cmp"
	"fmt"
	"math/rand"
	"slices"

	cosmossdk_io_math "cosmossdk.io/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
)

// SimulationData stores the active set of states we think we're in
// so that we can choose to take a transition that is valid
// right now it doesn't need mutexes, if we parallelize this test ever it will
// to read and write out of the simulation data
type SimulationData struct {
	epochLength        int64
	actors             []Actor
	counts             StateTransitionCounts
	registeredWorkers  *testcommon.RandomKeyMap[Registration, struct{}]
	registeredReputers *testcommon.RandomKeyMap[Registration, struct{}]
	reputerStakes      *testcommon.RandomKeyMap[Registration, cosmossdk_io_math.Int]
	delegatorStakes    *testcommon.RandomKeyMap[Delegation, cosmossdk_io_math.Int]
	failOnErr          bool
}

// String is the stringer for SimulationData
func (s *SimulationData) String() string {
	return fmt.Sprintf(
		"SimulationData{\nepochLength: %d,\nactors: %v,\n counts: %s,\nregisteredWorkers: %v,\nregisteredReputers: %v,\nreputerStakes: %v,\ndelegatorStakes: %v,\nnoFail: %v}",
		s.epochLength,
		s.actors,
		s.counts,
		s.registeredWorkers,
		s.registeredReputers,
		s.reputerStakes,
		s.delegatorStakes,
		s.failOnErr,
	)
}

type Registration struct {
	TopicId uint64
	Actor   Actor
}

type Delegation struct {
	TopicId   uint64
	Delegator Actor
	Reputer   Actor
}

// addWorkerRegistration adds a worker registration to the simulation data
func (s *SimulationData) addWorkerRegistration(topicId uint64, actor Actor) {
	s.registeredWorkers.Upsert(Registration{
		TopicId: topicId,
		Actor:   actor,
	}, struct{}{})
}

// removeWorkerRegistration removes a worker registration from the simulation data
func (s *SimulationData) removeWorkerRegistration(topicId uint64, actor Actor) {
	s.registeredWorkers.Delete(Registration{
		TopicId: topicId,
		Actor:   actor,
	})
}

// addReputerRegistration adds a reputer registration to the simulation data
func (s *SimulationData) addReputerRegistration(topicId uint64, actor Actor) {
	s.registeredReputers.Upsert(Registration{
		TopicId: topicId,
		Actor:   actor,
	}, struct{}{})
}

// removeReputerRegistration removes a reputer registration from the simulation data
func (s *SimulationData) removeReputerRegistration(topicId uint64, actor Actor) {
	s.registeredReputers.Delete(Registration{
		TopicId: topicId,
		Actor:   actor,
	})
}

// pickRandomRegisteredWorker picks a random worker that is currently registered
func (s *SimulationData) pickRandomRegisteredWorker() (Actor, uint64, error) {
	ret, err := s.registeredWorkers.RandomKey()
	if err != nil {
		return Actor{}, 0, err
	}
	return ret.Actor, ret.TopicId, nil
}

// pickRandomRegisteredReputer picks a random reputer that is currently registered
func (s *SimulationData) pickRandomRegisteredReputer() (Actor, uint64, error) {
	ret, err := s.registeredReputers.RandomKey()
	if err != nil {
		return Actor{}, 0, err
	}
	return ret.Actor, ret.TopicId, nil
}

// pickRandomStakedReputer picks a random reputer that is currently staked
func (s *SimulationData) pickRandomStakedReputer() (Actor, uint64, error) {
	ret, err := s.reputerStakes.RandomKey()
	if err != nil {
		return Actor{}, 0, err
	}
	return ret.Actor, ret.TopicId, nil
}

// pickRandomDelegator picks a random delegator that is currently staked
func (s *SimulationData) pickRandomStakedDelegator() (Actor, Actor, uint64, error) {
	ret, err := s.delegatorStakes.RandomKey()
	if err != nil {
		return Actor{}, Actor{}, 0, err
	}
	return ret.Delegator, ret.Reputer, ret.TopicId, nil
}

// addReputerStake adds a reputer stake to the simulation data
func (s *SimulationData) addReputerStake(
	topicId uint64,
	actor Actor,
	amount cosmossdk_io_math.Int,
) {
	reg := Registration{
		TopicId: topicId,
		Actor:   actor,
	}
	prevStake, exists := s.reputerStakes.Get(reg)
	if !exists {
		prevStake = cosmossdk_io_math.ZeroInt()
	}
	newValue := prevStake.Add(amount)
	s.reputerStakes.Upsert(reg, newValue)
}

// markStakeRemovalReputerStake marks a reputer stake for removal in the simulation data
// rather than try to keep copy of such complex state, we just pretend removals are instant
func (s *SimulationData) markStakeRemovalReputerStake(
	topicId uint64,
	actor Actor,
	amount *cosmossdk_io_math.Int,
) {
	reg := Registration{
		TopicId: topicId,
		Actor:   actor,
	}
	prevStake, exists := s.reputerStakes.Get(reg)
	if !exists {
		prevStake = cosmossdk_io_math.ZeroInt()
	}
	newValue := prevStake.Sub(*amount)
	if newValue.IsNegative() {
		panic(
			fmt.Sprintf(
				"negative stake disallowed, topic id %d actor %s amount %s",
				topicId,
				actor,
				amount,
			),
		)
	}
	if !newValue.IsZero() {
		s.reputerStakes.Upsert(reg, newValue)
	} else {
		s.reputerStakes.Delete(reg)
	}
}

// markStakeRemovalDelegatorStake marks a delegator stake for removal in the simulation data
func (s *SimulationData) markStakeRemovalDelegatorStake(
	topicId uint64,
	delegator Actor,
	reputer Actor,
	amount *cosmossdk_io_math.Int,
) {
	del := Delegation{
		TopicId:   topicId,
		Delegator: delegator,
		Reputer:   reputer,
	}
	prevStake, exists := s.delegatorStakes.Get(del)
	if !exists {
		prevStake = cosmossdk_io_math.ZeroInt()
	}
	newValue := prevStake.Sub(*amount)
	if newValue.IsNegative() {
		panic(
			fmt.Sprintf(
				"negative stake disallowed, topic id %d delegator %s reputer %s amount %s",
				topicId,
				delegator,
				reputer,
				amount,
			),
		)
	}
	if !newValue.IsZero() {
		s.delegatorStakes.Upsert(del, newValue)
	} else {
		s.delegatorStakes.Delete(del)
	}
}

// take a percentage of the stake, either 1/10, 1/3, 1/2, 6/7, or the full amount
func pickPercentOf(rand *rand.Rand, stake cosmossdk_io_math.Int) cosmossdk_io_math.Int {
	percent := rand.Intn(5)
	switch percent {
	case 0:
		return stake.QuoRaw(10)
	case 1:
		return stake.QuoRaw(3)
	case 2:
		return stake.QuoRaw(2)
	case 3:
		return stake.MulRaw(6).QuoRaw(7)
	default:
		return stake
	}
}

// pickPercentOfStakeByReputer picks a random percent (1/10, 1/3, 1/2, 6/7, or full amount) of the stake by a reputer
func (s *SimulationData) pickPercentOfStakeByReputer(
	rand *rand.Rand,
	topicId uint64,
	actor Actor,
) cosmossdk_io_math.Int {
	reg := Registration{
		TopicId: topicId,
		Actor:   actor,
	}
	stake, exists := s.reputerStakes.Get(reg)
	if !exists {
		return cosmossdk_io_math.ZeroInt()
	}
	return pickPercentOf(rand, stake)
}

// pick a random percent (1/10, 1/3, 1/2, 6/7, or full amount) of the stake that a delegator has in a reputer
func (s *SimulationData) pickPercentOfStakeByDelegator(
	rand *rand.Rand,
	topicId uint64,
	delegator Actor,
	reputer Actor,
) cosmossdk_io_math.Int {
	del := Delegation{
		TopicId:   topicId,
		Delegator: delegator,
		Reputer:   reputer,
	}
	stake, exists := s.delegatorStakes.Get(del)
	if !exists {
		return cosmossdk_io_math.ZeroInt()
	}
	return pickPercentOf(rand, stake)

}

// addDelegatorStake adds a delegator stake to the simulation data
func (s *SimulationData) addDelegatorStake(
	topicId uint64,
	delegator Actor,
	reputer Actor,
	amount cosmossdk_io_math.Int,
) {
	delegation := Delegation{
		TopicId:   topicId,
		Delegator: delegator,
		Reputer:   reputer,
	}
	prevStake, exists := s.delegatorStakes.Get(delegation)
	if !exists {
		prevStake = cosmossdk_io_math.ZeroInt()
	}
	newValue := prevStake.Add(amount)
	s.delegatorStakes.Upsert(delegation, newValue)
}

// isReputerRegistered checks if a reputer is registered
func (s *SimulationData) isReputerRegistered(topicId uint64, actor Actor) bool {
	_, exists := s.registeredReputers.Get(Registration{
		TopicId: topicId,
		Actor:   actor,
	})
	return exists
}

// pick a random worker from a topic. This function is O(n) over the list of workers
func (s *SimulationData) pickRandomWorkerRegisteredInTopic(rand *rand.Rand, topicId uint64) (Actor, error) {
	workers, _ := s.registeredWorkers.Filter(func(reg Registration) bool {
		return reg.TopicId == topicId
	})
	if len(workers) == 0 {
		return Actor{}, fmt.Errorf("no workers in topic %d", topicId)
	}
	randIndex := rand.Intn(len(workers))
	return workers[randIndex].Actor, nil
}

// pick a random reputer registered in a topic. This function is O(n) over the list of reputers
func (s *SimulationData) pickRandomReputerRegisteredInTopic(rand *rand.Rand, topicId uint64) (Actor, error) {
	reputers, _ := s.registeredReputers.Filter(func(reg Registration) bool {
		return reg.TopicId == topicId
	})
	if len(reputers) == 0 {
		return Actor{}, fmt.Errorf("no reputers in topic %d", topicId)

	}
	randIndex := rand.Intn(len(reputers))
	return reputers[randIndex].Actor, nil
}

// isAnyWorkerRegisteredInTopic checks if any worker is registered in a topic
func (s *SimulationData) isAnyWorkerRegisteredInTopic(topicId uint64) bool {
	workers, _ := s.registeredWorkers.Filter(func(reg Registration) bool {
		return reg.TopicId == topicId
	})
	return len(workers) > 0
}

// isAnyReputerRegisteredInTopic checks if any reputer is registered in a topic
func (s *SimulationData) isAnyReputerRegisteredInTopic(topicId uint64) bool {
	reputers, _ := s.registeredReputers.Filter(func(reg Registration) bool {
		return reg.TopicId == topicId
	})
	return len(reputers) > 0
}

// get all workers for a topic, this function is iterates over the list of workers multiple times
// for determinism, the workers are sorted by their address
func (s *SimulationData) getWorkersForTopic(topicId uint64) []Actor {
	workers, _ := s.registeredWorkers.Filter(func(reg Registration) bool {
		return reg.TopicId == topicId
	})
	ret := make([]Actor, len(workers))
	for i, worker := range workers {
		ret[i] = worker.Actor
	}
	slices.SortFunc(ret, func(a, b Actor) int {
		return cmp.Compare(a.addr, b.addr)
	})
	return ret
}

// get all reputers with nonzero stake for a topic, this function is iterates over the list of reputers multiple times
// for determinism, the reputers are sorted by their address
func (s *SimulationData) getReputersForTopicWithStake(topicId uint64) []Actor {
	reputerRegs, _ := s.reputerStakes.Filter(func(reg Registration) bool {
		return reg.TopicId == topicId
	})
	rmap := make(map[string]Actor)
	for _, reputerReg := range reputerRegs {
		rmap[reputerReg.Actor.addr] = reputerReg.Actor
	}
	reputerDels, _ := s.delegatorStakes.Filter(func(del Delegation) bool {
		return del.TopicId == topicId
	})
	for _, del := range reputerDels {
		rmap[del.Reputer.addr] = del.Reputer
	}
	ret := make([]Actor, 0)
	for _, reputer := range rmap {
		ret = append(ret, reputer)
	}
	slices.SortFunc(ret, func(a, b Actor) int {
		return cmp.Compare(a.addr, b.addr)
	})
	return ret
}

// get an actor object from an address
func (s *SimulationData) getActorFromAddr(addr string) (Actor, bool) {
	for _, actor := range s.actors {
		if actor.addr == addr {
			return actor, true
		}
	}
	return Actor{}, false
}
