package math

import (
	"testing"

	"github.com/cockroachdb/apd/v3"
	"github.com/stretchr/testify/require"
)

//nolint:exhaustruct // tests construct partial apd/Dec values on purpose
func TestClampMagnitude(t *testing.T) {
	dec := MustNewDecFromString
	minM := dec("1e-40")
	maxM := dec("1e40")

	// finite cases against the standard [1e-40, 1e40] window
	cases := []struct {
		name string
		in   Dec
		want Dec
	}{
		{"zero", ZeroDec(), ZeroDec()},
		{"negative zero", dec("-0"), ZeroDec()},
		{"positive interior", dec("0.5"), dec("0.5")},
		{"negative interior", dec("-0.5"), dec("-0.5")},
		{"equal min", minM, minM},
		{"equal -min", dec("-1e-40"), dec("-1e-40")},
		{"equal max", maxM, maxM},
		{"equal -max", dec("-1e40"), dec("-1e40")},
		{"just below min +", dec("1e-41"), minM},
		{"just below min -", dec("-1e-41"), dec("-1e-40")},
		{"just above max +", dec("1e41"), maxM},
		{"just above max -", dec("-1e41"), dec("-1e40")},
		{"extreme tiny +", dec("1.5e-37962"), minM},
		{"extreme tiny -", dec("-1.5e-37962"), dec("-1e-40")},
		{"extreme huge +", dec("1e37962"), maxM},
		{"extreme huge -", dec("-1e37962"), dec("-1e40")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClampMagnitude(tc.in, minM, maxM)
			require.Truef(t, got.Equal(tc.want), "got %s want %s", got.String(), tc.want.String())
		})
	}

	t.Run("NaN passes through", func(t *testing.T) {
		require.True(t, ClampMagnitude(NewNaN(), minM, maxM).IsNaN())
	})

	t.Run("infinity passes through", func(t *testing.T) {
		posInf := Dec{dec: apd.Decimal{Form: apd.Infinite}}
		negInf := Dec{dec: apd.Decimal{Form: apd.Infinite, Negative: true}}
		require.False(t, ClampMagnitude(posInf, minM, maxM).IsFinite())
		require.False(t, ClampMagnitude(negInf, minM, maxM).IsFinite())
	})

	t.Run("degenerate window min==max", func(t *testing.T) {
		one := dec("1")
		require.True(t, ClampMagnitude(dec("0.5"), one, one).Equal(one)) // raised
		require.True(t, ClampMagnitude(dec("2"), one, one).Equal(one))   // lowered
		require.True(t, ClampMagnitude(one, one, one).Equal(one))        // unchanged
	})

	t.Run("zero lower bound disables lower clamp", func(t *testing.T) {
		got := ClampMagnitude(dec("1e-100"), ZeroDec(), maxM)
		require.True(t, got.Equal(dec("1e-100")))
	})

	t.Run("input is not mutated", func(t *testing.T) {
		in := dec("1.5e-37962")
		snap := dec("1.5e-37962")
		_ = ClampMagnitude(in, minM, maxM)
		require.True(t, in.Equal(snap))
	})

	t.Run("clamped output is short", func(t *testing.T) {
		require.LessOrEqual(t, len(ClampMagnitude(dec("1.5e-37962"), minM, maxM).String()), 45)
	})
}
