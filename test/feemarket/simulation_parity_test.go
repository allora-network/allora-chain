package integration_test

import (
	"math/rand"
	"testing"

	"github.com/allora-network/allora-chain/app"
	abci "github.com/cometbft/cometbft/abci/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	"github.com/stretchr/testify/suite"
)

// parityGasLimit is a gas limit comfortably above the execution cost of a
// single MsgSend, leaving room for the fee path.
const parityGasLimit uint64 = 2_000_000

// parityMaxSurplus bounds how far simulated gas may exceed delivered gas.
// The surplus is the simulation compensation minus the real fee path cost,
// so it must stay below the compensation itself: the fee path at deliver has
// to cost more than the single bank send the feemarket post handler already
// compensates, otherwise the compensation over-covers the fee path (for
// example after a fee path change or a dependency change that double
// counts).
const parityMaxSurplus = app.FeePathSimGasCompensation

type SimulationParityTestSuite struct {
	suite.Suite
	coordinator *ibctesting.Coordinator
	chain       *ibctesting.TestChain
	app         *app.AlloraApp
}

func TestSimulationParityTestSuite(t *testing.T) {
	suite.Run(t, new(SimulationParityTestSuite))
}

func (suite *SimulationParityTestSuite) SetupTest() {
	app.UseFeeMarketDecorator = true
	ibctesting.DefaultTestingAppInit = alloraAppInitializer
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1)
	OverrideSendMsgs(suite.coordinator.Chains, sdk.NewInt64Coin(sdk.DefaultBondDenom, LargeFeeAmount), LargeGasLimit)

	chain, ok := suite.coordinator.Chains[ibctesting.GetChainID(1)]
	suite.Require().True(ok, "chain not found")
	suite.chain = chain
	suite.chain.CurrentHeader.ProposerAddress = sdk.ConsAddress(suite.chain.Vals.Validators[0].Address)

	testApp, ok := chain.App.(*app.AlloraApp)
	suite.Require().True(ok, "expected App to be AlloraApp")
	suite.app = testApp
}

// simulateAndDeliver signs the given messages with the given fee and gas
// limit, simulates the signed bytes against the committed state, then
// delivers the exact same bytes and returns both gas measurements.
func (suite *SimulationParityTestSuite) simulateAndDeliver(msgs []sdk.Msg, fee sdk.Coin, gasLimit uint64) (simGas uint64, deliverGas uint64) {
	suite.T().Helper()

	sender := suite.chain.SenderAccount

	tx, err := simtestutil.GenSignedMockTx(
		rand.New(rand.NewSource(1)),
		suite.chain.TxConfig,
		msgs,
		sdk.Coins{fee},
		gasLimit,
		suite.chain.ChainID,
		[]uint64{sender.GetAccountNumber()},
		[]uint64{sender.GetSequence()},
		suite.chain.SenderPrivKey,
	)
	suite.Require().NoError(err)

	txBytes, err := suite.chain.TxConfig.TxEncoder()(tx)
	suite.Require().NoError(err)

	gasInfo, _, err := suite.app.GetBaseApp().Simulate(txBytes)
	suite.Require().NoError(err, "simulation failed")

	resp, err := suite.app.GetBaseApp().FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:             suite.app.GetBaseApp().LastBlockHeight() + 1,
		Time:               suite.chain.CurrentHeader.GetTime(),
		NextValidatorsHash: suite.chain.NextVals.Hash(),
		Txs:                [][]byte{txBytes},
		DecidedLastCommit: abci.CommitInfo{
			Round: 0,
			Votes: nil,
		},
		Hash:            nil,
		Misbehavior:     nil,
		ProposerAddress: suite.chain.CurrentHeader.ProposerAddress,
	})
	suite.Require().NoError(err)
	suite.Require().Len(resp.TxResults, 1)
	txResult := resp.TxResults[0]
	suite.Require().Zero(txResult.Code, "tx execution failed: %s", txResult.Log)

	err = sender.SetSequence(sender.GetSequence() + 1)
	suite.Require().NoError(err)

	return gasInfo.GasUsed, uint64(txResult.GasUsed)
}

// Wallets derive gas limits from the simulate endpoint and apply a small gas
// adjustment on top. Simulated gas must therefore cover the delivered gas of
// the same transaction, otherwise every standard estimation lands below the
// true execution cost and transactions fail with out of gas.
func (suite *SimulationParityTestSuite) TestSimulatedGasCoversDeliveredGas() {
	msg := banktypes.NewMsgSend(
		suite.chain.SenderAccount.GetAddress(),
		suite.chain.SenderAccounts[1].SenderAccount.GetAddress(),
		sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)),
	)
	fee := sdk.NewInt64Coin(sdk.DefaultBondDenom, LargeFeeAmount)
	simGas, deliverGas := suite.simulateAndDeliver([]sdk.Msg{msg}, fee, parityGasLimit)

	suite.T().Logf("sim gas: %d, deliver gas: %d", simGas, deliverGas)
	suite.GreaterOrEqual(simGas, deliverGas, "simulated gas must cover delivered gas")
	suite.Less(simGas-deliverGas, parityMaxSurplus, "simulated gas over-covers delivered gas")
}

