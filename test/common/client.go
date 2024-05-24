package testcommon

import (
	"context"
	"testing"

	"github.com/allora-network/allora-chain/app/params"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	"github.com/stretchr/testify/require"
)

type RpcConnectionType = uint8

const (
	SingleRpc RpcConnectionType = iota
	RoundRobin
	RandomBasedOnDeterministicSeed
)

// where to get ahold of a node
type Client struct {
	RpcConnectionType  RpcConnectionType               // what kind of rpc connection to use
	Clients            []cosmosclient.Client           // clients to attach to
	QueryAuths         []authtypes.QueryClient         // query clients for auth
	QueryBanks         []banktypes.QueryClient         // query clients for bank
	QueryDistributions []distributiontypes.QueryClient // query clients for distribution
	QueryEmissionses   []emissionstypes.QueryClient    // query clients for emissions
	QueryMints         []minttypes.QueryClient         // query clients for mint
}

// create a new appchain client that we can use
func NewClient(
	t *testing.T,
	rpcConnectionType RpcConnectionType,
	nodeRpcAddresses []string,
	alloraHomeDir string,
) Client {
	client := Client{}
	ctx := context.Background()
	client.RpcConnectionType = rpcConnectionType
	client.Clients = make([]cosmosclient.Client, len(nodeRpcAddresses))
	client.QueryAuths = make([]authtypes.QueryClient, len(nodeRpcAddresses))
	client.QueryBanks = make([]banktypes.QueryClient, len(nodeRpcAddresses))
	client.QueryDistributions = make([]distributiontypes.QueryClient, len(nodeRpcAddresses))
	client.QueryEmissionses = make([]emissionstypes.QueryClient, len(nodeRpcAddresses))
	client.QueryMints = make([]minttypes.QueryClient, len(nodeRpcAddresses))

	for i, rpcAddress := range nodeRpcAddresses {
		cosmosClient, err := cosmosclient.New(
			ctx,
			cosmosclient.WithNodeAddress(rpcAddress),
			cosmosclient.WithAddressPrefix(params.HumanCoinUnit),
			cosmosclient.WithHome(alloraHomeDir),
			cosmosclient.WithGas("auto"),
			cosmosclient.WithGasAdjustment(1.2),
		)
		require.NoError(t, err)
		ccCtx := cosmosClient.Context()

		client.Clients[i] = cosmosClient
		client.QueryAuths[i] = authtypes.NewQueryClient(ccCtx)
		client.QueryBanks[i] = banktypes.NewQueryClient(ccCtx)
		client.QueryDistributions[i] = distributiontypes.NewQueryClient(ccCtx)
		client.QueryEmissionses[i] = emissionstypes.NewQueryClient(ccCtx)
		client.QueryMints[i] = minttypes.NewQueryClient(ccCtx)

		// this is terrible, no isConnected as part of this code path
		require.NotEqual(t, ccCtx.ChainID, "")
	}
	return client
}

/// Accessors for Query Clients

func (c Client) QueryAuth() authtypes.QueryClient {
}

func (c Client) QueryBank() banktypes.QueryClient {
}

func (c Client) QueryDistribution() distributiontypes.QueryClient {
}

func (c Client) QueryEmissions() emissionstypes.QueryClient {
}

func (c Client) QueryMint() minttypes.QueryClient {

}

/// Wrappers for cosmosclient functions

func (c Client) BroadcastTx(
	ctx context.Context,
	account cosmosaccount.Account,
	msgs ...sdktypes.Msg,
) (cosmosclient.Response, error) {
}

func (c Client) Context() sdkclient.Context {

}

func (c Client) WaitForNextBlock(ctx context.Context) error {
}

func (c Client) WaitForTx(ctx context.Context, hash string) (*coretypes.ResultTx, error) {

}

// For the account client functions,
// because they are have to do with
// being able to take actions on behalf of different
// private keys, for the sake of the round robin code
// we just always make sure every node has a copy of every
// private key account.

func (c Client) AccountRegistryCreate(name string) (
	acc cosmosaccount.Account,
	mnemonic string,
	err error,
) {
}

func (c Client) AccountRegistryGetByName(name string) (
	cosmosaccount.Account,
	error,
) {
}
