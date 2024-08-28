package types

import alloraMath "github.com/allora-network/allora-chain/math"

type BlockHeight = int64
type TopicId = uint64
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

// contains a score and a pointer to the
// actor with the next lowest score,
// for updating the lowest scores in the keeper
type ScoreAndNextLowest struct {
	Score      Score
	NextLowest string
}
