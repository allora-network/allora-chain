package inference_synthesis

import (
	"sort"

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

func SortByBlockHeight(r []*emissions.ReputerRequestNonce) {
	sort.Slice(r, func(i, j int) bool {
		// Sorting in descending order (bigger values first)
		return r[i].ReputerNonce.BlockHeight > r[j].ReputerNonce.BlockHeight
	})
}

// Select the top N latest reputer nonces
func SelectTopNReputerNonces(reputerRequestNonces *emissions.ReputerRequestNonces, N int, currentBlockHeight, groundTruthLag, epochLength int64) []*emissions.ReputerRequestNonce {
	topN := make([]*emissions.ReputerRequestNonce, 0)
	// sort reputerRequestNonces by reputer block height

	// Create a copy of the original slice to avoid modifying chain state
	sortedSlice := make([]*emissions.ReputerRequestNonce, len(reputerRequestNonces.Nonces))
	copy(sortedSlice, reputerRequestNonces.Nonces)
	SortByBlockHeight(sortedSlice)

	// loop reputerRequestNonces
	for _, nonce := range sortedSlice {
		nonceCopy := nonce
		// Only select when the ground truth is available.
		if currentBlockHeight >= nonceCopy.ReputerNonce.BlockHeight+groundTruthLag+epochLength {
			topN = append(topN, nonceCopy)
		}
		if len(topN) >= N {
			break
		}
	}
	return topN
}

// Select the top N latest worker nonces
func SelectTopNWorkerNonces(workerNonces emissions.Nonces, N int) []*emissions.Nonce {
	if len(workerNonces.Nonces) <= N {
		return workerNonces.Nonces
	}
	return workerNonces.Nonces[:N]
}
