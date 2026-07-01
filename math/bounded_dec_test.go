package math

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewBoundedExp40Dec(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid small number",
			input:   "1.23",
			wantErr: false,
		},
		{
			name:    "valid zero",
			input:   "0",
			wantErr: false,
		},
		{
			name:    "valid negative",
			input:   "-1.23",
			wantErr: false,
		},
		{
			name:    "valid at upper bound",
			input:   "1e40",
			wantErr: false,
		},
		{
			name:    "valid at lower bound",
			input:   "1e-40",
			wantErr: false,
		},
		{
			name:    "invalid over upper bound",
			input:   "1e41",
			wantErr: true,
		},
		{
			name:    "invalid under lower bound",
			input:   "1e-41",
			wantErr: true,
		},
		{
			name:    "negative: valid at upper bound",
			input:   "-1e40",
			wantErr: false,
		},
		{
			name:    "negative: valid at lower bound",
			input:   "-1e-40",
			wantErr: false,
		},
		{
			name:    "negative: invalid over upper bound",
			input:   "-1e41",
			wantErr: true,
		},
		{
			name:    "negative: invalid under lower bound",
			input:   "-1e-41",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec, err := NewDecFromString(tt.input)
			require.NoError(t, err, "NewDecFromString failed")

			bounded, err := NewBoundedExp40Dec(dec)
			if tt.wantErr {
				require.Error(t, err, "NewBoundedExp40Dec should have failed")
				return
			}
			require.NoError(t, err, "NewBoundedExp40Dec failed")

			// Verify the value was stored correctly
			original := bounded.ToDec()
			require.NoError(t, err)
			require.True(t, dec.Equal(original), "stored value should equal input value")
		})
	}
}

func TestNewCappedBoundedExp40Dec(t *testing.T) {
	tests := []struct {
		name          string
		input         Dec
		expectedValue Dec
		wantErr       bool
	}{
		{
			name:          "within bounds: positive",
			input:         MustNewDecFromString("1.23"),
			expectedValue: MustNewDecFromString("1.23"),
			wantErr:       false,
		},
		{
			name:          "within bounds: zero",
			input:         ZeroDec(),
			expectedValue: ZeroDec(),
			wantErr:       false,
		},
		{
			name:          "within bounds: negative",
			input:         MustNewDecFromString("-1.23"),
			expectedValue: MustNewDecFromString("-1.23"),
			wantErr:       false,
		},
		{
			name:          "at upper bound",
			input:         MustNewDecFromString("1e40"),
			expectedValue: MustNewDecFromString("1e40"),
			wantErr:       false,
		},
		{
			name:          "at lower bound",
			input:         MustNewDecFromString("1e-40"),
			expectedValue: MustNewDecFromString("1e-40"),
			wantErr:       false,
		},
		{
			name:          "caps to upper bound",
			input:         MustNewDecFromString("1e41"),
			expectedValue: MustNewDecFromString("1e40"),
			wantErr:       false,
		},
		{
			name:          "caps to lower bound",
			input:         MustNewDecFromString("1e-41"),
			expectedValue: MustNewDecFromString("1e-40"),
			wantErr:       false,
		},
		{
			name:          "negative: at upper bound",
			input:         MustNewDecFromString("-1e40"),
			expectedValue: MustNewDecFromString("-1e40"),
			wantErr:       false,
		},
		{
			name:          "negative: at lower bound",
			input:         MustNewDecFromString("-1e-40"),
			expectedValue: MustNewDecFromString("-1e-40"),
			wantErr:       false,
		},
		{
			name:          "negative: caps to upper bound",
			input:         MustNewDecFromString("-1e41"),
			expectedValue: MustNewDecFromString("-1e40"),
			wantErr:       false,
		},
		{
			name:          "negative: caps to lower bound",
			input:         MustNewDecFromString("-1e-41"),
			expectedValue: MustNewDecFromString("-1e-40"),
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bounded, err := NewCappedBoundedExp40Dec(tt.input)
			if tt.wantErr {
				require.Error(t, err, "NewCappedBoundedExp40Dec should have failed")
				return
			}
			require.NoError(t, err, "NewCappedBoundedExp40Dec failed")

			actual := bounded.ToDec()
			require.NoError(t, err, "ToDec failed")
			require.True(t, tt.expectedValue.Equal(actual),
				"capped value should equal expected value. got: %v, want: %v",
				actual, tt.expectedValue)
		})
	}
}

