package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
)

func TestNextInflationRate(t *testing.T) {
	params := DefaultParams()
	halvingInterval := int64(25246080)

	tests := []struct {
		blockHeight     int64
		currenInflation math.LegacyDec
		expInflation    math.LegacyDec
	}{
		{0, params.InflationRateChange, params.InflationRateChange},                                           // Initial inflation
		{halvingInterval, params.InflationRateChange, params.InflationRateChange.QuoInt64(2)},                 // First halving
		{halvingInterval * 2, params.InflationRateChange.QuoInt64(2), params.InflationRateChange.QuoInt64(4)}, // Second halving                       // Between halvings
	}

	for i, tc := range tests {
		minter := DefaultInitialMinter()
		params.InflationRateChange = tc.currenInflation
		inflation := minter.NextInflationRate(params, tc.blockHeight)
		require.True(t, inflation.Equal(tc.expInflation),
			"test %d: expected inflation %s, got %s", i, tc.expInflation.String(), inflation.String())
	}
}

func TestNextAnnualProvisions(t *testing.T) {
	params := DefaultParams()
	totalCirculatingSupply := math.NewInt(1000000) // Example circulating supply

	// Define test cases with different inflation rates to simulate post-halving adjustments
	tests := []struct {
		currentInflation       math.LegacyDec
		totalCirculatingSupply math.Int
		expAnnualProvisions    math.LegacyDec
	}{
		{params.InflationRateChange, totalCirculatingSupply, params.InflationRateChange.MulInt(totalCirculatingSupply)},
		// After the first halving
		{params.InflationRateChange.QuoInt64(2), totalCirculatingSupply, params.InflationRateChange.QuoInt64(2).MulInt(totalCirculatingSupply)},
		// After the second halving
		{params.InflationRateChange.QuoInt64(4), totalCirculatingSupply, params.InflationRateChange.QuoInt64(4).MulInt(totalCirculatingSupply)},
	}

	for i, tc := range tests {
		minter := NewMinter(tc.currentInflation, math.LegacyNewDec(0)) // Initialize minter with the test case inflation

		minter.Inflation = tc.currentInflation

		annualProvisions := minter.NextAnnualProvisions(params, tc.totalCirculatingSupply)
		require.True(t, annualProvisions.Equal(tc.expAnnualProvisions),
			"test %d: expected annual provisions %s, got %s", i, tc.expAnnualProvisions.String(), annualProvisions.String())
	}
}
