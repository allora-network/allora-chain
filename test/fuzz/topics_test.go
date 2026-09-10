package fuzz_test

import (
	"context"
	"encoding/hex"
	"fmt"

	cosmossdk_io_math "cosmossdk.io/math"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	alloraMath "github.com/allora-network/allora-chain/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// newCreateTopicRequest builds the topic creation message every setup and fuzz topic uses.
func newCreateTopicRequest(actor Actor, epochLength int64, iteration int) *emissionstypes.CreateNewTopicRequest {
	return &emissionstypes.CreateNewTopicRequest{
		MaxTopInferersToReward:   0,
		Creator:                  actor.addr,
		Metadata:                 fmt.Sprintf("Created topic iteration %d", iteration),
		LossMethod:               "mse",
		EpochLength:              epochLength,
		GroundTruthLag:           epochLength,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		WorkerSubmissionWindow:   10,
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.05"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.05"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.05"),
		EnableWorkerWhitelist:    true,
		EnableReputerWhitelist:   true,
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
		TopicType:                emissionstypes.TopicType_TOPIC_TYPE_REGRESSION,
		OutputArity:              emissionstypes.TopicOutputArity_TOPIC_OUTPUT_ARITY_SINGLE,
		RequireUnity:             false,
		UnityTolerance:           alloraMath.Dec{},
		MaxLabelsPerSubmission:   emissionstypes.DefaultMaxLabelsPerSubmission,
		LabelWhitelist:           nil,
		LabelDefaultValue:        alloraMath.ZeroDec(),
		LabelCaseSensitive:       true,
	}
}

// registerCreatedTopic records a newly created topic in the simulation data.
func registerCreatedTopic(data *SimulationData, topicId uint64, creator Actor) {
	data.counts.incrementCreateTopicCount()
	data.setTopicCreator(topicId, creator)
	data.enableTopicWorkersWhitelist(topicId)
	data.enableTopicReputersWhitelist(topicId)
}

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
	createTopicRequest := newCreateTopicRequest(actor, data.epochLength, iteration)
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

	createTopicResponse := &emissionstypes.CreateNewTopicResponse{} //nolint:exhaustruct // the fields are populated by decode
	err = txResp.Decode(createTopicResponse)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, actor, "failed to create topic", "tx decode error", err)
		return false
	}

	registerCreatedTopic(data, createTopicResponse.TopicId, actor)
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

// decodeMsgResponses decodes every message response of a transaction into a fresh value
// from newResponse, in message order. cosmosclient.Response.Decode only returns the first.
func decodeMsgResponses[T proto.Message](
	m *testcommon.TestConfig,
	txResp cosmosclient.Response,
	newResponse func() T,
) ([]T, error) {
	raw, err := hex.DecodeString(txResp.Data)
	if err != nil {
		return nil, fmt.Errorf("decoding tx data: %w", err)
	}
	var txMsgData sdktypes.TxMsgData
	if err := m.Cdc.Unmarshal(raw, &txMsgData); err != nil {
		return nil, fmt.Errorf("unmarshalling tx msg data: %w", err)
	}
	responses := make([]T, 0, len(txMsgData.MsgResponses))
	for i, anyResp := range txMsgData.MsgResponses {
		resp := newResponse()
		want := "/" + proto.MessageName(resp)
		if anyResp.TypeUrl != want {
			return nil, fmt.Errorf("message response %d has type %s, want %s", i, anyResp.TypeUrl, want)
		}
		if err := proto.Unmarshal(anyResp.Value, resp); err != nil {
			return nil, fmt.Errorf("unmarshalling message response %d: %w", i, err)
		}
		responses = append(responses, resp)
	}
	return responses, nil
}

// createTopicsInOneTx creates numTopics topics from actor in a single transaction, so that
// they are all created in the same block. Returns the new topic ids in message order.
func createTopicsInOneTx(
	m *testcommon.TestConfig,
	actor Actor,
	numTopics int,
	data *SimulationData,
	iteration int,
) (topicIds []uint64, success bool) {
	iterLog(m.T, iteration, actor, "creating", numTopics, "new topics in one transaction")
	msgs := make([]sdktypes.Msg, 0, numTopics)
	for i := 0; i < numTopics; i++ {
		msgs = append(msgs, newCreateTopicRequest(actor, data.epochLength, iteration))
	}

	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, msgs...)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, actor, "failed to create topics", "tx broadcast error", err)
		return nil, false
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, actor, "failed to create topics", "tx wait error", err)
		return nil, false
	}
	responses, err := decodeMsgResponses(m, txResp, func() *emissionstypes.CreateNewTopicResponse {
		return &emissionstypes.CreateNewTopicResponse{} //nolint:exhaustruct // populated by decode
	})
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, actor, "failed to create topics", "tx decode error", err)
		return nil, false
	}
	if len(responses) != numTopics {
		err = fmt.Errorf("expected %d topic creation responses, got %d", numTopics, len(responses))
		failIfOnErr(m.T, data.failOnErr, err)
		iterFailLog(m.T, iteration, actor, "failed to create topics", err)
		return nil, false
	}

	topicIds = make([]uint64, 0, numTopics)
	for _, resp := range responses {
		registerCreatedTopic(data, resp.TopicId, actor)
		topicIds = append(topicIds, resp.TopicId)
	}
	iterSuccessLog(m.T, iteration, actor, "created topics", topicIds)
	return topicIds, true
}

