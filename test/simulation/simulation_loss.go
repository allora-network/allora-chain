package simulation

import (
	"encoding/hex"
	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"
	"math/rand"
	"strings"
)

func getGroundTruth() alloraMath.Dec {
	return alloraMath.NewDecFromInt64(int64(rand.Intn(51) + 50))
}
func lossFunc(yTrue alloraMath.Dec, yPred alloraMath.Dec) alloraMath.Dec {
	value, err := yTrue.Sub(yPred)
	if err != nil {
		return alloraMath.ZeroDec()
	}
	return value.Abs()
}

func qFunc(lossData, preData, alpha alloraMath.Dec) (alloraMath.Dec, error) {
	left, err := alpha.Mul(lossData)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	one := alloraMath.OneDec()
	right, err := one.Sub(alpha)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	right, err = right.Mul(preData)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	newVal, err := left.Add(right)
	if err != nil {
		return alloraMath.ZeroDec(), err
	}
	return newVal, nil
}
func calculateLoss(
	valueBundle *emissionstypes.ValueBundle,
	groundTruth alloraMath.Dec,
) *emissionstypes.ValueBundle {
	combinedValue := lossFunc(groundTruth, valueBundle.CombinedValue)
	naiveValue := lossFunc(groundTruth, valueBundle.NaiveValue)
	infererValues := make([]*emissionstypes.WorkerAttributedValue, 0)
	forecasterValues := make([]*emissionstypes.WorkerAttributedValue, 0)
	oneOutInfererValues := make([]*emissionstypes.WithheldWorkerAttributedValue, 0)
	oneOutForecasterValues := make([]*emissionstypes.WithheldWorkerAttributedValue, 0)
	oneInForecasterValues := make([]*emissionstypes.WorkerAttributedValue, 0)
	for _, infererVal := range valueBundle.InfererValues {
		newVal := lossFunc(groundTruth, infererVal.Value)
		infererValues = append(infererValues, &emissionstypes.WorkerAttributedValue{
			Worker: infererVal.Worker,
			Value:  newVal,
		})
	}
	for _, forecasterVal := range valueBundle.ForecasterValues {
		newVal := lossFunc(groundTruth, forecasterVal.Value)
		forecasterValues = append(forecasterValues, &emissionstypes.WorkerAttributedValue{
			Worker: forecasterVal.Worker,
			Value:  newVal,
		})
	}
	for _, infererVal := range valueBundle.OneOutInfererValues {
		newVal := lossFunc(groundTruth, infererVal.Value)
		oneOutInfererValues = append(oneOutInfererValues, &emissionstypes.WithheldWorkerAttributedValue{
			Worker: infererVal.Worker,
			Value:  newVal,
		})
	}
	for _, forecasterVal := range valueBundle.OneOutForecasterValues {
		newVal := lossFunc(groundTruth, forecasterVal.Value)
		oneOutForecasterValues = append(oneOutForecasterValues, &emissionstypes.WithheldWorkerAttributedValue{
			Worker: forecasterVal.Worker,
			Value:  newVal,
		})
	}
	for _, infererVal := range valueBundle.OneInForecasterValues {
		newVal := lossFunc(groundTruth, infererVal.Value)
		oneInForecasterValues = append(oneInForecasterValues, &emissionstypes.WorkerAttributedValue{
			Worker: infererVal.Worker,
			Value:  newVal,
		})
	}
	return &emissionstypes.ValueBundle{
		TopicId:                valueBundle.TopicId,
		CombinedValue:          combinedValue,
		InfererValues:          infererValues,
		ForecasterValues:       forecasterValues,
		NaiveValue:             naiveValue,
		OneOutInfererValues:    oneOutInfererValues,
		OneOutForecasterValues: oneOutForecasterValues,
		OneInForecasterValues:  oneInForecasterValues,
		ReputerRequestNonce:    valueBundle.ReputerRequestNonce,
	}
}

