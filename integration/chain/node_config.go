package chain_test

import (
	"context"
	"testing"

	"github.com/allora-network/allora-chain/app/params"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
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
	NodeClient  NodeConfig
	Client      cosmosclient.Client
	QueryClient emissionstypes.QueryClient
	AliceAcc    cosmosaccount.Account
	BobAcc      cosmosaccount.Account
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

	//// restore from mneumonic
	node.AliceAcc, err = node.Client.AccountRegistry.GetByName("alice")
	require.NoError(t, err)
	node.BobAcc, err = node.Client.AccountRegistry.GetByName("bob")
	require.NoError(t, err)

	// Create query client
	node.QueryClient = emissionstypes.NewQueryClient(node.Client.Context())

	// this is terrible, no isConnected as part of this code path
	require.NotEqual(t, node.Client.Context().ChainID, "")
	return node, nil
}
