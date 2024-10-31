package newstress_test

import (
	"context"
	"encoding/hex"
	"math/rand"
	"time"

	alloraMath "github.com/allora-network/allora-chain/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

// Will check for nonce opened every 4s and if opened, will produce inferences and forecasts
func runTopicWorkersLoop(
	m *testcommon.TestConfig,
	data *SimulationData,
	topicId uint64,
) error {
	latestNonceHeightActedUpon := int64(0)
	for {
		latestOpenWorkerNonce, err := getLatestOpenWorkerNonceByTopicId(m, topicId)
		if err != nil {
			return err
		} else {
			if latestOpenWorkerNonce.BlockHeight > latestNonceHeightActedUpon {
				m.T.Logf("Worker nonce opened for topic: %d at height: %d", topicId, latestOpenWorkerNonce.BlockHeight)
				workers := data.getWorkersForTopic(topicId)
				m.T.Logf("Building and committing worker payload for topic: %d", topicId)
				wasError := createAndSendWorkerPayloads(m, data, topicId, workers, latestOpenWorkerNonce.BlockHeight)
				if wasError {
					m.T.Logf("Error building and committing worker payload for topic: %d", topicId)
				}
				m.T.Logf("Successfully built and committed worker payload for topic: %d for %v workers", topicId, len(workers))
				latestNonceHeightActedUpon = latestOpenWorkerNonce.BlockHeight
			}
		}
		time.Sleep(4 * time.Second)
	}
}

// Will check for nonce opened every 4s and if opened, will produce reputation
func runReputersProcess(
	m *testcommon.TestConfig,
	data *SimulationData,
	topicId uint64,
) error {
	latestNonceHeightActedUpon := int64(0)
	for {
		latestOpenReputerNonce, err := getOldestReputerNonceByTopicId(m, topicId)
		if err != nil {
			m.T.Logf("Error getting latest open reputer nonce on topic - node availability issue?: %v", err)
		} else {
			if latestOpenReputerNonce > latestNonceHeightActedUpon {
				m.T.Logf("Reputer nonce opened for topic: %d at height: %d", topicId, latestOpenReputerNonce)
				activeWorkersAddresses, err := getActiveWorkersForTopic(m, topicId, latestOpenReputerNonce)
				if err != nil {
					return err
				}
				reputers := data.getReputersForTopic(topicId)
				m.T.Logf("Building and committing reputer payload for topic: %d", topicId)
				wasError := createAndSendReputerPayloads(m, data, topicId, reputers, activeWorkersAddresses, latestOpenReputerNonce)
				if wasError {
					m.T.Logf("Error building and committing reputer payload for topic: %d", topicId)
				}
				m.T.Logf("Successfully built and committed reputer payload for topic: %d for %v reputers", topicId, len(reputers))
				latestNonceHeightActedUpon = latestOpenReputerNonce
			}
		}
		time.Sleep(4 * time.Second)
	}
}

// Get the latest open worker nonce for a topic
func getLatestOpenWorkerNonceByTopicId(m *testcommon.TestConfig, topicId uint64) (*emissionstypes.Nonce, error) {
	ctx := context.Background()

	res, err := m.Client.QueryEmissions().GetUnfulfilledWorkerNonces(
		ctx,
		&emissionstypes.GetUnfulfilledWorkerNoncesRequest{TopicId: topicId},
	)
	if err != nil {
		return &emissionstypes.Nonce{}, err
	}

	if len(res.Nonces.Nonces) == 0 {
		return &emissionstypes.Nonce{}, err
	}

	return res.Nonces.Nonces[0], nil
}

// Get the oldest reputer nonce for a topic
func getOldestReputerNonceByTopicId(m *testcommon.TestConfig, topicId uint64) (int64, error) {
	ctx := context.Background()

	res, err := m.Client.QueryEmissions().GetUnfulfilledReputerNonces(
		ctx,
		&emissionstypes.GetUnfulfilledReputerNoncesRequest{TopicId: topicId},
	)
	if err != nil {
		return 0, err
	}

	if len(res.Nonces.Nonces) == 0 {
		return 0, nil
	}

	return res.Nonces.Nonces[len(res.Nonces.Nonces)-1].ReputerNonce.BlockHeight, nil
}

// Get the active workers for a topic at a given block height to use for reputer payloads
func getActiveWorkersForTopic(m *testcommon.TestConfig, topicId uint64, blockHeight int64) ([]string, error) {
	ctx := context.Background()
	res, err := m.Client.QueryEmissions().GetInferencesAtBlock(
		ctx,
		&emissionstypes.GetInferencesAtBlockRequest{TopicId: topicId, BlockHeight: blockHeight},
	)
	if err != nil {
		return []string{}, err
	}

	workers := make([]string, 0)
	for _, inference := range res.Inferences.Inferences {
		workers = append(workers, inference.Inferer)
	}

	return workers, nil
}

// Create and send worker payloads
func createAndSendWorkerPayloads(
	m *testcommon.TestConfig,
	data *SimulationData,
	topicId uint64,
	workers []Actor,
	workerNonce int64,
) bool {
	wasErr := false
	ctx := context.Background()
	for _, worker := range workers {
		workerData, err := createWorkerDataBundle(m, topicId, workerNonce, worker, workers)
		if err != nil {
			m.T.Logf("Error creating worker data bundle: %v", err.Error())
		}
		requireNoError(m.T, data.failOnErr, err)
		m.Client.BroadcastTxAsync(ctx, worker.acc, &emissionstypes.InsertWorkerPayloadRequest{
			Sender:           worker.addr,
			WorkerDataBundle: workerData,
		})
	}

	return wasErr
}

// Create inferences and forecasts for a worker
func createWorkerDataBundle(
	m *testcommon.TestConfig,
	topicId uint64,
	blockHeight int64,
	inferer Actor,
	workers []Actor,
) (*emissionstypes.WorkerDataBundle, error) {
	// Iterate workerAddresses to get the worker address, and generate as many forecasts as there are workers
	forecastElements := make([]*emissionstypes.ForecastElement, 0)
	for key := range workers {
		forecastElements = append(forecastElements, &emissionstypes.ForecastElement{
			Inferer: workers[key].addr,
			Value:   alloraMath.NewDecFromInt64(int64(m.Client.Rand.Intn(51) + 50)),
		})
	}
	infererAddress := inferer.addr
	infererValue := alloraMath.NewDecFromInt64(int64(m.Client.Rand.Intn(300) + 3000))

	workerDataBundle := &emissionstypes.WorkerDataBundle{
		Worker: infererAddress,
		Nonce: &emissionstypes.Nonce{
			BlockHeight: blockHeight,
		},
		TopicId: topicId,
		InferenceForecastsBundle: &emissionstypes.InferenceForecastBundle{
			Inference: &emissionstypes.Inference{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     infererAddress,
				Value:       infererValue,
				ExtraData:   nil,
				Proof:       "",
			},
			Forecast: &emissionstypes.Forecast{
				TopicId:          topicId,
				BlockHeight:      blockHeight,
				Forecaster:       infererAddress,
				ForecastElements: forecastElements,
				ExtraData:        nil,
			},
		},
		InferencesForecastsBundleSignature: nil,
		Pubkey:                             "",
	}

	// Sign
	src := make([]byte, 0)
	src, err := workerDataBundle.InferenceForecastsBundle.XXX_Marshal(src, true)
	if err != nil {
		return nil, err
	}

	sig, pubKey, err := m.Client.Context().Keyring.Sign(inferer.name, src, signing.SignMode_SIGN_MODE_DIRECT)
	if err != nil {
		return nil, err
	}
	workerPublicKeyBytes := pubKey.Bytes()
	workerDataBundle.InferencesForecastsBundleSignature = sig
	workerDataBundle.Pubkey = hex.EncodeToString(workerPublicKeyBytes)

	return workerDataBundle, nil
}

// Create and send reputer payloads
func createAndSendReputerPayloads(
	m *testcommon.TestConfig,
	data *SimulationData,
	topicId uint64,
	reputers []Actor,
	workers []string,
	workerNonce int64,
) bool {
	wasErr := false
	reputerNonce := &emissionstypes.Nonce{
		BlockHeight: workerNonce,
	}
	ctx := context.Background()
	for _, reputer := range reputers {
		valueBundle, err := createReputerValueBundle(m, topicId, reputer, workers, reputerNonce)
		if err != nil {
			m.T.Logf("Error creating reputer value bundle: %v", err.Error())
		}
		requireNoError(m.T, data.failOnErr, err)
		m.Client.BroadcastTxAsync(ctx, reputer.acc, &emissionstypes.InsertReputerPayloadRequest{
			Sender:             reputer.addr,
			ReputerValueBundle: valueBundle,
		})
	}
	return wasErr
}

// Generate the same valueBundle for a reputer
func createReputerValueBundle(
	m *testcommon.TestConfig,
	topicId uint64,
	reputer Actor,
	workers []string,
	reputerNonce *emissionstypes.Nonce,
) (*emissionstypes.ReputerValueBundle, error) {
	valueBundle := emissionstypes.ValueBundle{
		TopicId:                topicId,
		Reputer:                reputer.addr,
		ExtraData:              nil,
		CombinedValue:          alloraMath.NewDecFromInt64(100),
		InfererValues:          generateWorkerAttributedValueLosses(m, workers, 3000, 3500),
		ForecasterValues:       generateWorkerAttributedValueLosses(m, workers, 50, 50),
		NaiveValue:             alloraMath.NewDecFromInt64(100),
		OneOutInfererValues:    generateWithheldWorkerAttributedValueLosses(workers, 50, 50),
		OneOutForecasterValues: generateWithheldWorkerAttributedValueLosses(workers, 50, 50),
		OneInForecasterValues:  generateWorkerAttributedValueLosses(m, workers, 50, 50),
		ReputerRequestNonce: &emissionstypes.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
		},
		OneOutInfererForecasterValues: nil,
	}
	// Sign
	src := make([]byte, 0)
	src, err := valueBundle.XXX_Marshal(src, true)
	if err != nil {
		return nil, err
	}

	valueBundleSignature, pubKey, err := m.Client.Context().Keyring.Sign(reputer.name, src, signing.SignMode_SIGN_MODE_DIRECT)
	if err != nil {
		return nil, err
	}
	reputerPublicKeyBytes := pubKey.Bytes()

	// Create a InsertReputerPayloadRequest message
	reputerValueBundle := &emissionstypes.ReputerValueBundle{
		ValueBundle: &valueBundle,
		Signature:   valueBundleSignature,
		Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
	}

	return reputerValueBundle, nil
}

// For every worker, generate a worker attributed value
func generateWorkerAttributedValueLosses(
	m *testcommon.TestConfig,
	workers []string,
	lowLimit,
	sum int,
) []*emissionstypes.WorkerAttributedValue {
	values := make([]*emissionstypes.WorkerAttributedValue, 0)
	for _, worker := range workers {
		values = append(values, &emissionstypes.WorkerAttributedValue{
			Worker: worker,
			Value:  alloraMath.NewDecFromInt64(int64(m.Client.Rand.Intn(lowLimit) + sum)),
		})
	}
	return values
}

// For every worker, generate a withheld worker attribute value
func generateWithheldWorkerAttributedValueLosses(
	workers []string,
	lowLimit,
	sum int,
) []*emissionstypes.WithheldWorkerAttributedValue {
	values := make([]*emissionstypes.WithheldWorkerAttributedValue, 0)
	for _, worker := range workers {
		values = append(values, &emissionstypes.WithheldWorkerAttributedValue{
			Worker: worker,
			Value:  alloraMath.NewDecFromInt64(int64(rand.Intn(lowLimit) + sum)),
		})
	}
	return values
}
