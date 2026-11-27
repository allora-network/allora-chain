package testcommon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// nextNodeThroughCopy takes the config by value, the way every integration and fuzz
// check does.
func nextNodeThroughCopy(m TestConfig) int64 {
	return m.Client.getNextClientNumber()
}

func TestRoundRobinRotatesAcrossTestConfigCopies(t *testing.T) {
	config := TestConfig{Client: newBarrierTestClient(t, RoundRobin, newFakeCometNode(t, 1, 1), newFakeCometNode(t, 1, 1), newFakeCometNode(t, 1, 1))} //nolint:exhaustruct // only the client is exercised

	got := make([]int64, 0, 7)
	for range 7 {
		got = append(got, nextNodeThroughCopy(config))
	}

	require.Equal(t, []int64{0, 1, 2, 0, 1, 2, 0}, got)
}
