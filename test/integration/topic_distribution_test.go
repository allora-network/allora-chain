package integration_test

import (
	"context"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app/params"
	alloraMath "github.com/allora-network/allora-chain/math"
	testCommon "github.com/allora-network/allora-chain/test/common"
	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/stretchr/testify/require"
)

// TopicWeightDistributionChecks tests that topic weights are distributed correctly
// even when topics have different epoch lengths.
func TopicWeightDistributionChecks(m testCommon.TestConfig) {
	ctx := context.Background()

	m.T.Log("--- Creating Topics for Weight Distribution Test ---")
	// Allow Alice to create topics
	addTopicCreator(m, m.AliceAddr)

	// Topic 1: Short Epoch Length
	topic1Id := createTestTopic(m, m.AliceAddr, "Topic 1 - Short Epoch", 10) // Example Epoch Length: 10
	m.T.Logf("Created Topic 1 with ID: %d and Epoch Length: 10", topic1Id)

	// Topic 2: Long Epoch Length (2× short epoch so both churn together when the long topic completes its first epoch)
	topic2Id := createTestTopic(m, m.AliceAddr, "Topic 2 - Long Epoch", 20) // Example Epoch Length: 20
	m.T.Logf("Created Topic 2 with ID: %d and Epoch Length: 20", topic2Id)

	m.T.Log("--- Funding Topics for Weight Distribution Test ---")
	// Fund Topic 1
	fundTestTopic(m, m.BobAcc, topic1Id, 10000)
	// Fund Topic 2
	fundTestTopic(m, m.BobAcc, topic2Id, 10000)

	m.T.Log("--- Registering Participants for Weight Distribution Test ---")
	// Register Bob as Worker in Topic 1
	registerActor(m, m.BobAcc, topic1Id, false)
	m.T.Logf("Registered Bob as Worker in Topic %d", topic1Id)
	// Register Bob as Worker in Topic 2
	registerActor(m, m.BobAcc, topic2Id, false)
	m.T.Logf("Registered Bob as Worker in Topic %d", topic2Id)

	// Register Alice as Reputer in Topic 1
	registerActor(m, m.AliceAcc, topic1Id, true)
	m.T.Logf("Registered Alice as Reputer in Topic %d", topic1Id)
	// Register Alice as Reputer in Topic 2
	registerActor(m, m.AliceAcc, topic2Id, true)
	m.T.Logf("Registered Alice as Reputer in Topic %d", topic2Id)

	m.T.Log("--- Staking Participants for Weight Distribution Test ---")
	const stakeAmount = 1000000
	const delegateStakeAmount = 200000
	// Alice stakes in Topic 1
	addStake(m, m.AliceAcc, topic1Id, stakeAmount)
	m.T.Logf("Alice staked %d in Topic %d", stakeAmount, topic1Id)
	// Alice stakes in Topic 2
	addStake(m, m.AliceAcc, topic2Id, stakeAmount)
	m.T.Logf("Alice staked %d in Topic %d", stakeAmount, topic2Id)

	// Bob delegates stake to Alice in Topic 1
	addDelegateStake(m, m.BobAcc, m.AliceAddr, topic1Id, delegateStakeAmount)
	m.T.Logf("Bob delegated %d to Alice in Topic %d", delegateStakeAmount, topic1Id)
	// Bob delegates stake to Alice in Topic 2
	addDelegateStake(m, m.BobAcc, m.AliceAddr, topic2Id, delegateStakeAmount)
	m.T.Logf("Bob delegated %d to Alice in Topic %d", delegateStakeAmount, topic2Id)

	m.T.Log("--- Waiting for Epoch End and Checking Weights for Weight Distribution Test ---")
	// Wait for the next epoch end for Topic 1
	m.T.Logf("Waiting for next churning block for Topic %d", topic1Id)
	updatedTopic1, err := waitForNextChurningBlock(m, topic1Id)
	require.NoError(m.T, err)
	m.T.Logf("Epoch ended for Topic %d at block %d", topic1Id, updatedTopic1.EpochLastEnded)

	// Query weights for Topic 1 (using previous weight after epoch end)
	m.T.Logf("Querying previous weights for Topic %d", topic1Id)
	topic1Weights, err := m.Client.QueryEmissions().GetPreviousTopicWeight(ctx, &types.GetPreviousTopicWeightRequest{TopicId: topic1Id})
	require.NoError(m.T, err)
	require.NotNil(m.T, topic1Weights, "Weights for topic 1 should not be nil")
	m.T.Logf("Topic %d Previous Weight: %s", topic1Id, topic1Weights.Weight.String())

	m.T.Logf("Waiting for next churning block for Topic %d", topic2Id)
	updatedTopic2, err := waitForNextChurningBlock(m, topic2Id)
	require.NoError(m.T, err)
	m.T.Logf("Epoch ended for Topic %d at block %d", topic2Id, updatedTopic2.EpochLastEnded)

	// Query weights for Topic 2 (using previous weight after epoch end)
	m.T.Logf("Querying previous weights for Topic %d", topic2Id)
	topic2Weights, err := m.Client.QueryEmissions().GetPreviousTopicWeight(ctx, &types.GetPreviousTopicWeightRequest{TopicId: topic2Id})
	require.NoError(m.T, err)
	require.NotNil(m.T, topic2Weights, "Weights for topic 2 should not be nil")
	m.T.Logf("Topic %d Previous Weight: %s", topic2Id, topic2Weights.Weight.String())

	// Assert weights are equal
	m.T.Logf("Comparing previous weights for Topic %d and Topic %d", topic1Id, topic2Id)
	require.True(
		m.T,
		topic1Weights.Weight.Equal(topic2Weights.Weight),
		"Topic previous weights should be equal after one epoch of the shorter topic. Topic 1: %s, Topic 2: %s",
		topic1Weights.Weight.String(),
		topic2Weights.Weight.String(),
	)
	m.T.Log("Topic previous weights are equal as expected.")
}

