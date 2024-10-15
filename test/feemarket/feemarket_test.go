package integration_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app"
	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	bam "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	feemarkettypes "github.com/skip-mev/feemarket/x/feemarket/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	LargeMsgNumber = 1000
	LargeFeeAmount = 1000000000
	LargeGasLimit  = simtestutil.DefaultGenTxGas * 10
)

type FeeMarketTestSuite struct {
	suite.Suite
	coordinator *ibctesting.Coordinator
	chain       *ibctesting.TestChain
	app         *app.AlloraApp
}

func TestFeeMarketTestSuite(t *testing.T) {
	feemarketSuite := &FeeMarketTestSuite{}
	suite.Run(t, feemarketSuite)
}

func (suite *FeeMarketTestSuite) SetupTest() {
	app.UseFeeMarketDecorator = true
	ibctesting.DefaultTestingAppInit = alloraAppInitializer
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1)
	OverrideSendMsgs(suite.coordinator.Chains, sdk.NewInt64Coin(sdk.DefaultBondDenom, LargeFeeAmount), LargeGasLimit)

	chain, ok := suite.coordinator.Chains[ibctesting.GetChainID(1)]
	suite.Require().True(ok, "chain not found")
	suite.chain = chain
	suite.chain.CurrentHeader.ProposerAddress = sdk.ConsAddress(suite.chain.Vals.Validators[0].Address)

	app, ok := chain.App.(*app.AlloraApp)
	suite.Require().True(ok, "expected App to be AlloraApp")
	suite.app = app
}

func (suite *FeeMarketTestSuite) TestBaseFeeAdjustment() {
	// BaseFee is initially set to DefaultMinBaseGasPrice
	ctx := suite.chain.GetContext()

	baseFee, err := suite.app.FeeMarketKeeper.GetBaseGasPrice(ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(feemarkettypes.DefaultMinBaseGasPrice, baseFee)

	// BaseFee can not be lower than DefaultMinBaseGasPrice, even after N empty blocks
	suite.coordinator.CommitNBlocks(suite.chain, 10)

	ctx = suite.chain.GetContext()
	baseFee, err = suite.app.FeeMarketKeeper.GetBaseGasPrice(ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(feemarkettypes.DefaultMinBaseGasPrice, baseFee)

	// Send a large transaction to consume a lot of gas
	sender := suite.chain.SenderAccounts[0].SenderAccount.GetAddress()
	receiver := suite.chain.SenderAccounts[1].SenderAccount.GetAddress()
	amount := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(10)))

	msgs := make([]sdk.Msg, LargeMsgNumber)
	for i := 0; i < LargeMsgNumber; i++ {
		bankSendMsg := banktypes.NewMsgSend(sender, receiver, amount)
		msgs[i] = bankSendMsg
	}

	_, err = suite.chain.SendMsgs(msgs...)
	suite.Require().NoError(err)

	// Check that BaseFee has increased due to the large gas usage
	ctx = suite.chain.GetContext()
	baseFee, err = suite.app.FeeMarketKeeper.GetBaseGasPrice(ctx)
	suite.Require().NoError(err)
	suite.Require().True(baseFee.GT(feemarkettypes.DefaultMinBaseGasPrice))

	// BaseFee should drop to DefaultMinBaseGasPrice after N empty blocks
	suite.coordinator.CommitNBlocks(suite.chain, 10)

	ctx = suite.chain.GetContext()
	baseFee, err = suite.app.FeeMarketKeeper.GetBaseGasPrice(ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(feemarkettypes.DefaultMinBaseGasPrice, baseFee)
}

func alloraAppInitializer() (ibctesting.TestingApp, map[string]json.RawMessage) {
	testApp, err := app.NewAlloraApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		simtestutil.EmptyAppOptions{},
	)
	if err != nil {
		return nil, nil
	}

	return testApp, testApp.DefaultGenesis()
}

// SendMsgs() behavior must be changed since the default one uses zero fees
func OverrideSendMsgs(chains map[string]*ibctesting.TestChain, feeAmount sdk.Coin, gasLimit uint64) {
	for _, chain := range chains {
		chain := chain
		chain.SendMsgsOverride = func(msgs ...sdk.Msg) (*abci.ExecTxResult, error) {
			return SendMsgsOverride(chain, feeAmount, gasLimit, msgs...)
		}
	}
}

func SendMsgsOverride(chain *ibctesting.TestChain, feeAmount sdk.Coin, gasLimit uint64, msgs ...sdk.Msg) (*abci.ExecTxResult, error) {
	// ensure the chain has the latest time
	chain.Coordinator.UpdateTimeForChain(chain)

	// increment acc sequence regardless of success or failure tx execution
	defer func() {
		err := chain.SenderAccount.SetSequence(chain.SenderAccount.GetSequence() + 1)
		if err != nil {
			panic(err)
		}
	}()

	resp, err := SignAndDeliver(
		chain.TB,
		chain.TxConfig,
		chain.App.GetBaseApp(),
		msgs,
		chain.ChainID,
		[]uint64{chain.SenderAccount.GetAccountNumber()},
		[]uint64{chain.SenderAccount.GetSequence()},
		true,
		chain.CurrentHeader.GetTime(),
		chain.NextVals.Hash(),
		feeAmount,
		gasLimit,
		chain.SenderPrivKey,
	)
	if err != nil {
		return nil, err
	}

	require.Len(chain.TB, resp.TxResults, 1)
	txResult := resp.TxResults[0]

	if txResult.Code != 0 {
		return txResult, fmt.Errorf("%s/%d: %q", txResult.Codespace, txResult.Code, txResult.Log)
	}

	chain.Coordinator.IncrementTime()

	return txResult, nil
}

func SignAndDeliver(
	tb testing.TB, txCfg client.TxConfig, app *bam.BaseApp, msgs []sdk.Msg,
	chainID string, accNums, accSeqs []uint64, expPass bool, blockTime time.Time, nextValHash []byte, feeAmount sdk.Coin, gasLimit uint64, priv ...cryptotypes.PrivKey,
) (*abci.ResponseFinalizeBlock, error) {
	tb.Helper()
	tx, err := simtestutil.GenSignedMockTx(
		rand.New(rand.NewSource(time.Now().UnixNano())),
		txCfg,
		msgs,
		sdk.Coins{feeAmount},
		gasLimit,
		chainID,
		accNums,
		accSeqs,
		priv...,
	)
	require.NoError(tb, err)

	txBytes, err := txCfg.TxEncoder()(tx)
	require.NoError(tb, err)

	return app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:             app.LastBlockHeight() + 1,
		Time:               blockTime,
		NextValidatorsHash: nextValHash,
		Txs:                [][]byte{txBytes},
	})
}
