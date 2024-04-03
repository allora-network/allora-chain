package types

import "cosmossdk.io/math"

// NewGenesisState creates a new GenesisState object
func NewGenesisState(
	params Params,
	previousRewardEmissionPerUnitStakedTokenNumerator math.Int,
	previousRewardEmissionPerUnitStakedTokenDenominator math.Int,
	ecosystemTokensMinted math.Int,
) *GenesisState {
	return &GenesisState{
		Params: params,
		PreviousRewardEmissionPerUnitStakedTokenNumerator:   previousRewardEmissionPerUnitStakedTokenNumerator,
		PreviousRewardEmissionPerUnitStakedTokenDenominator: previousRewardEmissionPerUnitStakedTokenDenominator,
		EcosystemTokensMinted:                               ecosystemTokensMinted,
	}
}

// DefaultGenesisState creates a default GenesisState object
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		PreviousRewardEmissionPerUnitStakedTokenNumerator:   DefaultPreviousRewardEmissionPerUnitStakedTokenNumerator(),
		PreviousRewardEmissionPerUnitStakedTokenDenominator: DefaultPreviousRewardEmissionPerUnitStakedTokenDenominator(),
		EcosystemTokensMinted:                               DefaultEcosystemTokensMinted(),
	}
}

// ValidateGenesis validates the provided genesis state to ensure the
// expected invariants holds.
func ValidateGenesis(data GenesisState) error {
	if err := data.Params.Validate(); err != nil {
		return err
	}

	if data.PreviousRewardEmissionPerUnitStakedTokenNumerator.IsNegative() {
		return ErrInvalidPreviousRewardEmissionPerUnitStakedToken
	}
	if data.PreviousRewardEmissionPerUnitStakedTokenDenominator.IsNegative() {
		return ErrInvalidPreviousRewardEmissionPerUnitStakedToken
	}

	if data.EcosystemTokensMinted.IsNegative() {
		return ErrInvalidEcosystemTokensMinted
	}

	return nil
}
