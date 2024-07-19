package stress_test

import (
	"context"
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
		reputerAccountName := getReputerAccountName(m.Seed, reputerIndex, topicId)
		reputerAccount, _, err := m.Client.AccountRegistryCreate(reputerAccountName)
		if err != nil {
			m.T.Log(topicLog(topicId, "Error creating funder address: ", reputerAccountName, " - ", err))
			continue
		}
		reputerAddressToFund, err := reputerAccount.Address(params.HumanCoinUnit)
		if err != nil {
			m.T.Log(topicLog(topicId, "Error creating funder address: ", reputerAccountName, " - ", err))
			continue
		}
		reputers[reputerAccountName] = AccountAndAddress{
			acc:  reputerAccount,
			addr: reputerAddressToFund,
		}
	}

	return reputers
}

// register all the created reputers for this iteration
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
		reputerName := getReputerAccountName(m.Seed, iteration*j, topicId)
		reputer := reputers[reputerName]
		err := RegisterReputerForTopic(
			m,
			NameAccountAndAddress{
				name: reputerName,
				aa:   reputer,
			},
			topicId,
		)
		if err != nil {
			m.T.Log(topicLog(topicId, "Error registering reputer address: ", reputer.addr, " - ", err))
			if makeReport {
				saveReputerError(topicId, reputerName, err)
				saveTopicError(topicId, err)
			}
			return countReputers
		}
		err = stakeReputer(
			m,
			topicId,
			NameAccountAndAddress{
				name: reputerName,
				aa:   reputer,
			},
			stakeToAdd,
		)
		if err != nil {
			m.T.Log(topicLog(topicId, "Error staking reputer address: ", reputer.addr, " - ", err))
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
	reputers NameToAccountMap,
	workers NameToAccountMap,
	insertedBlockHeight int64,
	retryTimes int,
	makeReport bool,
) error {
	leaderReputerAccountName, err := pickRandomKeyFromMap(reputers)
	if err != nil {
		m.T.Log(topicLog(topic.Id, "Error getting random worker address: ", err))
		if makeReport {
			saveReputerError(topic.Id, leaderReputerAccountName, err)
			saveTopicError(topic.Id, err)
		}
		return err
	}
	// ground truth lag is 10 blocks
	blockHeightCurrent := insertedBlockHeight + topic.EpochLength
	blockHeightEval := insertedBlockHeight

	startReputer := time.Now()
	for i := 0; i < retryTimes; i++ {
		err = insertReputerBulk(m, topic, leaderReputerAccountName, reputers, workers, blockHeightCurrent, blockHeightEval)
		if err != nil {
			if strings.Contains(err.Error(), "nonce already fulfilled") ||
				strings.Contains(err.Error(), "nonce still unfulfilled") {
				ctx := context.Background()
				// realign blockHeights before retrying
				topic, err = getLastTopic(ctx, m.Client.QueryEmissions(), topic.Id)
				if err == nil {
					blockHeightCurrent = topic.EpochLastEnded + topic.EpochLength
					blockHeightEval = blockHeightCurrent - topic.EpochLength
					m.T.Log(topicLog(topic.Id, "Reset ", leaderReputerAccountName, "blockHeights to (", blockHeightCurrent, ",", blockHeightEval, ")"))
				} else {
					m.T.Log(topicLog(topic.Id, "Error getting topic!"))
					if makeReport {
						saveReputerError(topic.Id, leaderReputerAccountName, err)
						saveTopicError(topic.Id, err)
					}
					return err
				}
			} else {
				m.T.Log(topicLog(topic.Id, "Error inserting reputer bulk: ", err))
				if makeReport {
					saveReputerError(topic.Id, leaderReputerAccountName, err)
					saveTopicError(topic.Id, err)
				}
				return err
			}
		} else {
			m.T.Log(topicLog(topic.Id, "Inserted reputer bulk, blockHeight: ", blockHeightCurrent, " with ", len(reputers), " reputers"))
			elapsedBulk := time.Since(startReputer)
			m.T.Log(topicLog(topic.Id, "Insert Reputer ", leaderReputerAccountName, " Elapsed time:", elapsedBulk))
			return nil
		}
	}
	return err
}

// for every worker, generate a worker attributed value
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

// for every worker, generate a withheld worker attribute value
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
	reputerNonce *emissionstypes.Nonce,
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
	reputerNonce *emissionstypes.Nonce) *emissionstypes.MsgInsertBulkReputerPayload {

	return &emissionstypes.MsgInsertBulkReputerPayload{
		Sender:  leaderReputerAddress,
		TopicId: topicId,
		ReputerRequestNonce: &emissionstypes.ReputerRequestNonce{
			ReputerNonce: reputerNonce,
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
	leaderReputer := reputerAddresses[leaderReputerAccountName]
	valueBundle := generateValueBundle(topicId, workerAddresses, reputerNonce)
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
	)
	LeaderAcc, err := m.Client.AccountRegistryGetByName(leaderReputerAccountName)
	if err != nil {
		m.T.Log(topicLog(topicId, "Error getting leader worker account: ", leaderReputerAccountName, " - ", err))
		return err
	}
	ctx := context.Background()
	txResp, err := m.Client.BroadcastTx(ctx, LeaderAcc, reputerValueBundleMsg)
	if err != nil {
		m.T.Log(topicLog(topicId, "Error broadcasting reputer value bundle: ", err))
		return err
	}
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
	return nil
}

// check that reputers stake values went up after receiving rewards
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
		reputerName := getReputerAccountName(m.Seed, reputerIndex, topicId)
		reputer := reputers[reputerName]
		ctx := context.Background()
		reputerStake, err := getReputerStake(
			ctx,
			m.Client.QueryEmissions(),
			topicId,
			reputer.addr,
		)
		if err != nil {
			m.T.Log(topicLog(topicId, "Error getting reputer stake for reputer: ", reputerName, err))
			if makeReport {
				saveReputerError(topicId, reputerName, err)
				saveTopicError(topicId, err)
			}
		} else {
			if reputerStake.Lte(alloraMath.NewDecFromInt64(int64(stakeToAdd))) {
				m.T.Log(topicLog(topicId, "Reputer ", reputerName, " stake is not greater than initial amount: ", reputerStake))
				if maxIterations > 20 && reputerIndex < 10 {
					m.T.Log(topicLog(topicId, "ERROR: Reputer", reputerName, "has insufficient stake:", reputerStake))
				}
				if makeReport {
					saveReputerError(topicId, reputerName, fmt.Errorf("Stake Not Greater"))
					saveTopicError(topicId, fmt.Errorf("Stake Not Greater"))
				}
			} else {
				m.T.Log(topicLog(topicId, "Reputer ", reputerIndex, " stake: ", reputerStake))
				rewardedReputersCount += 1
			}
		}
	}
	return rewardedReputersCount, err
}
