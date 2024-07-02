package invariant_test

import (
	"context"
	"encoding/hex"
	"math/rand"

	cosmossdk_io_math "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	testcommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"
)

func doInferenceAndReputation(
	m *testcommon.TestConfig,
	leaderWorker Actor,
	leaderReputer Actor,
	amount *cosmossdk_io_math.Int,
	topicId uint64,
	data *SimulationData,
	iteration int,
) {
	wasErr := false
	iterLog(
		m.T,
		iteration,
		"producing inference and reputation for topic id",
		topicId,
		" with leader worker",
		leaderWorker,
		" and leader reputer",
		leaderReputer,
	)
	ctx := context.Background()
	resp, err := m.Client.QueryEmissions().GetTopic(ctx, &emissionstypes.QueryTopicRequest{
		TopicId: topicId,
	})
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	topic := resp.Topic
	blockHeightEpochStart := topic.EpochLastEnded + topic.EpochLength
	workers := data.getWorkersForTopic(topicId)
	iterLog(m.T, iteration, " starting worker bulk topic id ", topicId, " leader worker ", leaderWorker, " workers ", workers)
	workerBulkErrored := insertWorkerBulk(m, data, topic, leaderWorker, workers, blockHeightEpochStart)
	if workerBulkErrored {
		iterFailLog(m.T, iteration, "worker bulk errored topic ", topicId)
		return
	}
	iterLog(m.T, iteration, "produced inference for topic id", topicId)
	iterLog(m.T, iteration, "waiting for block epoch start", blockHeightEpochStart)
	err = m.Client.WaitForBlockHeight(ctx, blockHeightEpochStart)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	blockHeightNow, err := m.Client.BlockHeight(ctx)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	reputers := data.getReputersForTopicWithStake(topicId)
	iterLog(
		m.T, iteration, " starting reputer bulk topic id ", topicId, "leader reputer ", leaderReputer,
		" workers ", workers, " reputers ", reputers, " block height epoch start ", blockHeightEpochStart,
		" block height  now ", blockHeightNow,
	)
	reputerBulkErrored := insertReputerBulk(m, data, topic, leaderReputer, reputers, workers, blockHeightNow, blockHeightEpochStart)
	if reputerBulkErrored {
		iterFailLog(m.T, iteration, "reputer bulk errored topic", topicId)
		return
	}
	if !wasErr {
		data.counts.incrementDoInferenceAndReputationCount()
		iterSuccessLog(m.T, iteration, "produced reputer and worker bulks for topic id ", topicId)
	} else {
		iterFailLog(m.T, iteration, "failed to produce reputer and worker bulks for topic id ", topicId)
	}
}

// determine if this state transition is worth trying based on our knowledge of the state
func findActiveTopics(
	m *testcommon.TestConfig,
	data *SimulationData,
) []*emissionstypes.Topic {
	// first off someone has to be registered for both working and reputing in general
	if !anyReputersRegistered(data) || !anyWorkersRegistered(data) {
		return nil
	}
	ctx := context.Background()
	response, err := m.Client.QueryEmissions().GetActiveTopics(ctx, &emissionstypes.QueryActiveTopicsRequest{
		Pagination: &emissionstypes.SimpleCursorPaginationRequest{
			Limit: 10,
		},
	})
	requireNoError(m.T, data.failOnErr, err)
	return response.Topics
}

// Inserts bulk inference and forecast data for a worker
func insertWorkerBulk(
	m *testcommon.TestConfig,
	data *SimulationData,
	topic *emissionstypes.Topic,
	leaderWorker Actor,
	workers []Actor,
	blockHeight int64,
) bool {
	// Get Bundles
	workerDataBundles := make([]*emissionstypes.WorkerDataBundle, 0)
	for _, worker := range workers {
		workerDataBundles = append(workerDataBundles,
			generateSingleWorkerBundle(m, topic.Id, blockHeight, worker, workers))
	}
	if len(workerDataBundles) == 0 {
		return true
	}
	return insertLeaderWorkerBulk(m, data, topic.Id, blockHeight, leaderWorker, workerDataBundles)
}

