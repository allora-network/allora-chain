package keeper_test

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	"github.com/allora-network/allora-chain/x/mint/types"
)

func (s *IntegrationTestSuite) TestTotalEmissionPerMonthSimple() {
	// 1. Set up the test inputs
	rewardEmissionPerUnitStakedToken := math.NewInt(5).ToLegacyDec()
	numStakedTokens := math.NewInt(100)

	// 2. Execute the test
	totalEmission := keeper.GetTotalEmissionPerMonth(
		rewardEmissionPerUnitStakedToken,
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
	s.stakingKeeper.EXPECT().TotalBondedTokens(s.ctx).Return(math.NewInt(0), nil)
	s.emissionsKeeper.EXPECT().GetTotalStake(s.ctx).Return(math.NewUint(0), nil)
	nst, err := keeper.GetNumStakedTokens(s.ctx, s.mintKeeper)
	s.NoError(err)
	s.False(nst.IsNegative())
}

func (s *IntegrationTestSuite) TestGetExponentialMovingAverageSimple() {
	// e_i = α_e * ^e_i + (1 − α_e)*e_{i−1}
	// random numbers for test
	// e_i = 0.1 * 1000 + (1 - 0.1) * 800
	// e_i = 100 + 720
	// e_i = 820

	result := keeper.GetExponentialMovingAverage(
		math.LegacyMustNewDecFromStr("1000"),
		math.LegacyMustNewDecFromStr("0.1"),
		math.LegacyMustNewDecFromStr("800"),
	)

	expectedValue := math.NewInt(820).ToLegacyDec()
	s.Require().True(expectedValue.Equal(result))
}

func (s *IntegrationTestSuite) TestNumberLockedTokensBeforeVest() {
	defaultParams := types.DefaultParams()
	fullInvestors := defaultParams.InvestorsPercentOfTotalSupply.
		Mul(defaultParams.MaxSupply.ToLegacyDec()).TruncateInt()
	fullTeam := defaultParams.TeamPercentOfTotalSupply.
		Mul(defaultParams.MaxSupply.ToLegacyDec()).TruncateInt()
	expectedLocked := fullInvestors.Add(fullTeam)

	s.emissionsKeeper.EXPECT().GetParamsBlocksPerMonth(s.ctx).Return(uint64(525960), nil)
	bpm, err := s.emissionsKeeper.GetParamsBlocksPerMonth(s.ctx)
	s.Require().NoError(err)
	result := keeper.GetLockedTokenSupply(
		bpm,
		math.NewInt(int64(bpm*2)),
		defaultParams,
	)
	s.Require().True(result.Equal(expectedLocked), "expected %s, got %s", expectedLocked, result)
}

func (s *IntegrationTestSuite) TestNumberLockedTokensDuringVest() {
	defaultParams := types.DefaultParams()
	// after 13 months investors and team should get 1/3 + 1/36 = 13/36
	fractionUnlocked := math.LegacyNewDec(13).Quo(math.LegacyNewDec(36))
	fractionLocked := math.LegacyNewDec(1).Sub(fractionUnlocked)
	investors := defaultParams.InvestorsPercentOfTotalSupply.
		Mul(defaultParams.MaxSupply.ToLegacyDec()).
		Mul(fractionLocked).TruncateInt()
	team := defaultParams.TeamPercentOfTotalSupply.
		Mul(defaultParams.MaxSupply.ToLegacyDec()).
		Mul(fractionLocked).TruncateInt()
	expectedLocked := investors.Add(team)
	s.emissionsKeeper.EXPECT().GetParamsBlocksPerMonth(s.ctx).Return(uint64(525960), nil)
	bpm, err := s.emissionsKeeper.GetParamsBlocksPerMonth(s.ctx)
	s.Require().NoError(err)
	result := keeper.GetLockedTokenSupply(
		bpm,
		math.NewInt(int64(bpm*13+1)),
		defaultParams,
	)
	s.Require().True(result.Equal(expectedLocked), "expected %s, got %s", expectedLocked, result)
}

func (s *IntegrationTestSuite) TestNumberLockedTokensAfterVest() {
	defaultParams := types.DefaultParams()
	s.emissionsKeeper.EXPECT().GetParamsBlocksPerMonth(s.ctx).Return(uint64(525960), nil)
	bpm, err := s.emissionsKeeper.GetParamsBlocksPerMonth(s.ctx)
	s.Require().NoError(err)
	result := keeper.GetLockedTokenSupply(
		bpm,
		math.NewInt(int64(bpm*40)),
		defaultParams,
	)
	s.Require().True(result.Equal(math.ZeroInt()))
}

func (s *IntegrationTestSuite) TestTargetRewardEmissionPerUnitStakedTokenSimple() {
	// ^e_i = ((f_e*T_{total,i}) / N_{staked,i}) * (N_{circ,i} / N_{total,i})
	// using some random sample values
	//  ^e_i = ((0.015*2000)/400)*(10000000/12000000)

	result, err := keeper.GetTargetRewardEmissionPerUnitStakedToken(
		math.LegacyMustNewDecFromStr("0.015"),
		math.NewInt(200000),
		math.NewInt(400),
		math.NewInt(10000000),
		math.NewInt(12000000),
	)
	s.Require().NoError(err)
	fmt.Println(result)
}
