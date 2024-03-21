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
