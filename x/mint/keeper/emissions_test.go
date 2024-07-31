package keeper_test

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/mint/keeper"
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
	expectedResult := s.epoch61Get("ehat_target_i")

	simulatorFEmission := cosmosMath.LegacyMustNewDecFromStr("0.025")
	networkTokensTotal := s.epoch61Get("network_tokens_total").SdkIntTrim()
	ecosystemTokensTotal := s.epoch61Get("ecosystem_tokens_total").SdkIntTrim()
	networkTokensCirculating := s.epoch61Get("network_tokens_circulating").SdkIntTrim()
	networkTokensStaked := s.epoch61Get("network_tokens_staked").SdkIntTrim()
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

/**
epoch,
time,
network_tokens_total,
network_tokens_emission,
network_tokens_circulating,
network_tokens_staked,
investors_preseed_tokens_total,
investors_preseed_tokens_emission,
investors_preseed_tokens_circulating,
investors_preseed_tokens_staked,
investors_seed_tokens_total,
investors_seed_tokens_emission,
investors_seed_tokens_circulating,
investors_seed_tokens_staked,
team_tokens_total,
team_tokens_emission,
team_tokens_circulating,
team_tokens_staked,
ecosystem_tokens_total,
ecosystem_tokens_emission,
ecosystem_tokens_circulating,
ecosystem_tokens_staked,
foundation_tokens_total,
foundation_tokens_emission,
foundation_tokens_circulating,
foundation_tokens_staked,
participants_tokens_total,
participants_tokens_emission,
participants_tokens_circulating,
participants_tokens_staked,
ehat_target_i,
ehat_max_i,
ehat_i,
e_i,
validator_rewards,
topics_rewards,
n_active_topics,
all_topic_weight_sum,
topic_id_0,
topic_months_live_0,
topic_n_workers_0,
topic_n_reputers_0,
topic_total_stake_0,
topic_total_fee_revenue_0,
topic_weights_0,
topic_rewards_0,
topic_id_1,
topic_months_live_1,
topic_n_workers_1,
topic_n_reputers_1,
topic_total_stake_1,
topic_total_fee_revenue_1,
topic_weights_1,
topic_rewards_1,
topic_id_2,
topic_months_live_2,
topic_n_workers_2,
topic_n_reputers_2,
topic_total_stake_2,
topic_total_fee_revenue_2,
topic_weights_2,
topic_rewards_2,
n_active_validators,
all_validator_stake_sum,
validator_id_0,
validator_months_live_0,
validator_stake_0,
validator_rewards_0,
validator_id_1,
validator_months_live_1,
validator_stake_1,
validator_rewards_1,
validator_id_2,
validator_months_live_2,
validator_stake_2,
validator_rewards_2
*/

/*
func (s *IntegrationTestSuite) TestEHatMaxIFromCsv() {
	expectedResult := s.epoch61Get("ehat_max_i")
	ehatTargetI := s.epoch61Get("ehat_target_i")


	keeper.GetMaximumMonthlyEmissionPerUnitStakedToken(

	)
}
*/

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
	expectedResult := s.epoch61Get("ehat_i")
	ehatMaxI := s.epoch61Get("ehat_max_i").SdkLegacyDec()
	ehatTargetI := s.epoch61Get("ehat_target_i").SdkLegacyDec()

	result := keeper.GetCappedTargetEmissionPerUnitStakedToken(
		ehatTargetI,
		ehatMaxI,
	)
	resultD, err := alloraMath.NewDecFromSdkLegacyDec(result)
	s.Require().NoError(err)
	testutil.InEpsilon5D(s.T(), resultD, expectedResult)
}

func (s *IntegrationTestSuite) TestESubIFromPrintfDebuggingPythonNotebook() {
	// from printf debugging the python notebook:
	//    if i == 61:
	//        e_i = (alpha_emission_per_token*target_emission_per_token+(1-alpha_emission_per_token)*old_emission_per_token)
	//        print(target_emission_per_token)
	//        print(old_emission_per_token)
	//        print(e_i)
	//
	// Values in the CSV are wrong!! not equal to the actual values used in the simulator!!
	expectedResultPrintf := alloraMath.MustNewDecFromString("0.0066204043241312")
	//expectedResult := s.epoch61Get("e_i")
	//fmt.Printf("%v | %v\n", expectedResult, expectedResultPrintf)
	// 0.006596912577740069 | 0.0066204043241312
	targetE_iPrintf := alloraMath.MustNewDecFromString("0.00545700205370986")
	//targetE_i := s.epoch61Get("ehat_target_i")
	//fmt.Printf("%v | %v\n", targetE_i, targetE_iPrintf)
	// 0.005316238669484353 | 0.00545700205370986
	previousE_iPrintf := alloraMath.MustNewDecFromString("0.006749671243066904")
	//previousE_i := s.epochGet[60]("e_i")
	//fmt.Printf("%v | %v\n", previousE_i, previousE_iPrintf)
	// 0.006749671243066904 | 0.006749671243066904

	// this is taken directly from the python notebook
	alpha_Emission := cosmosMath.LegacyMustNewDecFromStr("0.1")

	result := keeper.GetExponentialMovingAverage(
		targetE_iPrintf.SdkLegacyDec(),
		alpha_Emission,
		previousE_iPrintf.SdkLegacyDec(),
	)
	resultD, err := alloraMath.NewDecFromSdkLegacyDec(result)
	s.Require().NoError(err)
	testutil.InEpsilon5D(s.T(), resultD, expectedResultPrintf)
}
