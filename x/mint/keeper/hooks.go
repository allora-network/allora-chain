package keeper

import (
	"context"
	"fmt"

	epochstypes "github.com/allora-network/allora-chain/x/epochs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeforeEpochStart is a hook which is executed before the start of an epoch.
func (k Keeper) BeforeEpochStart(ctx context.Context, epochIdentifier string, epochNumber int64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Print to console - this will be very frequent!
	fmt.Printf("🚀🚀🚀 MINT HOOK: BeforeEpochStart - Epoch '%s' #%d starting at block %d 🚀🚀🚀\n",
		epochIdentifier, epochNumber, sdkCtx.BlockHeight())

	// Also log it with INFO level to ensure it appears
	sdkCtx.Logger().Info("🚀 MINT HOOK: BeforeEpochStart",
		"epochIdentifier", epochIdentifier,
		"epochNumber", epochNumber,
		"blockHeight", sdkCtx.BlockHeight())

	return nil
}

// AfterEpochEnd is a hook which is executed after the end of an epoch.
func (k Keeper) AfterEpochEnd(ctx context.Context, epochIdentifier string, epochNumber int64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Print to console - this will be very frequent!
	fmt.Printf("🎉🎉🎉 MINT HOOK: AfterEpochEnd - Epoch '%s' #%d ended at block %d 🎉🎉🎉\n",
		epochIdentifier, epochNumber, sdkCtx.BlockHeight())

	// Also log it with INFO level to ensure it appears
	sdkCtx.Logger().Info("🎉 MINT HOOK: AfterEpochEnd",
		"epochIdentifier", epochIdentifier,
		"epochNumber", epochNumber,
		"blockHeight", sdkCtx.BlockHeight())

	return nil
}

// ___________________________________________________________________________________________________

// Hooks wrapper struct for mint keeper.
type Hooks struct {
	k Keeper
}

var _ epochstypes.EpochHooks = Hooks{}

// Return the wrapper struct.
func (k Keeper) Hooks() epochstypes.EpochHooks {
	return Hooks{k}
}

// epochs hooks.
func (h Hooks) BeforeEpochStart(ctx context.Context, epochIdentifier string, epochNumber int64) error {
	return h.k.BeforeEpochStart(ctx, epochIdentifier, epochNumber)
}

func (h Hooks) AfterEpochEnd(ctx context.Context, epochIdentifier string, epochNumber int64) error {
	return h.k.AfterEpochEnd(ctx, epochIdentifier, epochNumber)
}
