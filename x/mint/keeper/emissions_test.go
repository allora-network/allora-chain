package keeper_test

import (
	"fmt"
	"math"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	"github.com/allora-network/allora-chain/x/mint/types"
)

func (s *MintKeeperTestSuite) TestTotalEmissionPerMonthSimple() {
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
func (s *MintKeeperTestSuite) TestGetNumStakedTokensNonNegative() {
	s.stakingKeeper.EXPECT().TotalBondedTokens(s.ctx).Return(cosmosMath.NewInt(0), nil)
	s.emissionsKeeper.EXPECT().GetTotalStake(s.ctx).Return(cosmosMath.NewInt(0), nil)
	nst, err := keeper.GetNumStakedTokens(s.ctx, s.mintKeeper)
	s.Require().NoError(err)
	s.False(nst.IsNegative())
}

func (s *MintKeeperTestSuite) TestGetExponentialMovingAverageSimple() {
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

func (s *MintKeeperTestSuite) TestTargetRewardEmissionPerUnitStakedTokenSimple() {
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
func (s *MintKeeperTestSuite) TestEHatTargetFromCsv() {
	epoch := s.epochGet[60]
	epoch61 := s.epochGet[61]
	// because of how the simulator is written, the target is
	// calculated based on the previous epoch's data
	expectedResult := epoch61("ehat_target_i")

	simulatorFEmission := cosmosMath.LegacyMustNewDecFromStr("0.025")
	networkTokensTotal, err := epoch("network_tokens_total").SdkIntTrim()
	s.Require().NoError(err)
	ecosystemTokensTotal, err := epoch("ecosystem_tokens_total").SdkIntTrim()
	s.Require().NoError(err)
	networkTokensCirculating, err := epoch("network_tokens_circulating").SdkIntTrim()
	s.Require().NoError(err)
	networkTokensStaked, err := epoch("network_tokens_staked").SdkIntTrim()
	s.Require().NoError(err)
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
	testutil.InEpsilon5Dec(s.T(), resultD, expectedResult)
}

func (s *MintKeeperTestSuite) TestEHatMaxAtGenesisFromCsv() {
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
	testutil.InEpsilon5Dec(s.T(), resultD, expectedResult)
}

func (s *MintKeeperTestSuite) TestEhatIFromCsv() {
	epoch := s.epochGet[61]
	expectedResult := epoch("ehat_i")
	ehatMaxI, err := epoch("ehat_max_i").SdkLegacyDec()
	s.Require().NoError(err)
	ehatTargetI, err := epoch("ehat_target_i").SdkLegacyDec()
	s.Require().NoError(err)

	result := keeper.GetCappedTargetEmissionPerUnitStakedToken(
		ehatTargetI,
		ehatMaxI,
	)
	resultD, err := alloraMath.NewDecFromSdkLegacyDec(result)
	s.Require().NoError(err)
	testutil.InEpsilon5Dec(s.T(), resultD, expectedResult)
}

// calculate e_i for the 61st epoch
func (s *MintKeeperTestSuite) TestESubIFromCsv() {
	expectedResult := s.epochGet[61]("e_i")
	targetE_i, err := s.epochGet[61]("ehat_target_i").SdkLegacyDec()
	s.Require().NoError(err)
	previousE_i, err := s.epochGet[60]("e_i").SdkLegacyDec()
	s.Require().NoError(err)

	// this is taken directly from the python notebook
	alpha_Emission := cosmosMath.LegacyMustNewDecFromStr("0.1")

	result := keeper.GetExponentialMovingAverage(
		targetE_i,
		alpha_Emission,
		previousE_i,
	)
	resultD, err := alloraMath.NewDecFromSdkLegacyDec(result)
	s.Require().NoError(err)
	testutil.InEpsilon5Dec(s.T(), resultD, expectedResult)
}

// calculate \cal E for the 61st epoch
// GetTotalEmissionPerMonth
func (s *MintKeeperTestSuite) TestCalEFromCsv() {
	expectedResult := s.epochGet[61]("ecosystem_tokens_emission")
	rewardEmissionPerUnitStakedToken, err := s.epochGet[61]("e_i").SdkLegacyDec()
	s.Require().NoError(err)
	// use the value from epoch 60 rather than 61 because the python notebook
	// updates the value AFTER calculating the total emission and handing out rewards
	numStakedTokens, err := s.epochGet[60]("network_tokens_staked").SdkIntTrim()
	s.Require().NoError(err)
	totalEmission := keeper.GetTotalEmissionPerMonth(
		rewardEmissionPerUnitStakedToken,
		numStakedTokens,
	)
	resultD, err := alloraMath.NewDecFromSdkInt(totalEmission)
	s.Require().NoError(err)
	testutil.InEpsilon5Dec(s.T(), resultD, expectedResult)
}

// ============================================================================
// COMPREHENSIVE TESTS FOR GetLockedVestingTokensNew
// ============================================================================

func generateNewMintParams() types.Params {
	params := types.DefaultParams()
	params.EcosystemTreasuryPercentOfTotalSupply = cosmosMath.LegacyMustNewDecFromStr("0.2145")
	params.FoundationTreasuryPercentOfTotalSupply = cosmosMath.LegacyMustNewDecFromStr("0.177")
	params.ParticipantsPercentOfTotalSupply = cosmosMath.LegacyMustNewDecFromStr("0.123")
	params.InvestorsPercentOfTotalSupply = cosmosMath.LegacyMustNewDecFromStr("0.3105")
	params.InvestorsPreseedPercentOfTotalSupply = cosmosMath.LegacyMustNewDecFromStr("0")
	params.TeamPercentOfTotalSupply = cosmosMath.LegacyMustNewDecFromStr("0.175")
	params.FEmission = cosmosMath.LegacyMustNewDecFromStr("0.035")
	return params
}

// TestGetLockedVestingTokensNewBasicFunctionality tests basic functionality at different time points
func (s *MintKeeperTestSuite) TestGetLockedVestingTokensNewBasicFunctionality() {
	params := generateNewMintParams()
	blocksPerMonth := uint64(525960) // ~1 month in blocks

	// Test at genesis (month 0)
	blockHeight := cosmosMath.ZeroInt()
	monthsUnlocked := cosmosMath.ZeroInt()

	totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, updatedMonths, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)

	expectedPreseed := params.InvestorsPreseedPercentOfTotalSupply.Mul(params.MaxSupply.ToLegacyDec()).TruncateInt()
	expectedInvestors := params.InvestorsPercentOfTotalSupply.Mul(params.MaxSupply.ToLegacyDec()).TruncateInt()
	expectedTeam := params.TeamPercentOfTotalSupply.Mul(params.MaxSupply.ToLegacyDec()).TruncateInt()
	expectedFoundation := params.FoundationTreasuryPercentOfTotalSupply.Mul(keeper.FoundationInitialLockedPercentage).Mul(params.MaxSupply.ToLegacyDec()).TruncateInt()
	expectedParticipants := cosmosMath.ZeroInt() // Participants are unlocked from the start
	expectedTotal := expectedPreseed.Add(expectedInvestors).Add(expectedTeam).Add(expectedFoundation).Add(expectedParticipants)

	s.mintKeeper.Logger(s.ctx).Debug("--------------------------------")
	s.mintKeeper.Logger(s.ctx).Debug("Test at genesis (month 0)")
	s.mintKeeper.Logger(s.ctx).Debug("Total locked", "value", totalLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Preseed locked", "value", preseedLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Investors locked", "value", investorsLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Team locked", "value", teamLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Foundation locked", "value", foundationLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Participants locked", "value", participantsLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Updated months", "value", updatedMonths.String())
	// At genesis, all tokens should be locked except what's unlocked at TGE
	s.Require().Equal(totalLocked, expectedTotal, "Total locked should be equal to the sum of all locked tokens at genesis")
	s.Require().Equal(preseedLocked, expectedPreseed, "Preseed should be locked at genesis")
	s.Require().Equal(investorsLocked, expectedInvestors, "Investors should be locked at genesis")
	s.Require().Equal(teamLocked, expectedTeam, "Team should be locked at genesis")
	s.Require().Equal(foundationLocked, expectedFoundation, "Foundation should be locked at genesis")
	s.Require().Equal(participantsLocked, expectedParticipants, "Participants should be unlocked at genesis")
	s.Require().Equal(cosmosMath.ZeroInt(), updatedMonths, "Months unlocked should be 0 at genesis")

	// Test after 11 months (month 11) - preseed, investors, team should start vesting
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 11)
	monthsUnlocked = cosmosMath.NewInt(11)

	totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, updatedMonths, err =
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)

	s.Require().True(expectedPreseed.Equal(cosmosMath.ZeroInt()) || preseedLocked.GT(cosmosMath.ZeroInt()), "Preseed should be locked at month 11")
	s.Require().True(expectedInvestors.Equal(cosmosMath.ZeroInt()) || investorsLocked.GT(cosmosMath.ZeroInt()), "Investors should be locked at month 11")
	s.Require().True(expectedTeam.Equal(cosmosMath.ZeroInt()) || teamLocked.GT(cosmosMath.ZeroInt()), "Team should be locked at month 11")
	s.Require().True(expectedFoundation.Equal(cosmosMath.ZeroInt()) || foundationLocked.GT(cosmosMath.ZeroInt()), "Foundation should be locked at month 11")
	s.Require().True(expectedParticipants.Equal(cosmosMath.ZeroInt()) || participantsLocked.Equal(cosmosMath.ZeroInt()), "Participants should be unlocked at month 11")
	s.Require().True(expectedTotal.Equal(cosmosMath.ZeroInt()) || totalLocked.GT(cosmosMath.ZeroInt()), "Total locked should be > 0 at month 11")
	s.Require().Equal(cosmosMath.NewInt(11), updatedMonths, "Months unlocked should be 11 at month 11")

	// Test at month 12. The investors, preseed, and team cliff is month 18,
	// so these buckets are still fully locked. Foundation has vested for 12 of 24 months.
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 12)
	monthsUnlocked = cosmosMath.NewInt(12)

	totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, updatedMonths, err =
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)

	expectedPreseed = params.InvestorsPreseedPercentOfTotalSupply.Mul(params.MaxSupply.ToLegacyDec()).TruncateInt()
	expectedInvestors = params.InvestorsPercentOfTotalSupply.Mul(params.MaxSupply.ToLegacyDec()).TruncateInt()
	expectedTeam = params.TeamPercentOfTotalSupply.Mul(params.MaxSupply.ToLegacyDec()).TruncateInt()
	expectedFoundation = params.FoundationTreasuryPercentOfTotalSupply.Mul(keeper.FoundationInitialLockedPercentage).Mul(params.MaxSupply.ToLegacyDec()).TruncateInt().Mul(cosmosMath.NewInt(12)).Quo(cosmosMath.NewInt(24))
	expectedParticipants = cosmosMath.ZeroInt()

	s.mintKeeper.Logger(s.ctx).Debug("--------------------------------")
	s.mintKeeper.Logger(s.ctx).Debug("Test at 12 months")
	s.mintKeeper.Logger(s.ctx).Debug("Total locked", "value", totalLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Preseed locked", "actual", preseedLocked.String(), "expected", expectedPreseed.String())
	s.mintKeeper.Logger(s.ctx).Debug("Investors locked", "actual", investorsLocked.String(), "expected", expectedInvestors.String())
	s.mintKeeper.Logger(s.ctx).Debug("Team locked", "actual", teamLocked.String(), "expected", expectedTeam.String())
	s.mintKeeper.Logger(s.ctx).Debug("Foundation locked", "actual", foundationLocked.String(), "expected", expectedFoundation.String())
	s.mintKeeper.Logger(s.ctx).Debug("Participants locked", "actual", participantsLocked.String(), "expected", expectedParticipants.String())
	s.mintKeeper.Logger(s.ctx).Debug("Updated months", "value", updatedMonths.String())
	s.Require().Equal(preseedLocked, expectedPreseed, "Preseed should still be fully locked at month 12")
	s.Require().Equal(investorsLocked, expectedInvestors, "Investors should still be fully locked at month 12")
	s.Require().Equal(teamLocked, expectedTeam, "Team should still be fully locked at month 12")
	s.Require().Equal(participantsLocked, expectedParticipants, "Participants should be fully unlocked at month 12")
	s.Require().Equal(foundationLocked, expectedFoundation, "Foundation should still be locked at month 12")
	s.Require().Equal(cosmosMath.NewInt(12), updatedMonths, "Months unlocked should be 12")

	// Month 18 is the investors, preseed, and team cliff. (36-18)/36 leaves half of each
	// bucket locked. Foundation is on its own 24-month line, with 6 months left.
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 18)
	monthsUnlocked = cosmosMath.NewInt(18)

	totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, updatedMonths, err =
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)

	expectedPreseed = params.InvestorsPreseedPercentOfTotalSupply.Mul(params.MaxSupply.ToLegacyDec()).TruncateInt().Mul(cosmosMath.NewInt(18)).Quo(cosmosMath.NewInt(36))
	expectedInvestors = params.InvestorsPercentOfTotalSupply.Mul(params.MaxSupply.ToLegacyDec()).TruncateInt().Mul(cosmosMath.NewInt(18)).Quo(cosmosMath.NewInt(36))
	expectedTeam = params.TeamPercentOfTotalSupply.Mul(params.MaxSupply.ToLegacyDec()).TruncateInt().Mul(cosmosMath.NewInt(18)).Quo(cosmosMath.NewInt(36))
	expectedFoundation = params.FoundationTreasuryPercentOfTotalSupply.Mul(keeper.FoundationInitialLockedPercentage).Mul(params.MaxSupply.ToLegacyDec()).TruncateInt().Mul(cosmosMath.NewInt(6)).Quo(cosmosMath.NewInt(24))
	expectedParticipants = cosmosMath.ZeroInt()

	s.Require().Equal(expectedPreseed, preseedLocked, "Preseed should be half locked at month 18")
	s.Require().Equal(expectedInvestors, investorsLocked, "Investors should be half locked at month 18")
	s.Require().Equal(expectedTeam, teamLocked, "Team should be half locked at month 18")
	s.Require().Equal(expectedParticipants, participantsLocked, "Participants should be fully unlocked at month 18")
	s.Require().Equal(expectedFoundation, foundationLocked, "Foundation should have 6 of 24 months left at month 18")
	s.Require().Equal(cosmosMath.NewInt(18), updatedMonths, "Months unlocked should be 18")

	// Month 24: one third of investors, preseed, and team remains locked. Foundation is fully unlocked.
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 24)
	monthsUnlocked = cosmosMath.NewInt(24)

	totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, updatedMonths, err =
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	expectedPreseed = params.InvestorsPreseedPercentOfTotalSupply.Mul(params.MaxSupply.ToLegacyDec()).TruncateInt().Mul(cosmosMath.NewInt(12)).Quo(cosmosMath.NewInt(36))
	expectedInvestors = params.InvestorsPercentOfTotalSupply.Mul(params.MaxSupply.ToLegacyDec()).TruncateInt().Mul(cosmosMath.NewInt(12)).Quo(cosmosMath.NewInt(36))
	expectedTeam = params.TeamPercentOfTotalSupply.Mul(params.MaxSupply.ToLegacyDec()).TruncateInt().Mul(cosmosMath.NewInt(12)).Quo(cosmosMath.NewInt(36))

	s.mintKeeper.Logger(s.ctx).Debug("--------------------------------")
	s.mintKeeper.Logger(s.ctx).Debug("Test at 24 months")
	s.mintKeeper.Logger(s.ctx).Debug("Total locked", "value", totalLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Preseed locked", "value", preseedLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Investors locked", "value", investorsLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Team locked", "value", teamLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Foundation locked", "value", foundationLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Participants locked", "value", participantsLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Updated months", "value", updatedMonths.String())
	s.Require().True(preseedLocked.Equal(expectedPreseed), "Preseed should be 2/3 unlocked at month 24")
	s.Require().True(investorsLocked.Equal(expectedInvestors), "Investors should be 2/3  unlocked at month 24")
	s.Require().True(teamLocked.Equal(expectedTeam), "Team should be 2/3 unlocked at month 24")
	s.Require().True(participantsLocked.Equal(cosmosMath.ZeroInt()), "Participants should be fully unlocked at month 12")
	s.Require().True(foundationLocked.Equal(cosmosMath.ZeroInt()), "Foundation should be fully unlocked at month 24")
	s.Require().Equal(cosmosMath.NewInt(24), updatedMonths, "Months unlocked should be 24")

	// Test after 3 years (month 36) - preseed, investors, team should be fully unlocked
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 36)
	monthsUnlocked = cosmosMath.NewInt(36)

	totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, updatedMonths, err =
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)

	s.mintKeeper.Logger(s.ctx).Debug("--------------------------------")
	s.mintKeeper.Logger(s.ctx).Debug("Test at 36 months")
	s.mintKeeper.Logger(s.ctx).Debug("Total locked", "value", totalLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Preseed locked", "value", preseedLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Investors locked", "value", investorsLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Team locked", "value", teamLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Foundation locked", "value", foundationLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Participants locked", "value", participantsLocked.String())
	s.mintKeeper.Logger(s.ctx).Debug("Updated months", "value", updatedMonths.String())
	s.Require().Equal(preseedLocked, cosmosMath.ZeroInt(), "Preseed should be fully unlocked at month 36")
	s.Require().Equal(investorsLocked, cosmosMath.ZeroInt(), "Investors should be fully unlocked at month 36")
	s.Require().Equal(teamLocked, cosmosMath.ZeroInt(), "Team should be fully unlocked at month 36")
	s.Require().Equal(participantsLocked, cosmosMath.ZeroInt(), "Participants should be fully unlocked at month 12")
	s.Require().Equal(foundationLocked, cosmosMath.ZeroInt(), "Foundation should still be locked at month 24")
	s.Require().Equal(cosmosMath.NewInt(36), updatedMonths, "Months unlocked should be 36")

	// Test after 7 years (month 84) - everything should be unlocked
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 84)
	monthsUnlocked = cosmosMath.NewInt(84)

	totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, updatedMonths, err =
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)

	s.Require().True(totalLocked.Equal(cosmosMath.ZeroInt()), "All tokens should be unlocked at month 84")
	s.Require().True(preseedLocked.Equal(cosmosMath.ZeroInt()), "Preseed should be fully unlocked at month 84")
	s.Require().True(investorsLocked.Equal(cosmosMath.ZeroInt()), "Investors should be fully unlocked at month 84")
	s.Require().True(teamLocked.Equal(cosmosMath.ZeroInt()), "Team should be fully unlocked at month 84")
	s.Require().True(foundationLocked.Equal(cosmosMath.ZeroInt()), "Foundation should be fully unlocked at month 84")
	s.Require().True(participantsLocked.Equal(cosmosMath.ZeroInt()), "Participants should be fully unlocked at month 84")
	s.Require().Equal(cosmosMath.NewInt(36), updatedMonths, "Months unlocked should be 36 because it got clamped to 36")

	// Test after 8 years (month 96) - everything should be unlocked, but updatedMonths should be 84
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 96)
	monthsUnlocked = cosmosMath.NewInt(96)

	totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, updatedMonths, err =
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	s.Require().True(totalLocked.Equal(cosmosMath.ZeroInt()), "All tokens should be unlocked at month 96")
	s.Require().True(preseedLocked.Equal(cosmosMath.ZeroInt()), "Preseed should be fully unlocked at month 96")
	s.Require().True(investorsLocked.Equal(cosmosMath.ZeroInt()), "Investors should be fully unlocked at month 96")
	s.Require().True(teamLocked.Equal(cosmosMath.ZeroInt()), "Team should be fully unlocked at month 96")
	s.Require().True(foundationLocked.Equal(cosmosMath.ZeroInt()), "Foundation should be fully unlocked at month 96")
	s.Require().True(participantsLocked.Equal(cosmosMath.ZeroInt()), "Participants should be fully unlocked at month 96")
	s.Require().Equal(cosmosMath.NewInt(36), updatedMonths, "Months unlocked should be 36 because it got clamped to 36")
}

