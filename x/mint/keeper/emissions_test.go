package keeper_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	mint "github.com/allora-network/allora-chain/x/mint/module"
	minttestutil "github.com/allora-network/allora-chain/x/mint/testutil"
	"github.com/allora-network/allora-chain/x/mint/types"
)

func (s *IntegrationTestSuite) TestTotalEmissionPerMonthSimple() {
	// 1. Set up the test inputs
	rewardEmissionPerUnitStakedToken := cosmosMath.NewInt(5).ToLegacyDec()
	numStakedTokens := cosmosMath.NewInt(100)

	// 2. Execute the test
	totalEmission := keeper.GetTotalEmissionPerMonth(
		rewardEmissionPerUnitStakedToken,
		numStakedTokens,
	)

	// 3. Check the results
	s.Require().Equal(cosmosMath.NewInt(500), totalEmission)
}

// in order to properly test this function we'd have to mock
// all the staking stuff which is a pain in the behind
// we will test that in integration, for now just test the value is non
// negative aka zero when you don't have stakers
func (s *IntegrationTestSuite) TestGetNumStakedTokensNonNegative() {
	s.stakingKeeper.EXPECT().TotalBondedTokens(s.ctx).Return(cosmosMath.NewInt(0), nil)
	s.emissionsKeeper.EXPECT().GetTotalStake(s.ctx).Return(cosmosMath.NewInt(0), nil)
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
		cosmosMath.LegacyMustNewDecFromStr("1000"),
		cosmosMath.LegacyMustNewDecFromStr("0.1"),
		cosmosMath.LegacyMustNewDecFromStr("800"),
	)

	expectedValue := cosmosMath.NewInt(820).ToLegacyDec()
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
		cosmosMath.NewInt(int64(bpm*2)),
		defaultParams,
	)
	s.Require().True(result.Equal(expectedLocked), "expected %s, got %s", expectedLocked, result)
}

func (s *IntegrationTestSuite) TestNumberLockedTokensDuringVest() {
	defaultParams := types.DefaultParams()
	// after 13 months investors and team should get 1/3 + 1/36 = 13/36
	fractionUnlocked := cosmosMath.LegacyNewDec(13).Quo(cosmosMath.LegacyNewDec(36))
	fractionLocked := cosmosMath.LegacyNewDec(1).Sub(fractionUnlocked)
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
		cosmosMath.NewInt(int64(bpm*13+1)),
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
		cosmosMath.NewInt(int64(bpm*40)),
		defaultParams,
	)
	s.Require().True(result.Equal(cosmosMath.ZeroInt()))
}

func (s *IntegrationTestSuite) TestTargetRewardEmissionPerUnitStakedTokenSimple() {
	// ^e_i = ((f_e*T_{total,i}) / N_{staked,i}) * (N_{circ,i} / N_{total,i})
	// using some random sample values
	//  ^e_i = ((0.015*2000)/400)*(10000000/12000000)

	_, err := keeper.GetTargetRewardEmissionPerUnitStakedToken(
		cosmosMath.LegacyMustNewDecFromStr("0.015"),
		cosmosMath.NewInt(200000),
		cosmosMath.NewInt(400),
		cosmosMath.NewInt(10000000),
		cosmosMath.NewInt(12000000),
	)
	s.Require().NoError(err)
}

// match ^e_i from row 61
func (s *IntegrationTestSuite) TestEHatTargetFromCsv() {
	epoch := s.epochGet[61]
	expectedResult := epoch("ehat_target_i")

	simulatorFEmission := cosmosMath.LegacyMustNewDecFromStr("0.025")
	networkTokensTotal := epoch("network_tokens_total").SdkIntTrim()
	ecosystemTokensTotal := epoch("ecosystem_tokens_total").SdkIntTrim()
	networkTokensCirculating := epoch("network_tokens_circulating").SdkIntTrim()
	networkTokensStaked := epoch("network_tokens_staked").SdkIntTrim()
	result, err := keeper.GetTargetRewardEmissionPerUnitStakedToken(
		simulatorFEmission,
		ecosystemTokensTotal,
		networkTokensStaked,
		networkTokensCirculating,
		networkTokensTotal,
	)
	s.Require().NoError(err)
	resultD, err := alloraMath.NewDecFromSdkLegacyDec(result)
	s.Require().NoError(err)
	testutil.InEpsilon5D(s.T(), resultD, expectedResult)
}