func calculateEmaLossArray[T emissionstypes.WorkerAttributedValue | emissionstypes.WithheldWorkerAttributedValue](
	infererValues []*emissionstypes.WorkerAttributedValue,
	previousValues []*emissionstypes.WorkerAttributedValue,
	alpha alloraMath.Dec,
) []*T {
	emaLossValues := make([]*T, 0)
	for index := 0; index < len(infererValues); index++ {
		infererInference := infererValues[index]
		worker := infererInference.Worker
		previousValueFound := false
		previousValue := alloraMath.ZeroDec()
		for pIndex := 0; pIndex < len(previousValues); pIndex++ {
			if previousValues[pIndex].Worker == worker {
				previousValueFound = true
				previousValue = previousValues[pIndex].Value
				break
			}
		}
		if previousValueFound {
			newVal, err := qFunc(infererInference.Value, previousValue, alpha)
			if err != nil {
				continue
			}
			emaLossValues = append(emaLossValues, &T{
				Worker: worker,
				Value:  newVal,
			})
		} else {
			emaLossValues = append(emaLossValues, &T{
				Worker: worker,
				Value:  infererInference.Value,
			})
		}
	}
	return emaLossValues
}

func calculateEmaLoss(
	lossData *emissionstypes.ValueBundle,
	previousLosses *emissionstypes.ValueBundle,
	alpha alloraMath.Dec,
) emissionstypes.ValueBundle {
	combinedValue := alloraMath.ZeroDec()
	naiveValue := alloraMath.ZeroDec()

	convertWorkerAttributedValueType := func(values []*emissionstypes.WithheldWorkerAttributedValue) []*emissionstypes.WorkerAttributedValue {
		res := make([]*emissionstypes.WorkerAttributedValue, 0)
		for _, value := range values {
			res = append(res, &emissionstypes.WorkerAttributedValue{
				Worker: value.Worker,
				Value:  value.Value,
			})
		}
		return res
	}

	if previousLosses.CombinedValue.Gt(alloraMath.ZeroDec()) {
		combinedValue, _ = qFunc(lossData.CombinedValue, previousLosses.CombinedValue, alpha)
	} else {
		combinedValue = lossData.CombinedValue
	}
	if previousLosses.NaiveValue.Gt(alloraMath.ZeroDec()) {
		naiveValue, _ = qFunc(lossData.NaiveValue, previousLosses.NaiveValue, alpha)
	} else {
		naiveValue = lossData.NaiveValue
	}
	infererValues := calculateEmaLossArray[emissionstypes.WorkerAttributedValue](
		lossData.InfererValues, previousLosses.InfererValues, alpha)
	forecasterValues := calculateEmaLossArray[emissionstypes.WorkerAttributedValue](
		lossData.ForecasterValues, previousLosses.ForecasterValues, alpha)
	oneOutInfererValues := calculateEmaLossArray[emissionstypes.WithheldWorkerAttributedValue](
		convertWorkerAttributedValueType(lossData.OneOutInfererValues),
		convertWorkerAttributedValueType(previousLosses.OneOutInfererValues), alpha)
	oneOutForecasterValues := calculateEmaLossArray[emissionstypes.WithheldWorkerAttributedValue](
		convertWorkerAttributedValueType(lossData.OneOutForecasterValues),
		convertWorkerAttributedValueType(previousLosses.OneOutForecasterValues), alpha)
	oneInForecasterValues := calculateEmaLossArray[emissionstypes.WorkerAttributedValue](
		lossData.OneInForecasterValues, previousLosses.OneInForecasterValues, alpha)
	return emissionstypes.ValueBundle{
		TopicId:                lossData.TopicId,
		CombinedValue:          combinedValue,
		NaiveValue:             naiveValue,
		InfererValues:          infererValues,
		ForecasterValues:       forecasterValues,
		OneOutInfererValues:    oneOutInfererValues,
		OneOutForecasterValues: oneOutForecasterValues,
		OneInForecasterValues:  oneInForecasterValues,
		ReputerRequestNonce:    lossData.ReputerRequestNonce,
	}
}