// TestGetLockedVestingTokensNewEdgeCases tests edge cases and boundary conditions
func (s *MintKeeperTestSuite) TestGetLockedVestingTokensNewEdgeCases() {
	params := generateNewMintParams()
	blocksPerMonth := uint64(525960)

	// Test with zero block height
	blockHeight := cosmosMath.ZeroInt()
	monthsUnlocked := cosmosMath.ZeroInt()

	totalLocked, _, _, _, _, _, updatedMonths, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	s.Require().True(totalLocked.GT(cosmosMath.ZeroInt()), "Should have locked tokens at zero block height")
	s.Require().Equal(cosmosMath.ZeroInt(), updatedMonths, "Months unlocked should be 0 at zero block height")

	// Test with very large block height (beyond 84 months)
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 100)
	monthsUnlocked = cosmosMath.ZeroInt()

	totalLocked, _, _, _, _, _, updatedMonths, err =
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	s.Require().True(totalLocked.Equal(cosmosMath.ZeroInt()), "All tokens should be unlocked beyond 84 months")
	s.Require().Equal(cosmosMath.NewInt(36), updatedMonths, "Months unlocked should be clamped to 36")

	// Test with zero blocks per month (should not panic)
	blockHeight = cosmosMath.NewInt(1000)
	monthsUnlocked = cosmosMath.ZeroInt()

	// This should handle division by zero gracefully
	totalLocked, _, _, _, _, _, updatedMonths, err =
		keeper.GetLockedVestingTokens(0, blockHeight, params, monthsUnlocked)
	s.Require().Error(err, "Should error with zero blocks per month")
	s.Require().Equal(cosmosMath.ZeroInt(), updatedMonths, "Months unlocked should be 0 on error")
	s.Require().True(totalLocked.Equal(cosmosMath.ZeroInt()), "Total locked should be zero on error")

	// Test with negative months already unlocked (should be handled gracefully)
	blockHeight = cosmosMath.NewInt(1000)
	monthsUnlocked = cosmosMath.NewInt(-1)

	totalLocked, _, _, _, _, _, updatedMonths, err =
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	s.Require().True(totalLocked.GT(cosmosMath.ZeroInt()), "Should handle negative months unlocked")
	s.Require().True(updatedMonths.GTE(cosmosMath.ZeroInt()), "Updated months should be non-negative")
}

