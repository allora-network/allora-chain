package types

import "cosmossdk.io/errors"

func NewGenesisState(tasks []Task) GenesisState {
	return GenesisState{
		Tasks: tasks,
	}
}

func DefaultGenesisState() GenesisState {
	return NewGenesisState(nil)
}

func (gs *GenesisState) Validate() error {
	ids := map[TaskID]struct{}{}
	for _, t := range gs.Tasks {
		if t.Interval != nil && *t.Interval <= 0 {
			return errors.Wrapf(ErrInvalidTask, "negative interval '%s'", t.Id)
		}

		if _, exists := ids[t.Id]; exists {
			return errors.Wrapf(ErrInvalidTask, "duplicate task id '%s'", t.Id)
		}

		ids[t.Id] = struct{}{}
	}
	return nil
}
