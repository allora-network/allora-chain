package simulation

import (
	"fmt"
	testCommon "github.com/allora-network/allora-chain/test/common"
	"time"
)

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
	FormatReport(inferers, forecasters)
	for index := 0; index < iteration; index++ {
		topic, _ := getTopic(m, topicId)
		startIteration := time.Now()
		m.T.Log(fmt.Sprintf("[%v/%v] Calculating...", index+1, iteration))
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
		WorkReport(m, topicId, index+1, blockHeight, inferers, forecasters, reputers)
	}
}
