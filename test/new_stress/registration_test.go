package newstress_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

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
	batchSize := 100
	sem := make(chan struct{}, batchSize)
	
	ctx := context.Background()
	start := time.Now()
	completed := atomic.Int32{}

	fmt.Printf("Starting registration of %d workers in topic: %d\n", numWorkers, topicId)

	// Process in batches
	for batchStart := 0; batchStart < numWorkers; batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > numWorkers {
			batchEnd = numWorkers
		}

		var batchWg sync.WaitGroup
		fmt.Printf("Processing batch %d-%d\n", batchStart, batchEnd)

		// Process current batch
		for i := batchStart; i < batchEnd; i++ {
			batchWg.Add(1)
			worker := actors[i]
			
			go func(worker Actor, idx int) {
				defer batchWg.Done()
				
				sem <- struct{}{}
				defer func() { 
					<-sem 
					count := completed.Add(1)
					if int(count)%batchSize == 0 || count == int32(numWorkers) {
						elapsed := time.Since(start)
						fmt.Printf("Processed %d/%d worker registrations (%.2f%%) in %s\n", 
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
				
				fmt.Printf("Registering worker: %s in topic: %d with address: %s\n", worker.name, topicId, worker.addr)
				
				m.Client.BroadcastTxAsync(ctx, worker.acc, request)
				data.addWorkerRegistration(topicId, worker)
			}(worker, i)
		}

		// Wait for current batch to complete before starting next batch
		batchWg.Wait()
		fmt.Printf("Completed batch %d-%d\n", batchStart, batchEnd)
	}

	totalTime := time.Since(start)
	fmt.Printf("Total worker registration time: %s\n", totalTime)

	return nil
}

// registerReputers registers numReputers as reputers in topicId
func registerReputers(
	m *testcommon.TestConfig,
	actors []Actor,
	topicId uint64,
	data *SimulationData,
	numReputers int,
) error {
	batchSize := 100
	sem := make(chan struct{}, batchSize)
	
	ctx := context.Background()
	start := time.Now()
	completed := atomic.Int32{}

	fmt.Printf("Starting registration of %d reputers in topic: %d\n", numReputers, topicId)

	// Process in batches
	for batchStart := 0; batchStart < numReputers; batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > numReputers {
			batchEnd = numReputers
		}

		var batchWg sync.WaitGroup
		fmt.Printf("Processing batch %d-%d\n", batchStart, batchEnd)

		// Process current batch
		for i := batchStart; i < batchEnd; i++ {
			batchWg.Add(1)
			reputer := actors[i]
			
			go func(reputer Actor, idx int) {
				defer batchWg.Done()
				
				sem <- struct{}{}
				defer func() { 
					<-sem 
					count := completed.Add(1)
					if int(count)%batchSize == 0 || count == int32(numReputers) {
						elapsed := time.Since(start)
						fmt.Printf("Processed %d/%d reputer registrations (%.2f%%) in %s\n", 
							count, numReputers, 
							float64(count)/float64(numReputers)*100,
							elapsed)
					}
				}()

				request := &emissionstypes.RegisterRequest{
					Sender:    reputer.addr,
					Owner:     reputer.addr,
					IsReputer: true,
					TopicId:   topicId,
				}
				
				fmt.Printf("Registering reputer: %s in topic: %d with address: %s\n", reputer.name, topicId, reputer.addr)
				
				m.Client.BroadcastTxAsync(ctx, reputer.acc, request)
				data.addReputerRegistration(topicId, reputer)
			}(reputer, i)
		}

		// Wait for current batch to complete before starting next batch
		batchWg.Wait()
		fmt.Printf("Completed batch %d-%d\n", batchStart, batchEnd)
	}

	totalTime := time.Since(start)
	fmt.Printf("Total reputer registration time: %s\n", totalTime)

	return nil
}
