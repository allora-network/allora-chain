package integration_test

import (
	"context"
	"fmt"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	testCommon "github.com/allora-network/allora-chain/test/common"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/stretchr/testify/require"
)

// get the amount of coins required to deposit for a proposal
func getDepositRequired(m testCommon.TestConfig) sdktypes.Coin {
	ctx := context.Background()
	queryGovParamsResponse, err := m.Client.QueryGov().Params(ctx, &govtypesv1.QueryParamsRequest{})
	require.NoError(m.T, err)
	return queryGovParamsResponse.Params.ExpeditedMinDeposit[0]
}

// have all three validators vote on a proposal
func voteOnProposal(m testCommon.TestConfig, proposalId uint64) {
	ctx := context.Background()
	validators := []struct {
		acc  cosmosaccount.Account
		addr string
	}{
		{m.Validator0Acc, m.Validator0Addr},
		{m.Validator1Acc, m.Validator1Addr},
		{m.Validator2Acc, m.Validator2Addr},
	}
	for _, validator := range validators {
		msgVote := &govtypesv1.MsgVote{
			ProposalId: proposalId,
			Voter:      validator.addr,
			Option:     govtypesv1.OptionYes,
		}
		txResp, err := m.Client.BroadcastTx(ctx, validator.acc, msgVote)
		require.NoError(m.T, err)
		_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
		require.NoError(m.T, err)
		msgVoteResponse := &govtypesv1.MsgVoteResponse{}
		err = txResp.Decode(msgVoteResponse)
		require.NoError(m.T, err)
		require.NotNil(m.T, msgVoteResponse)
	}
}

// propose an upgrade to the vintegration software version
func proposeUpgrade(m testCommon.TestConfig) (proposalId uint64, proposalHeight int64) {
	ctx := context.Background()
	name := "vintegration"
	summary := "Upgrade to vintegration software version"

	currHeight, err := m.Client.BlockHeight(ctx)
	require.NoError(m.T, err)
	proposalHeight = currHeight + 50 // 4 1/6 minutes
	m.T.Logf("Current Height: %d, proposing upgrade for %d", currHeight, proposalHeight)
	msgSoftwareUpgrade := &upgradetypes.MsgSoftwareUpgrade{
		Authority: authtypes.NewModuleAddress("gov").String(),
		Plan: upgradetypes.Plan{
			Name:   name,
			Height: proposalHeight,
			Info:   "{}",
		},
	}
	msgSubmitProposal := &govtypesv1.MsgSubmitProposal{
		Title:    name,
		Summary:  summary,
		Proposer: m.AliceAddr,
		Metadata: fmt.Sprintf(
			"{title:\"%s\",summary:\"%s\"}", name, summary,
		), // metadata must match title and summary exactly
		Expedited: true,
		InitialDeposit: sdktypes.NewCoins(
			getDepositRequired(m),
		),
	}
	msgSubmitProposal.SetMsgs([]sdktypes.Msg{msgSoftwareUpgrade})
	txResp, err := m.Client.BroadcastTx(ctx, m.AliceAcc, msgSubmitProposal)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
	submitProposalMsgResponse := &govtypesv1.MsgSubmitProposalResponse{}
	err = txResp.Decode(submitProposalMsgResponse)
	require.NoError(m.T, err)
	require.NotNil(m.T, submitProposalMsgResponse.ProposalId)
	return submitProposalMsgResponse.ProposalId, proposalHeight
}

func waitForProposalPass(m testCommon.TestConfig, proposalId uint64) {
	ctx := context.Background()
	proposalRequest := &govtypesv1.QueryProposalRequest{
		ProposalId: proposalId,
	}
	proposal, err := m.Client.QueryGov().Proposal(ctx, proposalRequest)
	require.NoError(m.T, err)
	require.NotNil(m.T, proposal)
	require.NotNil(m.T, proposal.Proposal)
	endTime := proposal.Proposal.VotingEndTime
	require.NotNil(m.T, endTime)
	diff := time.Until(*endTime) + 10*time.Second
	m.T.Logf("Voting End Time: %s, sleeping %s seconds", endTime, diff)
	time.Sleep(diff)

	proposal, err = m.Client.QueryGov().Proposal(ctx, proposalRequest)
	require.NoError(m.T, err)
	require.NotNil(m.T, proposal)
	require.NotNil(m.T, proposal.Proposal)
	require.Equal(m.T, govtypesv1.StatusPassed, proposal.Proposal.Status)
}

// query the current version of the emissions module
func getEmissionsVersion(m testCommon.TestConfig) uint64 {
	ctx := context.Background()
	queryModuleVersionsRequest := &upgradetypes.QueryModuleVersionsRequest{
		ModuleName: "emissions",
	}
	moduleVersions, err := m.Client.QueryUpgrade().ModuleVersions(ctx, queryModuleVersionsRequest)
	require.NoError(m.T, err)
	require.NotNil(m.T, moduleVersions)
	require.Len(m.T, moduleVersions.ModuleVersions, 1)
	require.NotNil(m.T, moduleVersions.ModuleVersions[0])
	return moduleVersions.ModuleVersions[0].Version
}

// wait for the block before the upgrade, then sleep to give
// the cosmovisor time to reboot the node software
func waitForUpgrade(m testCommon.TestConfig, proposalHeight int64) {
	ctx := context.Background()
	var timeToSleep time.Duration = 15
	m.Client.WaitForBlockHeight(ctx, proposalHeight-1)
	m.T.Logf("--- Block Height %d Reached, Preparing to Sleep %d while Upgrade Happens ---",
		proposalHeight-1, timeToSleep)
	time.Sleep(timeToSleep * time.Second)
}

func UpgradeChecks(m testCommon.TestConfig) {
	m.T.Log("--- Getting Emissions Module Version Before Upgrade ---")
	emissionsVersionBefore := getEmissionsVersion(m)
	m.T.Log("--- Propose Upgrade to vintegration software version from v0 ---")
	proposalId, proposalHeight := proposeUpgrade(m)
	m.T.Logf("--- Vote on Upgrade Proposal %d ---", proposalId)
	voteOnProposal(m, proposalId)
	m.T.Logf("--- Wating for Proposal %d to Pass ---", proposalId)
	waitForProposalPass(m, proposalId)
	m.T.Logf("--- Waiting for Upgrade to vintegration at height %d ---", proposalHeight)
	waitForUpgrade(m, proposalHeight)
	m.T.Log("--- Getting Emissions Module Version After Upgrade ---")
	emissionsVersionAfter := getEmissionsVersion(m)
	m.T.Log("--- Checking Emissions Module Version Has Been Upgraded ---")
	require.Greater(m.T, emissionsVersionAfter, emissionsVersionBefore)
}
