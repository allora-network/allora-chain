package testutil

import (
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	require "github.com/stretchr/testify/require"
)

func InEpsilon(t *testing.T, value alloraMath.Dec, target string, epsilon string) {
	epsilonDec := alloraMath.MustNewDecFromString(epsilon)
	targetDec := alloraMath.MustNewDecFromString(target)
	one := alloraMath.MustNewDecFromString("1")

	lowerMultiplier, err := one.Sub(epsilonDec)
	require.NoError(t, err)
	lowerBound, err := targetDec.Mul(lowerMultiplier)
	require.NoError(t, err)

	upperMultiplier, err := one.Add(epsilonDec)
	require.NoError(t, err)
	upperBound, err := targetDec.Mul(upperMultiplier)
	require.NoError(t, err)

	if lowerBound.Lt(upperBound) { // positive values, lower < value < upper
		require.True(t, value.Gte(lowerBound), "value: %s, lowerBound: %s", value.String(), lowerBound.String())
		require.True(t, value.Lte(upperBound), "value: %s, upperBound: %s", value.String(), upperBound.String())
	} else { // negative values, upper < value < lower
		require.True(t, value.Lte(lowerBound), "value: %s, lowerBound: %s", value.String(), lowerBound.String())
		require.True(t, value.Gte(upperBound), "value: %s, upperBound: %s", value.String(), upperBound.String())
	}
}

func InEpsilon2(t *testing.T, value alloraMath.Dec, target string) {
	InEpsilon(t, value, target, "0.01")
}

func InEpsilon3(t *testing.T, value alloraMath.Dec, target string) {
	InEpsilon(t, value, target, "0.001")
}

/*unused
func InEpsilon4(t *testing.T, value alloraMath.Dec, target string) {
       s.inEpsilon(t, value, target, "0.0001")
}*/

func InEpsilon5(t *testing.T, value alloraMath.Dec, target string) {
	InEpsilon(t, value, target, "0.00001")
}
