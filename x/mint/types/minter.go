package types

import (
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewMinter returns a new Minter object with the given inflation and annual
// provisions values.
func NewMinter(inflation, annualProvisions math.LegacyDec) Minter {
	return Minter{
		Inflation:        inflation,
		AnnualProvisions: annualProvisions,
	}
}

// InitialMinter returns an initial Minter object with a given inflation value.
func InitialMinter(inflation math.LegacyDec) Minter {
	return NewMinter(
		inflation,
		math.LegacyNewDec(0),
	)
}

// DefaultInitialMinter returns a default initial Minter object for a new chain
// which uses an inflation rate of 13%.
func DefaultInitialMinter() Minter {
	return InitialMinter(
		math.LegacyNewDecWithPrec(13, 2),
	)
}

// ValidateMinter does a basic validation on minter.
func ValidateMinter(minter Minter) error {
	if minter.Inflation.IsNegative() {
		return fmt.Errorf("mint parameter Inflation should be positive, is %s",
			minter.Inflation.String())
	}
	return nil
}

// NextInflationRate returns the new inflation rate for the next block.
func (m Minter) NextInflationRate(params Params, blockHeight int64) math.LegacyDec {
	// Block per year: 6311520
	// Halving interval (years): 4
	halvingInterval := int64(25246080)

	// Initial inflation rate
	currentInflationRate := params.InflationRateChange

	// Check for halving event
	if blockHeight > 0 && blockHeight%halvingInterval == 0 {
		// Halve the inflation rate
		currentInflationRate = currentInflationRate.QuoInt64(2)
	}

	return currentInflationRate
}

// NextAnnualProvisions returns the annual provisions based on current total
// circulating supply and inflation rate.
func (m Minter) NextAnnualProvisions(_ Params, totalCirculatingSupply math.Int) math.LegacyDec {
	return m.Inflation.MulInt(totalCirculatingSupply)
}

// BlockProvision returns the provisions for a block based on the annual
// provisions rate.
func (m Minter) BlockProvision(params Params) sdk.Coin {
	provisionAmt := m.AnnualProvisions.QuoInt(math.NewInt(int64(params.BlocksPerYear)))
	return sdk.NewCoin(params.MintDenom, provisionAmt.TruncateInt())
}
