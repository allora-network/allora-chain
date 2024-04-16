package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	chain_test "github.com/allora-network/allora-chain/integration/chain"
	"github.com/stretchr/testify/suite"
)

type ExternalTestSuite struct {
	suite.Suite
	ctx context.Context
	n   chain_test.Node
}

func (s *ExternalTestSuite) SetupTest() {
	var err error
	s.ctx = context.Background()
	userHomeDir, _ := os.UserHomeDir()
	home := filepath.Join(userHomeDir, ".allorad")
	nodeConfig := chain_test.NewNodeConfig(
		s.T(),
		"http://localhost:26657",
		home,
	)
	s.n, err = chain_test.NewNode(nodeConfig)
	s.Require().NoError(err)
}

func TestExternalTestSuite(t *testing.T) {
	suite.Run(t, new(ExternalTestSuite))
}
