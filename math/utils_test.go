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
	inDelta, err := alloraMath.InDelta(expected, result, alloraMath.MustNewDecFromString("0.0001"))
	require.NoError(t, err)
	require.True(t, inDelta)
}

func TestCalcEmaWithNoPrior(t *testing.T) {
	alpha := alloraMath.MustNewDecFromString("0.1")
	current := alloraMath.MustNewDecFromString("300")
	previous := alloraMath.MustNewDecFromString("200")

	// Current value should be returned if there is no prior value
	expected := alloraMath.MustNewDecFromString("300")

	result, err := alloraMath.CalcEma(alpha, current, previous, true)
	require.NoError(t, err)
	inDelta, err := alloraMath.InDelta(expected, result, alloraMath.MustNewDecFromString("0.0001"))
	require.NoError(t, err)
	require.True(t, inDelta)
}

func TestCalcEmaWithNaN(t *testing.T) {
	alpha := alloraMath.MustNewDecFromString("0.1")
	current := alloraMath.MustNewDecFromString("300")
	previous := alloraMath.NewNaN()

	_, err := alloraMath.CalcEma(alpha, current, previous, false)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)

	previous = alloraMath.MustNewDecFromString("200")
	current = alloraMath.NewNaN()
	_, err = alloraMath.CalcEma(alpha, current, previous, false)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)

	current = alloraMath.MustNewDecFromString("300")
	alpha = alloraMath.NewNaN()
	_, err = alloraMath.CalcEma(alpha, current, previous, false)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
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
			want: alloraMath.MustNewDecFromString("0.04323473788517746987174741957435394"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := alloraMath.StdDev(tt.data)
			require.NoError(t, err)
			inDelta, err := alloraMath.InDelta(tt.want, got, alloraMath.MustNewDecFromString("0.0001"))
			require.NoError(t, err)
			require.True(t, inDelta)
		})
	}
}

func TestStdDevOneValueShouldBeZero(t *testing.T) {
	stdDev, err := alloraMath.StdDev([]alloraMath.Dec{alloraMath.MustNewDecFromString("-0.00675")})

	require.NoError(t, err)
	inDelta, err := alloraMath.InDelta(
		alloraMath.MustNewDecFromString("0"),
		stdDev,
		alloraMath.MustNewDecFromString("0.0001"),
	)
	require.NoError(t, err)
	require.True(t, inDelta)
}

func TestStdDevWithNaN(t *testing.T) {
	_, err := alloraMath.StdDev([]alloraMath.Dec{alloraMath.OneDec(), alloraMath.NewNaN(), alloraMath.ZeroDec()})
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestPhiSimple(t *testing.T) {
	x := alloraMath.MustNewDecFromString("7.9997")
	p := alloraMath.NewDecFromInt64(3)
	c := alloraMath.MustNewDecFromString("0.75")
	// we expect a value very very close to 64
	result, err := alloraMath.Phi(p, c, x)
	require.NoError(t, err)
	inDelta, err := alloraMath.InDelta(alloraMath.NewDecFromInt64(64), result, alloraMath.MustNewDecFromString("0.001"))
	require.NoError(t, err)
	require.False(t, inDelta)
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
				inDelta, err := alloraMath.InDelta(
					tc.expected,
					result,
					alloraMath.MustNewDecFromString("0.00001"))
				require.NoError(t, err)
				require.True(
					t,
					inDelta,
					"result should match expected value within epsilon",
					tc.expected.String(),
					result.String(),
				)
			}
		})
	}
}

func TestGradientWithNaN(t *testing.T) {
	_, err := alloraMath.Gradient(alloraMath.MustNewDecFromString("2"), alloraMath.NewNaN(), alloraMath.MustNewDecFromString("1"))
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
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

func TestMedianWithNaN(t *testing.T) {
	_, err := alloraMath.Median([]alloraMath.Dec{alloraMath.OneDec(), alloraMath.NewNaN(), alloraMath.ZeroDec()})
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
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
		inDelta, err := alloraMath.InDelta(expected[i], r, alloraMath.MustNewDecFromString("0.000001"))
		require.NoError(t, err)
		require.True(t, inDelta)
	}
}

