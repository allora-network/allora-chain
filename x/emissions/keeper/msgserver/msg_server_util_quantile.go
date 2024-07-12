package msgserver

import (
	alloraMath "github.com/allora-network/allora-chain/math"
	"github.com/allora-network/allora-chain/x/emissions/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Returns the quantile of the given slice of scores inputted in descending order
func GetQuantileOfDescSliceAsAsc(
	scoresByActor map[Actor]Score,
	sortedIdsOfValues []Actor,
	quantile alloraMath.Dec,
) (alloraMath.Dec, error) {
	if len(sortedIdsOfValues) == 0 {
		return alloraMath.Dec{}, types.ErrEmptyArray
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

func (ms msgServer) UpdateScoresOfPassiveActorsWithActiveQuantile(
	ctx sdk.Context,
	blockHeight int64,
	maxTopWorkersToReward uint64,
	topicId TopicId,
	alphaRegret alloraMath.Dec,
	topicQuantile alloraMath.Dec,
	actorScoreEmas map[Actor]types.Score,
	topActors []Actor,
	allActorsSorted []Actor,
	actorIsTop map[Actor]bool,
	actorType types.ActorType,
) error {
	// if 1/max > topic.quantile, then just use 1/max as the quantile
	maxNum, err := alloraMath.NewDecFromUint64(maxTopWorkersToReward)
	if err != nil {
		return err
	}
	oneOverMax, err := alloraMath.OneDec().Quo(maxNum)
	if err != nil {
		return err
	}
	if oneOverMax.Gt(topicQuantile) {
		topicQuantile = oneOverMax
	}
	quantile, err := GetQuantileOfDescSliceAsAsc(actorScoreEmas, topActors, topicQuantile)
	if err != nil {
		return err
	}
	// Update score EMAs of all actors not in topActors
	for _, actor := range allActorsSorted {
		if _, ok := actorIsTop[actor]; !ok {
			// Update the score EMA
			newScore := types.Score{
				TopicId:     topicId,
				BlockHeight: blockHeight,
				Address:     actor,
				Score:       quantile,
			}
			switch actorType {
			case types.ActorType_INFERER:
				err := ms.k.UpdateInfererScoreEma(ctx, topicId, alphaRegret, actor, newScore)
				if err != nil {
					return err
				}
			case types.ActorType_FORECASTER:
				err := ms.k.UpdateForecasterScoreEma(ctx, topicId, alphaRegret, actor, newScore)
				if err != nil {
					return err
				}
			case types.ActorType_REPUTER:
				err := ms.k.UpdateReputerScoreEma(ctx, topicId, alphaRegret, actor, newScore)
				if err != nil {
					return err
				}
			default:
				return types.ErrInvalidActorType
			}
		}
	}
	return nil
}
