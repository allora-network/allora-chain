package actorutils

import (
	"math/rand"
	"slices"

	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Sorts the given actors by score, desc, breaking ties randomly
// Returns the top N actors as a map with the actor as the key and a boolean (True) as the value
func FindTopNByScoreDesc(
	ctx sdk.Context,
	n uint64,
	scores []emissionstypes.Score,
	randSeed int64,
) (topNActorsSorted []emissionstypes.Score, allActorsSorted []emissionstypes.Score, actorIsTop map[string]struct{}) {
	r := rand.New(rand.NewSource(randSeed)) //nolint:gosec // G404: Use of weak random number generator (math/rand or math/rand/v2 instead of crypto/rand)
	// in our tiebreaker, we never return that two elements are equal
	// so the sort function will never be called with two equal elements
	// so we don't care about sort stability because our tiebreaker determines the order
	// we also sort from largest to smallest so the sort function is inverted
	// from the usual smallest to largest sort order
	slices.SortFunc(scores, func(x, y emissionstypes.Score) int {
		if x.Score.Lt(y.Score) {
			return 1
		} else if x.Score.Gt(y.Score) {
			return -1
		} else {
			tiebreaker := r.Intn(2)
			if tiebreaker == 0 {
				return -1
			} else {
				return 1
			}
		}
	})

	// which is bigger, n or the length of the scores?
	N := int(n)
	if N > len(scores) {
		N = len(scores)
	}
	topNActorsSorted = make([]emissionstypes.Score, N)
	actorIsTop = make(map[string]struct{}, N)
	// populate top n actors sorted with only the top n
	// populate all with all
	// actor is top is a map of the top n actors
	for i := 0; i < N; i++ {
		topNActorsSorted[i] = scores[i]
		actorIsTop[scores[i].Address] = struct{}{}
	}

	return topNActorsSorted, scores, actorIsTop
}
