package mint

import (
	"context"
	"time"

	cosmosMath "cosmossdk.io/math"
	"github.com/allora-network/allora-chain/x/mint/keeper"
	"github.com/allora-network/allora-chain/x/mint/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO: MaxSupply should be defined somewhere else
var MaxSupply = cosmosMath.NewIntFromBigInt(cosmosMath.NewUintFromString("1000000000000000000000000000").BigInt())

// BeginBlocker mints new tokens for the previous block.
func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// fetch stored minter & params
	minter, err := k.Minter.Get(ctx)
	if err != nil {
		return err
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	inflationBeforeUpdates := minter.Inflation

	// fetch circulating uallo supply
	totalCirculatingSupply := k.GetSupply(ctx)

	if totalCirculatingSupply.Amount.LT(MaxSupply) {
		minter.Inflation = minter.NextInflationRate(params, sdkCtx.BlockHeight())
		minter.AnnualProvisions = minter.NextAnnualProvisions(params, totalCirculatingSupply.Amount)

		// mint coins, update supply
		mintedCoin := minter.BlockProvision(params)
		mintedCoins := sdk.NewCoins(mintedCoin)

		err = k.MintCoins(ctx, mintedCoins)
		if err != nil {
			return err
		}

		// send the minted coins to the fee collector account
		err = k.AddCollectedFees(ctx, mintedCoins)
		if err != nil {
			return err
		}

		if mintedCoin.Amount.IsInt64() {
			defer telemetry.ModuleSetGauge(types.ModuleName, float32(mintedCoin.Amount.Int64()), "minted_tokens")
		}

	} else {
		minter.Inflation = cosmosMath.LegacyZeroDec()
		minter.AnnualProvisions = cosmosMath.LegacyZeroDec()
	}

	// Just update if the current inflation is not zero to avoid unnecessary writes.
	if !inflationBeforeUpdates.Equal(cosmosMath.LegacyZeroDec()) {
		// set new minter inflation and annual provisions
		err = k.Minter.Set(ctx, minter)
		if err != nil {
			return err
		}

		//set new params
		params.InflationRateChange = minter.Inflation
		err = k.Params.Set(ctx, params)
		if err != nil {
			return err
		}
	}

	return nil
}
