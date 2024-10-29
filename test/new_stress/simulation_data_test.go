package newstress_test

import (
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
)

type SimulationData struct {
	epochLength               int64
	actors                    []Actor
	registeredWorkersByTopic  map[uint64][]string
	registeredReputersByTopic map[uint64][]string
	failOnErr                 bool
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
	s.registeredWorkersByTopic[topicId] = append(s.registeredWorkersByTopic[topicId], actor.addr)
}

// addReputerRegistration adds a reputer registration to the simulation data
func (s *SimulationData) addReputerRegistration(topicId uint64, actor Actor) {
	s.registeredReputersByTopic[topicId] = append(s.registeredReputersByTopic[topicId], actor.addr)
}

// get an actor object from an address
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
