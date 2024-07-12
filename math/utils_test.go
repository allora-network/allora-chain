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

func TestStdDevOneValueShouldBeZero(t *testing.T) {
	stdDev, err := alloraMath.StdDev([]alloraMath.Dec{alloraMath.MustNewDecFromString("-0.00675")})
	require.NoError(t, err)
	require.True(
		t,
		alloraMath.InDelta(
			alloraMath.MustNewDecFromString("0"),
			stdDev,
			alloraMath.MustNewDecFromString("0.0001"),
		),
	)
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

func TestMedian(t *testing.T) {
	tests := []struct {
		name     string
		data     []alloraMath.Dec
		expected alloraMath.Dec
	}{
		{
			name: "odd number of elements",
			data: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("1"),
				alloraMath.MustNewDecFromString("3"),
				alloraMath.MustNewDecFromString("5"),
				alloraMath.MustNewDecFromString("7"),
				alloraMath.MustNewDecFromString("9"),
			},
			expected: alloraMath.MustNewDecFromString("5"),
		},
		{
			name: "even number of elements",
			data: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("1"),
				alloraMath.MustNewDecFromString("3"),
				alloraMath.MustNewDecFromString("5"),
				alloraMath.MustNewDecFromString("7"),
			},
			expected: alloraMath.MustNewDecFromString("4"),
		},
		{
			name: "complex large values",
			data: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("123456789.123456789"),
				alloraMath.MustNewDecFromString("987654321.987654321"),
				alloraMath.MustNewDecFromString("555555555.555555555"),
				alloraMath.MustNewDecFromString("333333333.333333333"),
				alloraMath.MustNewDecFromString("111111111.111111111"),
			},
			expected: alloraMath.MustNewDecFromString("333333333.333333333"),
		},
		{
			name: "single element",
			data: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("7"),
			},
			expected: alloraMath.MustNewDecFromString("7"),
		},
		{
			name:     "empty slice",
			data:     []alloraMath.Dec{},
			expected: alloraMath.ZeroDec(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := alloraMath.Median(tt.data)
			require.NoError(t, err)
			require.True(t, tt.expected.Equal(result))
		})
	}
}

func TestWeightedInferences(t *testing.T) {
	data := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("1"),
		alloraMath.MustNewDecFromString("2"),
		alloraMath.MustNewDecFromString("3"),
		alloraMath.MustNewDecFromString("4"),
		alloraMath.MustNewDecFromString("5"),
	}
	weights := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("1"),
		alloraMath.MustNewDecFromString("1"),
		alloraMath.MustNewDecFromString("1"),
		alloraMath.MustNewDecFromString("1"),
		alloraMath.MustNewDecFromString("1"),
	}
	percentiles := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0"),
		alloraMath.MustNewDecFromString("50"),
		alloraMath.MustNewDecFromString("100"),
	}
	expected := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("1"),
		alloraMath.MustNewDecFromString("3"),
		alloraMath.MustNewDecFromString("5"),
	}

	result, err := alloraMath.WeightedPercentile(data, weights, percentiles)
	require.NoError(t, err)
	require.Len(t, result, len(expected))
	for i, r := range result {
		require.True(t, alloraMath.InDelta(expected[i], r, alloraMath.MustNewDecFromString("0.000001")))
	}
}

func TestWeightedInferences2(t *testing.T) {
	data := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("10"),
		alloraMath.MustNewDecFromString("20"),
		alloraMath.MustNewDecFromString("30"),
		alloraMath.MustNewDecFromString("40"),
		alloraMath.MustNewDecFromString("50"),
	}
	weights := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.1"),
		alloraMath.MustNewDecFromString("0.2"),
		alloraMath.MustNewDecFromString("0.3"),
		alloraMath.MustNewDecFromString("0.4"),
		alloraMath.MustNewDecFromString("0.5"),
	}
	percentiles := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("10"),
		alloraMath.MustNewDecFromString("25"),
		alloraMath.MustNewDecFromString("50"),
		alloraMath.MustNewDecFromString("75"),
		alloraMath.MustNewDecFromString("90"),
	}
	expected := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("16.666666666666664"),
		alloraMath.MustNewDecFromString("27"),
		alloraMath.MustNewDecFromString("38.57142857142857"),
		alloraMath.MustNewDecFromString("47.22222222222222"),
		alloraMath.MustNewDecFromString("50"),
	}

	result, err := alloraMath.WeightedPercentile(data, weights, percentiles)
	require.NoError(t, err)
	require.Len(t, result, len(expected))
	for i, r := range result {
		require.True(t, alloraMath.InDelta(expected[i], r, alloraMath.MustNewDecFromString("0.000001")))
	}
}
