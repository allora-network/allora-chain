package inference_synthesis

import (
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
type Stake = cosmosMath.Int

func CosmosIntOneE18() cosmosMath.Int {
	ret, ok := cosmosMath.NewIntFromString("1000000000000000000")
	if !ok {
		panic("1*10^18 is not a valid cosmos int")
	}
	return ret
}
