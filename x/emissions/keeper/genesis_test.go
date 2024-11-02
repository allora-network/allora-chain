package keeper_test

import (
	cosmossdk_io_math "cosmossdk.io/math"
)

// at minimum test that an import can be done from an export without error
func (s *KeeperTestSuite) TestImportExportGenesisNoError() {
	err := s.emissionsKeeper.SetTopicStake(s.ctx, 2, cosmossdk_io_math.OneInt())
	s.Require().NoError(err)
	intVal := int64(1234567890)
	err = s.emissionsKeeper.SetRewardCurrentBlockEmission(s.ctx, cosmossdk_io_math.NewInt(intVal))
	s.Require().NoError(err)
	genesisState, err := s.emissionsKeeper.ExportGenesis(s.ctx)
	s.Require().NoError(err)

	err = s.emissionsKeeper.InitGenesis(s.ctx, genesisState)
	s.Require().NoError(err)
	rewardCurrentBlockEmission, err := s.emissionsKeeper.GetRewardCurrentBlockEmission(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(rewardCurrentBlockEmission.String(), cosmossdk_io_math.NewInt(intVal).String())
}
