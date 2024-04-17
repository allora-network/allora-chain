package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	chain_test "github.com/allora-network/allora-chain/integration/chain"
	"github.com/stretchr/testify/require"
)

type TestMetadata struct {
	t   *testing.T
	ctx context.Context
	n   chain_test.Node
}

func Setup(t *testing.T) TestMetadata {
	ret := TestMetadata{}
	ret.t = t
	var err error
	ret.ctx = context.Background()
	userHomeDir, _ := os.UserHomeDir()
	home := filepath.Join(userHomeDir, ".allorad")
	node, err := chain_test.NewNode(
		t,
		chain_test.NodeConfig{
			NodeRPCAddress: "http://localhost:26657",
			AlloraHomeDir:  home,
		},
	)
	require.NoError(t, err)
	ret.n = node
	return ret
}

func TestExternalTestSuite(t *testing.T) {
	t.Log("Setting up connection to local node")
	m := Setup(t)
	t.Log("Test: GetParams")
	GetParams(m)
	t.Log("Test: CreateTopic")
	CreateTopic(m)
}
