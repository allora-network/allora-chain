package integration_test

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"
)

const defaultEpochLength = 10
const approximateBlockLengthSeconds = 5
const minWaitingNumberofEpochs = 3

func getNonZeroTopicEpochLastRan(ctx context.Context, query emissionstypes.QueryClient, topicID uint64, maxRetries int) (*emissionstypes.Topic, error) {
	sleepingTimeBlocks := defaultEpochLength
	// Retry loop for a maximum of 5 times
	for retries := 0; retries < maxRetries; retries++ {
		topicResponse, err := query.GetTopic(ctx, &emissionstypes.QueryTopicRequest{TopicId: topicID})
		if err == nil {
			storedTopic := topicResponse.Topic
			if storedTopic.EpochLastEnded != 0 {
				sleepingTimeSeconds := time.Duration(minWaitingNumberofEpochs*storedTopic.EpochLength*approximateBlockLengthSeconds) * time.Second
				fmt.Println(time.Now(), " Topic found, sleeping...", sleepingTimeSeconds)
				time.Sleep(sleepingTimeSeconds)
				fmt.Println(time.Now(), " Slept.")
				return topicResponse.Topic, nil
			}
			sleepingTimeBlocks = int(storedTopic.EpochLength)
		} else {
			fmt.Println("Error getting topic, retry...", err)
		}
		// Sleep for a while before retrying
		fmt.Println("Retrying sleeping for a default epoch, retry ", retries, " for sleeping time ", sleepingTimeBlocks)
		time.Sleep(time.Duration(sleepingTimeBlocks*approximateBlockLengthSeconds) * time.Second)
	}

	return nil, errors.New("topicEpochLastRan is still 0 after retrying")
}

