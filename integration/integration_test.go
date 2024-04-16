package integration_test

import (
	"context"
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
	nodeConfig := chain_test.NewNodeConfig(
		s.T(),
		"http://localhost:26657",
		"test",
		"test test test test test test test test test test test test test test test test test test test test test test test test",
		"test",
		"",
		"_",
		10,
	)
	s.n, err = chain_test.NewNode(nodeConfig)
	s.Require().NoError(err)
}

func TestExternalTestSuite(t *testing.T) {
	suite.Run(t, new(ExternalTestSuite))
}
