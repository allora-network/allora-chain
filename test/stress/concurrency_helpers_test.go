package stress_test

import (
	"fmt"
	"sync"
)

// convenience type aliases
type TOPIC_ID = uint64
type NAME = string

// global variables for the final summary report
var (
	workerErrors = make(map[TOPIC_ID][]struct {
		name NAME
		err  string
	})
	reputerErrors = make(map[TOPIC_ID][]struct {
		name NAME
		err  string
	})
	topicErrors   = make(map[TOPIC_ID]string)
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
func saveWorkerError(topic TOPIC_ID, name NAME, err error) {
	mutexWorkerErrors.Lock()
	workerErrors[topic] = append(workerErrors[topic], struct {
		name NAME
		err  string
	}{name, err.Error()})
	mutexWorkerErrors.Unlock()
}

// save a reputer error into the global map
func saveReputerError(topic TOPIC_ID, name NAME, err error) {
	mutexReputerErrors.Lock()
	reputerErrors[topic] = append(reputerErrors[topic], struct {
		name NAME
		err  string
	}{name, err.Error()})
	mutexReputerErrors.Unlock()
}

// save a topic error into the global map
func saveTopicError(topic TOPIC_ID, err error) {
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

// report the final summary statistics
func reportSummaryStatistics() {
	mutexTopicErrors.Lock()
	countTopicErrors := len(topicErrors)
	fmt.Print("Total topics with errors: ", countTopicErrors, " ")
	fmt.Println(topicErrors)
	mutexTopicErrors.Unlock()
	countReputersWithErrors := 0
	mutexReputerErrors.Lock()
	for topicId, topicReputerList := range reputerErrors {
		countReputersWithErrors++
		fmt.Print("Reputer Errors: Topic: ", topicId, " ")
		fmt.Println(topicReputerList)
	}
	mutexReputerErrors.Unlock()
	mutexWorkerErrors.Lock()
	countWorkersWithErrors := 0
	for topicId, topicWorkerList := range workerErrors {
		countWorkersWithErrors++
		fmt.Print("Worker Errors: Topic: ", topicId, " ")
		fmt.Println(topicWorkerList)
	}
	mutexWorkerErrors.Unlock()

	mutexCountTopics.Lock()
	mutexCountReputers.Lock()
	mutexCountWorkers.Lock()
	percentTopicsWithErrors := float64(countTopicErrors) / float64(countTopics) * 100
	percentReputersWithErrors := float64(countReputersWithErrors) / float64(countReputers) * 100
	percentWorkersWithErrors := float64(countWorkersWithErrors) / float64(countWorkers) * 100
	fmt.Printf("\n\nSummary Statistics:")
	fmt.Printf("Topics with errors: %d/%d | %.2f%%\n", countTopicErrors, countTopics, percentTopicsWithErrors)
	fmt.Printf("Reputers with errors: %d/%d | %.2f%%\n", countReputersWithErrors, countReputers, percentReputersWithErrors)
	fmt.Printf("Workers with errors: %d/%d  | %.2f%%\n", countWorkersWithErrors, countWorkers, percentWorkersWithErrors)
	mutexCountTopics.Unlock()
	mutexCountWorkers.Unlock()
	mutexCountReputers.Unlock()
}
