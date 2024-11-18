package stress_test

import (
	"sync"

	stresscommon "github.com/allora-network/allora-chain/test/stress/common"
)

type SimulationData struct {
	faucet                    SetupActor
	epochLength               int64
	actors                    []StressActor
	registeredWorkersByTopic  map[uint64][]StressActor
	registeredReputersByTopic map[uint64][]StressActor
	failOnErr                 bool
	mu                        sync.RWMutex
}

type Registration struct {
	TopicId uint64
	Actor   StressActor
}

// Add a worker registration to the simulation data
func (s *SimulationData) addWorkerRegistration(topicId uint64, actor StressActor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registeredWorkersByTopic[topicId] = append(s.registeredWorkersByTopic[topicId], actor)
}

// Add a reputer registration to the simulation data
func (s *SimulationData) addReputerRegistration(topicId uint64, actor StressActor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registeredReputersByTopic[topicId] = append(s.registeredReputersByTopic[topicId], actor)
}

// Get an actor object from an address
func (s *SimulationData) getActorFromAddr(addr string) (StressActor, bool) {
	for _, actor := range s.actors {
		if actor.addr == addr {
			return actor, true
		}
	}
	return StressActor{
		name:   "",
		addr:   "",
		params: stresscommon.TransactionParams{},
	}, false
}

// Get all workers for a topic
func (s *SimulationData) getWorkersForTopic(topicId uint64) []StressActor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.registeredWorkersByTopic[topicId]
}

// Get all reputers for a topic
func (s *SimulationData) getReputersForTopic(topicId uint64) []StressActor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.registeredReputersByTopic[topicId]
}
