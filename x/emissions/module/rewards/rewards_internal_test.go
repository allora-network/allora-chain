package rewards_test

import (
	"log"
	"strconv"
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MathTestSuite struct {
	suite.Suite
}

func (s *MathTestSuite) SetupTest() {
}

func TestMathTestSuite(t *testing.T) {
	suite.Run(t, new(MathTestSuite))
}

func (s *MathTestSuite) TestAdjustedStakeSimple() {
	// for this example we use
	// 3 reputers with stakes of 50_000, 100_000, 150_000
	// listening coefficients of 0.25, 0.18, 0.63 for those reputers
	// and we calculate the adjusted stake for reputer 2 (the 100_000)

	stake := alloraMath.NewDecFromInt64(100000)
	allStakes := []alloraMath.Dec{alloraMath.NewDecFromInt64(50000), stake, alloraMath.NewDecFromInt64(150000)}
	listeningCoefficient := alloraMath.MustNewDecFromString("0.18")
	allListeningCoefficients := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.25"),
		listeningCoefficient,
		alloraMath.MustNewDecFromString("0.63"),
	}
	numReputers := alloraMath.NewDecFromInt64(3)

	// use wolfram alpha to calculate the expected result
	// https://www.wolframalpha.com/input?i2d=true&i=1-%5C%2840%29%5C%2840%29Power%5B%5C%2840%29Power%5B%5C%2840%29ln%5C%2840%291%2BPower%5Be%2C20%5D%5C%2841%29%5C%2841%29%2C1%5D%5C%2841%29%2C-1%5D%5C%2841%29*Power%5B%5C%2840%29ln%5C%2840%291%2BPower%5Be%2C%5C%2840%29-20%5C%2840%29Divide%5B3*0.18*100%5C%2844%29000%2C0.18*100%5C%2844%29000+%2B+0.25*50%5C%2844%29000+%2B+0.63*150%5C%2844%29000%5D+-+1%5C%2841%29%5C%2841%29%5D%5C%2841%29%5C%2841%29%2C1%5D%5C%2841%29
	expected := alloraMath.MustNewDecFromString("0.4319994174428689223916439092220111693737492607160554179509")

	result, err := rewards.GetAdjustedStake(
		stake,
		allStakes,
		listeningCoefficient,
		allListeningCoefficients,
		numReputers,
	)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(expected, result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestNormalizeAgainstSlice() {
	a := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("2.0"),
		alloraMath.MustNewDecFromString("3.0"),
		alloraMath.MustNewDecFromString("5.0"),
	}
	expected := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.2"),
		alloraMath.MustNewDecFromString("0.3"),
		alloraMath.MustNewDecFromString("0.5"),
	}

	result, err := rewards.ModifiedRewardFractions(a)

	s.Require().NoError(err)
	for i := range expected {
		s.Require().True(alloraMath.InDelta(expected[i], result[i], alloraMath.MustNewDecFromString("0.0001")))
	}
}

