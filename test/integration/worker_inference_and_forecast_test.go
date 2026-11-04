package integration_test

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

func waitForNextChurningBlock(m testCommon.TestConfig, topicId uint64) (*types.Topic, error) {
	ctx := context.Background()
	topicResponse, err := m.Client.QueryEmissions().GetTopic(ctx, &types.GetTopicRequest{TopicId: topicId})
	if err != nil {
		return nil, err
	}
	nextBlockResponse, err := m.Client.QueryEmissions().GetNextChurningBlockByTopicId(ctx, &types.GetNextChurningBlockByTopicIdRequest{TopicId: topicId})
	if err != nil {
		return nil, err
	}
	m.T.Log(time.Now(), "Wait for next churning block ", nextBlockResponse.BlockHeight, " for topic ", topicId)
	err = m.Client.WaitForBlockHeight(ctx, nextBlockResponse.BlockHeight)
	return topicResponse.Topic, err
}

func InsertSingleWorkerPayload(m testCommon.TestConfig, topic *types.Topic, blockHeight int64) error {
	if len(m.InfererValues) == 0 {
		return errors.New("values can not be empty")
	}
	ctx := context.Background()
	topicId := topic.Id
	nonce := types.Nonce{BlockHeight: blockHeight}
	// Define inferer address as Bob's address
	InfererAddress1 := m.BobAddr

	workerMsg := &types.InsertWorkerPayloadRequest{
		Sender: InfererAddress1,
		WorkerDataBundle: &types.InputWorkerDataBundle{
			Worker:  InfererAddress1,
			Nonce:   &nonce,
			TopicId: topicId,
			InferenceForecastsBundle: &types.InputInferenceForecastBundle{
				Inference: &types.InputInference{
					TopicId:     topicId,
					BlockHeight: blockHeight,
					Inferer:     InfererAddress1,
					Value:       m.InfererValues[0].Value,
					Values:      m.InfererValues,
					ExtraData:   nil,
					Proof:       "",
				},
				Forecast: &types.InputForecast{
					TopicId:     topicId,
					BlockHeight: blockHeight,
					Forecaster:  InfererAddress1,
					ForecastElements: []*types.InputForecastElement{
						{
							Inferer: InfererAddress1,
							Value:   m.InfererValues[0].Value,
						},
					},
					ExtraData: nil,
				},
			},
			InferencesForecastsBundleSignature: nil,
			Pubkey:                             "",
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

// latestUnfulfilledWorkerNonce returns the newest worker nonce the scheduler has opened
// and not yet closed. Zero means none is open.
func latestUnfulfilledWorkerNonce(m testCommon.TestConfig, topicID uint64) (int64, error) {
	ctx := context.Background()
	resp, err := m.Client.QueryEmissions().GetUnfulfilledWorkerNonces(ctx, &types.GetUnfulfilledWorkerNoncesRequest{
		TopicId: topicID,
	})
	if err != nil {
		return 0, err
	}
	if resp.Nonces == nil {
		return 0, nil
	}
	var latest int64
	for _, nonce := range resp.Nonces.Nonces {
		if nonce != nil && nonce.BlockHeight > latest {
			latest = nonce.BlockHeight
		}
	}
	return latest, nil
}

// submitWorkerOnOpenNonce keeps trying the newest open worker nonce until one is accepted.
// The nonce is the epoch start height. EpochLastEnded is the churn height and is not opened.
func submitWorkerOnOpenNonce(m testCommon.TestConfig, topic *types.Topic) (int64, error) {
	deadline := time.Now().Add(90 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		nonce, err := latestUnfulfilledWorkerNonce(m, topic.Id)
		if err != nil {
			return 0, err
		}
		if nonce == 0 {
			time.Sleep(time.Second)
			continue
		}
		m.T.Log(time.Now(), "Inserting worker bundle for nonce", nonce)
		err = InsertSingleWorkerPayload(m, topic, nonce)
		if err == nil {
			return nonce, nil
		}
		if !retryableSubmissionErr(err) {
			return 0, err
		}
		lastErr = err
		m.T.Log(time.Now(), "Worker payload not accepted yet:", err)
		time.Sleep(time.Second)
	}
	if lastErr == nil {
		lastErr = errors.New("no unfulfilled worker nonce")
	}
	return 0, fmt.Errorf("worker window did not accept a payload: %w", lastErr)
}

// submitReputerOnOpenWindow submits losses for the worker nonce once its reputer window is open.
func submitReputerOnOpenWindow(m testCommon.TestConfig, topic *types.Topic, nonce int64) error {
	deadline := time.Now().Add(90 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		err := InsertReputerBundle(m, topic, nonce)
		if err == nil {
			return nil
		}
		if !retryableSubmissionErr(err) {
			return err
		}
		lastErr = err
		m.T.Log(time.Now(), "Reputer payload not accepted yet:", err)
		time.Sleep(time.Second)
	}
	if lastErr == nil {
		lastErr = errors.New("reputer window did not open")
	}
	return fmt.Errorf("reputer window did not accept a payload: %w", lastErr)
}

func waitForNetworkLossBundle(m testCommon.TestConfig, topicID uint64, nonce int64) error {
	ctx := context.Background()
	deadline := time.Now().Add(90 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		result, err := m.Client.QueryEmissions().GetNetworkLossBundleAtBlock(ctx, &types.GetNetworkLossBundleAtBlockRequest{
			TopicId:     topicID,
			BlockHeight: nonce,
		})
		if err == nil && result != nil && result.LossBundle != nil {
			return nil
		}
		lastErr = err
		time.Sleep(time.Second)
	}
	if lastErr == nil {
		lastErr = errors.New("loss bundle missing")
	}
	return fmt.Errorf("network loss bundle was not stored for nonce %d: %w", nonce, lastErr)
}

// retryableSubmissionErr reports errors that mean the epoch window is not ready yet.
func retryableSubmissionErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, fragment := range []string{
		"unfulfilled nonce not found",
		"nonce still unfulfilled",
		"window not available",
		"window not open",
		"not in worker submission",
		"not in reputer submission",
	} {
		if strings.Contains(msg, fragment) {
			return true
		}
	}
	return false
}

// register alice as a reputer in topic 1, then check success
func InsertReputerBundle(m testCommon.TestConfig, topic *types.Topic, BlockHeightCurrent int64) error {
	ctx := context.Background()
	topicId := topic.Id
	// Define inferer address as Bob's address, reputer as Alice's
	workerAddr := m.BobAddr
	reputerAddr := m.AliceAddr
	// Reputer Nonce
	reputerNonce := &types.Nonce{
		BlockHeight: BlockHeightCurrent,
	}

	reputerValueBundle := &types.InputValueBundle{
		TopicId: topicId,
		Reputer: reputerAddr,
		ReputerRequestNonce: &types.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
		},
		ExtraData:     nil,
		CombinedValue: alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		InfererValues: []*types.InputWorkerAttributedValue{
			{
				Worker: workerAddr,
				Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
			},
		},
		ForecasterValues: []*types.InputWorkerAttributedValue{
			{
				Worker: workerAddr,
				Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
			},
		},
		NaiveValue: alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
		OneOutInfererValues: []*types.InputWithheldWorkerAttributedValue{
			// There cannot be a 1-out inferer value if there is just 1 inferer => this will be ignored by msgserver
			{
				Worker: workerAddr,
				Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
			},
		},
		OneOutForecasterValues: []*types.InputWithheldWorkerAttributedValue{
			{
				Worker: workerAddr,
				Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
			},
		},
		// Just as valid:
		// OneOutInfererValues:    []*types.WithheldWorkerAttributedValue{},
		// OneOutForecasterValues: []*types.WithheldWorkerAttributedValue{},
		OneInForecasterValues: []*types.InputWorkerAttributedValue{
			{
				Worker: workerAddr,
				Value:  alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(100)),
			},
		},
		OneOutInfererForecasterValues: nil,
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
		ReputerValueBundle: &types.InputReputerValueBundle{
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

func ValidateGetNetworkLossBundle(m testCommon.TestConfig, topicId uint64, BlockHeightCurrent int64) {
	ctx := context.Background()
	result, err := m.Client.QueryEmissions().GetNetworkLossBundleAtBlock(ctx,
		&types.GetNetworkLossBundleAtBlockRequest{
			TopicId:     topicId,
			BlockHeight: BlockHeightCurrent,
		},
	)
	require.NoError(m.T, err)
	require.NotNil(m.T, result, "Result should not be nil")
	require.NotNil(m.T, result.LossBundle, "Retrieved data should match inserted data")
}

func addGlobalActor(m testCommon.TestConfig, address string) {
	ctx := context.Background()
	addGlobalActorRequest := &types.AddToGlobalWhitelistRequest{
		Sender:  m.AliceAddr,
		Address: address,
	}
	txResp, err := m.Client.BroadcastTx(ctx, m.AliceAcc, addGlobalActorRequest)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
	addGlobalActorResponse := &types.AddToGlobalWhitelistResponse{}
	err = txResp.Decode(addGlobalActorResponse)
	require.NoError(m.T, err)
}

// Register two actors and check their registrations went through
func WorkerInferenceAndForecastChecks(m testCommon.TestConfig) {
	ctx := context.Background()
	m.T.Log(time.Now(), "--- START  Worker Inference, Forecast and Reputation test ---")
	addGlobalActor(m, m.BobAddr)
	addGlobalActor(m, m.AliceAddr)

	topicResponse, err := m.Client.QueryEmissions().GetTopic(ctx, &types.GetTopicRequest{TopicId: m.TopicID})
	require.NoError(m.T, err)
	topic := topicResponse.Topic

	m.T.Log(time.Now(), "--- Insert Worker Bundle ---")
	nonce, err := submitWorkerOnOpenNonce(m, topic)
	if err != nil {
		m.T.Log(time.Now(), "--- Failed inserting worker payload ---")
		require.NoError(m.T, err)
	}

	m.T.Log(time.Now(), "--- Insert Reputer Bundle ---")
	err = submitReputerOnOpenWindow(m, topic, nonce)
	if err != nil {
		m.T.Log(time.Now(), "--- Failed inserting reputer payload ---")
		require.NoError(m.T, err)
	}

	m.T.Log(time.Now(), "--- Wait for network loss bundle ---")
	err = waitForNetworkLossBundle(m, topic.Id, nonce)
	if err != nil {
		m.T.Log(time.Now(), "--- Failed waiting for network loss bundle ---")
		require.NoError(m.T, err)
	}
	ValidateGetNetworkLossBundle(m, topic.Id, nonce)
	m.T.Log(time.Now(), "--- END  Worker Inference, Forecast and Reputation test ---")
}
