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

func TestCalcExpDecaySimple(t *testing.T) {
	decayFactor := alloraMath.MustNewDecFromString("0.1")
	currentRev := alloraMath.MustNewDecFromString("300")

	// (1 - 0.1) * 300
	// 0.9 * 300 = 270
	expected := alloraMath.MustNewDecFromString("270")

	result, err := alloraMath.CalcExpDecay(currentRev, decayFactor)
	require.NoError(t, err)
	require.True(t, alloraMath.InDelta(expected, result, alloraMath.MustNewDecFromString("0.0001")))
}

func TestCalcExpDecayZeroDecayFactor(t *testing.T) {
	decayFactor := alloraMath.MustNewDecFromString("0")
	currentRev := alloraMath.MustNewDecFromString("300")

	// (1 - 0) * 300
	// 1 * 300 = 300
	expected := alloraMath.MustNewDecFromString("300")

	result, err := alloraMath.CalcExpDecay(currentRev, decayFactor)
	require.NoError(t, err)
	require.True(t, alloraMath.InDelta(expected, result, alloraMath.MustNewDecFromString("0.0001")))
}

func TestStdDev(t *testing.T) {
	tests := []struct {
		name string
		data []alloraMath.Dec
		want alloraMath.Dec
	}{
		{
			name: "basic",
			data: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("-0.00675"),
				alloraMath.MustNewDecFromString("-0.00622"),
				alloraMath.MustNewDecFromString("-0.01502"),
				alloraMath.MustNewDecFromString("-0.01214"),
				alloraMath.MustNewDecFromString("0.00392"),
				alloraMath.MustNewDecFromString("0.00559"),
				alloraMath.MustNewDecFromString("0.0438"),
				alloraMath.MustNewDecFromString("0.04304"),
				alloraMath.MustNewDecFromString("0.09719"),
				alloraMath.MustNewDecFromString("0.09675"),
			},
			want: alloraMath.MustNewDecFromString("0.041014924273483966"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := alloraMath.StdDev(tt.data)
			require.NoError(t, err)
			require.True(t, alloraMath.InDelta(tt.want, got, alloraMath.MustNewDecFromString("0.0001")))
		})
	}
}

func TestPhiSimple(t *testing.T) {
	x := alloraMath.MustNewDecFromString("7.9997")
	p := alloraMath.NewDecFromInt64(3)
	c := alloraMath.MustNewDecFromString("0.75")
	// we expect a value very very close to 64
	result, err := alloraMath.Phi(p, c, x)
	require.NoError(t, err)
	require.False(t, alloraMath.InDelta(alloraMath.NewDecFromInt64(64), result, alloraMath.MustNewDecFromString("0.001")))
}

// φ'_p(x) = p / (exp(p * (c - x)) + 1)
func TestGradient(t *testing.T) {
	tests := []struct {
		name        string
		c           alloraMath.Dec
		p           alloraMath.Dec
		x           alloraMath.Dec
		expected    alloraMath.Dec
		expectedErr error
	}{
		{
			name: "normal operation 1",
			c:    alloraMath.MustNewDecFromString("0.75"),
			p:    alloraMath.MustNewDecFromString("2"),
			x:    alloraMath.MustNewDecFromString("1"),
			// φ'_p(x) = 2 / (exp(2 * (0.75 - 1)) + 1)
			// φ'_p(x) = 1.2449186624037092
			expected:    alloraMath.MustNewDecFromString("1.2449186624037092"),
			expectedErr: nil,
		},
		{
			name: "normal operation 2",
			c:    alloraMath.MustNewDecFromString("0.75"),
			p:    alloraMath.MustNewDecFromString("10"),
			x:    alloraMath.MustNewDecFromString("3"),
			// φ'_p(x) = 10 / (exp(10 * (0.75 - 3)) + 1)
			// φ'_p(x) = 9.999999998308102
			expected:    alloraMath.MustNewDecFromString("9.999999998308102"),
			expectedErr: nil,
		},
		{
			name: "normal operation 3",
			c:    alloraMath.MustNewDecFromString("0.75"),
			p:    alloraMath.MustNewDecFromString("9.2"),
			x:    alloraMath.MustNewDecFromString("3.4"),
			// φ'_p(x) = 9.2 / (exp(9.2 * (0.75 - 3.4)) + 1)
			expected:    alloraMath.MustNewDecFromString("9.199999999762486"),
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := alloraMath.Gradient(tc.p, tc.c, tc.x)

			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
				require.True(
					t,
					alloraMath.InDelta(
						tc.expected,
						result,
						alloraMath.MustNewDecFromString("0.00001")),
					"result should match expected value within epsilon",
					tc.expected.String(),
					result.String(),
				)
			}
		})
	}
}
