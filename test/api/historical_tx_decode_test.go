package api

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

// TestHistoricalTxDoesNotMoveConsensus is the end-to-end regression PR #985 lacked.
// It delivers, through a real AlloraApp FinalizeBlock, a historical v9 payload
// wrapped in a routable authz.MsgExec (the exact vector that made the global-registry
// fix consensus-breaking) and asserts consensus state is untouched: the tx is
// rejected at decode with no gas, no fee deduction, and no sequence increment. If a
// future change ever widened the consensus decoder, the ante handler would run and
// this test's balance/sequence assertions would fail.
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

	// Consensus decoder rejected it: non-zero code and, crucially, nothing executed.
	require.NotZero(t, resp.TxResults[0].Code, "wrapped historical tx must be rejected")
	require.Zero(t, resp.TxResults[0].GasUsed, "a rejected-at-decode tx must not consume gas")

	// The state the PR #985 hazard mutated must be unchanged.
	ctxAfter := chain.GetContext()
	balAfter := alloraApp.BankKeeper.GetBalance(ctxAfter, sender, sdk.DefaultBondDenom)
	seqAfter := alloraApp.AccountKeeper.GetAccount(ctxAfter, sender).GetSequence()
	require.Equal(t, balBefore.Amount.String(), balAfter.Amount.String(), "no fee may be deducted")
	require.Equal(t, seqBefore, seqAfter, "no sequence increment may occur")
}
