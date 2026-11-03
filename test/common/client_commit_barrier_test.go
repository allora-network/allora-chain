package testcommon

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/p2p"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	cmttypes "github.com/cometbft/cometbft/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

const barrierTestDeadline = 30 * time.Second

// fakeCometNode is a local CometBFT JSON-RPC endpoint with independent block
// store and application heights, the two values a real node exposes separately
// through /status and /abci_info between saving a block and committing it. The
// production rpc http client talks to it over real HTTP.
type fakeCometNode struct {
	server *httptest.Server

	blockStoreHeight atomic.Int64
	appHeight        atomic.Int64
	advancePerPoll   atomic.Int64 // added to both heights on each /abci_info call
	abciInfoPolls    atomic.Int64
	failABCIInfo     atomic.Bool  // /abci_info answers with HTTP 500
	hangABCIInfo     atomic.Bool  // /abci_info answers only when the caller gives up
	txHeight         atomic.Int64 // height /tx reports; zero means the tx is not found
	txCode           atomic.Uint32
}

func newFakeCometNode(t *testing.T, blockStoreHeight, appHeight int64) *fakeCometNode {
	t.Helper()
	n := new(fakeCometNode)
	n.blockStoreHeight.Store(blockStoreHeight)
	n.appHeight.Store(appHeight)
	n.server = httptest.NewServer(http.HandlerFunc(n.serve))
	t.Cleanup(n.server.Close)
	return n
}

// commit moves both heights to height, as a real node does on finishing a block.
func (n *fakeCometNode) commit(height int64) {
	n.blockStoreHeight.Store(height)
	n.appHeight.Store(height)
}

func (n *fakeCometNode) serve(w http.ResponseWriter, r *http.Request) {
	var req rpctypes.RPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Method == "abci_info" {
		n.abciInfoPolls.Add(1)
		if n.failABCIInfo.Load() {
			http.Error(w, "node restarting", http.StatusInternalServerError)
			return
		}
		if n.hangABCIInfo.Load() {
			<-r.Context().Done()
			return
		}
	}
	result, rpcErr := n.result(req)
	resp := rpctypes.NewRPCSuccessResponse(req.ID, result)
	if rpcErr != "" {
		resp = rpctypes.NewRPCErrorResponse(req.ID, -32603, "Internal error", rpcErr)
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		panic(err)
	}
}

func (n *fakeCometNode) result(req rpctypes.RPCRequest) (any, string) {
	switch req.Method {
	case "status":
		return &coretypes.ResultStatus{ //nolint:exhaustruct // the client reads network and height only
			NodeInfo: p2p.DefaultNodeInfo{Network: "barrier-test"},                     //nolint:exhaustruct // see above
			SyncInfo: coretypes.SyncInfo{LatestBlockHeight: n.blockStoreHeight.Load()}, //nolint:exhaustruct // see above
		}, ""
	case "abci_info":
		height := n.appHeight.Load()
		if step := n.advancePerPoll.Load(); step > 0 {
			height = n.appHeight.Add(step)
			n.blockStoreHeight.Store(height)
		}
		return &coretypes.ResultABCIInfo{
			Response: abci.ResponseInfo{LastBlockHeight: height}, //nolint:exhaustruct // the barrier reads height only
		}, ""
	case "abci_query":
		return n.abciQuery(req)
	case "broadcast_tx_sync":
		return &coretypes.ResultBroadcastTx{ //nolint:exhaustruct // a code of zero means CheckTx accepted the tx
			Hash: cmttypes.Tx("tx").Hash(),
		}, ""
	case "tx":
		height := n.txHeight.Load()
		if height == 0 {
			return nil, "tx not found"
		}
		return &coretypes.ResultTx{ //nolint:exhaustruct // the client reads hash, height and result
			Hash:   cmttypes.Tx("tx").Hash(),
			Height: height,
			TxResult: abci.ExecTxResult{ //nolint:exhaustruct // see above
				Code: n.txCode.Load(),
				Log:  "execution failed",
			},
		}, ""
	}
	return nil, "unexpected method " + req.Method
}

