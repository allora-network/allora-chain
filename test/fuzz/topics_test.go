package fuzz_test

import (
	"context"
	"fmt"

	cosmossdk_io_math "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// Use actor to create a new topic
func createTopic(
	m *testcommon.TestConfig,
	actor Actor,
	_ Actor,
	_ *cosmossdk_io_math.Int,
	_ uint64,
	data *SimulationData,
	iteration int,
) (success bool) {
	iterLog(m.T, iteration, actor, "creating new topic")
	createTopicRequest := &emissionstypes.CreateNewTopicRequest{
		Creator:                  actor.addr,
		Metadata:                 fmt.Sprintf("Created topic iteration %d", iteration),
		LossMethod:               "mse",
		EpochLength:              data.epochLength,
		GroundTruthLag:           data.epochLength,
		WorkerSubmissionWindow:   10,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
	}

	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, createTopicRequest)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, actor, "failed to create topic", "tx broadcast error", err)
		return false
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, actor, "failed to create topic", "tx wait error", err)
		return false
	}

	createTopicResponse := &emissionstypes.CreateNewTopicResponse{} // nolint:exhaustruct // the fields are populated by decode
	err = txResp.Decode(createTopicResponse)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, actor, "failed to create topic", "tx decode error", err)
		return false
	}

	data.counts.incrementCreateTopicCount()
	data.setTopicCreator(createTopicResponse.TopicId, actor)
	data.enableTopicWorkersWhitelist(createTopicResponse.TopicId)
	data.enableTopicReputersWhitelist(createTopicResponse.TopicId)
	iterSuccessLog(m.T, iteration, actor, "created topic", createTopicResponse.TopicId)
	return true
}

// use actor to fund topic, picked randomly
func fundTopic(
	m *testcommon.TestConfig,
	actor Actor,
	_ Actor,
	amount *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) (success bool) {
	iterLog(m.T, iteration, actor, "funding topic in amount", amount)
	fundTopicRequest := &emissionstypes.FundTopicRequest{
		Sender:  actor.addr,
		TopicId: topicId,
		Amount:  *amount,
	}

	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, fundTopicRequest)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, actor, "failed to fund topic", topicId, "tx broadcast error", err)
		return false
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, actor, "failed to fund topic", topicId, "tx wait error", err)
		return false
	}

	data.counts.incrementFundTopicCount()
	iterSuccessLog(m.T, iteration, actor, " funded topic ", topicId)
	return true
}
