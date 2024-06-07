package simulation

import (
	testCommon "github.com/allora-network/allora-chain/test/common"
	"time"
)

const EpochLength = 5
const RetryTime = 3

func ReputeSimulation(
	m testCommon.TestConfig,
	seed int,
	iteration int,
	infererCount int,
	forecasterCount int,
	reputerCount int,
	topicId uint64,
) {
	approximateSecondsPerBlock := getApproximateBlockTimeSeconds(m)
	iterationTime := time.Duration(EpochLength) * approximateSecondsPerBlock
	inferers, forecasters, reputers := getActors(m, infererCount, forecasterCount, reputerCount)
	var prevLossHeight int64 = 0
	for index := 0; index < iteration; index++ {
		topic, _ := getTopic(m, topicId)
		startIteration := time.Now()
		insertedBlockHeight, err := insertWorkerBulk(m, topic, inferers, forecasters)
		if err != nil {
			continue
		}
		elapsedIteration := time.Since(startIteration)
		sleepingTime := iterationTime - elapsedIteration
		time.Sleep(sleepingTime)
		blockHeight, err := insertReputerBulk(m, seed, topic, insertedBlockHeight, prevLossHeight, reputers)
		if err != nil {
			continue
		}
		prevLossHeight = blockHeight
		WorkReport(m, topicId, blockHeight, inferers, forecasters, reputers)
	}
}