// TestGetLockedVestingTokensNewVestingSchedules tests each vesting category individually
func (s *MintKeeperTestSuite) TestGetLockedVestingTokensNewVestingSchedules() {
	params := generateNewMintParams()
	blocksPerMonth := uint64(525960)

	// Test Foundation vesting (24 months)
	foundationVestingTests := []struct {
		months         int64
		shouldBeLocked bool
		description    string
	}{
		{0, true, "Foundation should be locked at month 0"},
		{12, true, "Foundation should be locked at month 12"},
		{24, false, "Foundation should be unlocked at month 24"},
		{36, false, "Foundation should be unlocked at month 36"},
	}

	for _, test := range foundationVestingTests {
		blockHeight := cosmosMath.NewIntFromUint64(blocksPerMonth * uint64(test.months))
		monthsUnlocked := cosmosMath.NewInt(test.months)

		_, _, _, _, foundationLocked, _, _, err :=
			keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
		s.Require().NoError(err)
		if test.shouldBeLocked {
			s.Require().True(foundationLocked.GT(cosmosMath.ZeroInt()), test.description)
		} else {
			s.Require().True(foundationLocked.Equal(cosmosMath.ZeroInt()), test.description)
		}
	}

	// Test Participants vesting - participants are unlocked from the start
	participantsVestingTests := []struct {
		months         int64
		shouldBeLocked bool
		description    string
	}{
		{0, false, "Participants should be unlocked at month 0"},
		{6, false, "Participants should be unlocked at month 6"},
		{12, false, "Participants should be unlocked at month 12"},
		{24, false, "Participants should be unlocked at month 24"},
	}

	for _, test := range participantsVestingTests {
		blockHeight := cosmosMath.NewIntFromUint64(blocksPerMonth * uint64(test.months))
		monthsUnlocked := cosmosMath.NewInt(test.months)

		_, _, _, _, _, participantsLocked, _, err :=
			keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
		s.Require().NoError(err)
		if test.shouldBeLocked {
			s.Require().True(participantsLocked.GT(cosmosMath.ZeroInt()), test.description)
		} else {
			s.Require().True(participantsLocked.Equal(cosmosMath.ZeroInt()), test.description)
		}
	}

	// Investors, preseed, and team: fully locked through month 17, half released
	// at month 18, fully unlocked at month 36.
	cliffVestingTests := []struct {
		months         int64
		shouldBeLocked bool
		description    string
	}{
		{0, true, "Cliff tokens should be locked at month 0"},
		{6, true, "Cliff tokens should be locked at month 6"},
		{12, true, "Cliff tokens should still be fully locked at month 12"},
		{17, true, "Cliff tokens should still be fully locked at month 17"},
		{18, true, "Cliff tokens should be partially locked at month 18"},
		{24, true, "Cliff tokens should be partially locked at month 24"},
		{36, false, "Cliff tokens should be unlocked at month 36"},
		{48, false, "Cliff tokens should be unlocked at month 48"},
	}

	for _, test := range cliffVestingTests {
		blockHeight := cosmosMath.NewIntFromUint64(blocksPerMonth * uint64(test.months))
		monthsUnlocked := cosmosMath.NewInt(test.months)

		_, preseedLocked, investorsLocked, teamLocked, _, _, _, err :=
			keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
		s.Require().NoError(err)
		if test.shouldBeLocked {
			s.Require().True(preseedLocked.GT(cosmosMath.ZeroInt()) || investorsLocked.GT(cosmosMath.ZeroInt()) || teamLocked.GT(cosmosMath.ZeroInt()), test.description)
		} else {
			s.Require().True(preseedLocked.Equal(cosmosMath.ZeroInt()) && investorsLocked.Equal(cosmosMath.ZeroInt()) && teamLocked.Equal(cosmosMath.ZeroInt()), test.description)
		}
	}
}

