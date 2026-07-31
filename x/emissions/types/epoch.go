package types

import (
	"time"

	"cosmossdk.io/collections"
	"github.com/allora-network/allora-chain/fsm"
)

func (e EpochState) Name() string {
	return e.String()
}

func (e *Epoch) CurrentState() fsm.State {
	return e.State
}

func (e *Epoch) Advance(to fsm.State) {
	e.State = to.(EpochState)
}

func (e *Epoch) Key() collections.Pair[TopicId, NonceV2] {
	return collections.Join(e.TopicId, e.Nonce)
}

// TopicExtraLag returns the alignment lag used so reputer windows land on epoch boundaries.
// Matches the legacy EndBlocker formula: when GroundTruthLag is not a multiple of EpochLength,
// pad to the next EpochLength multiple.
func TopicExtraLag(topic Topic) int64 {
	if topic.EpochLength <= 0 {
		return 0
	}
	rem := topic.GroundTruthLag % topic.EpochLength
	if rem == 0 {
		return 0
	}
	return topic.EpochLength - rem
}

func NewEpoch(nonce NonceV2, topic Topic, startAt time.Time) Epoch {
	extraLag := TopicExtraLag(topic)
	reputerOpenAt := startAt.Add(time.Duration(topic.GroundTruthLag+extraLag) * time.Second)
	return Epoch{
		Nonce:   nonce,
		TopicId: topic.Id,
		State:   EpochState_INIT,
		WorkerSubmissionWindow: &Window{
			OpenAt:  startAt,
			CloseAt: startAt.Add(time.Duration(topic.WorkerSubmissionWindow) * time.Second),
		},
		ReputerSubmissionWindow: &Window{
			OpenAt:  reputerOpenAt,
			CloseAt: reputerOpenAt.Add(time.Duration(topic.EpochLength) * time.Second),
		},
		Epsilon: topic.Epsilon,
	}
}
