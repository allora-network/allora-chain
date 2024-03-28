package types

import "cosmossdk.io/math"

// NewGenesisState creates a new GenesisState object
func NewGenesisState(minter Minter, params Params, previousReward math.Int) *GenesisState {
	return &GenesisState{
		Minter:         minter,
		Params:         params,
		PreviousReward: previousReward,
	}
}

// DefaultGenesisState creates a default GenesisState object
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Minter:         DefaultInitialMinter(),
		Params:         DefaultParams(),
		PreviousReward: DefaultPreviousReward(),
	}
}

// ValidateGenesis validates the provided genesis state to ensure the
// expected invariants holds.
func ValidateGenesis(data GenesisState) error {
	if err := data.Params.Validate(); err != nil {
		return err
	}

	return ValidateMinter(data.Minter)
}