// TestGetLockedVestingTokensNewMathematicalPrecision tests mathematical calculations
func (s *MintKeeperTestSuite) TestGetLockedVestingTokensNewMathematicalPrecision() {
	params := generateNewMintParams()
	blocksPerMonth := uint64(525960)

	// Test at month 6 - participants are unlocked from the start
	blockHeight := cosmosMath.NewIntFromUint64(blocksPerMonth * 6)
	monthsUnlocked := cosmosMath.NewInt(6)

	_, _, _, _, _, participantsLocked, _, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	// Participants are unlocked from the start, so always zero
	expectedParticipantsLocked := cosmosMath.ZeroInt()

	s.Require().True(participantsLocked.Equal(expectedParticipantsLocked),
		"Participants locked should be zero at month 6. Expected: %s, Got: %s",
		expectedParticipantsLocked, participantsLocked)

	// Test at month 18 - foundation should be 25% unlocked (18/24)
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 18)
	monthsUnlocked = cosmosMath.NewInt(18)

	_, _, _, _, foundationLocked, _, _, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	// Foundation: 88.5% of 10% locked initially, should be 25% unlocked at month 18
	expectedFoundationLocked := params.FoundationTreasuryPercentOfTotalSupply.
		Mul(keeper.FoundationInitialLockedPercentage).
		Mul(params.MaxSupply.ToLegacyDec()).TruncateInt().
		Mul(cosmosMath.NewInt(6)).Quo(cosmosMath.NewInt(24))

	s.Require().True(foundationLocked.Equal(expectedFoundationLocked),
		"Foundation locked should be 25% at month 18. Expected: %s, Got: %s",
		expectedFoundationLocked, foundationLocked)

	// Month 24 is six months into the linear tail: (36-24)/36 = 1/3 of each bucket stays locked.
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 24)
	monthsUnlocked = cosmosMath.NewInt(24)

	_, preseedLocked, investorsLocked, teamLocked, _, _, _, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	// Should be 1/3 locked (24 months remaining out of 36 total)
	expectedPreseedLocked := params.InvestorsPreseedPercentOfTotalSupply.
		Mul(params.MaxSupply.ToLegacyDec()).TruncateInt().
		Mul(cosmosMath.NewInt(12)).Quo(cosmosMath.NewInt(36))
	expectedInvestorsLocked := params.InvestorsPercentOfTotalSupply.
		Mul(params.MaxSupply.ToLegacyDec()).TruncateInt().
		Mul(cosmosMath.NewInt(12)).Quo(cosmosMath.NewInt(36))
	expectedTeamLocked := params.TeamPercentOfTotalSupply.
		Mul(params.MaxSupply.ToLegacyDec()).TruncateInt().
		Mul(cosmosMath.NewInt(12)).Quo(cosmosMath.NewInt(36))

	s.Require().True(preseedLocked.Equal(expectedPreseedLocked),
		"Preseed locked should be 1/3 at month 24. Expected: %s, Got: %s",
		expectedPreseedLocked, preseedLocked)
	s.Require().True(investorsLocked.Equal(expectedInvestorsLocked),
		"Investors locked should be 1/3 at month 24. Expected: %s, Got: %s",
		expectedInvestorsLocked, investorsLocked)
	s.Require().True(teamLocked.Equal(expectedTeamLocked),
		"Team locked should be 1/3 at month 24. Expected: %s, Got: %s",
		expectedTeamLocked, teamLocked)
}

