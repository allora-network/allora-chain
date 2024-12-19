package keeper_test

import (
	cosmossdk_io_math "cosmossdk.io/math"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// at minimum test that an import can be done from an export without error
func (s *KeeperTestSuite) TestImportExportGenesisNoError() {
	testAddr := s.addrs[0].String()
	err := s.emissionsKeeper.AddWhitelistAdmin(s.ctx, testAddr)
	s.Require().NoError(err)

	err = s.emissionsKeeper.SetTopicStake(s.ctx, 2, cosmossdk_io_math.OneInt())
	s.Require().NoError(err)
	genesisState, err := s.emissionsKeeper.ExportGenesis(s.ctx)
	s.Require().NoError(err)

	err = s.emissionsKeeper.InitGenesis(s.ctx, genesisState)
	s.Require().NoError(err)

	for _, addr := range types.DefaultCoreTeamAddresses() {
		admin, err := s.emissionsKeeper.IsWhitelistAdmin(s.ctx, addr)
		s.Require().NoError(err)
		s.Require().Equal(admin, true)
	}
	admin, err := s.emissionsKeeper.IsWhitelistAdmin(s.ctx, testAddr)
	s.Require().NoError(err)
	s.Require().Equal(admin, true)
}
