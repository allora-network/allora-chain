package invariant_test

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
) {
	wasErr := false
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
	}

	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, createTopicRequest)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		iterFailLog(m.T, iteration, actor, " failed to create topic")
		return
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	createTopicResponse := &emissionstypes.CreateNewTopicResponse{}
	err = txResp.Decode(createTopicResponse)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	if !wasErr {
		data.counts.incrementCreateTopicCount()
		iterSuccessLog(m.T, iteration, actor, " created topic ", createTopicResponse.TopicId)
		return
	} else {
		iterFailLog(m.T, iteration, actor, " failed to create topic")
		return
	}
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
) {
	wasErr := false
	iterLog(m.T, iteration, actor, "funding topic in amount ", amount)
	fundTopicRequest := &emissionstypes.FundTopicRequest{
		Sender:  actor.addr,
		TopicId: topicId,
		Amount:  *amount,
	}

	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, fundTopicRequest)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		iterFailLog(m.T, iteration, actor, " failed to fund topic ", topicId)
		return
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)

	if !wasErr {
		data.counts.incrementFundTopicCount()
		iterSuccessLog(m.T, iteration, actor, " funded topic ", topicId)
	} else {
		iterFailLog(m.T, iteration, actor, " failed to fund topic ", topicId)
	}
}
