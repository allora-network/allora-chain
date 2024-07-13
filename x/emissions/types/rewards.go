package types

import alloraMath "github.com/allora-network/allora-chain/math"

type TaskRewardType string

const (
	ReputerAndDelegatorRewardType TaskRewardType = "ReputerAndDelegator" // iota resets to 0 for the first constant in the block.
	WorkerInferenceRewardType                    = "WorkerInference"
	WorkerForecastRewardType                     = "WorkerForecast"
)

type TaskReward struct {
	Address string
	Reward  alloraMath.Dec
	TopicId TopicId
	Type    TaskRewardType
}
