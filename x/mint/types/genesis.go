package types

import "cosmossdk.io/math"

// NewGenesisState creates a new GenesisState object
func NewGenesisState(
	params Params,
	previousRewardEmissionPerUnitStakedToken math.LegacyDec,
	previousBlockEmission math.Int,
	ecosystemTokensMinted math.Int,
) *GenesisState {
	return &GenesisState{
		Params:                                   params,
		PreviousRewardEmissionPerUnitStakedToken: previousRewardEmissionPerUnitStakedToken,
		PreviousBlockEmission:                    previousBlockEmission,
		EcosystemTokensMinted:                    ecosystemTokensMinted,
	}
}

// DefaultGenesisState creates a default GenesisState object
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:                                   DefaultParams(),
		PreviousRewardEmissionPerUnitStakedToken: DefaultPreviousRewardEmissionPerUnitStakedToken(),
		PreviousBlockEmission:                    DefaultPreviousBlockEmission(),
		EcosystemTokensMinted:                    DefaultEcosystemTokensMinted(),
	}
}

// ValidateGenesis validates the provided genesis state to ensure the
// expected invariants holds.
func ValidateGenesis(data GenesisState) error {
	if err := data.Params.Validate(); err != nil {
		return err
	}

	if data.PreviousRewardEmissionPerUnitStakedToken.IsNegative() {
		return ErrInvalidPreviousRewardEmissionPerUnitStakedToken
	}

	if data.EcosystemTokensMinted.IsNegative() {
		return ErrInvalidEcosystemTokensMinted
	}

	return nil
}
