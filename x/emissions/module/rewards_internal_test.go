package module_test

import (
	"math"
	"testing"

	"github.com/allora-network/allora-chain/x/emissions/module"
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
	result, err := module.Phi(p, x)
	s.Require().NoError(err)
	s.Require().InDelta(64, result, 0.001)
}

func (s *MathTestSuite) TestFailPhiXTooLarge() {
	// values of x that are too large should overflow the size limit of a float64
	// max float value is 1.7976931348623157e+308
	// so we test that edge condition
	x := float64(709)
	p := float64(2)
	_, err := module.Phi(p, x)
	s.Require().NoError(err)

	x = float64(710)
	_, err = module.Phi(p, x)
	s.Require().ErrorIs(err, emissions.ErrEToTheXExponentiationIsInfinity)
}

func (s *MathTestSuite) TestFailPhiPTooLarge() {
	// values of p that are too large should overflow the size limit of a float64
	// so we test that edge condition
	x := float64(709)
	p := float64(108)
	_, err := module.Phi(p, x)
	s.Require().NoError(err)

	p = float64(109)
	_, err = module.Phi(p, x)
	s.Require().ErrorIs(err, emissions.ErrLnToThePExponentiationIsInfinity)
}

func (s *MathTestSuite) TestPhiInvalidInputs() {
	// test that invalid inputs return the correct error
	_, err := module.Phi(math.Inf(1), 3)
	s.Require().ErrorIs(err, emissions.ErrPhiInvalidInput)
	_, err = module.Phi(math.Inf(-1), 3)
	s.Require().ErrorIs(err, emissions.ErrPhiInvalidInput)
	_, err = module.Phi(3, math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrPhiInvalidInput)
	_, err = module.Phi(3, math.Inf(-1))
	s.Require().ErrorIs(err, emissions.ErrPhiInvalidInput)

	_, err = module.Phi(math.NaN(), 3)
	s.Require().ErrorIs(err, emissions.ErrPhiInvalidInput)
	_, err = module.Phi(3, math.NaN())
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

	result, err := module.GetAdjustedStake(
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

	result, err := module.ExponentialMovingAverage(alpha, current, previous)
	s.Require().NoError(err)
	s.Require().InDelta(expected, result, 0.0001)
}

func (s *MathTestSuite) TestNormalizeAgainstSlice() {
	v := 2.0
	a := []float64{2.0, 3.0, 5.0}
	expected := 0.2

	result, err := module.NormalizeAgainstSlice(v, a)

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
	result, err := module.Entropy(f_ij, N_i_eff, N_i, beta)
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
			_, err := module.Entropy(nil, tc.N_eff, tc.numParticipants, tc.beta)
			s.Require().ErrorIs(err, emissions.ErrEntropyInvalidInput)
		})
	}
}

func (s *MathTestSuite) TestEntropyInvalidInputAllFsInf() {
	allFs := []float64{1, 2, math.Inf(0), 4}
	_, err := module.Entropy(allFs, 100, 10, 0.5)
	s.Require().ErrorIs(err, emissions.ErrEntropyInvalidInput)
}

func (s *MathTestSuite) TestEntropyInvalidInputAllFsNan() {
	allFs := []float64{1, 2, math.NaN(), 4}
	_, err := module.Entropy(allFs, 100, 10, 0.5)
	s.Require().ErrorIs(err, emissions.ErrEntropyInvalidInput)
}

func (s *MathTestSuite) TestEntropyInfinity() {
	allFs := []float64{0.1, 0.2, 0.3, 0.4}

	_, err := module.Entropy(allFs, 100, 10, math.MaxFloat64)
	s.Require().ErrorIs(err, emissions.ErrEntropyIsInfinity)
}

func (s *MathTestSuite) TestEntropyNaN() {
	allFs := []float64{0, 0, 0, 0}

	_, err := module.Entropy(allFs, 100, 10, 0)
	s.Require().ErrorIs(err, emissions.ErrEntropyIsNaN)
}

func (s *MathTestSuite) TestNumberRatio() {
	rewardFractions := []float64{0.2, 0.3, 0.4, 0.5, 0.6, 0.7}

	// 1 / (0.2 *0.2 + 0.3*0.3 + 0.4*0.4 + 0.5*0.5 + 0.6*0.6 + 0.7*0.7)
	// 1 / (0.04 + 0.09 + 0.16 + 0.25 + 0.36 + 0.49)
	// 1 / 1.39 = 0.719424460431654676259005145797598627787307032590051458
	expected := 0.7194244604316546762589928057553956834532374100

	result, err := module.NumberRatio(rewardFractions)
	s.Require().NoError(err, "Error calculating number ratio")
	s.Require().InDelta(expected, result, 0.0001)
}