// create inferences and forecasts for a worker
func generateSingleWorkerBundle(
	m *testcommon.TestConfig,
	topicId uint64,
	blockHeight int64,
	inferer Actor,
	workers []Actor,
) *emissionstypes.WorkerDataBundle {
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

	// Create a MsgInsertBulkReputerPayload message
	workerDataBundle := &emissionstypes.WorkerDataBundle{
		Worker: infererAddress,
		InferenceForecastsBundle: &emissionstypes.InferenceForecastBundle{
			Inference: &emissionstypes.Inference{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Inferer:     infererAddress,
				Value:       infererValue,
			},
			Forecast: &emissionstypes.Forecast{
				TopicId:          topicId,
				BlockHeight:      blockHeight,
				Forecaster:       infererAddress,
				ForecastElements: forecastElements,
			},
		},
	}

	// Sign
	src := make([]byte, 0)
	src, err := workerDataBundle.InferenceForecastsBundle.XXX_Marshal(src, true)
	require.NoError(m.T, err, "Marshall reputer value bundle should not return an error")

	sig, pubKey, err := m.Client.Context().Keyring.Sign(inferer.name, src, signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(m.T, err, "Sign should not return an error")
	workerPublicKeyBytes := pubKey.Bytes()
	workerDataBundle.InferencesForecastsBundleSignature = sig
	workerDataBundle.Pubkey = hex.EncodeToString(workerPublicKeyBytes)

	return workerDataBundle
}

// Inserts worker bulk, given a topic, blockHeight, and leader worker address (which should exist in the keyring)
func insertLeaderWorkerBulk(
	m *testcommon.TestConfig,
	data *SimulationData,
	topicId uint64,
	blockHeight int64,
	leaderWorker Actor,
	WorkerDataBundles []*emissionstypes.WorkerDataBundle) bool {
	wasErr := false
	nonce := emissionstypes.Nonce{BlockHeight: blockHeight}

	// Create a MsgInsertBulkReputerPayload message
	workerMsg := &emissionstypes.MsgInsertBulkWorkerPayload{
		Sender:            leaderWorker.addr,
		Nonce:             &nonce,
		TopicId:           topicId,
		WorkerDataBundles: WorkerDataBundles,
	}
	// serialize workerMsg to json and print
	LeaderAcc, err := m.Client.AccountRegistryGetByName(leaderWorker.name)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, LeaderAcc, workerMsg)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		return wasErr
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	return orErr(wasErr, err)
}

// reputers submit their assessment of the quality of workers' work compared to ground truth
func insertReputerBulk(
	m *testcommon.TestConfig,
	data *SimulationData,
	topic *emissionstypes.Topic,
	leaderReputer Actor,
	reputers,
	workers []Actor,
	BlockHeightCurrent,
	BlockHeightEval int64,
) bool {
	wasErr := false
	// Nonce: calculate from EpochLastRan + EpochLength
	topicId := topic.Id
	// Nonces are last two blockHeights
	reputerNonce := &emissionstypes.Nonce{
		BlockHeight: BlockHeightCurrent,
	}
	workerNonce := &emissionstypes.Nonce{
		BlockHeight: BlockHeightEval,
	}
	valueBundle := generateValueBundle(m, topicId, workers, reputerNonce, workerNonce)
	reputerValueBundles := make([]*emissionstypes.ReputerValueBundle, 0)
	for _, reputer := range reputers {
		reputerValueBundle := generateSingleReputerValueBundle(m, reputer, valueBundle)
		reputerValueBundles = append(reputerValueBundles, reputerValueBundle)
	}
	if len(reputerValueBundles) == 0 {
		return true
	}

	reputerValueBundleMsg := generateReputerValueBundleMsg(
		topicId,
		reputerValueBundles,
		leaderReputer.addr,
		reputerNonce,
		workerNonce,
	)
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, leaderReputer.acc, reputerValueBundleMsg)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	if wasErr {
		return wasErr
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	requireNoError(m.T, data.failOnErr, err)
	wasErr = orErr(wasErr, err)
	return wasErr
}

