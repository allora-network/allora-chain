package stress_test

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/stretchr/testify/require"

	"github.com/allora-network/allora-chain/app/params"
	testCommon "github.com/allora-network/allora-chain/test/common"
)

// creates the reputer addresses in the account registry
func createReputerAddresses(
	m testCommon.TestConfig,
	topicId uint64,
	reputersMax int,
) (reputers NameToAccountMap) {
	reputers = make(map[string]AccountAndAddress)

	for reputerIndex := 0; reputerIndex < reputersMax; reputerIndex++ {
		reputerAccountName := getReputerAccountName(reputerIndex, topicId)
		workerAccount, _, err := m.Client.AccountRegistryCreate(reputerAccountName)
		if err != nil {
			fmt.Println("Error creating funder address: ", reputerAccountName, " - ", err)
			continue
		}
		reputerAddressToFund, err := workerAccount.Address(params.HumanCoinUnit)
		if err != nil {
			fmt.Println("Error creating funder address: ", reputerAccountName, " - ", err)
			continue
		}
		reputers[reputerAccountName] = AccountAndAddress{
			acc:  workerAccount,
			addr: reputerAddressToFund,
		}
	}

	return reputers
}

func registerReputersForIteration(
	m testCommon.TestConfig,
	topicId uint64,
	iteration int,
	reputersPerIteration int,
	countReputers int,
	maxReputersPerTopic int,
	reputers NameToAccountMap,
	makeReport bool,
) int {
	for j := 0; j < reputersPerIteration && countReputers < maxReputersPerTopic; j++ {
		reputerName := getReputerAccountName(iteration*j, topicId)
		reputer := reputers[reputerName]
		err := RegisterReputerForTopic(m, reputer.addr, reputer.acc, topicId)
		if err != nil {
			topicLog(topicId, "Error registering reputer address: ", reputer.addr, " - ", err)
			if makeReport {
				saveReputerError(topicId, reputerName, err)
				saveTopicError(topicId, err)
			}
			return countReputers
		}
		countReputers++
	}
	return countReputers
}

// Insert reputer bulk, choosing one random leader from the reputer accounts
func generateInsertReputerBulk(
	m testCommon.TestConfig,
	topic *emissionstypes.Topic,
	blockHeightCurrent int64,
	blockHeightEval int64,
	reputers NameToAccountMap,
	workers NameToAccountMap,
	makeReport bool,
) (int64, int64, error) {
	leaderReputerAccountName, err := pickRandomKeyFromMap(reputers)
	if err != nil {
		topicLog(topic.Id, "Error getting random worker address: ", err)
		if makeReport {
			saveTopicError(topic.Id, err)
		}
		return blockHeightCurrent, blockHeightEval, err
	}

	startReputer := time.Now()
	err = insertReputerBulk(m, topic, leaderReputerAccountName, reputers, workers, blockHeightCurrent, blockHeightEval)
	if err != nil {
		if strings.Contains(err.Error(), "nonce already fulfilled") ||
			strings.Contains(err.Error(), "nonce still unfulfilled") {
			// realign blockHeights before retrying
			topic, err = getLastTopic(m.Ctx, m.Client.QueryEmissions(), topic.Id)
			if err == nil {
				blockHeightCurrent = topic.EpochLastEnded + topic.EpochLength
				blockHeightEval = blockHeightCurrent - topic.EpochLength
				topicLog(topic.Id, "Reset blockHeights to (", blockHeightCurrent, ",", blockHeightEval, ")")
			} else {
				topicLog(topic.Id, "Error getting topic!")
				if makeReport {
					saveTopicError(topic.Id, err)
				}
			}
		}
		return blockHeightCurrent, blockHeightEval, err
	} else {
		topicLog(topic.Id, "Inserted reputer bulk, blockHeight: ", blockHeightCurrent, " with ", len(reputers), " reputers")
		elapsedBulk := time.Since(startReputer)
		topicLog(topic.Id, "Insert Reputer Elapsed time:", elapsedBulk)
	}
	return blockHeightCurrent, blockHeightEval, nil
}

func generateWorkerAttributedValueLosses(
	workerAddresses NameToAccountMap,
	lowLimit,
	sum int,
) []*emissionstypes.WorkerAttributedValue {
	values := make([]*emissionstypes.WorkerAttributedValue, 0)
	for key := range workerAddresses {
		values = append(values, &emissionstypes.WorkerAttributedValue{
			Worker: workerAddresses[key].addr,
			Value:  alloraMath.NewDecFromInt64(int64(rand.Intn(lowLimit) + sum)),
		})
	}
	return values
}

func generateWithheldWorkerAttributedValueLosses(
	workerAddresses NameToAccountMap,
	lowLimit,
	sum int,
) []*emissionstypes.WithheldWorkerAttributedValue {
	values := make([]*emissionstypes.WithheldWorkerAttributedValue, 0)
	for key := range workerAddresses {
		values = append(values, &emissionstypes.WithheldWorkerAttributedValue{
			Worker: workerAddresses[key].addr,
			Value:  alloraMath.NewDecFromInt64(int64(rand.Intn(lowLimit) + sum)),
		})
	}
	return values
}

