package types

import (
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewParams returns Params instance with the given values.
func NewParams(
	mintDenom string,
	maxSupply math.Int,
	fEmission math.LegacyDec,
	oneMonthSmoothingDegree math.LegacyDec,
	ecosystemPercentOfTotalSupply math.LegacyDec,
	foundationPercentOfTotalSupply math.LegacyDec,
	participantsPercentOfTotalSupply math.LegacyDec,
	investorsPercentOfTotalSupply math.LegacyDec,
	investorsPreseedPercentOfTotalSupply math.LegacyDec,
	teamPercentOfTotalSupply math.LegacyDec,
	maxMonthlyPercentageYield math.LegacyDec,
) Params {
	return Params{
		MintDenom:                              mintDenom,
		MaxSupply:                              maxSupply,
		FEmission:                              fEmission,
		OneMonthSmoothingDegree:                oneMonthSmoothingDegree,
		EcosystemTreasuryPercentOfTotalSupply:  ecosystemPercentOfTotalSupply,
		FoundationTreasuryPercentOfTotalSupply: foundationPercentOfTotalSupply,
		ParticipantsPercentOfTotalSupply:       participantsPercentOfTotalSupply,
		InvestorsPercentOfTotalSupply:          investorsPercentOfTotalSupply,
		InvestorsPreseedPercentOfTotalSupply:   investorsPreseedPercentOfTotalSupply,
		TeamPercentOfTotalSupply:               teamPercentOfTotalSupply,
		MaximumMonthlyPercentageYield:          maxMonthlyPercentageYield,
	}
}

// DefaultParams returns default x/mint module parameters.
func DefaultParams() Params {
	maxSupply, ok := math.NewIntFromString("1000000000000000000000000000") // 1e27
	if !ok {
		panic("failed to parse max supply")
	}
	return Params{
		MintDenom:                              sdk.DefaultBondDenom,
		MaxSupply:                              maxSupply,                              // 1 billion allo * 1e18 (exponent) = 1e27 uallo
		FEmission:                              math.LegacyMustNewDecFromStr("0.025"),  // 0.025 per month
		OneMonthSmoothingDegree:                math.LegacyMustNewDecFromStr("0.1"),    // 0.1 at 1 month cadence
		EcosystemTreasuryPercentOfTotalSupply:  math.LegacyMustNewDecFromStr("0.3595"), // 35.95%
		FoundationTreasuryPercentOfTotalSupply: math.LegacyMustNewDecFromStr("0.1"),    // 10%
		ParticipantsPercentOfTotalSupply:       math.LegacyMustNewDecFromStr("0.055"),  // 5.5%
		InvestorsPercentOfTotalSupply:          math.LegacyMustNewDecFromStr("0.2605"), // 26.05%
		InvestorsPreseedPercentOfTotalSupply:   math.LegacyMustNewDecFromStr("0.05"),   // 5%
		TeamPercentOfTotalSupply:               math.LegacyMustNewDecFromStr("0.175"),  // 17.5%
		MaximumMonthlyPercentageYield:          math.LegacyMustNewDecFromStr("0.0095"), // .95% per month
	}
}

// Default previous emission per token is zero
func DefaultPreviousRewardEmissionPerUnitStakedToken() math.LegacyDec {
	return math.ZeroInt().ToLegacyDec()
}

// no emission happened last block at genesis
func DefaultPreviousBlockEmission() math.Int {
	return math.ZeroInt()
}

// at genesis, nothing has been minted yet
func DefaultEcosystemTokensMinted() math.Int {
	return math.ZeroInt()
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	if err := validateMintDenom(p.MintDenom); err != nil {
		return err
	}
	if err := validateMaxSupply(p.MaxSupply); err != nil {
		return err
	}
	if err := validateAFractionValue(p.FEmission); err != nil {
		return err
	}
	if err := validateAFractionValue(p.OneMonthSmoothingDegree); err != nil {
		return err
	}
	if err := validateAFractionValue(p.EcosystemTreasuryPercentOfTotalSupply); err != nil {
		return err
	}
	if err := validateAFractionValue(p.FoundationTreasuryPercentOfTotalSupply); err != nil {
		return err
	}
	if err := validateAFractionValue(p.ParticipantsPercentOfTotalSupply); err != nil {
		return err
	}
	if err := validateAFractionValue(p.InvestorsPercentOfTotalSupply); err != nil {
		return err
	}
	if err := validateAFractionValue(p.InvestorsPreseedPercentOfTotalSupply); err != nil {
		return err
	}
	if err := validateAFractionValue(p.TeamPercentOfTotalSupply); err != nil {
		return err
	}
	if err := validateTokenSupplyAddsTo100Percent(
		p.EcosystemTreasuryPercentOfTotalSupply,
		p.FoundationTreasuryPercentOfTotalSupply,
		p.ParticipantsPercentOfTotalSupply,
		p.InvestorsPercentOfTotalSupply,
		p.TeamPercentOfTotalSupply,
	); err != nil {
		return err
	}
	if err := validateAFractionValue(p.MaximumMonthlyPercentageYield); err != nil {
		return err
	}
	return nil
}

func validateMintDenom(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if strings.TrimSpace(v) == "" {
		return errors.New("mint denom cannot be blank")
	}
	if err := sdk.ValidateDenom(v); err != nil {
		return err
	}

	return nil
}

func validateMaxSupply(i interface{}) error {
	v, ok := i.(math.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v.IsNil() {
		return fmt.Errorf("max supply cannot be nil: %s", v)
	}
	if v.LTE(math.NewInt(0)) {
		return fmt.Errorf("max supply must be positive: %s", v)
	}

	return nil
}

func validateAFractionValue(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v.IsNil() {
		return fmt.Errorf("fractional value cannot be nil: %s", v)
	}
	if v.LT(math.LegacyNewDec(0)) {
		return fmt.Errorf("fractional value should be between 0 and 1, greater or equal to 0: %s", v)
	}
	if v.GT(math.LegacyNewDec(1)) {
		return fmt.Errorf("fractional value should be between 0 and 1, less or equal to 1: %s", v)
	}

	return nil
}

func validateTokenSupplyAddsTo100Percent(
	ecosystem math.LegacyDec,
	foundation math.LegacyDec,
	participants math.LegacyDec,
	investors math.LegacyDec,
	team math.LegacyDec,
) error {
	one := math.OneInt().ToLegacyDec()
	equal100Percent := one.Equal(
		ecosystem.
			Add(foundation).
			Add(participants).
			Add(investors).
			Add(team),
	)
	if !equal100Percent {
		return fmt.Errorf(
			"total supply percentages do not add up to 100 percent: %s %s %s %s %s",
			ecosystem,
			foundation,
			participants,
			investors,
			team,
		)
	}
	return nil
}
