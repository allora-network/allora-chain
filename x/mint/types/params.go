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
	blocksPerMonth uint64,
	emissionCalibrationTimestepPerMonth uint64,
	maxSupply math.Int,
	fEmissionNumerator math.Int,
	fEmissionDenominator math.Int,
	oneMonthSmoothingDegreeNumerator math.Int,
	oneMonthSmoothingDegreeDenominator math.Int,
) Params {
	return Params{
		MintDenom:                            mintDenom,
		BlocksPerMonth:                       blocksPerMonth,
		EmissionCalibrationsTimestepPerMonth: emissionCalibrationTimestepPerMonth,
		MaxSupply:                            maxSupply,
		FEmissionNumerator:                   fEmissionNumerator,
		FEmissionDenominator:                 fEmissionDenominator,
		OneMonthSmoothingDegreeNumerator:     oneMonthSmoothingDegreeNumerator,
		OneMonthSmoothingDegreeDenominator:   oneMonthSmoothingDegreeDenominator,
	}
}

// DefaultParams returns default x/mint module parameters.
func DefaultParams() Params {
	maxSupply, ok := math.NewIntFromString("1000000000000000000000000000")
	if !ok {
		panic("failed to parse max supply")
	}
	return Params{
		MintDenom:                            sdk.DefaultBondDenom,
		BlocksPerMonth:                       DefaultBlocksPerMonth(),
		EmissionCalibrationsTimestepPerMonth: uint64(30),        // "daily" emission calibration
		MaxSupply:                            maxSupply,         //1 billion allo * 1e18 (exponent) = 1e27 uallo
		FEmissionNumerator:                   math.NewInt(15),   // 0.015 per month
		FEmissionDenominator:                 math.NewInt(1000), // 0.015 per month is 15 over 1000
		OneMonthSmoothingDegreeNumerator:     math.NewInt(1),    // 0.1 at 1 month cadence
		OneMonthSmoothingDegreeDenominator:   math.NewInt(10),   // 0.1 is 1 over 10
	}
}

// Default previous emission per token is zero
func DefaultPreviousRewardEmissionPerUnitStakedTokenNumerator() math.Int {
	return math.ZeroInt()
}

// default previous emission per token denominator is 1
func DefaultPreviousRewardEmissionPerUnitStakedTokenDenominator() math.Int {
	return math.OneInt()
}

// no emission happened last block at genesis
func DefaultPreviousBlockEmission() math.Int {
	return math.ZeroInt()
}

// at genesis, nothing has been minted yet
func DefaultEcosystemTokensMinted() math.Int {
	return math.ZeroInt()
}

// ~5 seconds block time, 6311520 per year, 525960 per month
func DefaultBlocksPerMonth() uint64 {
	return uint64(525960)
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	if err := validateMintDenom(p.MintDenom); err != nil {
		return err
	}
	if err := validateBlocksPerMonth(p.BlocksPerMonth); err != nil {
		return err
	}
	if err := validateMaxSupply(p.MaxSupply); err != nil {
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

func validateBlocksPerMonth(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("blocks per month must be positive: %d", v)
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