func TestNewCappedBoundedExp40DecFromString(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue Dec
		wantErr       bool
	}{
		{
			name:          "within bounds: positive",
			input:         "1.23",
			expectedValue: MustNewDecFromString("1.23"),
			wantErr:       false,
		},
		{
			name:          "within bounds: zero",
			input:         "0",
			expectedValue: ZeroDec(),
			wantErr:       false,
		},
		{
			name:          "within bounds: negative",
			input:         "-1.23",
			expectedValue: MustNewDecFromString("-1.23"),
			wantErr:       false,
		},
		{
			name:          "at upper bound",
			input:         "1e40",
			expectedValue: MustNewDecFromString("1e40"),
			wantErr:       false,
		},
		{
			name:          "at lower bound",
			input:         "1e-40",
			expectedValue: MustNewDecFromString("1e-40"),
			wantErr:       false,
		},
		{
			name:          "caps to upper bound",
			input:         "1e41",
			expectedValue: MustNewDecFromString("1e40"),
			wantErr:       false,
		},
		{
			name:          "caps to lower bound",
			input:         "1e-41",
			expectedValue: MustNewDecFromString("1e-40"),
			wantErr:       false,
		},
		{
			name:          "negative: at upper bound",
			input:         "-1e40",
			expectedValue: MustNewDecFromString("-1e40"),
			wantErr:       false,
		},
		{
			name:          "negative: at lower bound",
			input:         "-1e-40",
			expectedValue: MustNewDecFromString("-1e-40"),
			wantErr:       false,
		},
		{
			name:          "negative: caps to upper bound",
			input:         "-1e41",
			expectedValue: MustNewDecFromString("-1e40"),
			wantErr:       false,
		},
		{
			name:          "negative: caps to lower bound",
			input:         "-1e-41",
			expectedValue: MustNewDecFromString("-1e-40"),
			wantErr:       false,
		},
		{
			name:          "invalid decimal string",
			input:         "not.a.number",
			expectedValue: ZeroDec(),
			wantErr:       true,
		},
		{
			name:          "empty string",
			input:         "",
			expectedValue: ZeroDec(),
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bounded, err := NewCappedBoundedExp40DecFromString(tt.input)
			if tt.wantErr {
				require.Error(t, err, "NewCappedBoundedExp40DecFromString should have failed")
				return
			}
			require.NoError(t, err, "NewCappedBoundedExp40DecFromString failed")

			actual := bounded.ToDec()
			require.NoError(t, err, "ToDec failed")
			require.True(t, tt.expectedValue.Equal(actual),
				"capped value should equal expected value. got: %v, want: %v",
				actual, tt.expectedValue)
		})
	}
}

func TestMustNewCappedBoundedExp40Dec(t *testing.T) {
	tests := []struct {
		name      string
		input     Dec
		wantPanic bool
	}{
		{
			name:      "within bounds: positive",
			input:     MustNewDecFromString("1.23"),
			wantPanic: false,
		},
		{
			name:      "within bounds: zero",
			input:     ZeroDec(),
			wantPanic: false,
		},
		{
			name:      "within bounds: negative",
			input:     MustNewDecFromString("-1.23"),
			wantPanic: false,
		},
		{
			name:      "at upper bound",
			input:     MustNewDecFromString("1e40"),
			wantPanic: false,
		},
		{
			name:      "at lower bound",
			input:     MustNewDecFromString("1e-40"),
			wantPanic: false,
		},
		{
			name:      "caps to upper bound",
			input:     MustNewDecFromString("1e41"),
			wantPanic: false,
		},
		{
			name:      "caps to lower bound",
			input:     MustNewDecFromString("1e-41"),
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				require.Panics(t, func() {
					MustNewCappedBoundedExp40Dec(tt.input)
				}, "MustNewCappedBoundedExp40Dec should panic")
				return
			}

			require.NotPanics(t, func() {
				bounded := MustNewCappedBoundedExp40Dec(tt.input)
				actual := bounded.ToDec()

				// For values that get capped, verify they're within bounds
				absValue, err := actual.Abs()
				require.NoError(t, err)

				if !absValue.IsZero() {
					require.True(t, absValue.Gte(minBoundValue), "value should be >= minimum bound")
					require.True(t, absValue.Lte(maxBoundValue), "value should be <= maximum bound")
				}
			}, "MustNewCappedBoundedExp40Dec should not panic")
		})
	}
}

