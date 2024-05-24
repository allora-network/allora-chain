package testcommon

import (
	"context"
	"math/rand"
	"sync"
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
	RpcCounterSeed     int64                           // if round-robin which RPC to use next, if random, seed to use
	RpcCounterMutex    *sync.Mutex                     // mutex for the counter
	Rand               *rand.Rand                      // random number generator
}

// create a new appchain client that we can use
func NewClient(
	t *testing.T,
	rpcConnectionType RpcConnectionType,
	nodeRpcAddresses []string,
	alloraHomeDir string,
	seed int64,
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
	client.RpcCounterMutex = &sync.Mutex{}
	client.RpcCounterSeed = 0
	if rpcConnectionType == RandomBasedOnDeterministicSeed {
		client.RpcCounterSeed = seed
		client.Rand = rand.New(rand.NewSource(seed))
	}

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

func (c *Client) getNextClientNumber() int64 {
	if c.RpcConnectionType == SingleRpc {
		return 0
	} else if c.RpcConnectionType == RoundRobin {
		c.RpcCounterMutex.Lock()
		ret := c.RpcCounterSeed
		c.RpcCounterSeed = (c.RpcCounterSeed + 1) % int64(len(c.Clients))
		c.RpcCounterMutex.Unlock()
		return ret
	} else { //if c.RpcConnectionType == RandomBasedOnDeterministicSeed {
		c.RpcCounterMutex.Lock()
		ret := int64(c.Rand.Intn(len(c.Clients)))
		c.RpcCounterMutex.Unlock()
		return ret
	}
}

// / Accessors for Query Clients.
// These don't have to be concurrency aware
// because they are read only, and the RPC endpoint should
// handle concurrency.
func (c *Client) QueryAuth() authtypes.QueryClient {
	return c.QueryAuths[c.getNextClientNumber()]
}

func (c *Client) QueryBank() banktypes.QueryClient {
	return c.QueryBanks[c.getNextClientNumber()]
}

func (c *Client) QueryDistribution() distributiontypes.QueryClient {
	return c.QueryDistributions[c.getNextClientNumber()]
}

func (c *Client) QueryEmissions() emissionstypes.QueryClient {
	return c.QueryEmissionses[c.getNextClientNumber()]
}

func (c *Client) QueryMint() minttypes.QueryClient {
	return c.QueryMints[c.getNextClientNumber()]
}

/// Wrappers for cosmosclient functions
// broadcast etc shouldn't have to worry about concurrency
// because the RPC endpoint itself should handle that.

func (c *Client) BroadcastTx(
	ctx context.Context,
	account cosmosaccount.Account,
	msgs ...sdktypes.Msg,
) (cosmosclient.Response, error) {
	return c.Clients[c.getNextClientNumber()].BroadcastTx(ctx, account, msgs...)
}

func (c *Client) Context() sdkclient.Context {
	return c.Clients[c.getNextClientNumber()].Context()
}

func (c *Client) WaitForNextBlock(ctx context.Context) error {
	return c.Clients[c.getNextClientNumber()].WaitForNextBlock(ctx)
}

func (c *Client) WaitForTx(ctx context.Context, hash string) (*coretypes.ResultTx, error) {
	return c.Clients[c.getNextClientNumber()].WaitForTx(ctx, hash)
}

// For the account client functions,
// because they are have to do with
// being able to take actions on behalf of different
// private keys, for the sake of the round robin code
// we just always make sure every node has a copy of every
// private key account.

func (c *Client) AccountRegistryCreate(name string) (
	acc cosmosaccount.Account,
	mnemonic string,
	err error,
) {
	for _, client := range c.Clients {
		acc, mnemonic, err = client.AccountRegistry.Create(name)
		if err != nil {
			return acc, mnemonic, err
		}
	}
	return acc, mnemonic, err
}

func (c *Client) AccountRegistryGetByName(name string) (
	cosmosaccount.Account,
	error,
) {
	return c.Clients[c.getNextClientNumber()].AccountRegistry.GetByName(name)
}
