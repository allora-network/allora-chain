package integration_test

import (
	"context"
	"fmt"
	"strings"
	"sync"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	testCommon "github.com/allora-network/allora-chain/test/common"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// auditNodes holds one query client per node. The audit reads the clients only
// through queryEmissions and queryBank, which run each query on every node, so
// an answer that one node computes differently cannot pass unnoticed. Query
// answers are outside every consensus hash, which makes this comparison the only
// check on them. The audit's source files must not call the Client.Query*
// accessors; TestEventAuditUsesNoSingleNodeQueries enforces that.
type auditNodes struct {
	emissions []emissionstypes.QueryServiceClient
	bank      []banktypes.QueryClient
}

// newAuditNodes copies the per-node query clients the test config already built.
func newAuditNodes(client *testCommon.Client) auditNodes {
	return auditNodes{
		emissions: append([]emissionstypes.QueryServiceClient(nil), client.QueryEmissionses...),
		bank:      append([]banktypes.QueryClient(nil), client.QueryBanks...),
	}
}

// queryEmissions runs one emissions query at height on every node. See queryEveryNode.
func queryEmissions[Req, Resp proto.Message](
	a *eventAudit,
	height int64,
	method func(emissionstypes.QueryServiceClient, context.Context, Req, ...grpc.CallOption) (Resp, error),
	req Req,
) (Resp, error) {
	return queryEveryNode(a.ctx, a.failf, height, a.nodes.emissions, method, req)
}

// queryBank runs one bank query at height on every node. See queryEveryNode.
func queryBank[Req, Resp proto.Message](
	a *eventAudit,
	height int64,
	method func(banktypes.QueryClient, context.Context, Req, ...grpc.CallOption) (Resp, error),
	req Req,
) (Resp, error) {
	return queryEveryNode(a.ctx, a.failf, height, a.nodes.bank, method, req)
}

// queryEveryNode calls method with req on each client at height, concurrently, and
// records a violation through fail when the nodes do not give the same answer:
// responses that differ under proto.Equal, an error on some nodes but not on
// others, or errors with different gRPC status codes. It returns node 0's
// response and error so the caller's comparison is unchanged.
func queryEveryNode[C any, Req, Resp proto.Message](
	ctx context.Context,
	fail func(height int64, format string, args ...any),
	height int64,
	clients []C,
	method func(C, context.Context, Req, ...grpc.CallOption) (Resp, error),
	req Req,
) (Resp, error) {
	queryCtx := atHeight(ctx, height)
	responses := make([]Resp, len(clients))
	errs := make([]error, len(clients))
	var wg sync.WaitGroup
	for node := range clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			responses[node], errs[node] = method(clients[node], queryCtx, req)
		}()
	}
	wg.Wait()

	if len(clients) == 0 {
		var zero Resp
		fail(height, "query %s has no node to run on", proto.MessageName(req))
		return zero, fmt.Errorf("no node clients for %s", proto.MessageName(req))
	}
	if differing := nodesDisagreeing(responses, errs); len(differing) > 0 {
		fail(height, "query %s %s: node 0 differs from node(s) %s: %s",
			proto.MessageName(req), req.String(), joinNodes(differing), describeAnswers(responses, errs, differing))
	}
	return responses[0], errs[0]
}

// nodesDisagreeing returns the nodes whose answer differs from node 0's. An
// answer is either an error, compared by gRPC status code, or a response,
// compared with proto.Equal.
func nodesDisagreeing[Resp proto.Message](responses []Resp, errs []error) []int {
	var differing []int
	for node := 1; node < len(responses); node++ {
		switch {
		case (errs[0] != nil) != (errs[node] != nil):
			differing = append(differing, node)
		case errs[0] != nil:
			if status.Code(errs[0]) != status.Code(errs[node]) {
				differing = append(differing, node)
			}
		case !proto.Equal(responses[0], responses[node]):
			differing = append(differing, node)
		}
	}
	return differing
}

func joinNodes(nodes []int) string {
	parts := make([]string, len(nodes))
	for i, node := range nodes {
		parts[i] = fmt.Sprint(node)
	}
	return strings.Join(parts, ",")
}

// describeAnswers renders node 0's answer and the answer of each differing node.
func describeAnswers[Resp proto.Message](responses []Resp, errs []error, differing []int) string {
	render := func(node int) string {
		if errs[node] != nil {
			return fmt.Sprintf("node %d error: %v", node, errs[node])
		}
		return fmt.Sprintf("node %d response {%s}", node, responses[node].String())
	}
	parts := []string{render(0)}
	for _, node := range differing {
		parts = append(parts, render(node))
	}
	return strings.Join(parts, "; ")
}
