package integration_test

import (
	"context"
	"fmt"

	cosmosMath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// requireTxCodeZero waits until the tx is in a block and rejects a non-zero code.
func requireTxCodeZero(m testCommon.TestConfig, ctx context.Context, hash string) {
	m.T.Helper()
	res, err := m.Client.WaitForTx(ctx, hash)
	require.NoError(m.T, err)
	require.NotNil(m.T, res)
	require.Zerof(m.T, res.TxResult.Code, "tx %s failed: %s", hash, res.TxResult.Log)
}

// requireState retries a check across a few blocks. Queries rotate across
// validators, so the first read can land on one that has not applied the tx.
func requireState(m testCommon.TestConfig, ctx context.Context, check func() error) {
	m.T.Helper()
	var last error
	for range 8 {
		last = check()
		if last == nil {
			return
		}
		require.NoError(m.T, m.Client.WaitForNextBlock(ctx))
	}
	require.NoError(m.T, last)
}

func addTopicReputer(m testCommon.TestConfig, address string) {
	ctx := context.Background()
	addTopicReputerRequest := &emissionstypes.AddToTopicReputerWhitelistRequest{
		Sender:  m.AliceAddr,
		Address: address,
		TopicId: m.TopicID,
	}
	txResp, err := m.Client.BroadcastTx(ctx, m.AliceAcc, addTopicReputerRequest)
	require.NoError(m.T, err)
	requireTxCodeZero(m, ctx, txResp.TxHash)
	addTopicReputerResponse := &emissionstypes.AddToTopicReputerWhitelistResponse{}
	err = txResp.Decode(addTopicReputerResponse)
	require.NoError(m.T, err)
}

// register alice as a reputer in topic, then check success
func StakeAliceAsReputer(m testCommon.TestConfig) {
	ctx := context.Background()
	// Record Alice stake before adding more
	aliceStakedBefore, err := m.Client.QueryEmissions().GetReputerStakeInTopic(
		ctx,
		&emissionstypes.GetReputerStakeInTopicRequest{
			TopicId: m.TopicID,
			Address: m.AliceAddr,
		},
	)
	require.NoError(m.T, err)

	const stakeToAdd = 1000000

	// Allow Alice to stake as a reputer
	addTopicReputer(m, m.AliceAddr)

	// Have Alice stake more
	addStake := &emissionstypes.AddStakeRequest{
		Sender:  m.AliceAddr,
		TopicId: m.TopicID,
		Amount:  cosmosMath.NewInt(stakeToAdd),
	}
	txResp, err := m.Client.BroadcastTx(ctx, m.AliceAcc, addStake)
	require.NoError(m.T, err)
	requireTxCodeZero(m, ctx, txResp.TxHash)

	requireState(m, ctx, func() error {
		aliceStakedAfter, err := m.Client.QueryEmissions().GetReputerStakeInTopic(
			ctx,
			&emissionstypes.GetReputerStakeInTopicRequest{
				TopicId: m.TopicID,
				Address: m.AliceAddr,
			},
		)
		if err != nil {
			return err
		}
		got := aliceStakedAfter.Amount.Sub(aliceStakedBefore.Amount).String()
		if got != fmt.Sprint(stakeToAdd) {
			return fmt.Errorf("alice stake delta %s, want %d", got, stakeToAdd)
		}
		return nil
	})
}

// integration tests the ability of bob to stake on alice as a reputer
func StakeBobOnAliceAsReputer(m testCommon.TestConfig) {
	ctx := context.Background()
	// Record Bob stake before adding more
	bobStakedBefore, err := m.Client.QueryEmissions().GetStakeFromDelegatorInTopicInReputer(
		ctx,
		&emissionstypes.GetStakeFromDelegatorInTopicInReputerRequest{
			TopicId:          m.TopicID,
			DelegatorAddress: m.BobAddr,
			ReputerAddress:   m.AliceAddr,
		},
	)
	require.NoError(m.T, err)

	const stakeToAdd = 200000

	// Have bob stake
	addDelegateStake := &emissionstypes.DelegateStakeRequest{
		Sender:  m.BobAddr,
		Reputer: m.AliceAddr,
		TopicId: m.TopicID,
		Amount:  cosmosMath.NewInt(stakeToAdd),
	}
	txResp, err := m.Client.BroadcastTx(ctx, m.BobAcc, addDelegateStake)
	require.NoError(m.T, err)
	requireTxCodeZero(m, ctx, txResp.TxHash)

	requireState(m, ctx, func() error {
		bobStakedAfter, err := m.Client.QueryEmissions().GetStakeFromDelegatorInTopicInReputer(
			ctx,
			&emissionstypes.GetStakeFromDelegatorInTopicInReputerRequest{
				TopicId:          m.TopicID,
				DelegatorAddress: m.BobAddr,
				ReputerAddress:   m.AliceAddr,
			},
		)
		if err != nil {
			return err
		}
		got := bobStakedAfter.Amount.Sub(bobStakedBefore.Amount).String()
		if got != fmt.Sprint(stakeToAdd) {
			return fmt.Errorf("bob stake delta %s, want %d", got, stakeToAdd)
		}
		return nil
	})
}

// Register two actors and check their registrations went through
func StakingChecks(m testCommon.TestConfig) {
	ctx := context.Background()
	m.T.Log("--- Staking Alice as Reputer ---")
	StakeAliceAsReputer(m)

	res, _ := m.Client.QueryEmissions().GetTopic(ctx, &emissionstypes.GetTopicRequest{
		TopicId: m.TopicID,
	})
	// Topic is not expected to be funded yet => expect 0 weight => topic not active!
	// But we still have this conditional just in case there are > 0 funds
	if res.EffectiveRevenue != "0" {
		m.T.Logf("--- Check reactivating Topic %d ---", m.TopicID)
		CheckTopicActivated(m)
	}

	m.T.Log("--- Staking Bob on Alice as Reputer ---")
	StakeBobOnAliceAsReputer(m)
}

// Unstake Alice as a reputer in topic, then check success
func UnstakeAliceAsReputer(m testCommon.TestConfig) {
	ctx := context.Background()
	aliceStakeBefore, err := m.Client.QueryEmissions().GetStakeFromReputerInTopicInSelf(
		ctx,
		&emissionstypes.GetStakeFromReputerInTopicInSelfRequest{
			TopicId:        m.TopicID,
			ReputerAddress: m.AliceAddr,
		},
	)
	require.NoError(m.T, err)
	require.True(
		m.T,
		aliceStakeBefore.Amount.GT(cosmosMath.ZeroInt()),
		"Alice should have stake in topic %d",
		m.TopicID,
	)

	// Have Alice unstake
	unstake := &emissionstypes.RemoveStakeRequest{
		Sender:  m.AliceAddr,
		TopicId: m.TopicID,
		Amount:  aliceStakeBefore.Amount,
	}

	txResp, err := m.Client.BroadcastTx(ctx, m.AliceAcc, unstake)
	require.NoError(m.T, err)
	requireTxCodeZero(m, ctx, txResp.TxHash)

	var stakeRemoval *emissionstypes.GetStakeRemovalInfoResponse
	requireState(m, ctx, func() error {
		stakeRemoval, err = m.Client.QueryEmissions().GetStakeRemovalInfo(
			ctx,
			&emissionstypes.GetStakeRemovalInfoRequest{
				TopicId: m.TopicID,
				Reputer: m.AliceAddr,
			},
		)
		if err != nil {
			return err
		}
		if stakeRemoval == nil || stakeRemoval.Removal == nil || stakeRemoval.Removal.BlockRemovalCompleted == 0 {
			return fmt.Errorf("alice stake removal is not queued")
		}
		return nil
	})
	m.T.Log("--- Unstake removal is queued, waiting for block ", stakeRemoval.Removal.BlockRemovalCompleted, " ---")
	err = m.Client.WaitForBlockHeight(ctx, stakeRemoval.Removal.BlockRemovalCompleted+1)
	require.NoError(m.T, err)
	blockHeight, err := m.Client.BlockHeight(ctx)
	require.NoError(m.T, err)
	require.Greater(m.T, blockHeight, stakeRemoval.Removal.BlockRemovalCompleted)

	// Check Alice has zero stake left
	aliceStakedAfter, err := m.Client.QueryEmissions().GetStakeFromReputerInTopicInSelf(
		ctx,
		&emissionstypes.GetStakeFromReputerInTopicInSelfRequest{
			TopicId:        m.TopicID,
			ReputerAddress: m.AliceAddr,
		},
	)
	require.NoError(m.T, err)
	require.Truef(
		m.T,
		aliceStakedAfter.Amount.Equal(cosmosMath.ZeroInt()),
		"Alice should have zero stake in topic %d after unstake: %v, %s, %s",
		m.TopicID,
		stakeRemoval.Removal,
		aliceStakeBefore.Amount.String(),
		aliceStakedAfter.Amount.String(),
	)
}

// Unstake Bob as a delegator delegated to Alice in topic, then check success
func UnstakeBobAsDelegatorOnAlice(m testCommon.TestConfig) {
	ctx := context.Background()
	bobStake, err := m.Client.QueryEmissions().GetStakeFromDelegatorInTopicInReputer(
		ctx,
		&emissionstypes.GetStakeFromDelegatorInTopicInReputerRequest{
			TopicId:          m.TopicID,
			DelegatorAddress: m.BobAddr,
			ReputerAddress:   m.AliceAddr,
		},
	)
	require.NoError(m.T, err)
	require.True(
		m.T,
		bobStake.Amount.GT(cosmosMath.ZeroInt()),
		"Bob should have stake on Alice in topic %d",
		m.TopicID,
	)

	// Have Bob unstake
	unstake := &emissionstypes.RemoveDelegateStakeRequest{
		Sender:  m.BobAddr,
		Reputer: m.AliceAddr,
		TopicId: m.TopicID,
		Amount:  bobStake.Amount,
	}

	txResp, err := m.Client.BroadcastTx(ctx, m.BobAcc, unstake)
	require.NoError(m.T, err)
	requireTxCodeZero(m, ctx, txResp.TxHash)

	var stakeRemoval *emissionstypes.GetDelegateStakeRemovalInfoResponse
	requireState(m, ctx, func() error {
		stakeRemoval, err = m.Client.QueryEmissions().GetDelegateStakeRemovalInfo(
			ctx,
			&emissionstypes.GetDelegateStakeRemovalInfoRequest{
				TopicId:   m.TopicID,
				Delegator: m.BobAddr,
				Reputer:   m.AliceAddr,
			},
		)
		if err != nil {
			return err
		}
		if stakeRemoval == nil || stakeRemoval.Removal == nil || stakeRemoval.Removal.BlockRemovalCompleted == 0 {
			return fmt.Errorf("bob delegate stake removal is not queued")
		}
		return nil
	})
	m.T.Log("--- Unstake removal is queued, waiting for block ", stakeRemoval.Removal.BlockRemovalCompleted, " ---")
	err = m.Client.WaitForBlockHeight(ctx, stakeRemoval.Removal.BlockRemovalCompleted+1)
	require.NoError(m.T, err)

	// Check Bob has zero stake left
	bobStakedAfter, err := m.Client.QueryEmissions().GetStakeFromDelegatorInTopicInReputer(
		ctx,
		&emissionstypes.GetStakeFromDelegatorInTopicInReputerRequest{
			TopicId:          m.TopicID,
			DelegatorAddress: m.BobAddr,
			ReputerAddress:   m.AliceAddr,
		},
	)
	require.NoError(m.T, err)
	require.True(
		m.T,
		bobStakedAfter.Amount.Equal(cosmosMath.ZeroInt()),
		"Bob should have zero stake in topic 1 after unstake",
	)
}

// run checks for unstaking
func UnstakingChecks(m testCommon.TestConfig) {
	m.T.Log("--- Bob Unstaking as Delegator Upon Alice ---")
	UnstakeBobAsDelegatorOnAlice(m)
	m.T.Log("--- Unstaking Alice as Reputer ---")
	UnstakeAliceAsReputer(m)
}
