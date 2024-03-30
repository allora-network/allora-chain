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
	blocksPerYear uint64,
	maxSupply math.Int,
) Params {
	return Params{
		MintDenom:     mintDenom,
		BlocksPerYear: blocksPerYear,
		MaxSupply:     maxSupply,
	}
}

// DefaultParams returns default x/mint module parameters.
func DefaultParams() Params {
	maxSupply, ok := math.NewIntFromString("1000000000000000000000000000")
	if !ok {
		panic("failed to parse max supply")
	}
	return Params{
		MintDenom:                   sdk.DefaultBondDenom,
		BlocksPerYear:               uint64(60 * 60 * 8766 / 5), // assuming 5 second block times
		MaxSupply:                   maxSupply,                  //1 billion allo * 1e18 (exponent) = 1e27 uallo
		FEmission:                   math.NewInt(15),            //  0.015 per month
		FEmissionPrec:               2,                          // true value of FEmission is FEmission divided by 10^FEmissionPrec
		OneMonthSmoothingDegree:     math.NewInt(1),             // 0.1 at 1 month cadence
		OneMonthSmoothingDegreePrec: 1,                          // true value of OneMonthSmoothingDegree is OneMonthSmoothingDegree divided by 10^OneMonthSmoothingPrec
	}
}

// Default previous emission per token is zero
func DefaultPreviousRewardEmissionsPerUnitStakedToken() math.Int {
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
	if err := validateBlocksPerYear(p.BlocksPerYear); err != nil {
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
