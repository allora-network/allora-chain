package integration_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"
)

func waitForNextChurningBlock(m testCommon.TestConfig, topicId uint64) (*types.Topic, error) {
	ctx := context.Background()
	topicResponse, err := m.Client.QueryEmissions().GetTopic(ctx, &types.QueryTopicRequest{TopicId: topicId})
	if err != nil {
		return nil, err
	}
	nextBlockResponse, err := m.Client.QueryEmissions().GetNextChurningBlockByTopicId(ctx, &types.QueryNextChurningBlockByTopicIdRequest{TopicId: topicId})
	if err != nil {
		return nil, err
	}
	m.T.Log(time.Now(), "Wait for next churning block ", nextBlockResponse.BlockHeight, " for topic ", topicId)
	err = m.Client.WaitForBlockHeight(ctx, nextBlockResponse.BlockHeight)
	return topicResponse.Topic, err
}
func InsertSingleWorkerPayload(m testCommon.TestConfig, topic *types.Topic, blockHeight int64) error {
	ctx := context.Background()
	// Nonce: calculate from EpochLastRan + EpochLength
	topicId := topic.Id
	nonce := types.Nonce{BlockHeight: blockHeight}
	// Define inferer address as Bob's address
	InfererAddress1 := m.BobAddr

	workerMsg := &types.InsertWorkerPayloadRequest{
		Sender: InfererAddress1,
		WorkerDataBundle: &types.WorkerDataBundle{
			Worker:  InfererAddress1,
			Nonce:   &nonce,
			TopicId: topicId,
			InferenceForecastsBundle: &types.InferenceForecastBundle{
				Inference: &types.Inference{
					TopicId:     topicId,
					BlockHeight: blockHeight,
					Inferer:     InfererAddress1,
					Value:       alloraMath.NewDecFromInt64(100),
				},
				Forecast: &types.Forecast{
					TopicId:     topicId,
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
	}
	// Sign
	src := make([]byte, 0)
	src, err := workerMsg.WorkerDataBundle.InferenceForecastsBundle.XXX_Marshal(src, true)
	// require.NoError(m.T, err, "Marshall reputer value bundle should not return an error")
	if err != nil {
		return err
	}

	sig, pubKey, err := m.Client.Context().Keyring.Sign(m.BobAcc.Name, src, signing.SignMode_SIGN_MODE_DIRECT)
	// require.NoError(m.T, err, "Sign should not return an error")
	if err != nil {
		return err
	}
	workerPublicKeyBytes := pubKey.Bytes()
	workerMsg.WorkerDataBundle.InferencesForecastsBundleSignature = sig
	workerMsg.WorkerDataBundle.Pubkey = hex.EncodeToString(workerPublicKeyBytes)

	txResp, err := m.Client.BroadcastTx(ctx, m.BobAcc, workerMsg)
	// require.NoError(m.T, err)
	if err != nil {
		return err
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	// require.NoError(m.T, err)
	if err != nil {
		return err
	}

	return nil
}

// Worker Bob inserts inference and forecast
func InsertWorkerBundle(m testCommon.TestConfig, topic *types.Topic) (int64, error) {
	ctx := context.Background()
	currentBlock, err := m.Client.BlockHeight(ctx)
	if err != nil {
		return 0, err
	}
	topicResponse, err := m.Client.QueryEmissions().GetTopic(ctx, &types.QueryTopicRequest{TopicId: topic.Id})
	if err != nil {
		return 0, err
	}
	freshTopic := topicResponse.Topic

	// Insert and fulfill nonces for the last two epochs
	blockHeightEval := freshTopic.EpochLastEnded
	m.T.Log(time.Now(), "Inserting worker bundle for blockHeightEval: ", blockHeightEval, "; Current block: ", currentBlock)
	err = InsertSingleWorkerPayload(m, freshTopic, blockHeightEval)
	if err != nil {
		return 0, err
	}
	return blockHeightEval, nil
}

// register alice as a reputer in topic 1, then check success
func InsertReputerBundle(m testCommon.TestConfig, topic *types.Topic, BlockHeightCurrent int64) error {
	ctx := context.Background()
	// Nonce: calculate from EpochLastRan + EpochLength
	topicId := topic.Id
	// Define inferer address as Bob's address, reputer as Alice's
	workerAddr := m.BobAddr
	reputerAddr := m.AliceAddr
	// Reputer Nonce
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
	// require.NoError(m.T, err, "Marshall reputer value bundle should not return an error")
	if err != nil {
		return err
	}

	valueBundleSignature, pubKey, err := m.Client.Context().Keyring.Sign(m.AliceAcc.Name, src, signing.SignMode_SIGN_MODE_DIRECT)
	// require.NoError(m.T, err, "Sign should not return an error")
	if err != nil {
		return err
	}
	reputerPublicKeyBytes := pubKey.Bytes()

	lossesMsg := &types.InsertReputerPayloadRequest{
		Sender: reputerAddr,
		ReputerValueBundle: &types.ReputerValueBundle{
			ValueBundle: reputerValueBundle,
			Signature:   valueBundleSignature,
			Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
		},
	}

	txResp, err := m.Client.BroadcastTx(ctx, m.AliceAcc, lossesMsg)
	if err != nil {
		return err
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	if err != nil {
		return err
	}

	m.T.Log(time.Now(), "Inserted reputer payload for blockHeight: ", BlockHeightCurrent)
	return nil
}

func ValidateQueryNetworkLossBundle(m testCommon.TestConfig, topicId uint64, BlockHeightCurrent int64) {
	ctx := context.Background()
	result, err := m.Client.QueryEmissions().GetNetworkLossBundleAtBlock(ctx,
		&types.QueryNetworkLossBundleAtBlockRequest{
			TopicId:     topicId,
			BlockHeight: BlockHeightCurrent,
		},
	)
	require.NoError(m.T, err)
	require.NotNil(m.T, result, "Result should not be nil")
	require.NotNil(m.T, result.LossBundle, "Retrieved data should match inserted data")
}

// Register two actors and check their registrations went through
func WorkerInferenceAndForecastChecks(m testCommon.TestConfig) {
	ctx := context.Background()
	m.T.Log(time.Now(), "--- START  Worker Inference, Forecast and Reputation test ---")
	// Nonce: calculate from EpochLastRan + EpochLength

	topic, err := waitForNextChurningBlock(m, 1)
	if err != nil {
		require.NoError(m.T, err)
	}
	m.T.Log(time.Now(), "--- Insert Worker Bundle ---")
	// Waiting for ground truth lag to pass
	m.T.Log(time.Now(), "--- Waiting to Insert Worker Bundle ---")
	blockHeightNonce, err := RunWithRetry(m, 3, 2*time.Second, func() (int64, error) {
		topicResponse, err := m.Client.QueryEmissions().GetTopic(ctx, &types.QueryTopicRequest{TopicId: topic.Id})
		if err != nil {
			return 0, err
		}
		topic := topicResponse.Topic
		_, err = InsertWorkerBundle(m, topic) // Assuming InsertReputerBundle returns (int, error)
		if err != nil {
			return 0, err
		}
		return topic.EpochLastEnded, err
	})
	if err != nil {
		m.T.Log(time.Now(), "--- Failed inserting worker payload ---")
		require.NoError(m.T, err)
	}
	m.T.Log(time.Now(), fmt.Sprintf("--- Waiting for block %d ---", blockHeightNonce+topic.GroundTruthLag))
	err = m.Client.WaitForBlockHeight(ctx, blockHeightNonce+topic.GroundTruthLag)
	if err != nil {
		m.T.Log(time.Now(), "--- Failed waiting for ground truth lag ---")
		require.NoError(m.T, err)
	}

	m.T.Log(time.Now(), "--- Insert Reputer Bundle ---")
	err = InsertReputerBundle(m, topic, blockHeightNonce)
	if err != nil {
		m.T.Log(time.Now(), "--- Failed inserting reputer payload ---")
		require.NoError(m.T, err)
	}

	m.T.Log(time.Now(), fmt.Sprintf("--- Waiting for block %d ---", blockHeightNonce+topic.GroundTruthLag+topic.EpochLength))
	err = m.Client.WaitForBlockHeight(ctx, blockHeightNonce+topic.GroundTruthLag+topic.EpochLength)
	if err != nil {
		m.T.Log(time.Now(), "--- Failed waiting for epoch length ---")
		require.NoError(m.T, err)
	}

	ValidateQueryNetworkLossBundle(m, topic.Id, blockHeightNonce)
	m.T.Log(time.Now(), "--- END  Worker Inference, Forecast and Reputation test ---")
}

// RunWithRetry retries a function that returns an error, n times
func RunWithRetry(m testCommon.TestConfig, retryCount int, sleep time.Duration, operation func() (int64, error)) (int64, error) {
	var err error
	for i := 0; i < retryCount; i++ {
		val, err := operation()
		if err == nil {
			return val, nil // Success, no need to retry
		}
		m.T.Log(time.Now(), fmt.Sprintf("Attempt %d/%d failed, error: %s\n", i+1, retryCount, err))
		time.Sleep(sleep) // Optional: wait before retrying
	}
	return 0, fmt.Errorf("after %d attempts, last error: %s", retryCount, err)
}