// Generate the same valueBundle for a reputer
func generateValueBundle(
	m *testcommon.TestConfig,
	topicId uint64,
	workers []Actor,
	reputerNonce,
	workerNonce *emissionstypes.Nonce,
) emissionstypes.ValueBundle {
	return emissionstypes.ValueBundle{
		TopicId:                topicId,
		CombinedValue:          alloraMath.NewDecFromInt64(100),
		InfererValues:          generateWorkerAttributedValueLosses(m, workers, 3000, 3500),
		ForecasterValues:       generateWorkerAttributedValueLosses(m, workers, 50, 50),
		NaiveValue:             alloraMath.NewDecFromInt64(100),
		OneOutInfererValues:    generateWithheldWorkerAttributedValueLosses(workers, 50, 50),
		OneOutForecasterValues: generateWithheldWorkerAttributedValueLosses(workers, 50, 50),
		OneInForecasterValues:  generateWorkerAttributedValueLosses(m, workers, 50, 50),
		ReputerRequestNonce: &emissionstypes.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
	}
}

// Generate a ReputerValueBundle:of
func generateSingleReputerValueBundle(
	m *testcommon.TestConfig,
	reputer Actor,
	valueBundle emissionstypes.ValueBundle,
) *emissionstypes.ReputerValueBundle {
	valueBundle.Reputer = reputer.addr
	// Sign
	src := make([]byte, 0)
	src, err := valueBundle.XXX_Marshal(src, true)
	require.NoError(m.T, err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature, pubKey, err := m.Client.Context().Keyring.Sign(reputer.name, src, signing.SignMode_SIGN_MODE_DIRECT)
	require.NoError(m.T, err, "Sign should not return an error")
	reputerPublicKeyBytes := pubKey.Bytes()

	// Create a MsgInsertBulkReputerPayload message
	reputerValueBundle := &emissionstypes.ReputerValueBundle{
		ValueBundle: &valueBundle,
		Signature:   valueBundleSignature,
		Pubkey:      hex.EncodeToString(reputerPublicKeyBytes),
	}

	return reputerValueBundle
}

// create a MsgInsertBulkReputerPayload message of scores
func generateReputerValueBundleMsg(
	topicId uint64,
	reputerValueBundles []*emissionstypes.ReputerValueBundle,
	leaderReputerAddress string,
	reputerNonce, workerNonce *emissionstypes.Nonce) *emissionstypes.MsgInsertBulkReputerPayload {

	return &emissionstypes.MsgInsertBulkReputerPayload{
		Sender:  leaderReputerAddress,
		TopicId: topicId,
		ReputerRequestNonce: &emissionstypes.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
		ReputerValueBundles: reputerValueBundles,
	}
}

// for every worker, generate a worker attributed value
func generateWorkerAttributedValueLosses(
	m *testcommon.TestConfig,
	workers []Actor,
	lowLimit,
	sum int,
) []*emissionstypes.WorkerAttributedValue {
	values := make([]*emissionstypes.WorkerAttributedValue, 0)
	for _, worker := range workers {
		values = append(values, &emissionstypes.WorkerAttributedValue{
			Worker: worker.addr,
			Value:  alloraMath.NewDecFromInt64(int64(m.Client.Rand.Intn(lowLimit) + sum)),
		})
	}
	return values
}

// for every worker, generate a withheld worker attribute value
func generateWithheldWorkerAttributedValueLosses(
	workers []Actor,
	lowLimit,
	sum int,
) []*emissionstypes.WithheldWorkerAttributedValue {
	values := make([]*emissionstypes.WithheldWorkerAttributedValue, 0)
	for _, worker := range workers {
		values = append(values, &emissionstypes.WithheldWorkerAttributedValue{
			Worker: worker.addr,
			Value:  alloraMath.NewDecFromInt64(int64(rand.Intn(lowLimit) + sum)),
		})
	}
	return values
}
