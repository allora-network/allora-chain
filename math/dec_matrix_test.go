package math_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/allora-network/allora-chain/math"
)

func TestDecMatrix_Equal(t *testing.T) {
	testCases := []struct {
		name     string
		m1       math.DecMatrix
		m2       math.DecMatrix
		expected bool
	}{
		{
			name:     "both empty",
			m1:       math.DecMatrix{},
			m2:       math.DecMatrix{},
			expected: true,
		},
		{
			name:     "both nil",
			m1:       nil,
			m2:       nil,
			expected: true,
		},
		{
			name: "same rows same order",
			m1: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0")},
				math.DecArray{math.MustNewDecFromString("3.0"), math.MustNewDecFromString("4.0")},
			},
			m2: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0")},
				math.DecArray{math.MustNewDecFromString("3.0"), math.MustNewDecFromString("4.0")},
			},
			expected: true,
		},
		{
			name: "same elements different row order",
			m1: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0")},
				math.DecArray{math.MustNewDecFromString("3.0"), math.MustNewDecFromString("4.0")},
			},
			m2: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("3.0"), math.MustNewDecFromString("4.0")},
				math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0")},
			},
			expected: false,
		},
		{
			name: "different number of rows",
			m1: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0")},
			},
			m2: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0")},
				math.DecArray{math.MustNewDecFromString("3.0"), math.MustNewDecFromString("4.0")},
			},
			expected: false,
		},
		{
			name: "different row length",
			m1: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0")},
			},
			m2: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0")},
			},
			expected: false,
		},
		{
			name: "different elements",
			m1: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0")},
			},
			m2: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("9.0")},
			},
			expected: false,
		},
		{
			name: "nil vs empty",
			m1:   nil,
			m2:   math.DecMatrix{},
			// Match DecArray semantics: nil marshals as [] and should be treated equal to empty.
			// If you decide otherwise in DecMatrix.Equal, flip this expected value.
			expected: true,
		},
		{
			name: "row nil vs empty row",
			m1: math.DecMatrix{
				nil,
			},
			m2: math.DecMatrix{
				math.DecArray{},
			},
			// Match DecArray semantics: nil == empty.
			// If you decide otherwise, flip this expected value.
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.m1.Equal(tc.m2))
		})
	}
}

func TestDecMatrix_Marshal(t *testing.T) {
	testCases := []struct {
		name     string
		m        math.DecMatrix
		expected []byte
	}{
		{
			name:     "empty matrix",
			m:        math.DecMatrix{},
			expected: []byte(`[]`),
		},
		{
			name:     "nil matrix",
			m:        nil,
			expected: []byte(`[]`),
		},
		{
			name: "single row single element",
			m: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0")},
			},
			expected: []byte(`[["1.0"]]`),
		},
		{
			name: "simple matrix",
			m: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0")},
				math.DecArray{math.MustNewDecFromString("3.0"), math.MustNewDecFromString("4.0")},
			},
			expected: []byte(`[["1.0","2.0"],["3.0","4.0"]]`),
		},
		{
			name: "zero values",
			m: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("0"), math.MustNewDecFromString("0.0")},
				math.DecArray{},
			},
			expected: []byte(`[["0","0.0"],[]]`),
		},
		{
			name: "very small values",
			m: math.DecMatrix{
				math.DecArray{
					math.MustNewDecFromString("0.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"),
				},
			},
			expected: []byte(`[["0.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"]]`),
		},
		{
			name: "very large values",
			m: math.DecMatrix{
				math.DecArray{
					math.MustNewDecFromString("999999999999999999999999999999999999999999999999999999.999999999999999999999999999999999999999999999999999999"),
				},
			},
			expected: []byte(`[["999999999999999999999999999999999999999999999999999999.999999999999999999999999999999999999999999999999999999"]]`),
		},
		{
			name: "nil row",
			m: math.DecMatrix{
				nil,
			},
			expected: []byte(`[[]]`),
		},
		{
			name: "mixed empty and nil rows",
			m: math.DecMatrix{
				math.DecArray{},
				nil,
				math.DecArray{math.MustNewDecFromString("1.0")},
			},
			expected: []byte(`[[],[],["1.0"]]`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bz, err := tc.m.Marshal()
			require.NoError(t, err)
			require.Equal(t, tc.expected, bz)
		})
	}
}

