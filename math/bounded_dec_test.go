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