func (s *MathTestSuite) TestEntropySimple() {
	f_ij := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.2"),
		alloraMath.MustNewDecFromString("0.3"),
		alloraMath.MustNewDecFromString("0.5"),
	}
	N_i_eff := alloraMath.MustNewDecFromString("0.75")
	N_i := alloraMath.MustNewDecFromString("3.0")
	beta := alloraMath.MustNewDecFromString("0.25")

	// using wolfram alpha to get a sample result
	// https://www.wolframalpha.com/input?i2d=true&i=-Power%5B%5C%2840%29Divide%5B0.75%2C3%5D%5C%2841%29%2C0.25%5D*%5C%2840%290.2*ln%5C%2840%290.2%5C%2841%29+%2B+0.3*ln%5C%2840%290.3%5C%2841%29+%2B+0.5*ln%5C%2840%290.5%5C%2841%29%5C%2841%29
	expected := alloraMath.MustNewDecFromString("0.7280746285142275338742683350155248011115920866691059016669")
	result, err := rewards.Entropy(f_ij, N_i_eff, N_i, beta)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(expected, result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestNumberRatio() {
	rewardFractions := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.2"),
		alloraMath.MustNewDecFromString("0.3"),
		alloraMath.MustNewDecFromString("0.4"),
		alloraMath.MustNewDecFromString("0.5"),
		alloraMath.MustNewDecFromString("0.6"),
		alloraMath.MustNewDecFromString("0.7"),
	}

	// 1 / (0.2 *0.2 + 0.3*0.3 + 0.4*0.4 + 0.5*0.5 + 0.6*0.6 + 0.7*0.7)
	// 1 / (0.04 + 0.09 + 0.16 + 0.25 + 0.36 + 0.49)
	// 1 / 1.39 = 0.719424460431654676259005145797598627787307032590051458
	expected := alloraMath.MustNewDecFromString("0.719424460431654676259005145797598627787307032590051458")

	result, err := rewards.NumberRatio(rewardFractions)
	s.Require().NoError(err, "Error calculating number ratio")
	s.Require().True(alloraMath.InDelta(expected, result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestNumberRatioZeroFractions() {
	zeroFractions := []alloraMath.Dec{alloraMath.ZeroDec()}

	_, err := rewards.NumberRatio(zeroFractions)
	s.Require().ErrorIs(err, emissionstypes.ErrNumberRatioDivideByZero)
}

func (s *MathTestSuite) TestNumberRatioEmptyList() {
	emptyFractions := []alloraMath.Dec{}

	_, err := rewards.NumberRatio(emptyFractions)
	s.Require().ErrorIs(err, emissionstypes.ErrNumberRatioInvalidSliceLength)
}

func (s *MathTestSuite) TestInferenceRewardsSimple() {
	// T_i = log L naive - log L = 2 - 1 = 1
	// X = 0.5 when T_i >= 1
	// U_i = ((1 - 0.5) * 2 * 2 * 2 ) / (2 + 2 + 4)
	// U_i = 0.5 * 8 / 8
	// U_i = 0.5
	infererScores := []emissionstypes.Score{
		{Score: alloraMath.MustNewDecFromString("0.5")},
		{Score: alloraMath.MustNewDecFromString("0.5")},
	}
	previousForecasterScoreRatio := alloraMath.ZeroDec()
	alpha := alloraMath.OneDec()
	totalReward := alloraMath.MustNewDecFromString("2.0")
	chi, gamma, err := rewards.GetChiAndGamma(
		alloraMath.MustNewDecFromString("2"), // log10(L_i- (naive))
		alloraMath.MustNewDecFromString("1"), // log10(L_i (network))
		alloraMath.MustNewDecFromString("2.0"),
		alloraMath.MustNewDecFromString("2.0"),
		infererScores,
		previousForecasterScoreRatio,
		alpha,
	)
	s.Require().NoError(err)
	infRewards, err := rewards.GetRewardForInferenceTaskInTopic(
		alloraMath.MustNewDecFromString("2.0"), // F_i
		alloraMath.MustNewDecFromString("2.0"), // G_i
		alloraMath.MustNewDecFromString("4.0"), // H_i
		&totalReward,                           // E_i
		chi,
		gamma,
	)
	s.Require().NoError(err)
	expected := alloraMath.MustNewDecFromString("0.5")
	s.Require().True(
		alloraMath.InDelta(
			expected,
			infRewards,
			alloraMath.MustNewDecFromString("0.0001"),
		),
		"Expected ",
		expected.String(),
		" but got ",
		infRewards.String(),
	)
}

func (s *MathTestSuite) TestInferenceRewardsZero() {
	totalReward := alloraMath.ZeroDec()
	infererScores := []emissionstypes.Score{
		{Score: alloraMath.MustNewDecFromString("0.5")},
		{Score: alloraMath.MustNewDecFromString("0.5")},
	}
	previousForecasterScoreRatio := alloraMath.ZeroDec()
	alpha := alloraMath.OneDec()
	chi, gamma, err := rewards.GetChiAndGamma(
		alloraMath.MustNewDecFromString("2"), // log10(L_i- (naive))
		alloraMath.MustNewDecFromString("1"), // log10(L_i (network))
		alloraMath.MustNewDecFromString("2.0"),
		alloraMath.MustNewDecFromString("2.0"),
		infererScores,
		previousForecasterScoreRatio,
		alpha,
	)
	s.Require().NoError(err)
	result, err := rewards.GetRewardForInferenceTaskInTopic(
		alloraMath.MustNewDecFromString("2.0"), // F_i
		alloraMath.MustNewDecFromString("2.0"), // G_i
		alloraMath.MustNewDecFromString("4.0"), // H_i
		&totalReward,                           // E_i
		chi,
		gamma,
	)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.ZeroDec(), result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestForecastRewardsSimple() {
	// T_i = log L naive - log L = 2 - 1 = 1
	// X = 0.5 if T_i >= 1
	// V_i = (X * γ * G_i * E_i) / (F_i + G_i + H_i)
	// V_i = (0.5 * 2 * 2 * 2 ) / (2 + 2 + 4)
	// V_i = 0.5 * 8 / 8
	// V_i = 0.5
	infererScores := []emissionstypes.Score{
		{Score: alloraMath.MustNewDecFromString("0.5")},
		{Score: alloraMath.MustNewDecFromString("0.5")},
	}
	previousForecasterScoreRatio := alloraMath.ZeroDec()
	alpha := alloraMath.OneDec()
	totalReward := alloraMath.MustNewDecFromString("2.0")
	chi, gamma, err := rewards.GetChiAndGamma(
		alloraMath.MustNewDecFromString("2"), // log10(L_i- (naive))
		alloraMath.MustNewDecFromString("1"), // log10(L_i (network))
		alloraMath.MustNewDecFromString("2.0"),
		alloraMath.MustNewDecFromString("2.0"),
		infererScores,
		previousForecasterScoreRatio,
		alpha,
	)
	s.Require().NoError(err)
	result, err := rewards.GetRewardForInferenceTaskInTopic(
		alloraMath.MustNewDecFromString("2.0"), // F_i
		alloraMath.MustNewDecFromString("2.0"), // G_i
		alloraMath.MustNewDecFromString("4.0"), // H_i
		&totalReward,                           // E_i
		chi,
		gamma,
	)
	expected := alloraMath.MustNewDecFromString("0.5")
	s.Require().NoError(err)
	s.Require().True(
		alloraMath.InDelta(
			expected, result, alloraMath.MustNewDecFromString("0.0001"),
		),
		"Expected ",
		expected.String(),
		" but got ",
		result.String(),
	)
}

// Cross test of U_i / V_i
func (s *MathTestSuite) TestU_iOverV_i() {
	// U_i / V_i = ((1 - χ) * γ * F_i * E_i ) / (F_i + G_i + H_i) / (χ * γ * G_i * E_i) / (F_i + G_i + H_i)
	// U_i / V_i = ((1 - χ) * γ * F_i * E_i ) / (χ * γ * G_i * E_i)
	// U_i / V_i = ((1 - χ) * F_i ) / (χ  * G_i)
	// χ = 0.5 for values of T_i >= 1
	// U_i / V_i = ((1 - 0.5) * 2 ) / (0.5  * 2)
	// U_i / V_i = 1
	infererScores := []emissionstypes.Score{
		{Score: alloraMath.MustNewDecFromString("0.5")},
		{Score: alloraMath.MustNewDecFromString("0.5")},
	}
	previousForecasterScoreRatio := alloraMath.ZeroDec()
	alpha := alloraMath.OneDec()
	totalReward := alloraMath.MustNewDecFromString("2.0")
	chi, gamma, err := rewards.GetChiAndGamma(
		alloraMath.MustNewDecFromString("2"), // log10(L_i- (naive))
		alloraMath.MustNewDecFromString("1"), // log10(L_i (network))
		alloraMath.MustNewDecFromString("2.0"),
		alloraMath.MustNewDecFromString("2.0"),
		infererScores,
		previousForecasterScoreRatio,
		alpha,
	)
	s.Require().NoError(err)
	U_i, err := rewards.GetRewardForInferenceTaskInTopic(
		alloraMath.MustNewDecFromString("2.0"), // F_i
		alloraMath.MustNewDecFromString("2.0"), // G_i
		alloraMath.MustNewDecFromString("4.0"), // H_i
		&totalReward,                           // E_i
		chi,
		gamma,
	)
	s.Require().NoError(err)

	V_i, err := rewards.GetRewardForForecastingTaskInTopic(
		alloraMath.MustNewDecFromString("2.0"), // F_i
		alloraMath.MustNewDecFromString("2.0"), // G_i
		alloraMath.MustNewDecFromString("4.0"), // H_i
		&totalReward,                           // E_i
		chi,
		gamma,
	)
	s.Require().NoError(err)

	U_iOverV_i, err := U_i.Quo(V_i)
	s.Require().NoError(err)
	expected := alloraMath.OneDec()
	s.Require().True(
		alloraMath.InDelta(
			expected,
			U_iOverV_i, alloraMath.MustNewDecFromString("0.001"),
		),
		"expected ",
		expected,
		" got ",
		U_iOverV_i.String(),
	)
}

func (s *MathTestSuite) TestForecastRewardsZero() {
	totalReward := alloraMath.ZeroDec()
	infererScores := []emissionstypes.Score{
		{Score: alloraMath.MustNewDecFromString("0.5")},
		{Score: alloraMath.MustNewDecFromString("0.5")},
	}
	previousForecasterScoreRatio := alloraMath.ZeroDec()
	alpha := alloraMath.OneDec()
	chi, gamma, err := rewards.GetChiAndGamma(
		alloraMath.MustNewDecFromString("2"), // log10(L_i- (naive))
		alloraMath.MustNewDecFromString("1"), // log10(L_i (network))
		alloraMath.MustNewDecFromString("2.0"),
		alloraMath.MustNewDecFromString("2.0"),
		infererScores,
		previousForecasterScoreRatio,
		alpha,
	)
	s.Require().NoError(err)
	result, err := rewards.GetRewardForForecastingTaskInTopic(
		alloraMath.MustNewDecFromString("2.0"), // F_i
		alloraMath.MustNewDecFromString("2.0"), // G_i
		alloraMath.MustNewDecFromString("4.0"), // H_i
		&totalReward,                           // E_i
		chi,
		gamma,
	)

	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.ZeroDec(), result, alloraMath.ZeroDec()))
}

func (s *MathTestSuite) TestReputerRewardSimple() {
	// W_i = (2 * 2) / (4 + 2 + 2)
	// W_i = 4 / 8
	// W_i = 0.5
	totalReward := alloraMath.MustNewDecFromString("2.0")
	result, err := rewards.GetRewardForReputerTaskInTopic(
		alloraMath.MustNewDecFromString("4.0"),
		alloraMath.MustNewDecFromString("2.0"),
		alloraMath.MustNewDecFromString("2.0"),
		&totalReward,
	)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.MustNewDecFromString("0.5"), result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestReputerRewardZero() {
	totalReward := alloraMath.ZeroDec()
	result, err := rewards.GetRewardForReputerTaskInTopic(
		alloraMath.MustNewDecFromString("2"),
		alloraMath.MustNewDecFromString("2.0"),
		alloraMath.MustNewDecFromString("2.0"),
		&totalReward,
	)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.ZeroDec(), result, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestReputerRewardFromCsv() {
	epochGet := testutil.GetSimulatedValuesGetterForEpochs()
	epoch3Get := epochGet[300]
	totalReward, err := testutil.GetTotalRewardForTopicInEpoch(epoch3Get)
	s.Require().NoError(err)
	result, err := rewards.GetRewardForReputerTaskInTopic(
		epoch3Get("inferers_entropy"),
		epoch3Get("forecasters_entropy"),
		epoch3Get("reputers_entropy"),
		&totalReward,
	)
	s.Require().NoError(err)
	expectedTotalReputerReward, err := testutil.GetTotalReputerRewardForTopicInEpoch(epoch3Get)
	s.Require().NoError(err)
	testutil.InEpsilon5(s.T(), result, expectedTotalReputerReward.String())
}

func (s *MathTestSuite) TestForecastingPerformanceScoreSimple() {
	networkInferenceLoss := alloraMath.MustNewDecFromString("100.0")
	naiveNetworkInferenceLoss := alloraMath.MustNewDecFromString("1000.0")
	score, err := rewards.ForecastingPerformanceScore(naiveNetworkInferenceLoss, networkInferenceLoss)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.MustNewDecFromString("900.0"), score, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestSigmoidSimple() {
	x := alloraMath.MustNewDecFromString("-4")
	result, err := rewards.Sigmoid(x)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.MustNewDecFromString("0.01798621"), result, alloraMath.MustNewDecFromString("0.00000001")))
}

func (s *MathTestSuite) TestForecastingUtilitySimple() {
	alpha := alloraMath.OneDec()
	previousForecasterScoreRatio := alloraMath.ZeroDec()
	// Test case where score < 0
	negativeScore := alloraMath.MustNewDecFromString("-0.1")
	infererScores := []emissionstypes.Score{
		{Score: alloraMath.MustNewDecFromString("0.5")},
		{Score: alloraMath.MustNewDecFromString("0.5")},
	}
	ret, err := rewards.ForecastingUtility(negativeScore, infererScores, previousForecasterScoreRatio, alpha)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.MustNewDecFromString("0.1"), ret, alloraMath.MustNewDecFromString("0.0001")))

	// Test case where score > 1
	highScore := alloraMath.MustNewDecFromString("1.1")
	ret, err = rewards.ForecastingUtility(highScore, infererScores, previousForecasterScoreRatio, alpha)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.MustNewDecFromString("0.5"), ret, alloraMath.MustNewDecFromString("0.0001")))

	// Test case where 0 <= score <= 1
	forecastingPerformanceScore := alloraMath.MustNewDecFromString("0.125")
	expectedResult := alloraMath.MustNewDecFromString("0.2")

	ret, err = rewards.ForecastingUtility(forecastingPerformanceScore, infererScores, previousForecasterScoreRatio, alpha)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(expectedResult, ret, alloraMath.MustNewDecFromString("0.0001")))
}

func (s *MathTestSuite) TestNormalizationFactorSimple() {
	entropyInference := alloraMath.MustNewDecFromString("4.0")
	entropyForecasting := alloraMath.MustNewDecFromString("6.0")
	chi := alloraMath.MustNewDecFromString("0.5")

	// (4+6) / (1-0.5)*4 + 0.5*6
	// 10 / 2 + 3
	// 10 / 5
	// 2

	result, err := rewards.NormalizationFactor(entropyInference, entropyForecasting, chi)
	s.Require().NoError(err)

	s.Require().True(alloraMath.InDelta(alloraMath.NewDecFromInt64(2), result, alloraMath.MustNewDecFromString("0.0001")))
}

