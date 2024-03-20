package equations

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
