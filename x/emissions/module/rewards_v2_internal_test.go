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
