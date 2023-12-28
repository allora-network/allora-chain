package inflation

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"cosmossdk.io/math"
)

// CustomInflationCalculation calculates the inflation rate for a blockchain network, adjusting for halving events.
// It takes the current block height and checks if a halving should occur (every 210,000 blocks).
// If it's time for a halving event, the function halves the current inflation rate.
func CustomInflationCalculation(ctx context.Context, minter minttypes.Minter, params minttypes.Params, bondedRatio math.LegacyDec) math.LegacyDec {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	halvingInterval := int64(210000)
	currentBlockHeight := sdkCtx.BlockHeight()

	// Initial Bitcoin inflation rate
	currentInflationRate := params.InflationRateChange

	// Check for halving event
	if currentBlockHeight%halvingInterval == 0 {
		currentInflationRate = currentInflationRate.QuoInt64(2) // Halve the inflation rate
	}

	return currentInflationRate
}
