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
	state, ok := to.(EpochState)
	if !ok {
		panic("epoch FSM advanced to a non-EpochState")
	}
	e.State = state
}

func (e *Epoch) Key() collections.Pair[TopicId, NonceV2] {
	return collections.Join(e.TopicId, e.Nonce)
}

func NewEpoch(nonce NonceV2, topic Topic, startAt time.Time) Epoch {
	return Epoch{
		Nonce:   nonce,
		TopicId: topic.Id,
		State:   EpochState_INIT,
		WorkerSubmissionWindow: &Window{
			OpenAt:  startAt,
			CloseAt: startAt.Add(time.Duration(topic.WorkerSubmissionWindow) * time.Second),
		},
		ReputerSubmissionWindow: &Window{
			OpenAt:  startAt.Add(time.Duration(topic.GroundTruthLag) * time.Second),
			CloseAt: startAt.Add(time.Duration(topic.GroundTruthLag) * time.Second).Add(time.Duration(topic.EpochLength) * time.Second),
		},
		Epsilon: topic.Epsilon,
	}
}
