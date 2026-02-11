package math_test

import (
	"testing"

	"github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/require"
)

func TestLn(t *testing.T) {
	tests := []struct {
		name      string
		input     math.Dec
		expected  math.Dec
		shouldErr bool
	}{
		{
			name:      "NaN",
			input:     math.NewNaN(),
			expected:  math.Dec{},
			shouldErr: true,
		},
		{
			name:      "zero",
			input:     math.ZeroDec(),
			expected:  math.Dec{},
			shouldErr: true,
		},
		{
			name:      "negative number",
			input:     math.MustNewDecFromString("-1"),
			expected:  math.Dec{},
			shouldErr: true,
		},
		{
			name:      "one",
			input:     math.OneDec(),
			expected:  math.ZeroDec(),
			shouldErr: false,
		},
		{
			// See https://github.com/allora-network/allora-chain/pull/789
			name:      "special value for determinism check",
			input:     math.MustNewDecFromString("1.6285091944505809264504560045920167"),
			expected:  math.MustNewDecFromString("0.4876649916811116824516548471782887"),
			shouldErr: false,
		},
		{
			name:      "big number",
			input:     math.NewDecFromInt64(2048 * 2048 * 2048 * 2048 * 2048),
			expected:  math.MustNewDecFromString("38.12309493079699201794776668019971"),
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := math.Ln(tt.input)
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, result.Equal(tt.expected))
			}
		})
	}
}

func TestLog10(t *testing.T) {
	tests := []struct {
		name      string
		input     math.Dec
		expected  math.Dec
		shouldErr bool
	}{
		{
			name:      "NaN",
			input:     math.NewNaN(),
			expected:  math.Dec{},
			shouldErr: true,
		},
		{
			name:      "zero",
			input:     math.ZeroDec(),
			expected:  math.Dec{},
			shouldErr: true,
		},
		{
			name:      "negative number",
			input:     math.MustNewDecFromString("-1"),
			expected:  math.Dec{},
			shouldErr: true,
		},
		{
			name:      "one",
			input:     math.OneDec(),
			expected:  math.ZeroDec(),
			shouldErr: false,
		},
		{
			name:      "arbitrary value",
			input:     math.MustNewDecFromString("1.6285091944505809264504560045920167"),
			expected:  math.MustNewDecFromString("0.2117902149045020106508149568387278"),
			shouldErr: false,
		},
		{
			name:      "big number",
			input:     math.NewDecFromInt64(2048 * 2048 * 2048 * 2048 * 2048),
			expected:  math.MustNewDecFromString("16.55664976151896573675563920984712"),
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := math.Log10(tt.input)
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, result.Equal(tt.expected))
			}
		})
	}
}
