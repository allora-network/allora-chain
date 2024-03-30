package types

import "cosmossdk.io/math"

// NewGenesisState creates a new GenesisState object
func NewGenesisState(
	params Params,
	previousRewardEmissionsPerUnitStakedToken math.Int,
	ecosystemTokensMinted math.Int,
) *GenesisState {
	return &GenesisState{
		Params: params,
		PreviousRewardEmissionsPerUnitStakedToken: previousRewardEmissionsPerUnitStakedToken,
		EcosystemTokensMinted:                     ecosystemTokensMinted,
	}
}

// DefaultGenesisState creates a default GenesisState object
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		PreviousRewardEmissionsPerUnitStakedToken: DefaultPreviousRewardEmissionsPerUnitStakedToken(),
		EcosystemTokensMinted:                     DefaultEcosystemTokensMinted(),
	}
}

// ValidateGenesis validates the provided genesis state to ensure the
// expected invariants holds.
func ValidateGenesis(data GenesisState) error {
	if err := data.Params.Validate(); err != nil {
		return err
	}

	if data.PreviousRewardEmissionsPerUnitStakedToken.IsNegative() {
		return ErrInvalidPreviousRewardEmissionsPerUnitStakedToken
	}

	if data.EcosystemTokensMinted.IsNegative() {
		return ErrInvalidEcosystemTokensMinted
	}

	return nil
}
