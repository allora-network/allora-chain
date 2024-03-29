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
	inflationRateChange,
	inflationMax,
	inflationMin,
	goalBonded math.LegacyDec,
	blocksPerYear uint64,
	maxSupply math.Uint,
	halvingInterval uint64,
	currentBlockProvision math.Uint,
) Params {
	return Params{
		MintDenom:             mintDenom,
		InflationRateChange:   inflationRateChange,
		InflationMax:          inflationMax,
		InflationMin:          inflationMin,
		GoalBonded:            goalBonded,
		BlocksPerYear:         blocksPerYear,
		MaxSupply:             maxSupply,
		HalvingInterval:       halvingInterval,
		CurrentBlockProvision: currentBlockProvision,
	}
}

// DefaultParams returns default x/mint module parameters.
func DefaultParams() Params {
	return Params{
		MintDenom:             sdk.DefaultBondDenom,
		InflationRateChange:   math.LegacyNewDecWithPrec(3573582624, 7),
		InflationMax:          math.LegacyNewDecWithPrec(3573582624, 7),
		InflationMin:          math.LegacyNewDecWithPrec(0, 2),
		GoalBonded:            math.LegacyNewDecWithPrec(67, 2),
		BlocksPerYear:         uint64(60 * 60 * 8766 / 5),                             // assuming 5 second block times
		MaxSupply:             math.NewUintFromString("1000000000000000000000000000"), //1 billion allo * 1e18 (exponent) = 1e27 uallo
		HalvingInterval:       uint64(25246080),
		CurrentBlockProvision: math.NewUintFromString("2831000000000000000000"), // uallo per block
	}
}

// Default previous emission per token is zero
func DefaultPreviousReward() math.Int {
	return math.NewInt(0)
}

// at genesis, nothing has been minted yet
func DefaultEcosystemTokensMinted() math.Int {
	return math.NewInt(0)
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	if err := validateMintDenom(p.MintDenom); err != nil {
		return err
	}
	if err := validateInflationRateChange(p.InflationRateChange); err != nil {
		return err
	}
	if err := validateInflationMax(p.InflationMax); err != nil {
		return err
	}
	if err := validateInflationMin(p.InflationMin); err != nil {
		return err
	}
	if err := validateGoalBonded(p.GoalBonded); err != nil {
		return err
	}
	if err := validateBlocksPerYear(p.BlocksPerYear); err != nil {
		return err
	}
	if p.InflationMax.LT(p.InflationMin) {
		return fmt.Errorf(
			"max inflation (%s) must be greater than or equal to min inflation (%s)",
			p.InflationMax, p.InflationMin,
		)
	}
	if err := validateMaxSupply(p.MaxSupply); err != nil {
		return err
	}
	if err := validateHalvingInterval(p.HalvingInterval); err != nil {
		return err
	}
	if err := validateCurrentBlockProvision(p.CurrentBlockProvision); err != nil {
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

func validateInflationRateChange(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("inflation rate change cannot be nil: %s", v)
	}
	if v.IsNegative() {
		return fmt.Errorf("inflation rate change cannot be negative: %s", v)
	}

	return nil
}

func validateInflationMax(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("max inflation cannot be nil: %s", v)
	}
	if v.IsNegative() {
		return fmt.Errorf("max inflation cannot be negative: %s", v)
	}

	return nil
}

func validateInflationMin(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("min inflation cannot be nil: %s", v)
	}
	if v.IsNegative() {
		return fmt.Errorf("min inflation cannot be negative: %s", v)
	}
	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("min inflation too large: %s", v)
	}

	return nil
}

func validateGoalBonded(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("goal bonded cannot be nil: %s", v)
	}
	if v.IsNegative() || v.IsZero() {
		return fmt.Errorf("goal bonded must be positive: %s", v)
	}
	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("goal bonded too large: %s", v)
	}

	return nil
}

func validateBlocksPerYear(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("blocks per year must be positive: %d", v)
	}

	return nil
}

func validateMaxSupply(i interface{}) error {
	v, ok := i.(math.Uint)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v.IsNil() {
		return fmt.Errorf("max supply cannot be nil: %s", v)
	}
	if v.LTE(math.NewUint(0)) {
		return fmt.Errorf("max supply must be positive: %s", v)
	}

	return nil
}

func validateHalvingInterval(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == 0 {
		return fmt.Errorf("halving interval must be positive: %d", v)
	}

	return nil
}

func validateCurrentBlockProvision(i interface{}) error {
	v, ok := i.(math.Uint)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v.IsNil() {
		return fmt.Errorf("current block provision cannot be nil: %s", v)
	}
	if v.LT(math.NewUint(0)) {
		return fmt.Errorf("current block provision cannot be negative: %s", v)
	}

	return nil
}
