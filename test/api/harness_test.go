package api

// Shared harness for API regression tests. It boots a real AlloraApp on an
// in-memory db via the ibctesting scaffolding (funded sender, single validator)
// and delivers signed txs through a real FinalizeBlock, so tests exercise the
// whole ABCI path rather than a decoder in isolation.

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	"github.com/stretchr/testify/require"

	"github.com/allora-network/allora-chain/app"
)

const (
	// LargeFeeAmount and LargeGasLimit are generous so a delivered tx is never
	// rejected for fee/gas reasons; tests assert on decode/exec outcomes.
	LargeFeeAmount = 1_000_000_000
	LargeGasLimit  = simtestutil.DefaultGenTxGas * 10
)

// AppInitializer is the ibctesting hook that builds the AlloraApp under test.
func AppInitializer() (ibctesting.TestingApp, map[string]json.RawMessage) {
	testApp, err := app.NewAlloraApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		simtestutil.EmptyAppOptions{},
	)
	if err != nil {
		// The hook signature cannot return an error, and a nil app surfaces as an
		// unrelated nil dereference inside the coordinator.
		panic(fmt.Errorf("initialize AlloraApp test fixture: %w", err))
	}
	return testApp, testApp.DefaultGenesis()
}

// SetupChain returns a single-validator chain running AlloraApp plus the concrete
// app handle for reading state (balances, sequences, the query tx config).
func SetupChain(t *testing.T) (*ibctesting.TestChain, *app.AlloraApp) {
	t.Helper()
	app.UseFeeMarketDecorator = true
	ibctesting.DefaultTestingAppInit = AppInitializer

	coordinator := ibctesting.NewCoordinator(t, 1)
	chain, ok := coordinator.Chains[ibctesting.GetChainID(1)]
	require.True(t, ok, "chain not found")
	chain.CurrentHeader.ProposerAddress = sdk.ConsAddress(chain.Vals.Validators[0].Address)

	alloraApp, ok := chain.App.(*app.AlloraApp)
	require.True(t, ok, "expected App to be AlloraApp")
	return chain, alloraApp
}

// SignAndDeliver signs msgs from the chain's default sender and runs one block.
// It asserts only that the block itself was produced; per-tx success or failure
// is left for the caller to read from the returned TxResults.
func SignAndDeliver(t *testing.T, chain *ibctesting.TestChain, msgs ...sdk.Msg) *abci.ResponseFinalizeBlock {
	t.Helper()
	tx, err := simtestutil.GenSignedMockTx(
		rand.New(rand.NewSource(1)), // fixed seed: deterministic tx bytes
		chain.TxConfig,
		msgs,
		sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, LargeFeeAmount)},
		LargeGasLimit,
		chain.ChainID,
		[]uint64{chain.SenderAccount.GetAccountNumber()},
		[]uint64{chain.SenderAccount.GetSequence()},
		chain.SenderPrivKey,
	)
	require.NoError(t, err)

	txBytes, err := chain.TxConfig.TxEncoder()(tx)
	require.NoError(t, err)

	//nolint:exhaustruct // only these block fields are needed to deliver one tx
	resp, err := chain.App.GetBaseApp().FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:             chain.App.GetBaseApp().LastBlockHeight() + 1,
		Time:               chain.CurrentHeader.GetTime(),
		NextValidatorsHash: chain.NextVals.Hash(),
		Txs:                [][]byte{txBytes},
	})
	require.NoError(t, err, "block-level failure; per-tx result is in resp.TxResults")
	return resp
}
