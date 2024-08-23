package actorutils

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	emissionskeeper "github.com/allora-network/allora-chain/x/emissions/keeper"
	emissionstypes "github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
