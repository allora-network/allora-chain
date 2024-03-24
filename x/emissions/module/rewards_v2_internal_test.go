package module

import (
	"math"
	"testing"

	emissions "github.com/allora-network/allora-chain/x/emissions/types"
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

func (s *MathTestSuite) TestPhiSimple() {
	x := float64(7.9997)
	p := float64(2)
	// we expect a value very very close to 64
	result, err := phi(p, x)
	s.Require().NoError(err)
	s.Require().InDelta(64, result, 0.001)
}

func (s *MathTestSuite) TestFailPhiXTooLarge() {
	// values of x that are too large should overflow the size limit of a float64
	// max float value is 1.7976931348623157e+308
	// so we test that edge condition
	x := float64(709)
	p := float64(2)
	_, err := phi(p, x)
	s.Require().NoError(err)

	x = float64(710)
	_, err = phi(p, x)
	s.Require().ErrorIs(err, emissions.ErrEToTheXExponentiationIsInfinity)
}

func (s *MathTestSuite) TestFailPhiPTooLarge() {
	// values of p that are too large should overflow the size limit of a float64
	// so we test that edge condition
	x := float64(709)
	p := float64(108)
	_, err := phi(p, x)
	s.Require().NoError(err)

	p = float64(109)
	_, err = phi(p, x)
	s.Require().ErrorIs(err, emissions.ErrLnToThePExponentiationIsInfinity)
}

func (s *MathTestSuite) TestPhiInvalidInputs() {
	// test that invalid inputs return the correct error
	_, err := phi(math.Inf(1), 3)
	s.Require().ErrorIs(err, emissions.ErrPhiInvalidInput)
	_, err = phi(math.Inf(-1), 3)
	s.Require().ErrorIs(err, emissions.ErrPhiInvalidInput)
	_, err = phi(3, math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrPhiInvalidInput)
	_, err = phi(3, math.Inf(-1))
	s.Require().ErrorIs(err, emissions.ErrPhiInvalidInput)

	_, err = phi(math.NaN(), 3)
	s.Require().ErrorIs(err, emissions.ErrPhiInvalidInput)
	_, err = phi(3, math.NaN())
	s.Require().ErrorIs(err, emissions.ErrPhiInvalidInput)
}

func (s *MathTestSuite) TestAdjustedStakeSimple() {
	// for this example we use
	// 3 reputers with stakes of 50_000, 100_000, 150_000
	// listening coefficients of 0.25, 0.18, 0.63 for those reputers
	// and we calculate the adjusted stake for reputer 2 (the 100_000)

	var stake float64 = 100000
	allStakes := []float64{50000, stake, 150000}
	listeningCoefficient := 0.18
	allListeningCoefficients := []float64{0.25, listeningCoefficient, 0.63}
	var numReputers float64 = 3

	// use wolfram alpha to calculate the expected result
	// https://www.wolframalpha.com/input?i2d=true&i=1-%5C%2840%29%5C%2840%29Power%5B%5C%2840%29Power%5B%5C%2840%29ln%5C%2840%291%2BPower%5Be%2C20%5D%5C%2841%29%5C%2841%29%2C1%5D%5C%2841%29%2C-1%5D%5C%2841%29*Power%5B%5C%2840%29ln%5C%2840%291%2BPower%5Be%2C%5C%2840%29-20%5C%2840%29Divide%5B3*0.18*100%5C%2844%29000%2C0.18*100%5C%2844%29000+%2B+0.25*50%5C%2844%29000+%2B+0.63*150%5C%2844%29000%5D+-+1%5C%2841%29%5C%2841%29%5D%5C%2841%29%5C%2841%29%2C1%5D%5C%2841%29
	expected := 0.4319994174428689223916439092220111693737492607160554179509

	result, err := adjustedStake(
		stake,
		allStakes,
		listeningCoefficient,
		allListeningCoefficients,
		numReputers,
	)
	s.Require().NoError(err)
	s.Require().InDelta(expected, result, 0.0001)
}

func (s *MathTestSuite) TestExponentialMovingAverageSimple() {
	alpha := 0.1
	var current float64 = 300
	var previous float64 = 200

	// 0.1*300 + (1-0.1)*200
	// 30 + 180 = 210
	var expected float64 = 210

	result, err := exponentialMovingAverage(alpha, current, previous)
	s.Require().NoError(err)
	s.Require().InDelta(expected, result, 0.0001)
}

func (s *MathTestSuite) TestNormalizeAgainstSlice() {
	v := 2.0
	a := []float64{2.0, 3.0, 5.0}
	expected := 0.2

	result, err := normalizeAgainstSlice(v, a)

	s.Require().NoError(err)
	s.Require().InDelta(expected, result, 0.0001)
}

func (s *MathTestSuite) TestEntropySimple() {
	f_ij := []float64{0.2, 0.3, 0.5}
	N_i_eff := 0.75
	N_i := 3.0
	beta := 0.25

	// using wolfram alpha to get a sample result
	// https://www.wolframalpha.com/input?i2d=true&i=-Power%5B%5C%2840%29Divide%5B0.75%2C3%5D%5C%2841%29%2C0.25%5D*%5C%2840%290.2*ln%5C%2840%290.2%5C%2841%29+%2B+0.3*ln%5C%2840%290.3%5C%2841%29+%2B+0.5*ln%5C%2840%290.5%5C%2841%29%5C%2841%29
	expected := 0.7280746285142275338742683350155248011115920866691059016669
	result, err := entropy(f_ij, N_i_eff, N_i, beta)
	s.Require().NoError(err)
	s.Require().InDelta(expected, result, 0.0001)
}

func (s *MathTestSuite) TestEntropyInvalidInput() {
	testCases := []struct {
		name            string
		N_eff           float64
		numParticipants float64
		beta            float64
	}{
		{
			name:            "N_eff is Inf",
			N_eff:           math.Inf(0),
			numParticipants: 10,
			beta:            0.5,
		},
		{
			name:            "N_eff is NaN",
			N_eff:           math.NaN(),
			numParticipants: 10,
			beta:            0.5,
		},
		{
			name:            "numParticipants is Inf",
			N_eff:           100,
			numParticipants: math.Inf(0),
			beta:            0.5,
		},
		{
			name:            "numParticipants is NaN",
			N_eff:           100,
			numParticipants: math.NaN(),
			beta:            0.5,
		},
		{
			name:            "beta is Inf",
			N_eff:           100,
			numParticipants: 10,
			beta:            math.Inf(0),
		},
		{
			name:            "beta is NaN",
			N_eff:           100,
			numParticipants: 10,
			beta:            math.NaN(),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			_, err := entropy(nil, tc.N_eff, tc.numParticipants, tc.beta)
			s.Require().ErrorIs(err, emissions.ErrEntropyInvalidInput)
		})
	}
}

