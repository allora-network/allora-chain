package keeper_test

import (
	cosmossdk_io_math "cosmossdk.io/math"

	"github.com/allora-network/allora-chain/x/emissions/types"
)

// at minimum test that an import can be done from an export without error
func (s *KeeperTestSuite) TestImportExportGenesisNoError() {
	testAddr := s.AddrsStr(0)
	err := s.EmissionsKeeper().AddWhitelistAdmin(s.Ctx(), testAddr)
	s.Require().NoError(err)

	err = s.EmissionsKeeper().SetTopicStake(s.Ctx(), 1, cosmossdk_io_math.OneInt())
	s.Require().NoError(err)
	genesisState, err := s.EmissionsKeeper().ExportGenesis(s.Ctx())
	s.Require().NoError(err)

	err = s.EmissionsKeeper().InitGenesis(s.Ctx(), genesisState)
	s.Require().NoError(err)

	for _, addr := range types.DefaultCoreTeamAddresses() {
		admin, err := s.EmissionsKeeper().IsWhitelistAdmin(s.Ctx(), addr)
		s.Require().NoError(err)
		s.Require().Equal(admin, true)
	}
	admin, err := s.EmissionsKeeper().IsWhitelistAdmin(s.Ctx(), testAddr)
	s.Require().NoError(err)
	s.Require().Equal(admin, true)
}