func InsertSingleWorkerBulk(m TestMetadata, topic *types.Topic, blockHeight int64) {
	// Nonce: calculate from EpochLastRan + EpochLength
	topicId := topic.Id
	nonce := emissionstypes.Nonce{BlockHeight: blockHeight}
	// Define inferer address as Bob's address
	InfererAddress1 := m.n.BobAddr

	// Create a MsgInsertBulkReputerPayload message
	workerMsg := &types.MsgInsertBulkWorkerPayload{
		Sender:  InfererAddress1,
		Nonce:   &nonce,
		TopicId: topicId,
		WorkerDataBundles: []*types.WorkerDataBundle{
			{
				Worker: InfererAddress1,
				InferenceForecastsBundle: &types.InferenceForecastBundle{
					Inference: &types.Inference{
						TopicId:     topicId,
						BlockHeight: blockHeight,
						Inferer:     InfererAddress1,
						Value:       alloraMath.NewDecFromInt64(100),
					},
					Forecast: &types.Forecast{
						TopicId:     0,
						BlockHeight: blockHeight,
						Forecaster:  InfererAddress1,
						ForecastElements: []*types.ForecastElement{
							{
								Inferer: InfererAddress1,
								Value:   alloraMath.NewDecFromInt64(100),
							},
						},
					},
				},
			},
		},
	}
	// Sign
	src := make([]byte, 0)
	src, err := workerMsg.WorkerDataBundles[0].InferenceForecastsBundle.XXX_Marshal(src, true)
	require.NoError(m.t, err, "Marshall reputer value bundle should not return an error")

	sig, pubKey, err := m.n.Client.Context().Keyring.Sign(m.n.BobAcc.Name, src, signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(m.t, err, "Sign should not return an error")
	workerPublicKeyBytes := pubKey.Bytes()
	workerMsg.WorkerDataBundles[0].InferencesForecastsBundleSignature = sig
	workerMsg.WorkerDataBundles[0].Pubkey = hex.EncodeToString(workerPublicKeyBytes)

	txResp, err := m.n.Client.BroadcastTx(m.ctx, m.n.BobAcc, workerMsg)
	if err != nil && err.Error() == types.ErrNonceAlreadyFulfilled.Error() {
		// May already be fulfilled due to other tests or multiple runs of these tests
		log.Println("Nonce already fulfilled -- continuing")
		return
	}
	require.NoError(m.t, err)

	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	if err != nil && err.Error() == types.ErrNonceAlreadyFulfilled.Error() {
		// May already be fulfilled due to other tests or multiple runs of these tests
		log.Println("Nonce already fulfilled -- continuing")
		return
	}
	require.NoError(m.t, err)

	// Latest inference
	latestInference, err := m.n.QueryEmissions.GetWorkerLatestInferenceByTopicId(
		m.ctx,
		&emissionstypes.QueryWorkerLatestInferenceRequest{
			TopicId:       1,
			WorkerAddress: InfererAddress1,
		},
	)
	require.NoError(m.t, err)
	require.Equal(m.t, latestInference.LatestInference.Value, alloraMath.MustNewDecFromString("100"))
	require.Equal(m.t, latestInference.LatestInference.BlockHeight, blockHeight)
	require.Equal(m.t, latestInference.LatestInference.TopicId, topicId)
	require.Equal(m.t, latestInference.LatestInference.Inferer, InfererAddress1)
}

// Worker Bob inserts bulk inference and forecast
func InsertWorkerBulk(m TestMetadata, topic *types.Topic) (int64, int64) {
	topicResponse, err := m.n.QueryEmissions.GetTopic(m.ctx, &emissionstypes.QueryTopicRequest{TopicId: topic.Id})
	require.NoError(m.t, err)
	freshTopic := topicResponse.Topic

	// Insert and fulfill nonces for the last two epochs
	blockHeightEval := freshTopic.EpochLastEnded - freshTopic.EpochLength
	fmt.Println("Inserting worker bulk for blockHeightEval: ", blockHeightEval)
	InsertSingleWorkerBulk(m, freshTopic, blockHeightEval)

	blockHeightCurrent := freshTopic.EpochLastEnded
	fmt.Println("Inserting worker bulk for blockHeightCurrent: ", blockHeightCurrent)
	InsertSingleWorkerBulk(m, freshTopic, blockHeightCurrent)
	return blockHeightCurrent, blockHeightEval
}

// register alice as a reputer in topic 1, then check success
func InsertReputerBulk(m TestMetadata, topic *types.Topic, BlockHeightCurrent, BlockHeightEval int64) {
	// Nonce: calculate from EpochLastRan + EpochLength
	topicId := topic.Id
	// Define inferer address as Bob's address, reputer as Alice's
	workerAddr := m.n.BobAddr
	reputerAddr := m.n.AliceAddr
	// Nonces are last two blockHeights
	reputerNonce := &types.Nonce{
		BlockHeight: BlockHeightCurrent,
	}
	workerNonce := &types.Nonce{
		BlockHeight: BlockHeightEval,
	}

	reputerValueBundle := &types.ValueBundle{
		TopicId:       topicId,
		Reputer:       reputerAddr,
		CombinedValue: alloraMath.NewDecFromInt64(100),
		InfererValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr,
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		ForecasterValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr,
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		NaiveValue: alloraMath.NewDecFromInt64(100),
		OneOutInfererValues: []*types.WithheldWorkerAttributedValue{
			// There cannot be a 1-out inferer value if there is just 1 inferer => this will be ignored by msgserver
			{
				Worker: workerAddr,
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{
			{
				Worker: workerAddr,
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		// Just as valid:
		// OneOutInfererValues:    []*types.WithheldWorkerAttributedValue{},
		// OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{},
		OneInForecasterValues: []*types.WorkerAttributedValue{
			{
				Worker: workerAddr,
				Value:  alloraMath.NewDecFromInt64(100),
			},
		},
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
	}

	// Sign
	src := make([]byte, 0)
	src, err := reputerValueBundle.XXX_Marshal(src, true)
	require.NoError(m.t, err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature, pubKey, err := m.n.Client.Context().Keyring.Sign(m.n.AliceAcc.Name, src, signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(m.t, err, "Sign should not return an error")
	reputerPublicKeyBytes := pubKey.Bytes()

	// Create a MsgInsertBulkReputerPayload message
	lossesMsg := &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddr,
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				ValueBundle: reputerValueBundle,
				Signature:   valueBundleSignature,
				Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
			},
		},
	}

	txResp, err := m.n.Client.BroadcastTx(m.ctx, m.n.AliceAcc, lossesMsg)
	require.NoError(m.t, err)
	_, err = m.n.Client.WaitForTx(m.ctx, txResp.TxHash)
	require.NoError(m.t, err)

	result, err := m.n.QueryEmissions.GetNetworkLossBundleAtBlock(m.ctx,
		&emissionstypes.QueryNetworkLossBundleAtBlockRequest{
			TopicId:     topicId,
			BlockHeight: BlockHeightCurrent,
		},
	)
	require.NoError(m.t, err)
	require.NotNil(m.t, result)
	require.NotNil(m.t, result.LossBundle, "Retrieved data should match inserted data")

}

// Register two actors and check their registrations went through
func WorkerInferenceAndForecastChecks(m TestMetadata) {
	// Nonce: calculate from EpochLastRan + EpochLength
	topic, err := getNonZeroTopicEpochLastRan(m.ctx, m.n.QueryEmissions, 1, 5)
	if err != nil {
		m.t.Log("--- Failed getting a topic that was ran ---")
		require.NoError(m.t, err)
	}
	m.t.Log("--- Insert Worker Bulk ---")
	blockHeightCurrent, blockHeightEval := InsertWorkerBulk(m, topic)
	m.t.Log("--- Insert Reputer Bulk ---")
	InsertReputerBulk(m, topic, blockHeightCurrent, blockHeightEval)
}
