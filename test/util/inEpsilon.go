package alloratestutil

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/require"
)

func (s *AlloraTestUtilSuite) inEpsilon(value alloraMath.Dec, target string, epsilon string, require *require.Assertions) {
	epsilonDec := alloraMath.MustNewDecFromString(epsilon)
	targetDec := alloraMath.MustNewDecFromString(target)
	one := alloraMath.MustNewDecFromString("1")

	lowerMultiplier, err := one.Sub(epsilonDec)
	require.NoError(err)
	lowerBound, err := targetDec.Mul(lowerMultiplier)
	require.NoError(err)

	upperMultiplier, err := one.Add(epsilonDec)
	require.NoError(err)
	upperBound, err := targetDec.Mul(upperMultiplier)
	require.NoError(err)

	if lowerBound.Lt(upperBound) { // positive values, lower < value < upper
		require.True(value.Gte(lowerBound), "value: %s, lowerBound: %s", value.String(), lowerBound.String())
		require.True(value.Lte(upperBound), "value: %s, upperBound: %s", value.String(), upperBound.String())
	} else { // negative values, upper < value < lower
		require.True(value.Lte(lowerBound), "value: %s, lowerBound: %s", value.String(), lowerBound.String())
		require.True(value.Gte(upperBound), "value: %s, upperBound: %s", value.String(), upperBound.String())
	}
}

func (s *AlloraTestUtilSuite) GetEpsilon(require *require.Assertions) func(epsilon int, value alloraMath.Dec, target string) {
	return func(epsilon int, value alloraMath.Dec, target string) {
		switch epsilon {
		case 1:
			s.inEpsilon(value, target, "0.1", require)
		case 2:
			s.inEpsilon(value, target, "0.01", require)
		case 3:
			s.inEpsilon(value, target, "0.001", require)
		case 4:
			s.inEpsilon(value, target, "0.0001", require)
		case 5:
			s.inEpsilon(value, target, "0.00001", require)
		case 6:
			s.inEpsilon(value, target, "0.000001", require)
		case 7:
			s.inEpsilon(value, target, "0.0000001", require)
		case 8:
			s.inEpsilon(value, target, "0.00000001", require)
		case 9:
			s.inEpsilon(value, target, "0.000000001", require)
		default:
			require.Fail("Invalid epsilon value")

		}
	}
}