// TestGetLockedVestingTokensNewMonotonicity tests that locked amounts decrease monotonically
func (s *MintKeeperTestSuite) TestGetLockedVestingTokensNewMonotonicity() {
	params := generateNewMintParams()
	blocksPerMonth := uint64(525960)

	// Test that total locked decreases monotonically over time
	// Use a very large initial value instead of math.MaxUint64
	// Calculate the maximum possible locked amount (100% of max supply)
	maxPossibleLocked := params.MaxSupply
	previousTotal := maxPossibleLocked
	previousPreseed := maxPossibleLocked
	previousInvestors := maxPossibleLocked
	previousTeam := maxPossibleLocked
	previousFoundation := maxPossibleLocked
	previousParticipants := maxPossibleLocked

	for month := int64(0); month <= 100; month++ {
		blockHeight := cosmosMath.NewIntFromUint64(blocksPerMonth * uint64(month))
		monthsUnlocked := cosmosMath.NewInt(month)

		totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, _, err :=
			keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
		s.Require().NoError(err)
		// Total should never increase
		s.Require().True(totalLocked.LTE(previousTotal),
			"Total locked should never increase. Month %d: %s > %s",
			month, totalLocked, previousTotal)

		// Individual categories should never increase
		s.Require().True(preseedLocked.LTE(previousPreseed),
			"Preseed locked should never increase. Month %d: %s > %s",
			month, preseedLocked, previousPreseed)
		s.Require().True(investorsLocked.LTE(previousInvestors),
			"Investors locked should never increase. Month %d: %s > %s",
			month, investorsLocked, previousInvestors)
		s.Require().True(teamLocked.LTE(previousTeam),
			"Team locked should never increase. Month %d: %s > %s",
			month, teamLocked, previousTeam)
		s.Require().True(foundationLocked.LTE(previousFoundation),
			"Foundation locked should never increase. Month %d: %s > %s",
			month, foundationLocked, previousFoundation)
		s.Require().True(participantsLocked.LTE(previousParticipants),
			"Participants locked should never increase. Month %d: %s > %s",
			month, participantsLocked, previousParticipants)

		previousTotal = totalLocked
		previousPreseed = preseedLocked
		previousInvestors = investorsLocked
		previousTeam = teamLocked
		previousFoundation = foundationLocked
		previousParticipants = participantsLocked
	}
}

// TestGetLockedVestingTokensNewBoundaryConditions tests exact transition points
func (s *MintKeeperTestSuite) TestGetLockedVestingTokensNewBoundaryConditions() {
	params := generateNewMintParams()
	blocksPerMonth := uint64(525960)

	// Test exactly at month 12 (participants should be fully unlocked)
	blockHeight := cosmosMath.NewIntFromUint64(blocksPerMonth * 12)
	monthsUnlocked := cosmosMath.NewInt(12)

	_, _, _, _, _, participantsLocked, _, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	s.Require().True(participantsLocked.Equal(cosmosMath.ZeroInt()),
		"Participants should be exactly unlocked at month 12")

	// Test exactly at month 24 (foundation should be fully unlocked)
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 24)
	monthsUnlocked = cosmosMath.NewInt(24)

	_, _, _, _, foundationLocked, _, _, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	s.Require().True(foundationLocked.Equal(cosmosMath.ZeroInt()),
		"Foundation should be exactly unlocked at month 24")

	// Test exactly at month 36 (cliff tokens should be fully unlocked)
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 36)
	monthsUnlocked = cosmosMath.NewInt(36)

	_, preseedLocked, investorsLocked, teamLocked, _, _, _, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	s.Require().True(preseedLocked.Equal(cosmosMath.ZeroInt()),
		"Preseed should be exactly unlocked at month 36")
	s.Require().True(investorsLocked.Equal(cosmosMath.ZeroInt()),
		"Investors should be exactly unlocked at month 36")
	s.Require().True(teamLocked.Equal(cosmosMath.ZeroInt()),
		"Team should be exactly unlocked at month 36")

	// Test exactly at month 84 (everything should be unlocked)
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 84)
	monthsUnlocked = cosmosMath.NewInt(84)

	totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, _, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	s.Require().True(totalLocked.Equal(cosmosMath.ZeroInt()),
		"Total should be exactly zero at month 84")
	s.Require().True(preseedLocked.Equal(cosmosMath.ZeroInt()),
		"Preseed should be exactly zero at month 84")
	s.Require().True(investorsLocked.Equal(cosmosMath.ZeroInt()),
		"Investors should be exactly zero at month 84")
	s.Require().True(teamLocked.Equal(cosmosMath.ZeroInt()),
		"Team should be exactly zero at month 84")
	s.Require().True(foundationLocked.Equal(cosmosMath.ZeroInt()),
		"Foundation should be exactly zero at month 84")
	s.Require().True(participantsLocked.Equal(cosmosMath.ZeroInt()),
		"Participants should be exactly zero at month 84")
}

