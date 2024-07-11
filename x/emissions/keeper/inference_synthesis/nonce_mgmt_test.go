package inference_synthesis_test

import (
	"testing"

	"github.com/allora-network/allora-chain/x/emissions/keeper/inference_synthesis"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	"github.com/stretchr/testify/require"
)

func TestFilterNoncesWithinEpochLength(t *testing.T) {
	tests := []struct {
		name          string
		nonces        emissionstypes.Nonces
		blockHeight   int64
		epochLength   int64
		expectedNonce emissionstypes.Nonces
	}{
		{
			name: "Nonces within epoch length",
			nonces: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 10},
					{BlockHeight: 15},
				},
			},
			blockHeight: 20,
			epochLength: 10,
			expectedNonce: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 10},
					{BlockHeight: 15},
				},
			},
		},
		{
			name: "Nonces outside epoch length",
			nonces: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 5},
					{BlockHeight: 15},
				},
			},
			blockHeight: 20,
			epochLength: 10,
			expectedNonce: emissionstypes.Nonces{
				Nonces: []*emissionstypes.Nonce{
					{BlockHeight: 15},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := inference_synthesis.FilterNoncesWithinEpochLength(tc.nonces, tc.blockHeight, tc.epochLength)
			require.Equal(t, tc.expectedNonce, actual, "Filter nonces do not match")
		})
	}
}
