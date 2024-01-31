package module_test

import (
	cosmosMath "cosmossdk.io/math"
	"github.com/upshot-tech/protocol-state-machine-module/module"
)

func (s *ModuleTestSuite) TestGetCumulativeEmissionSomeBlocks() {
}

func (s *ModuleTestSuite) TestGetCumulativeEmissionZeroBlocks() {
	result := module.GetCumulativeEmission(s.ctx, s.appModule, 0)

	s.Require().Equal(cosmosMath.ZeroUint(), result)
}
