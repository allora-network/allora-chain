package newstress_test

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	cosmosmath "cosmossdk.io/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// registerWorkers registers numWorkers as workers in topicId
func registerWorkers(
	m *testcommon.TestConfig,
	actors []Actor,
	topicId uint64,
	data *SimulationData,
	numWorkers int,
) error {
	maxConcurrent := 100
	sem := make(chan struct{}, maxConcurrent)

	ctx := context.Background()
	start := time.Now()
	completed := atomic.Int32{}

	var wg sync.WaitGroup
	m.T.Logf("Starting registration of %d workers in topic: %d\n", numWorkers, topicId)

	// Process all workers without batching
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		worker := actors[i]

		go func(worker Actor, idx int) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() {
				<-sem
				count := completed.Add(1)
				if int(count)%100 == 0 || count == int32(numWorkers) {
					elapsed := time.Since(start)
					m.T.Logf("Processed %d/%d worker registrations (%.2f%%) in %s\n",
						count, numWorkers,
						float64(count)/float64(numWorkers)*100,
						elapsed)
				}
			}()

			request := &emissionstypes.RegisterRequest{
				Sender:    worker.addr,
				Owner:     worker.addr,
				IsReputer: false,
				TopicId:   topicId,
			}

			m.T.Logf("Registering worker: %s in topic: %d with address: %s\n", worker.name, topicId, worker.addr)

			m.Client.BroadcastTxAsync(ctx, worker.acc, request)
			data.addWorkerRegistration(topicId, worker)
		}(worker, i)
	}

	wg.Wait()

	totalTime := time.Since(start)
	m.T.Logf("Total worker registration time: %s\n", totalTime)

	return nil
}

// registerReputersAndStake registers numReputers as reputers in topicId and stakes them
func registerReputersAndStake(
	m *testcommon.TestConfig,
	actors []Actor,
	topicId uint64,
	data *SimulationData,
	numReputers int,
) error {
	maxConcurrent := 100
	sem := make(chan struct{}, maxConcurrent)

	ctx := context.Background()
	start := time.Now()
	completed := atomic.Int32{}

	var wg sync.WaitGroup
	m.T.Logf("Starting registration of %d reputers in topic: %d\n", numReputers, topicId)

	// Process all reputers without batching
	for i := 0; i < numReputers; i++ {
		wg.Add(1)
		reputer := actors[i]

		go func(reputer Actor, idx int) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() {
				<-sem
				count := completed.Add(1)
				if int(count)%100 == 0 || count == int32(numReputers) {
					elapsed := time.Since(start)
					m.T.Logf("Processed %d/%d reputer registrations (%.2f%%) in %s\n",
						count, numReputers,
						float64(count)/float64(numReputers)*100,
						elapsed)
				}
			}()

			registerRequest := &emissionstypes.RegisterRequest{
				Sender:    reputer.addr,
				Owner:     reputer.addr,
				IsReputer: true,
				TopicId:   topicId,
			}

			stakeRequest := &emissionstypes.AddStakeRequest{
				Sender:  reputer.addr,
				TopicId: topicId,
				Amount:  cosmosmath.NewIntFromUint64(stakeToAdd),
			}

			m.T.Logf("Registering reputer: %s in topic: %d with address: %s\n", reputer.name, topicId, reputer.addr)

			m.Client.BroadcastTxAsync(ctx, reputer.acc, registerRequest, stakeRequest)
			data.addReputerRegistration(topicId, reputer)
		}(reputer, i)
	}

	wg.Wait()

	totalTime := time.Since(start)
	m.T.Logf("Total reputer registration time: %s\n", totalTime)

	return nil
}