func (s *IntegrationTestSuite) TestEHatMaxAtGenesisFromCsv() {
	epoch0Get := s.epochGet[0]
	expectedResult := epoch0Get("ehat_max_i")
	// not exposed in csv, but taken looking directly from python notebook:
	// f_validators = 0.25
	// f_stake = f_validators+(1.-f_validators)/3.
	// calculated by hand:
	// >>> f_stake = 0.5
	// pick two values that will make f_stake equal to 0.5 like above:
	f_reputers := cosmosMath.LegacyMustNewDecFromStr("0.333333333333333333")
	f_validators := cosmosMath.LegacyMustNewDecFromStr("0.25")

	// max_apy = 0.12
	// max_mpy = (1.+max_apy)**(1./12.)-1.
	// >>> max_mpy = 0.009488792934583046
	max_mpy := cosmosMath.LegacyMustNewDecFromStr("0.009488792934583046")

	// max_emission_per_token = max_mpy/f_stake
	// >>> max_emission_per_token = 0.01897758586916609
	result := keeper.GetMaximumMonthlyEmissionPerUnitStakedToken(
		max_mpy,
		f_reputers,
		f_validators,
	)
	resultD, err := alloraMath.NewDecFromSdkLegacyDec(result)
	s.Require().NoError(err)
	testutil.InEpsilon5D(s.T(), resultD, expectedResult)
}

func (s *IntegrationTestSuite) TestEhatIFromCsv() {
	epoch := s.epochGet[61]
	expectedResult := epoch("ehat_i")
	ehatMaxI := epoch("ehat_max_i").SdkLegacyDec()
	ehatTargetI := epoch("ehat_target_i").SdkLegacyDec()

	result := keeper.GetCappedTargetEmissionPerUnitStakedToken(
		ehatTargetI,
		ehatMaxI,
	)
	resultD, err := alloraMath.NewDecFromSdkLegacyDec(result)
	s.Require().NoError(err)
	testutil.InEpsilon5D(s.T(), resultD, expectedResult)
}

// calculate e_i for the 61st epoch
func (s *IntegrationTestSuite) TestESubIFromCsv() {
	expectedResult := s.epochGet[61]("e_i")
	targetE_i := s.epochGet[61]("ehat_target_i").SdkLegacyDec()
	previousE_i := s.epochGet[60]("e_i").SdkLegacyDec()

	// this is taken directly from the python notebook
	alpha_Emission := cosmosMath.LegacyMustNewDecFromStr("0.1")

	result := keeper.GetExponentialMovingAverage(
		targetE_i,
		alpha_Emission,
		previousE_i,
	)
	resultD, err := alloraMath.NewDecFromSdkLegacyDec(result)
	s.Require().NoError(err)
	testutil.InEpsilon5D(s.T(), resultD, expectedResult)
}

// calculate \cal E for the 61st epoch
// GetTotalEmissionPerMonth
func (s *IntegrationTestSuite) TestCalEFromCsv() {
	expectedResult := s.epochGet[61]("ecosystem_tokens_emission")
	rewardEmissionPerUnitStakedToken := s.epochGet[61]("e_i").SdkLegacyDec()
	// use the value from epoch 60 rather than 61 because the python notebook
	// updates the value AFTER calculating the total emission and handing out rewards
	numStakedTokens := s.epochGet[60]("network_tokens_staked").SdkIntTrim()
	totalEmission := keeper.GetTotalEmissionPerMonth(
		rewardEmissionPerUnitStakedToken,
		numStakedTokens,
	)
	resultD, err := alloraMath.NewDecFromSdkInt(totalEmission)
	s.Require().NoError(err)
	testutil.InEpsilon5D(s.T(), resultD, expectedResult)
}

/* todo: think about how to get a grip on this
func (s *IntegrationTestSuite) TestGetLockedTokenSupply() {
	for i := 0; i < 20; i++ {
		epoch := s.epochGet[i]
		total := epoch("investors_preseed_tokens_total")
		emission := epoch("investors_preseed_tokens_emission")
		circulating := epoch("investors_preseed_tokens_circulating")
		staked := epoch("investors_preseed_tokens_staked")
		fmt.Println("epoch ", i, total, emission, circulating, staked)
	}
}
*/

// test the ABCI function that calculates \cal E
func (s *IntegrationTestSuite) TestGetEmissionPerMonth() {
	//prevEpoch := s.epochGet[60]
	epoch := s.epochGet[61]
	// we're mocking EVERYTHING
	mockMintKeeper := minttestutil.NewMockMintKeeper(s.ctrl)
	blocksPerMonth := uint64(1)
	params := types.Params{}
	ecosystemMintSupplyRemaining := cosmosMath.NewInt(1000000000000000000)
	validatorsPercent := cosmosMath.LegacyMustNewDecFromStr("0.25")
	calE, ehat_i, err := mint.GetEmissionPerMonth(
		s.ctx,
		mockMintKeeper,
		blocksPerMonth,
		params,
		ecosystemMintSupplyRemaining,
		validatorsPercent,
	)
	s.Require().NoError(err)
	calED, err := alloraMath.NewDecFromSdkInt(calE)
	s.Require().NoError(err)
	testutil.InEpsilon5D(s.T(), epoch("ecosystem_tokens_emission"), calED)
	ehat_iD, err := alloraMath.NewDecFromSdkLegacyDec(ehat_i)
	testutil.InEpsilon5D(s.T(), epoch("ehat_i"), ehat_iD)
}
