package chain_test

import (
	"context"
	"testing"

	"github.com/allora-network/allora-chain/app/params"
	emissions "github.com/allora-network/allora-chain/x/emissions/module"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	auth "github.com/cosmos/cosmos-sdk/x/auth"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distribution "github.com/cosmos/cosmos-sdk/x/distribution"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	"github.com/stretchr/testify/require"
)

// where to get ahold of a node
type NodeConfig struct {
	NodeRPCAddress string // rpc node to attach to
	AlloraHomeDir  string // home directory for the allora keystore
}

// handle to various node data
type Node struct {
	NodeClient        NodeConfig
	Client            cosmosclient.Client
	QueryEmissions    emissionstypes.QueryClient
	QueryAuth         authtypes.QueryClient
	QueryDistribution distributiontypes.QueryClient
	QueryBank         banktypes.QueryClient
	QueryMint         minttypes.QueryClient
	FaucetAcc         cosmosaccount.Account
	FaucetAddr        string
	UpshotAcc         cosmosaccount.Account
	UpshotAddr        string
	Cdc               codec.Codec
}

// create a new appchain client that we can use
func NewNode(t *testing.T, nc NodeConfig) (Node, error) {
	node := Node{NodeClient: nc}
	var err error

	// create a allora client instance
	ctx := context.Background()

	node.Client, err = cosmosclient.New(
		ctx,
		cosmosclient.WithNodeAddress(nc.NodeRPCAddress),
		cosmosclient.WithAddressPrefix(params.HumanCoinUnit),
		cosmosclient.WithHome(nc.AlloraHomeDir),
	)
	require.NoError(t, err)

	//// restore from mnemonic
	node.FaucetAcc, err = node.Client.AccountRegistry.GetByName("faucet")
	require.NoError(t, err)
	node.UpshotAcc, err = node.Client.AccountRegistry.GetByName("upshot")
	require.NoError(t, err)
	node.FaucetAddr, err = node.FaucetAcc.Address(params.HumanCoinUnit)
	require.NoError(t, err)
	node.UpshotAddr, err = node.UpshotAcc.Address(params.HumanCoinUnit)
	require.NoError(t, err)

	// Create query client
	node.QueryEmissions = emissionstypes.NewQueryClient(node.Client.Context())
	node.QueryAuth = authtypes.NewQueryClient(node.Client.Context())
	node.QueryDistribution = distributiontypes.NewQueryClient(node.Client.Context())
	node.QueryBank = banktypes.NewQueryClient(node.Client.Context())
	node.QueryMint = minttypes.NewQueryClient(node.Client.Context())

	encCfg := moduletestutil.MakeTestEncodingConfig(
		mint.AppModuleBasic{},
		emissions.AppModule{},
		auth.AppModule{},
		bank.AppModule{},
		distribution.AppModule{},
	)
	node.Cdc = codec.NewProtoCodec(encCfg.InterfaceRegistry)

	// this is terrible, no isConnected as part of this code path
	require.NotEqual(t, node.Client.Context().ChainID, "")
	return node, nil
}