// TestGetLockedVestingTokensNewParameterValidation tests with different parameter sets
func (s *MintKeeperTestSuite) TestGetLockedVestingTokensNewParameterValidation() {
	blocksPerMonth := uint64(525960)
	blockHeight := cosmosMath.NewIntFromUint64(blocksPerMonth * 12) // Month 12
	monthsUnlocked := cosmosMath.NewInt(12)

	// Test with zero max supply
	zeroSupplyParams := generateNewMintParams()
	zeroSupplyParams.MaxSupply = cosmosMath.ZeroInt()

	totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, _, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, zeroSupplyParams, monthsUnlocked)
	s.Require().NoError(err)
	s.Require().True(totalLocked.Equal(cosmosMath.ZeroInt()), "Total should be zero with zero max supply")
	s.Require().True(preseedLocked.Equal(cosmosMath.ZeroInt()), "Preseed should be zero with zero max supply")
	s.Require().True(investorsLocked.Equal(cosmosMath.ZeroInt()), "Investors should be zero with zero max supply")
	s.Require().True(teamLocked.Equal(cosmosMath.ZeroInt()), "Team should be zero with zero max supply")
	s.Require().True(foundationLocked.Equal(cosmosMath.ZeroInt()), "Foundation should be zero with zero max supply")
	s.Require().True(participantsLocked.Equal(cosmosMath.ZeroInt()), "Participants should be zero with zero max supply")

	// Test with very large max supply
	largeSupplyParams := generateNewMintParams()
	largeSupplyParams.MaxSupply = cosmosMath.NewIntFromUint64(math.MaxUint64)

	totalLocked, _, _, _, _, _, _, err =
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, largeSupplyParams, monthsUnlocked)
	s.Require().NoError(err)
	s.Require().True(totalLocked.GT(cosmosMath.ZeroInt()), "Should handle large max supply")
	s.Require().True(totalLocked.LT(largeSupplyParams.MaxSupply), "Total locked should be less than max supply")
	// Test with zero percentages (should result in zero locked amounts)
	zeroPercentParams := types.DefaultParams()
	zeroPercentParams.InvestorsPercentOfTotalSupply = cosmosMath.LegacyZeroDec()
	zeroPercentParams.InvestorsPreseedPercentOfTotalSupply = cosmosMath.LegacyZeroDec()
	zeroPercentParams.TeamPercentOfTotalSupply = cosmosMath.LegacyZeroDec()
	zeroPercentParams.FoundationTreasuryPercentOfTotalSupply = cosmosMath.LegacyZeroDec()
	zeroPercentParams.ParticipantsPercentOfTotalSupply = cosmosMath.LegacyZeroDec()

	totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, _, err =
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, zeroPercentParams, monthsUnlocked)
	s.Require().NoError(err)
	s.Require().True(totalLocked.Equal(cosmosMath.ZeroInt()), "Total should be zero with zero percentage")
	s.Require().True(preseedLocked.Equal(cosmosMath.ZeroInt()), "Preseed should be zero with zero percentage")
	s.Require().True(investorsLocked.Equal(cosmosMath.ZeroInt()), "Investors should be zero with zero percentage")
	s.Require().True(teamLocked.Equal(cosmosMath.ZeroInt()), "Team should be zero with zero percentage")
	s.Require().True(foundationLocked.Equal(cosmosMath.ZeroInt()), "Foundation should be zero with zero percentage")
	s.Require().True(participantsLocked.Equal(cosmosMath.ZeroInt()), "Participants should be zero with zero percentage")
}

// TestGetLockedVestingTokensNewConsistency tests consistency with different input combinations
func (s *MintKeeperTestSuite) TestGetLockedVestingTokensNewConsistency() {
	params := generateNewMintParams()
	blocksPerMonth := uint64(525960)

	// Test that monthsAlreadyUnlocked parameter is respected when it's higher than calculated
	blockHeight := cosmosMath.NewIntFromUint64(blocksPerMonth * 6) // Month 6
	monthsUnlocked := cosmosMath.NewInt(12)                        // But we say 12 months already unlocked

	_, _, _, _, _, participantsLocked1, updatedMonths1, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	// Should use the higher value (12 months)
	s.Require().Equal(cosmosMath.NewInt(12), updatedMonths1, "Should use higher months already unlocked")
	s.Require().True(participantsLocked1.Equal(cosmosMath.ZeroInt()), "Participants should be unlocked at 12 months")

	// Test that calculated months are used when they're higher than monthsAlreadyUnlocked
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 12) // Month 12
	monthsUnlocked = cosmosMath.NewInt(6)                          // But we say only 6 months unlocked

	totalLocked2, preseedLocked2, investorsLocked2, teamLocked2, foundationLocked2, participantsLocked2, updatedMonths2, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	// Should use the calculated value (12 months)
	s.Require().Equal(cosmosMath.NewInt(12), updatedMonths2, "Should use calculated months when higher")
	s.Require().True(participantsLocked2.Equal(cosmosMath.ZeroInt()), "Participants should be unlocked at 12 months")

	// Test that results are consistent when monthsAlreadyUnlocked equals calculated months
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth * 12) // Month 12
	monthsUnlocked = cosmosMath.NewInt(12)                         // Same as calculated

	totalLocked3, preseedLocked3, investorsLocked3, teamLocked3, foundationLocked3, participantsLocked3, updatedMonths3, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.NewInt(12), updatedMonths3, "Should use 12 months")
	s.Require().True(participantsLocked3.Equal(cosmosMath.ZeroInt()), "Participants should be unlocked at 12 months")

	// Results should be identical
	s.Require().True(totalLocked2.Equal(totalLocked3), "Results should be identical when months match")
	s.Require().True(preseedLocked2.Equal(preseedLocked3), "Preseed results should be identical")
	s.Require().True(investorsLocked2.Equal(investorsLocked3), "Investors results should be identical")
	s.Require().True(teamLocked2.Equal(teamLocked3), "Team results should be identical")
	s.Require().True(foundationLocked2.Equal(foundationLocked3), "Foundation results should be identical")
	s.Require().True(participantsLocked2.Equal(participantsLocked3), "Participants results should be identical")
}

