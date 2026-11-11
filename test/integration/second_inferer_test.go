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

// insertTwoWorkerReputerBundle submits Alice's losses for two inferers who also
// forecast, with distinct values so regrets, scores and rewards differ per worker.
func insertTwoWorkerReputerBundle(m testCommon.TestConfig, topicId uint64, nonce int64, workers [2]string) error {
	ctx := context.Background()
	attributed := func(first, second int64) []*types.InputWorkerAttributedValue {
		return []*types.InputWorkerAttributedValue{
			{Worker: workers[0], Value: boundedInt(first)},
			{Worker: workers[1], Value: boundedInt(second)},
		}
	}
	withheld := func(first, second int64) []*types.InputWithheldWorkerAttributedValue {
		return []*types.InputWithheldWorkerAttributedValue{
			{Worker: workers[0], Value: boundedInt(first)},
			{Worker: workers[1], Value: boundedInt(second)},
		}
	}
	valueBundle := &types.InputValueBundle{
		TopicId:                topicId,
		Reputer:                m.AliceAddr,
		ReputerRequestNonce:    &types.ReputerRequestNonce{ReputerNonce: &types.Nonce{BlockHeight: nonce}},
		ExtraData:              nil,
		CombinedValue:          boundedInt(100),
		NaiveValue:             boundedInt(103),
		InfererValues:          attributed(90, 110),
		ForecasterValues:       attributed(95, 115),
		OneOutInfererValues:    withheld(101, 99),
		OneOutForecasterValues: withheld(102, 98),
		OneInForecasterValues:  attributed(97, 104),
		OneOutInfererForecasterValues: []*types.InputOneOutInfererForecasterValues{
			{Forecaster: workers[0], OneOutInfererValues: withheld(96, 106)},
			{Forecaster: workers[1], OneOutInfererValues: withheld(94, 108)},
		},
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

// SecondInfererChecks runs one epoch with two real inferers who also forecast, so
// the multi-inferer network inference path and forecaster rewards run. A third
// worker forecasts only about an address that submits nothing, so the close discards
// that forecast. It must run after a first epoch has produced network losses.
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
	require.NoError(m.T, insertTwoWorkerReputerBundle(m, m.TopicID, nonce, workers))
	require.NoError(m.T, m.Client.WaitForBlockHeight(ctx, nonce+topic.GroundTruthLag+topic.EpochLength))
	ValidateGetNetworkLossBundle(m, m.TopicID, nonce)

	// A second epoch with the same two inferers so the topic no longer sits on
	// its all-new-inferer epoch: the combined inference now uses regret-informed
	// weights and the forecaster's submission contributes through the
	// forecast-implied inference, which is what the audit's recompute rule and
	// forecaster weight coverage need.
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
	require.NoError(m.T, insertTwoWorkerReputerBundle(m, m.TopicID, nextNonce, workers))
	require.NoError(m.T, m.Client.WaitForBlockHeight(ctx, nextNonce+nextTopic.GroundTruthLag+nextTopic.EpochLength))
	ValidateGetNetworkLossBundle(m, m.TopicID, nextNonce)
}
