package types

import (
	"time"

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

func NewEpoch(topic Topic, startAt time.Time) Epoch {
	// TODO: fetch last topic epoch nonce, from topic?
	lastNonce := ZeroNonce()

	return Epoch{
		Nonce:   lastNonce.NextNonce(),
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
