package integration_test

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"testing"

	cosmosMath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// fakeEmissionsNode answers IsTopicActive with a fixed response or error and
// records the block height header it was called with.
type fakeEmissionsNode struct {
	emissionstypes.QueryServiceClient
	active  bool
	err     error
	heights []string
}

func (f *fakeEmissionsNode) IsTopicActive(ctx context.Context, _ *emissionstypes.IsTopicActiveRequest, _ ...grpc.CallOption) (*emissionstypes.IsTopicActiveResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.heights = append(f.heights, md.Get("x-cosmos-block-height")...)
	if f.err != nil {
		return nil, f.err
	}
	return &emissionstypes.IsTopicActiveResponse{IsActive: f.active}, nil
}

func runIsTopicActive(nodes []*fakeEmissionsNode) (*emissionstypes.IsTopicActiveResponse, []string, error) {
	clients := make([]emissionstypes.QueryServiceClient, len(nodes))
	for i, node := range nodes {
		clients[i] = node
	}
	var violations []string
	response, err := queryEveryNode(context.Background(),
		func(height int64, format string, args ...any) {
			violations = append(violations, fmt.Sprintf("height %d: %s", height, fmt.Sprintf(format, args...)))
		},
		42, clients, emissionstypes.QueryServiceClient.IsTopicActive, &emissionstypes.IsTopicActiveRequest{TopicId: 7})
	return response, violations, err
}

func TestQueryEveryNodeAgreeingNodesRecordNothing(t *testing.T) {
	nodes := []*fakeEmissionsNode{
		{QueryServiceClient: nil, active: true, err: nil, heights: nil},
		{QueryServiceClient: nil, active: true, err: nil, heights: nil},
		{QueryServiceClient: nil, active: true, err: nil, heights: nil},
	}
	response, violations, err := runIsTopicActive(nodes)
	require.NoError(t, err)
	require.True(t, response.IsActive)
	require.Empty(t, violations)
	for i, node := range nodes {
		require.Equal(t, []string{"42"}, node.heights, "node %d must be queried at the audited height", i)
	}
}

func TestQueryEveryNodeDisagreeingResponseIsAViolation(t *testing.T) {
	nodes := []*fakeEmissionsNode{
		{QueryServiceClient: nil, active: true, err: nil, heights: nil},
		{QueryServiceClient: nil, active: true, err: nil, heights: nil},
		{QueryServiceClient: nil, active: false, err: nil, heights: nil},
	}
	response, violations, err := runIsTopicActive(nodes)
	require.NoError(t, err)
	require.True(t, response.IsActive, "node 0's response is returned")
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], "height 42")
	require.Contains(t, violations[0], "IsTopicActiveRequest")
	require.Contains(t, violations[0], "differs from node(s) 2")
	require.NotContains(t, violations[0], "node(s) 1")
}

func TestQueryEveryNodeNode0DisagreeingNamesEveryOtherNode(t *testing.T) {
	nodes := []*fakeEmissionsNode{
		{QueryServiceClient: nil, active: false, err: nil, heights: nil},
		{QueryServiceClient: nil, active: true, err: nil, heights: nil},
		{QueryServiceClient: nil, active: true, err: nil, heights: nil},
	}
	_, violations, _ := runIsTopicActive(nodes)
	require.Len(t, violations, 1)
	require.Contains(t, violations[0], "differs from node(s) 1,2")
}

func TestQueryEveryNodeErrorOnOneNodeIsAViolation(t *testing.T) {
	for failing := range 3 {
		nodes := []*fakeEmissionsNode{
			{QueryServiceClient: nil, active: true, err: nil, heights: nil},
			{QueryServiceClient: nil, active: true, err: nil, heights: nil},
			{QueryServiceClient: nil, active: true, err: nil, heights: nil},
		}
		nodes[failing].err = status.Error(codes.Unavailable, "node down")
		_, violations, _ := runIsTopicActive(nodes)
		require.Len(t, violations, 1, "failing node %d", failing)
		require.Contains(t, violations[0], "node down")
	}
}

func TestQueryEveryNodeSameErrorOnAllNodesIsAgreement(t *testing.T) {
	nodes := []*fakeEmissionsNode{
		{err: status.Error(codes.NotFound, "no topic"), QueryServiceClient: nil, active: false, heights: nil},
		{err: status.Error(codes.NotFound, "no topic"), QueryServiceClient: nil, active: false, heights: nil},
	}
	_, violations, err := runIsTopicActive(nodes)
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
	require.Empty(t, violations)
}

func TestQueryEveryNodeDifferentErrorCodesAreAViolation(t *testing.T) {
	nodes := []*fakeEmissionsNode{
		{err: status.Error(codes.NotFound, "no topic"), QueryServiceClient: nil, active: false, heights: nil},
		{err: status.Error(codes.Internal, "boom"), QueryServiceClient: nil, active: false, heights: nil},
	}
	_, violations, _ := runIsTopicActive(nodes)
	require.Len(t, violations, 1)
}

func TestQueryEveryNodeWithoutNodesIsAViolation(t *testing.T) {
	_, violations, err := runIsTopicActive(nil)
	require.Error(t, err)
	require.Len(t, violations, 1)
}

// TestEventAuditUsesNoSingleNodeQueries keeps every state query of the audit on
// the every-node helper: the audit's source files may not select any of the
// Client accessors or slices that hand out one node's query client.
func TestEventAuditUsesNoSingleNodeQueries(t *testing.T) {
	files, err := filepath.Glob("event_*_test.go")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	// event_audit_query_test.go is the one place that reads the per-node clients.
	allowed := "event_audit_query_test.go"
	forbidden := []string{
		"QueryAuth", "QueryBank", "QueryDistribution", "QueryEmissions", "QueryMint", "QueryGov", "QueryUpgrade",
		"QueryAuths", "QueryBanks", "QueryDistributions", "QueryEmissionses", "QueryMints", "QueryGovs", "QueryUpgrades",
	}
	for _, file := range files {
		if file == allowed {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		require.NoError(t, err)
		ast.Inspect(parsed, func(n ast.Node) bool {
			selector, ok := n.(*ast.SelectorExpr)
			if ok && slices.Contains(forbidden, selector.Sel.Name) {
				t.Errorf("%s uses %s; run audit queries through queryEmissions or queryBank so every node answers", file, selector.Sel.Name)
			}
			return true
		})
	}
}

// Equal numbers decoded from the wire and built in memory differ under proto.Equal,
// representation, so the comparison must look at the encoded data.
func TestSameResponseIgnoresGoRepresentationOfNumbers(t *testing.T) {
	built := &emissionstypes.GetTopicStakeResponse{Amount: cosmosMath.NewInt(1_000_000)}
	encoded, err := built.Marshal()
	require.NoError(t, err)
	var decoded emissionstypes.GetTopicStakeResponse
	require.NoError(t, decoded.Unmarshal(encoded))

	require.True(t, sameResponse(built, &decoded))
	require.False(t, sameResponse(built, &emissionstypes.GetTopicStakeResponse{Amount: cosmosMath.NewInt(1_000_001)}))
}