func (s *MathTestSuite) TestNumberRatioZeroFractions() {
	zeroFractions := []float64{0.0}

	_, err := module.NumberRatio(zeroFractions)
	s.Require().ErrorIs(err, emissions.ErrNumberRatioDivideByZero)
}

func (s *MathTestSuite) TestNumberRatioEmptyList() {
	emptyFractions := []float64{}

	_, err := module.NumberRatio(emptyFractions)
	s.Require().ErrorIs(err, emissions.ErrNumberRatioInvalidSliceLength)
}

func (s *MathTestSuite) TestNumberRatioNaNFractions() {
	invalidFractions := []float64{0.2, math.NaN(), 0.5}
	_, err := module.NumberRatio(invalidFractions)
	s.Require().ErrorIs(err, emissions.ErrNumberRatioInvalidInput)
}

func (s *MathTestSuite) TestNumberRatioInfiniteFractions() {

	infFractions := []float64{0.2, math.Inf(1), 0.5}

	_, err := module.NumberRatio(infFractions)
	s.Require().ErrorIs(err, emissions.ErrNumberRatioInvalidInput)
}

func (s *MathTestSuite) TestInferenceRewardsSimple() {
	// U_i = ((1 - 0.5) * 2 * 2 * 2 ) / (2 + 2 + 4)
	// U_i = 0.5 * 8 / 8
	// U_i = 0.5
	infRewards, err := module.InferenceRewards(
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
	_, err := module.InferenceRewards(math.NaN(), 2.0, 4.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, math.NaN(), 4.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, 2.0, math.NaN(), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, 2.0, 4.0, math.NaN(), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, 2.0, 4.0, 2.0, math.NaN(), 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, 2.0, 4.0, 2.0, 2.0, math.NaN())
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(math.Inf(1), 2.0, 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(math.Inf(-1), 2.0, 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, math.Inf(1), 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, math.Inf(-1), 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, 2.0, math.Inf(1), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, 2.0, math.Inf(-1), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, 2.0, 2.0, math.Inf(1), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, 2.0, 2.0, math.Inf(-1), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, 2.0, 2.0, 2.0, math.Inf(1), 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, 2.0, 2.0, 2.0, math.Inf(-1), 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, 2.0, 2.0, 2.0, 2.0, math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)

	_, err = module.InferenceRewards(0.5, 2.0, 2.0, 2.0, 2.0, math.Inf(-1))
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsInvalidInput)
}

func (s *MathTestSuite) TestInferenceRewardsInfinity() {
	_, err := module.InferenceRewards(math.MaxFloat64, math.MaxFloat64, math.MaxFloat64, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsIsInfinity)
}

func (s *MathTestSuite) TestInferenceRewardsNaN() {
	_, err := module.InferenceRewards(0.5, 2.0, 0, 0, 0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrInferenceRewardsIsNaN)
}

func (s *MathTestSuite) TestInferenceRewardsZero() {
	result, err := module.InferenceRewards(1, 2.0, 10, 20, 30, 2.0)
	s.Require().NoError(err)
	s.Require().InDelta(0, result, 0.0001)
}

func (s *MathTestSuite) TestForecastRewardsSimple() {
	// V_i = (2 * 3 * 4 * 5) / (6 + 4 + 10)
	// V_i = 120 / 20
	// V_i = 6
	result, err := module.ForecastingRewards(2.0, 3.0, 6.0, 4.0, 10.0, 5.0)
	s.Require().NoError(err)
	s.Require().InDelta(6.0, result, 0.0001)
}

func (s *MathTestSuite) TestForecastRewardsInvalidInput() {
	_, err := module.ForecastingRewards(math.NaN(), 2.0, 4.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, math.NaN(), 4.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, 2.0, math.NaN(), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, 2.0, 4.0, math.NaN(), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, 2.0, 4.0, 2.0, math.NaN(), 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, 2.0, 4.0, 2.0, 2.0, math.NaN())
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(math.Inf(1), 2.0, 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(math.Inf(-1), 2.0, 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, math.Inf(1), 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, math.Inf(-1), 2.0, 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, 2.0, math.Inf(1), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, 2.0, math.Inf(-1), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, 2.0, 2.0, math.Inf(1), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, 2.0, 2.0, math.Inf(-1), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, 2.0, 2.0, 2.0, math.Inf(1), 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, 2.0, 2.0, 2.0, math.Inf(-1), 2.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, 2.0, 2.0, 2.0, 2.0, math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)

	_, err = module.ForecastingRewards(0.5, 2.0, 2.0, 2.0, 2.0, math.Inf(-1))
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsInvalidInput)
}

func (s *MathTestSuite) TestForecastRewardsInfinity() {
	_, err := module.ForecastingRewards(math.MaxFloat64, 3.0, 4.0, 5.0, 6.0, 10.0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsIsInfinity)
}

func (s *MathTestSuite) TestForecastRewardsNaN() {
	_, err := module.ForecastingRewards(2.0, 3.0, 0, 0, 0, 0)
	s.Require().ErrorIs(err, emissions.ErrForecastingRewardsIsNaN)
}

func (s *MathTestSuite) TestForecastRewardsZero() {
	result, err := module.ForecastingRewards(0, 3.0, 4.0, 5.0, 6.0, 10.0)
	s.Require().NoError(err)
	s.Require().InDelta(0, result, 0.0001)
}

func (s *MathTestSuite) TestReputerRewardSimple() {
	// W_i = (2 * 2) / (4 + 2 + 2)
	// W_i = 4 / 8
	// W_i = 0.5
	result, err := module.ReputerRewards(4.0, 2.0, 2.0, 2.0)
	s.Require().NoError(err)
	s.Require().InDelta(0.5, result, 0.0001)
}

func (s *MathTestSuite) TestReputerRewardInvalidInput() {
	_, err := module.ReputerRewards(math.NaN(), 2.0, 4.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = module.ReputerRewards(0.5, math.NaN(), 4.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = module.ReputerRewards(0.5, 2.0, math.NaN(), 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = module.ReputerRewards(0.5, 2.0, 4.0, math.NaN())
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = module.ReputerRewards(math.Inf(1), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = module.ReputerRewards(math.Inf(-1), 2.0, 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = module.ReputerRewards(0.5, math.Inf(1), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = module.ReputerRewards(0.5, math.Inf(-1), 2.0, 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = module.ReputerRewards(0.5, 2.0, math.Inf(1), 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = module.ReputerRewards(0.5, 2.0, math.Inf(-1), 2.0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = module.ReputerRewards(0.5, 2.0, 2.0, math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)

	_, err = module.ReputerRewards(0.5, 2.0, 2.0, math.Inf(-1))
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsInvalidInput)
}

func (s *MathTestSuite) TestReputerRewardInfinity() {
	_, err := module.ReputerRewards(2.0, 2.0, 2.0, math.MaxFloat64)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsIsInfinity)
}

func (s *MathTestSuite) TestReputerRewardNaN() {
	_, err := module.ReputerRewards(0, 0, 0, 0)
	s.Require().ErrorIs(err, emissions.ErrReputerRewardsIsNaN)
}

func (s *MathTestSuite) TestReputerRewardZero() {
	result, err := module.ReputerRewards(2, 2.0, 2.0, 0)
	s.Require().NoError(err)
	s.Require().InDelta(0, result, 0.0001)
}

func (s *MathTestSuite) TestForecastingPerformanceScoreSimple() {
	networkInferenceLoss := 100.0
	naiveNetworkInferenceLoss := 1000.0
	score, err := module.ForecastingPerformanceScore(naiveNetworkInferenceLoss, networkInferenceLoss)
	s.Require().NoError(err)
	s.Require().InDelta(1, score, 0.0001)
}

func (s *MathTestSuite) TestForecastingPerformanceScoreInvalidInput() {

	_, err := module.ForecastingPerformanceScore(math.NaN(), 100)
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreInvalidInput)

	_, err = module.ForecastingPerformanceScore(100, math.NaN())
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreInvalidInput)

	_, err = module.ForecastingPerformanceScore(math.Inf(1), 100)
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreInvalidInput)

	_, err = module.ForecastingPerformanceScore(100, math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreInvalidInput)

	_, err = module.ForecastingPerformanceScore(math.Inf(-1), 100)
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreInvalidInput)

	_, err = module.ForecastingPerformanceScore(100, math.Inf(-1))
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreInvalidInput)
}

func (s *MathTestSuite) TestForecastingPerformanceScoreInfinity() {
	_, err := module.ForecastingPerformanceScore(0, 100)
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreIsInfinity)
}

func (s *MathTestSuite) TestForecastingPerformanceScoreNaN() {
	_, err := module.ForecastingPerformanceScore(0, 0)
	s.Require().ErrorIs(err, emissions.ErrForecastingPerformanceScoreIsNaN)
}

func (s *MathTestSuite) TestSigmoidSimple() {
	x := 0.5
	result, err := module.Sigmoid(x)
	s.Require().NoError(err)
	s.Require().InDelta(0.6224593312018546, result, 0.0001)
}
func (s *MathTestSuite) TestSigmoidInvalidInput() {
	_, err := module.Sigmoid(math.NaN())
	s.Require().ErrorIs(err, emissions.ErrSigmoidInvalidInput)
}
func (s *MathTestSuite) TestSigmoidInfinity() {
	_, err := module.Sigmoid(math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrSigmoidInvalidInput)
}

func (s *MathTestSuite) TestSigmoidNaN() {
	_, err := module.Sigmoid(math.Inf(-1))
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

	ret, err := module.ForecastingUtility(forecastingPerformanceScore, a, b)
	s.Require().NoError(err)
	s.Require().InDelta(0.34898373248074184, ret, 0.0001)
}

func (s *MathTestSuite) TestForecastingUtilityInvalidInput() {
	_, err := module.ForecastingUtility(math.NaN(), 0.5, 0.5)
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = module.ForecastingUtility(0.5, math.NaN(), 0.5)
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = module.ForecastingUtility(0.5, 0.5, math.NaN())
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = module.ForecastingUtility(math.Inf(1), 0.5, 0.5)
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = module.ForecastingUtility(0.5, math.Inf(1), 0.5)
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = module.ForecastingUtility(0.5, 0.5, math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = module.ForecastingUtility(math.Inf(-1), 0.5, 0.5)
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = module.ForecastingUtility(0.5, math.Inf(-1), 0.5)
	s.Require().ErrorIs(err, emissions.ErrForecastingUtilityInvalidInput)

	_, err = module.ForecastingUtility(0.5, 0.5, math.Inf(-1))
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

	result, err := module.NormalizationFactor(entropyInference, entropyForecasting, chi)
	s.Require().NoError(err)

	s.Require().InDelta(2.0, result, 0.0001)
}

func (s *MathTestSuite) TestNormalizationFactorInvalidInput() {
	_, err := module.NormalizationFactor(math.NaN(), 1, 1)
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = module.NormalizationFactor(1, math.NaN(), 1)
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = module.NormalizationFactor(1, 1, math.NaN())
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = module.NormalizationFactor(math.Inf(1), 1, 1)
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = module.NormalizationFactor(1, math.Inf(1), 1)
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = module.NormalizationFactor(1, 1, math.Inf(1))
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = module.NormalizationFactor(math.Inf(-1), 1, 1)
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = module.NormalizationFactor(1, math.Inf(-1), 1)
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)

	_, err = module.NormalizationFactor(1, 1, math.Inf(-1))
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorInvalidInput)
}

func (s *MathTestSuite) TestNormalizationFactorInfinity() {
	_, err := module.NormalizationFactor(math.MaxFloat64, math.MaxFloat64, 0.2)
	s.Require().ErrorIs(err, emissions.ErrNormalizationFactorIsInfinity)
}

func (s *MathTestSuite) TestNormalizationFactorNaN() {
	// todo, don't think this code path is actually possible
	// given that emissions.ErrNormilazationFactorInvalidInput will be thrown first
}

func TestStdDev(t *testing.T) {
	tests := []struct {
		name string
		data []float64
		want float64
	}{
		{
			name: "basic",
			data: []float64{-0.00675, -0.00622, -0.01502, -0.01214, 0.00392, 0.00559, 0.0438, 0.04304, 0.09719, 0.09675},
			want: 0.041014924273483966,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := module.StdDev(tt.data); math.Abs(got-tt.want) > 1e-5 {
				t.Errorf("StdDev() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetWorkerRewardFractions(t *testing.T) {
	tests := []struct {
		name    string
		scores  [][]float64
		preward float64
		want    []float64
		wantErr bool
	}{
		{
			name: "basic",
			scores: [][]float64{
				{-0.00675, -0.00622, -0.00388},
				{-0.01502, -0.01214, -0.01554},
				{0.00392, 0.00559, 0.00545},
				{0.0438, 0.04304, 0.03906},
				{0.09719, 0.09675, 0.09418},
			},
			preward: 1.5,
			want:    []float64{0.07671, 0.05531, 0.09829, 0.21537, 0.55432},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := module.GetWorkerRewardFractions(tt.scores, tt.preward)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetWorkerRewardFractions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !slicesAreApproxEqual(got, tt.want, 1e-4) {
				t.Errorf("GetWorkerRewardFractions() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetReputerRewardFractions(t *testing.T) {
	tests := []struct {
		name    string
		stakes  []float64
		scores  []float64
		preward float64
		want    []float64
		wantErr bool
	}{
		{
			name:    "basic",
			stakes:  []float64{1178377.89152, 385287.87376, 395488.13091, 208201.11762, 369044.55988},
			scores:  []float64{17.53839, 22.63517, 26.28035, 13.51383, 15.08629},
			preward: 1,
			want:    []float64{0.42911, 0.18108, 0.2158, 0.05842, 0.1156},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := module.GetReputerRewardFractions(tt.stakes, tt.scores, tt.preward)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetReputerRewardFractions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !slicesAreApproxEqual(got, tt.want, 1e-5) {
				t.Errorf("GetReputerRewardFractions() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStakeWeightedLossMatrix(t *testing.T) {
	tests := []struct {
		name                   string
		reputersAdjustedStakes []float64
		reputersReportedLosses [][]float64
		want                   []float64
		wantErr                bool
	}{
		{
			name:                   "basic",
			reputersAdjustedStakes: []float64{1.0, 0.76188, 0.7816, 0.40664, 0.71687},
			reputersReportedLosses: [][]float64{
				{0.0112, 0.00231, 0.02274, 0.01299, 0.02515, 0.0185, 0.01018, 0.02105, 0.01041, 0.0183, 0.01022, 0.01333, 0.01298, 0.01023, 0.01268, 0.01381, 0.01731, 0.01238, 0.01168, 0.00929, 0.01212, 0.01806, 0.01901, 0.01828, 0.01522, 0.01833, 0.0101, 0.01224, 0.01226, 0.01474, 0.01218, 0.01604, 0.01149, 0.02075, 0.00818, 0.0116, 0.01127, 0.01495, 0.00689, 0.0108, 0.01417, 0.0124, 0.01588, 0.01012, 0.01467, 0.0128, 0.01234, 0.0148, 0.01046, 0.01192, 0.01381, 0.01687, 0.01136, 0.01185, 0.01568, 0.00949, 0.01339},
				{0.01635, 0.00179, 0.03396, 0.0153, 0.01988, 0.00962, 0.01191, 0.01616, 0.01417, 0.01216, 0.01292, 0.01564, 0.01323, 0.01261, 0.01145, 0.0163, 0.014, 0.01373, 0.01453, 0.01207, 0.01641, 0.01601, 0.01114, 0.01259, 0.01589, 0.01229, 0.01309, 0.0138, 0.01162, 0.01145, 0.01013, 0.01208, 0.0111, 0.0118, 0.01374, 0.01428, 0.01791, 0.01288, 0.01161, 0.01151, 0.01148, 0.01284, 0.01239, 0.01023, 0.01712, 0.0116, 0.01639, 0.01043, 0.01308, 0.01455, 0.01607, 0.01205, 0.01357, 0.01108, 0.01633, 0.01208, 0.01278},
				{0.01345, 0.00209, 0.03249, 0.01688, 0.02126, 0.01338, 0.0116, 0.01605, 0.0133, 0.01407, 0.01367, 0.01244, 0.0145, 0.01262, 0.01348, 0.01684, 0.01148, 0.01705, 0.01714, 0.0124, 0.0125, 0.01462, 0.01274, 0.01407, 0.01667, 0.01316, 0.01628, 0.01373, 0.01409, 0.01603, 0.01378, 0.01143, 0.013, 0.01644, 0.01528, 0.01441, 0.01404, 0.01402, 0.01479, 0.01417, 0.01244, 0.0116, 0.01419, 0.01497, 0.01629, 0.01514, 0.01133, 0.01339, 0.01053, 0.01424, 0.01428, 0.01446, 0.01805, 0.01229, 0.01586, 0.01234, 0.01513},
				{0.01675, 0.00318, 0.02623, 0.02734, 0.03526, 0.02733, 0.01697, 0.01619, 0.01925, 0.02018, 0.01735, 0.01922, 0.02225, 0.0189, 0.01923, 0.03193, 0.01956, 0.01763, 0.01975, 0.01466, 0.02021, 0.01803, 0.01438, 0.01929, 0.02305, 0.02223, 0.02445, 0.01967, 0.02292, 0.01878, 0.01751, 0.02695, 0.01849, 0.01658, 0.01948, 0.01594, 0.02318, 0.01906, 0.01607, 0.01369, 0.01686, 0.01314, 0.01936, 0.01518, 0.018, 0.02212, 0.02259, 0.01674, 0.02944, 0.01796, 0.02187, 0.01895, 0.01637, 0.01594, 0.01608, 0.02203, 0.01486},
				{0.02093, 0.00213, 0.02462, 0.0203, 0.03115, 0.01, 0.01545, 0.01785, 0.01662, 0.01156, 0.02284, 0.01475, 0.01331, 0.01592, 0.01462, 0.02333, 0.01836, 0.01465, 0.0186, 0.01566, 0.01506, 0.01678, 0.01423, 0.01658, 0.01741, 0.03491, 0.01408, 0.01191, 0.01572, 0.01355, 0.01477, 0.01662, 0.01128, 0.02581, 0.01718, 0.01705, 0.01251, 0.02158, 0.01187, 0.01504, 0.0135, 0.02432, 0.01602, 0.01194, 0.0153, 0.0199, 0.01673, 0.01049, 0.02068, 0.01573, 0.01487, 0.02639, 0.01981, 0.02123, 0.02134, 0.0217, 0.01177},
			},
			want: []float64{
				0.01489, 0.00219, 0.02752, 0.01684, 0.02502, 0.01395, 0.01242, 0.01769, 0.01372,
				0.01469, 0.01416, 0.01442, 0.01423, 0.01304, 0.01354, 0.01813, 0.01556, 0.01455,
				0.0154, 0.01215, 0.01435, 0.01659, 0.01431, 0.01579, 0.01683, 0.01821, 0.01389,
				0.01348, 0.01405, 0.01439, 0.01301, 0.01501, 0.0123, 0.01788, 0.01325, 0.01417,
				0.01438, 0.01578, 0.01104, 0.0127, 0.01332, 0.01414, 0.01508, 0.01191, 0.01598,
				0.01506, 0.01459, 0.01277, 0.01406, 0.01426, 0.01532, 0.01683, 0.0151, 0.01364,
				0.01688, 0.01362, 0.01342,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := module.GetStakeWeightedLossMatrix(tt.reputersAdjustedStakes, tt.reputersReportedLosses)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStakeWeightedLossMatrix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// convert to 10^x
			for i, v := range got {
				got[i] = math.Pow(10, v)
			}
			if !slicesAreApproxEqual(got, tt.want, 1e-5) {
				t.Errorf("GetStakeWeightedLossMatrix() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStakeWeightedLoss(t *testing.T) {
	tests := []struct {
		name                   string
		reputersStakes         []float64
		reputersReportedLosses []float64
		want                   float64
		wantErr                bool
	}{
		{
			name:                   "simple average",
			reputersStakes:         []float64{1176644.37627, 384623.3607, 394676.13226, 207999.66194, 368582.76542},
			reputersReportedLosses: []float64{0.0112, 0.01635, 0.01345, 0.01675, 0.02093},
			want:                   0.01381883491416319,
			wantErr:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := module.GetStakeWeightedLoss(tt.reputersStakes, tt.reputersReportedLosses)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStakeWeightedLoss() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !(math.Abs(math.Pow(10, got)-tt.want) <= 1e-5) {
				t.Errorf("GetStakeWeightedLoss() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetWorkerScore(t *testing.T) {
	tests := []struct {
		name         string
		losses       float64
		lossesOneOut float64
		want         float64
	}{
		{"basic", 0.011411892282242868, 0.01344474872292, 0.07119502617735574},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := module.GetWorkerScore(tt.losses, tt.lossesOneOut)
			if !(math.Abs(got-tt.want) <= 1e-14) {
				t.Errorf("GetWorkerScore() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFinalWorkerScoreForecastTask(t *testing.T) {
	tests := []struct {
		name        string
		scoreOneIn  float64
		scoreOneOut float64
		fUniqueAgg  float64
		want        float64
	}{
		{"basic", 0.07300629674057668, -0.009510726019112292, 0.0625, -0.004353412096631731},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := module.GetFinalWorkerScoreForecastTask(tt.scoreOneIn, tt.scoreOneOut, tt.fUniqueAgg)
			if got != tt.want {
				t.Errorf("GetFinalWorkerScoreForecastTask() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAllConsensusScores(t *testing.T) {
	tests := []struct {
		name                     string
		allLosses                [][]float64
		stakes                   []float64
		allListeningCoefficients []float64
		numReputers              int
		want                     []float64
		wantErr                  bool
	}{
		{
			name: "basic",
			allLosses: [][]float64{
				{0.0112, 0.00231, 0.02274, 0.01299, 0.02515, 0.0185, 0.01018, 0.02105, 0.01041, 0.0183, 0.01022, 0.01333, 0.01298, 0.01023, 0.01268, 0.01381, 0.01731, 0.01238, 0.01168, 0.00929, 0.01212, 0.01806, 0.01901, 0.01828, 0.01522, 0.01833, 0.0101, 0.01224, 0.01226, 0.01474, 0.01218, 0.01604, 0.01149, 0.02075, 0.00818, 0.0116, 0.01127, 0.01495, 0.00689, 0.0108, 0.01417, 0.0124, 0.01588, 0.01012, 0.01467, 0.0128, 0.01234, 0.0148, 0.01046, 0.01192, 0.01381, 0.01687, 0.01136, 0.01185, 0.01568, 0.00949, 0.01339},
				{0.01635, 0.00179, 0.03396, 0.0153, 0.01988, 0.00962, 0.01191, 0.01616, 0.01417, 0.01216, 0.01292, 0.01564, 0.01323, 0.01261, 0.01145, 0.0163, 0.014, 0.01373, 0.01453, 0.01207, 0.01641, 0.01601, 0.01114, 0.01259, 0.01589, 0.01229, 0.01309, 0.0138, 0.01162, 0.01145, 0.01013, 0.01208, 0.0111, 0.0118, 0.01374, 0.01428, 0.01791, 0.01288, 0.01161, 0.01151, 0.01148, 0.01284, 0.01239, 0.01023, 0.01712, 0.0116, 0.01639, 0.01043, 0.01308, 0.01455, 0.01607, 0.01205, 0.01357, 0.01108, 0.01633, 0.01208, 0.01278},
				{0.01345, 0.00209, 0.03249, 0.01688, 0.02126, 0.01338, 0.0116, 0.01605, 0.0133, 0.01407, 0.01367, 0.01244, 0.0145, 0.01262, 0.01348, 0.01684, 0.01148, 0.01705, 0.01714, 0.0124, 0.0125, 0.01462, 0.01274, 0.01407, 0.01667, 0.01316, 0.01628, 0.01373, 0.01409, 0.01603, 0.01378, 0.01143, 0.013, 0.01644, 0.01528, 0.01441, 0.01404, 0.01402, 0.01479, 0.01417, 0.01244, 0.0116, 0.01419, 0.01497, 0.01629, 0.01514, 0.01133, 0.01339, 0.01053, 0.01424, 0.01428, 0.01446, 0.01805, 0.01229, 0.01586, 0.01234, 0.01513},
				{0.01675, 0.00318, 0.02623, 0.02734, 0.03526, 0.02733, 0.01697, 0.01619, 0.01925, 0.02018, 0.01735, 0.01922, 0.02225, 0.0189, 0.01923, 0.03193, 0.01956, 0.01763, 0.01975, 0.01466, 0.02021, 0.01803, 0.01438, 0.01929, 0.02305, 0.02223, 0.02445, 0.01967, 0.02292, 0.01878, 0.01751, 0.02695, 0.01849, 0.01658, 0.01948, 0.01594, 0.02318, 0.01906, 0.01607, 0.01369, 0.01686, 0.01314, 0.01936, 0.01518, 0.018, 0.02212, 0.02259, 0.01674, 0.02944, 0.01796, 0.02187, 0.01895, 0.01637, 0.01594, 0.01608, 0.02203, 0.01486},
				{0.02093, 0.00213, 0.02462, 0.0203, 0.03115, 0.01, 0.01545, 0.01785, 0.01662, 0.01156, 0.02284, 0.01475, 0.01331, 0.01592, 0.01462, 0.02333, 0.01836, 0.01465, 0.0186, 0.01566, 0.01506, 0.01678, 0.01423, 0.01658, 0.01741, 0.03491, 0.01408, 0.01191, 0.01572, 0.01355, 0.01477, 0.01662, 0.01128, 0.02581, 0.01718, 0.01705, 0.01251, 0.02158, 0.01187, 0.01504, 0.0135, 0.02432, 0.01602, 0.01194, 0.0153, 0.0199, 0.01673, 0.01049, 0.02068, 0.01573, 0.01487, 0.02639, 0.01981, 0.02123, 0.02134, 0.0217, 0.01177},
			},
			stakes:                   []float64{1176644.37627, 384623.3607, 394676.13226, 207999.66194, 368582.76542},
			allListeningCoefficients: []float64{1.0, 1.0, 1.0, 1.0, 1.0},
			numReputers:              5,
			want:                     []float64{17.4346, 20.13897, 24.08276, 11.41393, 15.33319},
			wantErr:                  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := module.GetAllConsensusScores(tt.allLosses, tt.stakes, tt.allListeningCoefficients, tt.numReputers)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllConsensusScores() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !slicesAreApproxEqual(got, tt.want, 1e-2) {
				t.Errorf("GetAllConsensusScores() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAllReputersOutput(t *testing.T) {
	tests := []struct {
		name                string
		allLosses           [][]float64
		stakes              []float64
		consensusScores     []float64
		initialCoefficients []float64
		numReputers         int
		wantScores          []float64
		wantCoefficients    []float64
		wantErr             bool
	}{
		{
			name: "basic",
			allLosses: [][]float64{
				{0.0112, 0.00231, 0.02274, 0.01299, 0.02515, 0.0185, 0.01018, 0.02105, 0.01041, 0.0183, 0.01022, 0.01333, 0.01298, 0.01023, 0.01268, 0.01381, 0.01731, 0.01238, 0.01168, 0.00929, 0.01212, 0.01806, 0.01901, 0.01828, 0.01522, 0.01833, 0.0101, 0.01224, 0.01226, 0.01474, 0.01218, 0.01604, 0.01149, 0.02075, 0.00818, 0.0116, 0.01127, 0.01495, 0.00689, 0.0108, 0.01417, 0.0124, 0.01588, 0.01012, 0.01467, 0.0128, 0.01234, 0.0148, 0.01046, 0.01192, 0.01381, 0.01687, 0.01136, 0.01185, 0.01568, 0.00949, 0.01339},
				{0.01635, 0.00179, 0.03396, 0.0153, 0.01988, 0.00962, 0.01191, 0.01616, 0.01417, 0.01216, 0.01292, 0.01564, 0.01323, 0.01261, 0.01145, 0.0163, 0.014, 0.01373, 0.01453, 0.01207, 0.01641, 0.01601, 0.01114, 0.01259, 0.01589, 0.01229, 0.01309, 0.0138, 0.01162, 0.01145, 0.01013, 0.01208, 0.0111, 0.0118, 0.01374, 0.01428, 0.01791, 0.01288, 0.01161, 0.01151, 0.01148, 0.01284, 0.01239, 0.01023, 0.01712, 0.0116, 0.01639, 0.01043, 0.01308, 0.01455, 0.01607, 0.01205, 0.01357, 0.01108, 0.01633, 0.01208, 0.01278},
				{0.01345, 0.00209, 0.03249, 0.01688, 0.02126, 0.01338, 0.0116, 0.01605, 0.0133, 0.01407, 0.01367, 0.01244, 0.0145, 0.01262, 0.01348, 0.01684, 0.01148, 0.01705, 0.01714, 0.0124, 0.0125, 0.01462, 0.01274, 0.01407, 0.01667, 0.01316, 0.01628, 0.01373, 0.01409, 0.01603, 0.01378, 0.01143, 0.013, 0.01644, 0.01528, 0.01441, 0.01404, 0.01402, 0.01479, 0.01417, 0.01244, 0.0116, 0.01419, 0.01497, 0.01629, 0.01514, 0.01133, 0.01339, 0.01053, 0.01424, 0.01428, 0.01446, 0.01805, 0.01229, 0.01586, 0.01234, 0.01513},
				{0.01675, 0.00318, 0.02623, 0.02734, 0.03526, 0.02733, 0.01697, 0.01619, 0.01925, 0.02018, 0.01735, 0.01922, 0.02225, 0.0189, 0.01923, 0.03193, 0.01956, 0.01763, 0.01975, 0.01466, 0.02021, 0.01803, 0.01438, 0.01929, 0.02305, 0.02223, 0.02445, 0.01967, 0.02292, 0.01878, 0.01751, 0.02695, 0.01849, 0.01658, 0.01948, 0.01594, 0.02318, 0.01906, 0.01607, 0.01369, 0.01686, 0.01314, 0.01936, 0.01518, 0.018, 0.02212, 0.02259, 0.01674, 0.02944, 0.01796, 0.02187, 0.01895, 0.01637, 0.01594, 0.01608, 0.02203, 0.01486},
				{0.02093, 0.00213, 0.02462, 0.0203, 0.03115, 0.01, 0.01545, 0.01785, 0.01662, 0.01156, 0.02284, 0.01475, 0.01331, 0.01592, 0.01462, 0.02333, 0.01836, 0.01465, 0.0186, 0.01566, 0.01506, 0.01678, 0.01423, 0.01658, 0.01741, 0.03491, 0.01408, 0.01191, 0.01572, 0.01355, 0.01477, 0.01662, 0.01128, 0.02581, 0.01718, 0.01705, 0.01251, 0.02158, 0.01187, 0.01504, 0.0135, 0.02432, 0.01602, 0.01194, 0.0153, 0.0199, 0.01673, 0.01049, 0.02068, 0.01573, 0.01487, 0.02639, 0.01981, 0.02123, 0.02134, 0.0217, 0.01177},
			},
			stakes:              []float64{1176644.37627, 384623.3607, 394676.13226, 207999.66194, 368582.76542},
			initialCoefficients: []float64{1.0, 1.0, 1.0, 1.0, 1.0},
			numReputers:         5,
			wantScores:          []float64{17.53436, 20.29489, 24.26994, 11.36754, 15.21749},
			wantCoefficients:    []float64{0.99942, 1.0, 1.0, 0.96574, 0.95346},
			wantErr:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotScores, gotCoefficients, err := module.GetAllReputersOutput(tt.allLosses, tt.stakes, tt.initialCoefficients, tt.numReputers)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllReputersOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !slicesAreApproxEqual(gotScores, tt.wantScores, 1e-4) {
				t.Errorf("GetAllReputersOutput() gotScores = %v, want %v", gotScores, tt.wantScores)
			}

			if !slicesAreApproxEqual(gotCoefficients, tt.wantCoefficients, 1e-4) {
				t.Errorf("GetAllReputersOutput() gotCoefficients = %v, want %v", gotCoefficients, tt.wantCoefficients)
			}
		})
	}
}

// Helper function to compare two slices of float64 within a tolerance
func slicesAreApproxEqual(a, b []float64, tolerance float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > tolerance {
			return false
		}
	}
	return true
}