func TestDecMatrix_Unmarshal(t *testing.T) {
	testCases := []struct {
		name      string
		data      []byte
		expectErr bool
		expected  math.DecMatrix
	}{
		{
			name:      "empty matrix",
			data:      []byte(`[]`),
			expectErr: false,
			expected:  nil,
		},
		{
			name:      "single row single element",
			data:      []byte(`[["1.0"]]`),
			expectErr: false,
			expected: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0")},
			},
		},
		{
			name:      "simple matrix",
			data:      []byte(`[["1.0","2.0"],["3.0","4.0"]]`),
			expectErr: false,
			expected: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0")},
				math.DecArray{math.MustNewDecFromString("3.0"), math.MustNewDecFromString("4.0")},
			},
		},
		{
			name:      "row empty",
			data:      []byte(`[[]]`),
			expectErr: false,
			expected:  math.DecMatrix{nil},
		},
		{
			name:      "invalid json - row not array",
			data:      []byte(`[["1.0"],"2.0"]`),
			expectErr: true,
			expected:  nil,
		},
		{
			name:      "invalid json - dec not string",
			data:      []byte(`[["1.0",2.0]]`),
			expectErr: true,
			expected:  nil,
		},
		{
			name:      "not a matrix",
			data:      []byte(`"1.0"`),
			expectErr: true,
			expected:  nil,
		},
		{
			name:      "not an array outer",
			data:      []byte(`{"a":1}`),
			expectErr: true,
			expected:  nil,
		},
		{
			name:      "empty string",
			data:      []byte(``),
			expectErr: true,
			expected:  nil,
		},
		{
			name:      "malformed - missing opening bracket",
			data:      []byte(`["1.0"]]`),
			expectErr: true,
			expected:  nil,
		},
		{
			name:      "malformed - missing closing bracket",
			data:      []byte(`[["1.0"]`),
			expectErr: true,
			expected:  nil,
		},
		{
			name:      "extra content after matrix",
			data:      []byte(`[["1.0"]]extra`),
			expectErr: true,
			expected:  nil,
		},
		{
			name:      "invalid dec value",
			data:      []byte(`[["invalid"]]`),
			expectErr: true,
			expected:  nil,
		},
		{
			name:      "null row not allowed (should error)",
			data:      []byte(`[null]`),
			expectErr: true,
			expected:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var m math.DecMatrix
			err := m.Unmarshal(tc.data)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, m)
			}
		})
	}
}

func TestDecMatrix_Size(t *testing.T) {
	testCases := []struct {
		name string
		m    math.DecMatrix
	}{
		{
			name: "empty matrix",
			m:    math.DecMatrix{},
		},
		{
			name: "single element",
			m: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0")},
			},
		},
		{
			name: "simple matrix",
			m: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0")},
				math.DecArray{math.MustNewDecFromString("3.0"), math.MustNewDecFromString("4.0")},
			},
		},
		{
			name: "nil matrix",
			m:    nil,
		},
		{
			name: "matrix with empty and nil rows",
			m: math.DecMatrix{
				math.DecArray{},
				nil,
				math.DecArray{math.MustNewDecFromString("1.0")},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			size := tc.m.Size()
			bz, err := tc.m.Marshal()
			require.NoError(t, err)
			require.Len(t, bz, size)
		})
	}
}

