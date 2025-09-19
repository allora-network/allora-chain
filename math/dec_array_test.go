package math_test

import (
	"testing"

	"github.com/allora-network/allora-chain/math"
	"github.com/stretchr/testify/require"
)

func TestDecArray_Equal(t *testing.T) {
	testCases := []struct {
		name     string
		arr1     math.DecArray
		arr2     math.DecArray
		expected bool
	}{
		{
			name:     "both empty",
			arr1:     math.DecArray{},
			arr2:     math.DecArray{},
			expected: true,
		},
		{
			name:     "same elements same order",
			arr1:     math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0"), math.MustNewDecFromString("3.0")},
			arr2:     math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0"), math.MustNewDecFromString("3.0")},
			expected: true,
		},
		{
			name:     "same elements different order",
			arr1:     math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0"), math.MustNewDecFromString("3.0")},
			arr2:     math.DecArray{math.MustNewDecFromString("3.0"), math.MustNewDecFromString("2.0"), math.MustNewDecFromString("1.0")},
			expected: false,
		},
		{
			name:     "different lengths",
			arr1:     math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0")},
			arr2:     math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0"), math.MustNewDecFromString("3.0")},
			expected: false,
		},
		{
			name:     "different elements",
			arr1:     math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0"), math.MustNewDecFromString("3.0")},
			arr2:     math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0"), math.MustNewDecFromString("4.0")},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.arr1.Equal(tc.arr2))
		})
	}
}

func TestDecArray_Marshal(t *testing.T) {
	testCases := []struct {
		name     string
		arr      math.DecArray
		expected []byte
	}{
		{
			name:     "empty array",
			arr:      math.DecArray{},
			expected: []byte(`[]`),
		},
		{
			name:     "nil array",
			arr:      nil,
			expected: []byte(`[]`),
		},
		{
			name:     "single element",
			arr:      math.DecArray{math.MustNewDecFromString("1.0")},
			expected: []byte(`["1.0"]`),
		},
		{
			name:     "simple array",
			arr:      math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0"), math.MustNewDecFromString("3.0")},
			expected: []byte(`["1.0","2.0","3.0"]`),
		},
		{
			name:     "zero values",
			arr:      math.DecArray{math.MustNewDecFromString("0"), math.MustNewDecFromString("0.0")},
			expected: []byte(`["0","0.0"]`),
		},
		{
			name:     "very small values",
			arr:      math.DecArray{math.MustNewDecFromString("0.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001")},
			expected: []byte(`["0.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"]`),
		},
		{
			name:     "very large values",
			arr:      math.DecArray{math.MustNewDecFromString("999999999999999999999999999999999999999999999999999999.999999999999999999999999999999999999999999999999999999")},
			expected: []byte(`["999999999999999999999999999999999999999999999999999999.999999999999999999999999999999999999999999999999999999"]`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bz, err := tc.arr.Marshal()
			require.NoError(t, err)
			require.Equal(t, tc.expected, bz)
		})
	}
}

func TestDecArray_Unmarshal(t *testing.T) {
	testCases := []struct {
		name      string
		data      []byte
		expectErr bool
		expected  math.DecArray
	}{
		{
			name:      "empty array",
			data:      []byte(`[]`),
			expectErr: false,
			expected:  nil,
		},
		{
			name:      "single element",
			data:      []byte(`["1.0"]`),
			expectErr: false,
			expected:  math.DecArray{math.MustNewDecFromString("1.0")},
		},
		{
			name:      "simple array",
			data:      []byte(`["1.0","2.0","3.0"]`),
			expectErr: false,
			expected:  math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0"), math.MustNewDecFromString("3.0")},
		},
		{
			name:      "invalid json",
			data:      []byte(`["1.0",2.0,"3.0"]`),
			expectErr: true,
			expected:  nil,
		},
		{
			name:      "not an array",
			data:      []byte(`"1.0"`),
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
			data:      []byte(`"1.0"]`),
			expectErr: true,
		},
		{
			name:      "malformed - missing closing bracket",
			data:      []byte(`["1.0"`),
			expectErr: true,
		},
		{
			name:      "extra content after array",
			data:      []byte(`["1.0"]extra`),
			expectErr: false,
			expected:  math.DecArray{math.MustNewDecFromString("1.0")},
		},
		{
			name:      "invalid dec value",
			data:      []byte(`["invalid"]`),
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var arr math.DecArray
			err := arr.Unmarshal(tc.data)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, arr)
			}
		})
	}
}

