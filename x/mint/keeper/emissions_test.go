package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/keeper"
)

func (s *IntegrationTestSuite) TestTotalEmissionPerTimestepSimple() {
	// 1. Set up the test inputs
	rewardEmissionPerUnitStakedToken := math.NewInt(10)
	numStakedTokens := math.NewInt(100)

	// 2. Execute the test
	totalEmission := keeper.TotalEmissionPerTimestep(rewardEmissionPerUnitStakedToken, numStakedTokens)

	// 3. Check the results
	s.Require().Equal(math.NewInt(1000), totalEmission)
}

// in order to properly test this function we'd have to mock
// all the staking stuff which is a pain in the behind
// we will test that in integration, for now just test the value is non
// negative aka zero when you don't have stakers
func (s *IntegrationTestSuite) TestGetNumStakedTokensNonNegative() {
	s.stakingKeeper.EXPECT().StakingTokenSupply(s.ctx).Return(math.NewInt(0), nil)
	s.emissionsKeeper.EXPECT().GetTotalStake(s.ctx).Return(math.NewUint(0), nil)
	nst, err := keeper.GetNumStakedTokens(s.ctx, s.mintKeeper)
	s.NoError(err)
	s.False(nst.IsNegative())
}
