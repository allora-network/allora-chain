package integration_test

import (
	"testing"

	chain_test "github.com/allora-network/allora-chain/integration/chain"
	emissions "github.com/allora-network/allora-chain/x/mint/module"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	auth "github.com/cosmos/cosmos-sdk/x/auth"
	bank "github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/stretchr/testify/suite"
)

type ExternalTestSuite struct {
	suite.Suite
	n chain_test.NodeConfig
}

func (s *ExternalTestSuite) SetupTest() {
	encCfg := moduletestutil.MakeTestEncodingConfig(
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		emissions.AppModule{},
		mint.AppModule{},
	)

	s.n = chain_test.NewNodeConfig(s.T(), "localhost", "1317", encCfg.Codec)
}

func TestExternalTestSuite(t *testing.T) {
	suite.Run(t, new(ExternalTestSuite))
}
