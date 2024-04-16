package math_test

import (
	"testing"

	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/require"
)

func TestCalcEmaSimple(t *testing.T) {
	alpha := alloraMath.MustNewDecFromString("0.1")
	current := alloraMath.MustNewDecFromString("300")
	previous := alloraMath.MustNewDecFromString("200")

	// 0.1*300 + (1-0.1)*200
	// 30 + 180 = 210
	expected := alloraMath.MustNewDecFromString("210")

	result, err := alloraMath.CalcEma(alpha, current, previous, false)
	require.NoError(t, err)
	require.True(t, alloraMath.InDelta(expected, result, alloraMath.MustNewDecFromString("0.0001")))
}

func TestCalcEmaWithNoPrior(t *testing.T) {
	alpha := alloraMath.MustNewDecFromString("0.1")
	current := alloraMath.MustNewDecFromString("300")
	previous := alloraMath.MustNewDecFromString("200")

	// Current value should be returned if there is no prior value
	expected := alloraMath.MustNewDecFromString("300")

	result, err := alloraMath.CalcEma(alpha, current, previous, true)
	require.NoError(t, err)
	require.True(t, alloraMath.InDelta(expected, result, alloraMath.MustNewDecFromString("0.0001")))
}