func TestDecArray_Size(t *testing.T) {
	testCases := []struct {
		name string
		arr  math.DecArray
	}{
		{
			name: "empty array",
			arr:  math.DecArray{},
		},
		{
			name: "single element",
			arr:  math.DecArray{math.MustNewDecFromString("1.0")},
		},
		{
			name: "simple array",
			arr:  math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0"), math.MustNewDecFromString("3.0")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			size := tc.arr.Size()
			bz, err := tc.arr.Marshal()
			require.NoError(t, err)
			require.Len(t, bz, size)
		})
	}
}

func TestDecArray_MarshalTo(t *testing.T) {
	testCases := []struct {
		name           string
		arr            math.DecArray
		bufferSize     int
		expectErr      bool
		expectedBytes  int
		expectedPrefix []byte
	}{
		{
			name:           "empty array - sufficient buffer",
			arr:            math.DecArray{},
			bufferSize:     10,
			expectErr:      false,
			expectedBytes:  2,
			expectedPrefix: []byte(`[]`),
		},
		{
			name:           "empty array - insufficient buffer",
			arr:            math.DecArray{},
			bufferSize:     1,
			expectErr:      false,
			expectedBytes:  1,
			expectedPrefix: []byte(`[`),
		},
		{
			name:           "nil array - sufficient buffer",
			arr:            nil,
			bufferSize:     10,
			expectErr:      false,
			expectedBytes:  2,
			expectedPrefix: []byte(`[]`),
		},
		{
			name:           "single element - sufficient buffer",
			arr:            math.DecArray{math.MustNewDecFromString("1.0")},
			bufferSize:     10,
			expectErr:      false,
			expectedBytes:  7,
			expectedPrefix: []byte(`["1.0"]`),
		},
		{
			name:           "single element - insufficient buffer",
			arr:            math.DecArray{math.MustNewDecFromString("1.0")},
			bufferSize:     3,
			expectErr:      false,
			expectedBytes:  3,
			expectedPrefix: []byte(`["1`),
		},
		{
			name:           "simple array - sufficient buffer",
			arr:            math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0"), math.MustNewDecFromString("3.0")},
			bufferSize:     20,
			expectErr:      false,
			expectedBytes:  19,
			expectedPrefix: []byte(`["1.0","2.0","3.0"]`),
		},
		{
			name:           "simple array - insufficient buffer",
			arr:            math.DecArray{math.MustNewDecFromString("1.0"), math.MustNewDecFromString("2.0"), math.MustNewDecFromString("3.0")},
			bufferSize:     5,
			expectErr:      false,
			expectedBytes:  5,
			expectedPrefix: []byte(`["1.0`),
		},
		{
			name:           "zero buffer size",
			arr:            math.DecArray{math.MustNewDecFromString("1.0")},
			bufferSize:     0,
			expectErr:      false,
			expectedBytes:  0,
			expectedPrefix: []byte(``),
		},
		{
			name:           "exact buffer size",
			arr:            math.DecArray{math.MustNewDecFromString("1.0")},
			bufferSize:     7,
			expectErr:      false,
			expectedBytes:  7,
			expectedPrefix: []byte(`["1.0"]`),
		},
		{
			name:           "zero values - sufficient buffer",
			arr:            math.DecArray{math.MustNewDecFromString("0"), math.MustNewDecFromString("0.0")},
			bufferSize:     15,
			expectErr:      false,
			expectedBytes:  11,
			expectedPrefix: []byte(`["0","0.0"]`),
		},
		{
			name:           "very small values - sufficient buffer",
			arr:            math.DecArray{math.MustNewDecFromString("0.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001")},
			bufferSize:     150,
			expectErr:      false,
			expectedBytes:  114,
			expectedPrefix: []byte(`["0.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"]`),
		},
		{
			name:           "very large values - sufficient buffer",
			arr:            math.DecArray{math.MustNewDecFromString("999999999999999999999999999999999999999999999999999999.999999999999999999999999999999999999999999999999999999")},
			bufferSize:     150,
			expectErr:      false,
			expectedBytes:  113,
			expectedPrefix: []byte(`["999999999999999999999999999999999999999999999999999999.999999999999999999999999999999999999999999999999999999"]`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buffer := make([]byte, tc.bufferSize)
			n, err := tc.arr.MarshalTo(buffer)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedBytes, n)
				require.Equal(t, tc.expectedPrefix, buffer[:n])
			}
		})
	}
}
