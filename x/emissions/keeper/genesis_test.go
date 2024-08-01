package keeper_test

import (
	cosmossdk_io_math "cosmossdk.io/math"
)

// at minimum test that an import can be done from an export without error
func (s *KeeperTestSuite) TestImportExportGenesisNoError() {
	s.emissionsKeeper.SetTopicStake(s.ctx, 0, cosmossdk_io_math.OneInt())
	genesisState, err := s.emissionsKeeper.ExportGenesis(s.ctx)
	s.Require().NoError(err)

	err = s.emissionsKeeper.InitGenesis(s.ctx, genesisState)
	s.Require().NoError(err)
}