func (s *MathTestSuite) TestEntropyInvalidInputAllFsInf() {
	allFs := []float64{1, 2, math.Inf(0), 4}
	_, err := entropy(allFs, 100, 10, 0.5)
	s.Require().ErrorIs(err, emissions.ErrEntropyInvalidInput)
}

func (s *MathTestSuite) TestEntropyInvalidInputAllFsNan() {
	allFs := []float64{1, 2, math.NaN(), 4}
	_, err := entropy(allFs, 100, 10, 0.5)
	s.Require().ErrorIs(err, emissions.ErrEntropyInvalidInput)
}

func (s *MathTestSuite) TestEntropyInfinity() {
	allFs := []float64{0.1, 0.2, 0.3, 0.4}

	_, err := entropy(allFs, 100, 10, math.MaxFloat64)
	s.Require().ErrorIs(err, emissions.ErrEntropyIsInfinity)
}

func (s *MathTestSuite) TestEntropyNaN() {
	allFs := []float64{0, 0, 0, 0}

	_, err := entropy(allFs, 100, 10, 0)
	s.Require().ErrorIs(err, emissions.ErrEntropyIsNaN)
}

func (s *MathTestSuite) TestNumberRatio() {
	rewardFractions := []float64{0.2, 0.3, 0.4, 0.5, 0.6, 0.7}

	// 1 / (0.2 *0.2 + 0.3*0.3 + 0.4*0.4 + 0.5*0.5 + 0.6*0.6 + 0.7*0.7)
	// 1 / (0.04 + 0.09 + 0.16 + 0.25 + 0.36 + 0.49)
	// 1 / 1.39 = 0.719424460431654676259005145797598627787307032590051458
	expected := 0.7194244604316546762589928057553956834532374100

	result, err := numberRatio(rewardFractions)
	s.Require().NoError(err, "Error calculating number ratio")
	s.Require().InDelta(expected, result, 0.0001)
}