func TestCalculateReputerRewardFractions(t *testing.T) {
	tests := []struct {
		name    string
		stakes  []alloraMath.Dec
		scores  []alloraMath.Dec
		preward alloraMath.Dec
		want    []alloraMath.Dec
		wantErr bool
	}{
		{
			name:    "basic",
			stakes:  []alloraMath.Dec{alloraMath.MustNewDecFromString("1178377.89152"), alloraMath.MustNewDecFromString("385287.87376"), alloraMath.MustNewDecFromString("395488.13091"), alloraMath.MustNewDecFromString("208201.11762"), alloraMath.MustNewDecFromString("369044.55988")},
			scores:  []alloraMath.Dec{alloraMath.MustNewDecFromString("17.53839"), alloraMath.MustNewDecFromString("22.63517"), alloraMath.MustNewDecFromString("26.28035"), alloraMath.MustNewDecFromString("13.51383"), alloraMath.MustNewDecFromString("15.08629")},
			preward: alloraMath.OneDec(),
			want:    []alloraMath.Dec{alloraMath.MustNewDecFromString("0.42911"), alloraMath.MustNewDecFromString("0.18108"), alloraMath.MustNewDecFromString("0.2158"), alloraMath.MustNewDecFromString("0.05842"), alloraMath.MustNewDecFromString("0.1156")},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rewards.CalculateReputerRewardFractions(tt.stakes, tt.scores, tt.preward)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetReputerRewardFractions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !alloraMath.SlicesInDelta(got, tt.want, alloraMath.MustNewDecFromString("0.00001")) {
				t.Errorf("GetReputerRewardFractions() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStakeWeightedLossMatrix(t *testing.T) {
	tests := []struct {
		name                   string
		reputersAdjustedStakes []alloraMath.Dec
		reputersReportedLosses [][]alloraMath.Dec
		want                   []alloraMath.Dec
		wantErr                bool
	}{
		{
			name: "basic",
			reputersAdjustedStakes: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("1.0"),
				alloraMath.MustNewDecFromString("0.76188"),
				alloraMath.MustNewDecFromString("0.7816"),
				alloraMath.MustNewDecFromString("0.40664"),
				alloraMath.MustNewDecFromString("0.71687"),
			},
			reputersReportedLosses: [][]alloraMath.Dec{
				{alloraMath.MustNewDecFromString("0.0112"), alloraMath.MustNewDecFromString("0.00231"), alloraMath.MustNewDecFromString("0.02274"), alloraMath.MustNewDecFromString("0.01299"), alloraMath.MustNewDecFromString("0.02515"), alloraMath.MustNewDecFromString("0.0185"), alloraMath.MustNewDecFromString("0.01018"), alloraMath.MustNewDecFromString("0.02105"), alloraMath.MustNewDecFromString("0.01041"), alloraMath.MustNewDecFromString("0.0183"), alloraMath.MustNewDecFromString("0.01022"), alloraMath.MustNewDecFromString("0.01333"), alloraMath.MustNewDecFromString("0.01298"), alloraMath.MustNewDecFromString("0.01023"), alloraMath.MustNewDecFromString("0.01268"), alloraMath.MustNewDecFromString("0.01381"), alloraMath.MustNewDecFromString("0.01731"), alloraMath.MustNewDecFromString("0.01238"), alloraMath.MustNewDecFromString("0.01168"), alloraMath.MustNewDecFromString("0.00929"), alloraMath.MustNewDecFromString("0.01212"), alloraMath.MustNewDecFromString("0.01806"), alloraMath.MustNewDecFromString("0.01901"), alloraMath.MustNewDecFromString("0.01828"), alloraMath.MustNewDecFromString("0.01522"), alloraMath.MustNewDecFromString("0.01833"), alloraMath.MustNewDecFromString("0.0101"), alloraMath.MustNewDecFromString("0.01224"), alloraMath.MustNewDecFromString("0.01226"), alloraMath.MustNewDecFromString("0.01474"), alloraMath.MustNewDecFromString("0.01218"), alloraMath.MustNewDecFromString("0.01604"), alloraMath.MustNewDecFromString("0.01149"), alloraMath.MustNewDecFromString("0.02075"), alloraMath.MustNewDecFromString("0.00818"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01127"), alloraMath.MustNewDecFromString("0.01495"), alloraMath.MustNewDecFromString("0.00689"), alloraMath.MustNewDecFromString("0.0108"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.0124"), alloraMath.MustNewDecFromString("0.01588"), alloraMath.MustNewDecFromString("0.01012"), alloraMath.MustNewDecFromString("0.01467"), alloraMath.MustNewDecFromString("0.0128"), alloraMath.MustNewDecFromString("0.01234"), alloraMath.MustNewDecFromString("0.0148"), alloraMath.MustNewDecFromString("0.01046"), alloraMath.MustNewDecFromString("0.01192"), alloraMath.MustNewDecFromString("0.01381"), alloraMath.MustNewDecFromString("0.01687"), alloraMath.MustNewDecFromString("0.01136"), alloraMath.MustNewDecFromString("0.01185"), alloraMath.MustNewDecFromString("0.01568"), alloraMath.MustNewDecFromString("0.00949"), alloraMath.MustNewDecFromString("0.01339")},
				{alloraMath.MustNewDecFromString("0.01635"), alloraMath.MustNewDecFromString("0.00179"), alloraMath.MustNewDecFromString("0.03396"), alloraMath.MustNewDecFromString("0.0153"), alloraMath.MustNewDecFromString("0.01988"), alloraMath.MustNewDecFromString("0.00962"), alloraMath.MustNewDecFromString("0.01191"), alloraMath.MustNewDecFromString("0.01616"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.01216"), alloraMath.MustNewDecFromString("0.01292"), alloraMath.MustNewDecFromString("0.01564"), alloraMath.MustNewDecFromString("0.01323"), alloraMath.MustNewDecFromString("0.01261"), alloraMath.MustNewDecFromString("0.01145"), alloraMath.MustNewDecFromString("0.0163"), alloraMath.MustNewDecFromString("0.014"), alloraMath.MustNewDecFromString("0.01373"), alloraMath.MustNewDecFromString("0.01453"), alloraMath.MustNewDecFromString("0.01207"), alloraMath.MustNewDecFromString("0.01641"), alloraMath.MustNewDecFromString("0.01601"), alloraMath.MustNewDecFromString("0.01114"), alloraMath.MustNewDecFromString("0.01259"), alloraMath.MustNewDecFromString("0.01589"), alloraMath.MustNewDecFromString("0.01229"), alloraMath.MustNewDecFromString("0.01309"), alloraMath.MustNewDecFromString("0.0138"), alloraMath.MustNewDecFromString("0.01162"), alloraMath.MustNewDecFromString("0.01145"), alloraMath.MustNewDecFromString("0.01013"), alloraMath.MustNewDecFromString("0.01208"), alloraMath.MustNewDecFromString("0.0111"), alloraMath.MustNewDecFromString("0.0118"), alloraMath.MustNewDecFromString("0.01374"), alloraMath.MustNewDecFromString("0.01428"), alloraMath.MustNewDecFromString("0.01791"), alloraMath.MustNewDecFromString("0.01288"), alloraMath.MustNewDecFromString("0.01161"), alloraMath.MustNewDecFromString("0.01151"), alloraMath.MustNewDecFromString("0.01148"), alloraMath.MustNewDecFromString("0.01284"), alloraMath.MustNewDecFromString("0.01239"), alloraMath.MustNewDecFromString("0.01023"), alloraMath.MustNewDecFromString("0.01712"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01639"), alloraMath.MustNewDecFromString("0.01043"), alloraMath.MustNewDecFromString("0.01308"), alloraMath.MustNewDecFromString("0.01455"), alloraMath.MustNewDecFromString("0.01607"), alloraMath.MustNewDecFromString("0.01205"), alloraMath.MustNewDecFromString("0.01357"), alloraMath.MustNewDecFromString("0.01108"), alloraMath.MustNewDecFromString("0.01633"), alloraMath.MustNewDecFromString("0.01208"), alloraMath.MustNewDecFromString("0.01278")},
				{alloraMath.MustNewDecFromString("0.01345"), alloraMath.MustNewDecFromString("0.00209"), alloraMath.MustNewDecFromString("0.03249"), alloraMath.MustNewDecFromString("0.01688"), alloraMath.MustNewDecFromString("0.02126"), alloraMath.MustNewDecFromString("0.01338"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01605"), alloraMath.MustNewDecFromString("0.0133"), alloraMath.MustNewDecFromString("0.01407"), alloraMath.MustNewDecFromString("0.01367"), alloraMath.MustNewDecFromString("0.01244"), alloraMath.MustNewDecFromString("0.0145"), alloraMath.MustNewDecFromString("0.01262"), alloraMath.MustNewDecFromString("0.01348"), alloraMath.MustNewDecFromString("0.01684"), alloraMath.MustNewDecFromString("0.01148"), alloraMath.MustNewDecFromString("0.01705"), alloraMath.MustNewDecFromString("0.01714"), alloraMath.MustNewDecFromString("0.0124"), alloraMath.MustNewDecFromString("0.0125"), alloraMath.MustNewDecFromString("0.01462"), alloraMath.MustNewDecFromString("0.01274"), alloraMath.MustNewDecFromString("0.01407"), alloraMath.MustNewDecFromString("0.01667"), alloraMath.MustNewDecFromString("0.01316"), alloraMath.MustNewDecFromString("0.01628"), alloraMath.MustNewDecFromString("0.01373"), alloraMath.MustNewDecFromString("0.01409"), alloraMath.MustNewDecFromString("0.01603"), alloraMath.MustNewDecFromString("0.01378"), alloraMath.MustNewDecFromString("0.01143"), alloraMath.MustNewDecFromString("0.013"), alloraMath.MustNewDecFromString("0.01644"), alloraMath.MustNewDecFromString("0.01528"), alloraMath.MustNewDecFromString("0.01441"), alloraMath.MustNewDecFromString("0.01404"), alloraMath.MustNewDecFromString("0.01402"), alloraMath.MustNewDecFromString("0.01479"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.01244"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01419"), alloraMath.MustNewDecFromString("0.01497"), alloraMath.MustNewDecFromString("0.01629"), alloraMath.MustNewDecFromString("0.01514"), alloraMath.MustNewDecFromString("0.01133"), alloraMath.MustNewDecFromString("0.01339"), alloraMath.MustNewDecFromString("0.01053"), alloraMath.MustNewDecFromString("0.01424"), alloraMath.MustNewDecFromString("0.01428"), alloraMath.MustNewDecFromString("0.01446"), alloraMath.MustNewDecFromString("0.01805"), alloraMath.MustNewDecFromString("0.01229"), alloraMath.MustNewDecFromString("0.01586"), alloraMath.MustNewDecFromString("0.01234"), alloraMath.MustNewDecFromString("0.01513")},
				{alloraMath.MustNewDecFromString("0.01675"), alloraMath.MustNewDecFromString("0.00318"), alloraMath.MustNewDecFromString("0.02623"), alloraMath.MustNewDecFromString("0.02734"), alloraMath.MustNewDecFromString("0.03526"), alloraMath.MustNewDecFromString("0.02733"), alloraMath.MustNewDecFromString("0.01697"), alloraMath.MustNewDecFromString("0.01619"), alloraMath.MustNewDecFromString("0.01925"), alloraMath.MustNewDecFromString("0.02018"), alloraMath.MustNewDecFromString("0.01735"), alloraMath.MustNewDecFromString("0.01922"), alloraMath.MustNewDecFromString("0.02225"), alloraMath.MustNewDecFromString("0.0189"), alloraMath.MustNewDecFromString("0.01923"), alloraMath.MustNewDecFromString("0.03193"), alloraMath.MustNewDecFromString("0.01956"), alloraMath.MustNewDecFromString("0.01763"), alloraMath.MustNewDecFromString("0.01975"), alloraMath.MustNewDecFromString("0.01466"), alloraMath.MustNewDecFromString("0.02021"), alloraMath.MustNewDecFromString("0.01803"), alloraMath.MustNewDecFromString("0.01438"), alloraMath.MustNewDecFromString("0.01929"), alloraMath.MustNewDecFromString("0.02305"), alloraMath.MustNewDecFromString("0.02223"), alloraMath.MustNewDecFromString("0.02445"), alloraMath.MustNewDecFromString("0.01967"), alloraMath.MustNewDecFromString("0.02292"), alloraMath.MustNewDecFromString("0.01878"), alloraMath.MustNewDecFromString("0.01751"), alloraMath.MustNewDecFromString("0.02695"), alloraMath.MustNewDecFromString("0.01849"), alloraMath.MustNewDecFromString("0.01658"), alloraMath.MustNewDecFromString("0.01948"), alloraMath.MustNewDecFromString("0.01594"), alloraMath.MustNewDecFromString("0.02318"), alloraMath.MustNewDecFromString("0.01906"), alloraMath.MustNewDecFromString("0.01607"), alloraMath.MustNewDecFromString("0.01369"), alloraMath.MustNewDecFromString("0.01686"), alloraMath.MustNewDecFromString("0.01314"), alloraMath.MustNewDecFromString("0.01936"), alloraMath.MustNewDecFromString("0.01518"), alloraMath.MustNewDecFromString("0.018"), alloraMath.MustNewDecFromString("0.02212"), alloraMath.MustNewDecFromString("0.02259"), alloraMath.MustNewDecFromString("0.01674"), alloraMath.MustNewDecFromString("0.02944"), alloraMath.MustNewDecFromString("0.01796"), alloraMath.MustNewDecFromString("0.02187"), alloraMath.MustNewDecFromString("0.01895"), alloraMath.MustNewDecFromString("0.01637"), alloraMath.MustNewDecFromString("0.01594"), alloraMath.MustNewDecFromString("0.01608"), alloraMath.MustNewDecFromString("0.02203"), alloraMath.MustNewDecFromString("0.01486")},
				{alloraMath.MustNewDecFromString("0.02093"), alloraMath.MustNewDecFromString("0.00213"), alloraMath.MustNewDecFromString("0.02462"), alloraMath.MustNewDecFromString("0.0203"), alloraMath.MustNewDecFromString("0.03115"), alloraMath.MustNewDecFromString("0.01"), alloraMath.MustNewDecFromString("0.01545"), alloraMath.MustNewDecFromString("0.01785"), alloraMath.MustNewDecFromString("0.01662"), alloraMath.MustNewDecFromString("0.01156"), alloraMath.MustNewDecFromString("0.02284"), alloraMath.MustNewDecFromString("0.01475"), alloraMath.MustNewDecFromString("0.01331"), alloraMath.MustNewDecFromString("0.01592"), alloraMath.MustNewDecFromString("0.01462"), alloraMath.MustNewDecFromString("0.02333"), alloraMath.MustNewDecFromString("0.01836"), alloraMath.MustNewDecFromString("0.01465"), alloraMath.MustNewDecFromString("0.0186"), alloraMath.MustNewDecFromString("0.01566"), alloraMath.MustNewDecFromString("0.01506"), alloraMath.MustNewDecFromString("0.01678"), alloraMath.MustNewDecFromString("0.01423"), alloraMath.MustNewDecFromString("0.01658"), alloraMath.MustNewDecFromString("0.01741"), alloraMath.MustNewDecFromString("0.03491"), alloraMath.MustNewDecFromString("0.01408"), alloraMath.MustNewDecFromString("0.01191"), alloraMath.MustNewDecFromString("0.01572"), alloraMath.MustNewDecFromString("0.01355"), alloraMath.MustNewDecFromString("0.01477"), alloraMath.MustNewDecFromString("0.01662"), alloraMath.MustNewDecFromString("0.01128"), alloraMath.MustNewDecFromString("0.02581"), alloraMath.MustNewDecFromString("0.01718"), alloraMath.MustNewDecFromString("0.01705"), alloraMath.MustNewDecFromString("0.01251"), alloraMath.MustNewDecFromString("0.02158"), alloraMath.MustNewDecFromString("0.01187"), alloraMath.MustNewDecFromString("0.01504"), alloraMath.MustNewDecFromString("0.0135"), alloraMath.MustNewDecFromString("0.02432"), alloraMath.MustNewDecFromString("0.01602"), alloraMath.MustNewDecFromString("0.01194"), alloraMath.MustNewDecFromString("0.0153"), alloraMath.MustNewDecFromString("0.0199"), alloraMath.MustNewDecFromString("0.01673"), alloraMath.MustNewDecFromString("0.01049"), alloraMath.MustNewDecFromString("0.02068"), alloraMath.MustNewDecFromString("0.01573"), alloraMath.MustNewDecFromString("0.01487"), alloraMath.MustNewDecFromString("0.02639"), alloraMath.MustNewDecFromString("0.01981"), alloraMath.MustNewDecFromString("0.02123"), alloraMath.MustNewDecFromString("0.02134"), alloraMath.MustNewDecFromString("0.0217"), alloraMath.MustNewDecFromString("0.01177")},
			},
			want: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("0.0152671"), alloraMath.MustNewDecFromString("0.002216"), alloraMath.MustNewDecFromString("0.02790"), alloraMath.MustNewDecFromString("0.017319"), alloraMath.MustNewDecFromString("0.025520"), alloraMath.MustNewDecFromString("0.0148812"), alloraMath.MustNewDecFromString("0.012625"), alloraMath.MustNewDecFromString("0.01780378"), alloraMath.MustNewDecFromString("0.0140014"),
				alloraMath.MustNewDecFromString("0.015013"), alloraMath.MustNewDecFromString("0.014774"), alloraMath.MustNewDecFromString("0.014550"), alloraMath.MustNewDecFromString("0.0144484"), alloraMath.MustNewDecFromString("0.0133076"), alloraMath.MustNewDecFromString("0.0137"), alloraMath.MustNewDecFromString("0.018843"), alloraMath.MustNewDecFromString("0.0158344"), alloraMath.MustNewDecFromString("0.014681"),
				alloraMath.MustNewDecFromString("0.015683"), alloraMath.MustNewDecFromString("0.012371"), alloraMath.MustNewDecFromString("0.014564"), alloraMath.MustNewDecFromString("0.0166473"), alloraMath.MustNewDecFromString("0.0145905"), alloraMath.MustNewDecFromString("0.01598"), alloraMath.MustNewDecFromString("0.01696"), alloraMath.MustNewDecFromString("0.01964"), alloraMath.MustNewDecFromString("0.0144"),
				alloraMath.MustNewDecFromString("0.0136411"), alloraMath.MustNewDecFromString("0.014375"), alloraMath.MustNewDecFromString("0.0145467"), alloraMath.MustNewDecFromString("0.01319248"), alloraMath.MustNewDecFromString("0.01555"), alloraMath.MustNewDecFromString("0.01246"), alloraMath.MustNewDecFromString("0.01849"), alloraMath.MustNewDecFromString("0.01386"), alloraMath.MustNewDecFromString("0.01430"),
				alloraMath.MustNewDecFromString("0.0148031"), alloraMath.MustNewDecFromString("0.016073"), alloraMath.MustNewDecFromString("0.01154604"), alloraMath.MustNewDecFromString("0.01281518"), alloraMath.MustNewDecFromString("0.01340968"), alloraMath.MustNewDecFromString("0.014733235"), alloraMath.MustNewDecFromString("0.01520795"), alloraMath.MustNewDecFromString("0.012093517"), alloraMath.MustNewDecFromString("0.0160167"),
				alloraMath.MustNewDecFromString("0.01547095"), alloraMath.MustNewDecFromString("0.01496103"), alloraMath.MustNewDecFromString("0.01296408"), alloraMath.MustNewDecFromString("0.0151219369"), alloraMath.MustNewDecFromString("0.014375538"), alloraMath.MustNewDecFromString("0.01548074"), alloraMath.MustNewDecFromString("0.017446629"), alloraMath.MustNewDecFromString("0.015452587"), alloraMath.MustNewDecFromString("0.01407107"),
				alloraMath.MustNewDecFromString("0.01700426"), alloraMath.MustNewDecFromString("0.014413132"), alloraMath.MustNewDecFromString("0.013480447"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := rewards.GetStakeWeightedLossMatrix(tt.reputersAdjustedStakes, tt.reputersReportedLosses)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStakeWeightedLossMatrix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !alloraMath.SlicesInDelta(got, tt.want, alloraMath.MustNewDecFromString("1e-5")) {
				t.Errorf("GetStakeWeightedLossMatrix() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStakeWeightedLossMatrixWithMissingLosses(t *testing.T) {
	tests := []struct {
		name                   string
		reputersAdjustedStakes []alloraMath.Dec
		reputersReportedLosses [][]alloraMath.Dec
		want                   []alloraMath.Dec
		wantErr                bool
	}{
		{
			name: "basic",
			reputersAdjustedStakes: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("1.0"),
				alloraMath.MustNewDecFromString("1.0"),
				alloraMath.MustNewDecFromString("1.0"),
			},
			reputersReportedLosses: [][]alloraMath.Dec{
				{alloraMath.MustNewDecFromString("1.0"), alloraMath.MustNewDecFromString("2.0"), alloraMath.MustNewDecFromString("3.0"), alloraMath.MustNewDecFromString("4.0")},
				{alloraMath.MustNewDecFromString("2.0"), alloraMath.NewNaN(), alloraMath.MustNewDecFromString("5.0"), alloraMath.MustNewDecFromString("3.0")},
				{alloraMath.NewNaN(), alloraMath.NewNaN(), alloraMath.MustNewDecFromString("1.0"), alloraMath.MustNewDecFromString("2.0")},
			},
			want: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("1.5"), alloraMath.MustNewDecFromString("2.00000"), alloraMath.MustNewDecFromString("3.0"), alloraMath.MustNewDecFromString("2.999999"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := rewards.GetStakeWeightedLossMatrix(tt.reputersAdjustedStakes, tt.reputersReportedLosses)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStakeWeightedLossMatrix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !alloraMath.SlicesInDelta(got, tt.want, alloraMath.MustNewDecFromString("1e-5")) {
				t.Errorf("GetStakeWeightedLossMatrix() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStakeWeightedLoss(t *testing.T) {
	tests := []struct {
		name                   string
		reputersStakes         []alloraMath.Dec
		reputersReportedLosses []alloraMath.Dec
		want                   alloraMath.Dec
		wantErr                bool
	}{
		{
			name:                   "simple average",
			reputersStakes:         []alloraMath.Dec{alloraMath.MustNewDecFromString("1176644.37627"), alloraMath.MustNewDecFromString("384623.3607"), alloraMath.MustNewDecFromString("394676.13226"), alloraMath.MustNewDecFromString("207999.66194"), alloraMath.MustNewDecFromString("368582.76542")},
			reputersReportedLosses: []alloraMath.Dec{alloraMath.MustNewDecFromString("0.0112"), alloraMath.MustNewDecFromString("0.01635"), alloraMath.MustNewDecFromString("0.01345"), alloraMath.MustNewDecFromString("0.01675"), alloraMath.MustNewDecFromString("0.02093")},
			want:                   alloraMath.MustNewDecFromString("0.0142047230098813"),
			wantErr:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rewards.GetStakeWeightedLoss(tt.reputersStakes, tt.reputersReportedLosses)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStakeWeightedLoss() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !(alloraMath.InDelta(tt.want, got, alloraMath.MustNewDecFromString("0.00001"))) {
				t.Errorf("GetStakeWeightedLoss() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFinalWorkerScoreForecastTask(t *testing.T) {
	tests := []struct {
		name        string
		scoreOneIn  alloraMath.Dec
		scoreOneOut alloraMath.Dec
		fUniqueAgg  alloraMath.Dec
		want        alloraMath.Dec
	}{
		{
			"basic",
			alloraMath.MustNewDecFromString("0.07300629674057668"),
			alloraMath.MustNewDecFromString("-0.009510726019112292"),
			alloraMath.MustNewDecFromString("0.0625"),
			alloraMath.MustNewDecFromString("-0.004353412096631731"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rewards.GetFinalWorkerScoreForecastTask(tt.scoreOneIn, tt.scoreOneOut, tt.fUniqueAgg)
			require.NoError(t, err)
			if !alloraMath.InDelta(tt.want, got, alloraMath.MustNewDecFromString("0.00001")) {
				t.Errorf("GetFinalWorkerScoreForecastTask() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAllConsensusScores(t *testing.T) {
	allLosses := [][]alloraMath.Dec{
		{alloraMath.MustNewDecFromString("0.0112"), alloraMath.MustNewDecFromString("0.00231"), alloraMath.MustNewDecFromString("0.02274"), alloraMath.MustNewDecFromString("0.01299"), alloraMath.MustNewDecFromString("0.02515"), alloraMath.MustNewDecFromString("0.0185"), alloraMath.MustNewDecFromString("0.01018"), alloraMath.MustNewDecFromString("0.02105"), alloraMath.MustNewDecFromString("0.01041"), alloraMath.MustNewDecFromString("0.0183"), alloraMath.MustNewDecFromString("0.01022"), alloraMath.MustNewDecFromString("0.01333"), alloraMath.MustNewDecFromString("0.01298"), alloraMath.MustNewDecFromString("0.01023"), alloraMath.MustNewDecFromString("0.01268"), alloraMath.MustNewDecFromString("0.01381"), alloraMath.MustNewDecFromString("0.01731"), alloraMath.MustNewDecFromString("0.01238"), alloraMath.MustNewDecFromString("0.01168"), alloraMath.MustNewDecFromString("0.00929"), alloraMath.MustNewDecFromString("0.01212"), alloraMath.MustNewDecFromString("0.01806"), alloraMath.MustNewDecFromString("0.01901"), alloraMath.MustNewDecFromString("0.01828"), alloraMath.MustNewDecFromString("0.01522"), alloraMath.MustNewDecFromString("0.01833"), alloraMath.MustNewDecFromString("0.0101"), alloraMath.MustNewDecFromString("0.01224"), alloraMath.MustNewDecFromString("0.01226"), alloraMath.MustNewDecFromString("0.01474"), alloraMath.MustNewDecFromString("0.01218"), alloraMath.MustNewDecFromString("0.01604"), alloraMath.MustNewDecFromString("0.01149"), alloraMath.MustNewDecFromString("0.02075"), alloraMath.MustNewDecFromString("0.00818"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01127"), alloraMath.MustNewDecFromString("0.01495"), alloraMath.MustNewDecFromString("0.00689"), alloraMath.MustNewDecFromString("0.0108"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.0124"), alloraMath.MustNewDecFromString("0.01588"), alloraMath.MustNewDecFromString("0.01012"), alloraMath.MustNewDecFromString("0.01467"), alloraMath.MustNewDecFromString("0.0128"), alloraMath.MustNewDecFromString("0.01234"), alloraMath.MustNewDecFromString("0.0148"), alloraMath.MustNewDecFromString("0.01046"), alloraMath.MustNewDecFromString("0.01192"), alloraMath.MustNewDecFromString("0.01381"), alloraMath.MustNewDecFromString("0.01687"), alloraMath.MustNewDecFromString("0.01136"), alloraMath.MustNewDecFromString("0.01185"), alloraMath.MustNewDecFromString("0.01568"), alloraMath.MustNewDecFromString("0.00949"), alloraMath.MustNewDecFromString("0.01339")},
		{alloraMath.MustNewDecFromString("0.01635"), alloraMath.MustNewDecFromString("0.00179"), alloraMath.MustNewDecFromString("0.03396"), alloraMath.MustNewDecFromString("0.0153"), alloraMath.MustNewDecFromString("0.01988"), alloraMath.MustNewDecFromString("0.00962"), alloraMath.MustNewDecFromString("0.01191"), alloraMath.MustNewDecFromString("0.01616"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.01216"), alloraMath.MustNewDecFromString("0.01292"), alloraMath.MustNewDecFromString("0.01564"), alloraMath.MustNewDecFromString("0.01323"), alloraMath.MustNewDecFromString("0.01261"), alloraMath.MustNewDecFromString("0.01145"), alloraMath.MustNewDecFromString("0.0163"), alloraMath.MustNewDecFromString("0.014"), alloraMath.MustNewDecFromString("0.01373"), alloraMath.MustNewDecFromString("0.01453"), alloraMath.MustNewDecFromString("0.01207"), alloraMath.MustNewDecFromString("0.01641"), alloraMath.MustNewDecFromString("0.01601"), alloraMath.MustNewDecFromString("0.01114"), alloraMath.MustNewDecFromString("0.01259"), alloraMath.MustNewDecFromString("0.01589"), alloraMath.MustNewDecFromString("0.01229"), alloraMath.MustNewDecFromString("0.01309"), alloraMath.MustNewDecFromString("0.0138"), alloraMath.MustNewDecFromString("0.01162"), alloraMath.MustNewDecFromString("0.01145"), alloraMath.MustNewDecFromString("0.01013"), alloraMath.MustNewDecFromString("0.01208"), alloraMath.MustNewDecFromString("0.0111"), alloraMath.MustNewDecFromString("0.0118"), alloraMath.MustNewDecFromString("0.01374"), alloraMath.MustNewDecFromString("0.01428"), alloraMath.MustNewDecFromString("0.01791"), alloraMath.MustNewDecFromString("0.01288"), alloraMath.MustNewDecFromString("0.01161"), alloraMath.MustNewDecFromString("0.01151"), alloraMath.MustNewDecFromString("0.01148"), alloraMath.MustNewDecFromString("0.01284"), alloraMath.MustNewDecFromString("0.01239"), alloraMath.MustNewDecFromString("0.01023"), alloraMath.MustNewDecFromString("0.01712"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01639"), alloraMath.MustNewDecFromString("0.01043"), alloraMath.MustNewDecFromString("0.01308"), alloraMath.MustNewDecFromString("0.01455"), alloraMath.MustNewDecFromString("0.01607"), alloraMath.MustNewDecFromString("0.01205"), alloraMath.MustNewDecFromString("0.01357"), alloraMath.MustNewDecFromString("0.01108"), alloraMath.MustNewDecFromString("0.01633"), alloraMath.MustNewDecFromString("0.01208"), alloraMath.MustNewDecFromString("0.01278")},
		{alloraMath.MustNewDecFromString("0.01345"), alloraMath.MustNewDecFromString("0.00209"), alloraMath.MustNewDecFromString("0.03249"), alloraMath.MustNewDecFromString("0.01688"), alloraMath.MustNewDecFromString("0.02126"), alloraMath.MustNewDecFromString("0.01338"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01605"), alloraMath.MustNewDecFromString("0.0133"), alloraMath.MustNewDecFromString("0.01407"), alloraMath.MustNewDecFromString("0.01367"), alloraMath.MustNewDecFromString("0.01244"), alloraMath.MustNewDecFromString("0.0145"), alloraMath.MustNewDecFromString("0.01262"), alloraMath.MustNewDecFromString("0.01348"), alloraMath.MustNewDecFromString("0.01684"), alloraMath.MustNewDecFromString("0.01148"), alloraMath.MustNewDecFromString("0.01705"), alloraMath.MustNewDecFromString("0.01714"), alloraMath.MustNewDecFromString("0.0124"), alloraMath.MustNewDecFromString("0.0125"), alloraMath.MustNewDecFromString("0.01462"), alloraMath.MustNewDecFromString("0.01274"), alloraMath.MustNewDecFromString("0.01407"), alloraMath.MustNewDecFromString("0.01667"), alloraMath.MustNewDecFromString("0.01316"), alloraMath.MustNewDecFromString("0.01628"), alloraMath.MustNewDecFromString("0.01373"), alloraMath.MustNewDecFromString("0.01409"), alloraMath.MustNewDecFromString("0.01603"), alloraMath.MustNewDecFromString("0.01378"), alloraMath.MustNewDecFromString("0.01143"), alloraMath.MustNewDecFromString("0.013"), alloraMath.MustNewDecFromString("0.01644"), alloraMath.MustNewDecFromString("0.01528"), alloraMath.MustNewDecFromString("0.01441"), alloraMath.MustNewDecFromString("0.01404"), alloraMath.MustNewDecFromString("0.01402"), alloraMath.MustNewDecFromString("0.01479"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.01244"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01419"), alloraMath.MustNewDecFromString("0.01497"), alloraMath.MustNewDecFromString("0.01629"), alloraMath.MustNewDecFromString("0.01514"), alloraMath.MustNewDecFromString("0.01133"), alloraMath.MustNewDecFromString("0.01339"), alloraMath.MustNewDecFromString("0.01053"), alloraMath.MustNewDecFromString("0.01424"), alloraMath.MustNewDecFromString("0.01428"), alloraMath.MustNewDecFromString("0.01446"), alloraMath.MustNewDecFromString("0.01805"), alloraMath.MustNewDecFromString("0.01229"), alloraMath.MustNewDecFromString("0.01586"), alloraMath.MustNewDecFromString("0.01234"), alloraMath.MustNewDecFromString("0.01513")},
		{alloraMath.MustNewDecFromString("0.01675"), alloraMath.MustNewDecFromString("0.00318"), alloraMath.MustNewDecFromString("0.02623"), alloraMath.MustNewDecFromString("0.02734"), alloraMath.MustNewDecFromString("0.03526"), alloraMath.MustNewDecFromString("0.02733"), alloraMath.MustNewDecFromString("0.01697"), alloraMath.MustNewDecFromString("0.01619"), alloraMath.MustNewDecFromString("0.01925"), alloraMath.MustNewDecFromString("0.02018"), alloraMath.MustNewDecFromString("0.01735"), alloraMath.MustNewDecFromString("0.01922"), alloraMath.MustNewDecFromString("0.02225"), alloraMath.MustNewDecFromString("0.0189"), alloraMath.MustNewDecFromString("0.01923"), alloraMath.MustNewDecFromString("0.03193"), alloraMath.MustNewDecFromString("0.01956"), alloraMath.MustNewDecFromString("0.01763"), alloraMath.MustNewDecFromString("0.01975"), alloraMath.MustNewDecFromString("0.01466"), alloraMath.MustNewDecFromString("0.02021"), alloraMath.MustNewDecFromString("0.01803"), alloraMath.MustNewDecFromString("0.01438"), alloraMath.MustNewDecFromString("0.01929"), alloraMath.MustNewDecFromString("0.02305"), alloraMath.MustNewDecFromString("0.02223"), alloraMath.MustNewDecFromString("0.02445"), alloraMath.MustNewDecFromString("0.01967"), alloraMath.MustNewDecFromString("0.02292"), alloraMath.MustNewDecFromString("0.01878"), alloraMath.MustNewDecFromString("0.01751"), alloraMath.MustNewDecFromString("0.02695"), alloraMath.MustNewDecFromString("0.01849"), alloraMath.MustNewDecFromString("0.01658"), alloraMath.MustNewDecFromString("0.01948"), alloraMath.MustNewDecFromString("0.01594"), alloraMath.MustNewDecFromString("0.02318"), alloraMath.MustNewDecFromString("0.01906"), alloraMath.MustNewDecFromString("0.01607"), alloraMath.MustNewDecFromString("0.01369"), alloraMath.MustNewDecFromString("0.01686"), alloraMath.MustNewDecFromString("0.01314"), alloraMath.MustNewDecFromString("0.01936"), alloraMath.MustNewDecFromString("0.01518"), alloraMath.MustNewDecFromString("0.018"), alloraMath.MustNewDecFromString("0.02212"), alloraMath.MustNewDecFromString("0.02259"), alloraMath.MustNewDecFromString("0.01674"), alloraMath.MustNewDecFromString("0.02944"), alloraMath.MustNewDecFromString("0.01796"), alloraMath.MustNewDecFromString("0.02187"), alloraMath.MustNewDecFromString("0.01895"), alloraMath.MustNewDecFromString("0.01637"), alloraMath.MustNewDecFromString("0.01594"), alloraMath.MustNewDecFromString("0.01608"), alloraMath.MustNewDecFromString("0.02203"), alloraMath.MustNewDecFromString("0.01486")},
		{alloraMath.MustNewDecFromString("0.02093"), alloraMath.MustNewDecFromString("0.00213"), alloraMath.MustNewDecFromString("0.02462"), alloraMath.MustNewDecFromString("0.0203"), alloraMath.MustNewDecFromString("0.03115"), alloraMath.MustNewDecFromString("0.01"), alloraMath.MustNewDecFromString("0.01545"), alloraMath.MustNewDecFromString("0.01785"), alloraMath.MustNewDecFromString("0.01662"), alloraMath.MustNewDecFromString("0.01156"), alloraMath.MustNewDecFromString("0.02284"), alloraMath.MustNewDecFromString("0.01475"), alloraMath.MustNewDecFromString("0.01331"), alloraMath.MustNewDecFromString("0.01592"), alloraMath.MustNewDecFromString("0.01462"), alloraMath.MustNewDecFromString("0.02333"), alloraMath.MustNewDecFromString("0.01836"), alloraMath.MustNewDecFromString("0.01465"), alloraMath.MustNewDecFromString("0.0186"), alloraMath.MustNewDecFromString("0.01566"), alloraMath.MustNewDecFromString("0.01506"), alloraMath.MustNewDecFromString("0.01678"), alloraMath.MustNewDecFromString("0.01423"), alloraMath.MustNewDecFromString("0.01658"), alloraMath.MustNewDecFromString("0.01741"), alloraMath.MustNewDecFromString("0.03491"), alloraMath.MustNewDecFromString("0.01408"), alloraMath.MustNewDecFromString("0.01191"), alloraMath.MustNewDecFromString("0.01572"), alloraMath.MustNewDecFromString("0.01355"), alloraMath.MustNewDecFromString("0.01477"), alloraMath.MustNewDecFromString("0.01662"), alloraMath.MustNewDecFromString("0.01128"), alloraMath.MustNewDecFromString("0.02581"), alloraMath.MustNewDecFromString("0.01718"), alloraMath.MustNewDecFromString("0.01705"), alloraMath.MustNewDecFromString("0.01251"), alloraMath.MustNewDecFromString("0.02158"), alloraMath.MustNewDecFromString("0.01187"), alloraMath.MustNewDecFromString("0.01504"), alloraMath.MustNewDecFromString("0.0135"), alloraMath.MustNewDecFromString("0.02432"), alloraMath.MustNewDecFromString("0.01602"), alloraMath.MustNewDecFromString("0.01194"), alloraMath.MustNewDecFromString("0.0153"), alloraMath.MustNewDecFromString("0.0199"), alloraMath.MustNewDecFromString("0.01673"), alloraMath.MustNewDecFromString("0.01049"), alloraMath.MustNewDecFromString("0.02068"), alloraMath.MustNewDecFromString("0.01573"), alloraMath.MustNewDecFromString("0.01487"), alloraMath.MustNewDecFromString("0.02639"), alloraMath.MustNewDecFromString("0.01981"), alloraMath.MustNewDecFromString("0.02123"), alloraMath.MustNewDecFromString("0.02134"), alloraMath.MustNewDecFromString("0.0217"), alloraMath.MustNewDecFromString("0.01177")},
	}
	stakes := []alloraMath.Dec{alloraMath.MustNewDecFromString("1176644.37627"), alloraMath.MustNewDecFromString("384623.3607"), alloraMath.MustNewDecFromString("394676.13226"), alloraMath.MustNewDecFromString("207999.66194"), alloraMath.MustNewDecFromString("368582.76542")}
	allListeningCoefficients := []alloraMath.Dec{alloraMath.MustNewDecFromString("1.0"), alloraMath.MustNewDecFromString("1.0"), alloraMath.MustNewDecFromString("1.0"), alloraMath.MustNewDecFromString("1.0"), alloraMath.MustNewDecFromString("1.0")}
	var numReputers int64 = 5
	reputerEpsilon := alloraMath.MustNewDecFromString("1e-2")
	epsilon := alloraMath.MustNewDecFromString("1e-4")
	want := []alloraMath.Dec{alloraMath.MustNewDecFromString("5.114259531"), alloraMath.MustNewDecFromString("5.339287075"), alloraMath.MustNewDecFromString("6.538380081"), alloraMath.MustNewDecFromString("2.5952235325"), alloraMath.MustNewDecFromString("3.5870524743")}
	wantErr := false

	got, err := rewards.GetAllConsensusScores(allLosses, stakes, allListeningCoefficients, numReputers, reputerEpsilon, epsilon)
	if (err != nil) != wantErr {
		t.Errorf("GetAllConsensusScores() error = %v, wantErr %v", err, wantErr)
		return
	}

	if !alloraMath.SlicesInDelta(got, want, alloraMath.MustNewDecFromString("0.01")) {
		t.Errorf("GetAllConsensusScores() got = %v, want %v", got, want)
	}
}

func (s *RewardsTestSuite) TestGetAllReputersOutput() {
	require := s.Require()

	params, err := s.emissionsKeeper.GetParams(s.ctx)
	require.NoError(err)

	epsilon := alloraMath.MustNewDecFromString("0.01")

	allLosses := [][]alloraMath.Dec{
		{alloraMath.MustNewDecFromString("0.0112"), alloraMath.MustNewDecFromString("0.00231"), alloraMath.MustNewDecFromString("0.02274"), alloraMath.MustNewDecFromString("0.01299"), alloraMath.MustNewDecFromString("0.02515"), alloraMath.MustNewDecFromString("0.0185"), alloraMath.MustNewDecFromString("0.01018"), alloraMath.MustNewDecFromString("0.02105"), alloraMath.MustNewDecFromString("0.01041"), alloraMath.MustNewDecFromString("0.0183"), alloraMath.MustNewDecFromString("0.01022"), alloraMath.MustNewDecFromString("0.01333"), alloraMath.MustNewDecFromString("0.01298"), alloraMath.MustNewDecFromString("0.01023"), alloraMath.MustNewDecFromString("0.01268"), alloraMath.MustNewDecFromString("0.01381"), alloraMath.MustNewDecFromString("0.01731"), alloraMath.MustNewDecFromString("0.01238"), alloraMath.MustNewDecFromString("0.01168"), alloraMath.MustNewDecFromString("0.00929"), alloraMath.MustNewDecFromString("0.01212"), alloraMath.MustNewDecFromString("0.01806"), alloraMath.MustNewDecFromString("0.01901"), alloraMath.MustNewDecFromString("0.01828"), alloraMath.MustNewDecFromString("0.01522"), alloraMath.MustNewDecFromString("0.01833"), alloraMath.MustNewDecFromString("0.0101"), alloraMath.MustNewDecFromString("0.01224"), alloraMath.MustNewDecFromString("0.01226"), alloraMath.MustNewDecFromString("0.01474"), alloraMath.MustNewDecFromString("0.01218"), alloraMath.MustNewDecFromString("0.01604"), alloraMath.MustNewDecFromString("0.01149"), alloraMath.MustNewDecFromString("0.02075"), alloraMath.MustNewDecFromString("0.00818"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01127"), alloraMath.MustNewDecFromString("0.01495"), alloraMath.MustNewDecFromString("0.00689"), alloraMath.MustNewDecFromString("0.0108"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.0124"), alloraMath.MustNewDecFromString("0.01588"), alloraMath.MustNewDecFromString("0.01012"), alloraMath.MustNewDecFromString("0.01467"), alloraMath.MustNewDecFromString("0.0128"), alloraMath.MustNewDecFromString("0.01234"), alloraMath.MustNewDecFromString("0.0148"), alloraMath.MustNewDecFromString("0.01046"), alloraMath.MustNewDecFromString("0.01192"), alloraMath.MustNewDecFromString("0.01381"), alloraMath.MustNewDecFromString("0.01687"), alloraMath.MustNewDecFromString("0.01136"), alloraMath.MustNewDecFromString("0.01185"), alloraMath.MustNewDecFromString("0.01568"), alloraMath.MustNewDecFromString("0.00949"), alloraMath.MustNewDecFromString("0.01339")},
		{alloraMath.MustNewDecFromString("0.01635"), alloraMath.MustNewDecFromString("0.00179"), alloraMath.MustNewDecFromString("0.03396"), alloraMath.MustNewDecFromString("0.0153"), alloraMath.MustNewDecFromString("0.01988"), alloraMath.MustNewDecFromString("0.00962"), alloraMath.MustNewDecFromString("0.01191"), alloraMath.MustNewDecFromString("0.01616"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.01216"), alloraMath.MustNewDecFromString("0.01292"), alloraMath.MustNewDecFromString("0.01564"), alloraMath.MustNewDecFromString("0.01323"), alloraMath.MustNewDecFromString("0.01261"), alloraMath.MustNewDecFromString("0.01145"), alloraMath.MustNewDecFromString("0.0163"), alloraMath.MustNewDecFromString("0.014"), alloraMath.MustNewDecFromString("0.01373"), alloraMath.MustNewDecFromString("0.01453"), alloraMath.MustNewDecFromString("0.01207"), alloraMath.MustNewDecFromString("0.01641"), alloraMath.MustNewDecFromString("0.01601"), alloraMath.MustNewDecFromString("0.01114"), alloraMath.MustNewDecFromString("0.01259"), alloraMath.MustNewDecFromString("0.01589"), alloraMath.MustNewDecFromString("0.01229"), alloraMath.MustNewDecFromString("0.01309"), alloraMath.MustNewDecFromString("0.0138"), alloraMath.MustNewDecFromString("0.01162"), alloraMath.MustNewDecFromString("0.01145"), alloraMath.MustNewDecFromString("0.01013"), alloraMath.MustNewDecFromString("0.01208"), alloraMath.MustNewDecFromString("0.0111"), alloraMath.MustNewDecFromString("0.0118"), alloraMath.MustNewDecFromString("0.01374"), alloraMath.MustNewDecFromString("0.01428"), alloraMath.MustNewDecFromString("0.01791"), alloraMath.MustNewDecFromString("0.01288"), alloraMath.MustNewDecFromString("0.01161"), alloraMath.MustNewDecFromString("0.01151"), alloraMath.MustNewDecFromString("0.01148"), alloraMath.MustNewDecFromString("0.01284"), alloraMath.MustNewDecFromString("0.01239"), alloraMath.MustNewDecFromString("0.01023"), alloraMath.MustNewDecFromString("0.01712"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01639"), alloraMath.MustNewDecFromString("0.01043"), alloraMath.MustNewDecFromString("0.01308"), alloraMath.MustNewDecFromString("0.01455"), alloraMath.MustNewDecFromString("0.01607"), alloraMath.MustNewDecFromString("0.01205"), alloraMath.MustNewDecFromString("0.01357"), alloraMath.MustNewDecFromString("0.01108"), alloraMath.MustNewDecFromString("0.01633"), alloraMath.MustNewDecFromString("0.01208"), alloraMath.MustNewDecFromString("0.01278")},
		{alloraMath.MustNewDecFromString("0.01345"), alloraMath.MustNewDecFromString("0.00209"), alloraMath.MustNewDecFromString("0.03249"), alloraMath.MustNewDecFromString("0.01688"), alloraMath.MustNewDecFromString("0.02126"), alloraMath.MustNewDecFromString("0.01338"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01605"), alloraMath.MustNewDecFromString("0.0133"), alloraMath.MustNewDecFromString("0.01407"), alloraMath.MustNewDecFromString("0.01367"), alloraMath.MustNewDecFromString("0.01244"), alloraMath.MustNewDecFromString("0.0145"), alloraMath.MustNewDecFromString("0.01262"), alloraMath.MustNewDecFromString("0.01348"), alloraMath.MustNewDecFromString("0.01684"), alloraMath.MustNewDecFromString("0.01148"), alloraMath.MustNewDecFromString("0.01705"), alloraMath.MustNewDecFromString("0.01714"), alloraMath.MustNewDecFromString("0.0124"), alloraMath.MustNewDecFromString("0.0125"), alloraMath.MustNewDecFromString("0.01462"), alloraMath.MustNewDecFromString("0.01274"), alloraMath.MustNewDecFromString("0.01407"), alloraMath.MustNewDecFromString("0.01667"), alloraMath.MustNewDecFromString("0.01316"), alloraMath.MustNewDecFromString("0.01628"), alloraMath.MustNewDecFromString("0.01373"), alloraMath.MustNewDecFromString("0.01409"), alloraMath.MustNewDecFromString("0.01603"), alloraMath.MustNewDecFromString("0.01378"), alloraMath.MustNewDecFromString("0.01143"), alloraMath.MustNewDecFromString("0.013"), alloraMath.MustNewDecFromString("0.01644"), alloraMath.MustNewDecFromString("0.01528"), alloraMath.MustNewDecFromString("0.01441"), alloraMath.MustNewDecFromString("0.01404"), alloraMath.MustNewDecFromString("0.01402"), alloraMath.MustNewDecFromString("0.01479"), alloraMath.MustNewDecFromString("0.01417"), alloraMath.MustNewDecFromString("0.01244"), alloraMath.MustNewDecFromString("0.0116"), alloraMath.MustNewDecFromString("0.01419"), alloraMath.MustNewDecFromString("0.01497"), alloraMath.MustNewDecFromString("0.01629"), alloraMath.MustNewDecFromString("0.01514"), alloraMath.MustNewDecFromString("0.01133"), alloraMath.MustNewDecFromString("0.01339"), alloraMath.MustNewDecFromString("0.01053"), alloraMath.MustNewDecFromString("0.01424"), alloraMath.MustNewDecFromString("0.01428"), alloraMath.MustNewDecFromString("0.01446"), alloraMath.MustNewDecFromString("0.01805"), alloraMath.MustNewDecFromString("0.01229"), alloraMath.MustNewDecFromString("0.01586"), alloraMath.MustNewDecFromString("0.01234"), alloraMath.MustNewDecFromString("0.01513")},
		{alloraMath.MustNewDecFromString("0.01675"), alloraMath.MustNewDecFromString("0.00318"), alloraMath.MustNewDecFromString("0.02623"), alloraMath.MustNewDecFromString("0.02734"), alloraMath.MustNewDecFromString("0.03526"), alloraMath.MustNewDecFromString("0.02733"), alloraMath.MustNewDecFromString("0.01697"), alloraMath.MustNewDecFromString("0.01619"), alloraMath.MustNewDecFromString("0.01925"), alloraMath.MustNewDecFromString("0.02018"), alloraMath.MustNewDecFromString("0.01735"), alloraMath.MustNewDecFromString("0.01922"), alloraMath.MustNewDecFromString("0.02225"), alloraMath.MustNewDecFromString("0.0189"), alloraMath.MustNewDecFromString("0.01923"), alloraMath.MustNewDecFromString("0.03193"), alloraMath.MustNewDecFromString("0.01956"), alloraMath.MustNewDecFromString("0.01763"), alloraMath.MustNewDecFromString("0.01975"), alloraMath.MustNewDecFromString("0.01466"), alloraMath.MustNewDecFromString("0.02021"), alloraMath.MustNewDecFromString("0.01803"), alloraMath.MustNewDecFromString("0.01438"), alloraMath.MustNewDecFromString("0.01929"), alloraMath.MustNewDecFromString("0.02305"), alloraMath.MustNewDecFromString("0.02223"), alloraMath.MustNewDecFromString("0.02445"), alloraMath.MustNewDecFromString("0.01967"), alloraMath.MustNewDecFromString("0.02292"), alloraMath.MustNewDecFromString("0.01878"), alloraMath.MustNewDecFromString("0.01751"), alloraMath.MustNewDecFromString("0.02695"), alloraMath.MustNewDecFromString("0.01849"), alloraMath.MustNewDecFromString("0.01658"), alloraMath.MustNewDecFromString("0.01948"), alloraMath.MustNewDecFromString("0.01594"), alloraMath.MustNewDecFromString("0.02318"), alloraMath.MustNewDecFromString("0.01906"), alloraMath.MustNewDecFromString("0.01607"), alloraMath.MustNewDecFromString("0.01369"), alloraMath.MustNewDecFromString("0.01686"), alloraMath.MustNewDecFromString("0.01314"), alloraMath.MustNewDecFromString("0.01936"), alloraMath.MustNewDecFromString("0.01518"), alloraMath.MustNewDecFromString("0.018"), alloraMath.MustNewDecFromString("0.02212"), alloraMath.MustNewDecFromString("0.02259"), alloraMath.MustNewDecFromString("0.01674"), alloraMath.MustNewDecFromString("0.02944"), alloraMath.MustNewDecFromString("0.01796"), alloraMath.MustNewDecFromString("0.02187"), alloraMath.MustNewDecFromString("0.01895"), alloraMath.MustNewDecFromString("0.01637"), alloraMath.MustNewDecFromString("0.01594"), alloraMath.MustNewDecFromString("0.01608"), alloraMath.MustNewDecFromString("0.02203"), alloraMath.MustNewDecFromString("0.01486")},
		{alloraMath.MustNewDecFromString("0.02093"), alloraMath.MustNewDecFromString("0.00213"), alloraMath.MustNewDecFromString("0.02462"), alloraMath.MustNewDecFromString("0.0203"), alloraMath.MustNewDecFromString("0.03115"), alloraMath.MustNewDecFromString("0.01"), alloraMath.MustNewDecFromString("0.01545"), alloraMath.MustNewDecFromString("0.01785"), alloraMath.MustNewDecFromString("0.01662"), alloraMath.MustNewDecFromString("0.01156"), alloraMath.MustNewDecFromString("0.02284"), alloraMath.MustNewDecFromString("0.01475"), alloraMath.MustNewDecFromString("0.01331"), alloraMath.MustNewDecFromString("0.01592"), alloraMath.MustNewDecFromString("0.01462"), alloraMath.MustNewDecFromString("0.02333"), alloraMath.MustNewDecFromString("0.01836"), alloraMath.MustNewDecFromString("0.01465"), alloraMath.MustNewDecFromString("0.0186"), alloraMath.MustNewDecFromString("0.01566"), alloraMath.MustNewDecFromString("0.01506"), alloraMath.MustNewDecFromString("0.01678"), alloraMath.MustNewDecFromString("0.01423"), alloraMath.MustNewDecFromString("0.01658"), alloraMath.MustNewDecFromString("0.01741"), alloraMath.MustNewDecFromString("0.03491"), alloraMath.MustNewDecFromString("0.01408"), alloraMath.MustNewDecFromString("0.01191"), alloraMath.MustNewDecFromString("0.01572"), alloraMath.MustNewDecFromString("0.01355"), alloraMath.MustNewDecFromString("0.01477"), alloraMath.MustNewDecFromString("0.01662"), alloraMath.MustNewDecFromString("0.01128"), alloraMath.MustNewDecFromString("0.02581"), alloraMath.MustNewDecFromString("0.01718"), alloraMath.MustNewDecFromString("0.01705"), alloraMath.MustNewDecFromString("0.01251"), alloraMath.MustNewDecFromString("0.02158"), alloraMath.MustNewDecFromString("0.01187"), alloraMath.MustNewDecFromString("0.01504"), alloraMath.MustNewDecFromString("0.0135"), alloraMath.MustNewDecFromString("0.02432"), alloraMath.MustNewDecFromString("0.01602"), alloraMath.MustNewDecFromString("0.01194"), alloraMath.MustNewDecFromString("0.0153"), alloraMath.MustNewDecFromString("0.0199"), alloraMath.MustNewDecFromString("0.01673"), alloraMath.MustNewDecFromString("0.01049"), alloraMath.MustNewDecFromString("0.02068"), alloraMath.MustNewDecFromString("0.01573"), alloraMath.MustNewDecFromString("0.01487"), alloraMath.MustNewDecFromString("0.02639"), alloraMath.MustNewDecFromString("0.01981"), alloraMath.MustNewDecFromString("0.02123"), alloraMath.MustNewDecFromString("0.02134"), alloraMath.MustNewDecFromString("0.0217"), alloraMath.MustNewDecFromString("0.01177")},
	}
	stakes := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("1176644.37627"),
		alloraMath.MustNewDecFromString("384623.3607"),
		alloraMath.MustNewDecFromString("394676.13226"),
		alloraMath.MustNewDecFromString("207999.66194"),
		alloraMath.MustNewDecFromString("368582.76542"),
	}
	initialCoefficients := []alloraMath.Dec{
		alloraMath.OneDec(),
		alloraMath.OneDec(),
		alloraMath.OneDec(),
		alloraMath.OneDec(),
		alloraMath.OneDec(),
	}
	var numReputers int64 = 5
	wantScores := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.016983"),
		alloraMath.MustNewDecFromString("0.017068"),
		alloraMath.MustNewDecFromString("0.016047"),
		alloraMath.MustNewDecFromString("0.011649"),
		alloraMath.MustNewDecFromString("0.013453"),
	}
	gotScores0, gotCoefficients0, err := rewards.GetAllReputersOutput(
		allLosses,
		stakes,
		initialCoefficients,
		numReputers,
		params.LearningRate,
		0,
		params.EpsilonReputer,
		epsilon,
		params.MinStakeFraction,
		params.MaxGradientThreshold,
	)
	require.NoError(err)

	gotScores1, gotCoefficients1, err := rewards.GetAllReputersOutput(
		allLosses,
		stakes,
		initialCoefficients,
		numReputers,
		params.LearningRate,
		2,
		params.EpsilonReputer,
		epsilon,
		params.MinStakeFraction,
		params.MaxGradientThreshold,
	)
	require.NoError(err)

	gotScores2, gotCoefficients2, err := rewards.GetAllReputersOutput(
		allLosses,
		stakes,
		initialCoefficients,
		numReputers,
		params.LearningRate,
		5,
		params.EpsilonReputer,
		epsilon,
		params.MinStakeFraction,
		params.MaxGradientThreshold,
	)
	require.NoError(err)

	gotScores3, gotCoefficients3, err := rewards.GetAllReputersOutput(
		allLosses,
		stakes,
		initialCoefficients,
		numReputers,
		params.LearningRate,
		20,
		params.EpsilonReputer,
		epsilon,
		params.MinStakeFraction,
		params.MaxGradientThreshold,
	)
	require.NoError(err)

	// Assumes that the inputs are of the same length
	getAdjustedStakes := func(coefficients []alloraMath.Dec) ([]alloraMath.Dec, error) {
		N_r := alloraMath.NewDecFromInt64(int64(len(stakes)))
		adjustedStakes := make([]alloraMath.Dec, len(stakes))
		adjustedStakeNumerators := make([]alloraMath.Dec, len(stakes))
		sumAdjustedStakes := alloraMath.ZeroDec()
		for i, stake := range stakes {
			adjustedStake, err := stake.Mul(coefficients[i])
			require.NoError(err)
			adjustedStake, err = adjustedStake.Mul(N_r)
			require.NoError(err)
			adjustedStakeNumerators[i] = adjustedStake
			sumAdjustedStakes, err = sumAdjustedStakes.Add(adjustedStake)
			require.NoError(err)
		}
		for i, adjustedStakeNumerator := range adjustedStakeNumerators {
			adjustedStake, err := adjustedStakeNumerator.Quo(sumAdjustedStakes)
			require.NoError(err)
			adjustedStakes[i] = alloraMath.Min(alloraMath.OneDec(), adjustedStake)
		}
		return adjustedStakes, nil
	}

	// Assumes that the inputs are the same length as the `stakes` array
	getTotalConsensusScore := func(scores []alloraMath.Dec, coefficients []alloraMath.Dec) (float64, error) {
		adjustedStakes, err := getAdjustedStakes(coefficients)
		require.NoError(err)
		require.Len(adjustedStakes, len(stakes))
		totalScore := alloraMath.ZeroDec()
		sumStake := alloraMath.ZeroDec()
		for i, score := range scores {
			stakeTimesScore, err := score.Mul(adjustedStakes[i])
			require.NoError(err)
			totalScore, err = totalScore.Add(stakeTimesScore)
			require.NoError(err)
			sumStake, err = sumStake.Add(adjustedStakes[i])
			require.NoError(err)
		}
		totalScore, err = totalScore.Quo(sumStake)
		require.NoError(err)
		output, err := strconv.ParseFloat(totalScore.String(), 64)
		require.NoError(err)
		return output, nil
	}

	startCoefficients := []alloraMath.Dec{
		alloraMath.OneDec(),
		alloraMath.OneDec(),
		alloraMath.OneDec(),
		alloraMath.OneDec(),
		alloraMath.OneDec(),
	}

	require.True(len(gotCoefficients0) == len(stakes))
	require.True(len(gotCoefficients1) == len(stakes))
	require.True(len(gotCoefficients2) == len(stakes))
	require.True(len(gotCoefficients3) == len(stakes))

	// Check that the total consensus score improves with successive invocations of the function with more iterations
	totalScore0, _ := getTotalConsensusScore(gotScores0, startCoefficients)
	totalScore1, _ := getTotalConsensusScore(gotScores1, gotCoefficients1)
	totalScore2, _ := getTotalConsensusScore(gotScores2, gotCoefficients2)
	totalScore3, _ := getTotalConsensusScore(gotScores3, gotCoefficients3)

	require.LessOrEqual(totalScore0, totalScore1)
	require.LessOrEqual(totalScore1, totalScore2)
	require.LessOrEqual(totalScore2, totalScore3)

	// Some simple checks of the scores
	require.True(len(gotScores1) == len(wantScores))
	require.True(len(gotScores2) == len(wantScores))
	require.True(len(gotScores3) == len(wantScores))

	// Verify score output matches that of GetAllConsensusScores()
	wantScores3, err := rewards.GetAllConsensusScores(allLosses, stakes, gotCoefficients3, numReputers, params.EpsilonReputer, epsilon)
	require.NoError(err)
	if !alloraMath.SlicesInDelta(gotScores3, wantScores3, alloraMath.MustNewDecFromString("0.01")) {
		log.Println("GetAllConsensusScores() got", gotScores3, "want", wantScores3)
	}
}
