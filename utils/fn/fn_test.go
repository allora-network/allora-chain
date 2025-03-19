package fn

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestElementsMatch(t *testing.T) {
	tcs := []struct {
		a, b []int
		want bool
	}{
		{[]int{1, 2, 3}, []int{1, 2, 4}, false},
		{[]int{1, 2, 3}, []int{1, 2}, false},
		{[]int{1}, []int{1}, true},
		{[]int{1, 2, 3}, []int{1, 2, 3}, true},
		{[]int{1, 2, 3}, []int{3, 2, 1}, true},
		{[]int{1, 2, 3}, []int{2, 3, 1}, true},
	}

	for _, tc := range tcs {
		require.Equal(t, tc.want, ElementsMatch(tc.a, tc.b))
	}
}
