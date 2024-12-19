package fuzz_test

import (
	"cmp"
	"fmt"
	"math/rand"
	"slices"

	cosmossdk_io_math "cosmossdk.io/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
	fuzzcommon "github.com/allora-network/allora-chain/test/fuzz/common"
)

// SimulationData stores the active set of states we think we're in
// so that we can choose to take a transition that is valid
// right now it doesn't need mutexes, if we parallelize this test ever it will
// to read and write out of the simulation data
type SimulationData struct {
	epochLength                   int64
	actors                        []Actor
	counts                        StateTransitionCounts
	registeredWorkers             *testcommon.RandomKeyMap[Registration, struct{}]
	registeredReputers            *testcommon.RandomKeyMap[Registration, struct{}]
	reputerStakes                 *testcommon.RandomKeyMap[Registration, struct{}]
	delegatorStakes               *testcommon.RandomKeyMap[Delegation, struct{}]
	topicCreators                 *testcommon.RandomKeyMap[uint64, Actor]
	adminWhitelist                *testcommon.RandomKeyMap[Actor, struct{}]
	globalWhitelist               *testcommon.RandomKeyMap[Actor, struct{}]
	topicCreatorsWhitelist        *testcommon.RandomKeyMap[Actor, struct{}]
	topicWorkersWhitelistEnabled  *testcommon.RandomKeyMap[uint64, struct{}]
	topicReputersWhitelistEnabled *testcommon.RandomKeyMap[uint64, struct{}]
	topicWorkersWhitelist         *testcommon.RandomKeyMap[TopicWhitelistEntry, struct{}]
	topicReputersWhitelist        *testcommon.RandomKeyMap[TopicWhitelistEntry, struct{}]

	failOnErr bool
	mode      fuzzcommon.SimulationMode
}