// A gas price above the base gas price pays a tip to the block proposer; the
// first payout to a proposer account creates it, which is the most expensive
// shape of the fee path at deliver. The simulated gas must still cover the
// delivered gas in that case.
func (suite *SimulationParityTestSuite) TestSimulatedGasCoversDeliveredGasWithFreshTipRecipient() {
	msg := banktypes.NewMsgSend(
		suite.chain.SenderAccount.GetAddress(),
		suite.chain.SenderAccounts[1].SenderAccount.GetAddress(),
		sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)),
	)
	fee := sdk.NewInt64Coin(sdk.DefaultBondDenom, 2*LargeFeeAmount)
	simGas, deliverGas := suite.simulateAndDeliver([]sdk.Msg{msg}, fee, parityGasLimit)

	suite.T().Logf("sim gas: %d, deliver gas: %d", simGas, deliverGas)
	suite.GreaterOrEqual(simGas, deliverGas, "simulated gas must cover delivered gas")
	suite.Less(simGas-deliverGas, parityMaxSurplus, "simulated gas over-covers delivered gas")
}

// The minimum valid fee under the fee market is ceil(min gas price × gas
// limit), the boundary where the post handler's fee check accepts the fee
// with equality (payCoin == required fee). The fee prices the gas limit while
// the tip is the difference against the gas actually consumed, so even at the
// base gas price the tip stays positive (a literal zero-tip tx would require
// gas used == gas limit) and the full fee path still runs at deliver. The
// simulation compensation must cover this boundary case as well.
func (suite *SimulationParityTestSuite) TestSimulatedGasCoversDeliveredGasMinFee() {
	msg := banktypes.NewMsgSend(
		suite.chain.SenderAccount.GetAddress(),
		suite.chain.SenderAccounts[1].SenderAccount.GetAddress(),
		sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)),
	)

	minGasPrice, err := suite.app.FeeMarketKeeper.GetMinGasPrice(suite.chain.GetContext(), sdk.DefaultBondDenom)
	suite.Require().NoError(err)
	feeAmount := minGasPrice.Amount.MulInt64(int64(parityGasLimit)).Ceil().RoundInt()
	fee := sdk.NewCoin(sdk.DefaultBondDenom, feeAmount)

	simGas, deliverGas := suite.simulateAndDeliver([]sdk.Msg{msg}, fee, parityGasLimit)

	suite.T().Logf("sim gas: %d, deliver gas: %d", simGas, deliverGas)
	suite.GreaterOrEqual(simGas, deliverGas, "simulated gas must cover delivered gas")
	suite.Less(simGas-deliverGas, parityMaxSurplus, "simulated gas over-covers delivered gas")
}

// Large fee amounts increase the bytes written by the fee path (escrow,
// deduction and tip payouts) at deliver via WritePerByte, so the simulation
// compensation must also hold for transactions carrying very large fees.
func (suite *SimulationParityTestSuite) TestSimulatedGasCoversDeliveredGasLargeFee() {
	msg := banktypes.NewMsgSend(
		suite.chain.SenderAccount.GetAddress(),
		suite.chain.SenderAccounts[1].SenderAccount.GetAddress(),
		sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)),
	)
	// A 19-digit fee (well above the 10-digit LargeFeeAmount used elsewhere) to
	// exercise the fee-path byte variance. The genesis sender is funded with
	// 1e19 stake, so it can pay.
	fee := sdk.NewInt64Coin(sdk.DefaultBondDenom, 1_000_000_000_000_000_000)
	simGas, deliverGas := suite.simulateAndDeliver([]sdk.Msg{msg}, fee, parityGasLimit)

	suite.T().Logf("sim gas: %d, deliver gas: %d", simGas, deliverGas)
	suite.GreaterOrEqual(simGas, deliverGas, "simulated gas must cover delivered gas")
	suite.Less(simGas-deliverGas, parityMaxSurplus, "simulated gas over-covers delivered gas")
}

// The fee path is per-transaction: one escrow, one fee deduction and one tip
// payout regardless of the payload size, so the simulation compensation must
// hold for heavy multi-message transactions just as for single-message ones.
func (suite *SimulationParityTestSuite) TestSimulatedGasCoversDeliveredGasHeavyTx() {
	sender := suite.chain.SenderAccount.GetAddress()
	receiver := suite.chain.SenderAccounts[1].SenderAccount.GetAddress()
	msgs := make([]sdk.Msg, 50)
	for i := range msgs {
		msgs[i] = banktypes.NewMsgSend(sender, receiver, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)))
	}

	fee := sdk.NewInt64Coin(sdk.DefaultBondDenom, LargeFeeAmount)
	simGas, deliverGas := suite.simulateAndDeliver(msgs, fee, 5_000_000)

	suite.T().Logf("sim gas: %d, deliver gas: %d", simGas, deliverGas)
	suite.GreaterOrEqual(simGas, deliverGas, "simulated gas must cover delivered gas")
	suite.Less(simGas-deliverGas, parityMaxSurplus, "simulated gas over-covers delivered gas")
}
