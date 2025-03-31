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
			original, err := bounded.ToDec()
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

			actual, err := bounded.ToDec()
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

			actual, err := bounded.ToDec()
			require.NoError(t, err, "ToDec failed")
			require.True(t, tt.expectedValue.Equal(actual),
				"capped value should equal expected value. got: %v, want: %v",
				actual, tt.expectedValue)
		})
	}
}
