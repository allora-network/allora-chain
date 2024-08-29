package actorutils

import (
	"math/rand"
	"slices"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Sorts the given actors by score, desc, breaking ties randomly
// Returns the top N actors sorted, all actors sorted, and a mapping of actor to whether they are in the top N
// used for permeability in the merit based sortition of the active set of inferers, forecasters, and reputers
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
	N := n
	if N > uint64(len(scores)) {
		N = uint64(len(scores))
	}
	topNActorsSorted = make([]emissionstypes.Score, N)
	actorIsTop = make(map[string]struct{}, N)
	// populate top n actors sorted with only the top n
	// populate all with all
	// actor is top is a map of the top n actors
	for i := uint64(0); i < N; i++ {
		topNActorsSorted[i] = scores[i]
		actorIsTop[scores[i].Address] = struct{}{}
	}

	return topNActorsSorted, scores, actorIsTop
}

// Returns the quantile value of the given sorted scores
// e.g. if quantile is 0.25 (25%), for all the scores sorted from greatest to smallest
// give me the value that is greater than 25% of the values and less than 75% of the values
// the domain of this quantile is assumed to be between 0 and 1
func GetQuantileOfScores(
	sortedScores []emissionstypes.Score,
	quantile alloraMath.Dec,
) (alloraMath.Dec, error) {
	// if there are no scores then the quantile of scores is 0
	if len(sortedScores) == 0 {
		return alloraMath.ZeroDec(), nil
	}
	// n elements, q quantile
	// position = (1 - q) * (n - 1)
	nLessOne, err := alloraMath.NewDecFromUint64(uint64(len(sortedScores) - 1))
	if err != nil {
		return alloraMath.Dec{}, err
	}
	oneLessQ, err := alloraMath.OneDec().Sub(quantile)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	position, err := oneLessQ.Mul(nLessOne)
	if err != nil {
		return alloraMath.Dec{}, err
	}

	lowerIndex, err := position.Floor()
	if err != nil {
		return alloraMath.Dec{}, err
	}
	lowerIndexInt, err := lowerIndex.Int64()
	if err != nil {
		return alloraMath.Dec{}, err
	}
	upperIndex, err := position.Ceil()
	if err != nil {
		return alloraMath.Dec{}, err
	}
	upperIndexInt, err := upperIndex.Int64()
	if err != nil {
		return alloraMath.Dec{}, err
	}

	if lowerIndex == upperIndex {
		return sortedScores[lowerIndexInt].Score, nil
	}

	// in cases where the quantile is between two values
	// return lowerValue + (upperValue-lowerValue)*(position-lowerIndex)
	lowerScore := sortedScores[lowerIndexInt]
	upperScore := sortedScores[upperIndexInt]
	positionMinusLowerIndex, err := position.Sub(lowerIndex)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	upperMinusLower, err := upperScore.Score.Sub(lowerScore.Score)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	product, err := positionMinusLowerIndex.Mul(upperMinusLower)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	ret, err := lowerScore.Score.Add(product)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return ret, nil
}