// abciQuery answers the account lookup and gas simulation that precede a
// broadcast, with the same protobuf responses a node returns.
func (n *fakeCometNode) abciQuery(req rpctypes.RPCRequest) (any, string) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, err.Error()
	}
	var value []byte
	var err error
	switch params.Path {
	case "/cosmos.auth.v1beta1.Query/Account":
		var account *codectypes.Any
		account, err = codectypes.NewAnyWithValue(&authtypes.BaseAccount{ //nolint:exhaustruct // sequence and number stay zero
			Address: sdktypes.AccAddress("barrier-test-sender").String(),
		})
		if err == nil {
			value, err = (&authtypes.QueryAccountResponse{Account: account}).Marshal()
		}
	case "/cosmos.tx.v1beta1.Service/Simulate":
		value, err = (&txtypes.SimulateResponse{ //nolint:exhaustruct // CalculateGas reads gas info only
			GasInfo: &sdktypes.GasInfo{GasWanted: 100000, GasUsed: 100000},
		}).Marshal()
	default:
		return nil, "unexpected query path " + params.Path
	}
	if err != nil {
		return nil, err.Error()
	}
	return &coretypes.ResultABCIQuery{
		Response: abci.ResponseQuery{Value: value}, //nolint:exhaustruct // a code of zero means success
	}, ""
}

// newBarrierTestClient builds a real Client through NewClient against the nodes,
// with a shorter poll interval than production.
func newBarrierTestClient(t *testing.T, mode RpcConnectionType, nodes ...*fakeCometNode) *Client {
	t.Helper()
	urls := make([]string, len(nodes))
	for i, node := range nodes {
		urls[i] = node.server.URL
	}
	c := NewClient(t, mode, urls, t.TempDir(), 1)
	c.commitPollInterval = 5 * time.Millisecond
	return c
}

// runBarrier runs wait on its own goroutine and returns a channel carrying its result.
func runBarrier(wait func() error) <-chan error {
	done := make(chan error, 1)
	go func() { done <- wait() }()
	return done
}

// requireBlockedThroughPolls fails if the barrier returns before it has polled
// the laggard's /abci_info several more times while the laggard stays behind.
func requireBlockedThroughPolls(t *testing.T, laggard *fakeCometNode, done <-chan error) {
	t.Helper()
	target := laggard.abciInfoPolls.Load() + 5
	deadline := time.After(barrierTestDeadline)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for laggard.abciInfoPolls.Load() < target {
		select {
		case err := <-done:
			t.Fatalf("barrier returned while a node's application was behind, err: %v", err)
		case <-deadline:
			t.Fatal("barrier stopped polling the lagging node's /abci_info")
		case <-ticker.C:
		}
	}
}

func requireBarrierResult(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(barrierTestDeadline):
		t.Fatal("barrier did not return")
		return nil
	}
}

// The laggard's block store is already at the target, so a barrier that trusts
// /status returns too early; only its application height holds the wait open.
func TestWaitForBlockHeightWaitsForLaggardApplication(t *testing.T) {
	ahead, peer, laggard := newFakeCometNode(t, 10, 10), newFakeCometNode(t, 10, 10), newFakeCometNode(t, 10, 9)
	c := newBarrierTestClient(t, RoundRobin, ahead, peer, laggard)

	done := runBarrier(func() error { return c.WaitForBlockHeight(context.Background(), 10) })
	requireBlockedThroughPolls(t, laggard, done)

	laggard.commit(10)
	require.NoError(t, requireBarrierResult(t, done))
}

func TestWaitForNextBlockTargetsHighestNodeOnEveryNode(t *testing.T) {
	ahead, laggard := newFakeCometNode(t, 10, 10), newFakeCometNode(t, 10, 9)
	c := newBarrierTestClient(t, RoundRobin, laggard, ahead)

	done := runBarrier(func() error { return c.WaitForNextBlock(context.Background()) })
	requireBlockedThroughPolls(t, laggard, done)

	ahead.commit(11)
	laggard.commit(10)
	requireBlockedThroughPolls(t, laggard, done)

	laggard.commit(11)
	require.NoError(t, requireBarrierResult(t, done))
}

