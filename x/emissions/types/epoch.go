package types

import "github.com/allora-network/allora-chain/fsm"

func (e EpochState) Name() string {
	return e.String()
}

func (e *Epoch) CurrentState() fsm.State {
	return e.State
}

func (e *Epoch) Advance(to fsm.State) {
	e.State = to.(EpochState)
}
