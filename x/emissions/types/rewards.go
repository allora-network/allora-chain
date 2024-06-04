package types

import alloraMath "github.com/allora-network/allora-chain/math"

type TaskRewardType int

const (
	ReputerAndDelegatorRewardType TaskRewardType = iota // iota resets to 0 for the first constant in the block.
	WorkerInferenceRewardType
	WorkerForecastRewardType
)

type TaskReward struct {
	Address string
	Reward  alloraMath.Dec
	TopicId TopicId
	Type    TaskRewardType
}