func TestMustNewCappedBoundedExp40DecFromString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantPanic bool
	}{
		{
			name:      "within bounds: positive",
			input:     "1.23",
			wantPanic: false,
		},
		{
			name:      "within bounds: zero",
			input:     "0",
			wantPanic: false,
		},
		{
			name:      "within bounds: negative",
			input:     "-1.23",
			wantPanic: false,
		},
		{
			name:      "at upper bound",
			input:     "1e40",
			wantPanic: false,
		},
		{
			name:      "at lower bound",
			input:     "1e-40",
			wantPanic: false,
		},
		{
			name:      "caps to upper bound",
			input:     "1e41",
			wantPanic: false,
		},
		{
			name:      "caps to lower bound",
			input:     "1e-41",
			wantPanic: false,
		},
		{
			name:      "invalid decimal string",
			input:     "not.a.number",
			wantPanic: true,
		},
		{
			name:      "non-finite: infinity",
			input:     "Inf",
			wantPanic: true,
		},
		{
			name:      "empty string",
			input:     "",
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				require.Panics(t, func() {
					MustNewCappedBoundedExp40DecFromString(tt.input)
				}, "MustNewCappedBoundedExp40DecFromString should panic")
				return
			}

			require.NotPanics(t, func() {
				bounded := MustNewCappedBoundedExp40DecFromString(tt.input)
				actual := bounded.ToDec()

				// For values that get capped, verify they're within bounds
				absValue, err := actual.Abs()
				require.NoError(t, err)

				if !absValue.IsZero() {
					require.True(t, absValue.Gte(minBoundValue), "value should be >= minimum bound")
					require.True(t, absValue.Lte(maxBoundValue), "value should be <= maximum bound")
				}
			}, "MustNewCappedBoundedExp40DecFromString should not panic")
		})
	}
}

func TestGetMaxPositiveBoundaryExp40Dec(t *testing.T) {
	require := require.New(t)

	maxBoundary := GetMaxPositiveBoundaryExp40Dec()
	value := maxBoundary.ToDec()

	// Verify it equals the max bound value
	require.True(value.Equal(maxBoundValue),
		"max boundary should equal maxBoundValue. got: %v, want: %v",
		value, maxBoundValue)

	// Verify we can't create a larger value
	largerDec := MustNewDecFromString("1e41")
	capped, err := NewCappedBoundedExp40Dec(largerDec)
	require.NoError(err)
	cappedValue := capped.ToDec()
	require.NoError(err)
	require.True(cappedValue.Equal(value),
		"capped larger value should equal max boundary. got: %v, want: %v",
		cappedValue, value)
}

func TestGetMinPositiveBoundaryExp40Dec(t *testing.T) {
	require := require.New(t)

	minBoundary := GetMinPositiveBoundaryExp40Dec()
	value := minBoundary.ToDec()

	// Verify it equals the min bound value
	require.True(value.Equal(minBoundValue),
		"min boundary should equal minBoundValue. got: %v, want: %v",
		value, minBoundValue)

	// Verify we can't create a smaller positive value
	smallerDec := MustNewDecFromString("1e-41")
	capped, err := NewCappedBoundedExp40Dec(smallerDec)
	require.NoError(err)
	cappedValue := capped.ToDec()
	require.NoError(err)
	require.True(cappedValue.Equal(value),
		"capped smaller value should equal min boundary. got: %v, want: %v",
		cappedValue, value)

	// Verify zero is still allowed to be smaller
	zeroDec := ZeroDec()
	zeroValue, err := NewCappedBoundedExp40Dec(zeroDec)
	require.NoError(err)
	zeroDecValue := zeroValue.ToDec()
	require.NoError(err)
	require.True(zeroDecValue.Lt(value),
		"zero should be less than min boundary")
}
