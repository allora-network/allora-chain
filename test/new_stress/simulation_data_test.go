package newstress_test

import (
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"sync"
)

type SimulationData struct {
	epochLength               int64
	actors                    []Actor
	registeredWorkersByTopic  map[uint64][]Actor
	registeredReputersByTopic map[uint64][]Actor
	failOnErr                 bool
	mu                        sync.RWMutex
}

type Registration struct {
	TopicId uint64
	Actor   Actor
}

// Add a worker registration to the simulation data
func (s *SimulationData) addWorkerRegistration(topicId uint64, actor Actor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registeredWorkersByTopic[topicId] = append(s.registeredWorkersByTopic[topicId], actor)
}

// Add a reputer registration to the simulation data
func (s *SimulationData) addReputerRegistration(topicId uint64, actor Actor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registeredReputersByTopic[topicId] = append(s.registeredReputersByTopic[topicId], actor)
}

// Get an actor object from an address
func (s *SimulationData) getActorFromAddr(addr string) (Actor, bool) {
	for _, actor := range s.actors {
		if actor.addr == addr {
			return actor, true
		}
	}
	return Actor{
		name: "",
		addr: "",
		acc:  cosmosaccount.Account{Name: "", Record: nil},
	}, false
}

// Get all workers for a topic
func (s *SimulationData) getWorkersForTopic(topicId uint64) []Actor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.registeredWorkersByTopic[topicId]
}

// Get all reputers for a topic
func (s *SimulationData) getReputersForTopic(topicId uint64) []Actor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.registeredReputersByTopic[topicId]
}
