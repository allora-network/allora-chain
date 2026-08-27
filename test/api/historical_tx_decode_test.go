package api

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/stretchr/testify/require"

	emissionsv3 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v3"
	emissionsv9 "github.com/allora-network/allora-chain/x/emissions/api/emissions/v9"
)

// historicalV9WorkerPayload is a pre-v10 worker payload: the shape that stopped
// decoding after the v10 proto bump.
func historicalV9WorkerPayload(sender string) sdk.Msg {
	//nolint:exhaustruct // only the fields exercised by the nested walk are set
	return &emissionsv9.InsertWorkerPayloadRequest{
		Sender: sender,
		WorkerDataBundle: &emissionsv9.InputWorkerDataBundle{
			Worker:  sender,
			Nonce:   &emissionsv3.Nonce{BlockHeight: 1},
			TopicId: 1,
			InferenceForecastsBundle: &emissionsv9.InputInferenceForecastBundle{
				Inference: &emissionsv9.InputInference{
					TopicId: 1, BlockHeight: 1, Inferer: sender, Value: "1.0",
				},
			},
			InferencesForecastsBundleSignature: []byte{0x01},
			Pubkey:                             "pubkey",
		},
	}
}

// TestHistoricalTxDoesNotMoveConsensus delivers a historical v9 payload wrapped in
// a routable authz.MsgExec through a real AlloraApp FinalizeBlock and asserts
// consensus state is untouched: rejected at decode, with no gas, no fee deduction
// and no sequence increment. A decoder that accepted the nested payload would run
// the ante handler and move all four. TestHarnessObservesExecutedTx is the positive
// control showing those reads can observe a tx that does execute.
func TestHistoricalTxDoesNotMoveConsensus(t *testing.T) {
	chain, alloraApp := SetupChain(t)
	sender := chain.SenderAccount.GetAddress()

	// Wrap the historical payload in a message that IS routable today, so it clears
	// baseapp's pre-ante routing check (which only inspects top-level messages).
	inner, err := codectypes.NewAnyWithValue(historicalV9WorkerPayload(sender.String()))
	require.NoError(t, err)
	//nolint:exhaustruct // only Grantee and Msgs matter
	wrapped := &authz.MsgExec{Grantee: sender.String(), Msgs: []*codectypes.Any{inner}}

	ctx := chain.GetContext()
	balBefore := alloraApp.BankKeeper.GetBalance(ctx, sender, sdk.DefaultBondDenom)
	seqBefore := alloraApp.AccountKeeper.GetAccount(ctx, sender).GetSequence()

	resp := SignAndDeliver(t, chain, wrapped)
	require.Len(t, resp.TxResults, 1)

	// Pinning the decode error is what distinguishes this from an ante-handler
	// rejection, which is the outcome a widened decoder would produce.
	require.Equal(t, sdkerrors.ErrTxDecode.ABCICode(), resp.TxResults[0].Code,
		"wrapped historical tx must be rejected at decode")
	require.Equal(t, sdkerrors.ErrTxDecode.Codespace(), resp.TxResults[0].Codespace)
	require.Zero(t, resp.TxResults[0].GasUsed, "a rejected-at-decode tx must not consume gas")

	// State a widened decoder would have moved must be unchanged.
	ctxAfter := chain.GetContext()
	balAfter := alloraApp.BankKeeper.GetBalance(ctxAfter, sender, sdk.DefaultBondDenom)
	seqAfter := alloraApp.AccountKeeper.GetAccount(ctxAfter, sender).GetSequence()
	require.Equal(t, balBefore.Amount.String(), balAfter.Amount.String(), "no fee may be deducted")
	require.Equal(t, seqBefore, seqAfter, "no sequence increment may occur")
}
