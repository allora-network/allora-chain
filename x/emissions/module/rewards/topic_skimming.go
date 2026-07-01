package rewards

import (
	"sort"

	alloraMath "github.com/allora-network/allora-chain/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Returns a map of topicId to weights of the top N topics by weight in descending order
// It is assumed that topicIds is of a reasonable size, throttled by perhaps MaxTopicsPerBlock global param
func SkimTopTopicsByWeightDesc(ctx sdk.Context, weights map[TopicId]*alloraMath.Dec, N uint64) (map[TopicId]*alloraMath.Dec, []TopicId, error) {
	topicIds := make([]TopicId, 0, len(weights))
	for topicId := range weights { //nolint:maprange // reason: iteration to array before sorting
		topicIds = append(topicIds, topicId)
	}
	// Sort topicIds by weight desc to ensure deterministic order. Tiebreak with topicId ascending
	sort.Slice(topicIds, func(i, j int) bool {
		if (*weights[topicIds[i]]).Equal(*weights[topicIds[j]]) {
			return topicIds[i] < topicIds[j]
		}
		return (*weights[topicIds[i]]).Gt(*weights[topicIds[j]])
	})

	numberToAdd := N
	if (uint64)(len(topicIds)) < N {
		numberToAdd = (uint64)(len(topicIds))
	}

	weightsOfTopN := make(map[TopicId]*alloraMath.Dec, numberToAdd)
	listOfTopN := make([]TopicId, numberToAdd)
	for i := uint64(0); i < numberToAdd; i++ {
		weightsOfTopN[topicIds[i]] = weights[topicIds[i]]
		listOfTopN[i] = topicIds[i]
	}

	Logger(ctx).Debug("SkimTopTopicsByWeightDesc took top", "number", numberToAdd, "from topics", len(topicIds))

	return weightsOfTopN, listOfTopN, nil
}
