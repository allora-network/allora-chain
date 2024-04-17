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

type NodeConfig struct {
	t              *testing.T
	NodeRPCAddress string // rpc node to attach to
	AlloraHomeDir  string // home directory for the allora keystore
}

type Node struct {
	NodeClient  NodeConfig
	Client      *cosmosclient.Client
	QueryClient emissionstypes.QueryClient
	AliceAcc    cosmosaccount.Account
	BobAcc      cosmosaccount.Account
}

func NewNodeConfig(
	t *testing.T,
	nodeRPCAddress,
	alloraHomeDir string,
) NodeConfig {
	return NodeConfig{
		t:              t,
		NodeRPCAddress: nodeRPCAddress,
		AlloraHomeDir:  alloraHomeDir,
	}
}

func getAlloraClient(nc NodeConfig) (*cosmosclient.Client, error) {
	// create a allora client instance
	ctx := context.Background()

	client, err := cosmosclient.New(
		ctx,
		cosmosclient.WithNodeAddress(nc.NodeRPCAddress),
		cosmosclient.WithAddressPrefix(params.HumanCoinUnit),
		cosmosclient.WithHome(nc.AlloraHomeDir),
	)
	require.NoError(nc.t, err)
	return &client, nil
}

// create a new appchain client that we can use
func NewNode(nc NodeConfig) (Node, error) {
	node := Node{NodeClient: nc}
	var err error
	node.Client, err = getAlloraClient(nc)
	if err != nil {
		return Node{}, err
	}

	//// restore from mneumonic
	node.AliceAcc, err = node.Client.AccountRegistry.GetByName("alice")
	require.NoError(nc.t, err)
	node.BobAcc, err = node.Client.AccountRegistry.GetByName("bob")
	require.NoError(nc.t, err)

	// Create query client
	node.QueryClient = emissionstypes.NewQueryClient(node.Client.Context())

	// this is terrible, no isConnected as part of this code path
	require.NotEqual(nc.t, node.Client.Context().ChainID, "")
	return node, nil
}
