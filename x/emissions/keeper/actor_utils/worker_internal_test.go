package actorutils

import (
	"testing"

	"github.com/stretchr/testify/require"

	alloraMath "github.com/allora-network/allora-chain/math"
)

func TestBuildSortedAddressWeightsSortedAndAligned(t *testing.T) {
	weightsByAddress := map[string]alloraMath.Dec{
		"allo1ccc": alloraMath.MustNewDecFromString("0.30"),
		"allo1aaa": alloraMath.MustNewDecFromString("0.10"),
		"allo1bbb": alloraMath.MustNewDecFromString("0.20"),
	}

	addresses, weights := buildSortedAddressWeights(weightsByAddress)

	require.Equal(t, []string{"allo1aaa", "allo1bbb", "allo1ccc"}, addresses)
	require.Len(t, weights, len(addresses))

	for i, addr := range addresses {
		expectedWeight := weightsByAddress[addr]
		require.Truef(t, weights[i].Equal(expectedWeight), "weight mismatch for address %s", addr)
	}
}

func TestBuildSortedAddressWeightsEmptyInput(t *testing.T) {
	addresses, weights := buildSortedAddressWeights(map[string]alloraMath.Dec{})
	require.Nil(t, addresses)
	require.Nil(t, weights)
}

