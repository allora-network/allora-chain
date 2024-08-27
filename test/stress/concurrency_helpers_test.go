package stress_test

import (
	"sync"
	"testing"
)

// convenience type aliases
type TopicID = uint64
type Name = string

// global variables for the final summary report
var (
	workerErrors = make(map[TopicID][]struct {
		name Name
		err  string
	})
	reputerErrors = make(map[TopicID][]struct {
		name Name
		err  string
	})
	topicErrors   = make(map[TopicID]string)
	countTopics   = 0
	countWorkers  = 0
	countReputers = 0

	mutexWorkerErrors  sync.Mutex
	mutexReputerErrors sync.Mutex
	mutexTopicErrors   sync.Mutex
	mutexCountTopics   sync.Mutex
	mutexCountWorkers  sync.Mutex
	mutexCountReputers sync.Mutex
)

// save a worker error into the global map
func saveWorkerError(topic TopicID, name Name, err error) {
	mutexWorkerErrors.Lock()
	workerErrors[topic] = append(workerErrors[topic], struct {
		name Name
		err  string
	}{name, err.Error()})
	mutexWorkerErrors.Unlock()
}

// save a reputer error into the global map
func saveReputerError(topic TopicID, name Name, err error) {
	mutexReputerErrors.Lock()
	reputerErrors[topic] = append(reputerErrors[topic], struct {
		name Name
		err  string
	}{name, err.Error()})
	mutexReputerErrors.Unlock()
}

// save a topic error into the global map
func saveTopicError(topic TopicID, err error) {
	mutexTopicErrors.Lock()
	topicErrors[topic] = err.Error()
	mutexTopicErrors.Unlock()
}

// increment the number of topics
func incrementCountTopics() {
	mutexCountTopics.Lock()
	countTopics = countTopics + 1
	mutexCountTopics.Unlock()
}

// increment the total number of reputers
func incrementCountReputers() {
	mutexCountReputers.Lock()
	countReputers = countReputers + 1
	mutexCountReputers.Unlock()
}

// increment the total number of workers
func incrementCountWorkers() {
	mutexCountWorkers.Lock()
	countWorkers = countWorkers + 1
	mutexCountWorkers.Unlock()
}

func reportShortStatistics(t *testing.T) {
	t.Helper()
	mutexTopicErrors.Lock()
	countTopicErrors := len(topicErrors)
	mutexTopicErrors.Unlock()
	mutexReputerErrors.Lock()
	countReputersWithErrors := len(reputerErrors)
	mutexReputerErrors.Unlock()
	mutexWorkerErrors.Lock()
	countWorkersWithErrors := len(workerErrors)
	mutexWorkerErrors.Unlock()
	mutexCountTopics.Lock()
	countTopicsLocal := countTopics
	mutexCountTopics.Unlock()
	mutexCountReputers.Lock()
	countReputersLocal := countReputers
	mutexCountReputers.Unlock()
	mutexCountWorkers.Lock()
	countWorkersLocal := countWorkers
	mutexCountWorkers.Unlock()

	t.Logf("Topics with errors: %d/%d\n", countTopicErrors, countTopicsLocal)
	t.Logf("Reputers with errors: %d/%d\n", countReputersWithErrors, countReputersLocal)
	t.Logf("Workers with errors: %d/%d\n", countWorkersWithErrors, countWorkersLocal)
}

// report the final summary statistics
func reportSummaryStatistics(t *testing.T) {
	t.Helper()
	mutexTopicErrors.Lock()
	countTopicErrors := len(topicErrors)
	t.Logf("Total topics with errors: %d %v", countTopicErrors, topicErrors)
	mutexTopicErrors.Unlock()
	countReputersWithErrors := 0
	mutexReputerErrors.Lock()
	for topicId, topicReputerList := range reputerErrors {
		countReputersWithErrors++
		t.Logf("Reputer Errors: Topic: %d %v", topicId, topicReputerList)
	}
	mutexReputerErrors.Unlock()
	mutexWorkerErrors.Lock()
	countWorkersWithErrors := 0
	for topicId, topicWorkerList := range workerErrors {
		countWorkersWithErrors++
		t.Logf("Worker Errors: Topic: %d %v", topicId, topicWorkerList)
	}
	mutexWorkerErrors.Unlock()

	mutexCountTopics.Lock()
	mutexCountReputers.Lock()
	mutexCountWorkers.Lock()
	percentTopicsWithErrors := float64(countTopicErrors) / float64(countTopics) * 100
	percentReputersWithErrors := float64(countReputersWithErrors) / float64(countReputers) * 100
	percentWorkersWithErrors := float64(countWorkersWithErrors) / float64(countWorkers) * 100
	t.Logf("\n\nSummary Statistics:")
	t.Logf("Topics with errors: %d/%d | %.2f%%\n", countTopicErrors, countTopics, percentTopicsWithErrors)
	t.Logf("Reputers with errors: %d/%d | %.2f%%\n", countReputersWithErrors, countReputers, percentReputersWithErrors)
	t.Logf("Workers with errors: %d/%d  | %.2f%%\n", countWorkersWithErrors, countWorkers, percentWorkersWithErrors)
	mutexCountTopics.Unlock()
	mutexCountWorkers.Unlock()
	mutexCountReputers.Unlock()
}