func generateValueBundle(
	m testCommon.TestConfig,
	topicId uint64,
	currentLossNonce,
	prevLossNonce *emissionstypes.Nonce,
) (emissionstypes.ValueBundle, error) {
	ALPHA := alloraMath.MustNewDecFromString("0.1")
	groundTruth := getGroundTruth()
	valueBundle, err := getNetworkInferencesAtBlock(m, topicId, currentLossNonce.BlockHeight, prevLossNonce.BlockHeight)
	if err != nil {
		return emissionstypes.ValueBundle{}, err
	}
	prevLoss := getNetworkLossBundleAtBlock(m, topicId, prevLossNonce.BlockHeight)
	lossData := calculateLoss(valueBundle, groundTruth)
	newLoss := calculateEmaLoss(lossData, prevLoss, ALPHA)
	return newLoss, nil
}

// Generate a ReputerValueBundle:of
func generateSingleReputerValueBundle(
	m testCommon.TestConfig,
	reputerAddressName,
	reputerAddress string,
	valueBundle emissionstypes.ValueBundle,
) *emissionstypes.ReputerValueBundle {
	valueBundle.Reputer = reputerAddress
	// Sign
	src := make([]byte, 0)
	src, err := valueBundle.XXX_Marshal(src, true)
	require.NoError(m.T, err, "Marshall reputer value bundle should not return an error")

	valueBundleSignature, pubKey, err := m.Client.Context().Keyring.Sign(reputerAddressName, src, signing.SignMode_SIGN_MODE_DIRECT)
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
func insertReputerBulk(
	m testCommon.TestConfig,
	seed int,
	topic *emissionstypes.Topic,
	insertedBlockHeight int64,
	prevLossHeight int64,
	reputers []testCommon.AccountAndAddress,
) (int64, error) {
	leaderIndex := rand.Intn(len(reputers))
	leaderReputer := reputers[leaderIndex]
	var blockHeightCurrent int64 = 0
	currentLossEpoch := &emissionstypes.Nonce{
		BlockHeight: insertedBlockHeight,
	}
	prevLossEpoch := &emissionstypes.Nonce{
		BlockHeight: prevLossHeight,
	}
	blockHeightCurrent = insertedBlockHeight
	for index := 0; index < RetryTime; index++ {
		// Nonces are last two blockHeights
		reputerNonce := &emissionstypes.Nonce{
			BlockHeight: blockHeightCurrent,
		}
		workerNonce := &emissionstypes.Nonce{
			BlockHeight: blockHeightCurrent,
		}
		reputerValueBundles := make([]*emissionstypes.ReputerValueBundle, 0)
		for index, reputer := range reputers {
			reputerAccountName := getActorsAccountName(REPUTER_TYPE, seed, index)
			valueBundle, err := generateValueBundle(m, topic.Id, currentLossEpoch, prevLossEpoch)
			if err != nil {
				continue
			}
			valueBundle.ReputerRequestNonce = &emissionstypes.ReputerRequestNonce{
				ReputerNonce: reputerNonce,
				WorkerNonce:  workerNonce,
			}
			reputerValueBundle := generateSingleReputerValueBundle(m, reputerAccountName, reputer.Addr, valueBundle)
			reputerValueBundles = append(reputerValueBundles, reputerValueBundle)
		}

		reputerValueBundleMsg := &emissionstypes.MsgInsertBulkReputerPayload{
			Sender:  leaderReputer.Addr,
			TopicId: topic.Id,
			ReputerRequestNonce: &emissionstypes.ReputerRequestNonce{
				ReputerNonce: reputerNonce,
				WorkerNonce:  workerNonce,
			},
			ReputerValueBundles: reputerValueBundles,
		}
		txResp, err := m.Client.BroadcastTx(m.Ctx, leaderReputer.Acc, reputerValueBundleMsg)
		if err != nil {
			m.T.Log("Error broadcasting reputer value bundle: ", err)
			if strings.Contains(err.Error(), "nonce already fulfilled") ||
				strings.Contains(err.Error(), "nonce still unfulfilled") {
				topic, err = getTopic(m, topic.Id)
				if err == nil {
					insertedBlockHeight = topic.EpochLastEnded
					continue
				}
			}
			return 0, err
		}
		_, err = m.Client.WaitForTx(m.Ctx, txResp.TxHash)
		m.T.Log("Inserted Reputer Bulk", blockHeightCurrent)
		break
	}
	return blockHeightCurrent, nil
}
