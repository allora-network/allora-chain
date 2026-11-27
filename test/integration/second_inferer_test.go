package integration_test

import (
	"context"
	"encoding/hex"
	"time"

	cosmosMath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/stretchr/testify/require"

	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	"github.com/allora-network/allora-chain/x/emissions/types"
)

// secondInfererFunding pays the extra workers' registration, data and tx fees.
const secondInfererFunding = 1_000_000_000_000_000_000

func boundedInt(value int64) alloraMath.BoundedExp40Dec {
	return alloraMath.MustNewBoundedExp40Dec(alloraMath.NewDecFromInt64(value))
}

func fundAccount(m testCommon.TestConfig, recipient string) {
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, m.BobAcc, &banktypes.MsgSend{
		FromAddress: m.BobAddr,
		ToAddress:   recipient,
		Amount:      sdk.NewCoins(sdk.NewCoin(params.BaseCoinUnit, cosmosMath.NewInt(secondInfererFunding))),
	})
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
}

// insertWorkerPayload submits one worker's signed payload. A nil inference value
// submits only the forecast, and forecastInferers names who the forecast is about.
func insertWorkerPayload(m testCommon.TestConfig, account cosmosaccount.Account, address string, topicId uint64, nonce int64, inferenceValue *alloraMath.BoundedExp40Dec, forecastInferers []string) error {
	ctx := context.Background()
	elements := make([]*types.InputForecastElement, len(forecastInferers))
	for i, inferer := range forecastInferers {
		elements[i] = &types.InputForecastElement{Inferer: inferer, Value: boundedInt(105 + int64(i))}
	}
	bundle := &types.InputInferenceForecastBundle{
		Inference: nil,
		Forecast: &types.InputForecast{
			TopicId:          topicId,
			BlockHeight:      nonce,
			Forecaster:       address,
			ForecastElements: elements,
			ExtraData:        nil,
		},
	}
	if inferenceValue != nil {
		values := make([]*types.InputLabeledValue, len(m.InfererValues))
		for i, labeled := range m.InfererValues {
			values[i] = &types.InputLabeledValue{Label: labeled.Label, Value: *inferenceValue}
		}
		bundle.Inference = &types.InputInference{
			TopicId:     topicId,
			BlockHeight: nonce,
			Inferer:     address,
			Value:       *inferenceValue,
			Values:      values,
			ExtraData:   nil,
			Proof:       "",
		}
	}
	src, err := bundle.XXX_Marshal(make([]byte, 0), true)
	if err != nil {
		return err
	}
	signature, pubKey, err := m.Client.Context().Keyring.Sign(account.Name, src, signing.SignMode_SIGN_MODE_DIRECT)
	if err != nil {
		return err
	}
	txResp, err := m.Client.BroadcastTx(ctx, account, &types.InsertWorkerPayloadRequest{
		Sender: address,
		WorkerDataBundle: &types.InputWorkerDataBundle{
			Worker:                             address,
			Nonce:                              &types.Nonce{BlockHeight: nonce},
			TopicId:                            topicId,
			InferenceForecastsBundle:           bundle,
			InferencesForecastsBundleSignature: signature,
			Pubkey:                             hex.EncodeToString(pubKey.Bytes()),
		},
	})
	if err != nil {
		return err
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	return err
}

// forecasterLosses holds the loss values a reputer reports for one forecaster.
type forecasterLosses struct {
	worker string
	// forecast is the loss of the forecaster's own forecast-implied inference.
	forecast int64
	// oneOut is the network loss with this forecaster left out.
	oneOut int64
	// oneIn is the loss of the inferers combined with only this forecaster.
	oneIn int64
	// oneOutInferers is the loss with each inferer left out of this forecaster's
	// forecast-implied inference, in the order of the bundle's inferers.
	oneOutInferers []int64
}

// reputerLosses is the set of values Alice reports for one nonce. Every slice of
// per-inferer values follows the order of the inferers passed to insertReputerBundle.
type reputerLosses struct {
	inferers       []int64
	oneOutInferers []int64
	forecasters    []forecasterLosses
}

// insertReputerBundle submits Alice's losses for the given inferers and forecasters,
// with distinct values so regrets, scores and rewards differ per worker.
func insertReputerBundle(m testCommon.TestConfig, topicId uint64, nonce int64, inferers []string, losses reputerLosses) error {
	ctx := context.Background()
	attributedInferers := func(values []int64) []*types.InputWorkerAttributedValue {
		rows := make([]*types.InputWorkerAttributedValue, len(inferers))
		for i, inferer := range inferers {
			rows[i] = &types.InputWorkerAttributedValue{Worker: inferer, Value: boundedInt(values[i])}
		}
		return rows
	}
	withheldInferers := func(values []int64) []*types.InputWithheldWorkerAttributedValue {
		rows := make([]*types.InputWithheldWorkerAttributedValue, len(inferers))
		for i, inferer := range inferers {
			rows[i] = &types.InputWithheldWorkerAttributedValue{Worker: inferer, Value: boundedInt(values[i])}
		}
		return rows
	}
	forecasterValues := make([]*types.InputWorkerAttributedValue, len(losses.forecasters))
	oneOutForecasterValues := make([]*types.InputWithheldWorkerAttributedValue, len(losses.forecasters))
	oneInForecasterValues := make([]*types.InputWorkerAttributedValue, len(losses.forecasters))
	oneOutInfererForecasterValues := make([]*types.InputOneOutInfererForecasterValues, len(losses.forecasters))
	for i, forecaster := range losses.forecasters {
		forecasterValues[i] = &types.InputWorkerAttributedValue{Worker: forecaster.worker, Value: boundedInt(forecaster.forecast)}
		oneOutForecasterValues[i] = &types.InputWithheldWorkerAttributedValue{Worker: forecaster.worker, Value: boundedInt(forecaster.oneOut)}
		oneInForecasterValues[i] = &types.InputWorkerAttributedValue{Worker: forecaster.worker, Value: boundedInt(forecaster.oneIn)}
		oneOutInfererForecasterValues[i] = &types.InputOneOutInfererForecasterValues{
			Forecaster:          forecaster.worker,
			OneOutInfererValues: withheldInferers(forecaster.oneOutInferers),
		}
	}
	valueBundle := &types.InputValueBundle{
		TopicId:                       topicId,
		Reputer:                       m.AliceAddr,
		ReputerRequestNonce:           &types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: nonce}},
		ExtraData:                     nil,
		CombinedValue:                 boundedInt(100),
		NaiveValue:                    boundedInt(103),
		InfererValues:                 attributedInferers(losses.inferers),
		ForecasterValues:              forecasterValues,
		OneOutInfererValues:           withheldInferers(losses.oneOutInferers),
		OneOutForecasterValues:        oneOutForecasterValues,
		OneInForecasterValues:         oneInForecasterValues,
		OneOutInfererForecasterValues: oneOutInfererForecasterValues,
	}
	src, err := valueBundle.XXX_Marshal(make([]byte, 0), true)
	if err != nil {
		return err
	}
	signature, pubKey, err := m.Client.Context().Keyring.Sign(m.AliceAcc.Name, src, signing.SignMode_SIGN_MODE_DIRECT)
	if err != nil {
		return err
	}
	txResp, err := m.Client.BroadcastTx(ctx, m.AliceAcc, &types.InsertReputerPayloadRequest{
		Sender: m.AliceAddr,
		ReputerValueBundle: &types.InputReputerValueBundle{
			ValueBundle: valueBundle,
			Signature:   signature,
			Pubkey:      hex.EncodeToString(pubKey.Bytes()),
		},
	})
	if err != nil {
		return err
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	return err
}

