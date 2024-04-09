package inference_synthesis

import (
	emissions "github.com/allora-network/allora-chain/x/emissions/types"
)

type Worker = string
type BlockHeight = int64
type TopicId = uint64
type Regret = float64
type Loss = float64
type Weight = float64
type InferenceValue = float64
type Stake = emissions.StakePlacement
