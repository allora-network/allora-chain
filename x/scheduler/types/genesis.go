package types

// GenesisState defines the scheduler module's genesis state.
type GenesisState struct {
	// Tasks defines the list of tasks at genesis
	Tasks []Task `json:"tasks,omitempty"`
}

// DefaultGenesisState returns the default genesis state
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Tasks: []Task{},
	}
}

// ValidateGenesis validates the scheduler module's genesis state.
func ValidateGenesis(data GenesisState) error {
	// Add validation logic here if needed
	return nil
}

