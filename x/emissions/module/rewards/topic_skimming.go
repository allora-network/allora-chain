package rewards

import (
	"fmt"
	"math/rand"
	"sort"

	"cosmossdk.io/errors"
	alloraMath "github.com/allora-network/allora-chain/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// A structure to hold the original value and a random tiebreaker
type SortableTopicId struct {
	Value      TopicId
	Weight     *alloraMath.Dec
	Tiebreaker uint32
}

// Sorts the given slice of topics in descending order according to their corresponding return, using pseudorandom tiebreaker
// e.g. ([]uint64{1, 2, 3}, map[uint64]uint64{1: 2, 2: 2, 3: 3}, 0) -> [3, 1, 2] or [3, 2, 1]
func SortTopicsByWeightDescWithRandomTiebreaker(topicIds []TopicId, weights map[TopicId]*alloraMath.Dec, randSeed BlockHeight) ([]TopicId, error) {
	// Convert the slice of Ts to a slice of SortableItems, each with a random tiebreaker
	r := rand.New(rand.NewSource(randSeed)) //nolint:gosec // G404: Use of weak random number generator (math/rand or math/rand/v2 instead of crypto/rand)
	items := make([]SortableTopicId, len(topicIds))
	for i, topicId := range topicIds {
		items[i] = SortableTopicId{topicId, weights[topicId], r.Uint32()}
	}

	// Sort the slice of SortableItems
	// If the values are equal, the tiebreaker will decide their order
	sort.Slice(items, func(i, j int) bool {
		if items[i].Value == items[j].Value {
			return items[i].Tiebreaker > items[j].Tiebreaker
		}
		return (*items[i].Weight).Gt(*items[j].Weight)
	})

	// Extract and print the sorted values to demonstrate the sorting
	sortedValues := make([]TopicId, len(topicIds))
	for i, item := range items {
		sortedValues[i] = item.Value
	}
	return sortedValues, nil
}

// Returns a map of topicId to weights of the top N topics by weight in descending order
// It is assumed that topicIds is of a reasonable size, throttled by perhaps MaxTopicsPerBlock global param
func SkimTopTopicsByWeightDesc(ctx sdk.Context, weights map[TopicId]*alloraMath.Dec, N uint64, block BlockHeight) (map[TopicId]*alloraMath.Dec, []TopicId, error) {
	topicIds := make([]TopicId, 0, len(weights))
	for topicId := range weights {
		topicIds = append(topicIds, topicId)
	}
	// Sort topicIds by weight desc to ensure deterministic order. Tiebreak with topicId ascending
	sort.Slice(topicIds, func(i, j int) bool {
		if (*weights[topicIds[i]]).Equal(*weights[topicIds[j]]) {
			return topicIds[i] < topicIds[j]
		}
		return (*weights[topicIds[i]]).Gt(*weights[topicIds[j]])
	})
	sortedTopicIds, err := SortTopicsByWeightDescWithRandomTiebreaker(topicIds, weights, block)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to sort topics by weight desc with random tiebreaker")
	}

	numberToAdd := N
	if (uint64)(len(sortedTopicIds)) < N {
		numberToAdd = (uint64)(len(sortedTopicIds))
	}

	weightsOfTopN := make(map[TopicId]*alloraMath.Dec, numberToAdd)
	listOfTopN := make([]TopicId, numberToAdd)
	for i := uint64(0); i < numberToAdd; i++ {
		weightsOfTopN[sortedTopicIds[i]] = weights[sortedTopicIds[i]]
		listOfTopN[i] = sortedTopicIds[i]
	}

	Logger(ctx).Debug(
		fmt.Sprintf("SkimTopTopicsByWeightDesc took top %d topics out of %d",
			numberToAdd, len(sortedTopicIds)))

	return weightsOfTopN, listOfTopN, nil
}
