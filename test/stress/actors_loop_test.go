package stress_test

import (
	"context"
	"encoding/hex"
	"math/rand"
	"sync/atomic"
	"time"

	alloraMath "github.com/allora-network/allora-chain/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
	stresscommon "github.com/allora-network/allora-chain/test/stress/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
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
				latestNonceHeightActedUpon = latestOpenWorkerNonce.BlockHeight
				m.T.Logf("Worker nonce opened for topic: %d at height: %d", topicId, latestOpenWorkerNonce.BlockHeight)
				workers := data.getWorkersForTopic(topicId)

				wasError := createAndSendWorkerPayloads(m, data, topicId, workers, latestOpenWorkerNonce.BlockHeight)
				if wasError {
					m.T.Logf("Error building and committing worker payload for topic: %d", topicId)
				}
				m.T.Logf("Successfully built and committed worker payload for topic: %d for %v workers", topicId, len(workers))
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
				latestNonceHeightActedUpon = latestOpenReputerNonce
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
	workers []StressActor,
	workerNonce int64,
) bool {
	completed := atomic.Int32{}
	start := time.Now()

	m.T.Logf("Starting worker payload creation for %d workers in topic: %d", len(workers), topicId)

	for _, worker := range workers {
		go func(worker StressActor) {
			defer func() {
				count := completed.Add(1)
				if int(count)%1000 == 0 || count == int32(len(workers)) {
					elapsed := time.Since(start)
					m.T.Logf("Processed %d/%d worker payloads (%.2f%%) for topic: %d in %s",
						count, len(workers),
						float64(count)/float64(len(workers))*100,
						topicId,
						elapsed)
				}
			}()

			workerData, err := createWorkerDataBundle(m, topicId, workerNonce, worker, workers)
			if err != nil {
				m.T.Logf("Error creating worker data bundle: %v", err.Error())
				requireNoError(m.T, data.failOnErr, err)
				return
			}

			_, updatedSeq, err := stresscommon.SendDataWithRetry(worker.params, &emissionstypes.InsertWorkerPayloadRequest{
				Sender:           worker.addr,
				WorkerDataBundle: workerData,
			})
			if err != nil {
				m.T.Logf("Error sending worker payload: %v", err.Error())
			}
			worker.params.Sequence = updatedSeq
		}(worker)
	}

	totalTime := time.Since(start)
	m.T.Logf("Total worker payload creation time: %s", totalTime)

	return false
}

// Create inferences and forecasts for a worker
func createWorkerDataBundle(
	m *testcommon.TestConfig,
	topicId uint64,
	blockHeight int64,
	inferer StressActor,
	workers []StressActor,
) (*emissionstypes.WorkerDataBundle, error) {
	// TODO: Add forecasts for specific workers (top workers)
	// Iterate workerAddresses to get the worker address, and generate as many forecasts as there are workers
	// forecastElements := make([]*emissionstypes.ForecastElement, 0)
	// for key := range workers {
	// 	forecastElements = append(forecastElements, &emissionstypes.ForecastElement{
	// 		Inferer: workers[key].addr,
	// 		Value:   alloraMath.NewDecFromInt64(int64(m.Client.Rand.Intn(51) + 50)),
	// 	})
	// }
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
			Forecast: nil,
			// Forecast: &emissionstypes.Forecast{
			// 	TopicId:          topicId,
			// 	BlockHeight:      blockHeight,
			// 	Forecaster:       infererAddress,
			// 	ForecastElements: nil,
			// 	ExtraData:        nil,
			// },
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

	sig, err := inferer.params.PrivKey.Sign(src)
	if err != nil {
		return nil, err
	}

	workerPublicKeyBytes := inferer.params.PubKey.Bytes()
	workerDataBundle.InferencesForecastsBundleSignature = sig
	workerDataBundle.Pubkey = hex.EncodeToString(workerPublicKeyBytes)

	return workerDataBundle, nil
}

// Create and send reputer payloads
func createAndSendReputerPayloads(
	m *testcommon.TestConfig,
	data *SimulationData,
	topicId uint64,
	reputers []StressActor,
	workers []string,
	workerNonce int64,
) bool {
	completed := atomic.Int32{}
	start := time.Now()

	reputerNonce := &emissionstypes.Nonce{
		BlockHeight: workerNonce,
	}

	m.T.Logf("Starting reputer payload creation for %d reputers in topic: %d", len(reputers), topicId)

	for _, reputer := range reputers {
		go func(reputer StressActor) {
			defer func() {
				count := completed.Add(1)
				if int(count)%1000 == 0 || count == int32(len(reputers)) {
					elapsed := time.Since(start)
					m.T.Logf("Processed %d/%d reputer payloads (%.2f%%) for topic: %d in %s",
						count, len(reputers),
						float64(count)/float64(len(reputers))*100,
						topicId,
						elapsed)
				}
			}()

			valueBundle, err := createReputerValueBundle(m, topicId, reputer, workers, reputerNonce)
			if err != nil {
				m.T.Logf("Error creating reputer value bundle: %v", err.Error())
				requireNoError(m.T, data.failOnErr, err)
				return
			}

			_, updatedSeq, err := stresscommon.SendDataWithRetry(reputer.params, &emissionstypes.InsertReputerPayloadRequest{
				Sender:             reputer.addr,
				ReputerValueBundle: valueBundle,
			})
			if err != nil {
				m.T.Logf("Error sending reputer payload: %v", err.Error())
			}
			reputer.params.Sequence = updatedSeq
		}(reputer)
	}

	totalTime := time.Since(start)
	m.T.Logf("Total reputer payload creation time: %s", totalTime)

	return false
}

// Generate the same valueBundle for a reputer
func createReputerValueBundle(
	m *testcommon.TestConfig,
	topicId uint64,
	reputer StressActor,
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

	sig, err := reputer.params.PrivKey.Sign(src)
	if err != nil {
		return nil, err
	}

	// Create a InsertReputerPayloadRequest message
	reputerValueBundle := &emissionstypes.ReputerValueBundle{
		ValueBundle: &valueBundle,
		Signature:   sig,
		Pubkey:      reputer.params.PubKey.String(),
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