func (s *MathTestSuite) TestNumberRatioZeroFractions() {
	zeroFractions := []float64{0.0}

	_, err := numberRatio(zeroFractions)
	s.Require().ErrorIs(err, emissions.ErrNumberRatioDivideByZero)
}

func (s *MathTestSuite) TestNumberRatioEmptyList() {
	emptyFractions := []float64{}

	_, err := numberRatio(emptyFractions)
	s.Require().ErrorIs(err, emissions.ErrNumberRatioInvalidSliceLength)
}

func (s *MathTestSuite) TestNumberRatioNaNFractions() {
	invalidFractions := []float64{0.2, math.NaN(), 0.5}
	_, err := numberRatio(invalidFractions)
	s.Require().ErrorIs(err, emissions.ErrNumberRatioInvalidInput)
}

func (s *MathTestSuite) TestNumberRatioInfiniteFractions() {

	infFractions := []float64{0.2, math.Inf(1), 0.5}

	_, err := numberRatio(infFractions)
	s.Require().ErrorIs(err, emissions.ErrNumberRatioInvalidInput)
}

func (s *MathTestSuite) TestInferenceRewardsSimple() {
	// U_i = ((1 - 0.5) * 2 * 2 * 2 ) / (2 + 2 + 4)
	// U_i = 0.5 * 8 / 8
	// U_i = 0.5
	infRewards, err := inferenceRewards(
		0.5,
		2.0,
		2.0,
		2.0,
		4.0,
		2.0,
	)
	s.Require().NoError(err)
	s.Require().InDelta(0.5, infRewards, 0.0001)
}

