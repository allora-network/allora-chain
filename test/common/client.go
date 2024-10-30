package testcommon

import (
	"context"
	"math/rand"
	"sync"
	"testing"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/allora-network/allora-chain/app/params"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
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
	RpcConnectionType  RpcConnectionType                   // what kind of rpc connection to use
	Clients            []cosmosclient.Client               // clients to attach to
	QueryAuths         []authtypes.QueryClient             // query clients for auth
	QueryBanks         []banktypes.QueryClient             // query clients for bank
	QueryDistributions []distributiontypes.QueryClient     // query clients for distribution
	QueryEmissionses   []emissionstypes.QueryServiceClient // query clients for emissions
	QueryMints         []minttypes.QueryClient             // query clients for mint
	QueryGovs          []govtypesv1.QueryClient            // query clients for gov
	QueryUpgrades      []upgradetypes.QueryClient          // query clients for upgrades
	RpcCounterSeed     int64                               // if round-robin which RPC to use next, if random, seed to use
	RpcCounterMutex    *sync.Mutex                         // mutex for the counter
	Rand               *rand.Rand                          // random number generator

	accountRegistry      cosmosaccount.Registry // registry for accounts
	accountRegistryMutex *sync.Mutex            // mutex for the registry
}

// create a new appchain client that we can use
func NewClient(
	t *testing.T,
	rpcConnectionType RpcConnectionType,
	nodeRpcAddresses []string,
	alloraHomeDir string,
	seed int64,
) Client {
	t.Helper()
	ctx := context.Background()

	clients := make([]cosmosclient.Client, len(nodeRpcAddresses))
	queryAuths := make([]authtypes.QueryClient, len(nodeRpcAddresses))
	queryBanks := make([]banktypes.QueryClient, len(nodeRpcAddresses))
	queryDistributions := make([]distributiontypes.QueryClient, len(nodeRpcAddresses))
	queryEmissionses := make([]emissionstypes.QueryServiceClient, len(nodeRpcAddresses))
	queryMints := make([]minttypes.QueryClient, len(nodeRpcAddresses))
	queryGovs := make([]govtypesv1.QueryClient, len(nodeRpcAddresses))
	queryUpgrades := make([]upgradetypes.QueryClient, len(nodeRpcAddresses))

	rpcCounterSeed := int64(0)
	if rpcConnectionType == RandomBasedOnDeterministicSeed {
		rpcCounterSeed = seed
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

		clients[i] = cosmosClient
		queryAuths[i] = authtypes.NewQueryClient(ccCtx)
		queryBanks[i] = banktypes.NewQueryClient(ccCtx)
		queryDistributions[i] = distributiontypes.NewQueryClient(ccCtx)
		queryEmissionses[i] = emissionstypes.NewQueryServiceClient(ccCtx)
		queryMints[i] = minttypes.NewQueryClient(ccCtx)
		queryGovs[i] = govtypesv1.NewQueryClient(ccCtx)
		queryUpgrades[i] = upgradetypes.NewQueryClient(ccCtx)

		// this is terrible, no isConnected as part of this code path
		require.NotEqual(t, "", ccCtx.ChainID)
	}

	accountRegistry, err := cosmosaccount.New(
		cosmosaccount.WithKeyringServiceName(sdktypes.KeyringServiceName()),
		cosmosaccount.WithKeyringBackend(cosmosaccount.KeyringTest),
		cosmosaccount.WithHome(alloraHomeDir),
	)
	require.NoError(t, err)

	return Client{
		RpcConnectionType:    rpcConnectionType,
		Clients:              clients,
		QueryAuths:           queryAuths,
		QueryBanks:           queryBanks,
		QueryDistributions:   queryDistributions,
		QueryEmissionses:     queryEmissionses,
		QueryMints:           queryMints,
		QueryGovs:            queryGovs,
		QueryUpgrades:        queryUpgrades,
		RpcCounterMutex:      &sync.Mutex{},
		RpcCounterSeed:       rpcCounterSeed,
		accountRegistry:      accountRegistry,
		accountRegistryMutex: &sync.Mutex{},
		Rand:                 rand.New(rand.NewSource(seed)), //nolint:gosec // G404: Use of weak random number generator (math/rand or math/rand/v2 instead of crypto/rand)
	}
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

func (c *Client) QueryEmissions() emissionstypes.QueryServiceClient {
	return c.QueryEmissionses[c.getNextClientNumber()]
}

func (c *Client) QueryMint() minttypes.QueryClient {
	return c.QueryMints[c.getNextClientNumber()]
}

func (c *Client) QueryGov() govtypesv1.QueryClient {
	return c.QueryGovs[c.getNextClientNumber()]
}

func (c *Client) QueryUpgrade() upgradetypes.QueryClient {
	return c.QueryUpgrades[c.getNextClientNumber()]
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

func (c *Client) BroadcastTxAsync(
	ctx context.Context,
	account cosmosaccount.Account,
	msgs ...sdktypes.Msg,
) (cosmosclient.Response, error) {
	client := c.Clients[c.getNextClientNumber()]
	sdkCtx := client.Context().WithBroadcastMode(tx.BroadcastMode_BROADCAST_MODE_ASYNC.String())
	return client.BroadcastTx(sdkCtx.CmdContext, account, msgs...)
}

func (c *Client) Context() sdkclient.Context {
	return c.Clients[c.getNextClientNumber()].Context()
}

func (c *Client) WaitForNextBlock(ctx context.Context) error {
	return c.Clients[c.getNextClientNumber()].WaitForNextBlock(ctx)
}

func (c *Client) WaitForBlockHeight(ctx context.Context, height int64) error {
	return c.Clients[c.getNextClientNumber()].WaitForBlockHeight(ctx, height)
}

func (c *Client) WaitForTx(ctx context.Context, hash string) (*coretypes.ResultTx, error) {
	return c.Clients[c.getNextClientNumber()].WaitForTx(ctx, hash)
}

func (c *Client) BlockHeight(ctx context.Context) (int64, error) {
	return c.Clients[c.getNextClientNumber()].LatestBlockHeight(ctx)
}

// account code has to be concurrency aware

func (c *Client) AccountRegistryCreate(name string) (
	acc cosmosaccount.Account,
	mnemonic string,
	err error,
) {
	c.accountRegistryMutex.Lock()
	acc, mnemonic, err = c.accountRegistry.Create(name)
	c.accountRegistryMutex.Unlock()
	return acc, mnemonic, err
}

func (c *Client) AccountRegistryGetByName(name string) (
	cosmosaccount.Account,
	error,
) {
	c.accountRegistryMutex.Lock()
	acc, err := c.accountRegistry.GetByName(name)
	c.accountRegistryMutex.Unlock()
	return acc, err
}