func TestBroadcastTxWaitsForEveryNodeAtTxHeight(t *testing.T) {
	for _, tc := range []struct {
		name    string
		txCode  uint32
		wantErr string
	}{
		{name: "successful tx", txCode: 0, wantErr: ""},
		{name: "tx failing in execution is committed too", txCode: 5, wantErr: "execution failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			submitter, laggard := newFakeCometNode(t, 20, 20), newFakeCometNode(t, 20, 19)
			submitter.txHeight.Store(20)
			submitter.txCode.Store(tc.txCode)
			c := newBarrierTestClient(t, RoundRobin, submitter, laggard)
			account, _, err := c.AccountRegistryCreate("alice")
			require.NoError(t, err)
			msg := &banktypes.MsgSend{ //nolint:exhaustruct // the fake node does not execute the message
				FromAddress: sdktypes.AccAddress("barrier-test-sender").String(),
				ToAddress:   sdktypes.AccAddress("barrier-test-target").String(),
			}

			done := runBarrier(func() error {
				resp, err := c.BroadcastTx(context.Background(), account, msg)
				if resp.TxResponse != nil && resp.Height != 20 {
					t.Errorf("response height = %d, want 20", resp.Height)
				}
				return err
			})
			requireBlockedThroughPolls(t, laggard, done)

			laggard.commit(20)
			err = requireBarrierResult(t, done)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}

func TestBarrierTimeoutNamesEveryNode(t *testing.T) {
	ahead, halted := newFakeCometNode(t, 10, 10), newFakeCometNode(t, 12, 9)
	c := newBarrierTestClient(t, RoundRobin, ahead, halted)
	c.nodeStallTimeout = 50 * time.Millisecond

	err := requireBarrierResult(t, runBarrier(func() error {
		return c.WaitForBlockHeight(context.Background(), 10)
	}))

	require.ErrorContains(t, err, "node 0 ("+ahead.server.URL+") committed height 10")
	require.ErrorContains(t, err, "node 1 ("+halted.server.URL+") committed height 9")
}

func TestBarrierTimesOutOnUnresponsiveNode(t *testing.T) {
	healthy, hung := newFakeCometNode(t, 10, 10), newFakeCometNode(t, 10, 10)
	hung.hangABCIInfo.Store(true)
	c := newBarrierTestClient(t, RoundRobin, healthy, hung)
	c.nodeStallTimeout = 100 * time.Millisecond
	c.nodeRPCTimeout = 20 * time.Millisecond

	err := requireBarrierResult(t, runBarrier(func() error {
		return c.WaitForBlockHeight(context.Background(), 10)
	}))

	require.ErrorContains(t, err, "node 1 ("+hung.server.URL+") committed height 0 (last poll failed:")
}

// A height many blocks away takes longer than the stall bound while every poll
// shows progress, so the wait must succeed.
func TestBarrierStallBoundIgnoresSlowProgress(t *testing.T) {
	node := newFakeCometNode(t, 1, 1)
	node.advancePerPoll.Store(1)
	c := newBarrierTestClient(t, SingleRpc, node)
	c.nodeStallTimeout = 500 * time.Millisecond
	c.commitPollInterval = 20 * time.Millisecond
	const blocksPastStallBound = 40

	start := time.Now()
	err := requireBarrierResult(t, runBarrier(func() error {
		return c.WaitForBlockHeight(context.Background(), 1+blocksPastStallBound)
	}))

	require.NoError(t, err)
	require.Greater(t, time.Since(start), c.nodeStallTimeout)
}

func TestBarrierSurvivesTransientNodeFailure(t *testing.T) {
	restarting := newFakeCometNode(t, 5, 5)
	restarting.failABCIInfo.Store(true)
	c := newBarrierTestClient(t, SingleRpc, restarting)

	done := runBarrier(func() error { return c.WaitForBlockHeight(context.Background(), 5) })
	requireBlockedThroughPolls(t, restarting, done)

	restarting.failABCIInfo.Store(false)
	require.NoError(t, requireBarrierResult(t, done))
}

func TestBarrierHonoursCallerContext(t *testing.T) {
	node := newFakeCometNode(t, 5, 5)
	c := newBarrierTestClient(t, SingleRpc, node)
	ctx, cancel := context.WithCancel(context.Background())

	done := runBarrier(func() error { return c.WaitForBlockHeight(ctx, 6) })
	requireBlockedThroughPolls(t, node, done)

	cancel()
	require.ErrorIs(t, requireBarrierResult(t, done), context.Canceled)
}

func TestSingleNodeNextBlock(t *testing.T) {
	node := newFakeCometNode(t, 7, 7)
	c := newBarrierTestClient(t, SingleRpc, node)

	done := runBarrier(func() error { return c.WaitForNextBlock(context.Background()) })
	requireBlockedThroughPolls(t, node, done)

	node.commit(8)
	require.NoError(t, requireBarrierResult(t, done))
}

// A tx the chain never includes leaves ignite waiting on its own node, so the
// client's bound and node report must apply there as well.
func TestWaitForTxBoundedOnHaltedChain(t *testing.T) {
	node := newFakeCometNode(t, 7, 7)
	c := newBarrierTestClient(t, SingleRpc, node)
	c.nodeStallTimeout = 50 * time.Millisecond

	err := requireBarrierResult(t, runBarrier(func() error {
		_, err := c.WaitForTx(context.Background(), hex.EncodeToString(cmttypes.Tx("tx").Hash()))
		return err
	}))

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorContains(t, err, "node 0 ("+node.server.URL+") committed height 7")
}
