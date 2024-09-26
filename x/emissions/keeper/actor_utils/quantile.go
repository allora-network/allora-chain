package actorutils

import (
	"slices"

	alloraMath "github.com/allora-network/allora-chain/math"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
)

// Returns the quantile value of the given sorted scores
// e.g. if quantile is 0.25 (25%), for all the scores sorted from greatest to smallest
// give me the value that is greater than 25% of the values and less than 75% of the values
// the domain of this quantile is assumed to be between 0 and 1.
// Scores should be of unique actors => no two elements have the same actor address.
func GetQuantileOfScores(
	scores []emissionstypes.Score,
	quantile alloraMath.Dec,
) (alloraMath.Dec, error) {
	decScores := make([]alloraMath.Dec, len(scores))
	for i, score := range scores {
		decScores[i] = score.Score
	}
	return GetQuantileOfDecs(decScores, quantile)
}

func GetQuantileOfDecs(
	decs []alloraMath.Dec,
	quantile alloraMath.Dec,
) (alloraMath.Dec, error) {
	// If there are no decs then the quantile of scores is 0.
	// This better ensures chain continuity without consequence because in this situation
	// there is no meaningful quantile to calculate.
	if len(decs) == 0 {
		return alloraMath.ZeroDec(), nil
	}

	// Sort decs in descending order. Address is used to break ties.
	slices.SortStableFunc(decs, func(x, y alloraMath.Dec) int {
		if x.Lt(y) {
			return 1
		}
		return -1
	})

	// n elements, q quantile
	// position = (1 - q) * (n - 1)
	nLessOne, err := alloraMath.NewDecFromUint64(uint64(len(decs) - 1))
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
		return decs[lowerIndexInt], nil
	}

	// in cases where the quantile is between two values
	// return lowerValue + (upperValue-lowerValue)*(position-lowerIndex)
	lowerDec := decs[lowerIndexInt]
	upperDec := decs[upperIndexInt]
	positionMinusLowerIndex, err := position.Sub(lowerIndex)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	upperMinusLower, err := upperDec.Sub(lowerDec)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	product, err := positionMinusLowerIndex.Mul(upperMinusLower)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	ret, err := lowerDec.Add(product)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return ret, nil
}
