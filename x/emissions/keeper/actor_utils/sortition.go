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

/*
// Returns the percentile-ith value of the given sorted scores
// e.g. if percentile is 25%, for all the scores sorted from greatest to smallest
// give me the value that is greater than 25% of the values and less than 75% of the values
func GetPercentileOfScores(
	sortedScores []emissionstypes.Score,
	percentile alloraMath.Dec,
) (alloraMath.Dec, error) {
	if len(sortedIdsOfValues) == 0 {
		return alloraMath.Dec{}, emissionstypes.ErrEmptyArray
	}

	// position = (1 - q) * (n - 1)
	nLessOne, err := alloraMath.NewDecFromUint64(uint64(len(sortedIdsOfValues) - 1))
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
		return scoresByActor[sortedIdsOfValues[lowerIndexInt]].Score, nil
	}

	// return lowerValue + (upperValue-lowerValue)*(position-float64(lowerIndex))
	lowerScore := scoresByActor[sortedIdsOfValues[lowerIndexInt]].Score
	upperScore := scoresByActor[sortedIdsOfValues[upperIndexInt]].Score
	positionMinusLowerIndex, err := position.Sub(lowerIndex)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	upperMinusLower, err := upperScore.Sub(lowerScore)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	product, err := positionMinusLowerIndex.Mul(upperMinusLower)
	if err != nil {
		return alloraMath.Dec{}, err
	}
	return lowerScore.Add(product)
}

func UpdateScoresOfPassiveActorsWithActivePercentile(
	ctx sdk.Context,
	k *emissionskeeper.Keeper,
	blockHeight int64,
	topicId uint64,
	alphaRegret alloraMath.Dec,
	topicPercentile alloraMath.Dec,
	topScoresSorted []emissionstypes.Score,
	allScoresSorted []emissionstypes.Score,
	actorIsTop map[string]struct{},
	actorType emissionstypes.ActorType,
) error {
	// if the length of topScoresSorted is the
	// same as allScoresSorted, then all actors are top actors
	// and we don't have to do anything
	if len(topScoresSorted) == len(allScoresSorted) {
		return nil
	}

	percentile, err := GetPercentileOfScores(topScoresSorted, topicPercentile)
	if err != nil {
		return err
	}
	// Update score EMAs of all actors not in topActors
	for _, actor := range allActorsSorted {
		if _, ok := actorIsTop[actor.Address]; !ok {
			// Update the score EMA
			newScore := emissionstypes.Score{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Address:     actor.Address,
				Score:       percentile,
			}
			switch actorType {
			case emissionstypes.ActorType_INFERER:
				err := k.UpdateInfererScoreEma(ctx, topicId, alphaRegret, actor.Address, newScore)
				if err != nil {
					return err
				}
			case emissionstypes.ActorType_FORECASTER:
				err := k.UpdateForecasterScoreEma(ctx, topicId, alphaRegret, actor.Address, newScore)
				if err != nil {
					return err
				}
			case emissionstypes.ActorType_REPUTER:
				err := k.UpdateReputerScoreEma(ctx, topicId, alphaRegret, actor.Address, newScore)
				if err != nil {
					return err
				}
			default:
				return emissionstypes.ErrInvalidActorType
			}
		}
	}
	return nil
}

*/
