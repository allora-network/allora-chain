package simulation

import (
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"sync"
	"time"
)

const EpochLength = 100

func getActors(
	m testCommon.TestConfig,
	infererCount int,
	forecasterCount int,
	reputerCount int,
) ([]testCommon.AccountAndAddress, []testCommon.AccountAndAddress, []testCommon.AccountAndAddress) {
	inferers := make([]testCommon.AccountAndAddress, 0)
	forecasters := make([]testCommon.AccountAndAddress, 0)
	reputers := make([]testCommon.AccountAndAddress, 0)

	for index := 0; index < infererCount; index++ {
		accountName := getActorsAccountName(INFERER_TYPE, m.Seed, index)
		inferer, err := getActorAccountAndAddress(m, accountName)
		if err != nil {
			continue
		}
		inferers = append(inferers, inferer)
	}

	for index := 0; index < forecasterCount; index++ {
		accountName := getActorsAccountName(FORECASTER_TYPE, m.Seed, index)
		inferer, err := getActorAccountAndAddress(m, accountName)
		if err != nil {
			continue
		}
		inferers = append(inferers, inferer)
	}

	for index := 0; index < reputerCount; index++ {
		accountName := getActorsAccountName(REPUTER_TYPE, m.Seed, index)
		inferer, err := getActorAccountAndAddress(m, accountName)
		if err != nil {
			continue
		}
		inferers = append(inferers, inferer)
	}

	return inferers, forecasters, reputers
}

func getTopic(
	m testCommon.TestConfig,
	topicId uint64,
) (*emissionstypes.Topic, error) {
	topicResponse, err := m.Client.QueryEmissions().GetTopic(
		m.Ctx,
		&emissionstypes.QueryTopicRequest{TopicId: topicId},
	)
	if err != nil {
		return nil, err
	}
	return topicResponse.Topic, nil
}
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
	infererCount int,
	forecasterCount int,
	reputerCount int,
	iteration int,
	topicId uint64,
) {
	approximateSecondsPerBlock := getApproximateBlockTimeSeconds(m)
	iterationTime := time.Duration(EpochLength) * approximateSecondsPerBlock
	inferers, forecasters, reputers := getActors(m, infererCount, forecasterCount, reputerCount)
	topic, _ := getTopic(m, topicId)
	var wg sync.WaitGroup
	for index := 0; index < iteration; index++ {
		startIteration := time.Now()
		wg.Add(1)
		blockHeight, _ := WorkRepute(seed, m, topic, inferers, forecasters, reputers)
		WorkReport(m, topicId, blockHeight, inferers, forecasters, reputers)
		elapsedIteration := time.Since(startIteration)
		sleepingTime := iterationTime - elapsedIteration
		time.Sleep(sleepingTime)
	}
	wg.Wait()
}
