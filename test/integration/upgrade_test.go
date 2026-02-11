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
	queryGovParamsResponse, err := m.Client.QueryGov().Params(ctx, &govtypesv1.QueryParamsRequest{
		ParamsType: "",
	})
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
			Metadata:   "",
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

// propose an upgrade to the specified software version
func proposeUpgrade(m testCommon.TestConfig, upgradeName string, planHeightDelta int64) (proposalId uint64, proposalHeight int64) {
	ctx := context.Background()
	name := upgradeName
	summary := "Upgrade to " + name + " software version"

	currHeight, err := m.Client.BlockHeight(ctx)
	require.NoError(m.T, err)
	if planHeightDelta <= 0 {
		planHeightDelta = DefaultPlanHeightDelta
	}
	proposalHeight = currHeight + planHeightDelta // 4 1/6 minutes
	m.T.Logf("Current Height: %d, proposing upgrade for %d", currHeight, proposalHeight)
	msgSoftwareUpgrade := &upgradetypes.MsgSoftwareUpgrade{
		Authority: authtypes.NewModuleAddress("gov").String(),
		Plan: upgradetypes.Plan{
			Name:                name,
			Height:              proposalHeight,
			Info:                "{}",
			Time:                time.Time{},
			UpgradedClientState: nil,
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
		Messages: nil,
	}
	err = msgSubmitProposal.SetMsgs([]sdktypes.Msg{msgSoftwareUpgrade})
	require.NoError(m.T, err)
	txResp, err := m.Client.BroadcastTx(ctx, m.AliceAcc, msgSubmitProposal)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(ctx, txResp.TxHash)
	require.NoError(m.T, err)
	submitProposalMsgResponse := &govtypesv1.MsgSubmitProposalResponse{} //nolint:exhaustruct // the fields are populated by decode
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

type moduleVersionSnapshot struct {
	Version uint64
	Found   bool
}

// query the current version of a module, returning whether it exists
func getModuleVersion(m testCommon.TestConfig, moduleName string) moduleVersionSnapshot {
	ctx := context.Background()
	queryModuleVersionsRequest := &upgradetypes.QueryModuleVersionsRequest{
		ModuleName: "",
	}
	moduleVersions, err := m.Client.QueryUpgrade().ModuleVersions(ctx, queryModuleVersionsRequest)
	require.NoError(m.T, err)
	require.NotNil(m.T, moduleVersions)

	for _, mv := range moduleVersions.ModuleVersions {
		if mv.Name == moduleName {
			return moduleVersionSnapshot{
				Version: mv.Version,
				Found:   true,
			}
		}
	}

	return moduleVersionSnapshot{
		Version: 0,
		Found:   false,
	}
}

// wait for the block before the upgrade, then sleep to give
// the cosmovisor time to reboot the node software
func waitForUpgrade(m testCommon.TestConfig, proposalHeight int64) {
	ctx := context.Background()
	var timeToSleep = 15 * time.Second
	err := m.Client.WaitForBlockHeight(ctx, proposalHeight-1)
	require.NoError(m.T, err)
	m.T.Logf("--- Block Height %d Reached, Preparing to Sleep %d while Upgrade Happens ---",
		proposalHeight-1, timeToSleep)
	time.Sleep(timeToSleep)
}

func getAppliedVersionHeight(m testCommon.TestConfig, version string) int64 {
	ctx := context.Background()

	queryAppliedPlanRequest := &upgradetypes.QueryAppliedPlanRequest{
		Name: version,
	}
	queryAppliedPlanResponse, err := m.Client.QueryUpgrade().AppliedPlan(ctx, queryAppliedPlanRequest)
	require.NoError(m.T, err)
	require.NotNil(m.T, queryAppliedPlanResponse)
	return queryAppliedPlanResponse.Height
}

func UpgradeChecks(m testCommon.TestConfig, upgradeName string) {
	scenario, ok := UpgradeScenarios[upgradeName]
	require.Truef(m.T, ok, "upgrade scenario %s not defined", upgradeName)

	planHeightDelta := scenario.PlanHeightDelta
	if planHeightDelta <= 0 {
		planHeightDelta = DefaultPlanHeightDelta
	}

	m.T.Logf("--- Preparing upgrade scenario for %s ---", scenario.UpgradeName)

	preUpgradeVersions := make(map[string]moduleVersionSnapshot, len(scenario.ModuleChecks))
	for _, check := range scenario.ModuleChecks {
		snapshot := getModuleVersion(m, check.ModuleName)
		preUpgradeVersions[check.ModuleName] = snapshot
		if snapshot.Found {
			m.T.Logf("Module %s pre-upgrade consensus version: %d", check.ModuleName, snapshot.Version)
		} else {
			m.T.Logf("Module %s not found before upgrade", check.ModuleName)
		}
	}

	m.T.Logf("--- Propose Upgrade to %s software version ---", scenario.UpgradeName)
	proposalId, proposalHeight := proposeUpgrade(m, scenario.UpgradeName, planHeightDelta)
	m.T.Logf("--- Vote on Upgrade Proposal %d ---", proposalId)
	voteOnProposal(m, proposalId)
	m.T.Logf("--- Waiting for Proposal %d to Pass ---", proposalId)
	waitForProposalPass(m, proposalId)
	m.T.Logf("--- Waiting for Upgrade to %s at height %d ---", scenario.UpgradeName, proposalHeight)
	waitForUpgrade(m, proposalHeight)

	for _, check := range scenario.ModuleChecks {
		m.T.Logf("--- Validating module %s ---", check.ModuleName)
		preSnapshot := preUpgradeVersions[check.ModuleName]
		postSnapshot := getModuleVersion(m, check.ModuleName)

		switch check.CheckType {
		case ModuleCheckExpectIncrease:
			require.Truef(m.T, preSnapshot.Found, "module %s expected to exist before upgrade", check.ModuleName)
			require.Truef(m.T, postSnapshot.Found, "module %s expected to exist after upgrade", check.ModuleName)
			require.Greaterf(m.T, postSnapshot.Version, preSnapshot.Version, "module %s consensus version did not increase", check.ModuleName)
		case ModuleCheckExpectEqual:
			require.Equalf(m.T, preSnapshot.Found, postSnapshot.Found, "module %s presence changed unexpectedly", check.ModuleName)
			if preSnapshot.Found {
				require.Equalf(m.T, preSnapshot.Version, postSnapshot.Version, "module %s consensus version changed unexpectedly", check.ModuleName)
			}
		case ModuleCheckExpectNew:
			require.Falsef(m.T, preSnapshot.Found, "module %s should not exist before upgrade", check.ModuleName)
			require.Truef(m.T, postSnapshot.Found, "module %s expected to exist after upgrade", check.ModuleName)
			if check.ExpectedVersion == nil {
				require.Positivef(m.T, postSnapshot.Version, "module %s consensus version should be greater than 0", check.ModuleName)
			}
		default:
			require.Failf(m.T, "unsupported module check type", "module %s has unsupported check type %s", check.ModuleName, check.CheckType)
		}

		if check.ExpectedVersion != nil {
			require.Truef(m.T, postSnapshot.Found, "module %s expected to exist after upgrade for version assertion", check.ModuleName)
			require.Equalf(m.T, *check.ExpectedVersion, postSnapshot.Version, "module %s consensus version mismatch", check.ModuleName)
		}
	}

	height := getAppliedVersionHeight(m, scenario.UpgradeName)
	m.T.Log("--- Checking upgrade has been applied at the proposed height ---")
	require.Equal(m.T, height, proposalHeight)
}
