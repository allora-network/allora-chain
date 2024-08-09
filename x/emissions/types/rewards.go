package types

import alloraMath "github.com/allora-network/allora-chain/math"

type TaskRewardType string

const (
	ReputerAndDelegatorRewardType TaskRewardType = "ReputerAndDelegator"
	WorkerInferenceRewardType     TaskRewardType = "WorkerInference"
	WorkerForecastRewardType      TaskRewardType = "WorkerForecast"
)

type TaskReward struct {
	Address string
	Reward  alloraMath.Dec
	TopicId TopicId
	Type    TaskRewardType
}
