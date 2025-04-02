package math

import (
	goMath "math"
	"testing"

	"github.com/cockroachdb/apd/v3"
	"github.com/stretchr/testify/require"
)

func TestEnforceDecimalPrecision(t *testing.T) {
	testCases := []struct {
		name      string
		precision uint32
		before    *apd.Decimal
		after     *apd.Decimal
	}{
		{
			name:      "no precision",
			precision: 0,
			before:    apd.New(1, 0), // 1
			after:     apd.New(1, 0),
		},
		{
			name:      "no decimals",
			precision: 5,
			before:    apd.New(1, 0), // 1
			after:     apd.New(100000, -5),
		},
		{
			name:      "no decimals but not 1",
			precision: 5,
			before:    apd.New(12345, 0), // 12345
			after:     apd.New(1234500000, -5),
		},
		{
			name:      "with decimals",
			precision: 5,
			before:    apd.New(1, -3), // 0.001
			after:     apd.New(100, -5),
		},
		{
			name:      "already at precision",
			precision: 5,
			before:    apd.New(1, -5), // 0.00001
			after:     apd.New(1, -5),
		},
		{
			name:      "overprecise, don't reduce precision",
			precision: 5,
			before:    apd.New(1, -10), // 0.0000000001
			after:     apd.New(1, -10),
		},
		{
			name:      "would overflow, don't touch",
			precision: goMath.MaxUint32,
			before:    apd.New(1, -5),
			after:     apd.New(1, -5),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.before
			enforceDecimalPrecision(r, tc.precision)
			require.Equal(t, tc.after.Coeff, r.Coeff)
			require.Equal(t, tc.after.Exponent, r.Exponent)
			require.Equal(t, tc.after.Form, r.Form)
			require.Equal(t, tc.after.Negative, r.Negative)
		})
	}
}
