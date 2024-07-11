package inference_synthesis

import (
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

// Filter nonces that are within the epoch length of the current block height
func FilterNoncesWithinEpochLength(n emissions.Nonces, blockHeight, epochLength int64) emissions.Nonces {
	var filtered emissions.Nonces
	for _, nonce := range n.Nonces {
		if blockHeight-nonce.BlockHeight <= epochLength {
			filtered.Nonces = append(filtered.Nonces, nonce)
		}
	}
	return filtered
}
