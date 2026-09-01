package api

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

// TestHarnessObservesExecutedTx is the positive control for the balance and
// sequence assertions in TestHistoricalTxDoesNotMoveConsensus: it proves those
// reads can observe a tx that executes, so an unchanged balance there is evidence
// rather than an artifact of the harness. FinalizeBlock writes the block's state
// transitions into the root multistore before returning, so chain.GetContext()
// sees them without a separate Commit.
func TestHarnessObservesExecutedTx(t *testing.T) {
	chain, alloraApp := SetupChain(t)
	sender := chain.SenderAccount.GetAddress()
	recipient := sdk.AccAddress([]byte("recipient___________"))

	//nolint:exhaustruct // only these fields matter for the control
	msg := &banktypes.MsgSend{
		FromAddress: sender.String(),
		ToAddress:   recipient.String(),
		Amount:      sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 12345)),
	}

	ctx := chain.GetContext()
	balBefore := alloraApp.BankKeeper.GetBalance(ctx, sender, sdk.DefaultBondDenom)
	seqBefore := alloraApp.AccountKeeper.GetAccount(ctx, sender).GetSequence()

	resp := SignAndDeliver(t, chain, msg)
	require.Len(t, resp.TxResults, 1)
	require.Zero(t, resp.TxResults[0].Code, "control tx must execute: %s", resp.TxResults[0].Log)
	require.NotZero(t, resp.TxResults[0].GasUsed, "an executed tx must consume gas")

	ctxAfter := chain.GetContext()
	balAfter := alloraApp.BankKeeper.GetBalance(ctxAfter, sender, sdk.DefaultBondDenom)
	seqAfter := alloraApp.AccountKeeper.GetAccount(ctxAfter, sender).GetSequence()
	require.True(t, balAfter.Amount.LT(balBefore.Amount), "an executed tx must move the balance")
	require.Equal(t, seqBefore+1, seqAfter, "an executed tx must increment the sequence")
}
