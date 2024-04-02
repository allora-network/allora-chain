package module_test

import (
	"math"
	"testing"

	"github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

func TestGradient(t *testing.T) {
	// Define test cases
	tests := []struct {
		name        string
		p           float64
		x           float64
		expected    float64
		expectedErr error
	}{
		{
			name:        "normal operation",
			p:           2,
			x:           1,
			expected:    1.92014,
			expectedErr: nil,
		},
		{
			name:        "normal operation",
			p:           10,
			x:           3,
			expected:    216664,
			expectedErr: nil,
		},
		{
			name:        "p is NaN",
			p:           math.NaN(),
			x:           1,
			expected:    0,
			expectedErr: types.ErrPhiInvalidInput,
		},
		{
			name:        "x is NaN",
			p:           2,
			x:           math.NaN(),
			expected:    0,
			expectedErr: types.ErrPhiInvalidInput,
		},
		{
			name:        "p is Inf",
			p:           math.Inf(1),
			x:           1,
			expected:    0,
			expectedErr: types.ErrPhiInvalidInput,
		},
		// Add more test cases as needed, especially to handle edge cases
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function under test
			result, err := network_summaries.Gradient(tc.p, tc.x)

			// Validate the results
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
				require.InEpsilon(t, tc.expected, result, 1e-6, "result should match expected value within epsilon")
			}
		})
	}
}
