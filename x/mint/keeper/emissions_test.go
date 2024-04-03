package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/keeper"
)

func (s *IntegrationTestSuite) TestTotalEmissionPerTimestepSimple() {
	// 1. Set up the test inputs
	rewardEmissionPerUnitStakedTokenNumerator := math.NewInt(10)
	rewardEmissionPerUnitStakedTokenDenominator := math.NewInt(2)
	numStakedTokens := math.NewInt(100)

	// 2. Execute the test
	totalEmission := keeper.TotalEmissionPerTimestep(
		rewardEmissionPerUnitStakedTokenNumerator,
		rewardEmissionPerUnitStakedTokenDenominator,
		numStakedTokens,
	)

	// 3. Check the results
	s.Require().Equal(math.NewInt(500), totalEmission)
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

// test the smoothing factor for a daily timestep
func (s *IntegrationTestSuite) TestSmoothingFactorPerBlockSimple() {
	// ^α_e = 1 - (1 - α_e)^(∆t/month)
	// default α_e is 0.1
	// ∆t = 1 day = 30 per month
	// ^α_e = 1 - (1 - 0.1)^(30)
	// ^α_e = 0.957608841724783796485705566799
	// ^α_e = 957608841724783796485705566799 / 1000000000000000000000000000000
	expectedNumerator, ok := math.NewIntFromString("957608841724783796485705566799")
	s.Require().True(ok)
	expectedDenominator, ok := math.NewIntFromString("1000000000000000000000000000000")
	s.Require().True(ok)

	resultNumerator, resultDenominator := keeper.SmoothingFactorPerBlock(
		s.ctx,
		s.mintKeeper,
		math.NewInt(1),  // 0.1 | 1 over 10, so numerator is 1
		math.NewInt(10), // 0.1 | 1 over 10 so denominator is 10
		30,              // there are 30 days in a month (shh, close enough)
	)

	s.Require().True(resultDenominator.GTE(resultNumerator)) // we should be dealing with a fraction
	s.Require().True(expectedNumerator.Equal(resultNumerator))
	s.Require().True(expectedDenominator.Equal(resultDenominator))
}

func (s *IntegrationTestSuite) TestRewardEmissionPerUnitStakedTokenSimple() {
	// e_i = α_e * ^e_i + (1 − α_e)*e_{i−1}
	// random numbers for test
	// e_i = 0.1 * 1000 + (1 - 0.1) * 800
	// e_i = 100 + 720
	// e_i = 820

	resultNumerator, resultDenominator := keeper.RewardEmissionPerUnitStakedToken(
		math.NewInt(1000),
		math.NewInt(1),
		math.NewInt(1),
		math.NewInt(10),
		math.NewInt(800),
		math.NewInt(1),
	)

	expectedValue := math.NewInt(820)
	s.Require().True(expectedValue.Equal(resultNumerator.Quo(resultDenominator)))
	s.Require().True(resultNumerator.Equal(math.NewInt(8200)))
	s.Require().True(resultDenominator.Equal(math.NewInt(10)))
}

func (s *IntegrationTestSuite) TestNumberLockedTokensSimple() {
	result := keeper.GetLockedTokenSupply()
	s.Require().True(result.Equal(math.NewInt(0)))
}

func (s *IntegrationTestSuite) TestTargetRewardEmissionPerUnitStakedTokenSimple() {
	// ^e_i = ((f_e*T_{total,i}) / N_{staked,i}) * (N_{circ,i} / N_{total,i})
	// using some random sample values
	//  ^e_i = ((0.015*2000)/400)*(10000000/12000000)

	resultNumerator, resultDenominator, err := keeper.TargetRewardEmissionPerUnitStakedToken(
		math.NewInt(15),
		math.NewInt(1000),
		math.NewInt(200000),
		math.NewInt(400),
		math.NewInt(10000000),
		math.NewInt(12000000),
	)
	s.Require().NoError(err)
	fmt.Println(resultNumerator, resultDenominator)
}
