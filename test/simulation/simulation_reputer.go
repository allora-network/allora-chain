package simulation

import (
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"time"
)

const EpochLength = 100

func WorkRepute(
	seed int,
	m testCommon.TestConfig,
	topic *emissionstypes.Topic,
	inferers []testCommon.AccountAndAddress,
	forecasters []testCommon.AccountAndAddress,
	reputers []testCommon.AccountAndAddress,
) (int64, error) {
	insertedBlockHeight, err := insertWorkerBulk(m, topic, inferers, forecasters)
	if err != nil {
		return 0, err
	}
	blockHeight, err := insertReputerBulk(m, seed, topic, insertedBlockHeight, reputers)
	if err != nil {
		return 0, err
	}
	return blockHeight, nil
}

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
	topic, _ := getTopic(m, topicId)
	for index := 0; index < iteration; index++ {
		startIteration := time.Now()
		blockHeight, err := WorkRepute(seed, m, topic, inferers, forecasters, reputers)
		if err != nil {
			break
		}
		WorkReport(m, topicId, blockHeight, inferers, forecasters, reputers)
		elapsedIteration := time.Since(startIteration)
		sleepingTime := iterationTime - elapsedIteration
		time.Sleep(sleepingTime)
	}
}
