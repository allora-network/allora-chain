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
