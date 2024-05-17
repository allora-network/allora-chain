package inference_synthesis

import (
	"sort"

	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
)

type Worker = string
type BlockHeight = int64
type TopicId = uint64
type Regret = alloraMath.Dec
type Loss = alloraMath.Dec
type Weight = alloraMath.Dec
type InferenceValue = alloraMath.Dec
type Stake = cosmosMath.Uint

const (
	oneE18 = "1000000000000000000"
)

func AlloraOneE18() (alloraMath.Dec, error) {
	oneE18, err := alloraMath.NewDecFromString(oneE18)
	return oneE18, err
}

func CosmosUintOneE18() cosmosMath.Uint {
	return cosmosMath.NewUintFromString(oneE18)
}

func GetSortedStringKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