func TestDecMatrix_MarshalTo(t *testing.T) {
	testCases := []struct {
		name           string
		m              math.DecMatrix
		bufferSize     int
		expectedBytes  int
		expectedPrefix []byte
	}{
		{
			name:           "empty matrix - sufficient buffer",
			m:              math.DecMatrix{},
			bufferSize:     10,
			expectedBytes:  2,
			expectedPrefix: []byte(`[]`),
		},
		{
			name:           "empty matrix - insufficient buffer",
			m:              math.DecMatrix{},
			bufferSize:     1,
			expectedBytes:  1,
			expectedPrefix: []byte(`[`),
		},
		{
			name:           "nil matrix - sufficient buffer",
			m:              nil,
			bufferSize:     10,
			expectedBytes:  2,
			expectedPrefix: []byte(`[]`),
		},
		{
			name: "single element - sufficient buffer",
			m: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0")},
			},
			bufferSize:     20,
			expectedBytes:  len([]byte(`[["1.0"]]`)),
			expectedPrefix: []byte(`[["1.0"]]`),
		},
		{
			name: "single element - insufficient buffer",
			m: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0")},
			},
			bufferSize:     4,
			expectedBytes:  4,
			expectedPrefix: []byte(`[["1`),
		},
		{
			name: "simple matrix - sufficient buffer",
			m: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0")},
				math.DecArray{math.MustNewDecFromString("3.0"), math.MustNewDecFromString("4.0")},
			},
			bufferSize:     50,
			expectedBytes:  len([]byte(`[["1.0","2.0"],["3.0","4.0"]]`)),
			expectedPrefix: []byte(`[["1.0","2.0"],["3.0","4.0"]]`),
		},
		{
			name: "simple matrix - insufficient buffer",
			m: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0")},
				math.DecArray{math.MustNewDecFromString("3.0"), math.MustNewDecFromString("4.0")},
			},
			bufferSize:     6,
			expectedBytes:  6,
			expectedPrefix: []byte(`[["1.0`),
		},
		{
			name: "zero buffer size",
			m: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0")},
			},
			bufferSize:     0,
			expectedBytes:  0,
			expectedPrefix: []byte(``),
		},
		{
			name: "exact buffer size",
			m: math.DecMatrix{
				math.DecArray{math.MustNewDecFromString("1.0")},
			},
			bufferSize:     len([]byte(`[["1.0"]]`)),
			expectedBytes:  len([]byte(`[["1.0"]]`)),
			expectedPrefix: []byte(`[["1.0"]]`),
		},
		{
			name: "very small values - sufficient buffer",
			m: math.DecMatrix{
				math.DecArray{
					math.MustNewDecFromString("0.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"),
				},
			},
			bufferSize:     200,
			expectedBytes:  len([]byte(`[["0.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"]]`)),
			expectedPrefix: []byte(`[["0.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"]]`),
		},
		{
			name: "very large values - sufficient buffer",
			m: math.DecMatrix{
				math.DecArray{
					math.MustNewDecFromString("999999999999999999999999999999999999999999999999999999.999999999999999999999999999999999999999999999999999999"),
				},
			},
			bufferSize:     200,
			expectedBytes:  len([]byte(`[["999999999999999999999999999999999999999999999999999999.999999999999999999999999999999999999999999999999999999"]]`)),
			expectedPrefix: []byte(`[["999999999999999999999999999999999999999999999999999999.999999999999999999999999999999999999999999999999999999"]]`),
		},
		{
			name: "nil row - sufficient buffer",
			m: math.DecMatrix{
				nil,
			},
			bufferSize:     10,
			expectedBytes:  len([]byte(`[[]]`)),
			expectedPrefix: []byte(`[[]]`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buffer := make([]byte, tc.bufferSize)
			n, err := tc.m.MarshalTo(buffer)

			require.NoError(t, err)
			require.Equal(t, tc.expectedBytes, n)
			require.Equal(t, tc.expectedPrefix, buffer[:n])
		})
	}
}