func (s *MathTestSuite) TestInferenceRewardsInvalidInput() {
	_, err := inferenceRewards(math.NaN(), 2.0, 4.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, math.NaN(), 4.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, 2.0, math.NaN(), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, 2.0, 4.0, math.NaN(), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, 2.0, 4.0, 2.0, math.NaN(), 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, 2.0, 4.0, 2.0, 2.0, math.NaN())
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(math.Inf(1), 2.0, 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(math.Inf(-1), 2.0, 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, math.Inf(1), 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, math.Inf(-1), 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, 2.0, math.Inf(1), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, 2.0, math.Inf(-1), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, 2.0, 2.0, math.Inf(1), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, 2.0, 2.0, math.Inf(-1), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, 2.0, 2.0, 2.0, math.Inf(1), 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, 2.0, 2.0, 2.0, math.Inf(-1), 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, 2.0, 2.0, 2.0, 2.0, math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = inferenceRewards(0.5, 2.0, 2.0, 2.0, 2.0, math.Inf(-1))
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)
}

func (s *MathTestSuite) TestInferenceRewardsInfinity() {
	_, err := inferenceRewards(math.MaxFloat64, math.MaxFloat64, math.MaxFloat64, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsIsInfinity)
}

func (s *MathTestSuite) TestInferenceRewardsNaN() {
	_, err := inferenceRewards(0.5, 2.0, 0, 0, 0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsIsNaN)
}

func (s *MathTestSuite) TestInferenceRewardsZero() {
	result, err := inferenceRewards(1, 2.0, 10, 20, 30, 2.0)
	s.Require().NoError(err)
	s.Require().InDelta(0, result, 0.0001)
}

func (s *MathTestSuite) TestForecastRewardsSimple() {
	// V_i = (2 * 3 * 4 * 5) / (6 + 4 + 10)
	// V_i = 120 / 20
	// V_i = 6
	result, err := forecastingRewards(2.0, 3.0, 6.0, 4.0, 10.0, 5.0)
	s.Require().NoError(err)
	s.Require().InDelta(6.0, result, 0.0001)
}

func (s *MathTestSuite) TestForecastRewardsInvalidInput() {
	_, err := forecastingRewards(math.NaN(), 2.0, 4.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, math.NaN(), 4.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, 2.0, math.NaN(), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, 2.0, 4.0, math.NaN(), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, 2.0, 4.0, 2.0, math.NaN(), 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, 2.0, 4.0, 2.0, 2.0, math.NaN())
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(math.Inf(1), 2.0, 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(math.Inf(-1), 2.0, 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, math.Inf(1), 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, math.Inf(-1), 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, 2.0, math.Inf(1), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, 2.0, math.Inf(-1), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, 2.0, 2.0, math.Inf(1), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, 2.0, 2.0, math.Inf(-1), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, 2.0, 2.0, 2.0, math.Inf(1), 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, 2.0, 2.0, 2.0, math.Inf(-1), 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, 2.0, 2.0, 2.0, 2.0, math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = forecastingRewards(0.5, 2.0, 2.0, 2.0, 2.0, math.Inf(-1))
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)
}

func (s *MathTestSuite) TestForecastRewardsInfinity() {
	_, err := forecastingRewards(math.MaxFloat64, 3.0, 4.0, 5.0, 6.0, 10.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsIsInfinity)
}

func (s *MathTestSuite) TestForecastRewardsNaN() {
	_, err := forecastingRewards(2.0, 3.0, 0, 0, 0, 0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsIsNaN)
}

func (s *MathTestSuite) TestForecastRewardsZero() {
	result, err := forecastingRewards(0, 3.0, 4.0, 5.0, 6.0, 10.0)
	s.Require().NoError(err)
	s.Require().InDelta(0, result, 0.0001)
}

func (s *MathTestSuite) TestReputerRewardSimple() {
	// W_i = (2 * 2) / (4 + 2 + 2)
	// W_i = 4 / 8
	// W_i = 0.5
	result, err := reputerRewards(4.0, 2.0, 2.0, 2.0)
	s.Require().NoError(err)
	s.Require().InDelta(0.5, result, 0.0001)
}

func (s *MathTestSuite) TestReputerRewardInvalidInput() {
	_, err := reputerRewards(math.NaN(), 2.0, 4.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = reputerRewards(0.5, math.NaN(), 4.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = reputerRewards(0.5, 2.0, math.NaN(), 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = reputerRewards(0.5, 2.0, 4.0, math.NaN())
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = reputerRewards(math.Inf(1), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = reputerRewards(math.Inf(-1), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = reputerRewards(0.5, math.Inf(1), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = reputerRewards(0.5, math.Inf(-1), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = reputerRewards(0.5, 2.0, math.Inf(1), 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = reputerRewards(0.5, 2.0, math.Inf(-1), 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = reputerRewards(0.5, 2.0, 2.0, math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = reputerRewards(0.5, 2.0, 2.0, math.Inf(-1))
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)
}

func (s *MathTestSuite) TestReputerRewardInfinity() {
	_, err := reputerRewards(2.0, 2.0, 2.0, math.MaxFloat64)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsIsInfinity)
}

func (s *MathTestSuite) TestReputerRewardNaN() {
	_, err := reputerRewards(0, 0, 0, 0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsIsNaN)
}

func (s *MathTestSuite) TestReputerRewardZero() {
	result, err := reputerRewards(2, 2.0, 2.0, 0)
	s.Require().NoError(err)
	s.Require().InDelta(0, result, 0.0001)
}

func (s *MathTestSuite) TestForecastingPerformanceScoreSimple() {
	networkInferenceLoss := 100.0
	naiveNetworkInferenceLoss := 1000.0
	score, err := forecastingPerformanceScore(naiveNetworkInferenceLoss, networkInferenceLoss)
	s.Require().NoError(err)
	s.Require().InDelta(1, score, 0.0001)
}

func (s *MathTestSuite) TestForecastingPerformanceScoreInvalidInput() {

	_, err := forecastingPerformanceScore(math.NaN(), 100)
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreInvalidInput)

	_, err = forecastingPerformanceScore(100, math.NaN())
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreInvalidInput)

	_, err = forecastingPerformanceScore(math.Inf(1), 100)
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreInvalidInput)

	_, err = forecastingPerformanceScore(100, math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreInvalidInput)

	_, err = forecastingPerformanceScore(math.Inf(-1), 100)
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreInvalidInput)

	_, err = forecastingPerformanceScore(100, math.Inf(-1))
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreInvalidInput)
}

func (s *MathTestSuite) TestForecastingPerformanceScoreInfinity() {
	_, err := forecastingPerformanceScore(0, 100)
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreIsInfinity)
}

func (s *MathTestSuite) TestForecastingPerformanceScoreNaN() {
	_, err := forecastingPerformanceScore(0, 0)
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreIsNaN)
}

func (s *MathTestSuite) TestSigmoidSimple() {
	x := 0.5
	result, err := sigmoid(x)
	s.Require().NoError(err)
	s.Require().InDelta(0.6224593312018546, result, 0.0001)
}
func (s *MathTestSuite) TestSigmoidInvalidInput() {
	_, err := sigmoid(math.NaN())
	s.Require().ErrorIs(err, emissions.ErrSigmoidInvalidInput)
}
func (s *MathTestSuite) TestSigmoidInfinity() {
	_, err := sigmoid(math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrSigmoidInvalidInput)
}

func (s *MathTestSuite) TestSigmoidNaN() {
	_, err := sigmoid(math.Inf(-1))
	s.Require().ErrorIs(err, emissions.ErrSigmoidInvalidInput)
}

func (s *MathTestSuite) TestForecastingUtilitySimple() {
	a := 8.0
	b := 0.5
	forecastingPerformanceScore := .125
	// 0.1 + 0.4 * sigma(8 * .125 - 0.5)
	// 0.1 + 0.4 * sigma(0.5)
	// 0.1 + 0.4 * 0.6224593312018546
	// 0.34898373248074184

	ret, err := forecastingUtility(forecastingPerformanceScore, a, b)
	s.Require().NoError(err)
	s.Require().InDelta(0.34898373248074184, ret, 0.0001)
}

func (s *MathTestSuite) TestForecastingUtilityInvalidInput() {
	_, err := forecastingUtility(math.NaN(), 0.5, 0.5)
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = forecastingUtility(0.5, math.NaN(), 0.5)
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = forecastingUtility(0.5, 0.5, math.NaN())
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = forecastingUtility(math.Inf(1), 0.5, 0.5)
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = forecastingUtility(0.5, math.Inf(1), 0.5)
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = forecastingUtility(0.5, 0.5, math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = forecastingUtility(math.Inf(-1), 0.5, 0.5)
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = forecastingUtility(0.5, math.Inf(-1), 0.5)
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = forecastingUtility(0.5, 0.5, math.Inf(-1))
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)
}

func (s *MathTestSuite) TestForecastingUtilityInfinity() {
	// todo, not sure if actually reachable code given that sigma will throw first
}

func (s *MathTestSuite) TestForecastingUtilityNaN() {
	// todo, not sure if actually reachable code given that sigma will throw first
}

func (s *MathTestSuite) TestNormalizationFactorSimple() {
	entropyInference := 4.0
	entropyForecasting := 6.0
	chi := 0.5

	// (4+6) / (1-0.5)*4 + 0.5*6
	// 10 / 2 + 3
	// 10 / 5
	// 2

	result, err := normalizationFactor(entropyInference, entropyForecasting, chi)
	s.Require().NoError(err)

	s.Require().InDelta(2.0, result, 0.0001)
}

func (s *MathTestSuite) TestNormalizationFactorInvalidInput() {
	_, err := normalizationFactor(math.NaN(), 1, 1)
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = normalizationFactor(1, math.NaN(), 1)
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = normalizationFactor(1, 1, math.NaN())
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = normalizationFactor(math.Inf(1), 1, 1)
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = normalizationFactor(1, math.Inf(1), 1)
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = normalizationFactor(1, 1, math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = normalizationFactor(math.Inf(-1), 1, 1)
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = normalizationFactor(1, math.Inf(-1), 1)
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = normalizationFactor(1, 1, math.Inf(-1))
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)
}

func (s *MathTestSuite) TestNormalizationFactorInfinity() {
	_, err := normalizationFactor(math.MaxFloat64, math.MaxFloat64, 0.2)
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorIsInfinity)
}

func (s *MathTestSuite) TestNormalizationFactorNaN() {
	// todo, don't think this code path is actually possible
	// given that emissions.ErrNormilazationFactorInvalidInput will be thrown first
}
