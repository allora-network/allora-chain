package types

import "cosmossdk.io/math"

// NewGenesisState creates a new GenesisState object
func NewGenesisState(
	params Params,
	previousRewardEmissionPerUnitStakedToken math.LegacyDec,
	previousBlockEmission math.Int,
	ecosystemTokensMinted math.Int,
	monthsUnlocked math.Int,
) *GenesisState {
	return &GenesisState{
		Params:                                   params,
		PreviousRewardEmissionPerUnitStakedToken: previousRewardEmissionPerUnitStakedToken,
		PreviousBlockEmission:                    previousBlockEmission,
		EcosystemTokensMinted:                    ecosystemTokensMinted,
		MonthsUnlocked:                           monthsUnlocked,
	}
}

// DefaultGenesisState creates a default GenesisState object
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:                                   DefaultParams(),
		PreviousRewardEmissionPerUnitStakedToken: DefaultPreviousRewardEmissionPerUnitStakedToken(),
		PreviousBlockEmission:                    DefaultPreviousBlockEmission(),
		EcosystemTokensMinted:                    DefaultEcosystemTokensMinted(),
		MonthsUnlocked:                           math.NewInt(0),
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

	if data.MonthsUnlocked.IsNegative() {
		return ErrInvalidMonthsUnlocked
	}

	thirtySix := math.NewInt(36)
	if data.MonthsUnlocked.GT(thirtySix) {
		return ErrInvalidMonthsUnlocked
	}

	return nil
}
