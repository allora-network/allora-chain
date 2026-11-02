package testcommon

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/allora-network/allora-chain/app/params"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	minttypes "github.com/allora-network/allora-chain/x/mint/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
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

const (
	defaultNodeStallTimeout   = 2 * time.Minute
	defaultNodeRPCTimeout     = 10 * time.Second
	defaultCommitPollInterval = 250 * time.Millisecond
)

// where to get ahold of a node
type Client struct {
	RpcConnectionType  RpcConnectionType                   // what kind of rpc connection to use
	Clients            []cosmosclient.Client               // clients to attach to
	QueryAuths         []authtypes.QueryClient             // query clients for auth
	QueryBanks         []banktypes.QueryClient             // query clients for bank
	QueryDistributions []distributiontypes.QueryClient     // query clients for distribution
	QueryEmissionses   []emissionstypes.QueryServiceClient // query clients for emissions
	QueryMints         []minttypes.QueryServiceClient      // query clients for mint
	QueryGovs          []govtypesv1.QueryClient            // query clients for gov
	QueryUpgrades      []upgradetypes.QueryClient          // query clients for upgrades
	RpcCounterSeed     int64                               // if round-robin which RPC to use next, if random, seed to use
	RpcCounterMutex    *sync.Mutex                         // mutex for the counter
	Rand               *rand.Rand                          // random number generator

	accountRegistry      cosmosaccount.Registry // registry for accounts
	accountRegistryMutex *sync.Mutex            // mutex for the registry

	nodeAddresses      []string      // rpc address of each entry in Clients
	nodeStallTimeout   time.Duration // bound on waiting for a tx, or for a lagging node to advance
	nodeRPCTimeout     time.Duration // bound on one /abci_info call
	commitPollInterval time.Duration // delay between /abci_info polls
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
	queryMints := make([]minttypes.QueryServiceClient, len(nodeRpcAddresses))
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
			cosmosclient.WithFees("10000000uallo"),
		)
		require.NoError(t, err)
		ccCtx := cosmosClient.Context()

		clients[i] = cosmosClient
		queryAuths[i] = authtypes.NewQueryClient(ccCtx)
		queryBanks[i] = banktypes.NewQueryClient(ccCtx)
		queryDistributions[i] = distributiontypes.NewQueryClient(ccCtx)
		queryEmissionses[i] = emissionstypes.NewQueryServiceClient(ccCtx)
		queryMints[i] = minttypes.NewQueryServiceClient(ccCtx)
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
		nodeAddresses:        nodeRpcAddresses,
		nodeStallTimeout:     defaultNodeStallTimeout,
		nodeRPCTimeout:       defaultNodeRPCTimeout,
		commitPollInterval:   defaultCommitPollInterval,
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

