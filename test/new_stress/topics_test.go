package newstress_test

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	proto "github.com/cosmos/gogoproto/proto"
)

// Creates multiple topics in a single broadcast or separate broadcasts
func createTopics(
	m *testcommon.TestConfig,
	actor Actor,
	numTopics int,
	epochLength int64,
	createTopicsSameBlock bool,
) ([]uint64, error) {
	m.T.Log("Creating", numTopics, "topics, same block:", createTopicsSameBlock)
	ctx := context.Background()

	// Get Next Block Id
	resp, err := m.Client.QueryEmissions().GetNextTopicId(ctx, &emissionstypes.GetNextTopicIdRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get block height: %w", err)
	}
	fmt.Println("resp: ")

	if createTopicsSameBlock {
		// Create all topics in one broadcast
		numTopics++ // TODO: need to increment to create the exact number of topics due to a bug in the iteration - investigate
		requests := make([]*emissionstypes.CreateNewTopicRequest, numTopics)
		topicIds := make([]uint64, numTopics)
		topicId := resp.NextTopicId
		for i := 0; i < numTopics; i++ {
			requests[i] = &emissionstypes.CreateNewTopicRequest{
				Creator:                  actor.addr,
				Metadata:                 fmt.Sprintf("Created topic %d", i+1),
				LossMethod:               "mse",
				EpochLength:              epochLength,
				GroundTruthLag:           epochLength,
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
			topicIds[i] = topicId
			topicId++
		}

		protoMsgs := make([]proto.Message, len(requests))
		for i, req := range requests {
			protoMsgs[i] = req
		}

		txResp, err := m.Client.BroadcastTx(ctx, actor.acc, protoMsgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to broadcast create topic requests: %w", err)
		}

		_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
		if err != nil {
			return nil, fmt.Errorf("failed waiting for create topic tx: %w", err)
		}

		m.T.Log("Created topics:", topicIds)
		return topicIds, nil

	} else {
		// Create topics in separate broadcasts
		topicIds := make([]uint64, numTopics)
		for i := 0; i < numTopics; i++ {
			request := &emissionstypes.CreateNewTopicRequest{
				Creator:                  actor.addr,
				Metadata:                 fmt.Sprintf("Created topic %d", i+1),
				LossMethod:               "mse",
				EpochLength:              epochLength,
				GroundTruthLag:           epochLength,
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

			txResp, err := m.Client.BroadcastTx(ctx, actor.acc, request)
			if err != nil {
				return nil, fmt.Errorf("failed to broadcast create topic request %d: %w", i, err)
			}

			_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
			if err != nil {
				return nil, fmt.Errorf("failed waiting for create topic tx %d: %w", i, err)
			}

			response := &emissionstypes.CreateNewTopicResponse{} // nolint:exhaustruct
			err = txResp.Decode(response)
			if err != nil {
				return nil, fmt.Errorf("failed to decode response %d: %w", i, err)
			}
			topicIds[i] = response.TopicId
		}

		m.T.Log("Created topics:", topicIds)
		return topicIds, nil
	}
}

// broadcast a tx to fund a topic
func fundTopics(
	m *testcommon.TestConfig,
	topicIds []uint64,
	sender Actor,
	amount int64,
) error {
	ctx := context.Background()
	requests := make([]*emissionstypes.FundTopicRequest, len(topicIds))
	for i, topicId := range topicIds {
		requests[i] = &emissionstypes.FundTopicRequest{
			Sender:  sender.addr,
			TopicId: topicId,
			Amount:  math.NewInt(amount),
		}
		fmt.Println("Funding topic: ", topicId, "with amount: ", amount, "from: ", sender.name)
	}

	protoMsgs := make([]proto.Message, len(requests))
	for i, req := range requests {
		protoMsgs[i] = req
	}

	txResp, err := m.Client.BroadcastTx(ctx, sender.acc, protoMsgs...)
	if err != nil {
		return fmt.Errorf("failed to broadcast fund topic requests: %w", err)
	}

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	if err != nil {
		return fmt.Errorf("failed waiting for fund topic tx: %w", err)
	}

	return nil
}
