package types

import (
	"fmt"

	"cosmossdk.io/math"
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
		// 2831000000000000000000 (initial uallo per block) * 6311520 (blocks per year) = 17867913120000000000000000 (initial annual provisions)
		math.LegacyNewDecFromBigInt(math.NewUintFromString("17867913120000000000000000").BigInt()),
	)
}

// DefaultInitialMinter returns a default initial Minter object for a new chain
// which uses an inflation rate of 15,03%.
func DefaultInitialMinter() Minter {
	return InitialMinter(
		math.LegacyNewDecWithPrec(3573582624, 7),
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

func (m Minter) NextInflationRate(totalCirculatingSupply math.Int, currentBlockProvision math.LegacyDec, blocksPerYear uint64) math.LegacyDec {
	return currentBlockProvision.Mul(math.LegacyNewDec(int64(blocksPerYear))).Quo(math.LegacyNewDecFromInt(totalCirculatingSupply))
}