// Generate the same valueBundle for a reputer
func generateValueBundle(
	topicId uint64,
	workerAddresses NameToAccountMap,
	reputerNonce,
	workerNonce *emissionstypes.Nonce,
) emissionstypes.ValueBundle {
	return emissionstypes.ValueBundle{
		TopicId:                topicId,
		CombinedValue:          alloraMath.NewDecFromInt64(100),
		InfererValues:          generateWorkerAttributedValueLosses(workerAddresses, 3000, 3500),
		ForecasterValues:       generateWorkerAttributedValueLosses(workerAddresses, 50, 50),
		NaiveValue:             alloraMath.NewDecFromInt64(100),
		OneOutInfererValues:    generateWithheldWorkerAttributedValueLosses(workerAddresses, 50, 50),
		OneOutForecasterValues: generateWithheldWorkerAttributedValueLosses(workerAddresses, 50, 50),
		OneInForecasterValues:  generateWorkerAttributedValueLosses(workerAddresses, 50, 50),
		ReputerRequestNonce: &emissionstypes.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
			WorkerNonce:  workerNonce,
		},
	}
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

// reputers submit their assessment of the quality of workers' work compared to ground truth
func insertReputerBulk(
	m testCommon.TestConfig,
	topic *emissionstypes.Topic,
	leaderReputerAccountName string,
	reputerAddresses,
	workerAddresses NameToAccountMap,
	BlockHeightCurrent,
	BlockHeightEval int64,
) error {
	// Nonce: calculate from EpochLastRan + EpochLength
	topicId := topic.Id
	// Nonces are last two blockHeights
	reputerNonce := &emissionstypes.Nonce{
		BlockHeight: BlockHeightCurrent,
	}
	workerNonce := &emissionstypes.Nonce{
		BlockHeight: BlockHeightEval,
	}
	leaderReputer := reputerAddresses[leaderReputerAccountName]
	valueBundle := generateValueBundle(topicId, workerAddresses, reputerNonce, workerNonce)
	reputerValueBundles := make([]*emissionstypes.ReputerValueBundle, 0)
	for reputerAddressName := range reputerAddresses {
		reputer := reputerAddresses[reputerAddressName]
		reputerValueBundle := generateSingleReputerValueBundle(m, reputerAddressName, reputer.addr, valueBundle)
		reputerValueBundles = append(reputerValueBundles, reputerValueBundle)
	}

	reputerValueBundleMsg := generateReputerValueBundleMsg(
		topicId,
		reputerValueBundles,
		leaderReputer.addr,
		reputerNonce,
		workerNonce,
	)
	LeaderAcc, err := m.Client.AccountRegistryGetByName(leaderReputerAccountName)
	if err != nil {
		fmt.Println("Error getting leader worker account: ", leaderReputerAccountName, " - ", err)
		return err
	}
	txResp, err := m.Client.BroadcastTx(m.Ctx, LeaderAcc, reputerValueBundleMsg)
	if err != nil {
		fmt.Println("Error broadcasting reputer value bundle: ", err)
		return err
	}
	_, err = m.Client.WaitForTx(m.Ctx, txResp.TxHash)
	require.NoError(m.T, err)
	return nil
}

func checkReputersReceivedRewards(
	m testCommon.TestConfig,
	topicId uint64,
	reputers NameToAccountMap,
	countReputers int,
	maxIterations int,
	makeReport bool,
) (rewardedReputersCount uint64, err error) {
	rewardedReputersCount = 0
	err = nil
	for reputerIndex := 0; reputerIndex < countReputers; reputerIndex++ {
		reputerName := getReputerAccountName(reputerIndex, topicId)
		reputer := reputers[reputerName]
		reputerStake, err := getReputerStake(
			m.Ctx,
			m.Client.QueryEmissions(),
			topicId,
			reputer.addr,
		)
		if err != nil {
			topicLog(topicId, "Error getting reputer stake for reputer: ", reputerName, err)
			if makeReport {
				saveReputerError(topicId, reputerName, err)
				saveTopicError(topicId, err)
			}
		} else {
			if reputerStake.Lte(alloraMath.NewDecFromInt64(stakeToAdd)) {
				topicLog(topicId, "Reputer ", reputerName, " stake is not greater than initial amount: ", reputerStake)
				if maxIterations > 20 && reputerIndex < 10 {
					topicLog(topicId, "ERROR: Reputer", reputerName, "has insufficient stake:", reputerStake)
				}
				if makeReport {
					saveReputerError(topicId, reputerName, fmt.Errorf("Stake Not Greater"))
					saveTopicError(topicId, fmt.Errorf("Stake Not Greater"))
				}
			} else {
				topicLog(topicId, "Reputer ", reputerIndex, " stake: ", reputerStake)
				rewardedReputersCount += 1
			}
		}
	}
	return rewardedReputersCount, err
}
