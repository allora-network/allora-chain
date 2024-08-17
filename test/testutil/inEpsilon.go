package testutil

import (
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	require "github.com/stretchr/testify/require"
)

func InEpsilon(t *testing.T, value alloraMath.Dec, target alloraMath.Dec, epsilon alloraMath.Dec) {
	t.Helper()
	one := alloraMath.MustNewDecFromString("1")

	lowerMultiplier, err := one.Sub(epsilon)
	require.NoError(t, err)
	lowerBound, err := target.Mul(lowerMultiplier)
	require.NoError(t, err)

	upperMultiplier, err := one.Add(epsilon)
	require.NoError(t, err)
	upperBound, err := target.Mul(upperMultiplier)
	require.NoError(t, err)

	if lowerBound.Lt(upperBound) { // positive values, lower < value < upper
		require.True(
			t, value.Gte(lowerBound),
			"value: %s, target: %s, lowerBound: %s",
			value.String(), target.String(), lowerBound.String(),
		)
		require.True(
			t, value.Lte(upperBound),
			"value: %s, target %s, upperBound: %s",
			value.String(), target.String(), upperBound.String(),
		)
	} else { // negative values, upper < value < lower
		require.True(
			t, value.Lte(lowerBound),
			"value: %s, target %s, lowerBound: %s",
			value.String(), target.String(), lowerBound.String(),
		)
		require.True(
			t, value.Gte(upperBound),
			"value: %s, target %s, upperBound: %s",
			value.String(), target.String(), upperBound.String(),
		)
	}
}

func InEpsilon2(t *testing.T, value alloraMath.Dec, target string) {
	t.Helper()
	epsilonDec := alloraMath.MustNewDecFromString("0.01")
	targetDec := alloraMath.MustNewDecFromString(target)
	InEpsilon(t, value, targetDec, epsilonDec)
}

func InEpsilon3(t *testing.T, value alloraMath.Dec, target string) {
	t.Helper()
	epsilonDec := alloraMath.MustNewDecFromString("0.001")
	targetDec := alloraMath.MustNewDecFromString(target)
	InEpsilon(t, value, targetDec, epsilonDec)
}

/*unused
func InEpsilon4(t *testing.T, value alloraMath.Dec, target string) {
       s.inEpsilon(t, value, target, "0.0001")
}*/

func InEpsilon5(t *testing.T, value alloraMath.Dec, target string) {
	t.Helper()
	epsilonDec := alloraMath.MustNewDecFromString("0.00001")
	targetDec := alloraMath.MustNewDecFromString(target)
	InEpsilon(t, value, targetDec, epsilonDec)
}

func InEpsilon5Dec(t *testing.T, value alloraMath.Dec, target alloraMath.Dec) {
	t.Helper()
	InEpsilon(t, value, target, alloraMath.MustNewDecFromString("0.00001"))
}
