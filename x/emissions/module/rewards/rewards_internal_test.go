package rewards_test

import (
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/test/testutil"
	actorutils "github.com/allora-network/allora-chain/x/emissions/keeper/actor_utils"
	"github.com/allora-network/allora-chain/x/emissions/module/rewards"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
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
	chi, gamma, _, err := rewards.GetChiAndGamma(
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
	chi, gamma, _, err := rewards.GetChiAndGamma(
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
	chi, gamma, _, err := rewards.GetChiAndGamma(
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
	chi, gamma, _, err := rewards.GetChiAndGamma(
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
	chi, gamma, _, err := rewards.GetChiAndGamma(
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
	result, err := actorutils.Sigmoid(x)
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
	ret, _, err := rewards.ForecastingUtility(negativeScore, infererScores, previousForecasterScoreRatio, alpha)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.MustNewDecFromString("0.1"), ret, alloraMath.MustNewDecFromString("0.0001")))

	// Test case where score > 1
	highScore := alloraMath.MustNewDecFromString("1.1")
	ret, _, err = rewards.ForecastingUtility(highScore, infererScores, previousForecasterScoreRatio, alpha)
	s.Require().NoError(err)
	s.Require().True(alloraMath.InDelta(alloraMath.MustNewDecFromString("0.5"), ret, alloraMath.MustNewDecFromString("0.0001")))

	// Test case where 0 <= score <= 1
	forecastingPerformanceScore := alloraMath.MustNewDecFromString("0.125")
	expectedResult := alloraMath.MustNewDecFromString("0.2")

	ret, _, err = rewards.ForecastingUtility(forecastingPerformanceScore, infererScores, previousForecasterScoreRatio, alpha)
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