// TestGetLockedVestingTokensNewParameterValidation tests with different parameter sets
func (s *MintKeeperTestSuite) TestGetLockedVestingTokensNewMonthsUnlockedClamping() {
	blocksPerMonth := uint64(525960)
	blockHeight := cosmosMath.NewIntFromUint64(blocksPerMonth * 37) // Month 12
	monthsUnlocked := cosmosMath.NewInt(36)

	// Test with zero max supply
	params := generateNewMintParams()
	_, _, _, _, _, _, updatedMonths, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	s.Require().Equal(cosmosMath.NewInt(36), updatedMonths, "Should clamp months to 36")
}

// TestGetLockedVestingTokensNewPrecisionAndRounding tests precision and rounding behavior
func (s *MintKeeperTestSuite) TestGetLockedVestingTokensNewPrecisionAndRounding() {
	params := generateNewMintParams()
	blocksPerMonth := uint64(525960)

	// Test that calculations are precise and don't accumulate rounding errors
	// Run the same calculation multiple times and ensure consistency
	blockHeight := cosmosMath.NewIntFromUint64(blocksPerMonth * 18) // Month 18
	monthsUnlocked := cosmosMath.NewInt(18)

	var previousTotal cosmosMath.Int
	for i := 0; i < 10; i++ {
		totalLocked, _, _, _, _, _, _, err :=
			keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
		s.Require().NoError(err)
		if i > 0 {
			s.Require().True(totalLocked.Equal(previousTotal),
				"Results should be identical across multiple calls. Call %d: %s != %s",
				i, totalLocked, previousTotal)
		}
		previousTotal = totalLocked
	}

	// Test with fractional months (should round down)
	blockHeight = cosmosMath.NewIntFromUint64(blocksPerMonth*18 + blocksPerMonth/2) // Month 18.5
	monthsUnlocked = cosmosMath.NewInt(18)

	totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, updatedMonths, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	// Should round down to 18 months
	s.Require().Equal(cosmosMath.NewInt(18), updatedMonths, "Should round down fractional months")

	// Results should be the same as month 18
	blockHeight18 := cosmosMath.NewIntFromUint64(blocksPerMonth * 18)
	totalLocked18, preseedLocked18, investorsLocked18, teamLocked18, foundationLocked18, participantsLocked18, _, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight18, params, monthsUnlocked)
	s.Require().NoError(err)
	s.Require().True(totalLocked.Equal(totalLocked18), "Fractional months should round down")
	s.Require().True(preseedLocked.Equal(preseedLocked18), "Preseed should be same for fractional months")
	s.Require().True(investorsLocked.Equal(investorsLocked18), "Investors should be same for fractional months")
	s.Require().True(teamLocked.Equal(teamLocked18), "Team should be same for fractional months")
	s.Require().True(foundationLocked.Equal(foundationLocked18), "Foundation should be same for fractional months")
	s.Require().True(participantsLocked.Equal(participantsLocked18), "Participants should be same for fractional months")
}