// Helper function to register an actor (worker or reputer) in a topic
func registerActor(m testCommon.TestConfig, account cosmosaccount.Account, topicId uint64, isReputer bool) {
	ctx := context.Background()
	addr, err := account.Address(params.HumanCoinUnit)
	require.NoError(m.T, err)

	registerRequest := &types.RegisterRequest{
		Sender:    addr,
		Owner:     addr,
		TopicId:   topicId,
		IsReputer: isReputer,
	}
	txResp, err := m.Client.BroadcastTx(ctx, account, registerRequest)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
	registerResponse := &types.RegisterResponse{} //nolint:exhaustruct // the fields are populated by decode
	err = txResp.Decode(registerResponse)
	require.NoError(m.T, err)
}

// Helper function to add stake for a reputer in a topic
func addStake(m testCommon.TestConfig, account cosmosaccount.Account, topicId uint64, amount int64) {
	ctx := context.Background()
	addr, err := account.Address(params.HumanCoinUnit)
	require.NoError(m.T, err)

	addStakeRequest := &types.AddStakeRequest{
		Sender:  addr,
		TopicId: topicId,
		Amount:  cosmosMath.NewInt(amount),
	}
	txResp, err := m.Client.BroadcastTx(ctx, account, addStakeRequest)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
}

// Helper function to add delegate stake for a delegator on a reputer in a topic
func addDelegateStake(m testCommon.TestConfig, delegatorAccount cosmosaccount.Account, reputerAddr string, topicId uint64, amount int64) {
	ctx := context.Background()
	delegatorAddr, err := delegatorAccount.Address(params.HumanCoinUnit)
	require.NoError(m.T, err)

	addDelegateStakeRequest := &types.DelegateStakeRequest{
		Sender:  delegatorAddr,
		Reputer: reputerAddr,
		TopicId: topicId,
		Amount:  cosmosMath.NewInt(amount),
	}
	txResp, err := m.Client.BroadcastTx(ctx, delegatorAccount, addDelegateStakeRequest)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
}

// Helper function to create a topic with a specific epoch length
func createTestTopic(m testCommon.TestConfig, creator string, metadata string, epochLength int64) uint64 {
	ctx := context.Background()
	createTopicRequest := &types.CreateNewTopicRequest{
		Creator:                  creator,
		Metadata:                 metadata,
		LossMethod:               "mse",
		EpochLength:              epochLength, // Use the provided epoch length
		GroundTruthLag:           epochLength,
		WorkerSubmissionWindow:   4,
		PNorm:                    alloraMath.NewDecFromInt64(3),
		AlphaRegret:              alloraMath.MustNewDecFromString("0.1"),
		AllowNegative:            true,
		Epsilon:                  alloraMath.MustNewDecFromString("0.01"),
		MeritSortitionAlpha:      alloraMath.MustNewDecFromString("0.1"),
		ActiveInfererQuantile:    alloraMath.MustNewDecFromString("0.2"),
		ActiveForecasterQuantile: alloraMath.MustNewDecFromString("0.2"),
		ActiveReputerQuantile:    alloraMath.MustNewDecFromString("0.2"),
		EnableWorkerWhitelist:    false, // Simplification for testing
		EnableReputerWhitelist:   false, // Simplification for testing
		CNorm:                    alloraMath.MustNewDecFromString("0.75"),
	}
	txResp, err := m.Client.BroadcastTx(ctx, m.AliceAcc, createTopicRequest)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
	createTopicResponse := &types.CreateNewTopicResponse{} //nolint:exhaustruct // the fields are populated by decode
	err = txResp.Decode(createTopicResponse)
	require.NoError(m.T, err)
	return createTopicResponse.TopicId
}

// Helper function to fund a topic
func fundTestTopic(m testCommon.TestConfig, funderAcc cosmosaccount.Account, topicId uint64, amount int64) {
	ctx := context.Background()
	funderAddr, err := funderAcc.Address(params.HumanCoinUnit)
	require.NoError(m.T, err)

	txResp, err := m.Client.BroadcastTx(
		ctx,
		funderAcc,
		&types.FundTopicRequest{
			Sender:  funderAddr,
			TopicId: topicId,
			Amount:  cosmosMath.NewInt(amount),
		},
	)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
	resp := &types.FundTopicResponse{}
	err = txResp.Decode(resp)
	require.NoError(m.T, err)
}
