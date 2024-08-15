package math

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// This function cannot run in the math_test package
// because it needs direct access to the apd.Coeff field
func TestDecReduce(t *testing.T) {
	// Test case 1
	x := NewDecFromInt64(12345678900)
	expectedY := NewDecFromInt64(123456789)
	expectedN := 2

	y, n := x.Reduce()

	require.Equal(t, expectedY.dec.Coeff, y.dec.Coeff)
	require.Equal(t, expectedN, n)

	// Test case 2
	x = NewDecFromInt64(0)
	expectedY = NewDecFromInt64(0)
	expectedN = 0

	y, n = x.Reduce()

	require.Equal(t, expectedY.dec.Coeff, y.dec.Coeff)
	require.Equal(t, expectedN, n)

	// Test case 3
	x = NewDecFromInt64(10000.000)
	expectedY = NewDecFromInt64(1)
	expectedN = 4

	y, n = x.Reduce()

	require.Equal(t, expectedY.dec.Coeff, y.dec.Coeff)
	require.Equal(t, expectedN, n)

	// Test case 4
	x = NewDecFromInt64(0000.000)
	expectedY = NewDecFromInt64(0)
	expectedN = 0

	y, n = x.Reduce()

	require.Equal(t, expectedY.dec.Coeff, y.dec.Coeff)
	require.Equal(t, expectedN, n)

	// Test case 5
	x = NewDecFromInt64(-1234560000.000)
	expectedY = NewDecFromInt64(123456)
	expectedN = 4

	y, n = x.Reduce()

	require.Equal(t, expectedY.dec.Coeff, y.dec.Coeff)
	require.True(t, y.dec.Negative)
	require.Equal(t, expectedN, n)

	// Test case 6
	x = NewDecFromInt64(-123456000000.000)
	expectedN = 6
	strX := "-123456000000"

	y, n = x.Reduce()

	require.Equal(t, strX, y.String())
	require.Equal(t, expectedN, n)
}
