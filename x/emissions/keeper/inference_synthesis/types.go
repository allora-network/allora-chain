package inferencesynthesis

import (
	cosmosMath "cosmossdk.io/math"
	alloraMath "github.com/allora-network/allora-chain/math"
)

type Worker = string
type Inferer = string
type Forecaster = string
type BlockHeight = int64
type TopicId = uint64
type Regret = alloraMath.Dec
type Loss = alloraMath.Dec
type Weight = alloraMath.Dec
type InferenceValue = alloraMath.Dec
type Stake = cosmosMath.Int

// Need to differentiate between the two types of regrets because workers may complete tasks
// for both roles and may have different regrets for those different roles
type RegretInformedWeights struct {
	inferers    map[Worker]Weight
	forecasters map[Worker]Weight
}