// TestGetLockedVestingTokensNewStressTest tests with extreme values
func (s *MintKeeperTestSuite) TestGetLockedVestingTokensNewStressTest() {
	params := generateNewMintParams()
	blocksPerMonth := uint64(525960)

	// Test with maximum possible block height
	maxBlockHeight := cosmosMath.NewIntFromUint64(math.MaxUint64)
	monthsUnlocked := cosmosMath.ZeroInt()

	totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, updatedMonths, err :=
		keeper.GetLockedVestingTokens(blocksPerMonth, maxBlockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	// Should clamp to 84 months and have zero locked
	s.Require().Equal(cosmosMath.NewInt(36), updatedMonths, "Should clamp to 36 months")
	s.Require().True(totalLocked.Equal(cosmosMath.ZeroInt()), "Should have zero locked at max block height")
	s.Require().True(preseedLocked.Equal(cosmosMath.ZeroInt()), "Preseed should be zero at max block height")
	s.Require().True(investorsLocked.Equal(cosmosMath.ZeroInt()), "Investors should be zero at max block height")
	s.Require().True(teamLocked.Equal(cosmosMath.ZeroInt()), "Team should be zero at max block height")
	s.Require().True(foundationLocked.Equal(cosmosMath.ZeroInt()), "Foundation should be zero at max block height")
	s.Require().True(participantsLocked.Equal(cosmosMath.ZeroInt()), "Participants should be zero at max block height")

	// Test with maximum possible months already unlocked
	blockHeight := cosmosMath.NewInt(1000)
	monthsUnlocked = cosmosMath.NewIntFromUint64(math.MaxUint64)

	totalLocked, preseedLocked, investorsLocked, teamLocked, foundationLocked, participantsLocked, updatedMonths, err =
		keeper.GetLockedVestingTokens(blocksPerMonth, blockHeight, params, monthsUnlocked)
	s.Require().NoError(err)
	// Should clamp to 36 months and have zero locked
	s.Require().Equal(cosmosMath.NewInt(36), updatedMonths, "Should clamp months to 36")
	s.Require().True(totalLocked.Equal(cosmosMath.ZeroInt()), "Should have zero locked with max months")
	s.Require().True(preseedLocked.Equal(cosmosMath.ZeroInt()), "Preseed should be zero with max months")
	s.Require().True(investorsLocked.Equal(cosmosMath.ZeroInt()), "Investors should be zero with max months")
	s.Require().True(teamLocked.Equal(cosmosMath.ZeroInt()), "Team should be zero with max months")
	s.Require().True(foundationLocked.Equal(cosmosMath.ZeroInt()), "Foundation should be zero with max months")
	s.Require().True(participantsLocked.Equal(cosmosMath.ZeroInt()), "Participants should be zero with max months")
}

// TestInvestorsTeamEighteenMonthCliff pins the investors, preseed, and team
// schedule: fully locked before month 18, half unlocked at month 18, and fully
// unlocked at month 36. Expected uallo amounts are literals, not a replay of
// the keeper formula.
func (s *MintKeeperTestSuite) TestInvestorsTeamEighteenMonthCliff() {
	blocksPerMonth := uint64(525960)
	params := generateNewMintParams()

	const (
		fullInvestors = "310500000000000000000000000"
		fullTeam      = "175000000000000000000000000"
	)

	steps := []struct {
		month         int64
		investors     string
		team          string
		foundation    string
		updatedMonths int64
	}{
		{0, fullInvestors, fullTeam, "88500000000000000000000000", 0},
		{11, fullInvestors, fullTeam, "47937500000000000000000000", 11},
		{12, fullInvestors, fullTeam, "44250000000000000000000000", 12},
		{17, fullInvestors, fullTeam, "25812500000000000000000000", 17},
		{18, "155250000000000000000000000", "87500000000000000000000000", "22125000000000000000000000", 18},
		{19, "146625000000000000000000000", "82638888888888888888888888", "18437500000000000000000000", 19},
		{24, "103500000000000000000000000", "58333333333333333333333333", "0", 24},
		{35, "8625000000000000000000000", "4861111111111111111111111", "0", 35},
		{36, "0", "0", "0", 36},
		{42, "0", "0", "0", 36},
		{48, "0", "0", "0", 36},
	}

	var previousInvestors cosmosMath.Int
	for _, tc := range steps {
		s.Run(fmt.Sprintf("month_%d", tc.month), func() {
			_, preseed, investors, team, foundation, participants, updated, err := keeper.GetLockedVestingTokens(
				blocksPerMonth,
				cosmosMath.NewInt(tc.month).Mul(cosmosMath.NewIntFromUint64(blocksPerMonth)),
				params,
				cosmosMath.NewInt(tc.month),
			)
			s.Require().NoError(err)
			s.Require().Equal("0", preseed.String(), "preseed")
			s.Require().Equal(tc.investors, investors.String(), "investors")
			s.Require().Equal(tc.team, team.String(), "team")
			s.Require().Equal(tc.foundation, foundation.String(), "foundation")
			s.Require().True(participants.IsZero(), "participants")
			s.Require().Equal(cosmosMath.NewInt(tc.updatedMonths).String(), updated.String(), "months")
			if tc.month > 0 {
				s.Require().False(investors.GT(previousInvestors), "locked investors must not increase")
			}
			previousInvestors = investors
		})
	}

	s.Run("every month from 0 through 48", func() {
		fullInv, ok := cosmosMath.NewIntFromString(fullInvestors)
		s.Require().True(ok)
		fullTm, ok := cosmosMath.NewIntFromString(fullTeam)
		s.Require().True(ok)
		var prev cosmosMath.Int
		for month := int64(0); month <= 48; month++ {
			_, _, investors, team, foundation, _, updated, err := keeper.GetLockedVestingTokens(
				blocksPerMonth,
				cosmosMath.NewInt(month).Mul(cosmosMath.NewIntFromUint64(blocksPerMonth)),
				params,
				cosmosMath.NewInt(month),
			)
			s.Require().NoError(err, "month %d", month)
			var wantInv, wantTeam cosmosMath.Int
			switch {
			case month < 18:
				wantInv, wantTeam = fullInv, fullTm
			case month < 36:
				remaining := cosmosMath.NewInt(36 - month)
				wantInv = fullInv.Mul(remaining).Quo(cosmosMath.NewInt(36))
				wantTeam = fullTm.Mul(remaining).Quo(cosmosMath.NewInt(36))
			default:
				wantInv, wantTeam = cosmosMath.ZeroInt(), cosmosMath.ZeroInt()
			}
			s.Require().Equal(wantInv.String(), investors.String(), "month %d investors", month)
			s.Require().Equal(wantTeam.String(), team.String(), "month %d team", month)
			if month >= 24 {
				s.Require().True(foundation.IsZero(), "month %d foundation", month)
			} else {
				s.Require().False(foundation.IsZero(), "month %d foundation", month)
			}
			wantMonths := month
			if wantMonths > 36 {
				wantMonths = 36
			}
			s.Require().Equal(cosmosMath.NewInt(wantMonths).String(), updated.String(), "month %d counter", month)
			if month > 0 {
				s.Require().False(investors.GT(prev), "month %d locked investors increased", month)
			}
			prev = investors
		}
	})

	s.Run("stored month wins over a lower block month", func() {
		_, _, investors, team, _, _, updated, err := keeper.GetLockedVestingTokens(
			blocksPerMonth,
			cosmosMath.NewInt(12).Mul(cosmosMath.NewIntFromUint64(blocksPerMonth)),
			params,
			cosmosMath.NewInt(17),
		)
		s.Require().NoError(err)
		s.Require().Equal(fullInvestors, investors.String())
		s.Require().Equal(fullTeam, team.String())
		s.Require().Equal(cosmosMath.NewInt(17).String(), updated.String())
	})

	s.Run("block month past the cliff uses the linear tail", func() {
		// Month 20 keeps 16/36 of each bucket locked.
		_, _, investors, team, _, _, updated, err := keeper.GetLockedVestingTokens(
			blocksPerMonth,
			cosmosMath.NewInt(20).Mul(cosmosMath.NewIntFromUint64(blocksPerMonth)),
			params,
			cosmosMath.ZeroInt(),
		)
		s.Require().NoError(err)
		s.Require().Equal("138000000000000000000000000", investors.String())
		s.Require().Equal("77777777777777777777777777", team.String())
		s.Require().Equal(cosmosMath.NewInt(20).String(), updated.String())
	})

	s.Run("preseed follows the same cliff", func() {
		p := generateNewMintParams()
		p.InvestorsPercentOfTotalSupply = cosmosMath.LegacyMustNewDecFromStr("0.2605")
		p.InvestorsPreseedPercentOfTotalSupply = cosmosMath.LegacyMustNewDecFromStr("0.05")

		_, preseed17, investors17, _, _, _, _, err := keeper.GetLockedVestingTokens(
			blocksPerMonth,
			cosmosMath.NewInt(17).Mul(cosmosMath.NewIntFromUint64(blocksPerMonth)),
			p,
			cosmosMath.NewInt(17),
		)
		s.Require().NoError(err)
		s.Require().Equal("50000000000000000000000000", preseed17.String())
		s.Require().Equal("260500000000000000000000000", investors17.String())

		_, preseed18, investors18, _, _, _, _, err := keeper.GetLockedVestingTokens(
			blocksPerMonth,
			cosmosMath.NewInt(18).Mul(cosmosMath.NewIntFromUint64(blocksPerMonth)),
			p,
			cosmosMath.NewInt(18),
		)
		s.Require().NoError(err)
		s.Require().Equal("25000000000000000000000000", preseed18.String())
		s.Require().Equal("130250000000000000000000000", investors18.String())
	})
}
