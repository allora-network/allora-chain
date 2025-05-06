package math_test

import (
	"testing"

	"github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/require"
)

func TestExp(t *testing.T) {
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
			expected:  math.OneDec(),
			shouldErr: false,
		},
		{
			name:      "arbitrary negative number",
			input:     math.MustNewDecFromString("-5.0000000000000024912482190471290471"),
			expected:  math.MustNewDecFromString("0.006737946999085450310737586917551778"),
			shouldErr: false,
		},
		{
			name:      "arbitrary positive number",
			input:     math.MustNewDecFromString("5.5632892391219024912482190471290471"),
			expected:  math.MustNewDecFromString("260.6788628270955670027740113863276"),
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := math.Exp(tt.input)
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, result.Equal(tt.expected))
			}
		})
	}
}
