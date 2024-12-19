package ibc

import (
	"cosmossdk.io/log"

	"encoding/json"

	dbm "github.com/cosmos/cosmos-db"

	"testing"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/app"
	"github.com/allora-network/allora-chain/app/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	"github.com/stretchr/testify/suite"
)

var (
	nativeDenom            = params.DefaultBondDenom
	ibcTransferAmount      = math.NewInt(100_000)
	genesisWalletAmount, _ = math.NewIntFromString("10000000000000000000")
)

type IBCTestSuite struct {
	suite.Suite

	coordinator  *ibctesting.Coordinator
	alloraChainA *ibctesting.TestChain // aka chainA
	alloraChainB *ibctesting.TestChain

	path *ibctesting.Path

	chainAAddr     sdk.AccAddress
	chainBAddr     sdk.AccAddress
	chainBIBCDenom string
}

func TestIBCTestSuite(t *testing.T) {
	suite.Run(t, new(IBCTestSuite))
}

func InitAlloraApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	app.UseFeeMarketDecorator = false
	alloraApp, err := app.NewAlloraApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		simtestutil.EmptyAppOptions{},
	)
	if err != nil {
		return nil, nil
	}

	return alloraApp, alloraApp.DefaultGenesis()
}

func (s *IBCTestSuite) SetupTest() {
	sdk.DefaultBondDenom = nativeDenom
	ibctesting.DefaultTestingAppInit = InitAlloraApp

	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.alloraChainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.alloraChainB = s.coordinator.GetChain(ibctesting.GetChainID(2))

	s.path = ibctesting.NewPath(s.alloraChainA, s.alloraChainB)
	s.path.EndpointA.ChannelConfig.PortID = ibctesting.TransferPort
	s.path.EndpointB.ChannelConfig.PortID = ibctesting.TransferPort
	s.path.EndpointA.ChannelConfig.Version = transfertypes.Version
	s.path.EndpointB.ChannelConfig.Version = transfertypes.Version

	s.coordinator.Setup(s.path)

	// Pre-compute some infos for convenience
	s.chainBIBCDenom = transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(
		transfertypes.PortID,
		s.path.EndpointB.ChannelID,
		nativeDenom,
	)).IBCDenom()
	s.chainAAddr = s.alloraChainA.SenderAccount.GetAddress()
	s.chainBAddr = s.alloraChainB.SenderAccount.GetAddress()

	// ensure genesis balances are as expected
	s.assertChainABalance(s.chainAAddr, nativeDenom, genesisWalletAmount)
	s.assertChainBBalance(s.chainBAddr, nativeDenom, genesisWalletAmount)
}

func (s *IBCTestSuite) TestIBCTransfer() {
	// A => B
	s.ibcTransfer(
		s.path,
		s.path.EndpointA,
		s.chainAAddr,
		s.chainBAddr,
		nativeDenom,
		ibcTransferAmount,
	)

	s.assertChainABalance(s.chainAAddr, nativeDenom, genesisWalletAmount.Sub(ibcTransferAmount))
	s.assertChainBBalance(s.chainBAddr, s.chainBIBCDenom, ibcTransferAmount)

	// B => A
	s.ibcTransfer(
		s.path,
		s.path.EndpointB,
		s.chainBAddr,
		s.chainAAddr,
		s.chainBIBCDenom,
		ibcTransferAmount,
	)

	s.assertChainBBalance(s.chainBAddr, s.chainBIBCDenom, math.NewInt(0))
	s.assertChainABalance(s.chainAAddr, nativeDenom, genesisWalletAmount)
}

func (s *IBCTestSuite) ibcTransfer(
	path *ibctesting.Path,
	sourceEndpoint *ibctesting.Endpoint,
	fromAddr sdk.AccAddress,
	toAddr sdk.AccAddress,
	transferDenom string,
	transferAmount math.Int,
) {
	timeoutHeight := clienttypes.NewHeight(1, 110)

	// Create Transfer Msg
	transferMsg := transfertypes.NewMsgTransfer(sourceEndpoint.ChannelConfig.PortID,
		sourceEndpoint.ChannelID,
		sdk.NewCoin(transferDenom, transferAmount),
		fromAddr.String(),
		toAddr.String(),
		timeoutHeight,
		0,
		"",
	)

	// Send message from src chain
	res, err := sourceEndpoint.Chain.SendMsgs(transferMsg)
	s.Require().NoError(err)

	// Relay transfer msg to dst chain
	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	s.Require().NoError(err)

	//nolint:errcheck // this will return an error for multi-hop routes; that's expected
	path.RelayPacket(packet)
}

func (s *IBCTestSuite) assertChainABalance(addr sdk.AccAddress, denom string, amount math.Int) {
	s.assertBalance(s.alloraChainA, addr, denom, amount)
}

func (s *IBCTestSuite) assertChainBBalance(addr sdk.AccAddress, denom string, amount math.Int) {
	s.assertBalance(s.alloraChainB, addr, denom, amount)
}

func (s *IBCTestSuite) assertBalance(
	chain *ibctesting.TestChain,
	addr sdk.AccAddress,
	denom string,
	expectedAmt math.Int,
) {
	alloraApp, _ := chain.App.(*app.AlloraApp)
	actualAmt := s.getBalance(alloraApp.BankKeeper, chain, addr, denom).Amount
	s.Equal(expectedAmt, actualAmt, "Expected amount of %s: %s; Got: %s", denom, expectedAmt, actualAmt)
}

func (s *IBCTestSuite) getBalance(
	bk bankkeeper.Keeper,
	chain *ibctesting.TestChain,
	addr sdk.AccAddress,
	denom string,
) sdk.Coin {
	ctx := chain.GetContext()
	return bk.GetBalance(ctx, addr, denom)
}
