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

// LegacyNonce returns the transitional BlockHeight-keyed nonce for the legacy data plane
// (submissions, inference/loss stores, rewards, existing window events).
// Canonical epoch identity remains Nonce (NonceV2); do not coerce Nonce.Payload() to a height.
func (e Epoch) LegacyNonce() Nonce {
	return Nonce{BlockHeight: e.StartBlockHeight}
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
	// Reputer opens at GroundTruthLag (legacy EndBlocker / spec), not GTL+ExtraLag.
	// ExtraLag extends the close so the window still lands on an epoch boundary.
	reputerOpenAt := startAt.Add(time.Duration(topic.GroundTruthLag) * time.Second)
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
			CloseAt: reputerOpenAt.Add(time.Duration(extraLag+topic.EpochLength) * time.Second),
		},
		Epsilon: topic.Epsilon,
	}
}
