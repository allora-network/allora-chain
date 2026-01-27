package math_test

import (
	"testing"

	"github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/require"
)

func TestPow(t *testing.T) {
	tests := []struct {
		name      string
		base      math.Dec
		exponent  math.Dec
		expected  math.Dec
		shouldErr bool
	}{
		{
			name:      "base NaN",
			base:      math.NewNaN(),
			exponent:  math.OneDec(),
			expected:  math.Dec{},
			shouldErr: true,
		},
		{
			name:      "exponent NaN",
			base:      math.OneDec(),
			exponent:  math.NewNaN(),
			expected:  math.Dec{},
			shouldErr: true,
		},
		{
			name:      "negative base",
			base:      math.MustNewDecFromString("-1"),
			exponent:  math.NewNaN(),
			expected:  math.Dec{},
			shouldErr: true,
		},
		{
			name:      "zero base",
			base:      math.ZeroDec(),
			exponent:  math.MustNewDecFromString("23"),
			expected:  math.ZeroDec(),
			shouldErr: false,
		},
		{
			name:      "zero base with zero exponent",
			base:      math.ZeroDec(),
			exponent:  math.ZeroDec(),
			expected:  math.ZeroDec(),
			shouldErr: false,
		},
		{
			name:      "zero exponent",
			base:      math.NewDecFromInt64(234),
			exponent:  math.ZeroDec(),
			expected:  math.OneDec(),
			shouldErr: false,
		},
		{
			name:      "one exponent",
			base:      math.NewDecFromInt64(234),
			exponent:  math.OneDec(),
			expected:  math.NewDecFromInt64(234),
			shouldErr: false,
		},
		{
			name:      "arbitrary values",
			base:      math.NewDecFromInt64(23),
			exponent:  math.NewDecFromInt64(12),
			expected:  math.NewDecFromInt64(21914624432020321),
			shouldErr: false,
		},
		{
			name:      "arbitrary decimal values",
			base:      math.MustNewDecFromString("5.5632892391219024912482190471290471"),
			exponent:  math.MustNewDecFromString("26.6788628270955670027740113863276"),
			expected:  math.MustNewDecFromString("76665607007690068287.65692825015465"),
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := math.Pow(tt.base, tt.exponent)
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, result.Equal(tt.expected))
			}
		})
	}
}