// fundTopicsInOneTx funds several topics from actor in a single transaction, so that the
// topics whose weight becomes positive with this revenue all activate in the same block.
// amounts[i] funds topicIds[i].
func fundTopicsInOneTx(
	m *testcommon.TestConfig,
	actor Actor,
	topicIds []uint64,
	amounts []cosmossdk_io_math.Int,
	data *SimulationData,
	iteration int,
) (success bool) {
	iterLog(m.T, iteration, actor, "funding topics", topicIds, "in one transaction with amounts", amounts)
	msgs := make([]sdktypes.Msg, 0, len(topicIds))
	for i, topicId := range topicIds {
		msgs = append(msgs, &emissionstypes.FundTopicRequest{
			Sender:  actor.addr,
			TopicId: topicId,
			Amount:  amounts[i],
		})
	}

	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, actor.acc, msgs...)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, actor, "failed to fund topics", topicIds, "tx broadcast error", err)
		return false
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, actor, "failed to fund topics", topicIds, "tx wait error", err)
		return false
	}

	for range topicIds {
		data.counts.incrementFundTopicCount()
	}
	iterSuccessLog(m.T, iteration, actor, "funded topics", topicIds)
	return true
}

// activateTopicsInOneTx makes every topic's weight positive in a single transaction: the
// reputer registers on each topic, stakes on it and funds it. Whatever the chain's minimum
// weight, the first message that activates a topic is in this transaction for all of them,
// so they all activate in the same block. stakes[i] and funds[i] apply to topicIds[i].
func activateTopicsInOneTx(
	m *testcommon.TestConfig,
	reputer Actor,
	topicIds []uint64,
	stakes []cosmossdk_io_math.Int,
	funds []cosmossdk_io_math.Int,
	data *SimulationData,
	iteration int,
) (success bool) {
	iterLog(m.T, iteration, reputer, "registering, staking and funding topics", topicIds, "in one transaction; stakes", stakes, "funds", funds)
	msgs := make([]sdktypes.Msg, 0, 3*len(topicIds))
	for i, topicId := range topicIds {
		msgs = append(msgs,
			&emissionstypes.RegisterRequest{
				Sender:    reputer.addr,
				Owner:     reputer.addr,
				IsReputer: true,
				TopicId:   topicId,
			},
			&emissionstypes.AddStakeRequest{
				Sender:  reputer.addr,
				TopicId: topicId,
				Amount:  stakes[i],
			},
			&emissionstypes.FundTopicRequest{
				Sender:  reputer.addr,
				TopicId: topicId,
				Amount:  funds[i],
			},
		)
	}

	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, reputer.acc, msgs...)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, reputer, "failed to activate topics", topicIds, "tx broadcast error", err)
		return false
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, reputer, "failed to activate topics", topicIds, "tx wait error", err)
		return false
	}

	for _, topicId := range topicIds {
		data.addReputerRegistration(topicId, reputer)
		data.counts.incrementRegisterReputerCount()
		data.addReputerStaked(topicId, reputer)
		data.counts.incrementStakeAsReputerCount()
		data.counts.incrementFundTopicCount()
	}
	iterSuccessLog(m.T, iteration, reputer, "activated topics", topicIds)
	return true
}

// setMaxActiveTopicsPerBlock changes the per-block topic limit through UpdateParams, sent by
// an admin (the faucet is one on the local testnet), and returns the value it replaced. No
// transaction is sent when the limit already has the requested value.
func setMaxActiveTopicsPerBlock(
	m *testcommon.TestConfig,
	admin Actor,
	limit uint64,
	data *SimulationData,
	iteration int,
) (previous uint64, success bool) {
	ctx := context.Background()
	paramsResp, err := m.Client.QueryEmissions().GetParams(ctx, &emissionstypes.GetParamsRequest{})
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, admin, "failed to query params", err)
		return 0, false
	}
	previous = paramsResp.Params.MaxActiveTopicsPerBlock
	if previous == limit {
		return previous, true
	}
	iterLog(m.T, iteration, admin, "setting max active topics per block from", previous, "to", limit)
	msg := &emissionstypes.UpdateParamsRequest{
		Sender: admin.addr,
		Params: &emissionstypes.OptionalParams{MaxActiveTopicsPerBlock: []uint64{limit}}, //nolint:exhaustruct // only the field under change
	}
	txResp, err := m.Client.BroadcastTx(ctx, admin.acc, msg)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, admin, "failed to update max active topics per block", "tx broadcast error", err)
		return previous, false
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil {
		iterFailLog(m.T, iteration, admin, "failed to update max active topics per block", "tx wait error", err)
		return previous, false
	}
	paramsResp, err = m.Client.QueryEmissions().GetParams(ctx, &emissionstypes.GetParamsRequest{})
	failIfOnErr(m.T, data.failOnErr, err)
	if err != nil || paramsResp.Params.MaxActiveTopicsPerBlock != limit {
		iterFailLog(m.T, iteration, admin, "max active topics per block not updated", err)
		return previous, false
	}
	iterSuccessLog(m.T, iteration, admin, "set max active topics per block to", limit)
	return previous, true
}