func TestPhiWithNaN(t *testing.T) {
	_, err := alloraMath.Phi(alloraMath.MustNewDecFromString("2"), alloraMath.NewNaN(), alloraMath.MustNewDecFromString("1"))
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestCumulativeSumWithNaN(t *testing.T) {
	_, err := alloraMath.CumulativeSum([]alloraMath.Dec{alloraMath.OneDec(), alloraMath.NewNaN(), alloraMath.ZeroDec()})
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestLinearInterpolationWithNaN(t *testing.T) {
	x := []alloraMath.Dec{alloraMath.MustNewDecFromString("2"), alloraMath.NewNaN(), alloraMath.MustNewDecFromString("1")}
	xp := []alloraMath.Dec{alloraMath.MustNewDecFromString("1"), alloraMath.MustNewDecFromString("2"), alloraMath.MustNewDecFromString("3")}
	fp := []alloraMath.Dec{alloraMath.MustNewDecFromString("1"), alloraMath.MustNewDecFromString("2"), alloraMath.MustNewDecFromString("3")}
	_, err := alloraMath.LinearInterpolation(x, xp, fp)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)

	x = []alloraMath.Dec{alloraMath.MustNewDecFromString("2"), alloraMath.MustNewDecFromString("1")}
	xp = []alloraMath.Dec{alloraMath.MustNewDecFromString("1"), alloraMath.NewNaN(), alloraMath.MustNewDecFromString("3")}
	fp = []alloraMath.Dec{alloraMath.MustNewDecFromString("1"), alloraMath.MustNewDecFromString("2"), alloraMath.MustNewDecFromString("3")}
	_, err = alloraMath.LinearInterpolation(x, xp, fp)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)

	x = []alloraMath.Dec{alloraMath.MustNewDecFromString("2"), alloraMath.MustNewDecFromString("1")}
	xp = []alloraMath.Dec{alloraMath.MustNewDecFromString("1"), alloraMath.MustNewDecFromString("2"), alloraMath.MustNewDecFromString("3")}
	fp = []alloraMath.Dec{alloraMath.MustNewDecFromString("1"), alloraMath.MustNewDecFromString("2"), alloraMath.NewNaN()}
	_, err = alloraMath.LinearInterpolation(x, xp, fp)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
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
		inDelta, err := alloraMath.InDelta(expected[i], r, alloraMath.MustNewDecFromString("0.000001"))
		require.NoError(t, err)
		require.True(t, inDelta)
	}
}

func TestWeightedPercentileWithNaN(t *testing.T) {
	data := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("10"),
		alloraMath.MustNewDecFromString("20"),
		alloraMath.NewNaN(),
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
	_, err := alloraMath.WeightedPercentile(data, weights, percentiles)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)

	data = []alloraMath.Dec{
		alloraMath.MustNewDecFromString("10"),
		alloraMath.MustNewDecFromString("20"),
		alloraMath.MustNewDecFromString("30"),
		alloraMath.MustNewDecFromString("40"),
		alloraMath.MustNewDecFromString("50"),
	}
	weights = []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.1"),
		alloraMath.MustNewDecFromString("0.2"),
		alloraMath.NewNaN(),
		alloraMath.MustNewDecFromString("0.4"),
		alloraMath.MustNewDecFromString("0.5"),
	}
	_, err = alloraMath.WeightedPercentile(data, weights, percentiles)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)

	weights = []alloraMath.Dec{
		alloraMath.MustNewDecFromString("0.1"),
		alloraMath.MustNewDecFromString("0.2"),
		alloraMath.MustNewDecFromString("0.3"),
		alloraMath.MustNewDecFromString("0.4"),
		alloraMath.MustNewDecFromString("0.5"),
	}
	percentiles = []alloraMath.Dec{
		alloraMath.MustNewDecFromString("10"),
		alloraMath.MustNewDecFromString("25"),
		alloraMath.MustNewDecFromString("50"),
		alloraMath.NewNaN(),
		alloraMath.MustNewDecFromString("90"),
	}
	_, err = alloraMath.WeightedPercentile(data, weights, percentiles)
	require.Error(t, err)
	require.ErrorIs(t, err, alloraMath.ErrNaN)
}

func TestGetSortedKeys(t *testing.T) {
	m := map[int32]struct{}{
		5: {},
		3: {},
		2: {},
		4: {},
		1: {},
	}

	expected := []int32{1, 2, 3, 4, 5}
	keys := alloraMath.GetSortedKeys[int32](m)
	require.Equal(t, expected, keys)

	m2 := map[string]struct{}{
		"5.5":  {},
		"3.25": {},
		"2.1":  {},
		"4.75": {},
		"1":    {},
	}
	expected2 := []string{
		"1",
		"2.1",
		"3.25",
		"4.75",
		"5.5",
	}
	keys2 := alloraMath.GetSortedKeys(m2)
	require.Equal(t, expected2, keys2)
}

func TestGetSortedKeysWithEmptyMap(t *testing.T) {
	m := map[int32]struct{}{}
	keys := alloraMath.GetSortedKeys(m)
	require.Equal(t, []int32{}, keys)
}

