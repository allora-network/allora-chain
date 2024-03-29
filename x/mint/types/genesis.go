package types

import "cosmossdk.io/math"

// NewGenesisState creates a new GenesisState object
func NewGenesisState(minter Minter, params Params, previousReward math.Int, ecosystemTokensMinted math.Int) *GenesisState {
	return &GenesisState{
		Minter:                minter,
		Params:                params,
		PreviousReward:        previousReward,
		EcosystemTokensMinted: ecosystemTokensMinted,
	}
}

// DefaultGenesisState creates a default GenesisState object
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Minter:                DefaultInitialMinter(),
		Params:                DefaultParams(),
		PreviousReward:        DefaultPreviousReward(),
		EcosystemTokensMinted: DefaultEcosystemTokensMinted(),
	}
}

// ValidateGenesis validates the provided genesis state to ensure the
// expected invariants holds.
func ValidateGenesis(data GenesisState) error {
	if err := data.Params.Validate(); err != nil {
		return err
	}

	if err := ValidateMinter(data.Minter); err != nil {
		return err
	}

	if data.PreviousReward.IsNegative() {
		return ErrInvalidPreviousReward
	}

	if data.EcosystemTokensMinted.IsNegative() {
		return ErrInvalidEcosystemTokensMinted
	}

	return nil
}
