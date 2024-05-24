package stress_test

import (
	"fmt"
	"sync"
)

type TOPIC_ID = uint64
type NAME = string

var (
	workerErrors = make(map[TOPIC_ID][]struct {
		name NAME
		err  error
	})
	reputerErrors = make(map[TOPIC_ID][]struct {
		name NAME
		err  error
	})
	topicErrors   = make(map[TOPIC_ID]error)
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

func saveWorkerError(topic TOPIC_ID, name NAME, err error) {
	mutexWorkerErrors.Lock()
	workerErrors[topic] = append(workerErrors[topic], struct {
		name NAME
		err  error
	}{name, err})
	mutexWorkerErrors.Unlock()
}

func saveReputerError(topic TOPIC_ID, name NAME, err error) {
	mutexReputerErrors.Lock()
	workerErrors[topic] = append(reputerErrors[topic], struct {
		name NAME
		err  error
	}{name, err})
	mutexReputerErrors.Unlock()
}

func saveTopicError(topic TOPIC_ID, err error) {
	mutexTopicErrors.Lock()
	topicErrors[topic] = err
	mutexTopicErrors.Unlock()
}

func incrementCountTopics() {
	mutexCountTopics.Lock()
	countTopics = countTopics + 1
	mutexCountTopics.Unlock()
}

func incrementCountReputers() {
	mutexCountReputers.Lock()
	countReputers = countReputers + 1
	mutexCountReputers.Unlock()
}

func incrementCountWorkers() {
	mutexCountWorkers.Lock()
	countWorkers = countWorkers + 1
	mutexCountWorkers.Unlock()
}

func reportSummaryStatistics() {
	mutexTopicErrors.Lock()
	countTopicErrors := len(topicErrors)
	fmt.Print("Total topics with errors: ", countTopicErrors, " ")
	fmt.Println(topicErrors)
	mutexTopicErrors.Unlock()
	countReputersWithErrors := 0
	mutexReputerErrors.Lock()
	for topicId, topicReputerList := range reputerErrors {
		countReputersWithErrors += len(topicReputerList)
		fmt.Print("Reputer Errors: Topic: ", topicId, " ")
		fmt.Println(topicReputerList)
	}
	mutexReputerErrors.Unlock()
	mutexWorkerErrors.Lock()
	countWorkersWithErrors := 0
	for topicId, topicWorkerList := range workerErrors {
		countWorkersWithErrors += len(topicWorkerList)
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
	fmt.Printf("Count of topics with errors: %d\n", countTopicErrors)
	fmt.Printf("Count of reputers with errors: %d\n", countReputersWithErrors)
	fmt.Printf("Count of workers with errors: %d\n", countWorkersWithErrors)
	mutexCountTopics.Unlock()
	mutexCountWorkers.Unlock()
	mutexCountReputers.Unlock()

	fmt.Printf("\n\nSummary Statistics:")
	fmt.Printf("Percent of topics with some error: %.2f%%\n", percentTopicsWithErrors)
	fmt.Printf("Percent of reputers with some error: %.2f%%\n", percentReputersWithErrors)
	fmt.Printf("Percent of workers with some error: %.2f%%\n", percentWorkersWithErrors)
}