// String is the stringer for SimulationData
func (s *SimulationData) String() string {
	return fmt.Sprintf(
		"SimulationData{\nepochLength: %d,\nactors: %v,\n counts: %s,\nregisteredWorkers: %v,\nregisteredReputers: %v,\nreputerStakes: %v,\ndelegatorStakes: %v,\nfailOnErr: %v,\nmode: %s}",
		s.epochLength,
		s.actors,
		s.counts,
		s.registeredWorkers,
		s.registeredReputers,
		s.reputerStakes,
		s.delegatorStakes,
		s.failOnErr,
		s.mode,
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

type TopicWhitelistEntry struct {
	TopicId uint64
	Actor   Actor
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

// addReputerStaked adds a reputer stake to the list of staked reputers in the simulation data
func (s *SimulationData) addReputerStaked(topicId uint64, actor Actor) {
	s.reputerStakes.Upsert(Registration{
		TopicId: topicId,
		Actor:   actor,
	}, struct{}{})
}

// addDelegatorDelegated adds a delegator stake to the list of staked delegators in the simulation data
func (s *SimulationData) addDelegatorDelegated(topicId uint64, delegator Actor, reputer Actor) {
	s.delegatorStakes.Upsert(Delegation{
		TopicId:   topicId,
		Delegator: delegator,
		Reputer:   reputer,
	}, struct{}{})
}

// removeReputerRegistration removes a reputer registration from the simulation data
func (s *SimulationData) removeReputerRegistration(topicId uint64, actor Actor) {
	s.registeredReputers.Delete(Registration{
		TopicId: topicId,
		Actor:   actor,
	})
}

// removeReputerStaked removes a reputer stake from the list of staked reputers in the simulation data
func (s *SimulationData) removeReputerStaked(topicId uint64, actor Actor) {
	s.reputerStakes.Delete(Registration{
		TopicId: topicId,
		Actor:   actor,
	})
}

// removeDelegatorDelegated removes a delegator stake from the list of staked delegators in the simulation data
func (s *SimulationData) removeDelegatorDelegated(topicId uint64, delegator Actor, reputer Actor) {
	s.delegatorStakes.Delete(Delegation{
		TopicId:   topicId,
		Delegator: delegator,
		Reputer:   reputer,
	})
}

// setTopicCreator sets the topic's creator in the simulation data
func (s *SimulationData) setTopicCreator(topicId uint64, actor Actor) {
	s.topicCreators.Upsert(topicId, actor)
}

// addAdminWhitelist adds an actor to the admin whitelist in the simulation data
func (s *SimulationData) addAdminWhitelist(actor Actor) {
	s.adminWhitelist.Upsert(actor, struct{}{})
}

// addGlobalWhitelist adds an actor to the global whitelist in the simulation data
func (s *SimulationData) addGlobalWhitelist(actor Actor) {
	s.globalWhitelist.Upsert(actor, struct{}{})
}

// addTopicCreatorWhitelist adds an actor to the topic creator whitelist in the simulation data
func (s *SimulationData) addTopicCreatorWhitelist(actor Actor) {
	s.topicCreatorsWhitelist.Upsert(actor, struct{}{})
}

// addTopicWorkerWhitelist adds a worker to a topic whitelist in the simulation data
func (s *SimulationData) addTopicWorkerWhitelist(topicId uint64, worker Actor) {
	s.topicWorkersWhitelist.Upsert(TopicWhitelistEntry{
		TopicId: topicId,
		Actor:   worker,
	}, struct{}{})
}

// addTopicReputerWhitelist adds a reputer to a topic whitelist in the simulation data
func (s *SimulationData) addTopicReputerWhitelist(topicId uint64, reputer Actor) {
	s.topicReputersWhitelist.Upsert(TopicWhitelistEntry{
		TopicId: topicId,
		Actor:   reputer,
	}, struct{}{})
}

// enableTopicWorkersWhitelist enables a topic workers whitelist in the simulation data
func (s *SimulationData) enableTopicWorkersWhitelist(topicId uint64) {
	s.topicWorkersWhitelistEnabled.Upsert(topicId, struct{}{})
}

// enableTopicReputersWhitelist enables a topic reputers whitelist in the simulation data
func (s *SimulationData) enableTopicReputersWhitelist(topicId uint64) {
	s.topicReputersWhitelistEnabled.Upsert(topicId, struct{}{})
}

// removeAdminWhitelist removes an actor to the admin whitelist in the simulation data
func (s *SimulationData) removeAdminWhitelist(actor Actor) {
	s.adminWhitelist.Delete(actor)
}

// removeGlobalWhitelist removes an actor to the global whitelist in the simulation data
func (s *SimulationData) removeGlobalWhitelist(actor Actor) {
	s.globalWhitelist.Delete(actor)
}

// removeTopicCreatorWhitelist removes an actor to the topic creator whitelist in the simulation data
func (s *SimulationData) removeTopicCreatorWhitelist(actor Actor) {
	s.topicCreatorsWhitelist.Delete(actor)
}

// removeTopicWorkerWhitelist removes a worker to a topic whitelist in the simulation data
func (s *SimulationData) removeTopicWorkerWhitelist(topicId uint64, worker Actor) {
	s.topicWorkersWhitelist.Delete(TopicWhitelistEntry{
		TopicId: topicId,
		Actor:   worker,
	})
}

// removeTopicReputerWhitelist removes a reputer to a topic whitelist in the simulation data
func (s *SimulationData) removeTopicReputerWhitelist(topicId uint64, reputer Actor) {
	s.topicReputersWhitelist.Delete(TopicWhitelistEntry{
		TopicId: topicId,
		Actor:   reputer,
	})
}

// disableTopicWorkersWhitelist disables a topic workers whitelist in the simulation data
func (s *SimulationData) disableTopicWorkersWhitelist(topicId uint64) {
	s.topicWorkersWhitelistEnabled.Delete(topicId)
}

// disableTopicReputersWhitelist disables a topic reputers whitelist in the simulation data
func (s *SimulationData) disableTopicReputersWhitelist(topicId uint64) {
	s.topicReputersWhitelistEnabled.Delete(topicId)
}

// getTopics returns all the topics in the simulation data
func (s *SimulationData) getTopics() []uint64 {
	return s.topicCreators.GetKeys()
}

// pickRandomTopicId picks a random topic id in the simulation data
func (s *SimulationData) pickRandomTopicId() (uint64, error) {
	t, err := s.topicCreators.RandomKey()
	if err != nil {
		return 0, err
	}
	return *t, nil
}

// pickNRandomActors picks n random actors in the simulation data
// panics if n is negative or less than the number of actors
func (s *SimulationData) pickNRandomActors(m *testcommon.TestConfig, n int) []Actor {
	if n < 0 || n > len(s.actors) {
		panic("invalid number of actors")
	}

	shuffled := make([]Actor, n)
	perm := m.Client.Rand.Perm(len(s.actors))
	for i := range n {
		shuffled[i] = s.actors[perm[i]]
	}

	return shuffled
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
	actor, topicId, err := s.pickRandomRegisteredReputer()
	if err != nil {
		return Actor{}, 0, err
	}
	reg := Registration{
		TopicId: topicId,
		Actor:   actor,
	}
	_, exists := s.reputerStakes.Get(reg)
	if !exists {
		return Actor{}, 0, fmt.Errorf("Registered reputer %s is not staked", actor.addr)
	}
	return actor, topicId, nil
}

// pickRandomDelegator picks a random delegator that is currently staked
func (s *SimulationData) pickRandomStakedDelegator() (Actor, Actor, uint64, error) {
	ret, err := s.delegatorStakes.RandomKey()
	if err != nil {
		return Actor{}, Actor{}, 0, err
	}

	if !s.isReputerRegisteredInTopic(ret.TopicId, ret.Reputer) {
		return Actor{}, Actor{}, 0, fmt.Errorf(
			"Delegator %s is staked in reputer %s, but reputer is not registered",
			ret.Delegator.addr,
			ret.Reputer.addr,
		)
	}

	return ret.Delegator, ret.Reputer, ret.TopicId, nil
}

// take a percentage of the stake, either 1/10, 1/3, 1/2, 6/7, or the full amount
func pickPercentOf(rand *rand.Rand, stake cosmossdk_io_math.Int) cosmossdk_io_math.Int {
	if stake.Equal(cosmossdk_io_math.ZeroInt()) {
		return cosmossdk_io_math.ZeroInt()
	}
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

func (s *SimulationData) pickRandomAdmin() (Actor, error) {
	a, err := s.adminWhitelist.RandomKey()
	if err != nil {
		return Actor{}, err
	}
	return *a, nil
}

func (s *SimulationData) pickRandomTopicAdmin(m *testcommon.TestConfig, topicId uint64) (Actor, error) {
	opt1, _ := s.adminWhitelist.RandomKey()
	var opt2 *Actor
	if opt, ok := s.topicCreators.Get(topicId); ok {
		opt2 = &opt
	}
	actor, err := mayPickOneOfTwo(m.Client.Rand, opt1, opt2)
	if err != nil {
		return Actor{}, fmt.Errorf("no topic creator found")
	}
	return *actor, nil
}

func (s *SimulationData) pickRandomTopicCreator(m *testcommon.TestConfig) (Actor, error) {
	opt1, _ := s.globalWhitelist.RandomKey()
	opt2, _ := s.topicCreatorsWhitelist.RandomKey()
	actor, err := mayPickOneOfTwo(m.Client.Rand, opt1, opt2)
	if err != nil {
		return Actor{}, fmt.Errorf("no topic creator found")
	}
	return *actor, nil
}

func mayPickOneOfTwo[T any](rand *rand.Rand, opt1, opt2 *T) (*T, error) {
	if opt1 == nil && opt2 == nil {
		return nil, fmt.Errorf("both options are nil")
	}
	if opt1 == nil {
		return opt2, nil
	}
	if opt2 == nil {
		return opt1, nil
	}
	// if neither are nil, pick one at random
	if rand.Intn(2) == 0 {
		return opt1, nil
	}
	return opt2, nil
}

func (s *SimulationData) hasTopic(topicId uint64) bool {
	_, exists := s.topicCreators.Get(topicId)
	return exists
}

// isWorkerRegisteredInTopic checks if a worker is registered in a topic
func (s *SimulationData) isWorkerRegisteredInTopic(topicId uint64, actor Actor) bool {
	_, exists := s.registeredWorkers.Get(Registration{
		TopicId: topicId,
		Actor:   actor,
	})
	return exists
}

// isReputerRegisteredInTopic checks if a reputer is registered
func (s *SimulationData) isReputerRegisteredInTopic(topicId uint64, actor Actor) bool {
	_, exists := s.registeredReputers.Get(Registration{
		TopicId: topicId,
		Actor:   actor,
	})
	return exists
}

// isActorInGlobalWhitelist checks if an actor is in the global whitelist
func (s *SimulationData) isActorInGlobalWhitelist(actor Actor) bool {
	_, exists := s.globalWhitelist.Get(actor)
	return exists
}

// isWorkerWhitelistedInTopic checks if a worker is whitelisted in a topic
func (s *SimulationData) isWorkerWhitelistedInTopic(topicId uint64, actor Actor) bool {
	if _, exists := s.topicWorkersWhitelistEnabled.Get(topicId); !exists {
		return true
	}

	_, exists := s.topicWorkersWhitelist.Get(TopicWhitelistEntry{
		TopicId: topicId,
		Actor:   actor,
	})

	return exists || s.isActorInGlobalWhitelist(actor)
}

// isReputerWhitelistedInTopic checks if a worker is whitelisted in a topic
func (s *SimulationData) isReputerWhitelistedInTopic(topicId uint64, actor Actor) bool {
	if _, exists := s.topicReputersWhitelistEnabled.Get(topicId); !exists {
		return true
	}

	_, exists := s.topicReputersWhitelist.Get(TopicWhitelistEntry{
		TopicId: topicId,
		Actor:   actor,
	})

	return exists || s.isActorInGlobalWhitelist(actor)
}

// isAnyWorkerRegisteredInTopic checks if any worker is registered and whitelisted in a topic
func (s *SimulationData) isAnyWorkerRegisteredInTopic(topicId uint64) bool {
	workers, _ := s.registeredWorkers.Filter(func(reg Registration) bool {
		if reg.TopicId != topicId {
			return false
		}
		return s.isWorkerWhitelistedInTopic(topicId, reg.Actor)
	})
	return len(workers) > 0
}

// isAnyReputerRegisteredInTopic checks if any reputer is registered and whitelisted in a topic
func (s *SimulationData) isAnyReputerRegisteredInTopic(topicId uint64) bool {
	reputers, _ := s.registeredReputers.Filter(func(reg Registration) bool {
		if reg.TopicId != topicId {
			return false
		}
		return s.isReputerWhitelistedInTopic(topicId, reg.Actor)
	})
	return len(reputers) > 0
}

// get all workers for a topic, this function is iterates over the list of workers multiple times
// for determinism, the workers are sorted by their address
func (s *SimulationData) getWorkersForTopic(topicId uint64) []Actor {
	workers, _ := s.registeredWorkers.Filter(func(reg Registration) bool {
		if reg.TopicId != topicId {
			return false
		}
		return s.isWorkerWhitelistedInTopic(topicId, reg.Actor)
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
		if reg.TopicId != topicId {
			return false
		}
		return s.isReputerWhitelistedInTopic(topicId, reg.Actor)
	})
	rmap := make(map[string]Actor)
	for _, reputerReg := range reputerRegs {
		rmap[reputerReg.Actor.addr] = reputerReg.Actor
	}
	reputerDels, _ := s.delegatorStakes.Filter(func(del Delegation) bool {
		if del.TopicId != topicId {
			return false
		}
		return s.isReputerWhitelistedInTopic(topicId, del.Reputer)
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

// randomly flip the fail on err case to decide whether to be aggressive and fuzzy or
// behaved state transitions
func (s *SimulationData) randomlyFlipFailOnErr(f *fuzzcommon.FuzzConfig, iteration int) {
	flip := f.TestConfig.Client.Rand.Intn(100)
	// f.AlternateWeight % likelihood to flip the failOnErr mode
	if flip < f.AlternateWeight {
		iterLog(f.TestConfig.T, iteration, "Changing fuzzer mode: failOnErr changing from", s.failOnErr, "to", !s.failOnErr)
		s.failOnErr = !s.failOnErr
	}
}