// SecondInfererChecks runs two epochs with two real inferers, Bob and Validator0,
// who also forecast, so the multi-inferer network inference path and forecaster
// rewards run. In the first epoch Validator1 forecasts only about an address that
// submits nothing, so the close discards that forecast and the reputer scores two
// forecasters. In the second epoch Validator1 forecasts about the two inferers, so
// the close keeps its forecast and the reputer scores three forecasters. It must run
// after a first epoch has produced network losses.
func SecondInfererChecks(m testCommon.TestConfig) {
	ctx := context.Background()
	fundAccount(m, m.Validator0Addr)
	fundAccount(m, m.Validator1Addr)
	addGlobalActor(m, m.Validator0Addr)
	addGlobalActor(m, m.Validator1Addr)
	registerActor(m, m.Validator0Acc, m.TopicID, false)
	registerActor(m, m.Validator1Acc, m.TopicID, false)

	topic, err := waitForNextChurningBlock(m, m.TopicID)
	require.NoError(m.T, err)
	topicResponse, err := m.Client.QueryEmissions().GetTopic(ctx, &types.GetTopicRequest{TopicId: m.TopicID})
	require.NoError(m.T, err)
	nonce := topicResponse.Topic.EpochLastEnded
	m.T.Log(time.Now(), "--- Two inferers and one discarded forecaster for nonce ", nonce, " ---")

	workers := [2]string{m.BobAddr, m.Validator0Addr}
	firstValue, secondValue := boundedInt(100), boundedInt(112)
	require.NoError(m.T, insertWorkerPayload(m, m.BobAcc, m.BobAddr, m.TopicID, nonce, &firstValue, workers[:]))
	require.NoError(m.T, insertWorkerPayload(m, m.Validator0Acc, m.Validator0Addr, m.TopicID, nonce, &secondValue, workers[:]))
	require.NoError(m.T, insertWorkerPayload(m, m.Validator1Acc, m.Validator1Addr, m.TopicID, nonce, nil, []string{m.Validator2Addr}))

	require.NoError(m.T, m.Client.WaitForBlockHeight(ctx, nonce+topic.GroundTruthLag))
	require.NoError(m.T, insertReputerBundle(m, m.TopicID, nonce, workers[:], reputerLosses{
		inferers:       []int64{90, 110},
		oneOutInferers: []int64{101, 99},
		forecasters: []forecasterLosses{
			{worker: workers[0], forecast: 95, oneOut: 102, oneIn: 97, oneOutInferers: []int64{96, 106}},
			{worker: workers[1], forecast: 115, oneOut: 98, oneIn: 104, oneOutInferers: []int64{94, 108}},
		},
	}))
	require.NoError(m.T, m.Client.WaitForBlockHeight(ctx, nonce+topic.GroundTruthLag+topic.EpochLength))
	ValidateGetNetworkLossBundle(m, m.TopicID, nonce)

	// A second epoch with the same two inferers so the topic no longer sits on
	// its all-new-inferer epoch: the combined inference now uses regret-informed
	// weights, and the forecasts of Bob, Validator0 and Validator1 contribute
	// through forecast-implied inferences, which is what the audit's recompute
	// rule and forecaster weight coverage need.
	_, err = waitForNextChurningBlock(m, m.TopicID)
	require.NoError(m.T, err)
	nextTopicResponse, err := m.Client.QueryEmissions().GetTopic(ctx, &types.GetTopicRequest{TopicId: m.TopicID})
	require.NoError(m.T, err)
	nextTopic := nextTopicResponse.Topic
	nextNonce := nextTopic.EpochLastEnded
	m.T.Log(time.Now(), "--- Repeat two-inferer epoch with regret-informed weights for nonce ", nextNonce, " ---")
	require.NoError(m.T, insertWorkerPayload(m, m.BobAcc, m.BobAddr, m.TopicID, nextNonce, &firstValue, workers[:]))
	require.NoError(m.T, insertWorkerPayload(m, m.Validator0Acc, m.Validator0Addr, m.TopicID, nextNonce, &secondValue, workers[:]))
	require.NoError(m.T, insertWorkerPayload(m, m.Validator1Acc, m.Validator1Addr, m.TopicID, nextNonce, nil, workers[:]))
	require.NoError(m.T, m.Client.WaitForBlockHeight(ctx, nextNonce+nextTopic.GroundTruthLag))
	require.NoError(m.T, insertReputerBundle(m, m.TopicID, nextNonce, workers[:], reputerLosses{
		inferers:       []int64{90, 110},
		oneOutInferers: []int64{101, 99},
		forecasters: []forecasterLosses{
			{worker: workers[0], forecast: 95, oneOut: 102, oneIn: 97, oneOutInferers: []int64{96, 106}},
			{worker: workers[1], forecast: 115, oneOut: 98, oneIn: 104, oneOutInferers: []int64{94, 108}},
			{worker: m.Validator1Addr, forecast: 93, oneOut: 100, oneIn: 99, oneOutInferers: []int64{92, 107}},
		},
	}))
	require.NoError(m.T, m.Client.WaitForBlockHeight(ctx, nextNonce+nextTopic.GroundTruthLag+nextTopic.EpochLength))
	ValidateGetNetworkLossBundle(m, m.TopicID, nextNonce)
}