func TestGetSortedElementsByDecWeightDesc(t *testing.T) {
	dec1 := alloraMath.MustNewDecFromString("0.5")
	dec2 := alloraMath.MustNewDecFromString("0.2")
	dec3 := alloraMath.MustNewDecFromString("0.1")
	dec4 := alloraMath.MustNewDecFromString("0.7")
	dec5 := alloraMath.MustNewDecFromString("0.4")
	m := map[int32]*alloraMath.Dec{
		1: &dec1,
		2: &dec2,
		3: &dec3,
		4: &dec4,
		5: &dec5,
	}

	expected := []int32{4, 1, 5, 2, 3}
	keys := alloraMath.GetSortedElementsByDecWeightDesc(m)
	require.Equal(t, expected, keys)
}

func TestGetQuantileOfDecs(t *testing.T) {
	m := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("1"),
		alloraMath.MustNewDecFromString("2"),
		alloraMath.MustNewDecFromString("3"),
		alloraMath.MustNewDecFromString("4"),
		alloraMath.MustNewDecFromString("5"),
		alloraMath.MustNewDecFromString("6"),
		alloraMath.MustNewDecFromString("7"),
		alloraMath.MustNewDecFromString("8"),
		alloraMath.MustNewDecFromString("9"),
		alloraMath.MustNewDecFromString("10"),
	}
	negativeData := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("-5"),
		alloraMath.MustNewDecFromString("-3"),
		alloraMath.MustNewDecFromString("-1"),
		alloraMath.MustNewDecFromString("-4"),
		alloraMath.MustNewDecFromString("-2"),
	}
	mixedData := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("-5"),
		alloraMath.MustNewDecFromString("-3"),
		alloraMath.MustNewDecFromString("0"),
		alloraMath.MustNewDecFromString("-4"),
		alloraMath.MustNewDecFromString("2"),
	}

	tests := []struct {
		name        string
		data        []alloraMath.Dec
		quantile    alloraMath.Dec
		expected    alloraMath.Dec
		expectedErr error
	}{
		{
			data:        m,
			quantile:    alloraMath.MustNewDecFromString("0.25"),
			expected:    alloraMath.MustNewDecFromString("3.25"),
			expectedErr: nil,
			name:        "0.25 quantile",
		},
		{
			data:        m,
			quantile:    alloraMath.MustNewDecFromString("0.5"),
			expected:    alloraMath.MustNewDecFromString("5.5"),
			expectedErr: nil,
			name:        "0.5 quantile",
		},
		{
			data:        m,
			quantile:    alloraMath.MustNewDecFromString("0.75"),
			expected:    alloraMath.MustNewDecFromString("7.75"),
			expectedErr: nil,
			name:        "0.75 quantile",
		},
		{
			data:        []alloraMath.Dec{},
			quantile:    alloraMath.MustNewDecFromString("0.5"),
			expected:    alloraMath.ZeroDec(),
			expectedErr: nil,
			name:        "empty data",
		},
		{
			data: []alloraMath.Dec{
				alloraMath.MustNewDecFromString("42"),
			},
			quantile:    alloraMath.MustNewDecFromString("0.5"),
			expected:    alloraMath.MustNewDecFromString("42"),
			expectedErr: nil,
			name:        "single data point",
		},
		{
			data:        negativeData,
			quantile:    alloraMath.MustNewDecFromString("0.5"),
			expected:    alloraMath.MustNewDecFromString("-3"),
			expectedErr: nil,
			name:        "negative data",
		},
		{
			data:        mixedData,
			quantile:    alloraMath.MustNewDecFromString("0.5"),
			expected:    alloraMath.MustNewDecFromString("-3"),
			expectedErr: nil,
			name:        "mixed data",
		},
		{
			data: []alloraMath.Dec{
				alloraMath.NewNaN(),
				alloraMath.NewNaN(),
				alloraMath.MustNewDecFromString("3"),
			},
			quantile:    alloraMath.MustNewDecFromString("0.5"),
			expected:    alloraMath.NewNaN(),
			expectedErr: alloraMath.ErrNaN,
			name:        "data is NaN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := alloraMath.GetQuantileOfDecs(tt.data, tt.quantile)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetQuantileOfDecsWithInvalidQuantile(t *testing.T) {
	data := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("1"),
		alloraMath.MustNewDecFromString("2"),
		alloraMath.MustNewDecFromString("3"),
	}

	invalidQuantiles := []alloraMath.Dec{
		alloraMath.MustNewDecFromString("1.1"),
		alloraMath.MustNewDecFromString("-0.1"),
		alloraMath.NewNaN(),
	}

	for _, quantile := range invalidQuantiles {
		_, err := alloraMath.GetQuantileOfDecs(data, quantile)
		require.Error(t, err)
	}
}
