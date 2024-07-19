package integration_test

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"
)

const defaultEpochLength = 10
const approximateBlockLengthSeconds = 5
const minWaitingNumberofEpochs = 3

func getNonZeroTopicEpochLastRan(m testCommon.TestConfig, topicID uint64, maxRetries int) (*types.Topic, error) {
	ctx := context.Background()
	sleepingTimeBlocks := defaultEpochLength
	// Retry loop for a maximum of 5 times
	for retries := 0; retries < maxRetries; retries++ {
		topicResponse, err := m.Client.QueryEmissions().GetTopic(ctx, &types.QueryTopicRequest{TopicId: topicID})
		if err == nil {
			storedTopic := topicResponse.Topic
			if storedTopic.EpochLastEnded != 0 {
				sleepingTime := time.Duration(minWaitingNumberofEpochs*storedTopic.EpochLength*approximateBlockLengthSeconds) * time.Second
				m.T.Log(time.Now(), " Topic found, sleeping...", sleepingTime)
				time.Sleep(sleepingTime)
				m.T.Log(time.Now(), " Slept.")
				return topicResponse.Topic, nil
			}
			sleepingTimeBlocks = int(storedTopic.EpochLength)
		} else {
			m.T.Log("Error getting topic, retry...", err)
		}
		// Sleep for a while before retrying
		m.T.Log("Retrying sleeping for a default epoch, retry ", retries, " for sleeping time ", sleepingTimeBlocks)
		time.Sleep(time.Duration(sleepingTimeBlocks*approximateBlockLengthSeconds) * time.Second)
	}

	return nil, errors.New("topicEpochLastRan is still 0 after retrying")
}

func InsertSingleWorkerBulk(m testCommon.TestConfig, topic *types.Topic, blockHeight int64) {
	ctx := context.Background()
	// Nonce: calculate from EpochLastRan + EpochLength
	topicId := topic.Id
	nonce := types.Nonce{BlockHeight: blockHeight}
	// Define inferer address as Bob's address
	InfererAddress1 := m.BobAddr

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
	require.NoError(m.T, err, "Marshall reputer value bundle should not return an error")

	sig, pubKey, err := m.Client.Context().Keyring.Sign(m.BobAcc.Name, src, signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(m.T, err, "Sign should not return an error")
	workerPublicKeyBytes := pubKey.Bytes()
	workerMsg.WorkerDataBundles[0].InferencesForecastsBundleSignature = sig
	workerMsg.WorkerDataBundles[0].Pubkey = hex.EncodeToString(workerPublicKeyBytes)

	txResp, err := m.Client.BroadcastTx(ctx, m.BobAcc, workerMsg)
	require.NoError(m.T, err)

	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)

	// Latest inference
	latestInference, err := m.Client.QueryEmissions().GetWorkerLatestInferenceByTopicId(
		ctx,
		&types.QueryWorkerLatestInferenceRequest{
			TopicId:       1,
			WorkerAddress: InfererAddress1,
		},
	)
	require.NoError(m.T, err)
	require.Equal(m.T, latestInference.LatestInference.Value, alloraMath.MustNewDecFromString("100"))
	require.Equal(m.T, latestInference.LatestInference.BlockHeight, blockHeight)
	require.Equal(m.T, latestInference.LatestInference.TopicId, topicId)
	require.Equal(m.T, latestInference.LatestInference.Inferer, InfererAddress1)
}

// Worker Bob inserts bulk inference and forecast
func InsertWorkerBulk(m testCommon.TestConfig, topic *types.Topic) (int64, int64) {
	ctx := context.Background()
	topicResponse, err := m.Client.QueryEmissions().GetTopic(ctx, &types.QueryTopicRequest{TopicId: topic.Id})
	require.NoError(m.T, err)
	freshTopic := topicResponse.Topic

	// Insert and fulfill nonces for the last two epochs
	blockHeightEval := freshTopic.EpochLastEnded - freshTopic.EpochLength
	m.T.Log("Inserting worker bulk for blockHeightEval: ", blockHeightEval)
	InsertSingleWorkerBulk(m, freshTopic, blockHeightEval)

	blockHeightCurrent := freshTopic.EpochLastEnded
	m.T.Log("Inserting worker bulk for blockHeightCurrent: ", blockHeightCurrent)
	InsertSingleWorkerBulk(m, freshTopic, blockHeightCurrent)
	return blockHeightCurrent, blockHeightEval
}

// register alice as a reputer in topic 1, then check success
func InsertReputerBulk(m testCommon.TestConfig, topic *types.Topic, BlockHeightCurrent, BlockHeightEval int64) {
	ctx := context.Background()
	// Nonce: calculate from EpochLastRan + EpochLength
	topicId := topic.Id
	// Define inferer address as Bob's address, reputer as Alice's
	workerAddr := m.BobAddr
	reputerAddr := m.AliceAddr
	// Nonces are last two blockHeights
	reputerNonce := &types.Nonce{
		BlockHeight: BlockHeightCurrent,
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
		},
	}

	// Sign
	src := make([]byte, 0)
	src, err := reputerValueBundle.XXX_Marshal(src, true)
	require.NoError(m.T, err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature, pubKey, err := m.Client.Context().Keyring.Sign(m.AliceAcc.Name, src, signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(m.T, err, "Sign should not return an error")
	reputerPublicKeyBytes := pubKey.Bytes()

	// Create a MsgInsertBulkReputerPayload message
	lossesMsg := &types.MsgInsertBulkReputerPayload{
		Sender:  reputerAddr,
		TopicId: topicId,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
		},
		ReputerValueBundles: []*types.ReputerValueBundle{
			{
				ValueBundle: reputerValueBundle,
				Signature:   valueBundleSignature,
				Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
			},
		},
	}

	txResp, err := m.Client.BroadcastTx(ctx, m.AliceAcc, lossesMsg)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)

	result, err := m.Client.QueryEmissions().GetNetworkLossBundleAtBlock(ctx,
		&types.QueryNetworkLossBundleAtBlockRequest{
			TopicId:     topicId,
			BlockHeight: BlockHeightCurrent,
		},
	)
	require.NoError(m.T, err)
	require.NotNil(m.T, result)
	require.NotNil(m.T, result.LossBundle, "Retrieved data should match inserted data")

}

// Register two actors and check their registrations went through
func WorkerInferenceAndForecastChecks(m testCommon.TestConfig) {
	// Nonce: calculate from EpochLastRan + EpochLength
	topic, err := getNonZeroTopicEpochLastRan(m, 1, 5)
	if err != nil {
		m.T.Log("--- Failed getting a topic that was ran ---")
		require.NoError(m.T, err)
	}
	m.T.Log("--- Insert Worker Bulk ---")
	blockHeightCurrent, blockHeightEval := InsertWorkerBulk(m, topic)
	m.T.Log("--- Insert Reputer Bulk ---")
	InsertReputerBulk(m, topic, blockHeightCurrent, blockHeightEval)
}
