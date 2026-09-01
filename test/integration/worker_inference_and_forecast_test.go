package integration_test

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
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
	// Nonce: calculate from EpochLastRan + EpochLength
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

// Worker Bob inserts inference and forecast against a scheduler epoch's LegacyNonce
// (StartBlockHeight), not Topic.EpochLastEnded + EpochLength.
func InsertWorkerBundle(m testCommon.TestConfig, topic *types.Topic, blockHeight int64) error {
	ctx := context.Background()
	currentBlock, err := m.Client.BlockHeight(ctx)
	if err != nil {
		return err
	}
	m.T.Log(time.Now(), "Inserting worker bundle for start_block_height: ", blockHeight, "; Current block: ", currentBlock)
	return InsertSingleWorkerPayload(m, topic, blockHeight)
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

// Wall-clock epoch lifecycle on scheduler-managed topics:
//   - Live worker/reputer nonces are Epoch.StartBlockHeight (LegacyNonce), not
//     Topic.EpochLastEnded + EpochLength. EpochLength / WorkerSubmissionWindow /
//     GroundTruthLag are seconds on this path.
//   - Scheduler due-checks use BlockTime.After, so a task scheduled at T is not
//     due at T; the next block after T must run BeginBlock.
//   - CloseEpochReputerWindow and CompleteEpoch are both due at reputer CloseAt;
//     type dependencies run close before complete. Cross-topic Complete vs
//     StartNewEpoch is weight-arbitrated. Height-based fuzz waits mixed with
//     second-valued windows will desync from the live epoch.
const epochLifecycleTimeout = 2 * time.Minute

func queryTopicEpochs(m testCommon.TestConfig, topicId uint64) ([]*types.Epoch, error) {
	resp, err := m.Client.QueryEmissions().GetTopicEpochs(
		context.Background(),
		&types.GetTopicEpochsRequest{TopicId: topicId},
	)
	if err != nil {
		return nil, err
	}
	return resp.Epochs, nil
}

func findOpenWorkerEpoch(epochs []*types.Epoch) *types.Epoch {
	now := time.Now()
	var best *types.Epoch
	for _, epoch := range epochs {
		if epoch == nil || epoch.State != types.EpochState_WORKER_SUBMISSION || epoch.WorkerSubmissionWindow == nil {
			continue
		}
		window := epoch.WorkerSubmissionWindow
		if now.Before(window.OpenAt) || !now.Before(window.CloseAt.Add(-500*time.Millisecond)) {
			continue
		}
		if best == nil || epoch.StartBlockHeight > best.StartBlockHeight {
			best = epoch
		}
	}
	return best
}

func waitForOpenWorkerEpoch(m testCommon.TestConfig, topicId uint64) (*types.Epoch, error) {
	deadline := time.Now().Add(epochLifecycleTimeout)
	for time.Now().Before(deadline) {
		epochs, err := queryTopicEpochs(m, topicId)
		if err != nil {
			return nil, err
		}
		if epoch := findOpenWorkerEpoch(epochs); epoch != nil {
			return epoch, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil, fmt.Errorf("no open worker epoch for topic %d within %s", topicId, epochLifecycleTimeout)
}

func waitForEpochState(m testCommon.TestConfig, topicId uint64, startBlockHeight int64, want types.EpochState) (*types.Epoch, error) {
	deadline := time.Now().Add(epochLifecycleTimeout)
	for time.Now().Before(deadline) {
		epochs, err := queryTopicEpochs(m, topicId)
		if err != nil {
			return nil, err
		}
		for _, epoch := range epochs {
			if epoch != nil && epoch.StartBlockHeight == startBlockHeight && epoch.State == want {
				return epoch, nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil, fmt.Errorf("epoch at start_block_height %d for topic %d never reached %s", startBlockHeight, topicId, want)
}

func waitForEpochAbsent(m testCommon.TestConfig, topicId uint64, startBlockHeight int64) error {
	deadline := time.Now().Add(epochLifecycleTimeout)
	for time.Now().Before(deadline) {
		epochs, err := queryTopicEpochs(m, topicId)
		if err != nil {
			return err
		}
		found := false
		for _, epoch := range epochs {
			if epoch != nil && epoch.StartBlockHeight == startBlockHeight {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("epoch at start_block_height %d for topic %d still in-flight after %s", startBlockHeight, topicId, epochLifecycleTimeout)
}

// Register two actors and check their registrations went through
func WorkerInferenceAndForecastChecks(m testCommon.TestConfig) {
	m.T.Log(time.Now(), "--- START  Worker Inference, Forecast and Reputation test ---")
	addGlobalActor(m, m.BobAddr)
	addGlobalActor(m, m.AliceAddr)

	topicResponse, err := m.Client.QueryEmissions().GetTopic(
		context.Background(),
		&types.GetTopicRequest{TopicId: m.TopicID},
	)
	require.NoError(m.T, err)
	topic := topicResponse.Topic

	m.T.Log(time.Now(), "--- Waiting for open worker epoch ---")
	var submittedHeight int64
	submittedHeight, err = RunWithRetry(m, 5, time.Second, func() (int64, error) {
		epoch, err := waitForOpenWorkerEpoch(m, topic.Id)
		if err != nil {
			return 0, err
		}
		m.T.Log(time.Now(), "--- Insert Worker Bundle ---")
		if err := InsertWorkerBundle(m, topic, epoch.StartBlockHeight); err != nil {
			return 0, err
		}
		return epoch.StartBlockHeight, nil
	})
	require.NoError(m.T, err, "inserting worker payload")

	m.T.Log(time.Now(), "--- Waiting for reputer submission window ---")
	_, err = waitForEpochState(m, topic.Id, submittedHeight, types.EpochState_REPUTER_SUBMISSION)
	require.NoError(m.T, err, "waiting for reputer window")

	m.T.Log(time.Now(), "--- Insert Reputer Bundle ---")
	err = InsertReputerBundle(m, topic, submittedHeight)
	require.NoError(m.T, err, "inserting reputer payload")

	m.T.Log(time.Now(), "--- Waiting for epoch completion ---")
	err = waitForEpochAbsent(m, topic.Id, submittedHeight)
	require.NoError(m.T, err, "waiting for completed epoch to leave store")

	ValidateGetNetworkLossBundle(m, topic.Id, submittedHeight)
	m.T.Log(time.Now(), "--- END  Worker Inference, Forecast and Reputation test ---")
}

// RunWithRetry retries a function that returns an error, n times
func RunWithRetry(m testCommon.TestConfig, retryCount int, sleep time.Duration, operation func() (int64, error)) (int64, error) {
	var (
		err error
		val int64
	)
	for i := 0; i < retryCount; i++ {
		val, err = operation()
		if err == nil {
			return val, nil // Success, no need to retry
		}
		m.T.Log(time.Now(), fmt.Sprintf("Attempt %d/%d failed, error: %s\n", i+1, retryCount, err))
		time.Sleep(sleep) // Optional: wait before retrying
	}
	return 0, fmt.Errorf("after %d attempts, last error: %s", retryCount, err)
}
