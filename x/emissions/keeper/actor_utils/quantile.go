package actorutils

import (
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
	return alloraMath.GetQuantileOfDecs(decScores, quantile)
}