func (c *Client) QueryMint() minttypes.QueryServiceClient {
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

// BroadcastTx sends msgs through the next node, then waits until every node's
// application has committed the tx's height, so any node can serve the next read.
func (c *Client) BroadcastTx(
	ctx context.Context,
	account cosmosaccount.Account,
	msgs ...sdktypes.Msg,
) (cosmosclient.Response, error) {
	txCtx, cancel := context.WithTimeout(ctx, c.nodeStallTimeout)
	defer cancel()
	resp, err := c.Clients[c.getNextClientNumber()].BroadcastTx(txCtx, account, msgs...)
	if resp.TxResponse == nil {
		return resp, c.explainDeadline(ctx, err)
	}
	// A tx that fails in execution is still committed and still advances the
	// sender's sequence, so the wait applies to it too.
	return resp, errors.Join(err, c.waitForCommittedHeight(ctx, resp.Height))
}

func (c *Client) Context() sdkclient.Context {
	return c.Clients[c.getNextClientNumber()].Context()
}

// WaitForNextBlock waits until every node's application has committed the block
// after the highest height any node has committed.
func (c *Client) WaitForNextBlock(ctx context.Context) error {
	nodes := c.snapshotCommits(ctx)
	var highest int64
	answered := false
	for i := range nodes {
		answered = answered || nodes[i].err == nil
		highest = max(highest, nodes[i].height)
	}
	if !answered {
		return fmt.Errorf("no node answered /abci_info: %s", c.commitReport(nodes))
	}
	return c.waitForCommittedHeight(ctx, highest+1)
}

// WaitForBlockHeight waits until every node's application has committed height.
func (c *Client) WaitForBlockHeight(ctx context.Context, height int64) error {
	return c.waitForCommittedHeight(ctx, height)
}

// WaitForTx waits for the tx to be indexed on the next node, for at most
// nodeStallTimeout.
func (c *Client) WaitForTx(ctx context.Context, hash string) (*coretypes.ResultTx, error) {
	txCtx, cancel := context.WithTimeout(ctx, c.nodeStallTimeout)
	defer cancel()
	res, err := c.Clients[c.getNextClientNumber()].WaitForTx(txCtx, hash)
	return res, c.explainDeadline(ctx, err)
}

// nodeCommit is what a node's application last reported through /abci_info.
type nodeCommit struct {
	height       int64
	err          error
	lastProgress time.Time
}

// waitForCommittedHeight blocks until every node's application has committed
// height. It reads /abci_info because /status reports the block store, which
// cometbft saves before the application executes and commits the block. The
// wait has no overall deadline, since height can be many blocks away; it fails
// when a node still below height has not advanced for nodeStallTimeout.
func (c *Client) waitForCommittedHeight(ctx context.Context, height int64) error {
	now := time.Now()
	nodes := make([]nodeCommit, len(c.Clients))
	for i := range nodes {
		nodes[i].lastProgress = now
	}
	ticker := time.NewTicker(c.commitPollInterval)
	defer ticker.Stop()

	for {
		lagging, stalled := false, false
		for i := range nodes {
			if nodes[i].height >= height {
				continue
			}
			c.pollCommit(ctx, i, &nodes[i])
			if nodes[i].height < height {
				lagging = true
				stalled = stalled || time.Since(nodes[i].lastProgress) >= c.nodeStallTimeout
			}
		}
		if !lagging {
			return nil
		}
		if stalled {
			return fmt.Errorf(
				"waiting for every node to commit height %d: no progress for %s: %s",
				height, c.nodeStallTimeout, c.commitReport(nodes),
			)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"waiting for every node to commit height %d: %w: %s",
				height, ctx.Err(), c.commitReport(nodes),
			)
		case <-ticker.C:
		}
	}
}

// pollCommit refreshes one node's committed height. Failures stay in n.err and
// leave the wait running, because a node may be restarting.
func (c *Client) pollCommit(ctx context.Context, index int, n *nodeCommit) {
	callCtx, cancel := context.WithTimeout(ctx, c.nodeRPCTimeout)
	defer cancel()
	info, err := c.Clients[index].RPC.ABCIInfo(callCtx)
	n.err = err
	if err != nil {
		return
	}
	if h := info.Response.LastBlockHeight; h > n.height {
		n.height = h
		n.lastProgress = time.Now()
	}
}

// snapshotCommits polls every node once.
func (c *Client) snapshotCommits(ctx context.Context) []nodeCommit {
	nodes := make([]nodeCommit, len(c.Clients))
	for i := range nodes {
		c.pollCommit(ctx, i, &nodes[i])
	}
	return nodes
}

// commitReport names each node with its committed height and last poll error.
func (c *Client) commitReport(nodes []nodeCommit) string {
	parts := make([]string, len(nodes))
	for i := range nodes {
		parts[i] = fmt.Sprintf("node %d (%s) committed height %d", i, c.nodeAddresses[i], nodes[i].height)
		if nodes[i].err != nil {
			parts[i] += fmt.Sprintf(" (last poll failed: %v)", nodes[i].err)
		}
	}
	return strings.Join(parts, "; ")
}

// explainDeadline adds each node's committed height to err when the client's own
// bound expired while the caller's ctx is still live.
func (c *Client) explainDeadline(ctx context.Context, err error) error {
	if ctx.Err() != nil || !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: %s", err, c.commitReport(c.snapshotCommits(ctx)))
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
