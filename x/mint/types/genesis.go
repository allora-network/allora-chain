package types

import "cosmossdk.io/math"

// NewGenesisState creates a new GenesisState object
func NewGenesisState(
	params Params,
	previousRewardEmissionPerUnitStakedToken math.LegacyDec,
	previousBlockEmission math.Int,
	ecosystemTokensMinted math.Int,
	monthsUnlocked math.Int,
	startingEmissionsBlockHeight int64,
) *GenesisState {
	return &GenesisState{
		Params:                                   params,
		PreviousRewardEmissionPerUnitStakedToken: previousRewardEmissionPerUnitStakedToken,
		PreviousBlockEmission:                    previousBlockEmission,
		EcosystemTokensMinted:                    ecosystemTokensMinted,
		MonthsUnlocked:                           monthsUnlocked,
		StartingEmissionsBlockHeight:             startingEmissionsBlockHeight,
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
		StartingEmissionsBlockHeight:             0,
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

	maxMonths := math.NewIntFromUint64(36 + data.Params.InvestorsTeamVestingDelayMonths)
	if data.MonthsUnlocked.GT(maxMonths) {
		return ErrInvalidMonthsUnlocked
	}

	return nil
}
