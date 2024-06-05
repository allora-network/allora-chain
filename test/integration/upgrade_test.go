package integration_test

import (
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	testCommon "github.com/allora-network/allora-chain/test/common"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/stretchr/testify/require"
)

func getDepositRequired(m testCommon.TestConfig) sdktypes.Coin {
	queryGovParamsResponse, err := m.Client.QueryGov().Params(m.Ctx, &govtypesv1.QueryParamsRequest{})
	require.NoError(m.T, err)
	return queryGovParamsResponse.Params.ExpeditedMinDeposit[0]
}

func voteOnProposal(m testCommon.TestConfig, proposalId uint64) {
	fmt.Println("todo")
}

func proposeUpgrade(m testCommon.TestConfig) uint64 {
	name := "vIntegration"
	summary := "Upgrade to vIntegration software version"
	msgSoftwareUpgrade := &upgradetypes.MsgSoftwareUpgrade{
		Authority: authtypes.NewModuleAddress("gov").String(),
		Plan: upgradetypes.Plan{
			Name:   name,
			Height: 1e6,
			Info:   "{}",
		},
	}
	submitProposalMsg := &govtypesv1.MsgSubmitProposal{
		Title:    name,
		Summary:  summary,
		Proposer: m.Validator0Addr,
		Metadata: fmt.Sprintf(
			"{title:\"%s\",summary:\"%s\"}", name, summary,
		), // metadata must match title and summary exactly
		Expedited: true,
		InitialDeposit: sdktypes.NewCoins(
			getDepositRequired(m),
		),
	}
	submitProposalMsg.SetMsgs([]sdktypes.Msg{msgSoftwareUpgrade})
	txResp, err := m.Client.BroadcastTx(m.Ctx, m.Validator0Acc, submitProposalMsg)
	require.NoError(m.T, err)
	_, err = m.Client.WaitForTx(m.Ctx, txResp.TxHash)
	require.NoError(m.T, err)
	submitProposalMsgResponse := &govtypesv1.MsgSubmitProposalResponse{}
	err = txResp.Decode(submitProposalMsgResponse)
	require.NoError(m.T, err)
	require.NotNil(m.T, submitProposalMsgResponse.ProposalId)
	return submitProposalMsgResponse.ProposalId
}

func UpgradeChecks(m testCommon.TestConfig) {
	m.T.Log("--- Propose Upgrade to vIntegration software version from v0 ---")
	proposalId := proposeUpgrade(m)
	m.T.Logf("--- Vote on Upgrade Proposal %d ---", proposalId)
	voteOnProposal(m, proposalId)
}
