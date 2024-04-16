package chain_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/allora-network/allora-chain/app/params"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	"github.com/stretchr/testify/require"
)

type NodeConfig struct {
	t                        *testing.T
	NodeRPCAddress           string // rpc node to attach to
	AddressKeyName           string // load a address by key from the keystore
	AddressRestoreMnemonic   string
	AddressAccountPassphrase string
	AlloraHomeDir            string // home directory for the allora keystore
	StringSeperator          string // string seperator used for key identifiers in allora
	ReconnectSeconds         uint64 // seconds to wait for reconnection
}

type Node struct {
	NodeClient  NodeConfig
	Client      *cosmosclient.Client
	QueryClient emissionstypes.QueryClient
	Accounts    []cosmosaccount.Account
}

func NewNodeConfig(
	t *testing.T,
	nodeRPCAddress,
	addressKeyName,
	addressRestoreMnemonic,
	addressAccountPassphrase,
	alloraHomeDir,
	stringSeperator string,
	reconnectSeconds uint64,
) NodeConfig {
	return NodeConfig{
		t:                        t,
		NodeRPCAddress:           nodeRPCAddress,
		AddressKeyName:           addressKeyName,
		AddressRestoreMnemonic:   addressRestoreMnemonic,
		AddressAccountPassphrase: addressAccountPassphrase,
		AlloraHomeDir:            alloraHomeDir,
		StringSeperator:          stringSeperator,
		ReconnectSeconds:         reconnectSeconds,
	}
}

func getAlloraClient(nc NodeConfig) (*cosmosclient.Client, error) {
	// create a allora client instance
	ctx := context.Background()
	userHomeDir, _ := os.UserHomeDir()
	alloraClientHome := filepath.Join(userHomeDir, ".allorad")
	if nc.AlloraHomeDir != "" {
		alloraClientHome = nc.AlloraHomeDir
	}

	client, err := cosmosclient.New(
		ctx,
		cosmosclient.WithNodeAddress(nc.NodeRPCAddress),
		cosmosclient.WithAddressPrefix(params.HumanCoinUnit),
		cosmosclient.WithHome(alloraClientHome),
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

	//var accounts []cosmosaccount.Account = make([]cosmosaccount.Account, 1)

	//// restore from mneumonic
	//accounts[0], err = node.Client.AccountRegistry.Import(
	//	nc.AddressKeyName,
	//	nc.AddressRestoreMnemonic,
	//	nc.AddressAccountPassphrase,
	//)
	//require.NoError(nc.t, err)
	//node.Accounts = accounts

	// Create query client
	node.QueryClient = emissionstypes.NewQueryClient(node.Client.Context())

	// this is terrible, no isConnected as part of this code path
	require.NotEqual(nc.t, node.Client.Context().ChainID, "")
	return node, nil
}
