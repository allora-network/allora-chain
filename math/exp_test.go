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
			name:      "below min boundary",
			input:     math.MustNewDecFromString("-230258.509299404568401799145468436422"),
			expected:  math.ZeroDec(),
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

func TestExp1DivExp1(t *testing.T) {
	testCases := []struct {
		name        string
		a           string
		b           string
		expectedStr string
		shouldError bool
		errorType   error
		tolerance   string // for approximate comparisons
	}{
		// Case 1: a <= 0 && b <= 0 - Direct computation: (e^a + 1) / (e^b + 1)
		{
			name:        "Case 1: Both zero",
			a:           "0",
			b:           "0",
			expectedStr: "1", // (e^0 + 1) / (e^0 + 1) = 2/2 = 1
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 1: Both negative small",
			a:           "-1",
			b:           "-1",
			expectedStr: "1", // (e^-1 + 1) / (e^-1 + 1) = 1
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 1: Both negative, a < b",
			a:           "-2",
			b:           "-1",
			expectedStr: "0.8299965984314521", // (e^-2 + 1) / (e^-1 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 1: Both negative, a > b",
			a:           "-1",
			b:           "-2",
			expectedStr: "1.204824214809825", // (e^-1 + 1) / (e^-2 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 1: Very negative values",
			a:           "-10",
			b:           "-10",
			expectedStr: "1", // Should be exactly 1
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 1: Very negative, different values",
			a:           "-10",
			b:           "-5",
			expectedStr: "0.9933522451505159", // (e^-10 + 1) / (e^-5 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},

		// Case 2: a > 0 && b <= 0 - Formula: e^a * (e^{-a} + 1) / (e^b + 1)
		{
			name:        "Case 2: a positive, b zero",
			a:           "1",
			b:           "0",
			expectedStr: "1.859140914229523", // e^1 * (e^-1 + 1) / (e^0 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 2: a positive, b negative",
			a:           "2",
			b:           "-1",
			expectedStr: "6.132891427731616", // e^2 * (e^-2 + 1) / (e^-1 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 2: Large positive a, negative b",
			a:           "5",
			b:           "-2",
			expectedStr: "131.6026739489939", // e^5 * (e^-5 + 1) / (e^-2 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 2: Small positive a, very negative b",
			a:           "0.1",
			b:           "-10",
			expectedStr: "2.105075347802713", // e^0.1 * (e^-0.1 + 1) / (e^-10 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},

		// Case 3: a <= 0 && b > 0 - Formula: e^{-b} * (e^a + 1) / (e^{-b} + 1)
		{
			name:        "Case 3: a zero, b positive",
			a:           "0",
			b:           "1",
			expectedStr: "0.5378828427399902", // e^-1 * (e^0 + 1) / (e^-1 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 3: a negative, b positive",
			a:           "-1",
			b:           "2",
			expectedStr: "0.1630552263616172", // e^-2 * (e^-1 + 1) / (e^-2 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 3: Very negative a, small positive b",
			a:           "-10",
			b:           "0.1",
			expectedStr: "0.4750423784325841", // e^-0.1 * (e^-10 + 1) / (e^-0.1 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 3: Negative a, large positive b",
			a:           "-2",
			b:           "5",
			expectedStr: "0.007598629799783373", // e^-5 * (e^-2 + 1) / (e^-5 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},

		// Case 4: a > 0 && b > 0 - Formula: e^{a-b} * (e^{-a} + 1) / (e^{-b} + 1)
		{
			name:        "Case 4: Both positive, equal",
			a:           "1",
			b:           "1",
			expectedStr: "1", // Should be exactly 1
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 4: Both positive, a > b",
			a:           "2",
			b:           "1",
			expectedStr: "2.256164671199036", // e^1 * (e^-2 + 1) / (e^-1 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 4: Both positive, a < b",
			a:           "1",
			b:           "2",
			expectedStr: "0.4432300588540602", // e^-1 * (e^-1 + 1) / (e^-2 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 4: Large positive values",
			a:           "5",
			b:           "3",
			expectedStr: "7.086049534658406", // e^2 * (e^-5 + 1) / (e^-3 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Case 4: Small positive values",
			a:           "0.1",
			b:           "0.2",
			expectedStr: "0.9476763771641353", // e^-0.1 * (e^-0.1 + 1) / (e^-0.2 + 1)
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},

		// Edge cases at boundaries
		{
			name:        "Boundary: a=0, b slightly positive",
			a:           "0",
			b:           "0.0001",
			expectedStr: "0.99995", // Very close to 1
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.001",
		},
		{
			name:        "Boundary: a slightly positive, b=0",
			a:           "0.0001",
			b:           "0",
			expectedStr: "1.00005", // Very close to 1
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.001",
		},
		{
			name:        "Boundary: a=0, b slightly negative",
			a:           "0",
			b:           "-0.0001",
			expectedStr: "1.00005", // Very close to 1
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.001",
		},
		{
			name:        "Boundary: a slightly negative, b=0",
			a:           "-0.0001",
			b:           "0",
			expectedStr: "0.99995", // Very close to 1
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.001",
		},

		// Extreme values to test numerical stability
		{
			name:        "Extreme: Very large positive values",
			a:           "10",
			b:           "10",
			expectedStr: "1", // Should be exactly 1
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Extreme: Very large negative values",
			a:           "-20",
			b:           "-20",
			expectedStr: "1", // Should be exactly 1
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.0000000001",
		},
		{
			name:        "Extreme: Large positive a, large negative b",
			a:           "10",
			b:           "-10",
			expectedStr: "22026.465794806718074", // Very large number
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.001",
		},
		{
			name:        "Extreme: Large negative a, large positive b",
			a:           "-10",
			b:           "10",
			expectedStr: "0.0000453999297624848", // Very small number
			shouldError: false,
			errorType:   nil,
			tolerance:   "0.000001",
		},

		// Error cases
		{
			name:        "Error: a is NaN",
			a:           "NaN",
			b:           "1",
			expectedStr: "",
			shouldError: true,
			errorType:   math.ErrNaN,
			tolerance:   "",
		},
		{
			name:        "Error: b is NaN",
			a:           "1",
			b:           "NaN",
			expectedStr: "",
			shouldError: true,
			errorType:   math.ErrNaN,
			tolerance:   "",
		},
		{
			name:        "Error: Both NaN",
			a:           "NaN",
			b:           "NaN",
			expectedStr: "",
			shouldError: true,
			errorType:   math.ErrNaN,
			tolerance:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var a, b math.Dec
			var err error

			// Parse input a
			if tc.a == "NaN" {
				a = math.NewNaN()
			} else {
				a, err = math.NewDecFromString(tc.a)
				require.NoError(t, err, "Failed to parse input a: %s", tc.a)
			}

			// Parse input b
			if tc.b == "NaN" {
				b = math.NewNaN()
			} else {
				b, err = math.NewDecFromString(tc.b)
				require.NoError(t, err, "Failed to parse input b: %s", tc.b)
			}

			// Call the function
			result, err := math.Exp1DivExp1(a, b)

			if tc.shouldError {
				require.Error(t, err, "Expected error but got none")
				if tc.errorType != nil {
					require.ErrorIs(t, err, tc.errorType, "Expected specific error type")
				}
			} else {
				require.NoError(t, err, "Unexpected error: %v", err)

				// Parse expected result
				expected, err := math.NewDecFromString(tc.expectedStr)
				require.NoError(t, err, "Failed to parse expected result: %s", tc.expectedStr)

				// Check if result is within tolerance
				if tc.tolerance != "" {
					tolerance, err := math.NewDecFromString(tc.tolerance)
					require.NoError(t, err, "Failed to parse tolerance: %s", tc.tolerance)

					withinTolerance, err := math.InDelta(expected, result, tolerance)
					require.NoError(t, err, "Error in tolerance check")
					require.True(t, withinTolerance,
						"Result %s not within tolerance %s of expected %s. Difference: %s",
						result.String(), tolerance.String(), expected.String(),
						func() string {
							diff, _ := result.Sub(expected)
							abs, _ := diff.Abs()
							return abs.String()
						}())
				} else {
					// Exact comparison
					require.True(t, expected.Equal(result),
						"Expected %s, got %s", expected.String(), result.String())
				}

				// Additional sanity checks
				require.True(t, result.IsFinite(), "Result should be finite")
				require.False(t, result.IsNaN(), "Result should not be NaN")
				require.True(t, result.IsPositive() || result.IsZero(), "Result should be non-negative")
			}
		})
	}
}

// TestExp1DivExp1SpecialProperties tests mathematical properties of the function
func TestExp1DivExp1SpecialProperties(t *testing.T) {
	t.Run("Identity: f(a,a) = 1", func(t *testing.T) {
		testValues := []string{"0", "1", "-1", "2", "-2", "0.5", "-0.5", "10", "-10"}
		for _, val := range testValues {
			a, err := math.NewDecFromString(val)
			require.NoError(t, err)

			result, err := math.Exp1DivExp1(a, a)
			require.NoError(t, err)

			one := math.OneDec()
			tolerance, _ := math.NewDecFromString("0.0000000001")
			withinTolerance, err := math.InDelta(one, result, tolerance)
			require.NoError(t, err)
			require.True(t, withinTolerance,
				"f(%s,%s) should equal 1, got %s", val, val, result.String())
		}
	})

	t.Run("Monotonicity: If a1 < a2 and b fixed, then f(a1,b) < f(a2,b)", func(t *testing.T) {
		b, _ := math.NewDecFromString("1")
		a1, _ := math.NewDecFromString("0")
		a2, _ := math.NewDecFromString("2")

		result1, err := math.Exp1DivExp1(a1, b)
		require.NoError(t, err)

		result2, err := math.Exp1DivExp1(a2, b)
		require.NoError(t, err)

		require.True(t, result1.Lt(result2),
			"f(%s,%s)=%s should be less than f(%s,%s)=%s",
			a1.String(), b.String(), result1.String(),
			a2.String(), b.String(), result2.String())
	})

	t.Run("Anti-monotonicity in b: If b1 < b2 and a fixed, then f(a,b1) > f(a,b2)", func(t *testing.T) {
		a, _ := math.NewDecFromString("1")
		b1, _ := math.NewDecFromString("0")
		b2, _ := math.NewDecFromString("2")

		result1, err := math.Exp1DivExp1(a, b1)
		require.NoError(t, err)

		result2, err := math.Exp1DivExp1(a, b2)
		require.NoError(t, err)

		require.True(t, result1.Gt(result2),
			"f(%s,%s)=%s should be greater than f(%s,%s)=%s",
			a.String(), b1.String(), result1.String(),
			a.String(), b2.String(), result2.String())
	})

	t.Run("Limit behavior: As b -> -∞, f(a,b) -> exp(a) + 1", func(t *testing.T) {
		a, _ := math.NewDecFromString("2")
		b, _ := math.NewDecFromString("-20") // Very negative

		result, err := math.Exp1DivExp1(a, b)
		require.NoError(t, err)

		expA, err := math.Exp(a)
		require.NoError(t, err)
		expected, err := expA.Add(math.OneDec())
		require.NoError(t, err)

		tolerance, _ := math.NewDecFromString("0.01")
		withinTolerance, err := math.InDelta(expected, result, tolerance)
		require.NoError(t, err)
		require.True(t, withinTolerance,
			"f(%s,%s)=%s should be close to exp(%s)+1=%s when b is very negative",
			a.String(), b.String(), result.String(), a.String(), expected.String())
	})
}
